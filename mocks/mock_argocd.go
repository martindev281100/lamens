package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/your-org/kargo-bootstrap/pkg/argocd"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
)

// MockArgoCDClient implements a mock ArgoCD client for testing
type MockArgoCDClient struct {
	dynamicClient dynamic.Interface
	namespace     string
	projects      map[string]*argocd.Project
	mutex         sync.RWMutex
}

// NewMockArgoCDClient creates a new mock ArgoCD client
func NewMockArgoCDClient(namespace string) *MockArgoCDClient {
	scheme := runtime.NewScheme()
	// We would add ArgoCD types to the scheme here if needed
	fakeDynamicClient := fake.NewSimpleDynamicClient(scheme)

	return &MockArgoCDClient{
		dynamicClient: fakeDynamicClient,
		namespace:     namespace,
		projects:      make(map[string]*argocd.Project),
	}
}

// GetDynamicClient returns the dynamic client
func (m *MockArgoCDClient) GetDynamicClient() dynamic.Interface {
	return m.dynamicClient
}

// ListProjects returns a list of available ArgoCD projects
func (m *MockArgoCDClient) ListProjects(ctx context.Context) ([]argocd.Project, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var projects []argocd.Project
	for _, project := range m.projects {
		projects = append(projects, *project)
	}

	return projects, nil
}

// GetProject gets details of a specific project
func (m *MockArgoCDClient) GetProject(ctx context.Context, name string) (*argocd.Project, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	project, exists := m.projects[name]
	if !exists {
		return nil, fmt.Errorf("project %s not found", name)
	}

	// Return a copy to avoid modification
	projectCopy := *project
	return &projectCopy, nil
}

// ValidateProject checks if a project exists and is accessible
func (m *MockArgoCDClient) ValidateProject(ctx context.Context, name string) error {
	_, err := m.GetProject(ctx, name)
	return err
}

// AddTestProject adds a project to the mock for testing
func (m *MockArgoCDClient) AddTestProject(project *argocd.Project) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.projects[project.Name] = project
}

// RemoveTestProject removes a project from the mock
func (m *MockArgoCDClient) RemoveTestProject(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.projects, name)
}

// IsArgoCDInstalled checks if ArgoCD is installed and accessible
func (m *MockArgoCDClient) IsArgoCDInstalled(ctx context.Context) error {
	// Always succeed for mock
	return nil
}

// GetDefaultProject returns the default project if it exists
func (m *MockArgoCDClient) GetDefaultProject(ctx context.Context) (*argocd.Project, error) {
	// Try to get the "default" project first
	project, err := m.GetProject(ctx, "default")
	if err == nil {
		return project, nil
	}

	// If default doesn't exist, get the first available project
	projects, err := m.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no ArgoCD projects found")
	}

	// Return the first project in the list
	return &projects[0], nil
}

// CreateTestApplication creates a test application in the mock
func (m *MockArgoCDClient) CreateTestApplication(ctx context.Context, name, project, repoURL, path, namespace string) error {
	// This would create an ArgoCD Application using the dynamic client
	// For now, we'll just return success
	return nil
}

// DeleteTestApplication deletes a test application from the mock
func (m *MockArgoCDClient) DeleteTestApplication(ctx context.Context, name string) error {
	// This would delete an ArgoCD Application using the dynamic client
	// For now, we'll just return success
	return nil
}

// GetTestApplication gets a test application from the mock
func (m *MockArgoCDClient) GetTestApplication(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	// This would get an ArgoCD Application using the dynamic client
	// For now, we'll return a simple mock application
	app := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Application",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": m.namespace,
			},
			"spec": map[string]interface{}{
				"project": "default",
				"source": map[string]interface{}{
					"repoURL":        "https://github.com/example/repo.git",
					"path":           "path/to/chart",
					"targetRevision": "main",
				},
				"destination": map[string]interface{}{
					"server":    "https://kubernetes.default.svc",
					"namespace": "default",
				},
			},
		},
	}
	return app, nil
}

// SyncApplication syncs an application
func (m *MockArgoCDClient) SyncApplication(ctx context.Context, name string) error {
	// Mock sync operation
	return nil
}

// GetApplicationStatus gets the status of an application
func (m *MockArgoCDClient) GetApplicationStatus(ctx context.Context, name string) (string, error) {
	// Return a mock status
	return "Healthy", nil
}

// ListApplications lists all applications
func (m *MockArgoCDClient) ListApplications(ctx context.Context) ([]*unstructured.Unstructured, error) {
	// Return a mock list of applications
	var apps []*unstructured.Unstructured

	// Add a few mock applications
	for i := 1; i <= 3; i++ {
		app := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "argoproj.io/v1alpha1",
				"kind":       "Application",
				"metadata": map[string]interface{}{
					"name":      fmt.Sprintf("app-%d", i),
					"namespace": m.namespace,
				},
				"spec": map[string]interface{}{
					"project": "default",
					"source": map[string]interface{}{
						"repoURL":        "https://github.com/example/repo.git",
						"path":           "path/to/chart",
						"targetRevision": "main",
					},
					"destination": map[string]interface{}{
						"server":    "https://kubernetes.default.svc",
						"namespace": "default",
					},
				},
			},
		}
		apps = append(apps, app)
	}

	return apps, nil
}

// CreateProject creates a new project
func (m *MockArgoCDClient) CreateProject(ctx context.Context, project *argocd.Project) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.projects[project.Name]; exists {
		return fmt.Errorf("project %s already exists", project.Name)
	}

	// Create a copy of the project
	projectCopy := *project
	m.projects[project.Name] = &projectCopy
	return nil
}

// UpdateProject updates an existing project
func (m *MockArgoCDClient) UpdateProject(ctx context.Context, project *argocd.Project) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.projects[project.Name]; !exists {
		return fmt.Errorf("project %s not found", project.Name)
	}

	// Update the project
	projectCopy := *project
	m.projects[project.Name] = &projectCopy
	return nil
}

// DeleteProject deletes a project
func (m *MockArgoCDClient) DeleteProject(ctx context.Context, name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.projects[name]; !exists {
		return fmt.Errorf("project %s not found", name)
	}

	delete(m.projects, name)
	return nil
}
