package mocks

import (
	"fmt"

	"github.com/your-org/kargo-bootstrap/pkg/git"
	"github.com/your-org/kargo-bootstrap/pkg/prompt"
)

// MockPromptClient implements a mock prompt client for testing
type MockPromptClient struct {
	responses map[string]interface{}
	questions []string
	index     int
}

// NewMockPromptClient creates a new mock prompt client
func NewMockPromptClient() *MockPromptClient {
	return &MockPromptClient{
		responses: make(map[string]interface{}),
		questions: []string{},
		index:     0,
	}
}

// SetResponse sets a predefined response for a question
func (m *MockPromptClient) SetResponse(question string, response interface{}) {
	m.responses[question] = response
}

// SetResponses sets multiple predefined responses
func (m *MockPromptClient) SetResponses(responses map[string]interface{}) {
	for k, v := range responses {
		m.responses[k] = v
	}
}

// GetQuestions returns all questions that were asked
func (m *MockPromptClient) GetQuestions() []string {
	return m.questions
}

// Reset resets the mock client
func (m *MockPromptClient) Reset() {
	m.responses = make(map[string]interface{})
	m.questions = []string{}
	m.index = 0
}

// Confirm prompts for confirmation
func (m *MockPromptClient) Confirm(message string, defaultValue bool) (bool, error) {
	m.questions = append(m.questions, fmt.Sprintf("CONFIRM: %s (default: %t)", message, defaultValue))

	if response, exists := m.responses[message]; exists {
		if result, ok := response.(bool); ok {
			return result, nil
		}
		return defaultValue, nil
	}

	return defaultValue, nil
}

// Input prompts for text input
func (m *MockPromptClient) Input(message, defaultValue string) (string, error) {
	m.questions = append(m.questions, fmt.Sprintf("INPUT: %s (default: %s)", message, defaultValue))

	if response, exists := m.responses[message]; exists {
		if result, ok := response.(string); ok {
			return result, nil
		}
		return defaultValue, nil
	}

	return defaultValue, nil
}

// Select prompts for selection from a list
func (m *MockPromptClient) Select(message string, options []string, defaultIndex int) (string, error) {
	m.questions = append(m.questions, fmt.Sprintf("SELECT: %s (options: %v, default: %d)", message, options, defaultIndex))

	if response, exists := m.responses[message]; exists {
		if result, ok := response.(string); ok {
			return result, nil
		}
		if idx, ok := response.(int); ok && idx >= 0 && idx < len(options) {
			return options[idx], nil
		}
		return options[defaultIndex], nil
	}

	if len(options) > 0 {
		return options[defaultIndex], nil
	}
	return "", fmt.Errorf("no options provided")
}

// MultiSelect prompts for multiple selections from a list
func (m *MockPromptClient) MultiSelect(message string, options []string, defaultIndices []int) ([]string, error) {
	m.questions = append(m.questions, fmt.Sprintf("MULTISELECT: %s (options: %v, default: %v)", message, options, defaultIndices))

	if response, exists := m.responses[message]; exists {
		if result, ok := response.([]string); ok {
			return result, nil
		}
		if indices, ok := response.([]int); ok {
			var selected []string
			for _, idx := range indices {
				if idx >= 0 && idx < len(options) {
					selected = append(selected, options[idx])
				}
			}
			return selected, nil
		}
		return getSelectedOptions(options, defaultIndices), nil
	}

	return getSelectedOptions(options, defaultIndices), nil
}

// Password prompts for password input
func (m *MockPromptClient) Password(message string) (string, error) {
	m.questions = append(m.questions, fmt.Sprintf("PASSWORD: %s", message))

	if response, exists := m.responses[message]; exists {
		if result, ok := response.(string); ok {
			return result, nil
		}
	}

	return "mock-password", nil
}

// PromptForAppName prompts for application name
func (m *MockPromptClient) PromptForAppName() (*prompt.AppNameInput, error) {
	m.questions = append(m.questions, "PROMPT_FOR_APP_NAME")

	name, _ := m.Input("Application name", "test-app")

	return &prompt.AppNameInput{
		Name:      name,
		Validated: true,
		Cancelled: false,
	}, nil
}

// PromptForRepository prompts for Git repository information
func (m *MockPromptClient) PromptForRepository() (*prompt.RepositoryInput, error) {
	m.questions = append(m.questions, "PROMPT_FOR_REPOSITORY")

	url, _ := m.Input("Git repository URL", "https://github.com/example/repo.git")

	return &prompt.RepositoryInput{
		URL:       url,
		Validated: true,
		Cancelled: false,
	}, nil
}

