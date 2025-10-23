package yaml

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"sigs.k8s.io/yaml"
)

// applicationTemplate is the Go template for Argo CD Application
const applicationTemplate = `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{.Name}}
  namespace: argocd
  {{- if .Labels }}
  labels:
    {{- range $key, $value := .Labels }}
    {{ $key }}: {{ $value }}
    {{- end }}
  {{- end }}
  {{- if .Annotations }}
  annotations:
    {{- range $key, $value := .Annotations }}
    {{ $key }}: {{ $value }}
    {{- end }}
  {{- end }}
spec:
  project: {{.Project}}
  source:
    repoURL: {{.Source.RepoURL}}
    path: {{.Source.Path}}
    targetRevision: {{.Source.TargetRevision}}
    {{- if .Source.Helm }}
    helm:
      {{- if .Source.Helm.ValueFiles }}
      valueFiles:
        {{- range .Source.Helm.ValueFiles }}
        - {{ . }}
        {{- end }}
      {{- end }}
      {{- if .Source.Helm.Values }}
      values: |
{{ .Source.Helm.Values | indent 8 }}
      {{- end }}
      {{- if .Source.Helm.Parameters }}
      parameters:
        {{- range .Source.Helm.Parameters }}
        - name: {{ .Name }}
          value: {{ .Value }}
        {{- end }}
      {{- end }}
      {{- if .Source.Helm.ReleaseName }}
      releaseName: {{ .Source.Helm.ReleaseName }}
      {{- end }}
    {{- end }}
  destination:
    server: {{.Destination.Server}}
    namespace: {{.Destination.Namespace}}
  {{- if .SyncPolicy }}
  syncPolicy:
    {{- if .SyncPolicy.Automated }}
    automated:
      prune: {{ .SyncPolicy.Automated.Prune }}
      selfHeal: {{ .SyncPolicy.Automated.SelfHeal }}
    {{- end }}
    {{- if .SyncPolicy.SyncOptions }}
    syncOptions:
      {{- range .SyncPolicy.SyncOptions }}
      - {{ . }}
      {{- end }}
    {{- end }}
    {{- if .SyncPolicy.Retry }}
    retry:
      limit: {{ .SyncPolicy.Retry.Limit }}
      {{- if .SyncPolicy.Retry.Backoff }}
      backoff:
        duration: {{ .SyncPolicy.Retry.Backoff.Duration }}
        factor: {{ .SyncPolicy.Retry.Backoff.Factor }}
        maxDuration: {{ .SyncPolicy.Retry.Backoff.MaxDuration }}
      {{- end }}
    {{- end }}
  {{- end }}
`

// helmValuesTemplate is the template for inline Helm values
const helmValuesTemplate = `{{ .ChartName }}:
  replicaCount: {{ .ReplicaCount }}
  port: {{ .Port }}
  image:
    repository: {{ .ImageRepository }}
    tag: {{ .ImageTag }}
  {{- if .ImagePullSecrets }}
  imagePullSecrets:
    {{- range .ImagePullSecrets }}
    - name: {{ . }}
    {{- end }}
  {{- end }}
  {{- if .EnvSecret }}
  envSecret: {{ .EnvSecret }}
  {{- end }}
  {{- if .PodLabels }}
  podLabels:
    {{- range $key, $value := .PodLabels }}
    {{ $key }}: {{ $value }}
    {{- end }}
  {{- end }}
  {{- if .EnvVars }}
  env:
    {{- range .EnvVars }}
    - name: {{ .Name }}
      value: {{ .Value }}
    {{- end }}
  {{- end }}
`

