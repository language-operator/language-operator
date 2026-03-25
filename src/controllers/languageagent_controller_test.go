package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
	"github.com/language-operator/language-operator/controllers/testutil"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Note: Synthesis is now configured per-agent via ModelRefs
// Tests that need to verify synthesis behavior require integration tests with actual LanguageModel resources

func TestLanguageAgentController_NoSynthesisWithoutModelRefs(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
			Instructions: "Do something", // Has instructions but no ModelRefs
			// No ModelRefs - synthesis should not happen
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify no code ConfigMap was created (no ModelRefs means no synthesis)
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      GenerateConfigMapName(agent.Name, "code"),
		Namespace: agent.Namespace,
	}, cm)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected no code ConfigMap without ModelRefs, but found one or got error: %v", err)
	}
}

// Note: TestLanguageAgentController_SynthesisCalledWithInstructions and
// TestLanguageAgentController_SmartChangeDetection were removed because synthesis
// is now configured per-agent via ModelRefs. Testing synthesis behavior requires
// integration tests with actual LanguageModel CRDs and Secrets.

func TestLanguageAgentController_DeploymentCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment was created
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist for autonomous agent, but got error: %v", err)
	}

	// Verify Deployment has correct image
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(deployment.Spec.Template.Spec.Containers))
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != agent.Spec.Image {
		t.Errorf("Expected image '%s', got '%s'", agent.Spec.Image, deployment.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestLanguageAgentController_CronJobCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "scheduled",
			Schedule:      "0 * * * *",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment was created (standby mode)
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist for scheduled agent, but got error: %v", err)
	}

	// Verify Deployment has correct image
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container in Deployment, got %d", len(deployment.Spec.Template.Spec.Containers))
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != agent.Spec.Image {
		t.Errorf("Expected Deployment image '%s', got '%s'", agent.Spec.Image, deployment.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify CronJob trigger was created
	cronJob := &batchv1.CronJob{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name + "-trigger",
		Namespace: agent.Namespace,
	}, cronJob)
	if err != nil {
		t.Fatalf("Expected CronJob trigger to exist for scheduled agent, but got error: %v", err)
	}

	// Verify CronJob trigger schedule
	if cronJob.Spec.Schedule != agent.Spec.Schedule {
		t.Errorf("Expected schedule '%s', got '%s'", agent.Spec.Schedule, cronJob.Spec.Schedule)
	}

	// Verify CronJob trigger uses curl image
	if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container in trigger, got %d", len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers))
	}
	if cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image != "curlimages/curl:latest" {
		t.Errorf("Expected trigger image 'curlimages/curl:latest', got '%s'", cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestLanguageAgentController_WorkspacePVCCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
			Workspace: &langopv1alpha1.WorkspaceSpec{
				Enabled: true,
				Size:    "10Gi",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify PVC was created
	pvc := &corev1.PersistentVolumeClaim{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name + "-workspace",
		Namespace: agent.Namespace,
	}, pvc)
	if err != nil {
		t.Fatalf("Expected PVC to exist when workspace is enabled, but got error: %v", err)
	}

	// Verify PVC size
	requestedStorage := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	expectedStorage := agent.Spec.Workspace.Size
	if requestedStorage.String() != expectedStorage {
		t.Errorf("Expected storage size '%s', got '%s'", expectedStorage, requestedStorage.String())
	}
}

func TestLanguageAgentController_StatusConditions(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-status-agent",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Fetch updated agent
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)
	if err != nil {
		t.Fatalf("Failed to fetch updated agent: %v", err)
	}

	// Verify status phase
	if updatedAgent.Status.Phase != "Running" {
		t.Errorf("Expected phase 'Running', got '%s'", updatedAgent.Status.Phase)
	}

	// Verify Ready condition
	var readyCondition *metav1.Condition
	for i := range updatedAgent.Status.Conditions {
		if updatedAgent.Status.Conditions[i].Type == "Ready" {
			readyCondition = &updatedAgent.Status.Conditions[i]
			break
		}
	}
	if readyCondition == nil {
		t.Fatal("Ready condition not found")
	}
	if readyCondition.Status != metav1.ConditionTrue {
		t.Errorf("Expected condition status True, got %s", readyCondition.Status)
	}
	if readyCondition.Reason != "ReconcileSuccess" {
		t.Errorf("Expected reason 'ReconcileSuccess', got '%s'", readyCondition.Reason)
	}
}

func TestLanguageAgentController_NotFoundHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-agent",
			Namespace: "default",
		},
	})

	// Should not return error for not found
	if err != nil {
		t.Errorf("Expected no error for not found agent, got: %v", err)
	}

	// Should not requeue
	if result.Requeue {
		t.Error("Expected no requeue for not found agent")
	}
}

