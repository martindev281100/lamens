package prompt

import (
	"testing"
)

func TestNewPrompter(t *testing.T) {
	prompter, err := NewPrompter()
	if err != nil {
		t.Fatalf("Failed to create prompter: %v", err)
	}

	if prompter == nil {
		t.Fatal("Prompter is nil")
	}

	if prompter.config == nil {
		t.Fatal("Prompter config is nil")
	}
}

func TestNewPrompterWithConfig(t *testing.T) {
	config := &PromptConfig{
		ShowHelp:         false,
		ShowDefaults:     false,
		ConfirmDangerous: false,
		TimeoutSeconds:   30,
		PageSize:         5,
		ColorOutput:      false,
	}

	prompter, err := NewPrompterWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create prompter with config: %v", err)
	}

	if prompter == nil {
		t.Fatal("Prompter is nil")
	}

	if prompter.config.ShowHelp {
		t.Error("ShowHelp should be false")
	}

	if prompter.config.PageSize != 5 {
		t.Errorf("Expected PageSize 5, got %d", prompter.config.PageSize)
	}
}

func TestDefaultPromptConfig(t *testing.T) {
	config := DefaultPromptConfig()

	if config == nil {
		t.Fatal("Default config is nil")
	}

	if !config.ShowHelp {
		t.Error("ShowHelp should be true by default")
	}

	if !config.ShowDefaults {
		t.Error("ShowDefaults should be true by default")
	}

	if !config.ConfirmDangerous {
		t.Error("ConfirmDangerous should be true by default")
	}

	if config.TimeoutSeconds != 0 {
		t.Error("TimeoutSeconds should be 0 by default")
	}

	if config.PageSize != 10 {
		t.Errorf("Expected PageSize 10, got %d", config.PageSize)
	}

	if !config.ColorOutput {
		t.Error("ColorOutput should be true by default")
	}
}

func TestValidateAppName(t *testing.T) {
	tests := []struct {
		name    string
		appName string
		wantErr bool
	}{
		{
			name:    "valid name",
			appName: "my-app",
			wantErr: false,
		},
		{
			name:    "valid name with numbers",
			appName: "app123",
			wantErr: false,
		},
		{
			name:    "valid name with dots",
			appName: "my.app",
			wantErr: false,
		},
		{
			name:    "empty name",
			appName: "",
			wantErr: true,
		},
		{
			name:    "name with spaces",
			appName: "my app",
			wantErr: true,
		},
		{
			name:    "name with uppercase",
			appName: "MyApp",
			wantErr: true,
		},
		{
			name:    "name too long",
			appName: "a-very-long-application-name-that-exceeds-the-sixty-three-character-limit",
			wantErr: true,
		},
		{
			name:    "name starting with dot",
			appName: ".my-app",
			wantErr: true,
		},
		{
			name:    "name ending with dot",
			appName: "my-app.",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAppName(tt.appName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAppName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRepositoryURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL",
			url:     "https://github.com/example/helm-charts.git",
			wantErr: false,
		},
		{
			name:    "valid HTTP URL",
			url:     "http://github.com/example/helm-charts.git",
			wantErr: false,
		},
		{
			name:    "valid SSH URL",
			url:     "git@github.com:example/helm-charts.git",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			wantErr: false, // This gets normalized to HTTPS
		},
		{
			name:    "URL without scheme",
			url:     "github.com/example/helm-charts.git",
			wantErr: false, // Should be normalized to HTTPS
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepositoryURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepositoryURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateDefaultEnvironments(t *testing.T) {
	envs := CreateDefaultEnvironments()

	if len(envs) != 3 {
		t.Errorf("Expected 3 environments, got %d", len(envs))
	}

	// Check development environment
	devFound := false
	for _, env := range envs {
		if env.Name == "development" {
			devFound = true
			if env.Type != EnvironmentTypeDevelopment {
				t.Errorf("Expected development type, got %s", env.Type)
			}
			if !env.Default {
				t.Error("Development should be default")
			}
			break
		}
	}
	if !devFound {
		t.Error("Development environment not found")
	}

	// Check staging environment
	stagingFound := false
	for _, env := range envs {
		if env.Name == "staging" {
			stagingFound = true
			if env.Type != EnvironmentTypeStaging {
				t.Errorf("Expected staging type, got %s", env.Type)
			}
			if env.Default {
				t.Error("Staging should not be default")
			}
			break
		}
	}
	if !stagingFound {
		t.Error("Staging environment not found")
	}

	// Check production environment
	prodFound := false
	for _, env := range envs {
		if env.Name == "production" {
			prodFound = true
			if env.Type != EnvironmentTypeProduction {
				t.Errorf("Expected production type, got %s", env.Type)
			}
			if env.Default {
				t.Error("Production should not be default")
			}
			break
		}
	}
	if !prodFound {
		t.Error("Production environment not found")
	}
}

func TestFilterOptions(t *testing.T) {
	options := []string{
		"option1 - description",
		"option2 - another description",
		"test option",
		"sample option",
	}

	// Test with empty search term
	filtered := FilterOptions(options, "")
	if len(filtered) != 4 {
		t.Errorf("Expected 4 options, got %d", len(filtered))
	}

	// Test with search term
	filtered = FilterOptions(options, "option")
	if len(filtered) != 4 {
		t.Errorf("Expected 4 options, got %d", len(filtered))
	}

	// Test with specific search term
	filtered = FilterOptions(options, "test")
	if len(filtered) != 1 {
		t.Errorf("Expected 1 option, got %d", len(filtered))
	}
	if filtered[0] != "test option" {
		t.Errorf("Expected 'test option', got '%s'", filtered[0])
	}
}

func TestPaginateOptions(t *testing.T) {
	options := []string{
		"option1",
		"option2",
		"option3",
		"option4",
		"option5",
	}

	// Test with page size 2
	pages := PaginateOptions(options, 2)
	if len(pages) != 3 {
		t.Errorf("Expected 3 pages, got %d", len(pages))
	}
	if len(pages[0]) != 2 {
		t.Errorf("Expected 2 items in first page, got %d", len(pages[0]))
	}
	if len(pages[2]) != 1 {
		t.Errorf("Expected 1 item in last page, got %d", len(pages[2]))
	}

	// Test with page size larger than options
	pages = PaginateOptions(options, 10)
	if len(pages) != 1 {
		t.Errorf("Expected 1 page, got %d", len(pages))
	}
	if len(pages[0]) != 5 {
		t.Errorf("Expected 5 items in page, got %d", len(pages[0]))
	}

	// Test with zero page size (should default to 10)
	pages = PaginateOptions(options, 0)
	if len(pages) != 1 {
		t.Errorf("Expected 1 page, got %d", len(pages))
	}
}
