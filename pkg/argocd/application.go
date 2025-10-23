package argocd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	yamlPkg "github.com/your-org/kargo-bootstrap/pkg/yaml"
)

// ApplicationStatus represents the status of an ArgoCD Application
type ApplicationStatus struct {
	Name      string
	Namespace string
	Health    ApplicationHealthStatus
	Sync      ApplicationSyncStatus
	Operation *ApplicationOperationStatus
}

// ApplicationHealthStatus represents the health status of an Application
type ApplicationHealthStatus struct {
	Status  string
	Message string
}

// ApplicationSyncStatus represents the sync status of an Application
type ApplicationSyncStatus struct {
	Status     string
	Revision   string
	ComparedAt *metav1.Time
}

// ApplicationOperationStatus represents the status of an ongoing operation
type ApplicationOperationStatus struct {
	Phase      string
	Message    string
	StartedAt  *metav1.Time
	FinishedAt *metav1.Time
}

// ApplicationSyncOptions represents options for syncing an Application
type ApplicationSyncOptions struct {
	Revision    string
	DryRun      bool
	Prune       bool
	Force       bool
	SyncOptions []string
	Manifests   []string
}

// CreateApplication creates an ArgoCD Application from YAML
func (c *Client) CreateApplication(ctx context.Context, yamlBytes []byte) (*ApplicationStatus, error) {
	klog.V(2).Info("Creating ArgoCD Application from YAML")

	// Parse the YAML into an unstructured object
	app, err := ParseApplicationYAML(yamlBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Application YAML: %w", err)
	}

	// Validate the Application before creation
	if err := ValidateApplication(app); err != nil {
		return nil, fmt.Errorf("Application validation failed: %w", err)
	}

	// Get the Application GVR
	gvr := getApplicationGVR()

	// Create the Application
	resourceInterface := c.dynamicClient.Resource(gvr).Namespace(c.namespace)
	createdApp, err := resourceInterface.Create(ctx, app, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("Application '%s' already exists", app.GetName())
		}
		return nil, fmt.Errorf("failed to create Application: %w", err)
	}

	klog.V(1).Infof("Successfully created Application '%s'", createdApp.GetName())

	// Get the initial status
	status, err := GetApplicationStatusFromUnstructured(*createdApp)
	if err != nil {
		klog.Warningf("Failed to get initial Application status: %v", err)
		// Return a basic status if we can't get the detailed one
		return &ApplicationStatus{
			Name:      createdApp.GetName(),
			Namespace: createdApp.GetNamespace(),
		}, nil
	}

	return status, nil
}

// UpdateApplication updates an existing ArgoCD Application
func (c *Client) UpdateApplication(ctx context.Context, yamlBytes []byte) (*ApplicationStatus, error) {
	klog.V(2).Info("Updating ArgoCD Application from YAML")

	// Parse the YAML into an unstructured object
	app, err := ParseApplicationYAML(yamlBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Application YAML: %w", err)
	}

	// Validate the Application before update
	if err := ValidateApplication(app); err != nil {
		return nil, fmt.Errorf("Application validation failed: %w", err)
	}

	// Get the Application GVR
	gvr := getApplicationGVR()

	// Update the Application
	resourceInterface := c.dynamicClient.Resource(gvr).Namespace(c.namespace)
	updatedApp, err := resourceInterface.Update(ctx, app, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("Application '%s' not found", app.GetName())
		}
		return nil, fmt.Errorf("failed to update Application: %w", err)
	}

	klog.V(1).Infof("Successfully updated Application '%s'", updatedApp.GetName())

	// Get the updated status
	status, err := GetApplicationStatusFromUnstructured(*updatedApp)
	if err != nil {
		klog.Warningf("Failed to get updated Application status: %v", err)
		// Return a basic status if we can't get the detailed one
		return &ApplicationStatus{
			Name:      updatedApp.GetName(),
			Namespace: updatedApp.GetNamespace(),
		}, nil
	}

	return status, nil
}

// DeleteApplication deletes an ArgoCD Application
func (c *Client) DeleteApplication(ctx context.Context, name string) error {
	klog.V(2).Infof("Deleting ArgoCD Application: %s", name)

	// Get the Application GVR
	gvr := getApplicationGVR()

	// Delete the Application
	resourceInterface := c.dynamicClient.Resource(gvr).Namespace(c.namespace)
	err := resourceInterface.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("Application '%s' not found", name)
		}
		return fmt.Errorf("failed to delete Application: %w", err)
	}

	klog.V(1).Infof("Successfully deleted Application '%s'", name)
	return nil
}

