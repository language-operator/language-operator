package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/pkg/events"
	langoplabels "github.com/language-operator/language-operator/pkg/labels"
	"github.com/language-operator/language-operator/pkg/network"
)

const (
	// agentTemplateName is the name of the single Argo template in an agent's
	// WorkflowTemplate, and the entrypoint every derived Workflow invokes.
	agentTemplateName = "agent"

	// labelWorkflowTemplate is the label Argo puts on every run of a
	// WorkflowTemplate, set to the template's name: by `argo submit --from` when it
	// creates the run, and by Argo's controller for any other workflowTemplateRef
	// run. Mirrors common.LabelKeyWorkflowTemplate, whose package is too heavy to
	// import for one string.
	labelWorkflowTemplate = workflow.WorkflowFullName + "/workflow-template"

	// workflowParamEvent and workflowParamTrigger are the WorkflowTemplate's
	// per-run inputs (`argo submit -p event=... -p trigger=...`). The agent sees
	// them as AGENT_EVENT and AGENT_TRIGGER, and the event also as eventFilePath.
	workflowParamEvent   = "event"
	workflowParamTrigger = "trigger"

	// eventFilePath is where a run's event is written. It sits beside config.yaml,
	// which is mounted as a single file so this path is not inside a volume.
	eventFilePath = "/etc/agent/event.json"

	// triggerSchedule is the trigger CronWorkflow runs pass.
	triggerSchedule = "schedule"

	// serviceRetryLimit is the retry limit applied to a service-mode agent. Argo
	// requires a concrete number, so this stands in for "restart forever" — the
	// Deployment-like behaviour a service agent is expected to have.
	serviceRetryLimit = 2147483647

	// defaultTaskTTLSeconds is how long a finished task run is retained before
	// Argo garbage-collects it, when the agent does not say otherwise.
	defaultTaskTTLSeconds = int32(86400)

	// annotationGeneration records the agent generation a Workflow was built from.
	// A Workflow spec cannot be meaningfully updated once running, so together with
	// the config hash this tells us when to replace it instead.
	annotationGeneration = "langop.io/generation"
)

// agentPodBuild is the assembled, mode-independent shape of an agent's pod:
// the container it runs, everything that runs alongside it, and the pod-level
// settings. It is built once and then wrapped in whichever Argo object the
// agent's execution mode calls for. The mode reaches the containers only as the
// AGENT_EXECUTION_MODE env var.
type agentPodBuild struct {
	container      corev1.Container
	initContainers []corev1.Container
	sidecars       []corev1.Container
	volumes        []corev1.Volume
	labels         map[string]string
	podLabels      map[string]string
	podAnnotations map[string]string

	serviceAccountName        string
	securityContext           *corev1.PodSecurityContext
	imagePullSecrets          []corev1.LocalObjectReference
	nodeSelector              map[string]string
	tolerations               []corev1.Toleration
	affinity                  *corev1.Affinity
	topologySpreadConstraints []corev1.TopologySpreadConstraint
	shareProcessNamespace     bool
}