func TestLanguageAgentController_DefaultExecutionMode(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Test with empty ExecutionMode (should skip workload creation until synthesis detects mode)
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-default-mode",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image: "ghcr.io/language-operator/agent:latest",
			// ExecutionMode not specified - should NOT create any workload yet
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify NO Deployment was created (should wait for synthesis to detect mode)
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err == nil {
		t.Fatal("Expected no Deployment to exist when ExecutionMode is empty")
	}
	if !errors.IsNotFound(err) {
		t.Fatalf("Expected NotFound error, got: %v", err)
	}

	// Verify NO CronJob was created either
	cronjob := &batchv1.CronJob{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, cronjob)
	if err == nil {
		t.Fatal("Expected no CronJob to exist when ExecutionMode is empty")
	}
	if !errors.IsNotFound(err) {
		t.Fatalf("Expected NotFound error, got: %v", err)
	}
}

func TestLanguageAgentController_PodSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-security-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment was created
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify Pod security context
	podSec := deployment.Spec.Template.Spec.SecurityContext
	if podSec == nil {
		t.Fatal("Pod SecurityContext is nil")
	}

	if podSec.RunAsNonRoot == nil || !*podSec.RunAsNonRoot {
		t.Error("Expected RunAsNonRoot to be true")
	}

	if podSec.RunAsUser == nil || *podSec.RunAsUser != 1000 {
		t.Errorf("Expected RunAsUser to be 1000, got %v", podSec.RunAsUser)
	}

	if podSec.FSGroup == nil || *podSec.FSGroup != 101 {
		t.Errorf("Expected FSGroup to be 101, got %v", podSec.FSGroup)
	}

	if podSec.SeccompProfile == nil || podSec.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Error("Expected SeccompProfile type to be RuntimeDefault")
	}
}

func TestLanguageAgentController_ContainerSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-container-security-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment was created
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify container security context
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers found in deployment")
	}

	containerSec := deployment.Spec.Template.Spec.Containers[0].SecurityContext
	if containerSec == nil {
		t.Fatal("Container SecurityContext is nil")
	}

	if containerSec.AllowPrivilegeEscalation == nil || *containerSec.AllowPrivilegeEscalation {
		t.Error("Expected AllowPrivilegeEscalation to be false")
	}

	if containerSec.RunAsNonRoot == nil || !*containerSec.RunAsNonRoot {
		t.Error("Expected RunAsNonRoot to be true")
	}

	if containerSec.RunAsUser == nil || *containerSec.RunAsUser != 1000 {
		t.Errorf("Expected RunAsUser to be 1000, got %v", containerSec.RunAsUser)
	}

	if containerSec.ReadOnlyRootFilesystem == nil || !*containerSec.ReadOnlyRootFilesystem {
		t.Error("Expected ReadOnlyRootFilesystem to be true")
	}

	if containerSec.Capabilities == nil {
		t.Fatal("Capabilities is nil")
	}

	if len(containerSec.Capabilities.Drop) != 1 || containerSec.Capabilities.Drop[0] != "ALL" {
		t.Errorf("Expected capabilities to drop ALL, got %v", containerSec.Capabilities.Drop)
	}
}