// GetApplicationStatus gets the status of an ArgoCD Application
func (c *Client) GetApplicationStatus(ctx context.Context, name string) (*ApplicationStatus, error) {
	klog.V(2).Infof("Getting status for ArgoCD Application: %s", name)

	// Get the Application GVR
	gvr := getApplicationGVR()

	// Get the Application
	resourceInterface := c.dynamicClient.Resource(gvr).Namespace(c.namespace)
	app, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("Application '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to get Application: %w", err)
	}

	// Extract the status
	status, err := GetApplicationStatusFromUnstructured(*app)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Application status: %w", err)
	}

	return status, nil
}

// SyncApplication manually triggers a sync of an ArgoCD Application
func (c *Client) SyncApplication(ctx context.Context, name string, options *ApplicationSyncOptions) error {
	klog.V(2).Infof("Syncing ArgoCD Application: %s", name)

	// Get the Application GVR
	gvr := getApplicationGVR()

	// Get the Application
	resourceInterface := c.dynamicClient.Resource(gvr).Namespace(c.namespace)
	app, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("Application '%s' not found", name)
		}
		return fmt.Errorf("failed to get Application: %w", err)
	}

	// Create the sync operation
	syncOperation := map[string]interface{}{
		"operation": map[string]interface{}{
			"sync": map[string]interface{}{
				"revision": options.Revision,
			},
		},
	}

	// Add optional sync parameters
	if options.DryRun {
		syncOperation["operation"].(map[string]interface{})["sync"].(map[string]interface{})["dryRun"] = true
	}
	if options.Prune {
		syncOperation["operation"].(map[string]interface{})["sync"].(map[string]interface{})["prune"] = true
	}
	if options.Force {
		syncOperation["operation"].(map[string]interface{})["sync"].(map[string]interface{})["force"] = true
	}
	if len(options.SyncOptions) > 0 {
		syncOperation["operation"].(map[string]interface{})["sync"].(map[string]interface{})["syncOptions"] = options.SyncOptions
	}
	if len(options.Manifests) > 0 {
		syncOperation["operation"].(map[string]interface{})["sync"].(map[string]interface{})["manifests"] = options.Manifests
	}

	// Apply the sync operation
	app.Object["operation"] = syncOperation["operation"]
	_, err = resourceInterface.Update(ctx, app, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to trigger sync for Application '%s': %w", name, err)
	}

	klog.V(1).Infof("Successfully triggered sync for Application '%s'", name)
	return nil
}

// WaitForApplicationSync waits for an Application to sync
func (c *Client) WaitForApplicationSync(ctx context.Context, name string, timeout time.Duration) error {
	klog.V(2).Infof("Waiting for ArgoCD Application to sync: %s", name)

	return wait.PollImmediateUntil(5*time.Second, func() (bool, error) {
		status, err := c.GetApplicationStatus(ctx, name)
		if err != nil {
			return false, err
		}

		// Check if the sync is completed and healthy
		if status.Sync.Status == "Synced" && IsApplicationHealthy(status) {
			klog.V(1).Infof("Application '%s' is synced and healthy", name)
			return true, nil
		}

		// Check if there's an ongoing operation
		if status.Operation != nil && status.Operation.Phase == "Running" {
			klog.V(3).Infof("Application '%s' operation is running: %s", name, status.Operation.Message)
			return false, nil
		}

		// Check if the sync failed
		if status.Sync.Status == "Failed" || status.Sync.Status == "Unknown" {
			return false, fmt.Errorf("Application sync failed with status: %s", status.Sync.Status)
		}

		return false, nil
	}, ctx.Done())
}

