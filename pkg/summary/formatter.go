package summary

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ANSI color codes
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
)

// ColorFunctions holds all color formatting functions
type ColorFunctions struct {
	// Header colors
	HeaderColor     func(string) string
	SectionColor    func(string) string
	SubsectionColor func(string) string

	// Status colors
	SuccessColor func(string) string
	WarningColor func(string) string
	ErrorColor   func(string) string
	InfoColor    func(string) string

	// Value colors
	KeyColor   func(string) string
	ValueColor func(string) string
}

// Global color functions
var colors *ColorFunctions

// Initialize color settings based on environment
func init() {
	// Disable colors if NO_COLOR environment variable is set or if not a TTY
	if os.Getenv("NO_COLOR") != "" || !isTTY() {
		colors = newColorFunctions(false)
	} else {
		colors = newColorFunctions(true)
	}
}

// newColorFunctions creates a new ColorFunctions instance
func newColorFunctions(enabled bool) *ColorFunctions {
	if enabled {
		return &ColorFunctions{
			HeaderColor:     func(s string) string { return Bold + Cyan + s + Reset },
			SectionColor:    func(s string) string { return Bold + Blue + s + Reset },
			SubsectionColor: func(s string) string { return Bold + Magenta + s + Reset },
			SuccessColor:    func(s string) string { return Green + s + Reset },
			WarningColor:    func(s string) string { return Yellow + s + Reset },
			ErrorColor:      func(s string) string { return Red + s + Reset },
			InfoColor:       func(s string) string { return Cyan + s + Reset },
			KeyColor:        func(s string) string { return Cyan + s + Reset },
			ValueColor:      func(s string) string { return s },
		}
	}
	return &ColorFunctions{
		HeaderColor:     func(s string) string { return s },
		SectionColor:    func(s string) string { return s },
		SubsectionColor: func(s string) string { return s },
		SuccessColor:    func(s string) string { return s },
		WarningColor:    func(s string) string { return s },
		ErrorColor:      func(s string) string { return s },
		InfoColor:       func(s string) string { return s },
		KeyColor:        func(s string) string { return s },
		ValueColor:      func(s string) string { return s },
	}
}

// isTTY checks if stdout is a terminal
func isTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// FormatDeploymentSummary formats a deployment summary for display
func FormatDeploymentSummary(summary *DeploymentSummary) string {
	if summary == nil {
		return "No deployment summary available"
	}

	var builder strings.Builder

	// Header
	builder.WriteString(colors.HeaderColor("\n📋 Deployment Summary"))
	builder.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Metadata
	formatMetadata(&builder, &summary.Metadata)

	// Kubernetes and ArgoCD
	formatKubernetes(&builder, &summary.Kubernetes)
	formatArgoCD(&builder, &summary.ArgoCD)

	// Repository and Chart
	formatRepository(&builder, &summary.Repository)
	formatChart(&builder, &summary.Chart)

	// Application
	formatApplication(&builder, &summary.Application)

	// Environments
	formatEnvironments(&builder, summary.Environments)

	// Warnings and Notes
	if len(summary.Warnings) > 0 {
		formatWarnings(&builder, summary.Warnings)
	}

	if len(summary.Notes) > 0 {
		formatNotes(&builder, summary.Notes)
	}

	return builder.String()
}

// formatMetadata formats the metadata section
func formatMetadata(builder *strings.Builder, metadata *SummaryMetadata) {
	builder.WriteString(colors.SectionColor("📅 Metadata") + "\n")
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Generated At"),
		colors.ValueColor(metadata.GeneratedAt.Format(time.RFC1123))))
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Mode"),
		getModeString(metadata.DryRun, metadata.AutoApprove)))
	builder.WriteString("\n")
}

