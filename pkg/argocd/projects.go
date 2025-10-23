package argocd

import (
	"context"
	"fmt"
	"strings"

	"github.com/your-org/kargo-bootstrap/pkg/errors"
	"github.com/your-org/kargo-bootstrap/pkg/k8s"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

// Project represents an ArgoCD AppProject
type Project struct {
	Name                       string
	Description                string
	SourceRepos                []string
	Destinations               []Destination
	ClusterResourceWhitelist   []metav1.GroupKind
	NamespaceResourceBlacklist []metav1.GroupKind
	SyncWindows                []SyncWindow
}

// Destination represents a deployment destination in an AppProject
type Destination struct {
	Server    string
	Namespace string
	Name      string
}

// SyncWindow represents a sync window in an AppProject
type SyncWindow struct {
	Kind         string
	Schedule     string
	Duration     string
	Applications []string
	ManualSync   bool
}

// Client represents an ArgoCD client
type Client struct {
	dynamicClient dynamic.Interface
	namespace     string
}

// NewClient creates a new ArgoCD client
func NewClient(k8sClient *k8s.Client, namespace string) (*Client, error) {
	if k8sClient == nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeMissingRequired,
			"kubernetes client is required",
			errors.ErrorSeverityError,
			false,
		).WithSuggestions(
			"Provide a valid Kubernetes client",
			"Initialize the Kubernetes client before creating ArgoCD client",
		)
	}

	return &Client{
		dynamicClient: k8sClient.GetDynamicClient(),
		namespace:     namespace,
	}, nil
}

// ListProjects returns a list of available ArgoCD projects
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	klog.V(2).Info("Listing ArgoCD projects")

	// Get the AppProject GVR
	gvr := getProjectGVR()

	// List all AppProjects in the specified namespace
	resourceInterface := c.dynamicClient.Resource(gvr).Namespace(c.namespace)
	projects, err := resourceInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.NewAppError(
				errors.ErrorTypeArgoCD,
				errors.ErrCodeArgoCDConnection,
				"ArgoCD AppProjects CRD not found. Please ensure ArgoCD is installed",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithSuggestions(
				"Install ArgoCD in the cluster",
				"Check if ArgoCD is installed in the correct namespace",
				"Verify the AppProject CRD is available",
			)
		}
		return nil, errors.NewAppError(
			errors.ErrorTypeArgoCD,
			errors.ErrCodeArgoCDConnection,
			"failed to list AppProjects",
			errors.ErrorSeverityError,
			true, // Network issues might be retryable
		).WithCause(err).WithOperation("list_projects").WithNamespace(c.namespace)
	}

	var result []Project
	for _, item := range projects.Items {
		project, err := parseProjectFromUnstructured(item)
		if err != nil {
			klog.Warningf("Failed to parse project %s: %v", item.GetName(), err)
			continue
		}
		result = append(result, *project)
	}

	klog.V(1).Infof("Found %d ArgoCD projects", len(result))
	return result, nil
}

// GetProject gets details of a specific project
func (c *Client) GetProject(ctx context.Context, name string) (*Project, error) {
	klog.V(2).Infof("Getting ArgoCD project: %s", name)

	// Get the AppProject GVR
	gvr := getProjectGVR()

	// Get the specific AppProject
	resourceInterface := c.dynamicClient.Resource(gvr).Namespace(c.namespace)
	item, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.NewAppError(
				errors.ErrorTypeProject,
				errors.ErrCodeProjectNotFound,
				fmt.Sprintf("project %s not found", name),
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithResource(name).WithNamespace(c.namespace).WithSuggestions(
				"Check if the project name is correct",
				"Verify the project exists in the specified namespace",
				"List available projects to see valid options",
			)
		}
		return nil, errors.NewAppError(
			errors.ErrorTypeProject,
			errors.ErrCodeArgoCDConnection,
			fmt.Sprintf("failed to get project %s", name),
			errors.ErrorSeverityError,
			true,
		).WithCause(err).WithResource(name).WithNamespace(c.namespace)
	}

	project, err := parseProjectFromUnstructured(*item)
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeProject,
			errors.ErrCodeArgoCDConnection,
			fmt.Sprintf("failed to parse project %s", name),
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithResource(name).WithSuggestions(
			"Check if the project is properly formatted",
			"Verify the project configuration is valid",
		)
	}

	return project, nil
}

// ValidateProject checks if a project exists and is accessible
func (c *Client) ValidateProject(ctx context.Context, name string) error {
	klog.V(2).Infof("Validating ArgoCD project: %s", name)

	_, err := c.GetProject(ctx, name)
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeProject,
			errors.ErrCodeArgoCDConnection,
			"project validation failed",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithResource(name)
	}

	klog.V(1).Infof("Project %s is valid and accessible", name)
	return nil
}