// ParseApplicationYAML parses YAML into an unstructured Application object
func ParseApplicationYAML(yamlBytes []byte) (*unstructured.Unstructured, error) {
	obj, err := parseYAMLToUnstructured(yamlBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate that it's an Application
	if obj.GetKind() != "Application" {
		return nil, fmt.Errorf("expected kind 'Application', got '%s'", obj.GetKind())
	}

	if obj.GetAPIVersion() != "argoproj.io/v1alpha1" {
		return nil, fmt.Errorf("expected apiVersion 'argoproj.io/v1alpha1', got '%s'", obj.GetAPIVersion())
	}

	return obj, nil
}

// ValidateApplication validates an Application before creation
func ValidateApplication(app *unstructured.Unstructured) error {
	collector := yamlPkg.NewErrorCollector()

	// Check required metadata fields
	if app.GetName() == "" {
		collector.Add(yamlPkg.NewValidationError("Application name is required", "metadata.name", ""))
	}

	// Check required spec fields
	spec, ok := app.Object["spec"].(map[string]interface{})
	if !ok {
		collector.Add(yamlPkg.NewValidationError("Application spec is required", "spec", ""))
		return collector.ToError()
	}

	// Check project
	if _, ok := spec["project"]; !ok {
		collector.Add(yamlPkg.NewValidationError("Project is required", "spec.project", ""))
	}

	// Check source
	source, ok := spec["source"].(map[string]interface{})
	if !ok {
		collector.Add(yamlPkg.NewValidationError("Source is required", "spec.source", ""))
	} else {
		if _, ok := source["repoURL"]; !ok {
			collector.Add(yamlPkg.NewValidationError("Repository URL is required", "spec.source.repoURL", ""))
		}
		if _, ok := source["path"]; !ok {
			collector.Add(yamlPkg.NewValidationError("Path is required", "spec.source.path", ""))
		}
	}

	// Check destination
	destination, ok := spec["destination"].(map[string]interface{})
	if !ok {
		collector.Add(yamlPkg.NewValidationError("Destination is required", "spec.destination", ""))
	} else {
		if _, ok := destination["server"]; !ok {
			collector.Add(yamlPkg.NewValidationError("Destination server is required", "spec.destination.server", ""))
		}
		if _, ok := destination["namespace"]; !ok {
			collector.Add(yamlPkg.NewValidationError("Destination namespace is required", "spec.destination.namespace", ""))
		}
	}

	return collector.ToError()
}

// GetApplicationStatusFromUnstructured extracts Application status from an unstructured object
func GetApplicationStatusFromUnstructured(app unstructured.Unstructured) (*ApplicationStatus, error) {
	status := &ApplicationStatus{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
	}

	// Extract health status
	if health, ok := app.Object["status"].(map[string]interface{})["health"].(map[string]interface{}); ok {
		if healthStatus, ok := health["status"].(string); ok {
			status.Health.Status = healthStatus
		}
		if healthMessage, ok := health["message"].(string); ok {
			status.Health.Message = healthMessage
		}
	}

	// Extract sync status
	if sync, ok := app.Object["status"].(map[string]interface{})["sync"].(map[string]interface{}); ok {
		if syncStatus, ok := sync["status"].(string); ok {
			status.Sync.Status = syncStatus
		}
		if syncRevision, ok := sync["revision"].(string); ok {
			status.Sync.Revision = syncRevision
		}
		if comparedAtStr, ok := sync["comparedAt"].(string); ok {
			if comparedAt, err := time.Parse(time.RFC3339, comparedAtStr); err == nil {
				t := metav1.NewTime(comparedAt)
				status.Sync.ComparedAt = &t
			}
		}
	}

	// Extract operation status
	if operation, ok := app.Object["status"].(map[string]interface{})["operation"].(map[string]interface{}); ok {
		status.Operation = &ApplicationOperationStatus{}
		if phase, ok := operation["phase"].(string); ok {
			status.Operation.Phase = phase
		}
		if message, ok := operation["message"].(string); ok {
			status.Operation.Message = message
		}
		if startedAtStr, ok := operation["startedAt"].(string); ok {
			if startedAt, err := time.Parse(time.RFC3339, startedAtStr); err == nil {
				t := metav1.NewTime(startedAt)
				status.Operation.StartedAt = &t
			}
		}
		if finishedAtStr, ok := operation["finishedAt"].(string); ok {
			if finishedAt, err := time.Parse(time.RFC3339, finishedAtStr); err == nil {
				t := metav1.NewTime(finishedAt)
				status.Operation.FinishedAt = &t
			}
		}
	}

	return status, nil
}

// FormatApplicationStatus formats Application status for display
func FormatApplicationStatus(status *ApplicationStatus) string {
	if status == nil {
		return "Application status not available"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Application: %s\n", status.Name))
	builder.WriteString(fmt.Sprintf("Namespace: %s\n", status.Namespace))

	builder.WriteString("\nHealth Status:\n")
	if status.Health.Status != "" {
		builder.WriteString(fmt.Sprintf("  Status: %s\n", status.Health.Status))
		if status.Health.Message != "" {
			builder.WriteString(fmt.Sprintf("  Message: %s\n", status.Health.Message))
		}
	} else {
		builder.WriteString("  Status: Unknown\n")
	}

	builder.WriteString("\nSync Status:\n")
	if status.Sync.Status != "" {
		builder.WriteString(fmt.Sprintf("  Status: %s\n", status.Sync.Status))
		if status.Sync.Revision != "" {
			builder.WriteString(fmt.Sprintf("  Revision: %s\n", status.Sync.Revision))
		}
		if status.Sync.ComparedAt != nil {
			builder.WriteString(fmt.Sprintf("  Compared At: %s\n", status.Sync.ComparedAt.Format(time.RFC3339)))
		}
	} else {
		builder.WriteString("  Status: Unknown\n")
	}

	if status.Operation != nil {
		builder.WriteString("\nOperation Status:\n")
		builder.WriteString(fmt.Sprintf("  Phase: %s\n", status.Operation.Phase))
		if status.Operation.Message != "" {
			builder.WriteString(fmt.Sprintf("  Message: %s\n", status.Operation.Message))
		}
		if status.Operation.StartedAt != nil {
			builder.WriteString(fmt.Sprintf("  Started At: %s\n", status.Operation.StartedAt.Format(time.RFC3339)))
		}
		if status.Operation.FinishedAt != nil {
			builder.WriteString(fmt.Sprintf("  Finished At: %s\n", status.Operation.FinishedAt.Format(time.RFC3339)))
		}
	}

	return builder.String()
}

// IsApplicationHealthy checks if an Application is healthy
func IsApplicationHealthy(status *ApplicationStatus) bool {
	if status == nil {
		return false
	}

	// Check if health status is Healthy or Degraded (which means partially healthy)
	return status.Health.Status == "Healthy" || status.Health.Status == "Progressing" || status.Health.Status == "Degraded"
}

// getApplicationGVR returns the GroupVersionResource for Application CRD
func getApplicationGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}
}

