package config

import (
	"os"
	"strings"
	"testing"
)

func TestNewEnvironmentManager(t *testing.T) {
	envManager := NewEnvironmentManager()
	if envManager == nil {
		t.Fatal("Expected non-nil environment manager")
	}
	if len(envManager.ListEnvironments()) != 0 {
		t.Fatalf("Expected empty environment list, got %d", len(envManager.ListEnvironments()))
	}
}

func TestLoadDefaultEnvironments(t *testing.T) {
	envManager := NewEnvironmentManager()
	envManager.LoadDefaultEnvironments()

	envs := envManager.ListEnvironments()
	if len(envs) != 3 {
		t.Fatalf("Expected 3 default environments, got %d", len(envs))
	}

	// Check that default environments exist
	envNames := envManager.ListEnvironmentNames()
	foundDev := false
	foundStaging := false
	foundProd := false

	for _, name := range envNames {
		if name == "development" {
			foundDev = true
		}
		if name == "staging" {
			foundStaging = true
		}
		if name == "production" {
			foundProd = true
		}
	}

	if !foundDev {
		t.Error("Expected to find 'development' environment")
	}
	if !foundStaging {
		t.Error("Expected to find 'staging' environment")
	}
	if !foundProd {
		t.Error("Expected to find 'production' environment")
	}
}

func TestAddEnvironment(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Add a valid environment
	env := &EnvironmentConfig{
		Name:        "test",
		Type:        "testing",
		Description: "Test environment",
		Namespace:   "test",
	}

	err := envManager.AddEnvironment(env)
	if err != nil {
		t.Fatalf("Expected no error adding environment, got %v", err)
	}

	// Verify the environment was added
	retrieved, err := envManager.GetEnvironment("test")
	if err != nil {
		t.Fatalf("Expected no error getting environment, got %v", err)
	}
	if retrieved.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", retrieved.Name)
	}
	if retrieved.Type != "testing" {
		t.Errorf("Expected type 'testing', got '%s'", retrieved.Type)
	}
	if retrieved.Description != "Test environment" {
		t.Errorf("Expected description 'Test environment', got '%s'", retrieved.Description)
	}
	if retrieved.Namespace != "test" {
		t.Errorf("Expected namespace 'test', got '%s'", retrieved.Namespace)
	}
}

func TestAddDuplicateEnvironment(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Add an environment
	env := &EnvironmentConfig{
		Name:        "test",
		Type:        "testing",
		Description: "Test environment",
		Namespace:   "test",
	}

	err := envManager.AddEnvironment(env)
	if err != nil {
		t.Fatalf("Expected no error adding environment, got %v", err)
	}

	// Try to add the same environment again
	err = envManager.AddEnvironment(env)
	if err == nil {
		t.Fatal("Expected error when adding duplicate environment")
	}
	if !contains(err.Error(), "already exists") {
		t.Errorf("Expected error to contain 'already exists', got '%s'", err.Error())
	}
}

func TestGetNonExistentEnvironment(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Try to get a non-existent environment
	_, err := envManager.GetEnvironment("nonexistent")
	if err == nil {
		t.Fatal("Expected error when getting non-existent environment")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Expected error to contain 'not found', got '%s'", err.Error())
	}
}

func TestUpdateEnvironment(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Add an environment
	env := &EnvironmentConfig{
		Name:        "test",
		Type:        "testing",
		Description: "Test environment",
		Namespace:   "test",
	}

	err := envManager.AddEnvironment(env)
	if err != nil {
		t.Fatalf("Expected no error adding environment, got %v", err)
	}

	// Update the environment
	updatedEnv := &EnvironmentConfig{
		Name:        "test",
		Type:        "staging",
		Description: "Updated test environment",
		Namespace:   "test-updated",
	}

	err = envManager.UpdateEnvironment("test", updatedEnv)
	if err != nil {
		t.Fatalf("Expected no error updating environment, got %v", err)
	}

	// Verify the environment was updated
	retrieved, err := envManager.GetEnvironment("test")
	if err != nil {
		t.Fatalf("Expected no error getting environment, got %v", err)
	}
	if retrieved.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", retrieved.Name)
	}
	if retrieved.Type != "staging" {
		t.Errorf("Expected type 'staging', got '%s'", retrieved.Type)
	}
	if retrieved.Description != "Updated test environment" {
		t.Errorf("Expected description 'Updated test environment', got '%s'", retrieved.Description)
	}
	if retrieved.Namespace != "test-updated" {
		t.Errorf("Expected namespace 'test-updated', got '%s'", retrieved.Namespace)
	}
}

