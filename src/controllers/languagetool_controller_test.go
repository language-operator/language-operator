package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/language-operator/language-operator/internal/testutil/gen"
	"github.com/language-operator/language-operator/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// mockRegistryManager is a mock implementation of RegistryManager for testing
type mockRegistryManager struct {
	registries []string
}

func (m *mockRegistryManager) GetRegistries() []string {
	if m.registries == nil {
		return []string{"docker.io", "gcr.io", "ghcr.io"}
	}
	return m.registries
}

func TestLanguageToolController_SidecarMode(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("test-sidecar-tool", "default",
		gen.SetToolImage("test:latest"),
		gen.SetToolDeploymentMode("sidecar"),
		gen.SetToolPort(8080),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      tool.Name,
			Namespace: tool.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify no deployment was created for sidecar mode
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      tool.Name,
		Namespace: tool.Namespace,
	}, deployment)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected deployment to not exist for sidecar mode, but got: %v", err)
	}

	// Verify no service was created for sidecar mode
	service := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      tool.Name,
		Namespace: tool.Namespace,
	}, service)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected service to not exist for sidecar mode, but got: %v", err)
	}

	// Verify status phase and Ready condition are set correctly for sidecar mode
	updatedTool := &langopv1alpha1.LanguageTool{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, updatedTool); err != nil {
		t.Fatalf("Failed to get updated tool: %v", err)
	}
	if updatedTool.Status.Phase != events.PhaseStatusRunning {
		t.Errorf("Expected status.phase %q, got %q", events.PhaseStatusRunning, updatedTool.Status.Phase)
	}
	var readyCond *metav1.Condition
	for i := range updatedTool.Status.Conditions {
		if updatedTool.Status.Conditions[i].Type == langopv1alpha1.ConditionReady {
			readyCond = &updatedTool.Status.Conditions[i]
			break
		}
	}
	if readyCond == nil {
		t.Fatal("Expected Ready condition to be set, but it was not found")
	}
	if readyCond.Status != metav1.ConditionTrue {
		t.Errorf("Expected Ready condition Status %q, got %q", metav1.ConditionTrue, readyCond.Status)
	}
	if readyCond.Reason != "ReconcileSuccess" {
		t.Errorf("Expected Ready condition Reason %q, got %q", "ReconcileSuccess", readyCond.Reason)
	}
}

func TestLanguageToolController_ServiceMode(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("test-service-tool", "default",
		gen.SetToolImage("test:latest"),
		gen.SetToolDeploymentMode("service"),
		gen.SetToolPort(8080),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      tool.Name,
			Namespace: tool.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify deployment was created for service mode
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      tool.Name,
		Namespace: tool.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected deployment to exist for service mode, but got error: %v", err)
	}
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(deployment.Spec.Template.Spec.Containers))
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != "test:latest" {
		t.Errorf("Expected image 'test:latest', got '%s'", deployment.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify service was created for service mode
	service := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      tool.Name,
		Namespace: tool.Namespace,
	}, service)
	if err != nil {
		t.Fatalf("Expected service to exist for service mode, but got error: %v", err)
	}
}

