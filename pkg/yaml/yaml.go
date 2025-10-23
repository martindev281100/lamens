package yaml

// Package yaml provides YAML processing functionality for kargo-bootstrap

import (
	"fmt"
	"text/template"
	"time"

	"github.com/your-org/kargo-bootstrap/pkg/config"
)

// Processor represents a YAML processor
type Processor struct {
	// Template functions for YAML processing
	funcMap template.FuncMap
	// Environment manager for multi-environment support
	envManager *config.EnvironmentManager
}

// NewProcessor creates a new YAML processor
func NewProcessor() (*Processor, error) {
	envManager := config.NewEnvironmentManager()
	envManager.LoadDefaultEnvironments()

	return &Processor{
		funcMap:    TemplateFuncs(),
		envManager: envManager,
	}, nil
}

// NewProcessorWithEnvManager creates a new YAML processor with a custom environment manager
func NewProcessorWithEnvManager(envManager *config.EnvironmentManager) (*Processor, error) {
	if envManager == nil {
		return nil, fmt.Errorf("environment manager is required")
	}

	return &Processor{
		funcMap:    TemplateFuncs(),
		envManager: envManager,
	}, nil
}

// RenderApplication renders an Argo CD Application as YAML
func (p *Processor) RenderApplication(config ApplicationConfig, options RenderOptions) ([]byte, error) {
	return RenderApplicationYAML(config, options)
}

// ValidateApplication validates an Application configuration
func (p *Processor) ValidateApplication(config ApplicationConfig) ValidationResult {
	yamlBytes, err := RenderApplicationYAML(config, RenderOptions{Validate: false})
	if err != nil {
		return ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("Failed to render YAML: %v", err)},
		}
	}
	return ValidateApplicationYAML(yamlBytes)
}

// CreateApplicationTemplate creates a new application template
func (p *Processor) CreateApplicationTemplate(name, description string) ApplicationTemplate {
	return CreateApplicationTemplate(name, description)
}

// MergeHelmValues merges default Helm values with user overrides
func (p *Processor) MergeHelmValues(defaultValues, userValues map[string]interface{}) map[string]interface{} {
	return MergeHelmValues(defaultValues, userValues)
}

// GenerateHelmValues generates Helm values for an application
func (p *Processor) GenerateHelmValues(appName, environment, imageRepository, imageTag string, envSecret string, uptraceConfig *UptraceConfig) map[string]interface{} {
	// Get environment configuration
	envConfig, err := p.envManager.GetEnvironment(environment)
	if err != nil {
		// Fall back to basic values if environment not found
		envConfig = &config.EnvironmentConfig{
			Name:         environment,
			ReplicaCount: 1,
			Values:       make(map[string]interface{}),
		}
	}

	// Generate Helm values using environment configuration
	values := map[string]interface{}{
		appName: map[string]interface{}{
			"replicaCount": envConfig.ReplicaCount,
			"port":         80,
			"image": map[string]interface{}{
				"repository": imageRepository,
				"tag":        imageTag,
			},
		},
	}

	// Add image pull secrets if specified
	if envSecret != "" {
		appValues := values[appName].(map[string]interface{})
		appValues["imagePullSecrets"] = []string{envSecret}
		appValues["envSecret"] = envSecret
	}

	// Add pod labels
	appValues := values[appName].(map[string]interface{})
	appValues["podLabels"] = map[string]interface{}{
		"app":         appName,
		"environment": environment,
		"version":     imageTag,
	}

	// Add environment-specific labels from config
	for k, v := range envConfig.Labels {
		appValues["podLabels"].(map[string]interface{})[k] = v
	}

	// Add uptrace configuration if provided
	if uptraceConfig != nil && uptraceConfig.Enabled {
		appValues := values[appName].(map[string]interface{})
		appValues["uptrace"] = map[string]interface{}{
			"enabled":     uptraceConfig.Enabled,
			"serviceName": uptraceConfig.ServiceName,
			"endpoint":    uptraceConfig.Endpoint,
			"headers":     uptraceConfig.Headers,
		}
	}

	// Merge environment-specific values
	for k, v := range envConfig.Values {
		appValues := values[appName].(map[string]interface{})
		appValues[k] = v
	}

	return values
}