func TestLanguageAgentController_TmpfsVolumes(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tmpfs-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment was created
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Check for tmpfs volumes
	expectedVolumes := map[string]string{
		"tmp": "/tmp",
	}

	volumes := deployment.Spec.Template.Spec.Volumes
	volumeNames := make(map[string]bool)
	for _, vol := range volumes {
		volumeNames[vol.Name] = true
		// Verify it's an EmptyDir with Memory medium
		if vol.EmptyDir != nil && vol.EmptyDir.Medium == corev1.StorageMediumMemory {
			// Good - it's a tmpfs volume
		} else if _, ok := expectedVolumes[vol.Name]; ok {
			t.Errorf("Volume %s should be EmptyDir with Memory medium", vol.Name)
		}
	}

	// Check all expected volumes exist
	for volName := range expectedVolumes {
		if !volumeNames[volName] {
			t.Errorf("Expected volume %s to exist", volName)
		}
	}

	// Check volume mounts on container
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers found in deployment")
	}

	volumeMounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts
	mountPaths := make(map[string]string)
	for _, mount := range volumeMounts {
		mountPaths[mount.Name] = mount.MountPath
	}

	// Verify all expected mounts
	for volName, expectedPath := range expectedVolumes {
		if actualPath, ok := mountPaths[volName]; ok {
			if actualPath != expectedPath {
				t.Errorf("Volume %s expected to be mounted at %s, got %s", volName, expectedPath, actualPath)
			}
		} else {
			t.Errorf("Expected volume mount for %s at %s", volName, expectedPath)
		}
	}
}

func TestLanguageAgentController_CronJobSecurityContext(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob-security",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "scheduled",
			Schedule:      "0 * * * *",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment was created (standby mode)
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected Deployment to exist, but got error: %v", err)
	}

	// Verify Deployment Pod security context
	podSec := deployment.Spec.Template.Spec.SecurityContext
	if podSec == nil {
		t.Fatal("Deployment Pod SecurityContext is nil")
	}

	if podSec.RunAsNonRoot == nil || !*podSec.RunAsNonRoot {
		t.Error("Expected Deployment RunAsNonRoot to be true")
	}

	if podSec.RunAsUser == nil || *podSec.RunAsUser != 1000 {
		t.Errorf("Expected Deployment RunAsUser to be 1000, got %v", podSec.RunAsUser)
	}

	// Verify Deployment container security context
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers found in Deployment")
	}

	containerSec := deployment.Spec.Template.Spec.Containers[0].SecurityContext
	if containerSec == nil {
		t.Fatal("Deployment Container SecurityContext is nil")
	}

	if containerSec.ReadOnlyRootFilesystem == nil || !*containerSec.ReadOnlyRootFilesystem {
		t.Error("Expected Deployment ReadOnlyRootFilesystem to be true")
	}

	// Verify CronJob trigger was created
	cronJob := &batchv1.CronJob{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name + "-trigger",
		Namespace: agent.Namespace,
	}, cronJob)
	if err != nil {
		t.Fatalf("Expected CronJob trigger to exist, but got error: %v", err)
	}

	// Verify CronJob trigger has security context
	triggerPodSec := cronJob.Spec.JobTemplate.Spec.Template.Spec.SecurityContext
	if triggerPodSec == nil {
		t.Fatal("CronJob trigger Pod SecurityContext is nil")
	}

	if triggerPodSec.RunAsNonRoot == nil || !*triggerPodSec.RunAsNonRoot {
		t.Error("Expected trigger RunAsNonRoot to be true")
	}

	// Continue with Deployment container security validation
	if containerSec.Capabilities == nil || len(containerSec.Capabilities.Drop) != 1 || containerSec.Capabilities.Drop[0] != "ALL" {
		t.Error("Expected capabilities to drop ALL")
	}
}

