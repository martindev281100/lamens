package yaml

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/your-org/kargo-bootstrap/pkg/errors"
	"go.yaml.in/yaml/v2"
)

// TemplateFuncs returns a map of template functions for use in YAML templates
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"indent":       indent,
		"nindent":      nindent,
		"trim":         strings.TrimSpace,
		"upper":        strings.ToUpper,
		"lower":        strings.ToLower,
		"title":        strings.Title,
		"replace":      strings.ReplaceAll,
		"contains":     strings.Contains,
		"hasPrefix":    strings.HasPrefix,
		"hasSuffix":    strings.HasSuffix,
		"split":        strings.Split,
		"join":         strings.Join,
		"default":      defaultValue,
		"dict":         dict,
		"list":         list,
		"format":       fmt.Sprintf,
		"quote":        quote,
		"unquote":      unquote,
		"toYaml":       toYaml,
		"fromJson":     fromJson,
		"toJson":       toJson,
		"envVar":       envVar,
		"secretRef":    secretRef,
		"configMapRef": configMapRef,
		"appLabel":     appLabel,
		"versionLabel": versionLabel,
	}
}

// nindent indents text by a specified number of spaces and adds a newline at the beginning
func nindent(spaces int, text string) string {
	if text == "" {
		return ""
	}
	padding := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = padding + line
		}
	}
	return "\n" + strings.Join(lines, "\n")
}

// defaultValue returns the default value if the given value is empty
func defaultValue(value, defaultValue interface{}) interface{} {
	if value == nil || value == "" {
		return defaultValue
	}
	return value
}

// dict creates a dictionary from key-value pairs
func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.NewAppError(
			errors.ErrorTypeValues,
			errors.ErrCodeValuesInvalid,
			"invalid dict call, need even number of arguments",
			errors.ErrorSeverityError,
			false,
		).WithSuggestions(
			"Provide an even number of arguments",
			"Ensure arguments are in key-value pairs",
		)
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.NewAppError(
				errors.ErrorTypeValues,
				errors.ErrCodeValuesInvalid,
				"dict keys must be strings",
				errors.ErrorSeverityError,
				false,
			).WithField("key", fmt.Sprintf("%v", values[i])).WithSuggestions(
				"Ensure all keys are strings",
				"Convert non-string keys to strings",
			)
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

// list creates a list from values
func list(values ...interface{}) []interface{} {
	return values
}

// quote adds quotes around a string
func quote(value interface{}) string {
	return fmt.Sprintf("%q", value)
}

// unquote removes quotes from a string
func unquote(value interface{}) string {
	str := fmt.Sprintf("%v", value)
	return strings.Trim(str, "\"")
}

// toYaml converts a value to YAML string
func toYaml(value interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(value)
	if err != nil {
		return "", errors.NewAppError(
			errors.ErrorTypeValues,
			errors.ErrCodeValuesInvalid,
			"failed to marshal value to YAML",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("value", fmt.Sprintf("%v", value)).WithSuggestions(
			"Check if the value is serializable",
			"Ensure the value is in a valid format",
		)
	}
	return string(yamlBytes), nil
}

// fromJson parses a JSON string into a value
func fromJson(jsonStr string) (interface{}, error) {
	var result interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeValues,
			errors.ErrCodeValuesInvalid,
			"failed to unmarshal JSON string",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("json", jsonStr).WithSuggestions(
			"Check if the JSON string is valid",
			"Ensure the JSON string is properly formatted",
		)
	}
	return result, nil
}

// toJson converts a value to JSON string
func toJson(value interface{}) (string, error) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return "", errors.NewAppError(
			errors.ErrorTypeValues,
			errors.ErrCodeValuesInvalid,
			"failed to marshal value to JSON",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("value", fmt.Sprintf("%v", value)).WithSuggestions(
			"Check if the value is serializable",
			"Ensure the value is in a valid format",
		)
	}
	return string(jsonBytes), nil
}

// envVar creates an environment variable reference
func envVar(name string) string {
	return fmt.Sprintf("$(%s)", name)
}

// secretRef creates a secret reference
func secretRef(name, key string) string {
	return fmt.Sprintf("${%s.%s}", name, key)
}

// configMapRef creates a ConfigMap reference
func configMapRef(name, key string) string {
	return fmt.Sprintf("${%s.%s}", name, key)
}

// appLabel creates an app label
func appLabel(appName string) string {
	return appName
}