// buildAgentPodSpec assembles everything an agent pod needs: resolved models and
// tools, the agent container, init containers, sidecars, and volumes.
//
// The caller must have already run reconcileRuntimeSecret, which mutates
// agent.Spec.Deployment.EnvFrom — this builder reads that field.
func (r *LanguageAgentReconciler) buildAgentPodSpec(ctx context.Context, agent *langopv1alpha1.LanguageAgent, configHash string) (*agentPodBuild, error) {
	// Resolve model URLs and names
	modelURLs, modelNames, err := r.resolveModels(ctx, agent)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve models: %w", err)
	}

	// Resolve tool URLs
	toolURLs, err := r.resolveTools(ctx, agent)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tools: %w", err)
	}

	// Resolve sidecar tools (and any scratch volumes their bridges need)
	sidecarContainers, sidecarVolumes, err := r.resolveSidecarTools(ctx, agent)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sidecar tools: %w", err)
	}

	// Build the oauth2-proxy sidecar when auth is enabled for this agent's cluster.
	oauthSidecar, err := r.buildOAuthProxySidecar(ctx, agent)
	if err != nil {
		return nil, fmt.Errorf("building oauth2-proxy sidecar: %w", err)
	}

	labels := GetCommonLabels(agent.Name, "LanguageAgent")
	labels[langoplabels.LabelKeyLangopComponent] = "agent"

	if err := ValidateClusterReference(ctx, r.Client, agent.Namespace); err != nil {
		return nil, err
	}

	cluster := &langopv1alpha1.LanguageCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: agent.Namespace}, cluster); err != nil {
		return nil, fmt.Errorf("failed to get cluster %s: %w", agent.Namespace, err)
	}

	labels[langoplabels.LabelKeyLangopCluster] = agent.Namespace

	build := &agentPodBuild{
		labels:                    labels,
		serviceAccountName:        r.getServiceAccountName(agent),
		imagePullSecrets:          agent.Spec.Deployment.ImagePullSecrets,
		nodeSelector:              agent.Spec.Deployment.NodeSelector,
		tolerations:               agent.Spec.Deployment.Tolerations,
		affinity:                  agent.Spec.Deployment.Affinity,
		topologySpreadConstraints: agent.Spec.Deployment.TopologySpreadConstraints,
	}

	// The agent container.
	build.container = corev1.Container{
		Name:            agentTemplateName,
		Image:           agent.Spec.Image,
		ImagePullPolicy: agent.Spec.Deployment.ImagePullPolicy,
		Command:         agent.Spec.Deployment.Command,
		Args:            agent.Spec.Deployment.Args,
		Env:             r.buildAgentEnv(ctx, agent, cluster, modelURLs, modelNames, toolURLs, append(buildGitEnv(agent), buildVendorEnv(agent)...)...),
		EnvFrom:         agent.Spec.Deployment.EnvFrom,
		Resources:       agent.Spec.Deployment.Resources,
		LivenessProbe:   agent.Spec.Deployment.LivenessProbe,
		ReadinessProbe:  agent.Spec.Deployment.ReadinessProbe,
		StartupProbe:    agent.Spec.Deployment.StartupProbe,
		Ports:           buildAgentContainerPorts(agentPorts(agent)),
		SecurityContext: buildContainerSecurityContext(),
	}

	// Open the runtime inside the cloned repository, when one is configured.
	if repoDir := repositoryDir(agent); repoDir != "" {
		build.container.WorkingDir = repoDir
	}

	// Sidecars run alongside the agent for its whole life. Argo terminates them
	// when the main container exits, which is the behaviour the native-sidecar
	// restartPolicy gave us under a Deployment.
	build.sidecars = sidecarContainers
	if oauthSidecar != nil {
		build.sidecars = append(build.sidecars, *oauthSidecar)
	}

	// Inject operator-managed env vars and volume mounts into user-specified init containers.
	// Per spec/agents.md, all contracted env vars must be present in every
	// container including init containers. The agent-config volume mount is also
	// injected so init containers (e.g. runtime adapters) can read /etc/agent/config.yaml.
	userInitContainers := make([]corev1.Container, len(agent.Spec.Deployment.InitContainers))
	copy(userInitContainers, agent.Spec.Deployment.InitContainers)
	agentEnv := r.buildAgentEnv(ctx, agent, cluster, modelURLs, modelNames, toolURLs)
	agentConfigMount := corev1.VolumeMount{
		Name:      "agent-config",
		MountPath: "/etc/agent",
		ReadOnly:  true,
	}
	for i := range userInitContainers {
		userInitContainers[i].Env = append(agentEnv, userInitContainers[i].Env...)
		userInitContainers[i].VolumeMounts = append([]corev1.VolumeMount{agentConfigMount}, userInitContainers[i].VolumeMounts...)
	}

	// Prepend the repository-clone init container (right before user init
	// containers), then the workspace-seeder ahead of it, so the final order is
	// [workspace-seeder, repository, ...userInit]. The workspace is populated and
	// the repo cloned before any user init containers run.
	build.initContainers = userInitContainers
	if repoContainer := buildRepositoryInitContainer(agent, r.gitImage(), r.GitImagePullPolicy); repoContainer != nil {
		build.initContainers = append([]corev1.Container{*repoContainer}, build.initContainers...)
	}
	if seedContainer := buildWorkspaceSeedInitContainer(agent, r.gitImage(), r.GitImagePullPolicy); seedContainer != nil {
		build.initContainers = append([]corev1.Container{*seedContainer}, build.initContainers...)
	}
	build.shareProcessNamespace = len(build.initContainers) > 0 || len(build.sidecars) > 0

	// Merge user pod labels; operator-managed labels take precedence to protect
	// the Service selector and NetworkPolicy podSelector.
	build.podLabels = make(map[string]string, len(labels)+len(agent.Spec.Deployment.PodLabels))
	maps.Copy(build.podLabels, agent.Spec.Deployment.PodLabels)
	maps.Copy(build.podLabels, labels)

	// Use user-supplied pod security context if set, otherwise apply operator defaults.
	build.securityContext = buildPodSecurityContext()
	if agent.Spec.Deployment.SecurityContext != nil {
		build.securityContext = agent.Spec.Deployment.SecurityContext
	}

	// Seed pod annotations with the operator-managed config-hash, then overlay user annotations.
	build.podAnnotations = map[string]string{langoplabels.LabelKeyLangopConfigHash: configHash}
	maps.Copy(build.podAnnotations, agent.Spec.Deployment.PodAnnotations)

	// Build operator-managed volumes and volume mounts, then append user-supplied ones.
	volumes, volumeMounts := r.buildVolumes(ctx, agent)
	// Append seed ConfigMap volumes (not mounted in main container; used by workspace-seeder init container).
	volumes = append(volumes, buildWorkspaceSeedVolumes(agent)...)
	// Append git credential and CLI config volumes (mounted in the repository init container and the agent container).
	volumes = append(volumes, buildRepositoryVolumes(agent)...)
	// Append scratch volumes for stdio sidecar bridges (mounted only in their sidecar containers).
	volumes = append(volumes, sidecarVolumes...)
	volumes = append(volumes, agent.Spec.Deployment.Volumes...)
	build.volumes = volumes
	volumeMounts = append(volumeMounts, buildRepositoryMounts(agent)...)
	build.container.VolumeMounts = append(volumeMounts, agent.Spec.Deployment.VolumeMounts...)

	return build, nil
}