// parseProjectFromUnstructured extracts project details from an unstructured Kubernetes resource
func parseProjectFromUnstructured(item unstructured.Unstructured) (*Project, error) {
	name := item.GetName()

	// Get description
	description, _, err := unstructured.NestedString(item.Object, "spec", "description")
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeProject,
			errors.ErrCodeArgoCDConnection,
			"failed to extract description",
			errors.ErrorSeverityWarning,
			false,
		).WithCause(err)
	}

	// Get source repositories
	sourceRepos, _, err := unstructured.NestedStringSlice(item.Object, "spec", "sourceRepos")
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeProject,
			errors.ErrCodeArgoCDConnection,
			"failed to extract sourceRepos",
			errors.ErrorSeverityWarning,
			false,
		).WithCause(err)
	}

	// Get destinations
	destinations := []Destination{}
	destinationsRaw, _, err := unstructured.NestedSlice(item.Object, "spec", "destinations")
	if err == nil {
		for _, destRaw := range destinationsRaw {
			if dest, ok := destRaw.(map[string]interface{}); ok {
				server, _, _ := unstructured.NestedString(dest, "server")
				namespace, _, _ := unstructured.NestedString(dest, "namespace")
				name, _, _ := unstructured.NestedString(dest, "name")

				destinations = append(destinations, Destination{
					Server:    server,
					Namespace: namespace,
					Name:      name,
				})
			}
		}
	}

	// Get cluster resource whitelist
	clusterResourceWhitelist := []metav1.GroupKind{}
	whitelistRaw, _, err := unstructured.NestedSlice(item.Object, "spec", "clusterResourceWhitelist")
	if err == nil {
		for _, itemRaw := range whitelistRaw {
			if item, ok := itemRaw.(map[string]interface{}); ok {
				group, _, _ := unstructured.NestedString(item, "group")
				kind, _, _ := unstructured.NestedString(item, "kind")

				clusterResourceWhitelist = append(clusterResourceWhitelist, metav1.GroupKind{
					Group: group,
					Kind:  kind,
				})
			}
		}
	}

	// Get namespace resource blacklist
	namespaceResourceBlacklist := []metav1.GroupKind{}
	blacklistRaw, _, err := unstructured.NestedSlice(item.Object, "spec", "namespaceResourceBlacklist")
	if err == nil {
		for _, itemRaw := range blacklistRaw {
			if item, ok := itemRaw.(map[string]interface{}); ok {
				group, _, _ := unstructured.NestedString(item, "group")
				kind, _, _ := unstructured.NestedString(item, "kind")

				namespaceResourceBlacklist = append(namespaceResourceBlacklist, metav1.GroupKind{
					Group: group,
					Kind:  kind,
				})
			}
		}
	}

	// Get sync windows
	syncWindows := []SyncWindow{}
	syncWindowsRaw, _, err := unstructured.NestedSlice(item.Object, "spec", "syncWindows")
	if err == nil {
		for _, windowRaw := range syncWindowsRaw {
			if window, ok := windowRaw.(map[string]interface{}); ok {
				kind, _, _ := unstructured.NestedString(window, "kind")
				schedule, _, _ := unstructured.NestedString(window, "schedule")
				duration, _, _ := unstructured.NestedString(window, "duration")
				manualSync, _, _ := unstructured.NestedBool(window, "manualSync")

				applications, _, _ := unstructured.NestedStringSlice(window, "applications")

				syncWindows = append(syncWindows, SyncWindow{
					Kind:         kind,
					Schedule:     schedule,
					Duration:     duration,
					Applications: applications,
					ManualSync:   manualSync,
				})
			}
		}
	}

	return &Project{
		Name:                       name,
		Description:                description,
		SourceRepos:                sourceRepos,
		Destinations:               destinations,
		ClusterResourceWhitelist:   clusterResourceWhitelist,
		NamespaceResourceBlacklist: namespaceResourceBlacklist,
		SyncWindows:                syncWindows,
	}, nil
}

// getProjectGVR returns the GroupVersionResource for AppProject CRD
func getProjectGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "appprojects",
	}
}

// IsArgoCDInstalled checks if ArgoCD is installed and accessible
func (c *Client) IsArgoCDInstalled(ctx context.Context) error {
	klog.V(2).Info("Checking if ArgoCD is installed")

	gvr := getProjectGVR()

	// Try to list AppProjects to check if the CRD is available
	resourceInterface := c.dynamicClient.Resource(gvr).Namespace(c.namespace)
	_, err := resourceInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return errors.NewAppError(
				errors.ErrorTypeArgoCD,
				errors.ErrCodeArgoCDConnection,
				"ArgoCD is not installed or AppProject CRD is not available",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithNamespace(c.namespace).WithSuggestions(
				"Install ArgoCD in the cluster",
				"Check if ArgoCD is installed in the correct namespace",
				"Verify the AppProject CRD is available",
			)
		}
		return errors.NewAppError(
			errors.ErrorTypeArgoCD,
			errors.ErrCodeArgoCDConnection,
			"failed to access ArgoCD",
			errors.ErrorSeverityError,
			true,
		).WithCause(err).WithNamespace(c.namespace).WithSuggestions(
			"Check if ArgoCD is running",
			"Verify network connectivity to ArgoCD",
			"Check RBAC permissions",
		)
	}

	klog.V(1).Info("ArgoCD is installed and accessible")
	return nil
}

