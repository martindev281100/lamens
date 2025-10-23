package argocd

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestParseApplicationYAML(t *testing.T) {
	yamlContent := `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: test-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/example/example.git
    targetRevision: main
    path: .
  destination:
    server: https://kubernetes.default.svc
    namespace: default
`

	app, err := ParseApplicationYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse Application YAML: %v", err)
	}

	if app.GetKind() != "Application" {
		t.Errorf("Expected kind 'Application', got '%s'", app.GetKind())
	}

	if app.GetAPIVersion() != "argoproj.io/v1alpha1" {
		t.Errorf("Expected apiVersion 'argoproj.io/v1alpha1', got '%s'", app.GetAPIVersion())
	}

	if app.GetName() != "test-app" {
		t.Errorf("Expected name 'test-app', got '%s'", app.GetName())
	}

	if app.GetNamespace() != "argocd" {
		t.Errorf("Expected namespace 'argocd', got '%s'", app.GetNamespace())
	}
}

func TestValidateApplication(t *testing.T) {
	tests := []struct {
		name        string
		app         *unstructured.Unstructured
		expectError bool
	}{
		{
			name: "valid application",
			app: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "argoproj.io/v1alpha1",
					"kind":       "Application",
					"metadata": map[string]interface{}{
						"name": "test-app",
					},
					"spec": map[string]interface{}{
						"project": "default",
						"source": map[string]interface{}{
							"repoURL": "https://github.com/example/example.git",
							"path":    ".",
						},
						"destination": map[string]interface{}{
							"server":    "https://kubernetes.default.svc",
							"namespace": "default",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing name",
			app: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "argoproj.io/v1alpha1",
					"kind":       "Application",
					"metadata":   map[string]interface{}{},
					"spec": map[string]interface{}{
						"project": "default",
						"source": map[string]interface{}{
							"repoURL": "https://github.com/example/example.git",
							"path":    ".",
						},
						"destination": map[string]interface{}{
							"server":    "https://kubernetes.default.svc",
							"namespace": "default",
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "missing spec",
			app: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "argoproj.io/v1alpha1",
					"kind":       "Application",
					"metadata": map[string]interface{}{
						"name": "test-app",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateApplication(tt.app)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateApplication() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateApplicationName(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		expectError bool
	}{
		{
			name:        "valid name",
			appName:     "test-app",
			expectError: false,
		},
		{
			name:        "empty name",
			appName:     "",
			expectError: true,
		},
		{
			name:        "too long name",
			appName:     "this-is-a-very-long-application-name-that-exceeds-the-maximum-allowed-length-of-sixty-three-characters",
			expectError: true,
		},
		{
			name:        "invalid characters",
			appName:     "test_app",
			expectError: true,
		},
		{
			name:        "starts with hyphen",
			appName:     "-test-app",
			expectError: true,
		},
		{
			name:        "ends with hyphen",
			appName:     "test-app-",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateApplicationName(tt.appName)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateApplicationName() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestGetApplicationStatusFromUnstructured(t *testing.T) {
	app := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Application",
			"metadata": map[string]interface{}{
				"name":      "test-app",
				"namespace": "argocd",
			},
			"status": map[string]interface{}{
				"health": map[string]interface{}{
					"status":  "Healthy",
					"message": "Application is healthy",
				},
				"sync": map[string]interface{}{
					"status":     "Synced",
					"revision":   "main-123456",
					"comparedAt": "2023-01-01T00:00:00Z",
				},
				"operation": map[string]interface{}{
					"phase":      "Succeeded",
					"message":    "Operation completed successfully",
					"startedAt":  "2023-01-01T00:00:00Z",
					"finishedAt": "2023-01-01T00:01:00Z",
				},
			},
		},
	}

	status, err := GetApplicationStatusFromUnstructured(*app)
	if err != nil {
		t.Fatalf("Failed to get Application status: %v", err)
	}

	if status.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got '%s'", status.Name)
	}

	if status.Namespace != "argocd" {
		t.Errorf("Expected namespace 'argocd', got '%s'", status.Namespace)
	}

	if status.Health.Status != "Healthy" {
		t.Errorf("Expected health status 'Healthy', got '%s'", status.Health.Status)
	}

	if status.Health.Message != "Application is healthy" {
		t.Errorf("Expected health message 'Application is healthy', got '%s'", status.Health.Message)
	}

	if status.Sync.Status != "Synced" {
		t.Errorf("Expected sync status 'Synced', got '%s'", status.Sync.Status)
	}

	if status.Sync.Revision != "main-123456" {
		t.Errorf("Expected sync revision 'main-123456', got '%s'", status.Sync.Revision)
	}

	if status.Operation.Phase != "Succeeded" {
		t.Errorf("Expected operation phase 'Succeeded', got '%s'", status.Operation.Phase)
	}
}

func TestIsApplicationHealthy(t *testing.T) {
	tests := []struct {
		name     string
		status   *ApplicationStatus
		expected bool
	}{
		{
			name:     "healthy status",
			status:   &ApplicationStatus{Health: ApplicationHealthStatus{Status: "Healthy"}},
			expected: true,
		},
		{
			name:     "progressing status",
			status:   &ApplicationStatus{Health: ApplicationHealthStatus{Status: "Progressing"}},
			expected: true,
		},
		{
			name:     "degraded status",
			status:   &ApplicationStatus{Health: ApplicationHealthStatus{Status: "Degraded"}},
			expected: true,
		},
		{
			name:     "unknown status",
			status:   &ApplicationStatus{Health: ApplicationHealthStatus{Status: "Unknown"}},
			expected: false,
		},
		{
			name:     "nil status",
			status:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsApplicationHealthy(tt.status)
			if result != tt.expected {
				t.Errorf("IsApplicationHealthy() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCreateApplication(t *testing.T) {
	// Create a fake dynamic client
	fakeClient := fake.NewSimpleDynamicClient(scheme.Scheme)
	client := &Client{
		dynamicClient: fakeClient,
		namespace:     "argocd",
	}

	yamlContent := `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: test-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/example/example.git
    targetRevision: main
    path: .
  destination:
    server: https://kubernetes.default.svc
    namespace: default
`

	// Test creating the Application
	status, err := client.CreateApplication(context.Background(), []byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to create Application: %v", err)
	}

	if status.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got '%s'", status.Name)
	}

	if status.Namespace != "argocd" {
		t.Errorf("Expected namespace 'argocd', got '%s'", status.Namespace)
	}
}

func TestSyncApplication(t *testing.T) {
	// Create a fake dynamic client
	// Create a fake Application
	app := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Application",
			"metadata": map[string]interface{}{
				"name":      "test-app",
				"namespace": "argocd",
			},
			"spec": map[string]interface{}{
				"project": "default",
				"source": map[string]interface{}{
					"repoURL": "https://github.com/example/example.git",
					"path":    ".",
				},
				"destination": map[string]interface{}{
					"server":    "https://kubernetes.default.svc",
					"namespace": "default",
				},
			},
		},
	}

	fakeClient := fake.NewSimpleDynamicClient(scheme.Scheme, app)
	client := &Client{
		dynamicClient: fakeClient,
		namespace:     "argocd",
	}

	options := &ApplicationSyncOptions{
		Revision: "main",
		DryRun:   false,
		Prune:    false,
		Force:    false,
	}

	// Test syncing the Application
	err := client.SyncApplication(context.Background(), "test-app", options)
	if err != nil {
		t.Fatalf("Failed to sync Application: %v", err)
	}
}

func TestGetApplicationGVR(t *testing.T) {
	gvr := getApplicationGVR()

	if gvr.Group != "argoproj.io" {
		t.Errorf("Expected group 'argoproj.io', got '%s'", gvr.Group)
	}

	if gvr.Version != "v1alpha1" {
		t.Errorf("Expected version 'v1alpha1', got '%s'", gvr.Version)
	}

	if gvr.Resource != "applications" {
		t.Errorf("Expected resource 'applications', got '%s'", gvr.Resource)
	}
}