func TestLanguageToolController_StatusPhases(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tests := []struct {
		name              string
		deploymentStatus  *appsv1.DeploymentStatus
		expectedPhase     string
		expectedCondition metav1.ConditionStatus
		expectedReason    string
	}{
		{
			name: "Pending - no replicas yet",
			deploymentStatus: &appsv1.DeploymentStatus{
				ReadyReplicas:       0,
				AvailableReplicas:   0,
				UpdatedReplicas:     1,
				UnavailableReplicas: 0,
			},
			expectedPhase:     "Pending",
			expectedCondition: metav1.ConditionFalse,
			expectedReason:    "Pending",
		},
		{
			name: "Failed - pods exist but none ready (CrashLoopBackOff)",
			deploymentStatus: &appsv1.DeploymentStatus{
				ReadyReplicas:       0,
				AvailableReplicas:   0,
				UpdatedReplicas:     1,
				UnavailableReplicas: 1,
			},
			expectedPhase:     "Failed",
			expectedCondition: metav1.ConditionFalse,
			expectedReason:    "PodsNotReady",
		},
		{
			name: "Running - at least one pod ready",
			deploymentStatus: &appsv1.DeploymentStatus{
				ReadyReplicas:       1,
				AvailableReplicas:   1,
				UpdatedReplicas:     1,
				UnavailableReplicas: 0,
			},
			expectedPhase:     "Running",
			expectedCondition: metav1.ConditionTrue,
			expectedReason:    "ReconcileSuccess",
		},
		{
			name: "Updating - not all replicas updated",
			deploymentStatus: &appsv1.DeploymentStatus{
				ReadyReplicas:       1,
				AvailableReplicas:   1,
				UpdatedReplicas:     0,
				UnavailableReplicas: 0,
			},
			expectedPhase:     "Updating",
			expectedCondition: metav1.ConditionFalse,
			expectedReason:    "Updating",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := gen.LanguageTool("test-tool", "default",
				gen.SetToolImage("test:latest"),
				gen.SetToolDeploymentMode("service"),
				gen.SetToolPort(8080),
			)
			tool.Generation = 1

			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tool.Name,
					Namespace: tool.Namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: func() *int32 { r := int32(1); return &r }(),
				},
				Status: *tt.deploymentStatus,
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(gen.ReadyCluster("default"), tool, deployment).
				WithStatusSubresource(tool).
				Build()

			reconciler := &LanguageToolReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				RegistryManager: &mockRegistryManager{},
			}

			ctx := context.Background()
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tool.Name,
					Namespace: tool.Namespace,
				},
			})
			if err != nil {
				t.Fatalf("Reconcile failed: %v", err)
			}

			// Fetch updated tool to check status
			updatedTool := &langopv1alpha1.LanguageTool{}
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      tool.Name,
				Namespace: tool.Namespace,
			}, updatedTool)
			if err != nil {
				t.Fatalf("Failed to fetch updated tool: %v", err)
			}

			// Verify phase
			if updatedTool.Status.Phase != tt.expectedPhase {
				t.Errorf("Expected phase '%s', got '%s'", tt.expectedPhase, updatedTool.Status.Phase)
			}

			// Verify Ready condition
			var readyCondition *metav1.Condition
			for i := range updatedTool.Status.Conditions {
				if updatedTool.Status.Conditions[i].Type == langopv1alpha1.ConditionReady {
					readyCondition = &updatedTool.Status.Conditions[i]
					break
				}
			}
			if readyCondition == nil {
				t.Fatalf("Ready condition not found")
			}
			if readyCondition.Status != tt.expectedCondition {
				t.Errorf("Expected condition status '%s', got '%s'", tt.expectedCondition, readyCondition.Status)
			}
			if readyCondition.Reason != tt.expectedReason {
				t.Errorf("Expected reason '%s', got '%s'", tt.expectedReason, readyCondition.Reason)
			}

			// Verify replica counts are copied from deployment
			if tt.expectedPhase != "Pending" {
				if updatedTool.Status.ReadyReplicas != tt.deploymentStatus.ReadyReplicas {
					t.Errorf("Expected ReadyReplicas %d, got %d", tt.deploymentStatus.ReadyReplicas, updatedTool.Status.ReadyReplicas)
				}
			}
		})
	}
}

func TestLanguageToolController_NotFoundHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-tool",
			Namespace: "default",
		},
	})

	// Should not return error for not found
	if err != nil {
		t.Errorf("Expected no error for not found tool, got: %v", err)
	}

	// Should not requeue
	if result.Requeue {
		t.Error("Expected no requeue for not found tool")
	}
}