// RenderApplicationYAML renders an Argo CD Application as YAML
func RenderApplicationYAML(config ApplicationConfig, options RenderOptions) ([]byte, error) {
	// Validate required fields
	if err := validateApplicationConfig(config); err != nil {
		return nil, WrapError(err, "validation failed")
	}

	// Set default values if not provided
	setDefaults(&config)

	// Parse and execute the template
	tmpl, err := template.New("application").Funcs(template.FuncMap{
		"indent": indent,
	}).Parse(applicationTemplate)
	if err != nil {
		return nil, NewTemplateError("failed to parse application template", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, NewTemplateError("failed to execute application template", err)
	}

	yamlBytes := buf.Bytes()

	// Validate the generated YAML if requested
	if options.Validate {
		if validation := ValidateApplicationYAML(yamlBytes); !validation.Valid {
			collector := NewErrorCollector()
			for _, errMsg := range validation.Errors {
				collector.Add(NewValidationError(errMsg, "", ""))
			}
			return nil, collector.ToError()
		}
	}

	return yamlBytes, nil
}

// CreateApplicationTemplate creates a template for Application resources
func CreateApplicationTemplate(name, description string) ApplicationTemplate {
	now := time.Now()

	return ApplicationTemplate{
		Name:        name,
		Description: description,
		Version:     "1.0.0",
		DefaultValues: ApplicationConfig{
			Source: ApplicationSource{
				TargetRevision: "main",
				Helm: &HelmConfig{
					ValueFiles: []string{"values.yaml"},
				},
			},
			Destination: ApplicationDestination{
				Server: "https://kubernetes.default.svc",
			},
			SyncPolicy: &ApplicationSyncPolicy{
				Automated: &AutomatedSyncPolicy{
					Prune:    true,
					SelfHeal: true,
				},
				SyncOptions: []string{
					"CreateNamespace=true",
				},
			},
		},
		Parameters: []TemplateParameter{
			{
				Name:        "name",
				Description: "Application name",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "project",
				Description: "Argo CD project name",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "repoURL",
				Description: "Git repository URL",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "path",
				Description: "Path to the Helm chart",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "namespace",
				Description: "Target namespace",
				Type:        "string",
				Required:    true,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// MergeHelmValues merges default values with user-provided overrides
func MergeHelmValues(defaultValues, userValues map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy default values
	for k, v := range defaultValues {
		result[k] = v
	}

	// Merge user values, overriding defaults
	for k, v := range userValues {
		result[k] = v
	}

	return result
}

// ValidateApplicationYAML validates the generated YAML
func ValidateApplicationYAML(yamlBytes []byte) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Parse the YAML to check for syntax errors
	var obj map[string]interface{}
	if err := yaml.Unmarshal(yamlBytes, &obj); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("YAML syntax error: %v", err))
		return result
	}

	// Check for required fields
	if apiVersion, ok := obj["apiVersion"]; !ok || apiVersion != "argoproj.io/v1alpha1" {
		result.Warnings = append(result.Warnings, "Missing or invalid apiVersion, expected 'argoproj.io/v1alpha1'")
	}

	if kind, ok := obj["kind"]; !ok || kind != "Application" {
		result.Warnings = append(result.Warnings, "Missing or invalid kind, expected 'Application'")
	}

	metadata, ok := obj["metadata"].(map[string]interface{})
	if !ok {
		result.Errors = append(result.Errors, "Missing or invalid metadata section")
		result.Valid = false
	} else {
		if _, ok := metadata["name"]; !ok {
			result.Errors = append(result.Errors, "Missing required field: metadata.name")
			result.Valid = false
		}
	}

	spec, ok := obj["spec"].(map[string]interface{})
	if !ok {
		result.Errors = append(result.Errors, "Missing or invalid spec section")
		result.Valid = false
	} else {
		requiredSpecFields := []string{"project", "source", "destination"}
		for _, field := range requiredSpecFields {
			if _, ok := spec[field]; !ok {
				result.Errors = append(result.Errors, fmt.Sprintf("Missing required field: spec.%s", field))
				result.Valid = false
			}
		}
	}

	return result
}

// RenderHelmValues renders inline Helm values
func RenderHelmValues(chartName, imageRepository, imageTag, envSecret string, replicaCount int, port int, imagePullSecrets []string, podLabels map[string]string, envVars []map[string]string) ([]byte, error) {
	data := struct {
		ChartName        string
		ImageRepository  string
		ImageTag         string
		EnvSecret        string
		ReplicaCount     int
		Port             int
		ImagePullSecrets []string
		PodLabels        map[string]string
		EnvVars          []map[string]string
	}{
		ChartName:        chartName,
		ImageRepository:  imageRepository,
		ImageTag:         imageTag,
		EnvSecret:        envSecret,
		ReplicaCount:     replicaCount,
		Port:             port,
		ImagePullSecrets: imagePullSecrets,
		PodLabels:        podLabels,
		EnvVars:          envVars,
	}

	tmpl, err := template.New("helmValues").Parse(helmValuesTemplate)
	if err != nil {
		return nil, NewTemplateError("failed to parse helm values template", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, NewTemplateError("failed to execute helm values template", err)
	}

	return buf.Bytes(), nil
}

// indent is a template function to indent text
func indent(spaces int, text string) string {
	padding := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = padding + line
		}
	}
	return strings.Join(lines, "\n")
}

// validateApplicationConfig validates the application configuration
func validateApplicationConfig(config ApplicationConfig) error {
	collector := NewErrorCollector()

	if err := ValidateRequiredField(config.Name, "name"); err != nil {
		collector.Add(err.(*YamlError))
	}

	if err := ValidateRequiredField(config.Project, "project"); err != nil {
		collector.Add(err.(*YamlError))
	}

	if err := ValidateRequiredField(config.Source.RepoURL, "source.repoURL"); err != nil {
		collector.Add(err.(*YamlError))
	} else if err := ValidateURL(config.Source.RepoURL, "source.repoURL"); err != nil {
		collector.Add(err.(*YamlError))
	}

	if err := ValidateRequiredField(config.Source.Path, "source.path"); err != nil {
		collector.Add(err.(*YamlError))
	}

	if err := ValidateRequiredField(config.Destination.Server, "destination.server"); err != nil {
		collector.Add(err.(*YamlError))
	} else if err := ValidateURL(config.Destination.Server, "destination.server"); err != nil {
		collector.Add(err.(*YamlError))
	}

	if err := ValidateRequiredField(config.Destination.Namespace, "destination.namespace"); err != nil {
		collector.Add(err.(*YamlError))
	} else if err := ValidateKubernetesName(config.Destination.Namespace, "destination.namespace"); err != nil {
		collector.Add(err.(*YamlError))
	}

	return collector.ToError()
}

// setDefaults sets default values for the application configuration
func setDefaults(config *ApplicationConfig) {
	if config.Source.TargetRevision == "" {
		config.Source.TargetRevision = "main"
	}

	if config.Destination.Server == "" {
		config.Destination.Server = "https://kubernetes.default.svc"
	}

	if config.SyncPolicy == nil {
		config.SyncPolicy = &ApplicationSyncPolicy{
			Automated: &AutomatedSyncPolicy{
				Prune:    true,
				SelfHeal: true,
			},
			SyncOptions: []string{"CreateNamespace=true"},
		}
	} else if config.SyncPolicy.Automated == nil {
		config.SyncPolicy.Automated = &AutomatedSyncPolicy{
			Prune:    true,
			SelfHeal: true,
		}
	}

	if config.Labels == nil {
		config.Labels = make(map[string]string)
	}

	if config.Annotations == nil {
		config.Annotations = make(map[string]string)
	}
}
