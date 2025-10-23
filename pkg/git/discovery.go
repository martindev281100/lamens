package git

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/your-org/kargo-bootstrap/pkg/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

// Note: ChartDiscoveryError is replaced by the centralized error system in pkg/errors

// ChartInfo contains information about a Helm chart
type ChartInfo struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	AppVersion  string   `yaml:"appVersion"`
	Description string   `yaml:"description"`
	Home        string   `yaml:"home"`
	Sources     []string `yaml:"sources"`
	Keywords    []string `yaml:"keywords"`
	Maintainers []struct {
		Name  string `yaml:"name"`
		Email string `yaml:"email"`
		URL   string `yaml:"url"`
	} `yaml:"maintainers"`
	Path string `yaml:"-"` // Not in Chart.yaml, added for convenience
}

// ChartFilter contains criteria for filtering charts
type ChartFilter struct {
	Name     string
	Version  string
	Keywords []string
}

// FindChartPaths scans a repository directory and returns paths containing Chart.yaml
func FindChartPaths(repoPath string, maxPaths int) ([]string, error) {
	klog.V(2).Infof("Scanning repository for Helm charts: %s", repoPath)

	// Check if repository path exists and is accessible
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			fmt.Sprintf("repository path does not exist: %s", repoPath),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("repository path", repoPath).WithSuggestions(
			"Check if the repository path is correct",
			"Verify the repository exists",
			"Ensure the repository is accessible",
		)
	}

	var chartPaths []string
	var scanErrors []error

	// Walk through the directory tree
	err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Check for permission errors
			if os.IsPermission(err) {
				scanErrors = append(scanErrors,
					errors.NewAppError(
						errors.ErrorTypePermission,
						errors.ErrCodePermissionDenied,
						fmt.Sprintf("permission denied accessing path: %s", path),
						errors.ErrorSeverityWarning,
						false,
					).WithCause(err).WithField("path", path).WithSuggestions(
						"Check file permissions",
						"Ensure the repository is accessible",
					))
				// Skip this directory but continue scanning
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Log other errors but continue scanning
			klog.V(2).Infof("Error accessing path %s: %v", path, err)
			return nil
		}

		// Skip directories that are likely not charts
		if d.IsDir() {
			// Skip hidden directories and common non-chart directories
			dirName := filepath.Base(path)
			if strings.HasPrefix(dirName, ".") ||
				dirName == "node_modules" ||
				dirName == "vendor" ||
				dirName == ".git" ||
				dirName == "docs" ||
				dirName == "examples" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if this is a Chart.yaml file
		if d.Name() == "Chart.yaml" {
			chartDir := filepath.Dir(path)
			// Convert to relative path from repo root
			relPath, err := filepath.Rel(repoPath, chartDir)
			if err != nil {
				klog.V(2).Infof("Error getting relative path for %s: %v", chartDir, err)
				return nil
			}

			chartPaths = append(chartPaths, relPath)

			// Check if we've reached the maximum number of paths
			if maxPaths > 0 && len(chartPaths) >= maxPaths {
				klog.V(2).Infof("Reached maximum chart limit (%d), stopping scan", maxPaths)
				return fmt.Errorf("found %d charts, stopping scan", maxPaths)
			}
		}

		return nil
	})

	// Check if we stopped early due to maxPaths limit
	if err != nil && maxPaths > 0 && strings.Contains(err.Error(), fmt.Sprintf("found %d charts, stopping scan", maxPaths)) {
		klog.V(2).Infof("Reached maximum chart limit (%d), stopping scan", maxPaths)
		err = nil
	}

	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"error scanning repository",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository is accessible",
			"Verify the repository path is correct",
		)
	}

	// If we found charts but had some permission errors, log them but don't fail
	if len(chartPaths) > 0 && len(scanErrors) > 0 {
		klog.V(2).Infof("Completed scan with %d error(s)", len(scanErrors))
		for _, scanErr := range scanErrors {
			klog.V(2).Infof("Scan error: %v", scanErr)
		}
	}

	// If we found no charts and had errors, return the errors
	if len(chartPaths) == 0 && len(scanErrors) > 0 {
		return nil, scanErrors[0] // Return the first error
	}

	// Sort paths alphabetically for consistent display
	SortChartPaths(chartPaths)

	klog.V(2).Infof("Found %d chart(s) in repository", len(chartPaths))
	return chartPaths, nil
}