// buildWorkflowSpec renders the pod build as a single-template Argo WorkflowSpec.
// This is the shared body: reconcileWorkflowTemplate stores it, while the derived
// Workflow and CronWorkflow reference it and layer on mode-specific behaviour.
func buildWorkflowSpec(build *agentPodBuild) (wfv1.WorkflowSpec, error) {
	container := build.container

	spec := wfv1.WorkflowSpec{
		Entrypoint:         agentTemplateName,
		ServiceAccountName: build.serviceAccountName,
		Volumes:            build.volumes,
		SecurityContext:    build.securityContext,
		ImagePullSecrets:   build.imagePullSecrets,
		NodeSelector:       build.nodeSelector,
		Tolerations:        build.tolerations,
		Affinity:           build.affinity,
		PodMetadata: &wfv1.Metadata{
			Labels:      build.podLabels,
			Annotations: build.podAnnotations,
		},
		// Every run of this template carries the agent's labels, however it was
		// started (cron, `argo submit --from`, another controller). Argo copies
		// them onto the Workflow on its first pass.
		WorkflowMetadata: &wfv1.WorkflowMetadata{
			Labels: maps.Clone(build.labels),
		},
		// Per-run inputs. Empty defaults keep a run submitted without them
		// identical to one from before they existed.
		Arguments: wfv1.Arguments{
			Parameters: []wfv1.Parameter{
				{Name: workflowParamEvent, Value: wfv1.AnyStringPtr("")},
				{Name: workflowParamTrigger, Value: wfv1.AnyStringPtr("")},
			},
		},
		Templates: []wfv1.Template{
			{
				Name:           agentTemplateName,
				Container:      &container,
				InitContainers: toUserContainers(build.initContainers),
				Sidecars:       toUserContainers(build.sidecars),
				// The event as a file too, for runtimes that prefer files and for
				// payloads too large to want in the environment. Argo's init container
				// writes the raw data and bind-mounts it into the agent container.
				Inputs: wfv1.Inputs{
					Artifacts: wfv1.Artifacts{
						{
							Name: workflowParamEvent,
							Path: eventFilePath,
							ArtifactLocation: wfv1.ArtifactLocation{
								Raw: &wfv1.RawArtifact{Data: workflowParameterRef(workflowParamEvent)},
							},
						},
					},
				},
			},
		},
	}

	// shareProcessNamespace and topologySpreadConstraints have no field on
	// WorkflowSpec, so they are applied as a strategic-merge patch over the pod
	// Argo generates.
	patch, err := buildPodSpecPatch(build)
	if err != nil {
		return wfv1.WorkflowSpec{}, err
	}
	spec.PodSpecPatch = patch

	return spec, nil
}

// workflowParameterRef is the Argo expression for a WorkflowTemplate parameter.
func workflowParameterRef(name string) string {
	return "{{workflow.parameters." + name + "}}"
}

// buildPodSpecPatch renders the pod fields Argo's WorkflowSpec has no equivalent
// for as a strategic-merge patch. Returns "" when there is nothing to patch.
func buildPodSpecPatch(build *agentPodBuild) (string, error) {
	patch := corev1.PodSpec{}
	empty := true

	if build.shareProcessNamespace {
		share := true
		patch.ShareProcessNamespace = &share
		empty = false
	}
	if len(build.topologySpreadConstraints) > 0 {
		patch.TopologySpreadConstraints = build.topologySpreadConstraints
		empty = false
	}
	if empty {
		return "", nil
	}

	// PodSpec marshals required-but-empty fields (containers: null), which Argo
	// rejects; round-trip through a map and drop them.
	raw, err := json.Marshal(patch)
	if err != nil {
		return "", fmt.Errorf("marshalling podSpecPatch: %w", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "", fmt.Errorf("unmarshalling podSpecPatch: %w", err)
	}
	for k, v := range fields {
		if v == nil {
			delete(fields, k)
		}
	}
	out, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("re-marshalling podSpecPatch: %w", err)
	}
	return string(out), nil
}

