package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	yamlLib "sigs.k8s.io/yaml"

	"github.com/your-org/kargo-bootstrap/pkg/yaml"
)

var (
	renderAppName        string
	renderProject        string
	renderRepoURL        string
	renderPath           string
	renderTargetRevision string
	renderNamespace      string
	renderEnvironment    string
	renderImageRepo      string
	renderImageTag       string
	renderOutputFile     string
	renderValidate       bool
)

// renderCmd represents the render command
var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render ArgoCD Application YAML for testing",
	Long: `Render ArgoCD Application YAML manifests for testing purposes.

This command generates ArgoCD Application YAML without applying it to the cluster,
allowing you to preview and validate the configuration before deployment.`,
	Example: `  # Render an application with default values
  kargo-bootstrap render application --name myapp --project default --repo https://github.com/example/repo --path charts/myapp

  # Render with custom values
  kargo-bootstrap render application \
    --name myapp \
    --project default \
    --repo https://github.com/example/repo \
    --path charts/myapp \
    --environment production \
    --image-repo myregistry/myapp \
    --image-tag v1.0.0 \
    --output myapp.yaml

  # Render with validation
  kargo-bootstrap render application \
    --name myapp \
    --project default \
    --repo https://github.com/example/repo \
    --path charts/myapp \
    --validate`,
}

// applicationRenderCmd represents the render application subcommand
var applicationRenderCmd = &cobra.Command{
	Use:   "application",
	Short: "Render an ArgoCD Application manifest",
	Long: `Render an ArgoCD Application manifest as YAML.

This command generates an ArgoCD Application manifest based on the provided parameters,
allowing you to preview the configuration before applying it to the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate required parameters
		if renderAppName == "" {
			fmt.Fprintf(os.Stderr, "Error: application name is required (--name)\n")
			os.Exit(1)
		}
		if renderProject == "" {
			fmt.Fprintf(os.Stderr, "Error: project name is required (--project)\n")
			os.Exit(1)
		}
		if renderRepoURL == "" {
			fmt.Fprintf(os.Stderr, "Error: repository URL is required (--repo)\n")
			os.Exit(1)
		}
		if renderPath == "" {
			fmt.Fprintf(os.Stderr, "Error: chart path is required (--path)\n")
			os.Exit(1)
		}

		// Set defaults
		if renderTargetRevision == "" {
			renderTargetRevision = "main"
		}
		if renderEnvironment == "" {
			renderEnvironment = "development"
		}
		if renderNamespace == "" {
			renderNamespace = fmt.Sprintf("%s-%s", renderAppName, renderEnvironment)
		}
		if renderImageRepo == "" {
			renderImageRepo = fmt.Sprintf("ethannguyen98/%s", renderAppName)
		}
		if renderImageTag == "" {
			renderImageTag = "v0.0.1"
		}

		// Initialize YAML processor
		yamlProcessor, err := yaml.NewProcessor()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing YAML processor: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Rendering ArgoCD Application for '%s' in environment '%s'...\n", renderAppName, renderEnvironment)

		// Generate Helm values
		helmValues := yamlProcessor.GenerateHelmValues(
			renderAppName,
			renderEnvironment,
			renderImageRepo,
			renderImageTag,
			renderAppName+"-"+renderEnvironment+"-secret",
			&yaml.UptraceConfig{
				Enabled:     true,
				ServiceName: renderAppName + "-" + renderEnvironment,
				Endpoint:    "http://uptrace-collector.uptrace.svc:4318",
				Headers: map[string]string{
					"uptrace-dsn": "http://token@uptrace-collector.uptrace.svc:4318/2",
				},
			},
		)

		// Generate application configuration
		appConfig := yamlProcessor.GenerateApplicationConfig(
			renderAppName,
			renderProject,
			renderRepoURL,
			renderPath,
			renderTargetRevision,
			renderNamespace,
			renderEnvironment,
			helmValues,
		)

		// Render the Application YAML
		renderOptions := yaml.RenderOptions{
			DryRun:   true,
			Validate: renderValidate,
		}

		yamlBytes, err := yamlProcessor.RenderApplication(appConfig, renderOptions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering Application YAML: %v\n", err)
			os.Exit(1)
		}

		// Validate if requested
		if renderValidate {
			validation := yamlProcessor.ValidateApplication(appConfig)
			if !validation.Valid {
				fmt.Fprintf(os.Stderr, "Validation failed:\n")
				for _, errMsg := range validation.Errors {
					fmt.Fprintf(os.Stderr, "  - %s\n", errMsg)
				}
				os.Exit(1)
			}

			if len(validation.Warnings) > 0 {
				fmt.Printf("⚠️  Validation warnings:\n")
				for _, warning := range validation.Warnings {
					fmt.Printf("  - %s\n", warning)
				}
			}
			fmt.Printf("✅ Validation passed\n\n")
		}

		// Output the YAML
		if renderOutputFile != "" {
			err = os.WriteFile(renderOutputFile, yamlBytes, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to file %s: %v\n", renderOutputFile, err)
				os.Exit(1)
			}
			fmt.Printf("✅ ArgoCD Application YAML written to %s\n", renderOutputFile)
		} else {
			fmt.Printf("--- Generated ArgoCD Application YAML ---\n")
			fmt.Printf("%s\n", string(yamlBytes))
			fmt.Printf("--- End YAML ---\n")
		}

		// Display summary
		fmt.Printf("\nSummary:\n")
		fmt.Printf("  Application: %s\n", renderAppName)
		fmt.Printf("  Environment: %s\n", renderEnvironment)
		fmt.Printf("  Project: %s\n", renderProject)
		fmt.Printf("  Repository: %s\n", renderRepoURL)
		fmt.Printf("  Path: %s\n", renderPath)
		fmt.Printf("  Target Revision: %s\n", renderTargetRevision)
		fmt.Printf("  Namespace: %s\n", renderNamespace)
		fmt.Printf("  Image: %s:%s\n", renderImageRepo, renderImageTag)
	},

	// Add completion support for the application subcommand
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

// helmValuesRenderCmd represents the render helm-values subcommand
var helmValuesRenderCmd = &cobra.Command{
	Use:   "helm-values",
	Short: "Render Helm values for an application",
	Long: `Render Helm values for an application based on the provided parameters.

This command generates the Helm values that would be used in an ArgoCD Application,
allowing you to preview and customize the values before deployment.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate required parameters
		if renderAppName == "" {
			fmt.Fprintf(os.Stderr, "Error: application name is required (--name)\n")
			os.Exit(1)
		}

		// Set defaults
		if renderEnvironment == "" {
			renderEnvironment = "development"
		}
		if renderImageRepo == "" {
			renderImageRepo = fmt.Sprintf("ethannguyen98/%s", renderAppName)
		}
		if renderImageTag == "" {
			renderImageTag = "v0.0.1"
		}

		// Initialize YAML processor
		yamlProcessor, err := yaml.NewProcessor()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing YAML processor: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Rendering Helm values for '%s' in environment '%s'...\n", renderAppName, renderEnvironment)

		// Generate Helm values
		helmValues := yamlProcessor.GenerateHelmValues(
			renderAppName,
			renderEnvironment,
			renderImageRepo,
			renderImageTag,
			renderAppName+"-"+renderEnvironment+"-secret",
			&yaml.UptraceConfig{
				Enabled:     true,
				ServiceName: renderAppName + "-" + renderEnvironment,
				Endpoint:    "http://uptrace-collector.uptrace.svc:4318",
				Headers: map[string]string{
					"uptrace-dsn": "http://token@uptrace-collector.uptrace.svc:4318/2",
				},
			},
		)

		// Convert to YAML
		yamlBytes, err := yamlLib.Marshal(helmValues)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling Helm values to YAML: %v\n", err)
			os.Exit(1)
		}

		// Output the YAML
		if renderOutputFile != "" {
			err = os.WriteFile(renderOutputFile, yamlBytes, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to file %s: %v\n", renderOutputFile, err)
				os.Exit(1)
			}
			fmt.Printf("✅ Helm values YAML written to %s\n", renderOutputFile)
		} else {
			fmt.Printf("--- Generated Helm Values YAML ---\n")
			fmt.Printf("%s\n", string(yamlBytes))
			fmt.Printf("--- End YAML ---\n")
		}

		// Display summary
		fmt.Printf("\nSummary:\n")
		fmt.Printf("  Application: %s\n", renderAppName)
		fmt.Printf("  Environment: %s\n", renderEnvironment)
		fmt.Printf("  Image: %s:%s\n", renderImageRepo, renderImageTag)
	},
}

