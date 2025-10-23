package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

// EnvironmentType represents the type of environment.
type EnvironmentType string

const (
	// EnvironmentTypeDevelopment represents a development environment.
	EnvironmentTypeDevelopment EnvironmentType = "development"
	// EnvironmentTypeStaging represents a staging environment.
	EnvironmentTypeStaging EnvironmentType = "staging"
	// EnvironmentTypeProduction represents a production environment.
	EnvironmentTypeProduction EnvironmentType = "production"
)

// Environment represents a deployment environment configuration.
type Environment struct {
	// Name is the name of the environment.
	Name string `yaml:"name"`
	// Type is the type of environment.
	Type EnvironmentType `yaml:"type"`
	// Description is the description of the environment.
	Description string `yaml:"description"`
	// Namespace is the namespace for the environment.
	Namespace string `yaml:"namespace"`
	// Default indicates if this is the default environment.
	Default bool `yaml:"default"`
	// AutoSync indicates if ArgoCD should automatically sync changes.
	AutoSync bool `yaml:"autoSync"`
	// Prune indicates if ArgoCD should prune resources.
	Prune bool `yaml:"prune"`
	// SelfHeal indicates if ArgoCD should self-heal.
	SelfHeal bool `yaml:"selfHeal"`
	// ReplicaCount is the number of replicas for the environment.
	ReplicaCount int `yaml:"replicaCount"`
	// Values contains additional values for the environment.
	Values map[string]interface{} `yaml:"values"`
	// Labels contains additional labels for the environment.
	Labels map[string]string `yaml:"labels"`
	// Annotations contains additional annotations for the environment.
	Annotations map[string]string `yaml:"annotations"`
}

// EnvironmentConfig represents the configuration for multiple environments.
type EnvironmentConfig struct {
	// Environments is the list of environments.
	Environments []Environment `yaml:"environments"`
}

// EnvironmentManager manages environment configurations.
type EnvironmentManager struct {
	// environments is the list of environments.
	environments []Environment
	// configPath is the path to the configuration file.
	configPath string
}

// NewEnvironmentManager creates a new environment manager.
//
// Returns:
//   - *EnvironmentManager: A new environment manager.
func NewEnvironmentManager() *EnvironmentManager {
	return &EnvironmentManager{
		environments: make([]Environment, 0),
	}
}

// LoadDefaultEnvironments loads the default environment configurations.
func (m *EnvironmentManager) LoadDefaultEnvironments() {
	// Add default environments
	m.environments = []Environment{
		{
			Name:         "development",
			Type:         EnvironmentTypeDevelopment,
			Description:  "Development environment for local testing",
			Namespace:    "dev",
			Default:      true,
			AutoSync:     true,
			Prune:        true,
			SelfHeal:     true,
			ReplicaCount: 1,
			Values: map[string]interface{}{
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    "500m",
						"memory": "512Mi",
					},
					"requests": map[string]interface{}{
						"cpu":    "100m",
						"memory": "128Mi",
					},
				},
				"env": map[string]interface{}{
					"LOG_LEVEL": "debug",
				},
			},
		},
		{
			Name:         "staging",
			Type:         EnvironmentTypeStaging,
			Description:  "Staging environment for integration testing",
			Namespace:    "staging",
			Default:      false,
			AutoSync:     true,
			Prune:        true,
			SelfHeal:     true,
			ReplicaCount: 2,
			Values: map[string]interface{}{
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    "1000m",
						"memory": "1Gi",
					},
					"requests": map[string]interface{}{
						"cpu":    "500m",
						"memory": "512Mi",
					},
				},
				"env": map[string]interface{}{
					"LOG_LEVEL": "info",
				},
			},
		},
		{
			Name:         "production",
			Type:         EnvironmentTypeProduction,
			Description:  "Production environment with high availability",
			Namespace:    "prod",
			Default:      false,
			AutoSync:     false,
			Prune:        false,
			SelfHeal:     true,
			ReplicaCount: 3,
			Values: map[string]interface{}{
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    "2000m",
						"memory": "2Gi",
					},
					"requests": map[string]interface{}{
						"cpu":    "1000m",
						"memory": "1Gi",
					},
				},
				"env": map[string]interface{}{
					"LOG_LEVEL": "warn",
				},
			},
		},
	}
}

