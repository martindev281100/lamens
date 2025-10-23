package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/your-org/kargo-bootstrap/pkg/argocd"
	"github.com/your-org/kargo-bootstrap/pkg/k8s"
	"github.com/your-org/kargo-bootstrap/pkg/summary"
)

var (
	showProjectDetails bool
	projectName        string
	applicationName    string
	yamlFilePath       string
	revision           string
	appDryRun          bool
	prune              bool
	force              bool
	argoAppAutoApprove bool
)

// argocdCmd represents the argocd command
var argocdCmd = &cobra.Command{
	Use:   "argocd",
	Short: "Interact with ArgoCD",
	Long: `Interact with ArgoCD to list projects and verify connectivity.

This command provides functionality to list ArgoCD projects, get details
about specific projects, and test the connection to ArgoCD.`,
	Example: `  # List all ArgoCD projects
  kargo-bootstrap argocd list

  # Get details about a specific project
  kargo-bootstrap argocd get --project my-project

  # Test ArgoCD connectivity
  kargo-bootstrap argocd test

  # List projects with detailed information
  kargo-bootstrap argocd list --details`,
}

// argocdListCmd represents the argocd list command
var argocdListCmd = &cobra.Command{
	Use:   "list",
	Short: "List ArgoCD projects",
	Long: `List all available ArgoCD projects in the configured namespace.

This command fetches and displays all ArgoCD AppProjects that are
available in the specified ArgoCD namespace.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create Kubernetes client configuration
		config := &k8s.Config{
			KubeconfigPath: getKubeconfigPath(),
		}

		// Create Kubernetes client
		client, err := k8s.NewClient(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
			os.Exit(1)
		}

		// Test the connection
		fmt.Println("Testing Kubernetes connection...")
		if err := client.TestConnection(); err != nil {
			fmt.Fprintf(os.Stderr, "Connection test failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Successfully connected to Kubernetes")

		// Initialize ArgoCD client
		fmt.Printf("Initializing ArgoCD client in namespace '%s'...\n", argoCDNamespace)
		argoClient, err := argocd.NewClient(client, argoCDNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating ArgoCD client: %v\n", err)
			os.Exit(1)
		}

		// Check if ArgoCD is installed and accessible
		if err := argoClient.IsArgoCDInstalled(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing ArgoCD: %v\n", err)
			fmt.Fprintf(os.Stderr, "Please ensure ArgoCD is installed in namespace '%s'\n", argoCDNamespace)
			os.Exit(1)
		}

		fmt.Println("✅ Successfully connected to ArgoCD")

		// List available projects
		fmt.Println("\nFetching ArgoCD projects...")
		projects, err := argoClient.ListProjects(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing ArgoCD projects: %v\n", err)
			os.Exit(1)
		}

		if len(projects) == 0 {
			fmt.Println("No ArgoCD projects found")
			return
		}

		fmt.Printf("✅ Found %d ArgoCD project(s)\n", len(projects))

		if showProjectDetails {
			// Show detailed information for each project
			for _, project := range projects {
				fmt.Printf("\n%s\n", argocd.FormatProjectDetails(&project))
				fmt.Println(strings.Repeat("-", 80))
			}
		} else {
			// Show tabular format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION\tSOURCE REPOS\tDESTINATIONS")
			fmt.Fprintln(w, "----\t-----------\t------------\t------------")

			for _, project := range projects {
				description := project.Description
				if description == "" {
					description = "No description"
				}

				sourceRepos := fmt.Sprintf("%d", len(project.SourceRepos))
				if len(project.SourceRepos) > 0 {
					if len(project.SourceRepos) == 1 {
						sourceRepos = project.SourceRepos[0]
						// Truncate if too long
						if len(sourceRepos) > 30 {
							sourceRepos = sourceRepos[:27] + "..."
						}
					} else {
						sourceRepos = fmt.Sprintf("%d repos", len(project.SourceRepos))
					}
				}

				destinations := fmt.Sprintf("%d", len(project.Destinations))
				if len(project.Destinations) > 0 {
					destinations = fmt.Sprintf("%d dest", len(project.Destinations))
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", project.Name, description, sourceRepos, destinations)
			}

			w.Flush()
		}
	},
}

// argocdGetCmd represents the argocd get command
var argocdGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a specific ArgoCD project",
	Long: `Get detailed information about a specific ArgoCD project.

This command fetches and displays detailed information about the
specified ArgoCD AppProject, including source repositories,
destinations, and sync windows.`,
	Example: `  # Get details about a specific project
  kargo-bootstrap argocd get --project my-project`,
	Run: func(cmd *cobra.Command, args []string) {
		if projectName == "" {
			fmt.Fprintf(os.Stderr, "Error: --project flag is required\n")
			os.Exit(1)
		}

		// Create Kubernetes client configuration
		config := &k8s.Config{
			KubeconfigPath: getKubeconfigPath(),
		}

		// Create Kubernetes client
		client, err := k8s.NewClient(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
			os.Exit(1)
		}

		// Initialize ArgoCD client
		argoClient, err := argocd.NewClient(client, argoCDNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating ArgoCD client: %v\n", err)
			os.Exit(1)
		}

		// Get the project details
		fmt.Printf("Getting details for project '%s'...\n", projectName)
		project, err := argoClient.GetProject(context.Background(), projectName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting project: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Project found")
		fmt.Println(argocd.FormatProjectDetails(project))
	},
}

// argocdTestCmd represents the argocd test command
var argocdTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test ArgoCD connectivity",
	Long: `Test the connection to ArgoCD and verify that it is properly installed.

This command tests the connection to the Kubernetes cluster and then
verifies that ArgoCD is installed and accessible in the specified namespace.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create Kubernetes client configuration
		config := &k8s.Config{
			KubeconfigPath: getKubeconfigPath(),
		}

		// Create Kubernetes client
		client, err := k8s.NewClient(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
			os.Exit(1)
		}

		// Test the Kubernetes connection
		fmt.Println("Testing Kubernetes connection...")
		if err := client.TestConnection(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Kubernetes connection test failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Successfully connected to Kubernetes")

		// Get current context if possible
		if contextName, err := k8s.GetCurrentContext(config.KubeconfigPath); err == nil {
			fmt.Printf("Current context: %s\n", contextName)
		} else {
			klog.V(2).Infof("Could not determine current context: %v", err)
		}

		// Initialize ArgoCD client
		fmt.Printf("\nTesting ArgoCD connectivity in namespace '%s'...\n", argoCDNamespace)
		argoClient, err := argocd.NewClient(client, argoCDNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error creating ArgoCD client: %v\n", err)
			os.Exit(1)
		}

		// Check if ArgoCD is installed and accessible
		if err := argoClient.IsArgoCDInstalled(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "❌ ArgoCD connection test failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "Please ensure ArgoCD is installed in namespace '%s'\n", argoCDNamespace)
			os.Exit(1)
		}
		fmt.Println("✅ Successfully connected to ArgoCD")

		// Try to list projects to verify full functionality
		fmt.Println("\nTesting project listing...")
		projects, err := argoClient.ListProjects(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error listing projects: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Successfully listed %d project(s)\n", len(projects))
		if len(projects) > 0 {
			fmt.Println("Available projects:")
			for _, project := range projects {
				fmt.Printf("  - %s\n", project.Name)
			}
		}

		fmt.Println("\n✅ All ArgoCD connectivity tests passed!")
	},
}

// argocdCreateCmd represents the argocd create command
var argocdCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an ArgoCD Application from YAML",
	Long: `Create an ArgoCD Application from a YAML file.

This command reads an Application definition from a YAML file and
creates it in the specified ArgoCD namespace.`,
	Example: `  # Create an Application from a YAML file
	kargo-bootstrap argocd create --file app.yaml

	# Create an Application in dry-run mode
	kargo-bootstrap argocd create --file app.yaml --dry-run

	# Create an Application with automatic approval
	kargo-bootstrap argocd create --file app.yaml --auto-approve`,
	Run: func(cmd *cobra.Command, args []string) {
		if yamlFilePath == "" {
			fmt.Fprintf(os.Stderr, "Error: --file flag is required\n")
			os.Exit(1)
		}

		// Create Kubernetes client configuration
		config := &k8s.Config{
			KubeconfigPath: getKubeconfigPath(),
		}

		// Create Kubernetes client
		client, err := k8s.NewClient(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
			os.Exit(1)
		}

		// Initialize ArgoCD client
		argoClient, err := argocd.NewClient(client, argoCDNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating ArgoCD client: %v\n", err)
			os.Exit(1)
		}

		// Read the YAML file
		yamlBytes, err := os.ReadFile(yamlFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading YAML file: %v\n", err)
			os.Exit(1)
		}

		// Parse the YAML to extract application details using a simple parser
		// Create a simple application structure for the summary
		application := &summary.ApplicationYAMLSummary{
			Name:        "unknown",
			Namespace:   argoCDNamespace,
			Project:     "default",
			SourceRepo:  "unknown",
			SourcePath:  "unknown",
			Destination: "unknown",
		}

		// Try to extract basic information from the YAML
		yamlStr := string(yamlBytes)
		lines := strings.Split(yamlStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "name:") {
				application.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			} else if strings.HasPrefix(line, "namespace:") {
				application.Namespace = strings.TrimSpace(strings.TrimPrefix(line, "namespace:"))
			} else if strings.HasPrefix(line, "project:") {
				application.Project = strings.TrimSpace(strings.TrimPrefix(line, "project:"))
			} else if strings.Contains(line, "repoURL:") {
				if idx := strings.Index(line, "repoURL:"); idx != -1 {
					application.SourceRepo = strings.TrimSpace(line[idx+8:])
				}
			} else if strings.Contains(line, "path:") {
				if idx := strings.Index(line, "path:"); idx != -1 {
					application.SourcePath = strings.TrimSpace(line[idx+5:])
				}
			} else if strings.Contains(line, "server:") {
				if idx := strings.Index(line, "server:"); idx != -1 {
					application.Destination = strings.TrimSpace(line[idx+7:])
				}
			}
		}

		// Build ArgoCD create summary
		summaryBuilder := summary.NewSummaryBuilder()
		summaryBuilder.WithMetadata(appDryRun, argoAppAutoApprove)

		// Get current context if possible
		contextName := ""
		if ctx, err := k8s.GetCurrentContext(config.KubeconfigPath); err == nil {
			contextName = ctx
		}

		summaryBuilder.WithKubernetes(getKubeconfigPath(), contextName)

		// Check if ArgoCD is installed
		argoCDInstalled := true
		if err := argoClient.IsArgoCDInstalled(context.Background()); err != nil {
			argoCDInstalled = false
		}

		summaryBuilder.WithArgoCD(argoCDNamespace, nil, argoCDInstalled)

		// Create a simple summary for the ArgoCD application
		createSummary := &summary.ArgoCDCreateSummary{
			Metadata: summary.SummaryMetadata{
				GeneratedAt: time.Now(),
				DryRun:      appDryRun,
				AutoApprove: argoAppAutoApprove,
			},
			Kubernetes: summary.KubernetesSummary{
				KubeconfigPath: getKubeconfigPath(),
				Context:        contextName,
			},
			ArgoCD: summary.ArgoCDSummary{
				Namespace: argoCDNamespace,
				Installed: argoCDInstalled,
			},
			Application: *application,
		}

		// Display the summary
		fmt.Println(summary.FormatArgoCDCreateSummary(createSummary))

		// In dry-run mode, show a prominent message and exit
		if appDryRun {
			fmt.Println("\n" + strings.Repeat("=", 60))
			fmt.Println("🔍 DRY-RUN MODE: No resources will be created")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Println("This is a preview of what would be created.")
			fmt.Println("To apply these changes, run the command without --dry-run flag.")
			fmt.Println(strings.Repeat("=", 60))
			return
		}

		// Confirm creation unless in auto-approve mode
		if !argoAppAutoApprove {
			confirmOptions := &summary.ConfirmationOptions{
				Message:     "Do you want to create this ArgoCD Application?",
				Default:     false,
				ShowHelp:    true,
				AllowModify: false,
				AllowBack:   false,
				AutoApprove: argoAppAutoApprove,
				DryRun:      appDryRun,
			}

			result, err := summary.ConfirmArgoCDCreate(createSummary, confirmOptions)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting confirmation: %v\n", err)
				os.Exit(1)
			}

			if !result.Confirmed || result.Cancelled {
				fmt.Println("Application creation cancelled by user")
				os.Exit(0)
			}
		} else {
			fmt.Println("Auto-approving application creation due to --auto-approve flag")
		}

		// Create the Application
		fmt.Printf("Creating Application from '%s'...\n", yamlFilePath)
		status, err := argoClient.CreateApplication(context.Background(), yamlBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Application: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Successfully created Application\n")
		if status != nil {
			fmt.Printf("Name: %s\n", status.Name)
			fmt.Printf("Namespace: %s\n", status.Namespace)
			fmt.Printf("Health: %s\n", status.Health.Status)
			fmt.Printf("Sync: %s\n", status.Sync.Status)
		}
	},
}