func init() {
	rootCmd.AddCommand(renderCmd)
	renderCmd.AddCommand(applicationRenderCmd)
	renderCmd.AddCommand(helmValuesRenderCmd)

	// Common flags for all render subcommands
	applicationRenderCmd.Flags().StringVar(&renderAppName, "name", "", "Application name (required)")
	applicationRenderCmd.Flags().StringVar(&renderProject, "project", "", "ArgoCD project name (required)")
	applicationRenderCmd.Flags().StringVar(&renderRepoURL, "repo", "", "Git repository URL (required)")
	applicationRenderCmd.Flags().StringVar(&renderPath, "path", "", "Path to Helm chart (required)")
	applicationRenderCmd.Flags().StringVar(&renderTargetRevision, "revision", "", "Target revision (default: main)")
	applicationRenderCmd.Flags().StringVar(&renderNamespace, "namespace", "", "Target namespace (default: <app>-<env>)")
	applicationRenderCmd.Flags().StringVar(&renderEnvironment, "environment", "", "Environment (default: development)")
	applicationRenderCmd.Flags().StringVar(&renderImageRepo, "image-repo", "", "Image repository (default: ethannguyen98/<app>)")
	applicationRenderCmd.Flags().StringVar(&renderImageTag, "image-tag", "", "Image tag (default: v0.0.1)")
	applicationRenderCmd.Flags().StringVar(&renderOutputFile, "output", "", "Output file (default: stdout)")
	applicationRenderCmd.Flags().BoolVar(&renderValidate, "validate", true, "Validate the generated YAML")

	// Flags for helm-values subcommand
	helmValuesRenderCmd.Flags().StringVar(&renderAppName, "name", "", "Application name (required)")
	helmValuesRenderCmd.Flags().StringVar(&renderEnvironment, "environment", "", "Environment (default: development)")
	helmValuesRenderCmd.Flags().StringVar(&renderImageRepo, "image-repo", "", "Image repository (default: ethannguyen98/<app>)")
	helmValuesRenderCmd.Flags().StringVar(&renderImageTag, "image-tag", "", "Image tag (default: v0.0.1)")
	helmValuesRenderCmd.Flags().StringVar(&renderOutputFile, "output", "", "Output file (default: stdout)")

	// Add completion support
	applicationRenderCmd.RegisterFlagCompletionFunc("project", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// TODO: Implement project completion from ArgoCD
		return []string{"default", "production", "staging"}, cobra.ShellCompDirectiveNoFileComp
	})

	applicationRenderCmd.RegisterFlagCompletionFunc("environment", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"development", "staging", "production"}, cobra.ShellCompDirectiveNoFileComp
	})
}