// GetChartInfo reads and parses a Chart.yaml file to extract chart metadata
func GetChartInfo(repoPath, chartPath string) (*ChartInfo, error) {
	chartYamlPath := filepath.Join(repoPath, chartPath, "Chart.yaml")

	// Check if the file exists
	if _, err := os.Stat(chartYamlPath); os.IsNotExist(err) {
		return nil, errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeChartNotFound,
			fmt.Sprintf("Chart.yaml not found at %s", chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Check if the chart directory is correct",
			"Verify Chart.yaml exists in the chart directory",
		)
	}

	// Check for permission errors
	if _, err := os.Stat(chartYamlPath); os.IsPermission(err) {
		return nil, errors.NewAppError(
			errors.ErrorTypePermission,
			errors.ErrCodePermissionDenied,
			fmt.Sprintf("permission denied reading Chart.yaml at %s", chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Check file permissions",
			"Ensure the chart is accessible",
		)
	}

	// Read the file
	data, err := os.ReadFile(chartYamlPath)
	if err != nil {
		if os.IsPermission(err) {
			return nil, errors.NewAppError(
				errors.ErrorTypePermission,
				errors.ErrCodePermissionDenied,
				fmt.Sprintf("permission denied reading Chart.yaml at %s", chartYamlPath),
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
				"Check file permissions",
				"Ensure the chart is accessible",
			)
		}
		return nil, errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			fmt.Sprintf("failed to read Chart.yaml at %s", chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Check if the file exists",
			"Verify the file is readable",
		)
	}

	// Check if the file is empty
	if len(data) == 0 {
		return nil, errors.NewAppError(
			errors.ErrorTypeValues,
			errors.ErrCodeValuesInvalid,
			fmt.Sprintf("Chart.yaml is empty at %s", chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Add content to Chart.yaml",
			"Ensure the chart is properly configured",
		)
	}

	// Parse the YAML
	var chartInfo ChartInfo
	if err := yaml.Unmarshal(data, &chartInfo); err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeValues,
			errors.ErrCodeValuesInvalid,
			fmt.Sprintf("failed to parse Chart.yaml at %s: %v", chartYamlPath, err),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Check if Chart.yaml is valid YAML",
			"Verify the chart configuration is correct",
		)
	}

	// Set the path for convenience
	chartInfo.Path = chartPath

	// Validate required fields
	if chartInfo.Name == "" {
		return nil, errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeChartNotFound,
			fmt.Sprintf("chart name is required in Chart.yaml at %s", chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Add a name field to Chart.yaml",
			"Ensure the chart name is valid",
		)
	}
	if chartInfo.Version == "" {
		return nil, errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeChartNotFound,
			fmt.Sprintf("chart version is required in Chart.yaml at %s", chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Add a version field to Chart.yaml",
			"Ensure the chart version is valid",
		)
	}

	// Validate chart name format (basic validation)
	if !isValidChartName(chartInfo.Name) {
		return nil, errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeInvalidChart,
			fmt.Sprintf("invalid chart name '%s' in Chart.yaml at %s", chartInfo.Name, chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithField("chart name", chartInfo.Name).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Use a valid chart name",
			"Ensure the chart name follows Helm naming conventions",
		)
	}

	// Validate chart version format (basic validation)
	if !isValidVersion(chartInfo.Version) {
		return nil, errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeInvalidChart,
			fmt.Sprintf("invalid chart version '%s' in Chart.yaml at %s", chartInfo.Version, chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithField("chart version", chartInfo.Version).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Use a valid chart version",
			"Ensure the chart version follows semantic versioning",
		)
	}

	return &chartInfo, nil
}