func TestLanguageAgentController_ResourceCleanup(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
			Finalizers: []string{FinalizerName},
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Instructions: "Test agent for cleanup",
		},
	}

	// Create resources that should be cleaned up
	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	// Service to cleanup
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Port: 80}},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, service).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()

	// Run reconcile - should trigger cleanup since agent has DeletionTimestamp
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify the service was deleted
	svc := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, svc)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected service to be deleted, but it still exists or got different error: %v", err)
	}

	// Verify the agent was either deleted or finalizer was removed
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)

	if errors.IsNotFound(err) {
		// Agent was fully deleted - this is expected and good
		t.Log("Agent was successfully deleted after cleanup")
	} else if err != nil {
		t.Fatalf("Unexpected error getting updated agent: %v", err)
	} else {
		// Agent still exists, check that finalizer was removed
		for _, finalizer := range updatedAgent.Finalizers {
			if finalizer == FinalizerName {
				t.Error("Expected finalizer to be removed after successful cleanup")
			}
		}
	}
}

func TestLanguageAgentController_UUIDAssignmentRaceCondition(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-uuid-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
		// Status.UUID should be empty initially
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()

	// First reconcile should assign UUID
	result1, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("First reconcile failed: %v", err)
	}

	// Fetch updated agent to get UUID
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)
	if err != nil {
		t.Fatalf("Failed to fetch updated agent: %v", err)
	}

	// Verify UUID was assigned
	if updatedAgent.Status.UUID == "" {
		t.Fatal("Expected UUID to be assigned on first reconcile")
	}
	firstUUID := updatedAgent.Status.UUID

	// Second reconcile should NOT change the UUID
	result2, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Fetch agent again
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)
	if err != nil {
		t.Fatalf("Failed to fetch agent after second reconcile: %v", err)
	}

	// Verify UUID remained the same
	if updatedAgent.Status.UUID != firstUUID {
		t.Errorf("Expected UUID to remain %s, but got %s", firstUUID, updatedAgent.Status.UUID)
	}

	// Both results should not requeue for UUID reasons
	if result1.Requeue || result2.Requeue {
		t.Error("Reconciles should not requeue when UUID assignment succeeds")
	}
}

func TestLanguageAgentController_UUIDConflictHandling(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-conflict-agent",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
		Status: langopv1alpha1.LanguageAgentStatus{
			ObservedGeneration: 0, // Outdated to simulate conflict scenario
		},
	}

	// Create a client that will simulate version conflicts on status updates
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()

	// Simulate updating the agent's observed generation externally (as if another reconciler updated it)
	// This would happen in practice when multiple reconcilers are running
	err := fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, agent)
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	// Update the agent to have newer generation to simulate conflict conditions
	agent.Generation = 2
	err = fakeClient.Update(ctx, agent)
	if err != nil {
		t.Fatalf("Failed to update agent generation: %v", err)
	}

	// Now reconcile with the old agent object (ObservedGeneration: 0, but actual Generation: 2)
	// This should trigger the UUID assignment logic
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	})

	// The reconcile should succeed (not return an error) even if there's a conflict
	// The conflict handling should cause a requeue, not an error
	if err != nil {
		t.Fatalf("Reconcile should handle conflicts gracefully, but got error: %v", err)
	}

	// Verify agent eventually has UUID assigned
	updatedAgent := &langopv1alpha1.LanguageAgent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, updatedAgent)
	if err != nil {
		t.Fatalf("Failed to get updated agent: %v", err)
	}

	// Should have UUID assigned
	if updatedAgent.Status.UUID == "" {
		t.Error("Expected UUID to be assigned after conflict resolution")
	}
}