// toUserContainers adapts plain containers to Argo's UserContainer wrapper.
func toUserContainers(containers []corev1.Container) []wfv1.UserContainer {
	if len(containers) == 0 {
		return nil
	}
	out := make([]wfv1.UserContainer, 0, len(containers))
	for _, c := range containers {
		out = append(out, wfv1.UserContainer{Container: c})
	}
	return out
}

// reconcileWorkflowTemplate renders the agent's WorkflowTemplate — the single
// source of truth for its podspec, and the object a manual run is submitted
// against (`argo submit --from workflowtemplate/<agent>`).
func (r *LanguageAgentReconciler) reconcileWorkflowTemplate(ctx context.Context, agent *langopv1alpha1.LanguageAgent, configHash string) error {
	build, err := r.buildAgentPodSpec(ctx, agent, configHash)
	if err != nil {
		return err
	}
	spec, err := buildWorkflowSpec(build)
	if err != nil {
		return err
	}

	tmpl := &wfv1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	}

	return CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, tmpl, func() error {
		tmpl.Labels = build.labels
		tmpl.Annotations = workflowRevisionAnnotations(agent, configHash)
		tmpl.Spec = spec
		return nil
	})
}

// workflowRevisionAnnotations identify which version of the agent an Argo object
// was rendered from, so a running Workflow can be recognised as stale.
func workflowRevisionAnnotations(agent *langopv1alpha1.LanguageAgent, configHash string) map[string]string {
	return map[string]string{
		langoplabels.LabelKeyLangopConfigHash: configHash,
		annotationGeneration:                  fmt.Sprintf("%d", agent.Generation),
	}
}

// reconcileServiceWorkflow maintains the long-lived Workflow behind a service-mode
// agent — the always-on pod that a Deployment used to provide.
//
// A Workflow's spec cannot be meaningfully updated once it is running, so this is
// deliberately not a CreateOrUpdate: a Workflow rendered from a stale generation or
// config hash is deleted and recreated.
func (r *LanguageAgentReconciler) reconcileServiceWorkflow(ctx context.Context, agent *langopv1alpha1.LanguageAgent, configHash string) error {
	desiredAnnotations := workflowRevisionAnnotations(agent, configHash)

	existing := &wfv1.Workflow{}
	err := r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, existing)
	switch {
	case err == nil:
		// Suspended agents keep their WorkflowTemplate but not a running pod.
		if agentSuspended(agent) {
			return r.deleteWorkflowIfExists(ctx, agent.Name, agent.Namespace)
		}
		if annotationsMatch(existing.Annotations, desiredAnnotations) {
			return nil
		}
		// Stale: tear down so the replacement below picks up the new spec.
		if err := r.Delete(ctx, existing); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete stale agent Workflow: %w", err)
		}
	case apierrors.IsNotFound(err):
		if agentSuspended(agent) {
			return nil
		}
	default:
		return fmt.Errorf("failed to get agent Workflow: %w", err)
	}

	limit := intstr.FromInt(serviceRetryLimit)
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:        agent.Name,
			Namespace:   agent.Namespace,
			Labels:      GetCommonLabels(agent.Name, "LanguageAgent"),
			Annotations: desiredAnnotations,
		},
		Spec: wfv1.WorkflowSpec{
			WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: agent.Name},
			// Restart forever: a service agent whose process dies should come back,
			// exactly as a Deployment would have restarted it.
			RetryStrategy: &wfv1.RetryStrategy{
				Limit:       &limit,
				RetryPolicy: wfv1.RetryPolicyAlways,
			},
			// No TTL and no pod GC — the pod is the agent, and it must stay.
			PodGC: &wfv1.PodGC{Strategy: wfv1.PodGCOnPodNone},
		},
	}
	if err := controllerutil.SetControllerReference(agent, wf, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, wf); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create agent Workflow: %w", err)
	}
	return nil
}

// reconcileCronWorkflow maintains the CronWorkflow that fires scheduled runs for a
// task-mode agent, and removes it when the schedule is cleared or the agent
// switches back to service mode.
func (r *LanguageAgentReconciler) reconcileCronWorkflow(ctx context.Context, agent *langopv1alpha1.LanguageAgent, configHash string) error {
	scheduled := agent.Spec.Execution.EffectiveExecutionMode() == langopv1alpha1.ExecutionModeTask &&
		agent.Spec.Execution.Schedule != ""

	if !scheduled {
		return r.deleteCronWorkflowIfExists(ctx, agent.Name, agent.Namespace)
	}

	cron := &wfv1.CronWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	}

	return CreateOrUpdateOwned(ctx, r.Client, r.Scheme, agent, cron, func() error {
		cron.Labels = GetCommonLabels(agent.Name, "LanguageAgent")
		cron.Annotations = workflowRevisionAnnotations(agent, configHash)

		concurrency := agent.Spec.Execution.ConcurrencyPolicy
		if concurrency == "" {
			concurrency = string(wfv1.ForbidConcurrent)
		}

		cron.Spec = wfv1.CronWorkflowSpec{
			Schedules:         []string{agent.Spec.Execution.Schedule},
			Timezone:          agent.Spec.Execution.Timezone,
			ConcurrencyPolicy: wfv1.ConcurrencyPolicy(concurrency),
			Suspend:           agentSuspended(agent),
			WorkflowMetadata: &metav1.ObjectMeta{
				Labels: GetCommonLabels(agent.Name, "LanguageAgent"),
			},
			WorkflowSpec: buildTaskWorkflowSpec(agent),
		}
		return nil
	})
}