func TestLanguageToolController_ServiceTypeAndAnnotations(t *testing.T) {
	tests := []struct {
		name            string
		serviceType     corev1.ServiceType
		annotations     map[string]string
		wantServiceType corev1.ServiceType
	}{
		{
			name:            "defaults to ClusterIP when unset",
			wantServiceType: corev1.ServiceTypeClusterIP,
		},
		{
			name:            "NodePort applied",
			serviceType:     corev1.ServiceTypeNodePort,
			wantServiceType: corev1.ServiceTypeNodePort,
		},
		{
			name:            "annotations propagated",
			annotations:     map[string]string{"example.com/foo": "bar"},
			wantServiceType: corev1.ServiceTypeClusterIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := testutil.SetupTestScheme(t)
			mods := []gen.LanguageToolModifier{
				gen.SetToolDeploymentMode("service"),
				gen.SetToolPort(8080),
			}
			if tt.serviceType != "" {
				mods = append(mods, gen.SetToolServiceType(tt.serviceType))
			}
			if tt.annotations != nil {
				mods = append(mods, gen.SetToolServiceAnnotations(tt.annotations))
			}
			tool := gen.LanguageTool("test-tool", "default", mods...)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(gen.ReadyCluster("default"), tool).
				WithStatusSubresource(tool).
				Build()

			reconciler := &LanguageToolReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				RegistryManager: &mockRegistryManager{},
			}

			ctx := context.Background()
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace},
			})
			if err != nil {
				t.Fatalf("Reconcile failed: %v", err)
			}

			svc := &corev1.Service{}
			if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, svc); err != nil {
				t.Fatalf("expected service to exist: %v", err)
			}
			if svc.Spec.Type != tt.wantServiceType {
				t.Errorf("service type = %q, want %q", svc.Spec.Type, tt.wantServiceType)
			}
			for k, v := range tt.annotations {
				if svc.Annotations[k] != v {
					t.Errorf("annotation %q = %q, want %q", k, svc.Annotations[k], v)
				}
			}
		})
	}
}

func TestLanguageToolController_SchedulingFields(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("test-sched-tool", "default",
		gen.SetToolImage("test:latest"),
		gen.SetToolPort(8080),
	)
	tool.Spec.Deployment.NodeSelector = map[string]string{"kubernetes.io/arch": "arm64"}
	tool.Spec.Deployment.Tolerations = []corev1.Toleration{
		{Key: "spot", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
	}
	tool.Spec.Deployment.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{
		{MaxSkew: 1, TopologyKey: "topology.kubernetes.io/zone", WhenUnsatisfiable: corev1.DoNotSchedule},
	}
	tool.Spec.Deployment.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "tool-registry-secret"}}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace},
	})
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	podSpec := deployment.Spec.Template.Spec

	if podSpec.NodeSelector["kubernetes.io/arch"] != "arm64" {
		t.Errorf("Expected NodeSelector kubernetes.io/arch=arm64, got %v", podSpec.NodeSelector)
	}
	if len(podSpec.Tolerations) == 0 || podSpec.Tolerations[0].Key != "spot" {
		t.Errorf("Expected Toleration spot, got %v", podSpec.Tolerations)
	}
	if len(podSpec.TopologySpreadConstraints) == 0 || podSpec.TopologySpreadConstraints[0].TopologyKey != "topology.kubernetes.io/zone" {
		t.Errorf("Expected TopologySpreadConstraint, got %v", podSpec.TopologySpreadConstraints)
	}
	if len(podSpec.ImagePullSecrets) == 0 || podSpec.ImagePullSecrets[0].Name != "tool-registry-secret" {
		t.Errorf("Expected ImagePullSecret tool-registry-secret, got %v", podSpec.ImagePullSecrets)
	}
}