func TestUpdateNonExistentEnvironment(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Try to update a non-existent environment
	env := &EnvironmentConfig{
		Name:        "nonexistent",
		Type:        "testing",
		Description: "Non-existent environment",
		Namespace:   "nonexistent",
	}

	err := envManager.UpdateEnvironment("nonexistent", env)
	if err == nil {
		t.Fatal("Expected error when updating non-existent environment")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Expected error to contain 'not found', got '%s'", err.Error())
	}
}

func TestRemoveEnvironment(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Add an environment
	env := &EnvironmentConfig{
		Name:        "test",
		Type:        "testing",
		Description: "Test environment",
		Namespace:   "test",
	}

	err := envManager.AddEnvironment(env)
	if err != nil {
		t.Fatalf("Expected no error adding environment, got %v", err)
	}

	// Remove the environment
	err = envManager.RemoveEnvironment("test")
	if err != nil {
		t.Fatalf("Expected no error removing environment, got %v", err)
	}

	// Verify the environment was removed
	_, err = envManager.GetEnvironment("test")
	if err == nil {
		t.Fatal("Expected error when getting removed environment")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Expected error to contain 'not found', got '%s'", err.Error())
	}
}

func TestRemoveNonExistentEnvironment(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Try to remove a non-existent environment
	err := envManager.RemoveEnvironment("nonexistent")
	if err == nil {
		t.Fatal("Expected error when removing non-existent environment")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Expected error to contain 'not found', got '%s'", err.Error())
	}
}

func TestValidateEnvironments(t *testing.T) {
	envManager := NewEnvironmentManager()
	envManager.LoadDefaultEnvironments()

	// Validate existing environments
	err := envManager.ValidateEnvironments([]string{"development", "staging"})
	if err != nil {
		t.Fatalf("Expected no error validating existing environments, got %v", err)
	}

	// Validate with a non-existent environment
	err = envManager.ValidateEnvironments([]string{"development", "nonexistent"})
	if err == nil {
		t.Fatal("Expected error when validating non-existent environment")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("Expected error to contain 'not found', got '%s'", err.Error())
	}

	// Validate with empty list
	err = envManager.ValidateEnvironments([]string{})
	if err == nil {
		t.Fatal("Expected error when validating empty environment list")
	}
	if !contains(err.Error(), "at least one environment") {
		t.Errorf("Expected error to contain 'at least one environment', got '%s'", err.Error())
	}
}

func TestLoadFromFile(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Create a test config file
	configData := `
environments:
  - name: "test-env"
    type: "testing"
    description: "Test environment from file"
    namespace: "test-env"
    replicaCount: 2
    autoSync: true
    prune: true
    selfHeal: true
`

	// Write to a temporary file
	tmpFile, err := os.CreateTemp("", "test-env-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configData); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Load from file
	err = envManager.LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Expected no error loading from file, got %v", err)
	}

	// Verify the environment was loaded
	env, err := envManager.GetEnvironment("test-env")
	if err != nil {
		t.Fatalf("Expected no error getting environment, got %v", err)
	}
	if env.Name != "test-env" {
		t.Errorf("Expected name 'test-env', got '%s'", env.Name)
	}
	if env.Type != "testing" {
		t.Errorf("Expected type 'testing', got '%s'", env.Type)
	}
	if env.Description != "Test environment from file" {
		t.Errorf("Expected description 'Test environment from file', got '%s'", env.Description)
	}
	if env.Namespace != "test-env" {
		t.Errorf("Expected namespace 'test-env', got '%s'", env.Namespace)
	}
	if env.ReplicaCount != 2 {
		t.Errorf("Expected replica count 2, got %d", env.ReplicaCount)
	}
	if !env.AutoSync {
		t.Error("Expected autoSync to be true")
	}
	if !env.Prune {
		t.Error("Expected prune to be true")
	}
	if !env.SelfHeal {
		t.Error("Expected selfHeal to be true")
	}
}

func TestSaveToFile(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Add an environment
	env := &EnvironmentConfig{
		Name:         "test-env",
		Type:         "testing",
		Description:  "Test environment for saving",
		Namespace:    "test-env",
		ReplicaCount: 2,
		AutoSync:     true,
		Prune:        true,
		SelfHeal:     true,
	}

	err := envManager.AddEnvironment(env)
	if err != nil {
		t.Fatalf("Expected no error adding environment, got %v", err)
	}

	// Save to a temporary file
	tmpFile, err := os.CreateTemp("", "test-env-save-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	err = envManager.SaveToFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Expected no error saving to file, got %v", err)
	}

	// Verify the file exists and contains the expected content
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	dataStr := string(data)
	if !contains(dataStr, "test-env") {
		t.Error("Expected file to contain 'test-env'")
	}
	if !contains(dataStr, "testing") {
		t.Error("Expected file to contain 'testing'")
	}
	if !contains(dataStr, "Test environment for saving") {
		t.Error("Expected file to contain 'Test environment for saving'")
	}
}