// buildTaskWorkflowSpec is the spec each task run is created with: a reference to
// the agent's WorkflowTemplate plus the run-scoped limits.
func buildTaskWorkflowSpec(agent *langopv1alpha1.LanguageAgent) wfv1.WorkflowSpec {
	spec := wfv1.WorkflowSpec{
		WorkflowTemplateRef:   &wfv1.WorkflowTemplateRef{Name: agent.Name},
		ActiveDeadlineSeconds: agent.Spec.Execution.ActiveDeadlineSeconds,
		// Only the CronWorkflow builds runs from this spec, so they all say so.
		Arguments: wfv1.Arguments{
			Parameters: []wfv1.Parameter{
				{Name: workflowParamTrigger, Value: wfv1.AnyStringPtr(triggerSchedule)},
			},
		},
	}

	ttl := defaultTaskTTLSeconds
	if agent.Spec.Execution.TTLSecondsAfterFinished != nil {
		ttl = *agent.Spec.Execution.TTLSecondsAfterFinished
	}
	spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterCompletion: &ttl}

	if agent.Spec.Execution.RetryLimit != nil {
		limit := intstr.FromInt32(*agent.Spec.Execution.RetryLimit)
		spec.RetryStrategy = &wfv1.RetryStrategy{
			Limit:       &limit,
			RetryPolicy: wfv1.RetryPolicyOnFailure,
		}
	}

	return spec
}

// syncWorkflowStatus reads Argo state back onto the agent's status and returns the
// phase the agent should report, plus whether any status field actually changed.
//
// agent is the stored object whose status is written; workingAgent carries the
// runtime-merged spec that decides which mode to read.
func (r *LanguageAgentReconciler) syncWorkflowStatus(
	ctx context.Context,
	agent, workingAgent *langopv1alpha1.LanguageAgent,
) (string, bool, error) {
	before := agent.Status.DeepCopy()

	var phase string
	var err error
	if workingAgent.Spec.Execution.EffectiveExecutionMode() == langopv1alpha1.ExecutionModeService {
		phase, err = r.syncServiceWorkflowStatus(ctx, agent, workingAgent)
	} else {
		phase, err = r.syncTaskWorkflowStatus(ctx, agent, workingAgent)
	}
	if err != nil {
		return events.PhaseStatusPending, false, err
	}

	changed := !workflowStatusEqual(before, &agent.Status)
	return phase, changed, nil
}

// syncServiceWorkflowStatus derives status from the long-lived Workflow behind a
// service-mode agent.
func (r *LanguageAgentReconciler) syncServiceWorkflowStatus(
	ctx context.Context,
	agent, workingAgent *langopv1alpha1.LanguageAgent,
) (string, error) {
	wf := &wfv1.Workflow{}
	err := r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, wf)
	if apierrors.IsNotFound(err) {
		agent.Status.ActiveWorkflowName = ""
		clearRunStatus(&agent.Status)
		if agentSuspended(workingAgent) {
			return events.PhaseStatusSuspended, nil
		}
		return events.PhaseStatusPending, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get agent Workflow: %w", err)
	}

	agent.Status.ActiveWorkflowName = wf.Name
	applyRunStatus(&agent.Status, wf)
	agent.Status.LastScheduledTime = nil

	return workflowPhaseToAgentPhase(wf.Status.Phase), nil
}

// syncTaskWorkflowStatus derives status from the most recent run of a task-mode
// agent, plus its CronWorkflow when one is scheduled.
func (r *LanguageAgentReconciler) syncTaskWorkflowStatus(
	ctx context.Context,
	agent, workingAgent *langopv1alpha1.LanguageAgent,
) (string, error) {
	agent.Status.ActiveWorkflowName = ""

	runs, err := r.listAgentRuns(ctx, agent)
	if err != nil {
		return "", err
	}

	var latest *wfv1.Workflow
	for i := range runs {
		if latest == nil || runNewer(&runs[i], latest) {
			latest = &runs[i]
		}
	}

	if latest == nil {
		clearRunStatus(&agent.Status)
	} else {
		applyRunStatus(&agent.Status, latest)
	}

	// Surface when the schedule last fired, so an agent that never runs is visible.
	agent.Status.LastScheduledTime = nil
	if workingAgent.Spec.Execution.Schedule != "" {
		cron := &wfv1.CronWorkflow{}
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, cron); err == nil {
			agent.Status.LastScheduledTime = cron.Status.LastScheduledTime
		} else if !apierrors.IsNotFound(err) {
			return "", fmt.Errorf("failed to get agent CronWorkflow: %w", err)
		}
	}

	if agentSuspended(workingAgent) {
		return events.PhaseStatusSuspended, nil
	}
	if latest == nil {
		return events.PhaseStatusPending, nil
	}
	return workflowPhaseToAgentPhase(latest.Status.Phase), nil
}