// GenerateApplicationConfig generates an ApplicationConfig for the given parameters
func (p *Processor) GenerateApplicationConfig(appName, project, repoURL, path, targetRevision, namespace, environment string, helmValues map[string]interface{}) ApplicationConfig {
	// Get environment configuration
	envConfig, err := p.envManager.GetEnvironment(environment)
	if err != nil {
		// Fall back to basic config if environment not found
		envConfig = &config.EnvironmentConfig{
			Name:     environment,
			AutoSync: true,
			Prune:    true,
			SelfHeal: true,
			Labels:   make(map[string]string),
		}
	}

	// Use environment-specific target revision if provided
	revision := targetRevision
	if envConfig.TargetRevision != "" {
		revision = envConfig.TargetRevision
	}

	// Create application config
	config := ApplicationConfig{
		Name:    fmt.Sprintf("%s-%s", appName, environment),
		Project: project,
		Source: ApplicationSource{
			RepoURL:        repoURL,
			Path:           path,
			TargetRevision: revision,
			Helm: &HelmConfig{
				Values: helmValues,
			},
		},
		Destination: ApplicationDestination{
			Server:    "https://kubernetes.default.svc",
			Namespace: namespace,
		},
		Labels: map[string]string{
			"app":                           appName,
			"environment":                   environment,
			"kargo-bootstrap.io/managed":    "true",
			"app.kubernetes.io/name":        appName,
			"app.kubernetes.io/component":   "application",
			"app.kubernetes.io/environment": environment,
		},
		Annotations: map[string]string{
			"kargo-bootstrap.io/app-name":    appName,
			"kargo-bootstrap.io/environment": environment,
			"kargo-bootstrap.io/project":     project,
			"kargo-bootstrap.io/repository":  repoURL,
			"kargo-bootstrap.io/chart-path":  path,
			"kargo-bootstrap.io/created-at":  time.Now().Format(time.RFC3339),
			"kargo-bootstrap.io/managed-by":  "kargo-bootstrap",
		},
		SyncPolicy: &ApplicationSyncPolicy{
			Automated: &AutomatedSyncPolicy{
				Prune:    envConfig.Prune,
				SelfHeal: envConfig.SelfHeal,
			},
			SyncOptions: []string{"CreateNamespace=true"},
		},
	}

	// Add environment-specific labels
	for k, v := range envConfig.Labels {
		config.Labels[k] = v
	}

	// Add environment-specific annotations
	for k, v := range envConfig.Annotations {
		config.Annotations[k] = v
	}

	// Disable automated sync if environment doesn't allow it
	if !envConfig.AutoSync {
		config.SyncPolicy.Automated = nil
	}

	return config
}

// GetEnvironmentManager returns the environment manager
func (p *Processor) GetEnvironmentManager() *config.EnvironmentManager {
	return p.envManager
}

// SetEnvironmentManager sets a new environment manager
func (p *Processor) SetEnvironmentManager(envManager *config.EnvironmentManager) {
	p.envManager = envManager
}

// GenerateEnvironmentSpecificApplications generates application configurations for multiple environments
func (p *Processor) GenerateEnvironmentSpecificApplications(appName, project, repoURL, chartPath, targetRevision string, environments []string) ([]ApplicationConfig, error) {
	var configs []ApplicationConfig

	for _, envName := range environments {
		// Validate environment exists
		_, err := p.envManager.GetEnvironment(envName)
		if err != nil {
			return nil, fmt.Errorf("environment '%s' not found: %w", envName, err)
		}

		// Generate namespace name
		namespaceName := fmt.Sprintf("%s-%s", appName, envName)

		// Generate Helm values for this environment
		helmValues := p.GenerateHelmValues(
			appName,
			envName,
			"ethannguyen98/"+appName,      // Default image repository
			"v0.0.1",                      // Default image tag
			appName+"-"+envName+"-secret", // Default secret name
			&UptraceConfig{
				Enabled:     true,
				ServiceName: appName + "-" + envName,
				Endpoint:    "http://uptrace-collector.uptrace.svc:4318",
				Headers: map[string]string{
					"uptrace-dsn": "http://token@uptrace-collector.uptrace.svc:4318/2",
				},
			},
		)

		// Generate application config
		appConfig := p.GenerateApplicationConfig(
			appName,
			project,
			repoURL,
			chartPath,
			targetRevision,
			namespaceName,
			envName,
			helmValues,
		)

		configs = append(configs, appConfig)
	}

	return configs, nil
}

// RenderEnvironmentSpecificApplications renders ArgoCD Application YAMLs for multiple environments
func (p *Processor) RenderEnvironmentSpecificApplications(appName, project, repoURL, chartPath, targetRevision string, environments []string, options RenderOptions) (map[string][]byte, error) {
	configs, err := p.GenerateEnvironmentSpecificApplications(appName, project, repoURL, chartPath, targetRevision, environments)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for _, config := range configs {
		yamlBytes, err := p.RenderApplication(config, options)
		if err != nil {
			return nil, fmt.Errorf("failed to render application '%s': %w", config.Name, err)
		}
		result[config.Name] = yamlBytes
	}

	return result, nil
}

// ValidateEnvironmentSetup validates the setup for multiple environments
func (p *Processor) ValidateEnvironmentSetup(environments []string) config.EnvironmentValidationResult {
	return p.envManager.ValidateEnvironmentSetup(environments)
}