// formatKubernetes formats the Kubernetes section
func formatKubernetes(builder *strings.Builder, k8s *KubernetesSummary) {
	builder.WriteString(colors.SectionColor("☸️  Kubernetes") + "\n")
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Kubeconfig"),
		colors.ValueColor(k8s.KubeconfigPath)))

	if k8s.Context != "" {
		builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Context"),
			colors.ValueColor(k8s.Context)))
	}

	builder.WriteString("\n")
}

// formatArgoCD formats the ArgoCD section
func formatArgoCD(builder *strings.Builder, argocd *ArgoCDSummary) {
	builder.WriteString(colors.SectionColor("🚢 ArgoCD") + "\n")
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Namespace"),
		colors.ValueColor(argocd.Namespace)))

	status := "❌ Not Installed"
	if argocd.Installed {
		status = colors.SuccessColor("✅ Installed")
	}
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Status"), status))

	if argocd.Project != nil {
		builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Project"),
			colors.ValueColor(argocd.Project.Name)))
		if argocd.Project.Description != "" {
			builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Description"),
				colors.ValueColor(argocd.Project.Description)))
		}
	}

	builder.WriteString("\n")
}

// formatRepository formats the repository section
func formatRepository(builder *strings.Builder, repo *RepositorySummary) {
	builder.WriteString(colors.SectionColor("📦 Repository") + "\n")
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("URL"),
		colors.ValueColor(repo.URL)))

	if repo.Revision != nil {
		builder.WriteString(fmt.Sprintf("%s: %s (%s)\n", colors.KeyColor("Revision"),
			colors.ValueColor(repo.Revision.Value), repo.Revision.Type))
	}

	if repo.Auth.Method != "" && repo.Auth.Method != "none" {
		builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Authentication"),
			colors.ValueColor(repo.Auth.Method)))
	}

	builder.WriteString("\n")
}

// formatChart formats the chart section
func formatChart(builder *strings.Builder, chart *ChartSummary) {
	if chart.Name == "" {
		return
	}

	builder.WriteString(colors.SectionColor("📊 Helm Chart") + "\n")
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Name"),
		colors.ValueColor(chart.Name)))
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Version"),
		colors.ValueColor(chart.Version)))
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Path"),
		colors.ValueColor(chart.Path)))
	builder.WriteString("\n")
}

// formatApplication formats the application section
func formatApplication(builder *strings.Builder, app *ApplicationSummary) {
	builder.WriteString(colors.SectionColor("🚀 Application") + "\n")
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Name"),
		colors.ValueColor(app.Name)))
	builder.WriteString("\n")
}