// ValidateChart validates that a chart directory has the required structure
func ValidateChart(repoPath, chartPath string) error {
	chartDir := filepath.Join(repoPath, chartPath)

	// Check if the directory exists
	if _, err := os.Stat(chartDir); os.IsNotExist(err) {
		return errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeChartNotFound,
			fmt.Sprintf("chart directory does not exist: %s", chartDir),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Check if the chart directory is correct",
			"Verify the chart exists",
		)
	}

	// Check for permission errors
	if _, err := os.Stat(chartDir); os.IsPermission(err) {
		return errors.NewAppError(
			errors.ErrorTypePermission,
			errors.ErrCodePermissionDenied,
			fmt.Sprintf("permission denied accessing chart directory: %s", chartDir),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Check file permissions",
			"Ensure the chart is accessible",
		)
	}

	// Check for Chart.yaml
	chartYamlPath := filepath.Join(chartDir, "Chart.yaml")
	if _, err := os.Stat(chartYamlPath); os.IsNotExist(err) {
		return errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeChartNotFound,
			fmt.Sprintf("Chart.yaml not found in chart directory: %s", chartPath),
			errors.ErrorSeverityError,
			false,
		).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Add Chart.yaml to the chart directory",
			"Ensure the chart is properly configured",
		)
	}

	// Try to parse the Chart.yaml to ensure it's valid
	_, err := GetChartInfo(repoPath, chartPath)
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeInvalidChart,
			fmt.Sprintf("invalid Chart.yaml in chart %s: %v", chartPath, err),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
			"Check if Chart.yaml is valid",
			"Verify the chart configuration is correct",
		)
	}

	// Check for templates directory (optional but recommended)
	templatesDir := filepath.Join(chartDir, "templates")
	if stat, err := os.Stat(templatesDir); err != nil {
		if os.IsNotExist(err) {
			klog.V(2).Infof("Warning: templates directory not found in chart: %s", chartPath)
		} else if os.IsPermission(err) {
			return errors.NewAppError(
				errors.ErrorTypePermission,
				errors.ErrCodePermissionDenied,
				fmt.Sprintf("permission denied accessing templates directory in chart: %s", chartPath),
				errors.ErrorSeverityWarning,
				false,
			).WithCause(err).WithField("chart path", chartPath).WithResource(chartPath).WithSuggestions(
				"Check file permissions",
				"Ensure the templates directory is accessible",
			)
		} else {
			klog.V(2).Infof("Warning: error accessing templates directory in chart %s: %v", chartPath, err)
		}
	} else if !stat.IsDir() {
		klog.V(2).Infof("Warning: templates path exists but is not a directory in chart: %s", chartPath)
	}

	return nil
}

// ListChartsInRepo combines cloning/fetching with chart discovery
func ListChartsInRepo(opts *CloneOptions) ([]*ChartInfo, error) {
	// Clone the repository
	repoInfo, err := CloneRepository(opts)
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"failed to clone repository",
			errors.ErrorSeverityError,
			true, // Network issues might be retryable
		).WithCause(err).WithSuggestions(
			"Check if the repository URL is correct",
			"Verify network connectivity to the repository",
			"Check authentication credentials",
		)
	}
	defer CleanupTempDir(repoInfo.Path)

	// Find chart paths
	chartPaths, err := FindChartPaths(repoInfo.Path, 30) // Limit to ~30 paths as mentioned in PRD
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			"failed to find chart paths",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the repository contains any charts",
			"Verify the repository structure is correct",
		)
	}

	// If no charts found, return a specific error
	if len(chartPaths) == 0 {
		return nil, errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeChartNotFound,
			"no Helm charts found in repository",
			errors.ErrorSeverityError,
			false,
		).WithSuggestions(
			"Add charts to the repository",
			"Check if charts are in the correct location",
		)
	}

	// Get chart info for each path
	var charts []*ChartInfo
	var chartErrors []error

	for _, chartPath := range chartPaths {
		chartInfo, err := GetChartInfo(repoInfo.Path, chartPath)
		if err != nil {
			// Log the error but continue processing other charts
			klog.V(2).Infof("Error getting chart info for %s: %v", chartPath, err)
			chartErrors = append(chartErrors, err)
			continue
		}
		charts = append(charts, chartInfo)
	}

	// If we couldn't process any charts successfully, return an error
	if len(charts) == 0 && len(chartErrors) > 0 {
		return nil, errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeInvalidChart,
			fmt.Sprintf("failed to process any charts (%d errors encountered)", len(chartErrors)),
			errors.ErrorSeverityError,
			false,
		).WithCause(chartErrors[0]).WithSuggestions(
			"Check if the charts are valid",
			"Verify the chart configurations are correct",
		)
	}

	// Log warnings for charts that couldn't be processed
	if len(chartErrors) > 0 {
		klog.V(2).Infof("Processed %d charts with %d error(s)", len(charts), len(chartErrors))
		for _, chartErr := range chartErrors {
			klog.V(2).Infof("Chart processing error: %v", chartErr)
		}
	}

	return charts, nil
}

// IsChartDirectory checks if a directory contains a valid Chart.yaml
func IsChartDirectory(repoPath, chartPath string) bool {
	chartYamlPath := filepath.Join(repoPath, chartPath, "Chart.yaml")

	// Check if the file exists
	if _, err := os.Stat(chartYamlPath); os.IsNotExist(err) {
		return false
	}

	// Try to parse the Chart.yaml to ensure it's valid
	_, err := GetChartInfo(repoPath, chartPath)
	return err == nil
}