// PromptForRevision prompts for Git revision selection
func (m *MockPromptClient) PromptForRevision() (*prompt.RevisionSelection, error) {
	m.questions = append(m.questions, "PROMPT_FOR_REVISION")

	options := []string{"main", "develop", "v1.0.0"}
	selection, _ := m.Select("Select revision", options, 0)

	// Determine the type based on the selection
	revisionType := "branch"
	if selection == "v1.0.0" {
		revisionType = "tag"
	}

	return &prompt.RevisionSelection{
		Type:      revisionType,
		Value:     selection,
		Confirmed: true,
		Cancelled: false,
	}, nil
}

// PromptForChart prompts for chart selection
func (m *MockPromptClient) PromptForChart() (*prompt.ChartSelection, error) {
	m.questions = append(m.questions, "PROMPT_FOR_CHART")

	options := []string{"chart1", "chart2", "chart3"}
	selection, _ := m.Select("Select chart", options, 0)

	return &prompt.ChartSelection{
		Chart: &git.ChartInfo{
			Name:    selection,
			Path:    selection,
			Version: "1.0.0",
		},
		Confirmed: true,
		Cancelled: false,
	}, nil
}

// PromptForEnvironment prompts for environment selection
func (m *MockPromptClient) PromptForEnvironment() (*prompt.EnvironmentSelection, error) {
	m.questions = append(m.questions, "PROMPT_FOR_ENVIRONMENT")

	options := []string{"development", "staging", "production"}
	selected, _ := m.MultiSelect("Select environments", options, []int{0})

	return &prompt.EnvironmentSelection{
		Environments: selected,
		Confirmed:    true,
		Cancelled:    false,
	}, nil
}

// PromptForHelmValues prompts for Helm values
func (m *MockPromptClient) PromptForHelmValues() (map[string]interface{}, error) {
	m.questions = append(m.questions, "PROMPT_FOR_HELM_VALUES")

	// Return mock Helm values
	values := map[string]interface{}{
		"replicaCount": 1,
		"image": map[string]interface{}{
			"repository": "nginx",
			"tag":        "latest",
		},
		"service": map[string]interface{}{
			"type": "ClusterIP",
			"port": 80,
		},
	}

	return values, nil
}

// ConfirmDeployment prompts for deployment confirmation
func (m *MockPromptClient) ConfirmDeployment(appName, namespace, project string) (bool, error) {
	message := fmt.Sprintf("Deploy %s to namespace %s using project %s?", appName, namespace, project)
	return m.Confirm(message, true)
}

// PromptForCluster prompts for cluster selection
func (m *MockPromptClient) PromptForCluster() (string, error) {
	m.questions = append(m.questions, "PROMPT_FOR_CLUSTER")

	options := []string{"cluster-1", "cluster-2", "cluster-3"}
	selection, _ := m.Select("Select cluster", options, 0)

	return selection, nil
}

// getSelectedOptions returns the selected options based on indices
func getSelectedOptions(options []string, indices []int) []string {
	var selected []string
	for _, idx := range indices {
		if idx >= 0 && idx < len(options) {
			selected = append(selected, options[idx])
		}
	}
	return selected
}

// MockPromptHelper provides helper functions for testing prompts
type MockPromptHelper struct {
	client *MockPromptClient
}

// NewMockPromptHelper creates a new mock prompt helper
func NewMockPromptHelper() *MockPromptHelper {
	return &MockPromptHelper{
		client: NewMockPromptClient(),
	}
}

// GetClient returns the mock prompt client
func (h *MockPromptHelper) GetClient() *MockPromptClient {
	return h.client
}

// SetupAppInfoMock sets up mock responses for app info prompts
func (h *MockPromptHelper) SetupAppInfoMock(name, description string) {
	h.client.SetResponse("Application name", name)
	h.client.SetResponse("Application description", description)
}

// SetupProjectInfoMock sets up mock responses for project info prompts
func (h *MockPromptHelper) SetupProjectInfoMock(name, description string) {
	h.client.SetResponse("Project name", name)
	h.client.SetResponse("Project description", description)
}

// SetupGitInfoMock sets up mock responses for Git info prompts
func (h *MockPromptHelper) SetupGitInfoMock(url, path, branch string) {
	h.client.SetResponse("Git repository URL", url)
	h.client.SetResponse("Path to Helm chart", path)
	h.client.SetResponse("Branch", branch)
}

// SetupDeploymentInfoMock sets up mock responses for deployment info prompts
func (h *MockPromptHelper) SetupDeploymentInfoMock(namespace, server string) {
	h.client.SetResponse("Target namespace", namespace)
	h.client.SetResponse("Target server", server)
}

// SetupConfirmationMock sets up mock responses for confirmation prompts
func (h *MockPromptHelper) SetupConfirmationMock(confirmation bool) {
	h.client.SetResponse("Deploy test-app to namespace default using project test-project?", confirmation)
}
