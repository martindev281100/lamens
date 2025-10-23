package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/your-org/kargo-bootstrap/pkg/errors"
)

var (
	// Global flags
	kubeconfigPath  string
	argoCDNamespace string
	verbose         bool
	version         = "dev"
	commit          = "none"
	date            = "unknown"

	// Global error handler
	errorHandler *errors.ErrorHandler
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kargo-bootstrap",
	Short: "Bootstrap Kargo projects with ArgoCD integration",
	Long: `kargo-bootstrap is a CLI tool to bootstrap Kargo projects with ArgoCD integration.

This tool helps you set up and manage Kargo projects with ArgoCD integration,
providing a streamlined workflow for deploying applications to Kubernetes clusters
using GitOps principles with Helm charts.

Key Features:
- Interactive deployment workflow with step-by-step guidance
- Git repository integration with support for private repositories
- Multi-environment deployment (development, staging, production)
- ArgoCD Application management and monitoring
- Helm chart discovery and validation
- Environment configuration and customization
- Dry-run mode for previewing changes
- Non-interactive mode for CI/CD automation

For more information, visit: https://github.com/your-org/kargo-bootstrap`,
	Example: `  # Get help for any command
  kargo-bootstrap help [command]

  # Deploy an application interactively
  kargo-bootstrap deploy

  # Deploy with specific parameters
  kargo-bootstrap deploy --repository-url https://github.com/example/helm-charts --environments staging,production

  # Preview deployment without making changes
  kargo-bootstrap deploy --dry-run

  # Test ArgoCD connectivity
  kargo-bootstrap argocd test

  # List available ArgoCD projects
  kargo-bootstrap argocd list

  # Discover Helm charts in a repository
  kargo-bootstrap git list-charts --url https://github.com/example/helm-charts

  # List available environments
  kargo-bootstrap env list

  # Render ArgoCD Application YAML
  kargo-bootstrap render application --name myapp --project default --repo https://github.com/example/helm-charts --path charts/myapp`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize the global error handler with the current verbose setting
		errorHandler = errors.NewErrorHandler(verbose)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Handle the error with the global error handler
		errorHandler.HandleWithExit(err)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&kubeconfigPath, "kubeconfig", "",
		"Path to the kubeconfig file (default is $HOME/.kube/config)")
	rootCmd.PersistentFlags().StringVar(&argoCDNamespace, "argocd-namespace", "argocd",
		"Namespace where ArgoCD is installed")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"Enable verbose output with additional details and suggestions")

	// Add version command
	rootCmd.AddCommand(versionCmd)
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long: `Display version information for kargo-bootstrap.

This command shows the current version, commit hash, and build date
of the kargo-bootstrap binary.`,
	Example: `  # Display version information
  kargo-bootstrap version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kargo-bootstrap version %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
	},
}