// formatEnvironments formats the environments section
func formatEnvironments(builder *strings.Builder, environments []EnvironmentSummary) {
	if len(environments) == 0 {
		return
	}

	builder.WriteString(fmt.Sprintf(colors.SectionColor("🌍 Environments")+" (%d)\n", len(environments)))

	for i, env := range environments {
		builder.WriteString(fmt.Sprintf(colors.SubsectionColor("  Environment %d: %s")+"\n", i+1, env.Name))
		builder.WriteString(fmt.Sprintf("    %s: %s\n", colors.KeyColor("Namespace"),
			colors.ValueColor(env.Namespace)))
		builder.WriteString(fmt.Sprintf("    %s: %s\n", colors.KeyColor("Type"),
			colors.ValueColor(env.Config.Type)))

		if env.Config.Description != "" {
			builder.WriteString(fmt.Sprintf("    %s: %s\n", colors.KeyColor("Description"),
				colors.ValueColor(env.Config.Description)))
		}

		// Resources
		builder.WriteString(colors.SubsectionColor("    Resources:") + "\n")
		builder.WriteString(fmt.Sprintf("      %s: %d\n", colors.KeyColor("Replicas"),
			env.Resources.ReplicaCount))

		if env.Resources.CPURequest != "" || env.Resources.CPULimit != "" {
			builder.WriteString(fmt.Sprintf("      %s: ", colors.KeyColor("CPU")))
			if env.Resources.CPURequest != "" {
				builder.WriteString(colors.ValueColor(fmt.Sprintf("request=%s", env.Resources.CPURequest)))
			}
			if env.Resources.CPULimit != "" {
				if env.Resources.CPURequest != "" {
					builder.WriteString(", ")
				}
				builder.WriteString(colors.ValueColor(fmt.Sprintf("limit=%s", env.Resources.CPULimit)))
			}
			builder.WriteString("\n")
		}

		if env.Resources.MemoryRequest != "" || env.Resources.MemoryLimit != "" {
			builder.WriteString(fmt.Sprintf("      %s: ", colors.KeyColor("Memory")))
			if env.Resources.MemoryRequest != "" {
				builder.WriteString(colors.ValueColor(fmt.Sprintf("request=%s", env.Resources.MemoryRequest)))
			}
			if env.Resources.MemoryLimit != "" {
				if env.Resources.MemoryRequest != "" {
					builder.WriteString(", ")
				}
				builder.WriteString(colors.ValueColor(fmt.Sprintf("limit=%s", env.Resources.MemoryLimit)))
			}
			builder.WriteString("\n")
		}

		// ArgoCD Settings
		builder.WriteString(colors.SubsectionColor("    ArgoCD Settings:") + "\n")
		builder.WriteString(fmt.Sprintf("      %s: %s\n", colors.KeyColor("Auto Sync"),
			getBoolString(env.Config.AutoSync)))
		builder.WriteString(fmt.Sprintf("      %s: %s\n", colors.KeyColor("Prune"),
			getBoolString(env.Config.Prune)))
		builder.WriteString(fmt.Sprintf("      %s: %s\n", colors.KeyColor("Self Heal"),
			getBoolString(env.Config.SelfHeal)))

		// Custom Values
		if len(env.Values) > 0 {
			builder.WriteString(colors.SubsectionColor("    Custom Values:") + "\n")
			for k, v := range env.Values {
				builder.WriteString(fmt.Sprintf("      %s: %v\n", colors.KeyColor(k), v))
			}
		}

		builder.WriteString("\n")
	}
}

// formatWarnings formats the warnings section
func formatWarnings(builder *strings.Builder, warnings []string) {
	builder.WriteString(fmt.Sprintf(colors.WarningColor("⚠️  Warnings")+" (%d)\n", len(warnings)))
	for i, warning := range warnings {
		builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, warning))
	}
	builder.WriteString("\n")
}

// formatNotes formats the notes section
func formatNotes(builder *strings.Builder, notes []string) {
	builder.WriteString(fmt.Sprintf(colors.InfoColor("ℹ️  Notes")+" (%d)\n", len(notes)))
	for i, note := range notes {
		builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, note))
	}
	builder.WriteString("\n")
}

// FormatArgoCDCreateSummary formats an ArgoCD create summary for display
func FormatArgoCDCreateSummary(summary *ArgoCDCreateSummary) string {
	if summary == nil {
		return "No ArgoCD create summary available"
	}

	var builder strings.Builder

	// Header
	builder.WriteString(colors.HeaderColor("\n📋 ArgoCD Application Creation Summary"))
	builder.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Metadata
	formatMetadata(&builder, &summary.Metadata)

	// Kubernetes and ArgoCD
	formatKubernetes(&builder, &summary.Kubernetes)
	formatArgoCD(&builder, &summary.ArgoCD)

	// Application
	formatApplicationYAML(&builder, &summary.Application)

	// Warnings and Notes
	if len(summary.Warnings) > 0 {
		formatWarnings(&builder, summary.Warnings)
	}

	if len(summary.Notes) > 0 {
		formatNotes(&builder, summary.Notes)
	}

	return builder.String()
}