// applyRunStatus copies a Workflow's run history onto the agent status.
// listAgentRuns returns every run of the agent's WorkflowTemplate, however it was
// started. A run carries the agent's labels once Argo has processed it (they come
// from the template's workflowMetadata, or the CronWorkflow's), and Argo's
// workflow-template label from creation when it came from `argo submit --from`.
// Matching either tracks a run from the moment it exists.
func (r *LanguageAgentReconciler) listAgentRuns(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]wfv1.Workflow, error) {
	selectors := []client.MatchingLabels{
		{langoplabels.LabelKeyK8sName: agent.Name, langoplabels.LabelKeyLangopKind: "LanguageAgent"},
		{labelWorkflowTemplate: agent.Name},
	}
	seen := map[string]bool{}
	var runs []wfv1.Workflow
	for _, sel := range selectors {
		list := &wfv1.WorkflowList{}
		if err := r.List(ctx, list, client.InNamespace(agent.Namespace), sel); err != nil {
			return nil, fmt.Errorf("failed to list agent Workflow runs: %w", err)
		}
		for _, run := range list.Items {
			if !seen[run.Name] {
				seen[run.Name] = true
				runs = append(runs, run)
			}
		}
	}
	return runs, nil
}

// runNewer reports whether run a is more recent than run b. A run that has not
// started yet (Pending) has no StartedAt, so it is placed by when it was created;
// otherwise a just-submitted run would never count as the latest. Ties break on
// name so the choice is stable across reconciles.
func runNewer(a, b *wfv1.Workflow) bool {
	ta, tb := runTime(a), runTime(b)
	if ta.Equal(tb) {
		return a.Name > b.Name
	}
	return ta.After(tb)
}

func runTime(run *wfv1.Workflow) time.Time {
	if !run.Status.StartedAt.IsZero() {
		return run.Status.StartedAt.Time
	}
	return run.CreationTimestamp.Time
}

func applyRunStatus(status *langopv1alpha1.LanguageAgentStatus, wf *wfv1.Workflow) {
	status.LastRunName = wf.Name
	status.LastRunPhase = string(wf.Status.Phase)
	status.LastRunStartedAt = nilIfZeroTime(wf.Status.StartedAt)
	status.LastRunFinishedAt = nilIfZeroTime(wf.Status.FinishedAt)
}

// clearRunStatus zeroes the run history fields when no run exists.
func clearRunStatus(status *langopv1alpha1.LanguageAgentStatus) {
	status.LastRunName = ""
	status.LastRunPhase = ""
	status.LastRunStartedAt = nil
	status.LastRunFinishedAt = nil
}

// nilIfZeroTime maps Argo's zero-valued timestamps to nil so the status field is
// simply absent rather than reporting the epoch.
func nilIfZeroTime(t metav1.Time) *metav1.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// workflowPhaseToAgentPhase maps an Argo Workflow phase onto the agent's phase.
// Argo's Error (an infrastructure failure) is not meaningfully different from
// Failed to someone looking at an agent, so both report Failed.
func workflowPhaseToAgentPhase(phase wfv1.WorkflowPhase) string {
	switch phase {
	case wfv1.WorkflowRunning:
		return events.PhaseStatusRunning
	case wfv1.WorkflowSucceeded:
		return events.PhaseStatusSucceeded
	case wfv1.WorkflowFailed, wfv1.WorkflowError:
		return events.PhaseStatusFailed
	default:
		return events.PhaseStatusPending
	}
}

// workflowStatusEqual compares only the workflow-derived status fields.
func workflowStatusEqual(a, b *langopv1alpha1.LanguageAgentStatus) bool {
	return a.ActiveWorkflowName == b.ActiveWorkflowName &&
		a.LastRunName == b.LastRunName &&
		a.LastRunPhase == b.LastRunPhase &&
		a.LastRunStartedAt.Equal(b.LastRunStartedAt) &&
		a.LastRunFinishedAt.Equal(b.LastRunFinishedAt) &&
		a.LastScheduledTime.Equal(b.LastScheduledTime)
}

// agentSuspended reports whether the agent is explicitly suspended.
func agentSuspended(agent *langopv1alpha1.LanguageAgent) bool {
	return agent.Spec.Execution.Suspend != nil && *agent.Spec.Execution.Suspend
}