func TestLanguageToolController_DeploymentSpec(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("test-full-tool", "default",
		gen.SetToolImage("test:latest"),
		gen.SetToolPort(8080),
	)

	// imagePullPolicy
	tool.Spec.Deployment.ImagePullPolicy = corev1.PullAlways

	// envFrom
	tool.Spec.Deployment.EnvFrom = []corev1.EnvFromSource{
		{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "tool-config"}}},
	}

	// serviceAccountName
	tool.Spec.Deployment.ServiceAccountName = "custom-tool-sa"

	// volumes + volumeMounts
	tool.Spec.Deployment.Volumes = []corev1.Volume{
		{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
	tool.Spec.Deployment.VolumeMounts = []corev1.VolumeMount{
		{Name: "data", MountPath: "/data"},
	}

	// podAnnotations + podLabels
	tool.Spec.Deployment.PodAnnotations = map[string]string{"prometheus.io/scrape": "true"}
	tool.Spec.Deployment.PodLabels = map[string]string{"custom-label": "value"}

	// initContainers
	tool.Spec.Deployment.InitContainers = []corev1.Container{
		{Name: "init-setup", Image: "busybox:latest", Command: []string{"sh", "-c", "echo ready"}},
	}

	// probes
	probe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromInt32(8080)},
		},
		InitialDelaySeconds: 5,
	}
	tool.Spec.Deployment.LivenessProbe = probe
	tool.Spec.Deployment.ReadinessProbe = probe
	tool.Spec.Deployment.StartupProbe = probe

	// resources
	tool.Spec.Deployment.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	r := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}}); err != nil {
			t.Fatalf("Reconcile %d failed: %v", i+1, err)
		}
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	podSpec := deployment.Spec.Template.Spec
	container := podSpec.Containers[0]

	// imagePullPolicy
	if container.ImagePullPolicy != corev1.PullAlways {
		t.Errorf("ImagePullPolicy = %q, want Always", container.ImagePullPolicy)
	}

	// envFrom
	if len(container.EnvFrom) == 0 || container.EnvFrom[0].ConfigMapRef.Name != "tool-config" {
		t.Errorf("EnvFrom not applied: %v", container.EnvFrom)
	}

	// serviceAccountName
	if podSpec.ServiceAccountName != "custom-tool-sa" {
		t.Errorf("ServiceAccountName = %q, want custom-tool-sa", podSpec.ServiceAccountName)
	}

	// volumes
	if len(podSpec.Volumes) == 0 || podSpec.Volumes[0].Name != "data" {
		t.Errorf("Volumes not applied: %v", podSpec.Volumes)
	}

	// volumeMounts
	if len(container.VolumeMounts) == 0 || container.VolumeMounts[0].MountPath != "/data" {
		t.Errorf("VolumeMounts not applied: %v", container.VolumeMounts)
	}

	// podAnnotations
	if deployment.Spec.Template.Annotations["prometheus.io/scrape"] != "true" {
		t.Errorf("PodAnnotations not applied: %v", deployment.Spec.Template.Annotations)
	}

	// podLabels — user label present, operator labels not overwritten
	if deployment.Spec.Template.Labels["custom-label"] != "value" {
		t.Errorf("PodLabels not applied: %v", deployment.Spec.Template.Labels)
	}
	if deployment.Spec.Template.Labels["app.kubernetes.io/name"] != tool.Name {
		t.Errorf("Operator labels overwritten: %v", deployment.Spec.Template.Labels)
	}

	// initContainers
	if len(podSpec.InitContainers) == 0 || podSpec.InitContainers[0].Name != "init-setup" {
		t.Errorf("InitContainers not applied: %v", podSpec.InitContainers)
	}

	// probes
	if container.LivenessProbe == nil || container.LivenessProbe.HTTPGet.Path != "/health" {
		t.Errorf("LivenessProbe not applied: %v", container.LivenessProbe)
	}
	if container.ReadinessProbe == nil || container.ReadinessProbe.HTTPGet.Path != "/health" {
		t.Errorf("ReadinessProbe not applied: %v", container.ReadinessProbe)
	}
	if container.StartupProbe == nil || container.StartupProbe.HTTPGet.Path != "/health" {
		t.Errorf("StartupProbe not applied: %v", container.StartupProbe)
	}

	// resources
	if container.Resources.Limits.Memory().Cmp(resource.MustParse("256Mi")) != 0 {
		t.Errorf("Resources not applied: %v", container.Resources)
	}
}

func TestLanguageToolController_DefaultSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("test-sec-tool", "default",
		gen.SetToolImage("test:latest"),
		gen.SetToolPort(8080),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	r := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}}); err != nil {
			t.Fatalf("Reconcile %d failed: %v", i+1, err)
		}
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	podSpec := deployment.Spec.Template.Spec

	// Operator default pod security context applied when user doesn't specify one
	if podSpec.SecurityContext == nil {
		t.Fatal("Expected pod SecurityContext to be set")
	}
	if podSpec.SecurityContext.RunAsNonRoot == nil || !*podSpec.SecurityContext.RunAsNonRoot {
		t.Error("Expected RunAsNonRoot=true on pod SecurityContext")
	}
	if podSpec.SecurityContext.RunAsUser == nil || *podSpec.SecurityContext.RunAsUser != LangopUserID {
		t.Errorf("Expected RunAsUser=%d, got %v", LangopUserID, podSpec.SecurityContext.RunAsUser)
	}

	// Container-level security context
	container := podSpec.Containers[0]
	if container.SecurityContext == nil {
		t.Fatal("Expected container SecurityContext to be set")
	}
	if container.SecurityContext.AllowPrivilegeEscalation == nil || *container.SecurityContext.AllowPrivilegeEscalation {
		t.Error("Expected AllowPrivilegeEscalation=false")
	}
	if container.SecurityContext.ReadOnlyRootFilesystem == nil || !*container.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("Expected ReadOnlyRootFilesystem=true")
	}
}