// argocdDeleteCmd represents the argocd delete command
var argocdDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an ArgoCD Application",
	Long: `Delete an ArgoCD Application by name.

This command deletes the specified Application from the ArgoCD namespace.`,
	Example: `  # Delete an Application
  kargo-bootstrap argocd delete --name my-app`,
	Run: func(cmd *cobra.Command, args []string) {
		if applicationName == "" {
			fmt.Fprintf(os.Stderr, "Error: --name flag is required\n")
			os.Exit(1)
		}

		// Create Kubernetes client configuration
		config := &k8s.Config{
			KubeconfigPath: getKubeconfigPath(),
		}

		// Create Kubernetes client
		client, err := k8s.NewClient(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
			os.Exit(1)
		}

		// Initialize ArgoCD client
		argoClient, err := argocd.NewClient(client, argoCDNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating ArgoCD client: %v\n", err)
			os.Exit(1)
		}

		// Delete the Application
		fmt.Printf("Deleting Application '%s'...\n", applicationName)
		err = argoClient.DeleteApplication(context.Background(), applicationName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting Application: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Successfully deleted Application '%s'\n", applicationName)
	},
}

// argocdStatusCmd represents the argocd status command
var argocdStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of an ArgoCD Application",
	Long: `Check the status of an ArgoCD Application by name.

This command retrieves and displays the current status of the specified Application.`,
	Example: `  # Check the status of an Application
  kargo-bootstrap argocd status --name my-app`,
	Run: func(cmd *cobra.Command, args []string) {
		if applicationName == "" {
			fmt.Fprintf(os.Stderr, "Error: --name flag is required\n")
			os.Exit(1)
		}

		// Create Kubernetes client configuration
		config := &k8s.Config{
			KubeconfigPath: getKubeconfigPath(),
		}

		// Create Kubernetes client
		client, err := k8s.NewClient(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
			os.Exit(1)
		}

		// Initialize ArgoCD client
		argoClient, err := argocd.NewClient(client, argoCDNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating ArgoCD client: %v\n", err)
			os.Exit(1)
		}

		// Get the Application status
		fmt.Printf("Getting status for Application '%s'...\n", applicationName)
		status, err := argoClient.GetApplicationStatus(context.Background(), applicationName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting Application status: %v\n", err)
			os.Exit(1)
		}

		// Display the status
		fmt.Println(argocd.FormatApplicationStatus(status))
	},
}