// ListApplications returns a list of available ArgoCD Applications
func (c *Client) ListApplications(ctx context.Context) ([]unstructured.Unstructured, error) {
	klog.V(2).Info("Listing ArgoCD Applications")

	// Get the Application GVR
	gvr := getApplicationGVR()

	// List all Applications in the specified namespace
	resourceInterface := c.dynamicClient.Resource(gvr).Namespace(c.namespace)
	appList, err := resourceInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("ArgoCD Applications CRD not found. Please ensure ArgoCD is installed")
		}
		return nil, fmt.Errorf("failed to list Applications: %w", err)
	}

	klog.V(1).Infof("Found %d ArgoCD Applications", len(appList.Items))
	return appList.Items, nil
}

// GetApplication gets details of a specific Application
func (c *Client) GetApplication(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	klog.V(2).Infof("Getting ArgoCD Application: %s", name)

	// Get the Application GVR
	gvr := getApplicationGVR()

	// Get the specific Application
	resourceInterface := c.dynamicClient.Resource(gvr).Namespace(c.namespace)
	app, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("Application %s not found", name)
		}
		return nil, fmt.Errorf("failed to get Application %s: %w", name, err)
	}

	return app, nil
}

// ValidateApplicationName checks if an application name is valid
func ValidateApplicationName(name string) error {
	if name == "" {
		return fmt.Errorf("application name cannot be empty")
	}

	// Basic validation - could be extended with more specific rules
	if len(name) > 63 {
		return fmt.Errorf("application name cannot be longer than 63 characters")
	}

	// Check for invalid characters (basic check)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.') {
			return fmt.Errorf("application name can only contain lowercase alphanumeric characters, hyphens, and dots")
		}
	}

	// Check that it starts and ends with alphanumeric
	if name[0] == '-' || name[0] == '.' || name[len(name)-1] == '-' || name[len(name)-1] == '.' {
		return fmt.Errorf("application name must start and end with an alphanumeric character")
	}

	return nil
}

// parseYAMLToUnstructured parses YAML bytes into an unstructured object
func parseYAMLToUnstructured(yamlBytes []byte) (*unstructured.Unstructured, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal(yamlBytes, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &unstructured.Unstructured{Object: obj}, nil
}