func TestLanguageToolController_UserSecurityContextOverride(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("test-sec-override", "default",
		gen.SetToolImage("test:latest"),
		gen.SetToolPort(8080),
	)
	// User provides a custom pod security context
	runAsUser := int64(2000)
	tool.Spec.Deployment.SecurityContext = &corev1.PodSecurityContext{
		RunAsUser: &runAsUser,
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	r := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}}); err != nil {
			t.Fatalf("Reconcile %d failed: %v", i+1, err)
		}
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	podSpec := deployment.Spec.Template.Spec
	if podSpec.SecurityContext == nil || podSpec.SecurityContext.RunAsUser == nil || *podSpec.SecurityContext.RunAsUser != 2000 {
		t.Errorf("Expected user-supplied SecurityContext.RunAsUser=2000, got %v", podSpec.SecurityContext)
	}
}

func TestLanguageToolController_PhaseFailedOnRegistryError(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Image uses ghcr.io but registry whitelist only allows docker.io → validation fails
	tool := &langopv1alpha1.LanguageTool{
		ObjectMeta: metav1.ObjectMeta{Name: "blocked-tool", Namespace: "default"},
		Spec: langopv1alpha1.LanguageToolSpec{
			Image: "ghcr.io/language-operator/tool:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{registries: []string{"docker.io"}},
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}}

	// Single reconcile: finalizer added then registry check fires immediately → error
	_, err := reconciler.Reconcile(ctx, req)
	if err == nil {
		t.Fatal("Expected reconcile error from registry validation, got nil")
	}

	updatedTool := &langopv1alpha1.LanguageTool{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updatedTool); err != nil {
		t.Fatalf("Failed to get tool: %v", err)
	}
	if updatedTool.Status.Phase != events.PhaseStatusFailed {
		t.Errorf("Expected phase %q after registry rejection, got %q", events.PhaseStatusFailed, updatedTool.Status.Phase)
	}

	var regCond *metav1.Condition
	for i := range updatedTool.Status.Conditions {
		if updatedTool.Status.Conditions[i].Type == langopv1alpha1.ConditionRegistryValidated {
			regCond = &updatedTool.Status.Conditions[i]
			break
		}
	}
	if regCond == nil {
		t.Fatal("Expected RegistryValidated condition to be set")
	}
	if regCond.Status != metav1.ConditionFalse {
		t.Errorf("Expected RegistryValidated status %q, got %q", metav1.ConditionFalse, regCond.Status)
	}
	if regCond.Reason != "RegistryNotAllowed" {
		t.Errorf("Expected RegistryValidated reason %q, got %q", "RegistryNotAllowed", regCond.Reason)
	}
}

func TestLanguageToolController_NetworkPolicy_FromRule(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("from-rule-tool", "default",
		gen.SetToolImage("ghcr.io/language-operator/tool:latest"),
		gen.SetToolPort(8080),
		gen.SetToolNetworkPolicies([]langopv1alpha1.NetworkRule{
			{
				From: &langopv1alpha1.NetworkPeer{
					Group: "trusted-agents",
				},
				Ports: []langopv1alpha1.NetworkPort{
					{Port: 8080},
				},
			},
		}),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	if err := reconciler.reconcileNetworkPolicy(ctx, tool); err != nil {
		t.Fatalf("reconcileNetworkPolicy failed: %v", err)
	}

	np := &networkingv1.NetworkPolicy{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, np); err != nil {
		t.Fatalf("NetworkPolicy not found: %v", err)
	}

	t.Run("policy_type_ingress_added", func(t *testing.T) {
		hasIngress := false
		for _, pt := range np.Spec.PolicyTypes {
			if pt == networkingv1.PolicyTypeIngress {
				hasIngress = true
			}
		}
		if !hasIngress {
			t.Error("expected PolicyTypeIngress to be set when From rules are present")
		}
	})

	t.Run("ingress_rule_created", func(t *testing.T) {
		if len(np.Spec.Ingress) == 0 {
			t.Fatal("expected at least one ingress rule")
		}
		rule := np.Spec.Ingress[0]
		if len(rule.From) == 0 {
			t.Fatal("expected From peers in ingress rule")
		}
		peer := rule.From[0]
		if peer.PodSelector == nil {
			t.Fatal("expected PodSelector for Group-based From rule")
		}
		if peer.PodSelector.MatchLabels[LabelKeyLangopGroup] != "trusted-agents" {
			t.Errorf("expected langop.io/group=trusted-agents, got %v", peer.PodSelector.MatchLabels)
		}
		if len(rule.Ports) == 0 || rule.Ports[0].Port.IntVal != 8080 {
			t.Error("expected port 8080 in ingress rule")
		}
	})
}

func TestLanguageToolController_Deletion(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("del-tool", "default",
		gen.SetToolImage("ghcr.io/language-operator/tool:latest"),
		gen.SetToolDeploymentMode("service"),
		gen.SetToolPort(8080),
	)
	tool.Finalizers = []string{FinalizerName}
	tool.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}}
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile returned unexpected error: %v", err)
	}

	// Finalizer should be removed
	updated := &langopv1alpha1.LanguageTool{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, updated); err == nil {
		for _, f := range updated.Finalizers {
			if f == FinalizerName {
				t.Error("Expected finalizer to be removed after deletion")
			}
		}
	}
}