// annotationsMatch reports whether every desired annotation is present with the
// same value in have.
func annotationsMatch(have, want map[string]string) bool {
	for k, v := range want {
		if have[k] != v {
			return false
		}
	}
	return true
}

// deleteWorkflowIfExists removes a Workflow, tolerating its absence.
func (r *LanguageAgentReconciler) deleteWorkflowIfExists(ctx context.Context, name, namespace string) error {
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if err := r.Delete(ctx, wf); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete agent Workflow: %w", err)
	}
	return nil
}

// deleteCronWorkflowIfExists removes a CronWorkflow, tolerating its absence.
func (r *LanguageAgentReconciler) deleteCronWorkflowIfExists(ctx context.Context, name, namespace string) error {
	cron := &wfv1.CronWorkflow{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if err := r.Delete(ctx, cron); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete agent CronWorkflow: %w", err)
	}
	return nil
}

func (r *LanguageAgentReconciler) resolveModels(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]string, []string, error) {
	var modelURLs []string
	var modelNames []string

	for _, modelRef := range agent.Spec.Models {
		// Fetch the LanguageModel (always in the agent's namespace)
		model := &langopv1alpha1.LanguageModel{}
		if err := r.Get(ctx, types.NamespacedName{Name: modelRef.Name, Namespace: agent.Namespace}, model); err != nil {
			return nil, nil, fmt.Errorf("failed to get model %s/%s: %w", agent.Namespace, modelRef.Name, err)
		}

		// All models in a cluster are served by the shared gateway
		// in the cluster namespace. Deduplicate: only add the gateway URL once.
		gatewayURL := serviceURL("gateway", agent.Namespace, network.GatewayServicePort)
		alreadyAdded := false
		for _, u := range modelURLs {
			if u == gatewayURL {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			modelURLs = append(modelURLs, gatewayURL)
		}

		// The name the agent calls the gateway with.
		name, err := gatewayModelName(model, modelRef)
		if err != nil {
			return nil, nil, err
		}
		if name != "" {
			modelNames = append(modelNames, name)
		}
	}

	return modelURLs, modelNames, nil
}

// gatewayModelName is the name an agent calls the gateway with for one of its
// model references. For a wildcard LanguageModel (modelName "*") that is
// "<LanguageModel name>/<ref.model>", which is how the gateway registers it;
// otherwise the LanguageModel's own modelName.
func gatewayModelName(model *langopv1alpha1.LanguageModel, ref langopv1alpha1.ModelReference) (string, error) {
	if model.Spec.ModelName == langopv1alpha1.WildcardModelName {
		if ref.Model == "" {
			return "", fmt.Errorf("model %q is a wildcard: set spec.models[].model to the model to use from it", model.Name)
		}
		return model.Name + "/" + ref.Model, nil
	}
	if ref.Model != "" {
		return "", fmt.Errorf("model %q is not a wildcard: spec.models[].model only applies to LanguageModels with modelName \"*\"", model.Name)
	}
	return model.Spec.ModelName, nil
}

func (r *LanguageAgentReconciler) resolveSidecarTools(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]corev1.Container, []corev1.Volume, error) {
	var sidecarContainers []corev1.Container
	var sidecarVolumes []corev1.Volume

	for _, toolRef := range agent.Spec.Tools {
		// Skip tools explicitly disabled by the user
		if toolRef.Enabled != nil && !*toolRef.Enabled {
			continue
		}

		// External MCP servers run elsewhere; there is nothing to run alongside the agent.
		if toolRef.External() {
			continue
		}

		// Fetch the LanguageTool (always in the agent's namespace)
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: agent.Namespace}, tool); err != nil {
			return nil, nil, fmt.Errorf("failed to get tool %s/%s: %w", agent.Namespace, toolRef.Name, err)
		}

		// Only process sidecar tools
		if tool.Spec.DeploymentMode != "sidecar" {
			continue
		}

		// Build sidecar container spec
		port := tool.Spec.Port
		if port == 0 {
			port = 8080 // Default MCP port
		}

		container := corev1.Container{
			Name: fmt.Sprintf("tool-%s", tool.Name),
			Ports: []corev1.ContainerPort{
				{
					Name:          "mcp",
					ContainerPort: port,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Env: tool.Spec.Deployment.Env,
		}

		if isStdioTool(tool) {
			// transport=stdio: wrap the stdio command in the operator-controlled persistent
			// bridge. The volume prefix is the (pod-unique) container name so multiple stdio
			// sidecars don't collide on scratch volume names.
			prefix := container.Name
			container.Image = resolveMCPBridgeImage(r.MCPBridgeImage)
			container.ImagePullPolicy = r.MCPBridgeImagePullPolicy
			container.Command = bridgeContainerCommand()
			container.Args = buildBridgeArgs(stdioCommandOf(tool), port)
			container.Env = append(container.Env, bridgeCacheEnv()...)
			container.VolumeMounts = append(container.VolumeMounts, bridgeVolumeMounts(prefix)...)
			container.ReadinessProbe = defaultBridgeReadinessProbe(port)
			sidecarVolumes = append(sidecarVolumes, bridgeVolumes(prefix)...)
		} else {
			container.Image = tool.Spec.Image
			container.ReadinessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(int(port)),
					},
				},
				InitialDelaySeconds: 2,
				PeriodSeconds:       2,
				TimeoutSeconds:      1,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			}
		}

		// Add resource requirements if specified, otherwise use sensible defaults for tool containers
		if tool.Spec.Deployment.Resources.Requests == nil && tool.Spec.Deployment.Resources.Limits == nil {
			container.Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			}
		} else {
			container.Resources = tool.Spec.Deployment.Resources
		}

		// Mount workspace if agent has workspace enabled
		if agent.Spec.Workspace != nil && (agent.Spec.Workspace.Enabled == nil || *agent.Spec.Workspace.Enabled) {
			mountPath := agent.Spec.Workspace.MountPath
			if mountPath == "" {
				mountPath = "/workspace"
			}

			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      "workspace",
				MountPath: mountPath,
			})
		}

		sidecarContainers = append(sidecarContainers, container)
	}

	return sidecarContainers, sidecarVolumes, nil
}

