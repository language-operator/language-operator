package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	langopv1alpha1 "github.com/based/language-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	// Timeout for waiting on resources
	DefaultTimeout = 2 * time.Minute
	// Poll interval when waiting
	PollInterval = 2 * time.Second
)

// TestEnvironment encapsulates the test environment
type TestEnvironment struct {
	cfg       *rest.Config
	k8sClient client.Client
	clientset *kubernetes.Clientset
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
}

// SetupTestEnvironment creates a new test environment with envtest
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	ctx, cancel := context.WithCancel(context.Background())

	// Register our custom types
	err := langopv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Fatalf("Failed to add langop scheme: %v", err)
	}

	// Create test environment
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{"../../src/config/crd/bases"},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("Failed to start test environment: %v", err)
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatalf("Failed to create k8s client: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create clientset: %v", err)
	}

	return &TestEnvironment{
		cfg:       cfg,
		k8sClient: k8sClient,
		clientset: clientset,
		testEnv:   testEnv,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Teardown cleans up the test environment
func (e *TestEnvironment) Teardown(t *testing.T) {
	e.cancel()
	if err := e.testEnv.Stop(); err != nil {
		t.Errorf("Failed to stop test environment: %v", err)
	}
}

// CreateNamespace creates a test namespace
func (e *TestEnvironment) CreateNamespace(t *testing.T, name string) *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := e.k8sClient.Create(e.ctx, ns)
	if err != nil {
		t.Fatalf("Failed to create namespace %s: %v", name, err)
	}

	return ns
}

// DeleteNamespace deletes a test namespace
func (e *TestEnvironment) DeleteNamespace(t *testing.T, name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := e.k8sClient.Delete(e.ctx, ns)
	if err != nil {
		t.Logf("Failed to delete namespace %s: %v", name, err)
	}
}

// CreateLanguageAgent creates a LanguageAgent resource
func (e *TestEnvironment) CreateLanguageAgent(t *testing.T, agent *langopv1alpha1.LanguageAgent) {
	err := e.k8sClient.Create(e.ctx, agent)
	if err != nil {
		t.Fatalf("Failed to create LanguageAgent %s/%s: %v", agent.Namespace, agent.Name, err)
	}
}

// GetLanguageAgent retrieves a LanguageAgent resource
func (e *TestEnvironment) GetLanguageAgent(t *testing.T, namespace, name string) *langopv1alpha1.LanguageAgent {
	agent := &langopv1alpha1.LanguageAgent{}
	err := e.k8sClient.Get(e.ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, agent)

	if err != nil {
		t.Fatalf("Failed to get LanguageAgent %s/%s: %v", namespace, name, err)
	}

	return agent
}

// WaitForCondition waits for a specific condition on a LanguageAgent
func (e *TestEnvironment) WaitForCondition(t *testing.T, namespace, name string, conditionType string, status metav1.ConditionStatus) error {
	return wait.PollImmediate(PollInterval, DefaultTimeout, func() (bool, error) {
		agent := &langopv1alpha1.LanguageAgent{}
		err := e.k8sClient.Get(e.ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, agent)

		if err != nil {
			return false, err
		}

		for _, condition := range agent.Status.Conditions {
			if condition.Type == conditionType && condition.Status == status {
				return true, nil
			}
		}

		return false, nil
	})
}

// WaitForDeployment waits for a deployment to be ready
func (e *TestEnvironment) WaitForDeployment(t *testing.T, namespace, name string) error {
	return wait.PollImmediate(PollInterval, DefaultTimeout, func() (bool, error) {
		deployment := &appsv1.Deployment{}
		err := e.k8sClient.Get(e.ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, deployment)

		if err != nil {
			return false, err
		}

		if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
			return true, nil
		}

		return false, nil
	})
}

// WaitForPod waits for a pod to be in Running phase
func (e *TestEnvironment) WaitForPod(t *testing.T, namespace, labelSelector string) (*corev1.Pod, error) {
	var pod *corev1.Pod

	err := wait.PollImmediate(PollInterval, DefaultTimeout, func() (bool, error) {
		podList := &corev1.PodList{}
		err := e.k8sClient.List(e.ctx, podList, client.InNamespace(namespace), client.MatchingLabels(map[string]string{
			"app": labelSelector,
		}))

		if err != nil {
			return false, err
		}

		if len(podList.Items) == 0 {
			return false, nil
		}

		// Take the first pod
		pod = &podList.Items[0]
		return pod.Status.Phase == corev1.PodRunning, nil
	})

	return pod, err
}

// GetPodLogs retrieves logs from a pod
func (e *TestEnvironment) GetPodLogs(t *testing.T, namespace, podName string) (string, error) {
	req := e.clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{})
	logs, err := req.DoRaw(e.ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}

	return string(logs), nil
}

// CreateConfigMap creates a ConfigMap
func (e *TestEnvironment) CreateConfigMap(t *testing.T, cm *corev1.ConfigMap) {
	err := e.k8sClient.Create(e.ctx, cm)
	if err != nil {
		t.Fatalf("Failed to create ConfigMap %s/%s: %v", cm.Namespace, cm.Name, err)
	}
}

// GetConfigMap retrieves a ConfigMap
func (e *TestEnvironment) GetConfigMap(t *testing.T, namespace, name string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	err := e.k8sClient.Get(e.ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cm)

	if err != nil {
		t.Fatalf("Failed to get ConfigMap %s/%s: %v", namespace, name, err)
	}

	return cm
}

// WaitForConfigMap waits for a ConfigMap to exist
func (e *TestEnvironment) WaitForConfigMap(t *testing.T, namespace, name string) error {
	return wait.PollImmediate(PollInterval, DefaultTimeout, func() (bool, error) {
		cm := &corev1.ConfigMap{}
		err := e.k8sClient.Get(e.ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, cm)

		if err != nil {
			return false, nil
		}

		return true, nil
	})
}

// NewTestLanguageAgent creates a LanguageAgent with required fields populated for testing
func NewTestLanguageAgent(namespace, name string, spec langopv1alpha1.LanguageAgentSpec) *langopv1alpha1.LanguageAgent {
	// Set required fields if not provided
	if spec.Image == "" {
		spec.Image = "git.theryans.io/language-operator/agent:latest"
	}
	if len(spec.ModelRefs) == 0 {
		spec.ModelRefs = []langopv1alpha1.ModelReference{
			{Name: "test-model"},
		}
	}

	return &langopv1alpha1.LanguageAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
	}
}