// formatApplicationYAML formats the application YAML section
func formatApplicationYAML(builder *strings.Builder, app *ApplicationYAMLSummary) {
	builder.WriteString(colors.SectionColor("🚀 Application") + "\n")
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Name"),
		colors.ValueColor(app.Name)))
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Namespace"),
		colors.ValueColor(app.Namespace)))
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Project"),
		colors.ValueColor(app.Project)))
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Source Repository"),
		colors.ValueColor(app.SourceRepo)))
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Source Path"),
		colors.ValueColor(app.SourcePath)))
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Destination"),
		colors.ValueColor(app.Destination)))
	builder.WriteString("\n")
}

// FormatEnvOperationSummary formats an environment operation summary for display
func FormatEnvOperationSummary(summary *EnvOperationSummary) string {
	if summary == nil {
		return "No environment operation summary available"
	}

	var builder strings.Builder

	// Header
	builder.WriteString(colors.HeaderColor("\n📋 Environment Operation Summary"))
	builder.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Metadata
	formatMetadata(&builder, &summary.Metadata)

	// Operation
	builder.WriteString(colors.SectionColor("🔧 Operation") + "\n")
	builder.WriteString(fmt.Sprintf("%s: %s\n", colors.KeyColor("Type"),
		colors.ValueColor(string(summary.Operation))))
	builder.WriteString("\n")

	// Environments
	formatEnvConfigs(&builder, summary.Environments)

	// Warnings and Notes
	if len(summary.Warnings) > 0 {
		formatWarnings(&builder, summary.Warnings)
	}

	if len(summary.Notes) > 0 {
		formatNotes(&builder, summary.Notes)
	}

	return builder.String()
}

// formatEnvConfigs formats environment configurations
func formatEnvConfigs(builder *strings.Builder, configs []EnvConfigSummary) {
	if len(configs) == 0 {
		return
	}

	builder.WriteString(fmt.Sprintf(colors.SectionColor("🌍 Environments")+" (%d)\n", len(configs)))

	for i, config := range configs {
		builder.WriteString(fmt.Sprintf(colors.SubsectionColor("  Environment %d: %s")+"\n", i+1, config.Name))
		builder.WriteString(fmt.Sprintf("    %s: %s\n", colors.KeyColor("Type"),
			colors.ValueColor(config.Type)))
		builder.WriteString(fmt.Sprintf("    %s: %s\n", colors.KeyColor("Namespace"),
			colors.ValueColor(config.Namespace)))

		if config.Description != "" {
			builder.WriteString(fmt.Sprintf("    %s: %s\n", colors.KeyColor("Description"),
				colors.ValueColor(config.Description)))
		}

		builder.WriteString(fmt.Sprintf("    %s: %d\n", colors.KeyColor("Replicas"),
			config.ReplicaCount))
		builder.WriteString(fmt.Sprintf("    %s: %s\n", colors.KeyColor("Auto Sync"),
			getBoolString(config.AutoSync)))
		builder.WriteString(fmt.Sprintf("    %s: %s\n", colors.KeyColor("Prune"),
			getBoolString(config.Prune)))
		builder.WriteString(fmt.Sprintf("    %s: %s\n", colors.KeyColor("Self Heal"),
			getBoolString(config.SelfHeal)))
		builder.WriteString("\n")
	}
}

// Helper functions

// getModeString returns a string representation of the deployment mode
func getModeString(dryRun, autoApprove bool) string {
	if dryRun {
		return colors.WarningColor("Dry Run")
	}
	if autoApprove {
		return colors.InfoColor("Auto Approved")
	}
	return colors.SuccessColor("Interactive")
}

// getBoolString returns a colored string representation of a boolean value
func getBoolString(b bool) string {
	if b {
		return colors.SuccessColor("Enabled")
	}
	return colors.ValueColor("Disabled")
}

// DisableColors disables all color formatting
func DisableColors() {
	colors = newColorFunctions(false)
}

// EnableColors enables color formatting (default)
func EnableColors() {
	colors = newColorFunctions(true)
}