// argocdSyncCmd represents the argocd sync command
var argocdSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manually sync an ArgoCD Application",
	Long: `Manually trigger a sync of an ArgoCD Application.

This command triggers a manual sync of the specified Application with its source repository.`,
	Example: `  # Sync an Application
  kargo-bootstrap argocd sync --name my-app

  # Sync an Application with a specific revision
  kargo-bootstrap argocd sync --name my-app --revision main`,
	Run: func(cmd *cobra.Command, args []string) {
		if applicationName == "" {
			fmt.Fprintf(os.Stderr, "Error: --name flag is required\n")
			os.Exit(1)
		}

		// Create Kubernetes client configuration
		config := &k8s.Config{
			KubeconfigPath: getKubeconfigPath(),
		}

		// Create Kubernetes client
		client, err := k8s.NewClient(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
			os.Exit(1)
		}

		// Initialize ArgoCD client
		argoClient, err := argocd.NewClient(client, argoCDNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating ArgoCD client: %v\n", err)
			os.Exit(1)
		}

		// Create sync options
		options := &argocd.ApplicationSyncOptions{
			Revision: revision,
			DryRun:   appDryRun,
			Prune:    prune,
			Force:    force,
		}

		// Sync the Application
		fmt.Printf("Syncing Application '%s'...\n", applicationName)
		if revision != "" {
			fmt.Printf("Using revision: %s\n", revision)
		}
		err = argoClient.SyncApplication(context.Background(), applicationName, options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error syncing Application: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Successfully triggered sync for Application '%s'\n", applicationName)
	},
}

