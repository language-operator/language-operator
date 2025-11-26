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
	"k8s.io/apimachinery/pkg/api/errors"
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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

	// Verify CronJob was created
	cronJob := &batchv1.CronJob{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, cronJob)
	if err != nil {
		t.Fatalf("Expected CronJob to exist for scheduled agent, but got error: %v", err)
	}

	// Verify CronJob schedule
	if cronJob.Spec.Schedule != agent.Spec.Schedule {
		t.Errorf("Expected schedule '%s', got '%s'", agent.Spec.Schedule, cronJob.Spec.Schedule)
	}

	// Verify CronJob has correct image
	if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers))
	}
	if cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image != agent.Spec.Image {
		t.Errorf("Expected image '%s', got '%s'", agent.Spec.Image, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		"tmp":         "/tmp",
		"ruby-bundle": "/home/langop/.bundle",
		"ruby-gem":    "/home/langop/.gem",
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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

	// Verify CronJob was created
	cronJob := &batchv1.CronJob{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, cronJob)
	if err != nil {
		t.Fatalf("Expected CronJob to exist, but got error: %v", err)
	}

	// Verify Pod security context
	podSec := cronJob.Spec.JobTemplate.Spec.Template.Spec.SecurityContext
	if podSec == nil {
		t.Fatal("Pod SecurityContext is nil")
	}

	if podSec.RunAsNonRoot == nil || !*podSec.RunAsNonRoot {
		t.Error("Expected RunAsNonRoot to be true")
	}

	if podSec.RunAsUser == nil || *podSec.RunAsUser != 1000 {
		t.Errorf("Expected RunAsUser to be 1000, got %v", podSec.RunAsUser)
	}

	// Verify container security context
	if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("No containers found in cronjob")
	}

	containerSec := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].SecurityContext
	if containerSec == nil {
		t.Fatal("Container SecurityContext is nil")
	}

	if containerSec.ReadOnlyRootFilesystem == nil || !*containerSec.ReadOnlyRootFilesystem {
		t.Error("Expected ReadOnlyRootFilesystem to be true")
	}

	if containerSec.Capabilities == nil || len(containerSec.Capabilities.Drop) != 1 || containerSec.Capabilities.Drop[0] != "ALL" {
		t.Error("Expected capabilities to drop ALL")
	}
}

func TestLanguageAgentController_OptimizedAnnotationSkipsSynthesis(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)

	agent := &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: langopv1alpha1.LanguageAgentSpec{
			Image:        "ghcr.io/language-operator/agent:latest",
			Instructions: "Do something",
			ModelRefs: []langopv1alpha1.ModelReference{
				{Name: "test-model"},
			},
		},
	}

	// Create a code ConfigMap with the optimized annotation
	codeConfigMapName := GenerateConfigMapName(agent.Name, "code")
	optimizedConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      codeConfigMapName,
			Namespace: agent.Namespace,
			Annotations: map[string]string{
				"langop.io/optimized":      "true",
				"langop.io/optimized-at":   "2025-11-21T16:50:00Z",
				"langop.io/optimized-task": "read_existing_story",
			},
		},
		Data: map[string]string{
			"agent.rb": "# Optimized code that should not be overwritten",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent, optimizedConfigMap).
		WithStatusSubresource(agent).
		Build()

	reconciler := &LanguageAgentReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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

	// Verify the ConfigMap still has the optimized code (not overwritten)
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      codeConfigMapName,
		Namespace: agent.Namespace,
	}, cm)
	if err != nil {
		t.Fatalf("Expected code ConfigMap to exist, but got error: %v", err)
	}

	// The optimized annotation should still be present
	if cm.Annotations["langop.io/optimized"] != "true" {
		t.Error("Expected langop.io/optimized annotation to be preserved")
	}

	// The original data should be preserved
	if cm.Data["agent.rb"] != "# Optimized code that should not be overwritten" {
		t.Errorf("Expected optimized code to be preserved, got: %s", cm.Data["agent.rb"])
	}

	// Owner reference should be set for proper garbage collection
	if len(cm.OwnerReferences) == 0 {
		t.Error("Expected owner reference to be set on optimized ConfigMap")
	} else {
		ownerRef := cm.OwnerReferences[0]
		if ownerRef.Name != agent.Name {
			t.Errorf("Expected owner reference name to be %s, got %s", agent.Name, ownerRef.Name)
		}
		if ownerRef.Kind != "LanguageAgent" {
			t.Errorf("Expected owner reference kind to be LanguageAgent, got %s", ownerRef.Kind)
		}
		if !*ownerRef.Controller {
			t.Error("Expected owner reference to have controller=true")
		}
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
		Client:   fakeClient,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Recorder: &record.FakeRecorder{},
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
