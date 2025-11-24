package synthesis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

var configMapTracer = otel.Tracer("language-operator/configmap-manager")

// ConfigMapManager manages versioned ConfigMaps for agent code synthesis
type ConfigMapManager struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
}

// RetentionPolicy defines ConfigMap retention settings
type RetentionPolicy struct {
	KeepLastN        int32         // Number of latest versions to keep (0 = unlimited)
	CleanupAfterDays int32         // Delete versions older than N days (0 = never)
	AlwaysKeepInitial bool          // Always preserve v1 (initial synthesis)
	CleanupInterval  time.Duration // How often to run cleanup
}

// ConfigMapVersion represents metadata about a versioned ConfigMap
type ConfigMapVersion struct {
	Name           string            `json:"name"`
	Version        int32             `json:"version"`
	SynthesisType  string            `json:"synthesisType"`  // initial, learned, manual
	PreviousVersion *int32           `json:"previousVersion,omitempty"`
	LearnedTask    string            `json:"learnedTask,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	Labels         map[string]string `json:"labels"`
	Annotations    map[string]string `json:"annotations"`
}

// CreateVersionedConfigMap creates a new versioned ConfigMap with enhanced metadata tracking
func (cm *ConfigMapManager) CreateVersionedConfigMap(ctx context.Context, agent *langopv1alpha1.LanguageAgent, options *ConfigMapOptions) (*corev1.ConfigMap, error) {
	ctx, span := configMapTracer.Start(ctx, "configmap.create_versioned")
	defer span.End()

	// Validate inputs
	if options == nil {
		return nil, fmt.Errorf("ConfigMapOptions cannot be nil")
	}
	if options.Version <= 0 {
		return nil, fmt.Errorf("version must be positive, got: %d", options.Version)
	}

	configMapName := fmt.Sprintf("%s-v%d", agent.Name, options.Version)

	// Build labels with enhanced tracking
	labels := map[string]string{
		"langop.io/agent":          agent.Name,
		"langop.io/version":        fmt.Sprintf("%d", options.Version),
		"langop.io/synthesis-type": options.SynthesisType,
		"langop.io/component":      "agent-code",
	}

	// Add optional learned task information
	if options.LearnedTask != "" {
		labels["langop.io/learned-task"] = options.LearnedTask
	}

	// Add previous version tracking if specified
	if options.PreviousVersion != nil && *options.PreviousVersion > 0 {
		labels["langop.io/previous-version"] = fmt.Sprintf("%d", *options.PreviousVersion)
	}

	// Build annotations with enhanced metadata
	annotations := map[string]string{
		"langop.io/created-at":    time.Now().Format(time.RFC3339),
		"langop.io/learned-from":  options.LearningSource,
	}

	// Add custom annotations
	for k, v := range options.CustomAnnotations {
		annotations[k] = v
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        configMapName,
			Namespace:   agent.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: map[string]string{
			"agent.rb": options.Code,
		},
	}

	// Set owner reference for garbage collection
	if err := controllerutil.SetControllerReference(agent, configMap, cm.Scheme); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Create the ConfigMap
	if err := cm.Create(ctx, configMap); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create versioned ConfigMap %s: %w", configMapName, err)
	}

	// Add telemetry attributes
	span.SetAttributes(
		attribute.String("configmap.name", configMapName),
		attribute.Int("configmap.version", int(options.Version)),
		attribute.String("configmap.synthesis_type", options.SynthesisType),
	)

	cm.Log.Info("Created versioned ConfigMap",
		"configmap", configMapName,
		"version", options.Version,
		"synthesis_type", options.SynthesisType,
		"learned_task", options.LearnedTask)

	return configMap, nil
}

// ConfigMapOptions contains parameters for creating versioned ConfigMaps
type ConfigMapOptions struct {
	Code               string            `json:"code"`                         // Agent code content
	Version            int32             `json:"version"`                      // Version number (must be positive)
	SynthesisType      string            `json:"synthesisType"`                // initial, learned, manual
	PreviousVersion    *int32            `json:"previousVersion,omitempty"`    // Previous version for tracking
	LearnedTask        string            `json:"learnedTask,omitempty"`        // Task that triggered learning
	LearningSource     string            `json:"learningSource,omitempty"`     // Source of learning: pattern-detection, error-recovery, manual
	CustomAnnotations  map[string]string `json:"customAnnotations,omitempty"`  // Additional annotations
}

// GetVersionedConfigMaps retrieves all versioned ConfigMaps for an agent
func (cm *ConfigMapManager) GetVersionedConfigMaps(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]*ConfigMapVersion, error) {
	ctx, span := configMapTracer.Start(ctx, "configmap.get_versions")
	defer span.End()

	// List ConfigMaps with agent label
	configMapList := &corev1.ConfigMapList{}
	labelSelector := labels.SelectorFromSet(map[string]string{
		"langop.io/agent":     agent.Name,
		"langop.io/component": "agent-code",
	})

	listOpts := []client.ListOption{
		client.InNamespace(agent.Namespace),
		client.MatchingLabelsSelector{Selector: labelSelector},
	}

	if err := cm.List(ctx, configMapList, listOpts...); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list versioned ConfigMaps for agent %s: %w", agent.Name, err)
	}

	var versions []*ConfigMapVersion
	for _, cmItem := range configMapList.Items {
		version, err := parseConfigMapVersion(&cmItem)
		if err != nil {
			cm.Log.Error(err, "Failed to parse ConfigMap version", "configmap", cmItem.Name)
			continue
		}
		versions = append(versions, version)
	}

	span.SetAttributes(attribute.Int("configmap.versions_found", len(versions)))
	return versions, nil
}

// parseConfigMapVersion extracts version metadata from a ConfigMap
func parseConfigMapVersion(cm *corev1.ConfigMap) (*ConfigMapVersion, error) {
	versionStr, exists := cm.Labels["langop.io/version"]
	if !exists {
		return nil, fmt.Errorf("ConfigMap %s missing version label", cm.Name)
	}

	versionInt, err := strconv.Atoi(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version format in ConfigMap %s: %s", cm.Name, versionStr)
	}

	version := &ConfigMapVersion{
		Name:          cm.Name,
		Version:       int32(versionInt),
		SynthesisType: cm.Labels["langop.io/synthesis-type"],
		LearnedTask:   cm.Labels["langop.io/learned-task"],
		Labels:        cm.Labels,
		Annotations:   cm.Annotations,
	}

	// Parse previous version if present
	if prevVersionStr, exists := cm.Labels["langop.io/previous-version"]; exists {
		if prevVersion, err := strconv.Atoi(prevVersionStr); err == nil {
			version.PreviousVersion = ptr.To(int32(prevVersion))
		}
	}

	// Parse creation timestamp
	if createdAtStr, exists := cm.Annotations["langop.io/created-at"]; exists {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			version.CreatedAt = createdAt
		}
	} else {
		// Fallback to Kubernetes creation timestamp
		version.CreatedAt = cm.CreationTimestamp.Time
	}

	return version, nil
}

// ApplyRetentionPolicy applies retention policy to versioned ConfigMaps
func (cm *ConfigMapManager) ApplyRetentionPolicy(ctx context.Context, agent *langopv1alpha1.LanguageAgent, policy *RetentionPolicy) error {
	ctx, span := configMapTracer.Start(ctx, "configmap.apply_retention_policy")
	defer span.End()

	if policy == nil {
		return nil // No retention policy to apply
	}

	versions, err := cm.GetVersionedConfigMaps(ctx, agent)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to get versioned ConfigMaps: %w", err)
	}

	if len(versions) == 0 {
		return nil // No ConfigMaps to clean up
	}

	// Sort versions by version number (newest first)
	sortVersionsByNumber(versions)

	var toDelete []*ConfigMapVersion
	now := time.Now()

	for i, version := range versions {
		shouldDelete := false

		// Always preserve v1 if policy requires it
		if policy.AlwaysKeepInitial && version.Version == 1 {
			continue
		}

		// Keep last N versions
		if policy.KeepLastN > 0 && int32(i) >= policy.KeepLastN {
			shouldDelete = true
		}

		// Delete versions older than specified days
		if policy.CleanupAfterDays > 0 {
			ageInDays := int32(now.Sub(version.CreatedAt).Hours() / 24)
			if ageInDays > policy.CleanupAfterDays {
				shouldDelete = true
			}
		}

		if shouldDelete {
			toDelete = append(toDelete, version)
		}
	}

	// Delete selected ConfigMaps
	for _, version := range toDelete {
		if err := cm.deleteConfigMap(ctx, agent, version); err != nil {
			cm.Log.Error(err, "Failed to delete ConfigMap during retention cleanup",
				"configmap", version.Name,
				"version", version.Version)
			// Continue with other deletions instead of failing entirely
		}
	}

	span.SetAttributes(
		attribute.Int("configmap.total_versions", len(versions)),
		attribute.Int("configmap.deleted_count", len(toDelete)),
	)

	if len(toDelete) > 0 {
		cm.Log.Info("Applied retention policy",
			"agent", agent.Name,
			"total_versions", len(versions),
			"deleted_versions", len(toDelete))
	}

	return nil
}

// deleteConfigMap deletes a specific ConfigMap version
func (cm *ConfigMapManager) deleteConfigMap(ctx context.Context, agent *langopv1alpha1.LanguageAgent, version *ConfigMapVersion) error {
	ctx, span := configMapTracer.Start(ctx, "configmap.delete_version")
	defer span.End()

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      version.Name,
			Namespace: agent.Namespace, // Use agent's namespace directly
		},
	}

	if err := cm.Delete(ctx, configMap); err != nil {
		if errors.IsNotFound(err) {
			// Already deleted, not an error
			return nil
		}
		span.RecordError(err)
		return fmt.Errorf("failed to delete ConfigMap %s: %w", version.Name, err)
	}

	span.SetAttributes(
		attribute.String("configmap.name", version.Name),
		attribute.Int("configmap.version", int(version.Version)),
	)

	cm.Log.V(1).Info("Deleted ConfigMap version",
		"configmap", version.Name,
		"version", version.Version)

	return nil
}

// CreateCleanupCronJob creates a Kubernetes CronJob to periodically apply retention policy
func (cm *ConfigMapManager) CreateCleanupCronJob(ctx context.Context, agent *langopv1alpha1.LanguageAgent, policy *RetentionPolicy) error {
	ctx, span := configMapTracer.Start(ctx, "configmap.create_cleanup_cronjob")
	defer span.End()

	if policy == nil || policy.CleanupInterval == 0 {
		return nil // No cleanup job needed
	}

	// Convert duration to cron schedule (simplified: daily at 3 AM for intervals >= 24h)
	cronSchedule := "0 3 * * *" // Daily at 3 AM
	if policy.CleanupInterval < 24*time.Hour {
		// For shorter intervals, use hourly (simplified)
		cronSchedule = "0 * * * *" // Every hour
	}

	jobName := fmt.Sprintf("%s-configmap-cleanup", agent.Name)

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: agent.Namespace,
			Labels: map[string]string{
				"langop.io/agent":     agent.Name,
				"langop.io/component": "configmap-cleanup",
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          cronSchedule,
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:  "cleanup",
									Image: "ghcr.io/language-operator/aictl:latest", // Use existing CLI image
									Command: []string{
										"/usr/local/bin/aictl",
										"agent", "cleanup",
										"--agent", agent.Name,
										"--namespace", agent.Namespace,
										"--keep-last", fmt.Sprintf("%d", policy.KeepLastN),
										"--cleanup-after-days", fmt.Sprintf("%d", policy.CleanupAfterDays),
										"--always-keep-initial=" + fmt.Sprintf("%t", policy.AlwaysKeepInitial),
									},
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("10m"),
											corev1.ResourceMemory: resource.MustParse("32Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("100m"),
											corev1.ResourceMemory: resource.MustParse("128Mi"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(agent, cronJob, cm.Scheme); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to set controller reference on cleanup CronJob: %w", err)
	}

	// Create or update the CronJob
	existingCronJob := &batchv1.CronJob{}
	err := cm.Get(ctx, types.NamespacedName{Name: jobName, Namespace: agent.Namespace}, existingCronJob)

	if errors.IsNotFound(err) {
		// Create new CronJob
		if err := cm.Create(ctx, cronJob); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to create cleanup CronJob: %w", err)
		}
		cm.Log.Info("Created ConfigMap cleanup CronJob", "cronjob", jobName, "schedule", cronSchedule)
	} else if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to check existing cleanup CronJob: %w", err)
	} else {
		// Update existing CronJob
		existingCronJob.Spec = cronJob.Spec
		if err := cm.Update(ctx, existingCronJob); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to update cleanup CronJob: %w", err)
		}
		cm.Log.V(1).Info("Updated ConfigMap cleanup CronJob", "cronjob", jobName, "schedule", cronSchedule)
	}

	span.SetAttributes(
		attribute.String("cronjob.name", jobName),
		attribute.String("cronjob.schedule", cronSchedule),
	)

	return nil
}

// GetLatestVersion returns the highest version number of ConfigMaps for an agent
func (cm *ConfigMapManager) GetLatestVersion(ctx context.Context, agent *langopv1alpha1.LanguageAgent) (int32, error) {
	versions, err := cm.GetVersionedConfigMaps(ctx, agent)
	if err != nil {
		return 0, err
	}

	if len(versions) == 0 {
		return 0, nil // No versions exist yet
	}

	var maxVersion int32
	for _, version := range versions {
		if version.Version > maxVersion {
			maxVersion = version.Version
		}
	}

	return maxVersion, nil
}

// Helper functions

// sortVersionsByNumber sorts ConfigMap versions by version number (newest first)
func sortVersionsByNumber(versions []*ConfigMapVersion) {
	// Simple selection sort by version number (descending)
	for i := 0; i < len(versions)-1; i++ {
		maxIdx := i
		for j := i + 1; j < len(versions); j++ {
			if versions[j].Version > versions[maxIdx].Version {
				maxIdx = j
			}
		}
		if maxIdx != i {
			versions[i], versions[maxIdx] = versions[maxIdx], versions[i]
		}
	}
}