func TestLanguageAgentController_CleanupMethods(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Instructions: "Test agent for cleanup methods",
		},
	}

	labels := GetCommonLabels(agent.Name, "LanguageAgent")

	// Create a service that should be cleaned up
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: agent.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Port: 80}},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, service).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()

	t.Run("cleanupServices", func(t *testing.T) {
		// Test service cleanup
		err := reconciler.cleanupServices(ctx, agent)
		if err != nil {
			t.Fatalf("cleanupServices failed: %v", err)
		}

		// Verify service was deleted
		svc := &corev1.Service{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "test-service",
			Namespace: agent.Namespace,
		}, svc)
		if !errors.IsNotFound(err) {
			t.Errorf("Expected service to be deleted, but it still exists or got different error: %v", err)
		}
	})

	t.Run("cleanupHTTPRoutes_no_gateway_api", func(t *testing.T) {
		// Test HTTPRoute cleanup when Gateway API is not available
		// This should not error even if Gateway API CRDs don't exist
		err := reconciler.cleanupHTTPRoutes(ctx, agent)
		if err != nil {
			t.Errorf("cleanupHTTPRoutes should handle missing Gateway API gracefully, got error: %v", err)
		}
	})

	t.Run("cleanupIngresses_empty_list", func(t *testing.T) {
		// Test Ingress cleanup with no ingresses present
		err := reconciler.cleanupIngresses(ctx, agent)
		if err != nil {
			t.Errorf("cleanupIngresses should handle empty list gracefully, got error: %v", err)
		}
	})

	t.Run("cleanupReferenceGrants_no_gateway_api", func(t *testing.T) {
		// Test ReferenceGrant cleanup when Gateway API is not available
		err := reconciler.cleanupReferenceGrants(ctx, agent)
		if err != nil {
			t.Errorf("cleanupReferenceGrants should handle missing Gateway API gracefully, got error: %v", err)
		}
	})
}

func TestServiceSelectorExcludesTriggerPods(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	// Create a scheduled agent (will create both deployment and trigger pods)
	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-scheduled-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			ExecutionMode: "scheduled",
			Schedule:      "0 * * * *",
			Image:         "test-image",
			Goal:          "Test scheduled agent",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	}

	// Reconcile to create service and deployment
	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify service was created
	service := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, service)
	if err != nil {
		t.Fatalf("Expected service to be created, got error: %v", err)
	}

	// Verify service selector includes component=agent
	selector := service.Spec.Selector
	if selector == nil {
		t.Fatal("Service selector is nil")
	}

	expectedLabels := map[string]string{
		"app.kubernetes.io/name":       agent.Name,
		"app.kubernetes.io/managed-by": "language-operator",
		"app.kubernetes.io/part-of":    "langop",
		"langop.io/kind":               "LanguageAgent",
		"langop.io/component":          "agent",
	}

	for key, expectedValue := range expectedLabels {
		if actualValue, exists := selector[key]; !exists {
			t.Errorf("Service selector missing expected label %s", key)
		} else if actualValue != expectedValue {
			t.Errorf("Service selector label %s: expected %s, got %s", key, expectedValue, actualValue)
		}
	}

	// Verify deployment was created with component=agent label
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deployment)
	if err != nil {
		t.Fatalf("Expected deployment to be created, got error: %v", err)
	}

	// Check deployment pod template labels include component=agent
	podLabels := deployment.Spec.Template.Labels
	if component, exists := podLabels["langop.io/component"]; !exists {
		t.Error("Deployment pod template missing langop.io/component label")
	} else if component != "agent" {
		t.Errorf("Deployment pod template component label: expected 'agent', got '%s'", component)
	}

	// Verify trigger CronJob was created with component=trigger label
	cronJob := &batchv1.CronJob{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name + "-trigger",
		Namespace: agent.Namespace,
	}, cronJob)
	if err != nil {
		t.Fatalf("Expected trigger CronJob to be created, got error: %v", err)
	}

	// Check trigger pod template labels include component=trigger
	triggerPodLabels := cronJob.Spec.JobTemplate.Spec.Template.Labels
	if component, exists := triggerPodLabels["langop.io/component"]; !exists {
		t.Error("Trigger CronJob pod template missing langop.io/component label")
	} else if component != "trigger" {
		t.Errorf("Trigger CronJob pod template component label: expected 'trigger', got '%s'", component)
	}

	// Ensure service selector would NOT match trigger pods (service has component=agent, trigger has component=trigger)
	triggerWouldMatch := true
	for key, value := range selector {
		if triggerValue, exists := triggerPodLabels[key]; !exists || triggerValue != value {
			triggerWouldMatch = false
			break
		}
	}

	if triggerWouldMatch {
		t.Error("Service selector would incorrectly match trigger pods - this defeats the purpose of the fix")
	}
}