// versionLabel creates a version label
func versionLabel(version string) string {
	return version
}

// EnvironmentSpecificTemplate generates environment-specific values
func EnvironmentSpecificTemplate(baseValues map[string]interface{}, envValues map[string]map[string]interface{}, environment string) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy base values
	for k, v := range baseValues {
		result[k] = v
	}

	// Override with environment-specific values
	if envValues, ok := envValues[environment]; ok {
		for k, v := range envValues {
			result[k] = v
		}
	}

	return result
}

// GenerateHelmValues generates Helm values for an application
func GenerateHelmValues(appName, environment, imageRepository, imageTag string, envSecret string, uptraceConfig *UptraceConfig) map[string]interface{} {
	values := make(map[string]interface{})

	// Base application configuration
	appConfig := map[string]interface{}{
		"replicaCount": 1,
		"image": map[string]interface{}{
			"repository": imageRepository,
			"tag":        imageTag,
		},
		"podLabels": map[string]interface{}{
			"app":     appName,
			"version": imageTag,
		},
	}

	// Add environment secret if provided
	if envSecret != "" {
		appConfig["envSecret"] = envSecret
	}

	// Add image pull secrets
	appConfig["imagePullSecrets"] = []interface{}{
		map[string]interface{}{"name": "docker"},
	}

	// Add Uptrace configuration if enabled
	if uptraceConfig != nil && uptraceConfig.Enabled {
		envVars := []map[string]interface{}{
			{
				"name":  "ENVIRONMENT",
				"value": environment,
			},
			{
				"name":  "LOG_LEVEL",
				"value": "info",
			},
		}

		if uptraceConfig.ServiceName != "" {
			envVars = append(envVars, map[string]interface{}{
				"name":  "OTEL_SERVICE_NAME",
				"value": uptraceConfig.ServiceName,
			})
		}

		if uptraceConfig.Endpoint != "" {
			envVars = append(envVars, map[string]interface{}{
				"name":  "OTEL_EXPORTER_OTLP_ENDPOINT",
				"value": uptraceConfig.Endpoint,
			})
		}

		if uptraceConfig.Headers != nil && len(uptraceConfig.Headers) > 0 {
			for key, value := range uptraceConfig.Headers {
				envVars = append(envVars, map[string]interface{}{
					"name":  "OTEL_EXPORTER_OTLP_HEADERS",
					"value": fmt.Sprintf("%s=%s", key, value),
				})
			}
		}

		appConfig["env"] = envVars
	}

	// Add the app configuration to the values
	values[appName] = appConfig

	return values
}

// GenerateApplicationConfig generates an ApplicationConfig for the given parameters
func GenerateApplicationConfig(appName, project, repoURL, path, targetRevision, namespace, environment string, helmValues map[string]interface{}) ApplicationConfig {
	config := ApplicationConfig{
		Name:    fmt.Sprintf("%s-%s", appName, environment),
		Project: project,
		Source: ApplicationSource{
			RepoURL:        repoURL,
			Path:           path,
			TargetRevision: targetRevision,
			Helm: &HelmConfig{
				ValueFiles: []string{"values.yaml"},
			},
		},
		Destination: ApplicationDestination{
			Server:    "https://kubernetes.default.svc",
			Namespace: fmt.Sprintf("%s-%s", appName, environment),
		},
		SyncPolicy: &ApplicationSyncPolicy{
			Automated: &AutomatedSyncPolicy{
				Prune:    true,
				SelfHeal: true,
			},
			SyncOptions: []string{"CreateNamespace=true"},
		},
		Labels: map[string]string{
			"app.kubernetes.io/name":      appName,
			"app.kubernetes.io/component": "application",
			"kargo-bootstrap.io/app":      appName,
			"kargo-bootstrap.io/env":      environment,
		},
		Annotations: map[string]string{
			"kargo-bootstrap.io/managed-by": "kargo-bootstrap",
			"kargo-bootstrap.io/project":    project,
			"kargo-bootstrap.io/repo":       repoURL,
		},
	}

	// Add inline Helm values if provided
	if helmValues != nil {
		yamlStr, err := toYaml(helmValues)
		if err == nil {
			config.Source.Helm.Values = yamlStr
		} else {
			// Log the error but continue with the configuration
			// In a real implementation, you might want to handle this more gracefully
			_ = err
		}
	}

	return config
}