func TestMergeValues(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Add an environment with custom values
	env := &EnvironmentConfig{
		Name:        "test",
		Type:        "testing",
		Description: "Test environment",
		Namespace:   "test",
		Values: map[string]interface{}{
			"debug":    false,
			"logLevel": "info",
			"custom":   "value",
		},
	}

	err := envManager.AddEnvironment(env)
	if err != nil {
		t.Fatalf("Expected no error adding environment, got %v", err)
	}

	// Define base values
	baseValues := map[string]interface{}{
		"debug":  true,
		"global": "base",
	}

	// Merge values
	merged, err := envManager.MergeValues(baseValues, "test")
	if err != nil {
		t.Fatalf("Expected no error merging values, got %v", err)
	}

	// Verify the merged values
	if merged["debug"] != false {
		t.Errorf("Expected debug to be false (environment override), got %v", merged["debug"])
	}
	if merged["global"] != "base" {
		t.Errorf("Expected global to be 'base' (preserved), got %v", merged["global"])
	}
	if merged["logLevel"] != "info" {
		t.Errorf("Expected logLevel to be 'info' (added), got %v", merged["logLevel"])
	}
	if merged["custom"] != "value" {
		t.Errorf("Expected custom to be 'value' (added), got %v", merged["custom"])
	}
}

func TestValidateEnvironmentSetup(t *testing.T) {
	envManager := NewEnvironmentManager()
	envManager.LoadDefaultEnvironments()

	// Validate valid setup
	result := envManager.ValidateEnvironmentSetup([]string{"development", "staging"})
	if !result.Valid {
		t.Error("Expected valid setup to be marked as valid")
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors for valid setup, got %d", len(result.Errors))
	}

	// Validate with non-existent environment
	result = envManager.ValidateEnvironmentSetup([]string{"development", "nonexistent"})
	if result.Valid {
		t.Error("Expected invalid setup to be marked as invalid")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for invalid setup")
	}
	if !contains(result.Errors[0], "not found") {
		t.Errorf("Expected error to contain 'not found', got '%s'", result.Errors[0])
	}

	// Validate with conflicting namespaces (add a custom env with same namespace)
	customEnv := &EnvironmentConfig{
		Name:        "custom",
		Type:        "custom",
		Description: "Custom environment",
		Namespace:   "dev", // Same as development
	}

	err := envManager.AddEnvironment(customEnv)
	if err != nil {
		t.Fatalf("Expected no error adding environment, got %v", err)
	}

	result = envManager.ValidateEnvironmentSetup([]string{"development", "custom"})
	if !result.Valid {
		t.Error("Expected setup with conflicting namespaces to be marked as valid (with warnings)")
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for setup with conflicting namespaces")
	}
	if !contains(result.Warnings[0], "same namespace") {
		t.Errorf("Expected warning to contain 'same namespace', got '%s'", result.Warnings[0])
	}

	// Validate production with auto-sync (should generate warning)
	result = envManager.ValidateEnvironmentSetup([]string{"production"})
	if !result.Valid {
		t.Error("Expected production setup to be marked as valid")
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for production with auto-sync")
	}
	if !contains(result.Warnings[0], "auto-sync enabled") {
		t.Errorf("Expected warning to contain 'auto-sync enabled', got '%s'", result.Warnings[0])
	}
}

func TestGetEnvironmentSummary(t *testing.T) {
	envManager := NewEnvironmentManager()

	// Empty manager should return a specific message
	summary := envManager.GetEnvironmentSummary()
	if !contains(summary, "No environments configured") {
		t.Errorf("Expected summary to contain 'No environments configured', got '%s'", summary)
	}

	// Load defaults and check summary
	envManager.LoadDefaultEnvironments()
	summary = envManager.GetEnvironmentSummary()
	if !contains(summary, "Configured Environments (3)") {
		t.Errorf("Expected summary to contain 'Configured Environments (3)', got '%s'", summary)
	}
	if !contains(summary, "Name: development") {
		t.Errorf("Expected summary to contain 'Name: development', got '%s'", summary)
	}
	if !contains(summary, "Name: staging") {
		t.Errorf("Expected summary to contain 'Name: staging', got '%s'", summary)
	}
	if !contains(summary, "Name: production") {
		t.Errorf("Expected summary to contain 'Name: production', got '%s'", summary)
	}
	if !contains(summary, "Type: development") {
		t.Errorf("Expected summary to contain 'Type: development', got '%s'", summary)
	}
	if !contains(summary, "Type: staging") {
		t.Errorf("Expected summary to contain 'Type: staging', got '%s'", summary)
	}
	if !contains(summary, "Type: production") {
		t.Errorf("Expected summary to contain 'Type: production', got '%s'", summary)
	}
}