// resolveTools returns the MCP URLs that go into MCP_SERVERS: every enabled LanguageTool in
// the agent's namespace, plus external servers (spec.tools[].url) that need no headers. A
// server with headers, inline or a remote LanguageTool, is left out, because the
// comma-separated variable has no room for them; runtimes read it from config.yaml instead.
func (r *LanguageAgentReconciler) resolveTools(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]string, error) {
	var toolURLs []string

	for _, toolRef := range agent.Spec.Tools {
		// Skip tools explicitly disabled by the user
		if toolRef.Enabled != nil && !*toolRef.Enabled {
			continue
		}

		// External MCP servers are not looked up or deployed; the URL is used as given.
		if toolRef.External() {
			if len(toolRef.Headers) == 0 {
				toolURLs = append(toolURLs, toolRef.URL)
			}
			continue
		}

		// Fetch the LanguageTool (always in the agent's namespace)
		tool := &langopv1alpha1.LanguageTool{}
		if err := r.Get(ctx, types.NamespacedName{Name: toolRef.Name, Namespace: agent.Namespace}, tool); err != nil {
			return nil, fmt.Errorf("failed to get tool %s/%s: %w", agent.Namespace, toolRef.Name, err)
		}

		if tool.Remote() && len(tool.Spec.Headers) > 0 {
			continue
		}

		// Full Streamable HTTP MCP URL (includes /mcp); sidecar tools resolve to localhost.
		toolURLs = append(toolURLs, mcpToolEndpoint(tool, agent.Namespace))
	}

	return toolURLs, nil
}

// buildVolumes creates the volumes and volume mounts for agent pods
func (r *LanguageAgentReconciler) buildVolumes(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// /tmp - general scratch space for the read-only root filesystem. Memory-backed
	// (tmpfs) and capped: see AgentTmpSizeLimit. Large data belongs on the workspace PVC.
	tmpLimit := resource.MustParse(AgentTmpSizeLimit)
	volumes = append(volumes, corev1.Volume{
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium:    corev1.StorageMediumMemory,
				SizeLimit: &tmpLimit,
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "tmp",
		MountPath: "/tmp",
	})

	// Mount config.yaml from the agent ConfigMap as a single file. The volume
	// projects only that key (the others back env vars), and the subPath mount
	// leaves /etc/agent itself outside any volume, so Argo can bind-mount a run's
	// event at /etc/agent/event.json; an artifact path inside a volume would be
	// written into that volume, and this one is read-only. Losing live ConfigMap
	// updates costs nothing: a config change replaces the pod.
	agentConfigMapName := GenerateConfigMapName(agent.Name, "agent")
	volumes = append(volumes, corev1.Volume{
		Name: "agent-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: agentConfigMapName,
				},
				Items: []corev1.KeyToPath{{Key: agentConfigKeyConfig, Path: agentConfigKeyConfig}},
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "agent-config",
		MountPath: "/etc/agent/" + agentConfigKeyConfig,
		SubPath:   agentConfigKeyConfig,
		ReadOnly:  true,
	})

	// Add workspace volume if enabled
	if agent.Spec.Workspace != nil && (agent.Spec.Workspace.Enabled == nil || *agent.Spec.Workspace.Enabled) {
		mountPath := agent.Spec.Workspace.MountPath
		if mountPath == "" {
			mountPath = "/workspace"
		}

		volumes = append(volumes, corev1.Volume{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: GeneratePVCName(agent.Name),
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "workspace",
			MountPath: mountPath,
		})
	}

	return volumes, volumeMounts
}