// ParseChartYAML parses Chart.yaml and extracts metadata
func ParseChartYAML(chartYamlPath string) (*ChartInfo, error) {
	// Check if the file exists
	if _, err := os.Stat(chartYamlPath); os.IsNotExist(err) {
		return nil, errors.NewAppError(
			errors.ErrorTypeChart,
			errors.ErrCodeChartNotFound,
			fmt.Sprintf("Chart.yaml not found at %s", chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartYamlPath).WithSuggestions(
			"Check if the chart directory is correct",
			"Verify Chart.yaml exists in the chart directory",
		)
	}

	// Check for permission errors
	if _, err := os.Stat(chartYamlPath); os.IsPermission(err) {
		return nil, errors.NewAppError(
			errors.ErrorTypePermission,
			errors.ErrCodePermissionDenied,
			fmt.Sprintf("permission denied reading Chart.yaml at %s", chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartYamlPath).WithSuggestions(
			"Check file permissions",
			"Ensure the chart is accessible",
		)
	}

	// Read the file
	data, err := os.ReadFile(chartYamlPath)
	if err != nil {
		if os.IsPermission(err) {
			return nil, errors.NewAppError(
				errors.ErrorTypePermission,
				errors.ErrCodePermissionDenied,
				fmt.Sprintf("permission denied reading Chart.yaml at %s", chartYamlPath),
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithField("chart path", chartYamlPath).WithSuggestions(
				"Check file permissions",
				"Ensure the chart is accessible",
			)
		}
		return nil, errors.NewAppError(
			errors.ErrorTypeRepository,
			errors.ErrCodeRepoNotFound,
			fmt.Sprintf("failed to read Chart.yaml at %s", chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartYamlPath).WithSuggestions(
			"Check if the file exists",
			"Verify the file is readable",
		)
	}

	// Check if the file is empty
	if len(data) == 0 {
		return nil, errors.NewAppError(
			errors.ErrorTypeValues,
			errors.ErrCodeValuesInvalid,
			fmt.Sprintf("Chart.yaml is empty at %s", chartYamlPath),
			errors.ErrorSeverityError,
			false,
		).WithField("chart path", chartYamlPath).WithSuggestions(
			"Add content to Chart.yaml",
			"Ensure the chart is properly configured",
		)
	}

	// Parse the YAML
	var chartInfo ChartInfo
	if err := yaml.Unmarshal(data, &chartInfo); err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeValues,
			errors.ErrCodeValuesInvalid,
			fmt.Sprintf("failed to parse Chart.yaml at %s: %v", chartYamlPath, err),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("chart path", chartYamlPath).WithSuggestions(
			"Check if Chart.yaml is valid YAML",
			"Verify the chart configuration is correct",
		)
	}

	return &chartInfo, nil
}

// FilterCharts filters charts based on criteria
func FilterCharts(charts []*ChartInfo, filter *ChartFilter) []*ChartInfo {
	if filter == nil {
		return charts
	}

	var filtered []*ChartInfo

	for _, chart := range charts {
		// Filter by name
		if filter.Name != "" && !strings.EqualFold(chart.Name, filter.Name) {
			continue
		}

		// Filter by version
		if filter.Version != "" && chart.Version != filter.Version {
			continue
		}

		// Filter by keywords
		if len(filter.Keywords) > 0 {
			hasKeyword := false
			for _, chartKeyword := range chart.Keywords {
				for _, filterKeyword := range filter.Keywords {
					if strings.EqualFold(chartKeyword, filterKeyword) {
						hasKeyword = true
						break
					}
				}
				if hasKeyword {
					break
				}
			}
			if !hasKeyword {
				continue
			}
		}

		filtered = append(filtered, chart)
	}

	return filtered
}

// SortChartPaths sorts chart paths alphabetically for consistent display
func SortChartPaths(chartPaths []string) {
	sort.Strings(chartPaths)
}

// SortCharts sorts charts by name and then by version
func SortCharts(charts []*ChartInfo) {
	sort.Slice(charts, func(i, j int) bool {
		if charts[i].Name == charts[j].Name {
			return charts[i].Version < charts[j].Version
		}
		return charts[i].Name < charts[j].Name
	})
}

// isValidChartName performs basic validation of a chart name
func isValidChartName(name string) bool {
	if name == "" {
		return false
	}

	// Basic validation: should not contain path separators or spaces
	if strings.ContainsAny(name, "/\\ \t\n\r") {
		return false
	}

	// Should not start with a dot or dash
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "-") {
		return false
	}

	return true
}

// isValidVersion performs basic validation of a version string
func isValidVersion(version string) bool {
	if version == "" {
		return false
	}

	// Basic validation: should not contain path separators or spaces
	if strings.ContainsAny(version, "/\\ \t\n\r") {
		return false
	}

	return true
}