func TestValidateEnvironmentConfig(t *testing.T) {
	// Valid config
	validConfig := &EnvironmentConfig{
		Name:          "test",
		Type:          "testing",
		Description:   "Test environment",
		Namespace:     "test",
		ReplicaCount:  1,
		CPURequest:    "100m",
		CPULimit:      "500m",
		MemoryRequest: "128Mi",
		MemoryLimit:   "512Mi",
	}

	err := ValidateEnvironmentConfig(validConfig)
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}

	// Nil config
	err = ValidateEnvironmentConfig(nil)
	if err == nil {
		t.Fatal("Expected error for nil config")
	}
	if !contains(err.Error(), "cannot be nil") {
		t.Errorf("Expected error to contain 'cannot be nil', got '%s'", err.Error())
	}

	// Empty name
	invalidConfig := &EnvironmentConfig{
		Name:        "",
		Type:        "testing",
		Description: "Test environment",
		Namespace:   "test",
	}

	err = ValidateEnvironmentConfig(invalidConfig)
	if err == nil {
		t.Fatal("Expected error for empty name")
	}
	if !contains(err.Error(), "name cannot be empty") {
		t.Errorf("Expected error to contain 'name cannot be empty', got '%s'", err.Error())
	}

	// Empty namespace
	invalidConfig = &EnvironmentConfig{
		Name:        "test",
		Type:        "testing",
		Description: "Test environment",
		Namespace:   "",
	}

	err = ValidateEnvironmentConfig(invalidConfig)
	if err == nil {
		t.Fatal("Expected error for empty namespace")
	}
	if !contains(err.Error(), "namespace cannot be empty") {
		t.Errorf("Expected error to contain 'namespace cannot be empty', got '%s'", err.Error())
	}

	// Negative replica count
	invalidConfig = &EnvironmentConfig{
		Name:         "test",
		Type:         "testing",
		Description:  "Test environment",
		Namespace:    "test",
		ReplicaCount: -1,
	}

	err = ValidateEnvironmentConfig(invalidConfig)
	if err == nil {
		t.Fatal("Expected error for negative replica count")
	}
	if !contains(err.Error(), "replica count cannot be negative") {
		t.Errorf("Expected error to contain 'replica count cannot be negative', got '%s'", err.Error())
	}

	// Invalid CPU format
	invalidConfig = &EnvironmentConfig{
		Name:        "test",
		Type:        "testing",
		Description: "Test environment",
		Namespace:   "test",
		CPURequest:  "invalid",
	}

	err = ValidateEnvironmentConfig(invalidConfig)
	if err == nil {
		t.Fatal("Expected error for invalid CPU format")
	}
	if !contains(err.Error(), "invalid CPU request format") {
		t.Errorf("Expected error to contain 'invalid CPU request format', got '%s'", err.Error())
	}

	// Invalid memory format
	invalidConfig = &EnvironmentConfig{
		Name:          "test",
		Type:          "testing",
		Description:   "Test environment",
		Namespace:     "test",
		MemoryRequest: "invalid",
	}

	err = ValidateEnvironmentConfig(invalidConfig)
	if err == nil {
		t.Fatal("Expected error for invalid memory format")
	}
	if !contains(err.Error(), "invalid memory request format") {
		t.Errorf("Expected error to contain 'invalid memory request format', got '%s'", err.Error())
	}
}

func TestFormatFunctions(t *testing.T) {
	// Test isValidCPUFormat
	if !isValidCPUFormat("100m") {
		t.Error("Expected '100m' to be valid CPU format")
	}
	if !isValidCPUFormat("1") {
		t.Error("Expected '1' to be valid CPU format")
	}
	if isValidCPUFormat("invalid") {
		t.Error("Expected 'invalid' to be invalid CPU format")
	}
	if isValidCPUFormat("") {
		t.Error("Expected empty string to be invalid CPU format")
	}

	// Test isValidMemoryFormat
	if !isValidMemoryFormat("128Mi") {
		t.Error("Expected '128Mi' to be valid memory format")
	}
	if !isValidMemoryFormat("1Gi") {
		t.Error("Expected '1Gi' to be valid memory format")
	}
	if !isValidMemoryFormat("512") {
		t.Error("Expected '512' to be valid memory format")
	}
	if isValidMemoryFormat("invalid") {
		t.Error("Expected 'invalid' to be invalid memory format")
	}
	if isValidMemoryFormat("") {
		t.Error("Expected empty string to be invalid memory format")
	}

	// Test isPlainNumber
	if !isPlainNumber("123") {
		t.Error("Expected '123' to be plain number")
	}
	if isPlainNumber("123m") {
		t.Error("Expected '123m' to not be plain number")
	}
	if isPlainNumber("abc") {
		t.Error("Expected 'abc' to not be plain number")
	}
	if isPlainNumber("") {
		t.Error("Expected empty string to not be plain number")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