// LoadFromFile loads environment configurations from a file.
//
// Parameters:
//   - configPath: The path to the configuration file.
//
// Returns:
//   - error: An error if the configuration could not be loaded.
func (m *EnvironmentManager) LoadFromFile(configPath string) error {
	if configPath == "" {
		return fmt.Errorf("config path cannot be empty")
	}

	// Check if the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", configPath)
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the YAML
	var config EnvironmentConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Update the environments
	m.environments = config.Environments
	m.configPath = configPath

	klog.V(2).Infof("Loaded %d environments from %s", len(m.environments), configPath)
	return nil
}

// SaveToFile saves environment configurations to a file.
//
// Parameters:
//   - configPath: The path to the configuration file.
//
// Returns:
//   - error: An error if the configuration could not be saved.
func (m *EnvironmentManager) SaveToFile(configPath string) error {
	if configPath == "" {
		return fmt.Errorf("config path cannot be empty")
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create the config
	config := EnvironmentConfig{
		Environments: m.environments,
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	m.configPath = configPath
	klog.V(2).Infof("Saved %d environments to %s", len(m.environments), configPath)
	return nil
}

// GetEnvironment gets an environment by name.
//
// Parameters:
//   - name: The name of the environment.
//
// Returns:
//   - *Environment: The environment.
//   - error: An error if the environment could not be found.
func (m *EnvironmentManager) GetEnvironment(name string) (*Environment, error) {
	for _, env := range m.environments {
		if env.Name == name {
			return &env, nil
		}
	}

	return nil, fmt.Errorf("environment not found: %s", name)
}

// GetEnvironments gets all environments.
//
// Returns:
//   - []Environment: The list of environments.
func (m *EnvironmentManager) GetEnvironments() []Environment {
	// Return a copy to prevent modification
	environments := make([]Environment, len(m.environments))
	copy(environments, m.environments)
	return environments
}

// AddEnvironment adds an environment.
//
// Parameters:
//   - env: The environment to add.
//
// Returns:
//   - error: An error if the environment could not be added.
func (m *EnvironmentManager) AddEnvironment(env Environment) error {
	// Check if an environment with the same name already exists
	for _, existingEnv := range m.environments {
		if existingEnv.Name == env.Name {
			return fmt.Errorf("environment already exists: %s", env.Name)
		}
	}

	// Validate the environment
	if err := m.validateEnvironment(&env); err != nil {
		return fmt.Errorf("invalid environment: %w", err)
	}

	// Add the environment
	m.environments = append(m.environments, env)
	klog.V(2).Infof("Added environment: %s", env.Name)
	return nil
}

// UpdateEnvironment updates an environment.
//
// Parameters:
//   - name: The name of the environment to update.
//   - env: The updated environment.
//
// Returns:
//   - error: An error if the environment could not be updated.
func (m *EnvironmentManager) UpdateEnvironment(name string, env Environment) error {
	// Find the environment
	for i, existingEnv := range m.environments {
		if existingEnv.Name == name {
			// Validate the environment
			if err := m.validateEnvironment(&env); err != nil {
				return fmt.Errorf("invalid environment: %w", err)
			}

			// Update the environment
			m.environments[i] = env
			klog.V(2).Infof("Updated environment: %s", name)
			return nil
		}
	}

	return fmt.Errorf("environment not found: %s", name)
}

// RemoveEnvironment removes an environment.
//
// Parameters:
//   - name: The name of the environment to remove.
//
// Returns:
//   - error: An error if the environment could not be removed.
func (m *EnvironmentManager) RemoveEnvironment(name string) error {
	// Find the environment
	for i, env := range m.environments {
		if env.Name == name {
			// Remove the environment
			m.environments = append(m.environments[:i], m.environments[i+1:]...)
			klog.V(2).Infof("Removed environment: %s", name)
			return nil
		}
	}

	return fmt.Errorf("environment not found: %s", name)
}

// GetDefaultEnvironment gets the default environment.
//
// Returns:
//   - *Environment: The default environment.
//   - error: An error if no default environment is found.
func (m *EnvironmentManager) GetDefaultEnvironment() (*Environment, error) {
	for _, env := range m.environments {
		if env.Default {
			return &env, nil
		}
	}

	// If no default is set, return the first environment
	if len(m.environments) > 0 {
		return &m.environments[0], nil
	}

	return nil, fmt.Errorf("no environments found")
}

// SetDefaultEnvironment sets the default environment.
//
// Parameters:
//   - name: The name of the environment to set as default.
//
// Returns:
//   - error: An error if the environment could not be set as default.
func (m *EnvironmentManager) SetDefaultEnvironment(name string) error {
	// Unset all existing defaults
	for i := range m.environments {
		m.environments[i].Default = false
	}

	// Set the new default
	for i, env := range m.environments {
		if env.Name == name {
			m.environments[i].Default = true
			klog.V(2).Infof("Set default environment: %s", name)
			return nil
		}
	}

	return fmt.Errorf("environment not found: %s", name)
}

// ValidateEnvironmentSetup validates the setup of the specified environments.
//
// Parameters:
//   - environmentNames: The names of the environments to validate.
//
// Returns:
//   - *ValidationResult: The validation result.
func (m *EnvironmentManager) ValidateEnvironmentSetup(environmentNames []string) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// Check if any environments are specified
	if len(environmentNames) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "no environments specified")
		return result
	}

	// Validate each environment
	for _, name := range environmentNames {
		env, err := m.GetEnvironment(name)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("environment not found: %s", name))
			continue
		}

		// Validate the environment
		if err := m.validateEnvironment(env); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("invalid environment %s: %v", name, err))
		}
	}

	return result
}