func TestLanguageToolController_SidecarMode_NoRunningPod(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("my-sidecar-tool", "default",
		gen.SetToolImage("ghcr.io/test/tool:latest"),
		gen.SetToolDeploymentMode("sidecar"),
		gen.SetToolPort(8080),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Should requeue because no pod is available yet
	if result.RequeueAfter == 0 {
		t.Error("Expected non-zero RequeueAfter when no agent pod is running")
	}

	updatedTool := &langopv1alpha1.LanguageTool{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, updatedTool); err != nil {
		t.Fatalf("Failed to get updated tool: %v", err)
	}

	// AvailableTools and ToolSchemas should be empty
	if len(updatedTool.Status.AvailableTools) != 0 {
		t.Errorf("Expected no AvailableTools, got %v", updatedTool.Status.AvailableTools)
	}
	if len(updatedTool.Status.ToolSchemas) != 0 {
		t.Errorf("Expected no ToolSchemas, got %v", updatedTool.Status.ToolSchemas)
	}

	// SchemasDiscovered condition should be False with reason NoRunningAgentPod
	var schemasCond *metav1.Condition
	for i := range updatedTool.Status.Conditions {
		if updatedTool.Status.Conditions[i].Type == langopv1alpha1.ConditionSchemasDiscovered {
			schemasCond = &updatedTool.Status.Conditions[i]
			break
		}
	}
	if schemasCond == nil {
		t.Fatal("Expected SchemasDiscovered condition to be set")
	}
	if schemasCond.Status != metav1.ConditionFalse {
		t.Errorf("Expected SchemasDiscovered=False, got %s", schemasCond.Status)
	}
	if schemasCond.Reason != "NoRunningAgentPod" {
		t.Errorf("Expected reason NoRunningAgentPod, got %s", schemasCond.Reason)
	}
}

func TestLanguageToolController_SidecarMode_SchemasDiscoveredViaPod(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Start a mock MCP server that returns two tools
	mcpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := MCPResponse{
			JSONRpc: "2.0",
			ID:      1,
			Result: json.RawMessage(`{"tools":[
				{"name":"read_file","description":"Read a file"},
				{"name":"write_file","description":"Write a file"}
			]}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mcpServer.Close()

	// The fake pod's IP must match what discoverSidecarSchemas will query.
	// We use the httptest server's host:port as the pod IP:port combo by pointing the
	// pod IP to 127.0.0.1 and using the server's port.
	serverHost := mcpServer.Listener.Addr().(*net.TCPAddr).IP.String()
	serverPort := int32(mcpServer.Listener.Addr().(*net.TCPAddr).Port)

	tool := gen.LanguageTool("my-sidecar-tool", "default",
		gen.SetToolImage("ghcr.io/test/tool:latest"),
		gen.SetToolDeploymentMode("sidecar"),
		gen.SetToolPort(serverPort),
	)

	// Create a running agent pod with the sidecar container ready
	agentPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-pod-1",
			Namespace: "default",
			Labels: map[string]string{
				"langop.io/kind": "LanguageAgent",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: serverHost,
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  fmt.Sprintf("tool-%s", tool.Name),
					Ready: true,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool, agentPod).
		WithStatusSubresource(tool).
		Build()

	reconciler := &LanguageToolReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		RegistryManager: &mockRegistryManager{},
		// Use the test server's transport so requests reach the httptest server
		HTTPClient: mcpServer.Client(),
	}

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Should NOT requeue because schemas were discovered
	if result.RequeueAfter != 0 {
		t.Errorf("Expected zero RequeueAfter after successful schema discovery, got %v", result.RequeueAfter)
	}

	updatedTool := &langopv1alpha1.LanguageTool{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}, updatedTool); err != nil {
		t.Fatalf("Failed to get updated tool: %v", err)
	}

	if len(updatedTool.Status.AvailableTools) != 2 {
		t.Errorf("Expected 2 AvailableTools, got %v", updatedTool.Status.AvailableTools)
	}
	if len(updatedTool.Status.ToolSchemas) != 2 {
		t.Errorf("Expected 2 ToolSchemas, got %v", updatedTool.Status.ToolSchemas)
	}

	// SchemasDiscovered condition should be True
	var schemasCond *metav1.Condition
	for i := range updatedTool.Status.Conditions {
		if updatedTool.Status.Conditions[i].Type == langopv1alpha1.ConditionSchemasDiscovered {
			schemasCond = &updatedTool.Status.Conditions[i]
			break
		}
	}
	if schemasCond == nil {
		t.Fatal("Expected SchemasDiscovered condition to be set")
	}
	if schemasCond.Status != metav1.ConditionTrue {
		t.Errorf("Expected SchemasDiscovered=True, got %s", schemasCond.Status)
	}
}

func TestLanguageToolController_Reconcile_NetworkPolicyConditionTrue(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("np-cond-tool", "default",
		gen.SetToolImage("ghcr.io/language-operator/tool:latest"),
		gen.SetToolDeploymentMode("service"),
		gen.SetToolPort(8080),
	)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		Build()

	r := &LanguageToolReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		RegistryManager:         &mockRegistryManager{},
		NetworkIsolationEnabled: true,
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}}

	// First reconcile: adds finalizer.
	_, err := r.Reconcile(ctx, req)
	require.NoError(t, err)

	// Second reconcile: creates resources and sets conditions.
	_, err = r.Reconcile(ctx, req)
	require.NoError(t, err)

	updated := &langopv1alpha1.LanguageTool{}
	require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, updated))

	var cond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionNetworkPolicyReady {
			cond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, cond, "expected ConditionNetworkPolicyReady to be set")
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, "NetworkPolicyReady", cond.Reason)
}

func TestLanguageToolController_Reconcile_NetworkPolicyConditionError(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	tool := gen.LanguageTool("np-err-tool", "default",
		gen.SetToolImage("ghcr.io/language-operator/tool:latest"),
		gen.SetToolDeploymentMode("service"),
		gen.SetToolPort(8080),
	)
	tool.Finalizers = []string{FinalizerName}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gen.ReadyCluster("default"), tool).
		WithStatusSubresource(tool).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*networkingv1.NetworkPolicy); ok {
					return fmt.Errorf("injected networkpolicy error")
				}
				return c.Create(ctx, obj, opts...)
			},
		}).
		Build()

	r := &LanguageToolReconciler{
		Client:                  fakeClient,
		Scheme:                  scheme,
		RegistryManager:         &mockRegistryManager{},
		NetworkIsolationEnabled: true,
	}

	ctx := context.Background()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: tool.Name, Namespace: tool.Namespace}}

	// Finalizer already set — single reconcile hits resource creation.
	_, err := r.Reconcile(ctx, req)
	require.Error(t, err)

	updated := &langopv1alpha1.LanguageTool{}
	require.NoError(t, fakeClient.Get(ctx, req.NamespacedName, updated))

	assert.Equal(t, events.PhaseStatusFailed, updated.Status.Phase)

	var cond *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == langopv1alpha1.ConditionReady {
			cond = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, cond, "expected ConditionReady to be set")
	assert.Equal(t, metav1.ConditionFalse, cond.Status)
	assert.Equal(t, "NetworkPolicyError", cond.Reason)
}
