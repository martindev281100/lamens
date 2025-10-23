package summary

import (
	"time"

	"github.com/your-org/kargo-bootstrap/pkg/argocd"
	"github.com/your-org/kargo-bootstrap/pkg/config"
	"github.com/your-org/kargo-bootstrap/pkg/git"
	"github.com/your-org/kargo-bootstrap/pkg/prompt"
)

// DeploymentSummary contains all information about a deployment
type DeploymentSummary struct {
	// Metadata
	Metadata SummaryMetadata `json:"metadata"`

	// Kubernetes and ArgoCD configuration
	Kubernetes KubernetesSummary `json:"kubernetes"`
	ArgoCD     ArgoCDSummary     `json:"argocd"`

	// Git and Helm configuration
	Repository RepositorySummary `json:"repository"`
	Chart      ChartSummary      `json:"chart"`

	// Application configuration
	Application ApplicationSummary `json:"application"`

	// Environment configurations
	Environments []EnvironmentSummary `json:"environments"`

	// Warnings and notes
	Warnings []string `json:"warnings,omitempty"`
	Notes    []string `json:"notes,omitempty"`
}

// SummaryMetadata contains metadata about the summary
type SummaryMetadata struct {
	GeneratedAt time.Time `json:"generatedAt"`
	DryRun      bool      `json:"dryRun"`
	AutoApprove bool      `json:"autoApprove"`
}

// KubernetesSummary contains Kubernetes cluster information
type KubernetesSummary struct {
	KubeconfigPath string `json:"kubeconfigPath"`
	Context        string `json:"context,omitempty"`
}

// ArgoCDSummary contains ArgoCD configuration
type ArgoCDSummary struct {
	Namespace string          `json:"namespace"`
	Project   *argocd.Project `json:"project"`
	Installed bool            `json:"installed"`
}

// RepositorySummary contains Git repository information
type RepositorySummary struct {
	URL      string                    `json:"url"`
	Revision *prompt.RevisionSelection `json:"revision,omitempty"`
	Auth     AuthSummary               `json:"auth,omitempty"`
}

// AuthSummary contains authentication information
type AuthSummary struct {
	Method string `json:"method"` // token, username-password, ssh-key, none
	// Note: Don't include actual credentials in the summary
}

// ChartSummary contains Helm chart information
type ChartSummary struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// ApplicationSummary contains application configuration
type ApplicationSummary struct {
	Name string `json:"name"`
}

// EnvironmentSummary contains environment-specific configuration
type EnvironmentSummary struct {
	Name      string                    `json:"name"`
	Config    *config.EnvironmentConfig `json:"config"`
	Namespace string                    `json:"namespace"`
	Resources ResourceSummary           `json:"resources"`
	Values    map[string]interface{}    `json:"values,omitempty"`
}

// ResourceSummary contains resource configuration
type ResourceSummary struct {
	ReplicaCount  int    `json:"replicaCount"`
	CPURequest    string `json:"cpuRequest,omitempty"`
	CPULimit      string `json:"cpuLimit,omitempty"`
	MemoryRequest string `json:"memoryRequest,omitempty"`
	MemoryLimit   string `json:"memoryLimit,omitempty"`
}

// ArgoCDCreateSummary contains information for ArgoCD create command
type ArgoCDCreateSummary struct {
	Metadata    SummaryMetadata        `json:"metadata"`
	Kubernetes  KubernetesSummary      `json:"kubernetes"`
	ArgoCD      ArgoCDSummary          `json:"argocd"`
	Application ApplicationYAMLSummary `json:"application"`
	Warnings    []string               `json:"warnings,omitempty"`
	Notes       []string               `json:"notes,omitempty"`
}

// ApplicationYAMLSummary contains application information from YAML
type ApplicationYAMLSummary struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Project     string `json:"project"`
	SourceRepo  string `json:"sourceRepo"`
	SourcePath  string `json:"sourcePath"`
	Destination string `json:"destination"`
}

// EnvOperationSummary contains information for environment operations
type EnvOperationSummary struct {
	Metadata     SummaryMetadata    `json:"metadata"`
	Operation    EnvOperationType   `json:"operation"`
	Environments []EnvConfigSummary `json:"environments"`
	Warnings     []string           `json:"warnings,omitempty"`
	Notes        []string           `json:"notes,omitempty"`
}

// EnvOperationType represents the type of environment operation
type EnvOperationType string

const (
	EnvOperationList     EnvOperationType = "list"
	EnvOperationAdd      EnvOperationType = "add"
	EnvOperationRemove   EnvOperationType = "remove"
	EnvOperationValidate EnvOperationType = "validate"
)

// EnvConfigSummary contains environment configuration summary
type EnvConfigSummary struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	Namespace    string `json:"namespace"`
	ReplicaCount int    `json:"replicaCount"`
	AutoSync     bool   `json:"autoSync"`
	Prune        bool   `json:"prune"`
	SelfHeal     bool   `json:"selfHeal"`
}

// SummaryBuilder helps build deployment summaries
type SummaryBuilder struct {
	summary *DeploymentSummary
}

// NewSummaryBuilder creates a new summary builder
func NewSummaryBuilder() *SummaryBuilder {
	return &SummaryBuilder{
		summary: &DeploymentSummary{
			Metadata: SummaryMetadata{
				GeneratedAt: time.Now(),
			},
			Environments: make([]EnvironmentSummary, 0),
			Warnings:     make([]string, 0),
			Notes:        make([]string, 0),
		},
	}
}

// WithMetadata sets the summary metadata
func (b *SummaryBuilder) WithMetadata(dryRun, autoApprove bool) *SummaryBuilder {
	b.summary.Metadata.DryRun = dryRun
	b.summary.Metadata.AutoApprove = autoApprove
	return b
}

