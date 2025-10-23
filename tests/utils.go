package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/your-org/kargo-bootstrap/mocks"
	"github.com/your-org/kargo-bootstrap/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// TestConfig holds configuration for tests
type TestConfig struct {
	K8sClient     K8sClientInterface
	TestNamespace string
	CleanupFuncs  []func() error
}

// K8sClientInterface defines the interface for Kubernetes clients in tests
type K8sClientInterface interface {
	NamespaceExists(ctx context.Context, namespaceName string) (bool, error)
	CreateNamespace(ctx context.Context, namespaceName string, labels map[string]string) error
	GetNamespace(ctx context.Context, namespaceName string) (*corev1.Namespace, error)
	AddTestNamespace(namespace *corev1.Namespace)
	RemoveTestNamespace(name string)
	ListNamespaces(ctx context.Context) ([]string, error)
	EnsureNamespace(ctx context.Context, namespaceName string, labels map[string]string) error
	AddLabelsToNamespace(ctx context.Context, namespaceName string, labels map[string]string) error
	AddAnnotationsToNamespace(ctx context.Context, namespaceName string, annotations map[string]string) error
	GetNamespaceStatus(ctx context.Context, namespaceName string) (corev1.NamespacePhase, error)
	WaitForNamespaceReady(ctx context.Context, namespaceName string, timeout time.Duration) error
	TestConnection() error
	CheckNamespaceExists(ctx context.Context, namespaceName string) (bool, error)
	CreateNamespaceWithLabels(ctx context.Context, namespaceName string, labels, annotations map[string]string) error
	GenerateNamespaceName(appName, environment string) (string, error)
	FormatNamespaceForDisplay(namespace *corev1.Namespace) k8s.NamespaceInfo
	ValidateNamespaceName(name string) error
	GetClientset() kubernetes.Interface
	GetDynamicClient() dynamic.Interface
	GetConfig() *rest.Config
}

// SetupTest creates a test environment with mock clients
func SetupTest(t *testing.T) *TestConfig {
	t.Helper()

	// Create mock Kubernetes client
	mockK8sClient := mocks.NewMockK8sClient()

	testNamespace := fmt.Sprintf("test-%d", time.Now().Unix())

	config := &TestConfig{
		K8sClient:     mockK8sClient,
		TestNamespace: testNamespace,
		CleanupFuncs:  []func() error{},
	}

	// Register cleanup function
	t.Cleanup(func() {
		if err := config.Cleanup(); err != nil {
			t.Errorf("Cleanup failed: %v", err)
		}
	})

	return config
}

// Cleanup runs all registered cleanup functions
func (tc *TestConfig) Cleanup() error {
	for _, cleanup := range tc.CleanupFuncs {
		if err := cleanup(); err != nil {
			return fmt.Errorf("cleanup function failed: %w", err)
		}
	}
	return nil
}

// CreateTestNamespace creates a test namespace
func (tc *TestConfig) CreateTestNamespace(t *testing.T, labels map[string]string) {
	t.Helper()

	ctx := context.Background()

	err := tc.K8sClient.CreateNamespace(ctx, tc.TestNamespace, labels)
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	// Register cleanup function
	tc.CleanupFuncs = append(tc.CleanupFuncs, func() error {
		tc.K8sClient.RemoveTestNamespace(tc.TestNamespace)
		return nil
	})
}

// CreateTestArgoCDProject creates a test ArgoCD project
func (tc *TestConfig) CreateTestArgoCDProject(t *testing.T, projectName string) {
	t.Helper()

	// This would create an ArgoCD project using the dynamic client
	// For now, we'll just register a cleanup function
	tc.CleanupFuncs = append(tc.CleanupFuncs, func() error {
		// Cleanup logic for ArgoCD project would go here
		return nil
	})
}

// GetTestConfig returns a test configuration
func GetTestConfig() *rest.Config {
	return &rest.Config{
		Host: "http://localhost:8080",
	}
}

// CreateTempDir creates a temporary directory for tests
func CreateTempDir(t *testing.T) string {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "kargo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Register cleanup function
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

// CreateTestFile creates a test file with the given content
func CreateTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()

	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}

	return filePath
}

// AssertNoError asserts that an error is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// AssertError asserts that an error is not nil
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
}

// AssertEquals asserts that two values are equal
func AssertEquals[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEquals asserts that two values are not equal
func AssertNotEquals[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected == actual {
		t.Fatalf("Expected values to be different, but both are %v", expected)
	}
}

// AssertContains asserts that a slice contains a value
func AssertContains[T comparable](t *testing.T, slice []T, value T) {
	t.Helper()
	for _, item := range slice {
		if item == value {
			return
		}
	}
	t.Fatalf("Expected slice %v to contain %v", slice, value)
}

// AssertNotContains asserts that a slice does not contain a value
func AssertNotContains[T comparable](t *testing.T, slice []T, value T) {
	t.Helper()
	for _, item := range slice {
		if item == value {
			t.Fatalf("Expected slice %v to not contain %v", slice, value)
		}
	}
}

// WaitForCondition waits for a condition to be true
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for condition: %s", message)
}

// SkipIfShort skips the test if testing.Short() is true
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// GetTestDataPath returns the path to a test data file
func GetTestDataPath(filename string) string {
	return filepath.Join("testdata", filename)
}

// ReadTestData reads a test data file
func ReadTestData(t *testing.T, filename string) []byte {
	t.Helper()

	path := GetTestDataPath(filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read test data file %s: %v", path, err)
	}

	return data
}