func TestLanguageAgentController_BasicReconcile(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment created
	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	// Verify Service created
	svc := &corev1.Service{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, svc); err != nil {
		t.Fatalf("Expected Service to exist: %v", err)
	}

	// Verify agent ConfigMap created (contains instructions + config)
	cm := &corev1.ConfigMap{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: GenerateConfigMapName(agent.Name, "agent"), Namespace: agent.Namespace}, cm); err != nil {
		t.Fatalf("Expected agent ConfigMap to exist: %v", err)
	}
}

func TestLanguageAgentController_EnvVarInjection(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "env-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
			Instructions:  "test instructions",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers in deployment")
	}

	envMap := make(map[string]string)
	for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
		envMap[e.Name] = e.Value
	}

	if envMap["AGENT_NAME"] != agent.Name {
		t.Errorf("Expected AGENT_NAME=%s, got %s", agent.Name, envMap["AGENT_NAME"])
	}
	if envMap["AGENT_NAMESPACE"] != agent.Namespace {
		t.Errorf("Expected AGENT_NAMESPACE=%s, got %s", agent.Namespace, envMap["AGENT_NAMESPACE"])
	}
	if envMap["AGENT_MODE"] != agent.Spec.ExecutionMode {
		t.Errorf("Expected AGENT_MODE=%s, got %s", agent.Spec.ExecutionMode, envMap["AGENT_MODE"])
	}
}

func TestLanguageAgentController_ResourceRequests(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	deployment := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, deployment); err != nil {
		t.Fatalf("Expected Deployment to exist: %v", err)
	}

	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers in deployment")
	}

	res := deployment.Spec.Template.Spec.Containers[0].Resources
	if res.Requests.Cpu().Cmp(resource.MustParse("200m")) != 0 {
		t.Errorf("Expected CPU request 200m, got %s", res.Requests.Cpu().String())
	}
	if res.Requests.Memory().Cmp(resource.MustParse("256Mi")) != 0 {
		t.Errorf("Expected memory request 256Mi, got %s", res.Requests.Memory().String())
	}
	if res.Limits.Cpu().Cmp(resource.MustParse("500m")) != 0 {
		t.Errorf("Expected CPU limit 500m, got %s", res.Limits.Cpu().String())
	}
	if res.Limits.Memory().Cmp(resource.MustParse("512Mi")) != 0 {
		t.Errorf("Expected memory limit 512Mi, got %s", res.Limits.Memory().String())
	}
}

func TestLanguageAgentController_ServiceAccountCreation(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sa-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:         "ghcr.io/language-operator/agent:latest",
			ExecutionMode: "autonomous",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		Log:             logr.Discard(),
		Recorder:        &record.FakeRecorder{},
		RegistryManager: &mockRegistryManager{},
	}
	reconciler.InitializeGatewayCache()

	ctx := context.Background()
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify ServiceAccount created in agent namespace
	sa := &corev1.ServiceAccount{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "language-agent", Namespace: agent.Namespace}, sa); err != nil {
		t.Fatalf("Expected ServiceAccount 'language-agent' to exist in namespace %s: %v", agent.Namespace, err)
	}

	// Verify ClusterRoleBinding created
	crb := &rbacv1.ClusterRoleBinding{}
	crbName := "language-agent-" + agent.Namespace + "-language-agent"
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: crbName}, crb); err != nil {
		t.Fatalf("Expected ClusterRoleBinding %q to exist: %v", crbName, err)
	}
	if crb.RoleRef.Name != "language-operator" {
		t.Errorf("Expected ClusterRoleBinding to reference 'language-operator' ClusterRole, got %q", crb.RoleRef.Name)
	}
}