// validateEnvironment validates an environment.
//
// Parameters:
//   - env: The environment to validate.
//
// Returns:
//   - error: An error if the environment is invalid.
func (m *EnvironmentManager) validateEnvironment(env *Environment) error {
	if env.Name == "" {
		return fmt.Errorf("environment name cannot be empty")
	}

	if env.Type == "" {
		return fmt.Errorf("environment type cannot be empty")
	}

	if env.Namespace == "" {
		return fmt.Errorf("environment namespace cannot be empty")
	}

	if env.ReplicaCount < 0 {
		return fmt.Errorf("environment replica count cannot be negative")
	}

	return nil
}

// ValidationResult represents the result of a validation.
type ValidationResult struct {
	// Valid indicates if the validation passed.
	Valid bool
	// Errors contains the validation errors.
	Errors []string
	// Warnings contains the validation warnings.
	Warnings []string
}

// GetEnvironmentForPrompt gets the environments formatted for use in prompts.
//
// Returns:
//   - []PromptEnvironment: The environments formatted for prompts.
func (m *EnvironmentManager) GetEnvironmentForPrompt() []PromptEnvironment {
	environments := make([]PromptEnvironment, 0, len(m.environments))

	for _, env := range m.environments {
		environments = append(environments, PromptEnvironment{
			Name:        env.Name,
			Type:        PromptEnvironmentType(env.Type),
			Description: env.Description,
			Default:     env.Default,
		})
	}

	return environments
}

// PromptEnvironment represents an environment formatted for use in prompts.
type PromptEnvironment struct {
	// Name is the name of the environment.
	Name string
	// Type is the type of environment.
	Type PromptEnvironmentType
	// Description is the description of the environment.
	Description string
	// Default indicates if this is the default environment.
	Default bool
}

// PromptEnvironmentType represents the type of environment for prompts.
type PromptEnvironmentType string

const (
	// PromptEnvironmentTypeDevelopment represents a development environment for prompts.
	PromptEnvironmentTypeDevelopment PromptEnvironmentType = "development"
	// PromptEnvironmentTypeStaging represents a staging environment for prompts.
	PromptEnvironmentTypeStaging PromptEnvironmentType = "staging"
	// PromptEnvironmentTypeProduction represents a production environment for prompts.
	PromptEnvironmentTypeProduction PromptEnvironmentType = "production"
)
