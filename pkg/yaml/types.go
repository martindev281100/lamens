package yaml

import (
	"time"
)

// ApplicationConfig represents the configuration for an Argo CD Application
type ApplicationConfig struct {
	// Basic application information
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Argo CD project configuration
	Project string `yaml:"project" json:"project"`

	// Source configuration
	Source ApplicationSource `yaml:"source" json:"source"`

	// Destination configuration
	Destination ApplicationDestination `yaml:"destination" json:"destination"`

	// Sync policy configuration
	SyncPolicy *ApplicationSyncPolicy `yaml:"syncPolicy,omitempty" json:"syncPolicy,omitempty"`

	// Labels and annotations
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// ApplicationSource represents the source configuration for an Argo CD Application
type ApplicationSource struct {
	RepoURL        string      `yaml:"repoURL" json:"repoURL"`
	Path           string      `yaml:"path" json:"path"`
	TargetRevision string      `yaml:"targetRevision" json:"targetRevision"`
	Helm           *HelmConfig `yaml:"helm,omitempty" json:"helm,omitempty"`
}

// ApplicationDestination represents the destination configuration for an Argo CD Application
type ApplicationDestination struct {
	Server    string `yaml:"server" json:"server"`
	Namespace string `yaml:"namespace" json:"namespace"`
}

// ApplicationSyncPolicy represents the sync policy for an Argo CD Application
type ApplicationSyncPolicy struct {
	Automated   *AutomatedSyncPolicy `yaml:"automated,omitempty" json:"automated,omitempty"`
	SyncOptions []string             `yaml:"syncOptions,omitempty" json:"syncOptions,omitempty"`
	Retry       *RetryStrategy       `yaml:"retry,omitempty" json:"retry,omitempty"`
}

// AutomatedSyncPolicy represents the automated sync policy
type AutomatedSyncPolicy struct {
	Prune    bool `yaml:"prune" json:"prune"`
	SelfHeal bool `yaml:"selfHeal" json:"selfHeal"`
}

// RetryStrategy represents the retry strategy for sync
type RetryStrategy struct {
	Limit   int64    `yaml:"limit" json:"limit"`
	Backoff *Backoff `yaml:"backoff,omitempty" json:"backoff,omitempty"`
}

// Backoff represents the backoff configuration
type Backoff struct {
	Duration    string `yaml:"duration" json:"duration"`
	Factor      int64  `yaml:"factor" json:"factor"`
	MaxDuration string `yaml:"maxDuration" json:"maxDuration"`
}

// HelmConfig represents the Helm configuration for an Argo CD Application
type HelmConfig struct {
	ValueFiles  []string    `yaml:"valueFiles,omitempty" json:"valueFiles,omitempty"`
	Values      interface{} `yaml:"values,omitempty" json:"values,omitempty"`
	Parameters  []HelmParam `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	ReleaseName string      `yaml:"releaseName,omitempty" json:"releaseName,omitempty"`
}

// HelmParam represents a Helm parameter
type HelmParam struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}

// HelmValues represents the Helm values configuration
type HelmValues struct {
	Values map[string]interface{} `yaml:"values" json:"values"`

	// Environment-specific overrides
	Environments map[string]map[string]interface{} `yaml:"environments,omitempty" json:"environments,omitempty"`
}

// ApplicationTemplate represents a template configuration for Argo CD Applications
type ApplicationTemplate struct {
	// Template metadata
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Version     string `yaml:"version" json:"version"`

	// Default values for the template
	DefaultValues ApplicationConfig `yaml:"defaultValues" json:"defaultValues"`

	// Template parameters
	Parameters []TemplateParameter `yaml:"parameters,omitempty" json:"parameters,omitempty"`

	// Template creation time
	CreatedAt time.Time `yaml:"createdAt" json:"createdAt"`

	// Template update time
	UpdatedAt time.Time `yaml:"updatedAt" json:"updatedAt"`
}

// TemplateParameter represents a parameter in a template
type TemplateParameter struct {
	Name        string      `yaml:"name" json:"name"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Type        string      `yaml:"type" json:"type"` // string, number, boolean, array, object
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	Required    bool        `yaml:"required" json:"required"`
}

// RenderOptions represents options for rendering YAML
type RenderOptions struct {
	// Dry run mode - only generate YAML without applying
	DryRun bool `yaml:"dryRun" json:"dryRun"`

	// Validate the generated YAML
	Validate bool `yaml:"validate" json:"validate"`

	// Output format (yaml, json)
	OutputFormat string `yaml:"outputFormat" json:"outputFormat"`

	// Include comments in the output
	IncludeComments bool `yaml:"includeComments" json:"includeComments"`

	// Indentation level for YAML output
	Indent int `yaml:"indent" json:"indent"`
}

// ValidationResult represents the result of YAML validation
type ValidationResult struct {
	Valid    bool     `yaml:"valid" json:"valid"`
	Errors   []string `yaml:"errors,omitempty" json:"errors,omitempty"`
	Warnings []string `yaml:"warnings,omitempty" json:"warnings,omitempty"`
}

// EnvironmentConfig represents environment-specific configuration
type EnvironmentConfig struct {
	Name      string                 `yaml:"name" json:"name"`
	Namespace string                 `yaml:"namespace" json:"namespace"`
	Server    string                 `yaml:"server,omitempty" json:"server,omitempty"`
	Values    map[string]interface{} `yaml:"values,omitempty" json:"values,omitempty"`

	// Environment-specific labels and annotations
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// DockerRegistryConfig represents Docker registry configuration
type DockerRegistryConfig struct {
	Server   string `yaml:"server" json:"server"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	Email    string `yaml:"email,omitempty" json:"email,omitempty"`
}

// UptraceConfig represents Uptrace/OTEL configuration
type UptraceConfig struct {
	Enabled       bool              `yaml:"enabled" json:"enabled"`
	ServiceName   string            `yaml:"serviceName" json:"serviceName"`
	Endpoint      string            `yaml:"endpoint" json:"endpoint"`
	Headers       map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	ProjectName   string            `yaml:"projectName,omitempty" json:"projectName,omitempty"`
	DSN           string            `yaml:"dsn,omitempty" json:"dsn,omitempty"`
	SampleRate    float64           `yaml:"sampleRate,omitempty" json:"sampleRate,omitempty"`
	ServicePrefix string            `yaml:"servicePrefix,omitempty" json:"servicePrefix,omitempty"`
	Attributes    map[string]string `yaml:"attributes,omitempty" json:"attributes,omitempty"`
}