// GetDefaultProject returns the default project if it exists
func (c *Client) GetDefaultProject(ctx context.Context) (*Project, error) {
	// Try to get the "default" project first
	project, err := c.GetProject(ctx, "default")
	if err == nil {
		return project, nil
	}

	// If default doesn't exist, get the first available project
	projects, err := c.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		return nil, errors.NewAppError(
			errors.ErrorTypeProject,
			errors.ErrCodeProjectNotFound,
			"no ArgoCD projects found",
			errors.ErrorSeverityError,
			false,
		).WithNamespace(c.namespace).WithSuggestions(
			"Create an ArgoCD project",
			"Check if projects exist in a different namespace",
			"Verify ArgoCD is properly installed",
		)
	}

	// Return the first project in the list
	klog.V(2).Infof("Using project '%s' as default", projects[0].Name)
	return &projects[0], nil
}

// FormatProjectList formats a list of projects for display
func FormatProjectList(projects []Project) string {
	if len(projects) == 0 {
		return "No ArgoCD projects found"
	}

	var builder strings.Builder
	builder.WriteString("Available ArgoCD Projects:\n")
	builder.WriteString("NAME\tDESCRIPTION\tSOURCE REPOS\n")
	builder.WriteString("----\t-----------\t-------------\n")

	for _, project := range projects {
		description := project.Description
		if description == "" {
			description = "No description"
		}

		sourceRepos := fmt.Sprintf("%d", len(project.SourceRepos))
		if len(project.SourceRepos) > 0 {
			if len(project.SourceRepos) == 1 {
				sourceRepos = project.SourceRepos[0]
			} else {
				sourceRepos = fmt.Sprintf("%d repos", len(project.SourceRepos))
			}
		}

		builder.WriteString(fmt.Sprintf("%s\t%s\t%s\n", project.Name, description, sourceRepos))
	}

	return builder.String()
}

// FormatProjectDetails formats detailed project information for display
func FormatProjectDetails(project *Project) string {
	if project == nil {
		return "Project not found"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Project: %s\n", project.Name))
	builder.WriteString(fmt.Sprintf("Description: %s\n", project.Description))

	builder.WriteString("\nSource Repositories:\n")
	if len(project.SourceRepos) == 0 {
		builder.WriteString("  None\n")
	} else {
		for _, repo := range project.SourceRepos {
			builder.WriteString(fmt.Sprintf("  - %s\n", repo))
		}
	}

	builder.WriteString("\nDestinations:\n")
	if len(project.Destinations) == 0 {
		builder.WriteString("  None\n")
	} else {
		for _, dest := range project.Destinations {
			if dest.Name != "" {
				builder.WriteString(fmt.Sprintf("  - %s (server: %s, namespace: %s)\n", dest.Name, dest.Server, dest.Namespace))
			} else {
				builder.WriteString(fmt.Sprintf("  - server: %s, namespace: %s\n", dest.Server, dest.Namespace))
			}
		}
	}

	builder.WriteString("\nCluster Resource Whitelist:\n")
	if len(project.ClusterResourceWhitelist) == 0 {
		builder.WriteString("  None\n")
	} else {
		for _, resource := range project.ClusterResourceWhitelist {
			builder.WriteString(fmt.Sprintf("  - %s/%s\n", resource.Group, resource.Kind))
		}
	}

	builder.WriteString("\nNamespace Resource Blacklist:\n")
	if len(project.NamespaceResourceBlacklist) == 0 {
		builder.WriteString("  None\n")
	} else {
		for _, resource := range project.NamespaceResourceBlacklist {
			builder.WriteString(fmt.Sprintf("  - %s/%s\n", resource.Group, resource.Kind))
		}
	}

	builder.WriteString("\nSync Windows:\n")
	if len(project.SyncWindows) == 0 {
		builder.WriteString("  None\n")
	} else {
		for _, window := range project.SyncWindows {
			builder.WriteString(fmt.Sprintf("  - %s: %s (duration: %s)\n", window.Kind, window.Schedule, window.Duration))
			if len(window.Applications) > 0 {
				builder.WriteString("    Applications: ")
				builder.WriteString(strings.Join(window.Applications, ", "))
				builder.WriteString("\n")
			}
			if window.ManualSync {
				builder.WriteString("    Manual sync enabled\n")
			}
		}
	}

	return builder.String()
}
