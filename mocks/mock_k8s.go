package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/your-org/kargo-bootstrap/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

// MockK8sClient implements a mock Kubernetes client for testing
type MockK8sClient struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
	config        *rest.Config
	namespaces    map[string]*corev1.Namespace
	mutex         sync.RWMutex
}

// NewMockK8sClient creates a new mock Kubernetes client
func NewMockK8sClient() *MockK8sClient {
	fakeClientset := fake.NewSimpleClientset()

	return &MockK8sClient{
		clientset:     fakeClientset,
		dynamicClient: nil, // We can add a fake dynamic client if needed
		config:        &rest.Config{Host: "http://localhost:8080"},
		namespaces:    make(map[string]*corev1.Namespace),
	}
}

// GetClientset returns the fake clientset
func (m *MockK8sClient) GetClientset() kubernetes.Interface {
	return m.clientset
}

// GetDynamicClient returns the dynamic client
func (m *MockK8sClient) GetDynamicClient() dynamic.Interface {
	return m.dynamicClient
}

// GetConfig returns the REST config
func (m *MockK8sClient) GetConfig() *rest.Config {
	return m.config
}

// NamespaceExists checks if a namespace exists
func (m *MockK8sClient) NamespaceExists(ctx context.Context, namespaceName string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if namespaceName == "" {
		return false, fmt.Errorf("namespace name cannot be empty")
	}

	_, exists := m.namespaces[namespaceName]
	return exists, nil
}

// CreateNamespace creates a namespace
func (m *MockK8sClient) CreateNamespace(ctx context.Context, namespaceName string, labels map[string]string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if namespaceName == "" {
		return fmt.Errorf("namespace name cannot be empty")
	}

	// Check if namespace already exists
	if _, exists := m.namespaces[namespaceName]; exists {
		return nil // Already exists, no error
	}

	// Create namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespaceName,
			Labels: labels,
		},
	}

	m.namespaces[namespaceName] = namespace
	return nil
}

// GetNamespace gets a namespace by name
func (m *MockK8sClient) GetNamespace(ctx context.Context, namespaceName string) (*corev1.Namespace, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if namespaceName == "" {
		return nil, fmt.Errorf("namespace name cannot be empty")
	}

	namespace, exists := m.namespaces[namespaceName]
	if !exists {
		return nil, apierrors.NewNotFound(corev1.Resource("namespaces"), namespaceName)
	}

	// Return a copy to avoid modification
	namespaceCopy := namespace.DeepCopy()
	return namespaceCopy, nil
}

// AddTestNamespace adds a namespace to the mock for testing
func (m *MockK8sClient) AddTestNamespace(namespace *corev1.Namespace) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.namespaces[namespace.Name] = namespace.DeepCopy()
}

// RemoveTestNamespace removes a namespace from the mock
func (m *MockK8sClient) RemoveTestNamespace(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.namespaces, name)
}

// ListNamespaces returns all namespaces
func (m *MockK8sClient) ListNamespaces(ctx context.Context) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var names []string
	for name := range m.namespaces {
		names = append(names, name)
	}
	return names, nil
}

// EnsureNamespace implements the k8s.Client interface
func (m *MockK8sClient) EnsureNamespace(ctx context.Context, namespaceName string, labels map[string]string) error {
	exists, err := m.NamespaceExists(ctx, namespaceName)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return m.CreateNamespace(ctx, namespaceName, labels)
}

// AddLabelsToNamespace adds labels to an existing namespace
func (m *MockK8sClient) AddLabelsToNamespace(ctx context.Context, namespaceName string, labels map[string]string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	namespace, exists := m.namespaces[namespaceName]
	if !exists {
		return fmt.Errorf("namespace %s not found", namespaceName)
	}

	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}

	for k, v := range labels {
		namespace.Labels[k] = v
	}

	return nil
}

// AddAnnotationsToNamespace adds annotations to an existing namespace
func (m *MockK8sClient) AddAnnotationsToNamespace(ctx context.Context, namespaceName string, annotations map[string]string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	namespace, exists := m.namespaces[namespaceName]
	if !exists {
		return fmt.Errorf("namespace %s not found", namespaceName)
	}

	if namespace.Annotations == nil {
		namespace.Annotations = make(map[string]string)
	}

	for k, v := range annotations {
		namespace.Annotations[k] = v
	}

	return nil
}

// GetNamespaceStatus returns the status of a namespace
func (m *MockK8sClient) GetNamespaceStatus(ctx context.Context, namespaceName string) (corev1.NamespacePhase, error) {
	namespace, err := m.GetNamespace(ctx, namespaceName)
	if err != nil {
		return "", err
	}
	return namespace.Status.Phase, nil
}

// WaitForNamespaceReady waits for a namespace to be ready
func (m *MockK8sClient) WaitForNamespaceReady(ctx context.Context, namespaceName string, timeout time.Duration) error {
	// For testing, we'll just check if the namespace exists
	_, err := m.GetNamespace(ctx, namespaceName)
	return err
}

// TestConnection tests the connection to the mock cluster
func (m *MockK8sClient) TestConnection() error {
	// Always succeed for mock
	return nil
}

// CheckNamespaceExists is an enhanced version with better error handling
func (m *MockK8sClient) CheckNamespaceExists(ctx context.Context, namespaceName string) (bool, error) {
	return m.NamespaceExists(ctx, namespaceName)
}

// CreateNamespaceWithLabels creates a namespace with specific labels and annotations
func (m *MockK8sClient) CreateNamespaceWithLabels(ctx context.Context, namespaceName string, labels, annotations map[string]string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if namespaceName == "" {
		return fmt.Errorf("namespace name cannot be empty")
	}

	// Check if namespace already exists
	if _, exists := m.namespaces[namespaceName]; exists {
		return nil // Already exists, no error
	}

	// Create namespace with labels and annotations
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        namespaceName,
			Labels:      labels,
			Annotations: annotations,
		},
	}

	m.namespaces[namespaceName] = namespace
	return nil
}

// GenerateNamespaceName generates namespace names based on app and environment
func (m *MockK8sClient) GenerateNamespaceName(appName, environment string) (string, error) {
	return k8s.GenerateNamespaceName(appName, environment)
}

// FormatNamespaceForDisplay formats namespace information for display
func (m *MockK8sClient) FormatNamespaceForDisplay(namespace *corev1.Namespace) k8s.NamespaceInfo {
	return k8s.FormatNamespaceForDisplay(namespace)
}

// ValidateNamespaceName validates namespace names against Kubernetes requirements
func (m *MockK8sClient) ValidateNamespaceName(name string) error {
	return k8s.ValidateNamespaceName(name)
}