// WithKubernetes sets the Kubernetes information
func (b *SummaryBuilder) WithKubernetes(kubeconfigPath, context string) *SummaryBuilder {
	b.summary.Kubernetes = KubernetesSummary{
		KubeconfigPath: kubeconfigPath,
		Context:        context,
	}
	return b
}

// WithArgoCD sets the ArgoCD information
func (b *SummaryBuilder) WithArgoCD(namespace string, project *argocd.Project, installed bool) *SummaryBuilder {
	b.summary.ArgoCD = ArgoCDSummary{
		Namespace: namespace,
		Project:   project,
		Installed: installed,
	}
	return b
}

// WithRepository sets the repository information
func (b *SummaryBuilder) WithRepository(url string, revision *prompt.RevisionSelection, authMethod string) *SummaryBuilder {
	auth := AuthSummary{Method: authMethod}
	if authMethod == "" {
		auth.Method = "none"
	}

	b.summary.Repository = RepositorySummary{
		URL:      url,
		Revision: revision,
		Auth:     auth,
	}
	return b
}

// WithChart sets the chart information
func (b *SummaryBuilder) WithChart(chart *git.ChartInfo) *SummaryBuilder {
	if chart != nil {
		b.summary.Chart = ChartSummary{
			Name:    chart.Name,
			Version: chart.Version,
			Path:    chart.Path,
		}
	}
	return b
}

// WithApplication sets the application information
func (b *SummaryBuilder) WithApplication(name string) *SummaryBuilder {
	b.summary.Application = ApplicationSummary{
		Name: name,
	}
	return b
}

// AddEnvironment adds an environment to the summary
func (b *SummaryBuilder) AddEnvironment(name string, config *config.EnvironmentConfig, namespace string, values map[string]interface{}) *SummaryBuilder {
	env := EnvironmentSummary{
		Name:      name,
		Config:    config,
		Namespace: namespace,
		Values:    values,
		Resources: ResourceSummary{
			ReplicaCount:  config.ReplicaCount,
			CPURequest:    config.CPURequest,
			CPULimit:      config.CPULimit,
			MemoryRequest: config.MemoryRequest,
			MemoryLimit:   config.MemoryLimit,
		},
	}

	b.summary.Environments = append(b.summary.Environments, env)
	return b
}

// AddWarning adds a warning to the summary
func (b *SummaryBuilder) AddWarning(warning string) *SummaryBuilder {
	b.summary.Warnings = append(b.summary.Warnings, warning)
	return b
}

// AddNote adds a note to the summary
func (b *SummaryBuilder) AddNote(note string) *SummaryBuilder {
	b.summary.Notes = append(b.summary.Notes, note)
	return b
}

// Build builds the deployment summary
func (b *SummaryBuilder) Build() *DeploymentSummary {
	return b.summary
}

// ArgoCDCreateSummaryBuilder helps build ArgoCD create summaries
type ArgoCDCreateSummaryBuilder struct {
	summary *ArgoCDCreateSummary
}

// NewArgoCDCreateSummaryBuilder creates a new ArgoCD create summary builder
func NewArgoCDCreateSummaryBuilder() *ArgoCDCreateSummaryBuilder {
	return &ArgoCDCreateSummaryBuilder{
		summary: &ArgoCDCreateSummary{
			Metadata: SummaryMetadata{
				GeneratedAt: time.Now(),
			},
			Warnings: make([]string, 0),
			Notes:    make([]string, 0),
		},
	}
}

// WithMetadata sets the summary metadata
func (b *ArgoCDCreateSummaryBuilder) WithMetadata(dryRun, autoApprove bool) *ArgoCDCreateSummaryBuilder {
	b.summary.Metadata.DryRun = dryRun
	b.summary.Metadata.AutoApprove = autoApprove
	return b
}

// WithKubernetes sets the Kubernetes information
func (b *ArgoCDCreateSummaryBuilder) WithKubernetes(kubeconfigPath, context string) *ArgoCDCreateSummaryBuilder {
	b.summary.Kubernetes = KubernetesSummary{
		KubeconfigPath: kubeconfigPath,
		Context:        context,
	}
	return b
}

// WithArgoCD sets the ArgoCD information
func (b *ArgoCDCreateSummaryBuilder) WithArgoCD(namespace string, installed bool) *ArgoCDCreateSummaryBuilder {
	b.summary.ArgoCD = ArgoCDSummary{
		Namespace: namespace,
		Installed: installed,
	}
	return b
}

// WithApplicationYAML sets the application YAML information
func (b *ArgoCDCreateSummaryBuilder) WithApplicationYAML(name, namespace, project, sourceRepo, sourcePath, destination string) *ArgoCDCreateSummaryBuilder {
	b.summary.Application = ApplicationYAMLSummary{
		Name:        name,
		Namespace:   namespace,
		Project:     project,
		SourceRepo:  sourceRepo,
		SourcePath:  sourcePath,
		Destination: destination,
	}
	return b
}

// AddWarning adds a warning to the summary
func (b *ArgoCDCreateSummaryBuilder) AddWarning(warning string) *ArgoCDCreateSummaryBuilder {
	b.summary.Warnings = append(b.summary.Warnings, warning)
	return b
}

// AddNote adds a note to the summary
func (b *ArgoCDCreateSummaryBuilder) AddNote(note string) *ArgoCDCreateSummaryBuilder {
	b.summary.Notes = append(b.summary.Notes, note)
	return b
}

// Build builds the ArgoCD create summary
func (b *ArgoCDCreateSummaryBuilder) Build() *ArgoCDCreateSummary {
	return b.summary
}