// argocdListAppsCmd represents the argocd list-apps command
var argocdListAppsCmd = &cobra.Command{
	Use:   "list-apps",
	Short: "List ArgoCD Applications",
	Long: `List all available ArgoCD Applications in the configured namespace.

This command fetches and displays all ArgoCD Applications that are
available in the specified ArgoCD namespace.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create Kubernetes client configuration
		config := &k8s.Config{
			KubeconfigPath: getKubeconfigPath(),
		}

		// Create Kubernetes client
		client, err := k8s.NewClient(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
			os.Exit(1)
		}

		// Initialize ArgoCD client
		argoClient, err := argocd.NewClient(client, argoCDNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating ArgoCD client: %v\n", err)
			os.Exit(1)
		}

		// List Applications
		fmt.Println("Fetching ArgoCD Applications...")
		apps, err := argoClient.ListApplications(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing ArgoCD Applications: %v\n", err)
			os.Exit(1)
		}

		if len(apps) == 0 {
			fmt.Println("No ArgoCD Applications found")
			return
		}

		fmt.Printf("✅ Found %d ArgoCD Application(s)\n", len(apps))

		// Display in tabular format
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tNAMESPACE\tPROJECT\tHEALTH\tSYNC\tREVISION")
		fmt.Fprintln(w, "----\t--------\t-------\t------\t----\t--------")

		for _, app := range apps {
			name := app.GetName()
			namespace := app.GetNamespace()

			// Get project from spec
			project := "N/A"
			if spec, ok := app.Object["spec"].(map[string]interface{}); ok {
				if p, ok := spec["project"].(string); ok {
					project = p
				}
			}

			// Get health status
			health := "Unknown"
			if status, ok := app.Object["status"].(map[string]interface{}); ok {
				if h, ok := status["health"].(map[string]interface{}); ok {
					if s, ok := h["status"].(string); ok {
						health = s
					}
				}
			}

			// Get sync status
			sync := "Unknown"
			revision := "N/A"
			if status, ok := app.Object["status"].(map[string]interface{}); ok {
				if s, ok := status["sync"].(map[string]interface{}); ok {
					if st, ok := s["status"].(string); ok {
						sync = st
					}
					if r, ok := s["revision"].(string); ok {
						revision = r
						// Truncate if too long
						if len(revision) > 7 {
							revision = revision[:7] + "..."
						}
					}
				}
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", name, namespace, project, health, sync, revision)
		}

		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(argocdCmd)
	argocdCmd.AddCommand(argocdListCmd)
	argocdCmd.AddCommand(argocdGetCmd)
	argocdCmd.AddCommand(argocdTestCmd)
	argocdCmd.AddCommand(argocdCreateCmd)
	argocdCmd.AddCommand(argocdDeleteCmd)
	argocdCmd.AddCommand(argocdStatusCmd)
	argocdCmd.AddCommand(argocdSyncCmd)
	argocdCmd.AddCommand(argocdListAppsCmd)

	// Flags for argocd list command
	argocdListCmd.Flags().BoolVar(&showProjectDetails, "details", false,
		"Show detailed information for each project")

	// Flags for argocd get command
	argocdGetCmd.Flags().StringVar(&projectName, "project", "",
		"Name of the project to get details for")
	argocdGetCmd.MarkFlagRequired("project")

	// Flags for argocd create command
	argocdCreateCmd.Flags().StringVar(&yamlFilePath, "file", "",
		"Path to the YAML file containing the Application definition")
	argocdCreateCmd.MarkFlagRequired("file")
	argocdCreateCmd.Flags().BoolVar(&appDryRun, "dry-run", false,
		"Perform a dry-run without creating the Application")
	argocdCreateCmd.Flags().BoolVar(&argoAppAutoApprove, "auto-approve", false,
		"Automatically approve application creation without confirmation")

	// Flags for argocd delete command
	argocdDeleteCmd.Flags().StringVar(&applicationName, "name", "",
		"Name of the Application to delete")
	argocdDeleteCmd.MarkFlagRequired("name")

	// Flags for argocd status command
	argocdStatusCmd.Flags().StringVar(&applicationName, "name", "",
		"Name of the Application to get status for")
	argocdStatusCmd.MarkFlagRequired("name")

	// Flags for argocd sync command
	argocdSyncCmd.Flags().StringVar(&applicationName, "name", "",
		"Name of the Application to sync")
	argocdSyncCmd.MarkFlagRequired("name")
	argocdSyncCmd.Flags().StringVar(&revision, "revision", "",
		"Specific revision to sync (default: latest)")
	argocdSyncCmd.Flags().BoolVar(&appDryRun, "dry-run", false,
		"Perform a dry-run without syncing")
	argocdSyncCmd.Flags().BoolVar(&prune, "prune", false,
		"Prune resources during sync")
	argocdSyncCmd.Flags().BoolVar(&force, "force", false,
		"Force sync even if no changes detected")
}
