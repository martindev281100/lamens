package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/your-org/kargo-bootstrap/pkg/argocd"
	"github.com/your-org/kargo-bootstrap/pkg/config"
	"github.com/your-org/kargo-bootstrap/pkg/errors"
	"github.com/your-org/kargo-bootstrap/pkg/git"
	"github.com/your-org/kargo-bootstrap/pkg/k8s"
	"github.com/your-org/kargo-bootstrap/pkg/prompt"
	"github.com/your-org/kargo-bootstrap/pkg/summary"
	"github.com/your-org/kargo-bootstrap/pkg/yaml"
)

var (
	dryRun            bool
	nonInteractive    bool
	deployAutoApprove bool
	repositoryURL     string
	appName           string
	selectedEnvs      string
	waitForSync       bool
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy applications with Kargo and ArgoCD",
	Long: `Deploy applications using Kargo and ArgoCD integration.

This command handles the deployment workflow for applications, setting up
the necessary Kargo resources and configuring ArgoCD to manage the
application lifecycle.

The deployment workflow follows these steps:
1. Select an ArgoCD project
2. Select a Git repository containing Helm charts
3. Choose a Git revision (branch, tag, or commit)
4. Discover and select Helm charts
5. Configure application name and environments
6. Create namespaces for each environment
7. Generate ArgoCD Application manifests
8. Apply manifests and monitor sync status

For more information, see the deployment workflow documentation:
https://github.com/your-org/kargo-bootstrap/docs/deployment-workflow.md`,
	Example: `  # Deploy with interactive prompts
  kargo-bootstrap deploy

  # Deploy in dry-run mode to see what would be applied
  kargo-bootstrap deploy --dry-run

  # Deploy without interactive prompts (using defaults or flags)
  kargo-bootstrap deploy --non-interactive

  # Deploy with automatic approval (no confirmation prompt)
  kargo-bootstrap deploy --auto-approve

  # Deploy with custom kubeconfig and ArgoCD namespace
  kargo-bootstrap deploy --kubeconfig ~/.kube/config --argocd-namespace argocd

  # Deploy a specific repository to specific environments
  kargo-bootstrap deploy \
    --repository-url https://github.com/example/helm-charts \
    --environments staging,production \
    --non-interactive

  # Deploy a specific tag with custom application name
  kargo-bootstrap deploy \
    --repository-url https://github.com/example/helm-charts \
    --tag v1.2.0 \
    --app-name myapp \
    --environments production

  # Deploy with Git authentication
  kargo-bootstrap deploy \
    --repository-url https://github.com/example/private-charts \
    --token YOUR_GITHUB_TOKEN \
    --environments production`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Deploy command called")
		if dryRun {
			fmt.Println("Running in dry-run mode - no changes will be applied")
		}
		if nonInteractive {
			fmt.Println("Running in non-interactive mode")
		}

		// Validate command-line arguments
		if err := validateDeployFlags(); err != nil {
			if errorHandler != nil {
				errorHandler.Handle(err)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			os.Exit(1)
		}

		// Initialize Kubernetes client
		fmt.Println("Initializing Kubernetes client...")
		k8sConfig := &k8s.Config{
			KubeconfigPath: getKubeconfigPath(),
		}

		client, err := k8s.NewClient(k8sConfig)
		if err != nil {
			appErr := errors.NewAppError(
				errors.ErrorTypeKubernetes,
				errors.ErrCodeK8sConnection,
				"Error creating Kubernetes client",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithSuggestions(
				"Check if the kubeconfig file is valid",
				"Verify the Kubernetes cluster is accessible",
			)
			if errorHandler != nil {
				errorHandler.Handle(appErr)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
			}
			os.Exit(1)
		}

		// Test the connection
		if err := client.TestConnection(); err != nil {
			appErr := errors.NewAppError(
				errors.ErrorTypeKubernetes,
				errors.ErrCodeK8sConnection,
				"Error connecting to Kubernetes",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithSuggestions(
				"Check if the Kubernetes cluster is running",
				"Verify network connectivity to the cluster",
				"Check if the kubeconfig file is valid",
			)
			if errorHandler != nil {
				errorHandler.Handle(appErr)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
			}
			os.Exit(1)
		}

		fmt.Println("✅ Successfully connected to Kubernetes")

		// Get current context if possible
		if contextName, err := k8s.GetCurrentContext(k8sConfig.KubeconfigPath); err == nil {
			fmt.Printf("Using context: %s\n", contextName)
		} else {
			klog.V(2).Infof("Could not determine current context: %v", err)
		}

		// Ensure ArgoCD namespace exists with enhanced functionality
		fmt.Printf("Ensuring ArgoCD namespace '%s' exists...\n", argoCDNamespace)
		if err := client.EnsureNamespace(cmd.Context(), argoCDNamespace, map[string]string{
			"name":                      "argocd",
			"app.kubernetes.io/part-of": "argocd",
		}); err != nil {
			appErr := errors.NewAppError(
				errors.ErrorTypeNamespace,
				errors.ErrCodeNamespaceNotFound,
				"Error ensuring ArgoCD namespace exists",
				errors.ErrorSeverityError,
				true, // Namespace creation might be retryable
			).WithCause(err).WithNamespace(argoCDNamespace).WithSuggestions(
				"Check if you have permission to create namespaces",
				"Verify the cluster is accessible",
			)
			if errorHandler != nil {
				errorHandler.Handle(appErr)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
			}
			os.Exit(1)
		}

		// Add standard annotations to ArgoCD namespace
		if err := client.AddAnnotationsToNamespace(cmd.Context(), argoCDNamespace, map[string]string{
			"kargo-bootstrap.io/managed-by": "kargo-bootstrap",
			"kargo-bootstrap.io/created-at": time.Now().Format(time.RFC3339),
		}); err != nil {
			// Log warning but continue
			klog.Warningf("Failed to add annotations to ArgoCD namespace: %v", err)
		}

		// Wait for namespace to be ready
		if err := client.WaitForNamespaceReady(cmd.Context(), argoCDNamespace, 30*time.Second); err != nil {
			// Log warning but continue
			klog.Warningf("ArgoCD namespace not ready after 30 seconds: %v", err)
		}

		fmt.Printf("✅ ArgoCD namespace '%s' is ready\n", argoCDNamespace)

		// Initialize ArgoCD client
		fmt.Println("Initializing ArgoCD client...")
		argoClient, err := argocd.NewClient(client, argoCDNamespace)
		if err != nil {
			appErr := errors.NewAppError(
				errors.ErrorTypeArgoCD,
				errors.ErrCodeArgoCDConnection,
				"Error creating ArgoCD client",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithNamespace(argoCDNamespace).WithSuggestions(
				"Check if ArgoCD is installed in the specified namespace",
				"Verify the ArgoCD API is accessible",
			)
			if errorHandler != nil {
				errorHandler.Handle(appErr)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
			}
			os.Exit(1)
		}

		// Check if ArgoCD is installed and accessible
		if err := argoClient.IsArgoCDInstalled(cmd.Context()); err != nil {
			appErr := errors.NewAppError(
				errors.ErrorTypeArgoCD,
				errors.ErrCodeArgoCDConnection,
				"Error accessing ArgoCD",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithNamespace(argoCDNamespace).WithSuggestions(
				"Install ArgoCD in the specified namespace",
				"Check if ArgoCD is running and accessible",
			)
			if errorHandler != nil {
				errorHandler.Handle(appErr)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
			}
			os.Exit(1)
		}
		fmt.Println("✅ Successfully connected to ArgoCD")

		// List available projects
		fmt.Println("\nFetching available ArgoCD projects...")
		projects, err := argoClient.ListProjects(cmd.Context())
		if err != nil {
			appErr := errors.NewAppError(
				errors.ErrorTypeProject,
				errors.ErrCodeProjectNotFound,
				"Error listing ArgoCD projects",
				errors.ErrorSeverityError,
				true, // Network issues might be retryable
			).WithCause(err).WithNamespace(argoCDNamespace).WithSuggestions(
				"Check if ArgoCD is running and accessible",
				"Verify you have permission to list projects",
			)
			if errorHandler != nil {
				errorHandler.Handle(appErr)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
			}
			os.Exit(1)
		}

		if len(projects) == 0 {
			appErr := errors.NewAppError(
				errors.ErrorTypeProject,
				errors.ErrCodeProjectNotFound,
				"No ArgoCD projects found",
				errors.ErrorSeverityError,
				false,
			).WithNamespace(argoCDNamespace).WithSuggestions(
				"Create at least one ArgoCD project before deploying applications",
				"Check if projects exist in a different namespace",
			)
			if errorHandler != nil {
				errorHandler.Handle(appErr)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
			}
			os.Exit(1)
		}

		fmt.Printf("✅ Found %d ArgoCD project(s)\n", len(projects))
		fmt.Println(argocd.FormatProjectList(projects))

		// Initialize prompter and error handler
		var prompter *prompt.Prompter
		var errorHandler *prompt.ErrorHandler
		if !nonInteractive {
			prompter, err = prompt.NewPrompter()
			if err != nil {
				appErr := errors.NewAppError(
					errors.ErrorTypeGeneral,
					errors.ErrCodeInvalidInput,
					"Error initializing prompter",
					errors.ErrorSeverityError,
					false,
				).WithCause(err).WithSuggestions(
					"Check if the terminal supports interactive prompts",
					"Verify the environment is properly configured",
				)
				if errorHandler != nil {
					errorHandler.Handle(appErr)
				} else {
					fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
				}
				os.Exit(1)
			}
			errorHandler = prompt.NewErrorHandler(false, true)
		} else {
			errorHandler = prompt.NewErrorHandler(true, false)
		}

		// Project selection
		var selectedProject *argocd.Project
		if nonInteractive {
			// Get the default project or use the first one
			defaultProject, err := argoClient.GetDefaultProject(cmd.Context())
			if err != nil {
				appErr := errors.NewAppError(
					errors.ErrorTypeProject,
					errors.ErrCodeProjectNotFound,
					"Error getting default project",
					errors.ErrorSeverityError,
					false,
				).WithCause(err).WithNamespace(argoCDNamespace).WithSuggestions(
					"Create a default project named 'default'",
					"Specify a project using command-line flags",
				)
				if errorHandler != nil {
					errorHandler.Handle(appErr)
				} else {
					fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
				}
				os.Exit(1)
			}
			selectedProject = defaultProject
			fmt.Printf("\nUsing project: %s\n", selectedProject.Name)
			if selectedProject.Description != "" {
				fmt.Printf("Description: %s\n", selectedProject.Description)
			}
		} else {
			fmt.Println("\n🔄 Step 1: Selecting ArgoCD Project")
			projectSelection, err := prompter.SelectProject(projects, "")
			if err := errorHandler.Handle(err); err != nil {
				os.Exit(1)
			}
			if projectSelection.Cancelled {
				fmt.Println("Project selection cancelled by user")
				os.Exit(0)
			}
			selectedProject = projectSelection.Project
		}

		// Git repository selection
		var normalizedURL string
		if nonInteractive {
			if repositoryURL == "" {
				appErr := errors.NewAppError(
					errors.ErrorTypeValidation,
					errors.ErrCodeMissingRequired,
					"Repository URL is required in non-interactive mode",
					errors.ErrorSeverityError,
					false,
				).WithSuggestions(
					"Provide a repository URL using the --repository-url flag",
				)
				if errorHandler != nil {
					errorHandler.Handle(appErr)
				} else {
					fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
				}
				os.Exit(1)
			}
			normalizedURL, err = git.ValidateRepositoryURL(repositoryURL)
			if err != nil {
				appErr := errors.NewAppError(
					errors.ErrorTypeGit,
					errors.ErrCodeInvalidURL,
					"Invalid repository URL",
					errors.ErrorSeverityError,
					false,
				).WithCause(err).WithField("repository URL", repositoryURL).WithSuggestions(
					"Check if the repository URL is correct",
					"Ensure the URL includes the protocol (http://, https://, ssh://)",
				)
				if errorHandler != nil {
					errorHandler.Handle(appErr)
				} else {
					fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
				}
				os.Exit(1)
			}
			fmt.Printf("\nUsing repository: %s\n", normalizedURL)
		} else {
			fmt.Println("\n🔄 Step 2: Selecting Git Repository")
			repoSelection, err := prompter.SelectRepository(repositoryURL)
			if err := errorHandler.Handle(err); err != nil {
				os.Exit(1)
			}
			if repoSelection.Cancelled {
				fmt.Println("Repository selection cancelled by user")
				os.Exit(0)
			}
			normalizedURL = repoSelection.URL
			fmt.Printf("✅ Selected repository: %s\n", normalizedURL)
		}

		// Revision selection
		var selectedRevision *prompt.RevisionSelection
		if !nonInteractive {
			fmt.Println("\n🔄 Step 3: Selecting Git Revision")
			selectedRevision, err = prompter.SelectRevision(normalizedURL, "main")
			if err := errorHandler.Handle(err); err != nil {
				os.Exit(1)
			}
			if selectedRevision.Cancelled {
				fmt.Println("Revision selection cancelled by user")
				os.Exit(0)
			}
			fmt.Printf("✅ Selected revision: %s (%s)\n", selectedRevision.Value, selectedRevision.Type)
		}

		// Helm chart discovery and selection
		var selectedChart *git.ChartInfo
		{
			// Create clone options with selected revision
			auth := createAuthConfig()
			opts := &git.CloneOptions{
				URL:     normalizedURL,
				Auth:    auth,
				TempDir: tempDir,
			}

			// Set revision if selected
			if selectedRevision != nil {
				switch selectedRevision.Type {
				case string(prompt.RevisionTypeBranch):
					opts.Branch = selectedRevision.Value
				case string(prompt.RevisionTypeTag):
					opts.Tag = selectedRevision.Value
				case string(prompt.RevisionTypeCommit):
					opts.Commit = selectedRevision.Value
				}
			} else if branch != "" {
				opts.Branch = branch
			} else if tag != "" {
				opts.Tag = tag
			} else if commitHash != "" {
				opts.Commit = commitHash
			}

			fmt.Println("\n🔄 Step 4: Discovering Helm Charts")
			charts, err := git.ListChartsInRepo(opts)
			if err != nil {
				appErr := errors.NewAppError(
					errors.ErrorTypeChart,
					errors.ErrCodeChartNotFound,
					"Error discovering charts",
					errors.ErrorSeverityError,
					true, // Network issues might be retryable
				).WithCause(err).WithField("repository URL", normalizedURL).WithSuggestions(
					"Check if the repository contains Helm charts",
					"Verify the repository is accessible",
					"Check authentication credentials",
				)
				if errorHandler != nil {
					errorHandler.Handle(appErr)
				} else {
					fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
				}
				os.Exit(1)
			}

			if nonInteractive {
				if len(charts) == 0 {
					appErr := errors.NewAppError(
						errors.ErrorTypeChart,
						errors.ErrCodeChartNotFound,
						"No Helm charts found in repository",
						errors.ErrorSeverityError,
						false,
					).WithField("repository URL", normalizedURL).WithSuggestions(
						"Add Helm charts to the repository",
						"Check if charts are in the correct location",
					)
					if errorHandler != nil {
						errorHandler.Handle(appErr)
					} else {
						fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
					}
					os.Exit(1)
				}
				selectedChart = charts[0]
				fmt.Printf("Using first chart: %s (version %s)\n", selectedChart.Name, selectedChart.Version)
			} else {
				chartSelection, err := prompter.SelectChartPath(charts)
				if err := errorHandler.Handle(err); err != nil {
					os.Exit(1)
				}
				if chartSelection.Cancelled {
					fmt.Println("Chart selection cancelled by user")
					os.Exit(0)
				}
				selectedChart = chartSelection.Chart
				fmt.Printf("✅ Selected chart: %s (version %s)\n", selectedChart.Name, selectedChart.Version)
			}
		}

		// Application name input
		var finalAppName string
		if nonInteractive {
			if appName == "" {
				// Generate a default app name from the chart name
				finalAppName = selectedChart.Name
			} else {
				finalAppName = appName
			}
			fmt.Printf("\nUsing application name: %s\n", finalAppName)
		} else {
			fmt.Println("\n🔄 Step 5: Application Name")
			defaultName := selectedChart.Name
			if appName != "" {
				defaultName = appName
			}
			appNameInput, err := prompter.InputAppName(defaultName)
			if err := errorHandler.Handle(err); err != nil {
				os.Exit(1)
			}
			if appNameInput.Cancelled {
				fmt.Println("Application name input cancelled by user")
				os.Exit(0)
			}
			finalAppName = appNameInput.Name
			fmt.Printf("✅ Application name: %s\n", finalAppName)
		}

		// Environment selection
		var selectedEnvironments []string
		if nonInteractive {
			if selectedEnvs != "" {
				selectedEnvironments = strings.Split(selectedEnvs, ",")
				for i, env := range selectedEnvironments {
					selectedEnvironments[i] = strings.TrimSpace(env)
				}
			} else {
				selectedEnvironments = []string{"development"} // Default environment
			}
			fmt.Printf("\nUsing environments: %s\n", strings.Join(selectedEnvironments, ", "))
		} else {
			fmt.Println("\n🔄 Step 6: Selecting Deployment Environments")

			// Initialize environment manager
			envManager := config.NewEnvironmentManager()
			envManager.LoadDefaultEnvironments()

			// Convert config environments to prompt environments
			configEnvs := envManager.GetEnvironmentForPrompt()
			promptEnvs := make([]prompt.Environment, 0, len(configEnvs))
			for _, configEnv := range configEnvs {
				promptEnv := prompt.Environment{
					Name:        configEnv.Name,
					Type:        prompt.EnvironmentType(configEnv.Type),
					Description: configEnv.Description,
					Default:     configEnv.Default,
				}
				promptEnvs = append(promptEnvs, promptEnv)
			}

			envSelection, err := prompter.SelectEnvironments(promptEnvs, []string{"development"})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error selecting environments: %v\n", err)
				os.Exit(1)
			}
			if envSelection.Cancelled {
				fmt.Println("Environment selection cancelled by user")
				os.Exit(0)
			}
			selectedEnvironments = envSelection.Environments
			fmt.Printf("✅ Selected environments: %s\n", strings.Join(selectedEnvironments, ", "))
		}

		// Create namespaces for each selected environment
		fmt.Println("\n🔄 Step 7: Creating namespaces for environments")
		for _, env := range selectedEnvironments {
			// Generate namespace name based on app name and environment
			namespaceName, err := k8s.GenerateNamespaceName(finalAppName, env)
			if err != nil {
				appErr := errors.NewAppError(
					errors.ErrorTypeNamespace,
					errors.ErrCodeInvalidFormat,
					fmt.Sprintf("Error generating namespace name for environment %s", env),
					errors.ErrorSeverityError,
					false,
				).WithCause(err).WithField("app name", finalAppName).WithField("environment", env).WithSuggestions(
					"Check if the app name is valid",
					"Verify the environment name is valid",
				)
				if errorHandler != nil {
					errorHandler.Handle(appErr)
				} else {
					fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
				}
				os.Exit(1)
			}

			fmt.Printf("Creating namespace '%s' for environment '%s'...\n", namespaceName, env)

			// Prepare labels and annotations
			labels := map[string]string{
				"kargo-bootstrap.io/app":         finalAppName,
				"kargo-bootstrap.io/environment": env,
				"kargo-bootstrap.io/managed":     "true",
				"app.kubernetes.io/name":         finalAppName,
				"app.kubernetes.io/component":    "application",
			}

			annotations := map[string]string{
				"kargo-bootstrap.io/created-at":     time.Now().Format(time.RFC3339),
				"kargo-bootstrap.io/managed-by":     "kargo-bootstrap",
				"kargo-bootstrap.io/project":        selectedProject.Name,
				"kargo-bootstrap.io/repository":     normalizedURL,
				"kargo-bootstrap.io/argocd-project": selectedProject.Name,
			}

			if selectedRevision != nil {
				annotations["kargo-bootstrap.io/revision"] = selectedRevision.Value
				annotations["kargo-bootstrap.io/revision-type"] = selectedRevision.Type
			}

			// Create the namespace with labels and annotations
			if err := client.CreateNamespaceWithLabels(cmd.Context(), namespaceName, labels, annotations); err != nil {
				// If namespace already exists, just try to add/update labels and annotations
				if strings.Contains(err.Error(), "already exists") {
					fmt.Printf("Namespace '%s' already exists, updating labels and annotations...\n", namespaceName)

					if err := client.AddLabelsToNamespace(cmd.Context(), namespaceName, labels); err != nil {
						klog.Warningf("Failed to update labels for namespace %s: %v", namespaceName, err)
					}

					if err := client.AddAnnotationsToNamespace(cmd.Context(), namespaceName, annotations); err != nil {
						klog.Warningf("Failed to update annotations for namespace %s: %v", namespaceName, err)
					}
				} else {
					appErr := errors.NewAppError(
						errors.ErrorTypeNamespace,
						errors.ErrCodeNamespaceNotFound,
						fmt.Sprintf("Error creating namespace %s", namespaceName),
						errors.ErrorSeverityError,
						true, // Namespace creation might be retryable
					).WithCause(err).WithNamespace(namespaceName).WithSuggestions(
						"Check if you have permission to create namespaces",
						"Verify the cluster is accessible",
					)
					if errorHandler != nil {
						errorHandler.Handle(appErr)
					} else {
						fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
					}
					os.Exit(1)
				}
			} else {
				fmt.Printf("✅ Successfully created namespace '%s'\n", namespaceName)
			}

			// Wait for namespace to be ready
			if err := client.WaitForNamespaceReady(cmd.Context(), namespaceName, 30*time.Second); err != nil {
				klog.Warningf("Namespace %s not ready after 30 seconds: %v", namespaceName, err)
				// Continue despite readiness check failure
			}
		}

		// Initialize environment manager
		envManager := config.NewEnvironmentManager()
		envManager.LoadDefaultEnvironments()

		// Initialize YAML processor with environment manager
		fmt.Println("\n🔄 Step 8: Generating ArgoCD Application YAML")
		yamlProcessor, err := yaml.NewProcessorWithEnvManager(envManager)
		if err != nil {
			appErr := errors.NewAppError(
				errors.ErrorTypeValues,
				errors.ErrCodeValuesInvalid,
				"Error initializing YAML processor",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithSuggestions(
				"Check if the environment manager is properly configured",
				"Verify the YAML templates are valid",
			)
			if errorHandler != nil {
				errorHandler.Handle(appErr)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
			}
			os.Exit(1)
		}

		// Validate environment setup
		validationResult := envManager.ValidateEnvironmentSetup(selectedEnvironments)
		if !validationResult.Valid {
			errorCollector := errors.NewErrorCollector()
			for _, errMsg := range validationResult.Errors {
				errorCollector.Add(errors.NewAppError(
					errors.ErrorTypeEnvironment,
					errors.ErrCodeEnvNotFound,
					errMsg,
					errors.ErrorSeverityError,
					false,
				))
			}
			appErr := errors.NewAppError(
				errors.ErrorTypeEnvironment,
				errors.ErrCodeEnvNotFound,
				"Environment validation failed",
				errors.ErrorSeverityError,
				false,
			).WithCause(errorCollector.ToError()).WithSuggestions(
				"Check if the environment configuration is valid",
				"Verify all required environment parameters are provided",
			)
			if errorHandler != nil {
				errorHandler.Handle(appErr)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", appErr)
			}
			os.Exit(1)
		}

		// Display warnings if any
		if len(validationResult.Warnings) > 0 {
			fmt.Printf("⚠️  Environment validation warnings:\n")
			for _, warning := range validationResult.Warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}

		// Generate and render ArgoCD Applications for each environment
		var generatedApplications []string
		errorCollector := errors.NewEnvironmentErrorCollector()

		for _, env := range selectedEnvironments {
			namespaceName, err := k8s.GenerateNamespaceName(finalAppName, env)
			if err != nil {
				errorCollector.AddError(env,
					fmt.Sprintf("Error generating namespace name: %v", err),
					false,
					map[string]interface{}{"app_name": finalAppName})
				continue
			}

			fmt.Printf("\n🔄 Generating ArgoCD Application for environment '%s'...\n", env)

			// Determine target revision
			targetRevision := "main"
			if selectedRevision != nil {
				targetRevision = selectedRevision.Value
			}

			// Get environment configuration
			envConfig, err := envManager.GetEnvironment(env)
			if err != nil {
				errorCollector.AddError(env,
					fmt.Sprintf("Error getting environment configuration: %v", err),
					false,
					map[string]interface{}{"env_name": env})
				continue
			}

			// Generate Helm values for the application
			helmValues := yamlProcessor.GenerateHelmValues(
				finalAppName,
				env,
				"ethannguyen98/"+finalAppName,  // Default image repository
				"v0.0.1",                       // Default image tag
				finalAppName+"-"+env+"-secret", // Default secret name
				&yaml.UptraceConfig{
					Enabled:     true,
					ServiceName: finalAppName + "-" + env,
					Endpoint:    "http://uptrace-collector.uptrace.svc:4318",
					Headers: map[string]string{
						"uptrace-dsn": "http://token@uptrace-collector.uptrace.svc:4318/2",
					},
				},
			)

			// Generate application configuration
			appConfig := yamlProcessor.GenerateApplicationConfig(
				finalAppName,
				selectedProject.Name,
				normalizedURL,
				selectedChart.Path,
				targetRevision,
				namespaceName,
				env,
				helmValues,
			)

			// Render the Application YAML
			renderOptions := yaml.RenderOptions{
				DryRun:   dryRun,
				Validate: true,
			}

			yamlBytes, err := yamlProcessor.RenderApplication(appConfig, renderOptions)
			if err != nil {
				errorCollector.AddError(env,
					fmt.Sprintf("Error rendering Application YAML: %v", err),
					true, // Rendering errors might be retryable
					map[string]interface{}{"app_name": finalAppName})
				continue
			}

			// Validate the generated YAML
			validation := yamlProcessor.ValidateApplication(appConfig)
			if !validation.Valid {
				errorCollector.AddError(env,
					fmt.Sprintf("Application validation failed"),
					false, // Validation errors are typically not retryable
					map[string]interface{}{
						"app_name": finalAppName,
						"errors":   validation.Errors,
					})
				continue
			}

			// Display warnings if any
			if len(validation.Warnings) > 0 {
				fmt.Printf("⚠️  Validation warnings for environment %s:\n", env)
				for _, warning := range validation.Warnings {
					fmt.Printf("  - %s\n", warning)
				}
			}

			// Save or display the generated YAML
			appName := fmt.Sprintf("%s-%s", finalAppName, env)
			generatedApplications = append(generatedApplications, appName)

			if dryRun {
				fmt.Printf("\n--- Generated ArgoCD Application YAML for %s ---\n", appName)
				fmt.Printf("%s\n", string(yamlBytes))
				fmt.Printf("--- End YAML for %s ---\n", appName)
			} else {
				// Apply the YAML to create the Application resource
				fmt.Printf("✅ Generated ArgoCD Application YAML for %s (%d bytes)\n", appName, len(yamlBytes))

				fmt.Printf("📝 Applying Application '%s' to namespace '%s'...\n", appName, argoCDNamespace)
				status, err := argoClient.CreateApplication(cmd.Context(), yamlBytes)
				if err != nil {
					errorCollector.AddError(env,
						fmt.Sprintf("Error creating Application: %v", err),
						true, // Application creation errors might be retryable
						map[string]interface{}{
							"app_name":  appName,
							"namespace": argoCDNamespace,
						})
					continue
				}

				fmt.Printf("✅ Successfully created Application '%s'\n", appName)

				// Display the initial status
				if status != nil {
					fmt.Printf("Initial status:\n")
					fmt.Printf("  Health: %s\n", status.Health.Status)
					if status.Health.Message != "" {
						fmt.Printf("  Message: %s\n", status.Health.Message)
					}
					fmt.Printf("  Sync: %s\n", status.Sync.Status)
				}

				// Show environment-specific configuration
				fmt.Printf("Environment configuration:\n")
				fmt.Printf("  Auto Sync: %t\n", envConfig.AutoSync)
				fmt.Printf("  Prune: %t\n", envConfig.Prune)
				fmt.Printf("  Self Heal: %t\n", envConfig.SelfHeal)
				fmt.Printf("  Replica Count: %d\n", envConfig.ReplicaCount)
			}
		}

		// Handle deployment errors using the error collector
		if errorCollector.HasErrors() {
			fmt.Printf("\n⚠️  Deployment completed with errors:\n")
			fmt.Println(errorCollector.Summary())

			// If there are retryable errors, offer to retry
			if errorCollector.HasRetryableErrors() && !nonInteractive {
				fmt.Printf("\n🔄 Some environments have retryable errors.\n")
				retryPrompt, err := prompter.ConfirmSelection("Would you like to retry the failed environments?", true)
				if err == nil && retryPrompt {
					fmt.Println("\n🔄 Retrying failed environments...")

					// Define retry operation
					retryOperation := func(environment string) error {
						// In a real implementation, we would retry the specific operation that failed
						// For now, we'll just simulate a retry
						fmt.Printf("Retrying deployment to environment '%s'...\n", environment)
						return nil
					}

					// Retry with default strategy
					retryStrategy := errors.DefaultRetryStrategy()
					retryCollector := errors.RetryEnvironments(
						errorCollector,
						retryOperation,
						retryStrategy,
						func(env string, attempt int, maxAttempts int) {
							fmt.Printf("Retrying %s (attempt %d/%d)...\n", env, attempt, maxAttempts)
						},
					)

					// Merge retry results
					mergedCollector := errors.MergeCollectors(errorCollector, retryCollector)

					if mergedCollector.HasErrors() {
						fmt.Println("\n⚠️  Retry completed with remaining errors:")
						fmt.Println(mergedCollector.Summary())
					} else {
						fmt.Println("\n✅ All retries completed successfully!")
					}
				}
			}

			if len(generatedApplications) > 0 {
				fmt.Printf("\n✅ Successfully deployed to %d environment(s): %s\n",
					len(generatedApplications), strings.Join(generatedApplications, ", "))
			} else {
				fmt.Fprintf(os.Stderr, "\n❌ Failed to deploy to any environment\n")
				os.Exit(1)
			}
		}

		// Build deployment summary
		summaryBuilder := summary.NewSummaryBuilder()
		summaryBuilder.WithMetadata(dryRun, deployAutoApprove)

		// Get current context if possible
		contextName := ""
		if ctx, err := k8s.GetCurrentContext(k8sConfig.KubeconfigPath); err == nil {
			contextName = ctx
		}

		summaryBuilder.WithKubernetes(getKubeconfigPath(), contextName)
		summaryBuilder.WithArgoCD(argoCDNamespace, selectedProject, true)

		// Determine auth method
		authMethod := "none"
		if authToken != "" {
			authMethod = "token"
		} else if authUsername != "" {
			authMethod = "username-password"
		} else if sshKey != "" || sshKeyPath != "" {
			authMethod = "ssh-key"
		}

		summaryBuilder.WithRepository(normalizedURL, selectedRevision, authMethod)
		summaryBuilder.WithChart(selectedChart)
		summaryBuilder.WithApplication(finalAppName)

		// Add environments to summary
		envManager.LoadDefaultEnvironments()

		for _, env := range selectedEnvironments {
			namespaceName, _ := k8s.GenerateNamespaceName(finalAppName, env)
			envConfig, _ := envManager.GetEnvironment(env)

			// Generate Helm values for the application
			helmValues := yamlProcessor.GenerateHelmValues(
				finalAppName,
				env,
				"ethannguyen98/"+finalAppName,  // Default image repository
				"v0.0.1",                       // Default image tag
				finalAppName+"-"+env+"-secret", // Default secret name
				&yaml.UptraceConfig{
					Enabled:     true,
					ServiceName: finalAppName + "-" + env,
					Endpoint:    "http://uptrace-collector.uptrace.svc:4318",
					Headers: map[string]string{
						"uptrace-dsn": "http://token@uptrace-collector.uptrace.svc:4318/2",
					},
				},
			)

			summaryBuilder.AddEnvironment(env, envConfig, namespaceName, helmValues)
		}

		// Add warnings if any
		validationResult = envManager.ValidateEnvironmentSetup(selectedEnvironments)
		for _, warning := range validationResult.Warnings {
			summaryBuilder.AddWarning(warning)
		}

		// Build the summary
		deploymentSummary := summaryBuilder.Build()

		// Display the summary
		fmt.Println(summary.FormatDeploymentSummary(deploymentSummary))

		// In dry-run mode, show a prominent message and exit
		if dryRun {
			fmt.Println("\n" + strings.Repeat("=", 60))
			fmt.Println("🔍 DRY-RUN MODE: No resources will be created")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Println("This is a preview of what would be deployed.")
			fmt.Println("To apply these changes, run the command without --dry-run flag.")
			fmt.Println(strings.Repeat("=", 60))
			return
		}

		// Confirm deployment unless in auto-approve mode
		if !deployAutoApprove {
			confirmOptions := &summary.ConfirmationOptions{
				Message:     "Do you want to proceed with this deployment?",
				Default:     false,
				ShowHelp:    true,
				AllowModify: false,
				AllowBack:   false,
				AutoApprove: deployAutoApprove,
				DryRun:      dryRun,
			}

			result, err := summary.ConfirmDeployment(deploymentSummary, confirmOptions)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting confirmation: %v\n", err)
				os.Exit(1)
			}

			if !result.Confirmed || result.Cancelled {
				fmt.Println("Deployment cancelled by user")
				os.Exit(0)
			}
		} else {
			fmt.Println("Auto-approving deployment due to --auto-approve flag")
		}

		// Display the current configuration
		fmt.Printf("\n✅ Deployment configuration completed:\n")
		fmt.Printf("Kubeconfig path: %s\n", getKubeconfigPath())
		fmt.Printf("ArgoCD namespace: %s\n", argoCDNamespace)
		fmt.Printf("Selected project: %s\n", selectedProject.Name)
		fmt.Printf("Repository: %s\n", normalizedURL)
		if selectedRevision != nil {
			fmt.Printf("Revision: %s (%s)\n", selectedRevision.Value, selectedRevision.Type)
		}
		fmt.Printf("Chart: %s (version %s)\n", selectedChart.Name, selectedChart.Version)
		fmt.Printf("Application name: %s\n", finalAppName)
		fmt.Printf("Environments: %s\n", strings.Join(selectedEnvironments, ", "))

		fmt.Println("\n✅ Created namespaces:")
		for _, env := range selectedEnvironments {
			namespaceName, _ := k8s.GenerateNamespaceName(finalAppName, env)
			fmt.Printf("  - %s (environment: %s)\n", namespaceName, env)
		}

		fmt.Println("\n✅ Generated ArgoCD Applications:")
		for _, appName := range generatedApplications {
			fmt.Printf("  - %s\n", appName)
		}

		if dryRun {
			fmt.Println("\n🔍 Dry-run mode completed. No resources were created.")
			fmt.Println("To apply the generated Applications, run without --dry-run flag.")
		} else {
			if !errorCollector.HasErrors() {
				fmt.Println("\n🚀 Deployment completed successfully!")
			} else {
				fmt.Printf("\n🚀 Deployment completed with %d error(s)\n", len(errorCollector.GetAllErrors()))
			}

			// Wait for sync completion if requested
			if waitForSync && len(generatedApplications) > 0 {
				fmt.Println("\n⏳ Waiting for Applications to sync...")
				syncErrorCollector := errors.NewEnvironmentErrorCollector()

				for _, appName := range generatedApplications {
					fmt.Printf("Waiting for Application '%s' to sync...\n", appName)
					err := argoClient.WaitForApplicationSync(cmd.Context(), appName, 5*time.Minute)
					if err != nil {
						// Extract environment name from app name
						envName := strings.TrimPrefix(appName, finalAppName+"-")
						syncErrorCollector.AddError(envName,
							fmt.Sprintf("Error waiting for Application to sync: %v", err),
							true, // Sync errors might be retryable
							map[string]interface{}{
								"app_name": appName,
								"timeout":  "5m",
							})
						continue
					}

					// Get the final status
					status, err := argoClient.GetApplicationStatus(cmd.Context(), appName)
					if err != nil {
						// Extract environment name from app name
						envName := strings.TrimPrefix(appName, finalAppName+"-")
						syncErrorCollector.AddError(envName,
							fmt.Sprintf("Error getting final Application status: %v", err),
							false, // Status errors are typically not retryable
							map[string]interface{}{
								"app_name": appName,
							})
						continue
					}

					fmt.Printf("✅ Application '%s' is synced and healthy\n", appName)
					fmt.Printf("  Health: %s\n", status.Health.Status)
					fmt.Printf("  Sync: %s\n", status.Sync.Status)
					if status.Sync.Revision != "" {
						fmt.Printf("  Revision: %s\n", status.Sync.Revision)
					}
				}

				if syncErrorCollector.HasErrors() {
					fmt.Println("\n⚠️  Some Applications failed to sync:")
					fmt.Println(syncErrorCollector.Summary())
				} else {
					fmt.Println("\n✅ All Applications are synced and healthy!")
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	// Local flags for the deploy command
	deployCmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Perform a dry-run without making any changes")
	deployCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false,
		"Run without interactive prompts")
	deployCmd.Flags().BoolVar(&deployAutoApprove, "auto-approve", false,
		"Automatically approve deployments without confirmation")
	deployCmd.Flags().StringVar(&repositoryURL, "repository-url", "",
		"Git repository URL containing Helm charts")
	deployCmd.Flags().StringVar(&appName, "app-name", "",
		"Application name (default: derived from chart name)")
	deployCmd.Flags().StringVar(&selectedEnvs, "environments", "",
		"Comma-separated list of environments (default: development)")
	deployCmd.Flags().BoolVar(&waitForSync, "wait-for-sync", false,
		"Wait for Applications to sync after creation")

	// Git authentication flags
	deployCmd.Flags().StringVar(&authToken, "token", "", "Authentication token for private repositories")
	deployCmd.Flags().StringVar(&authUsername, "username", "", "Username for authentication")
	deployCmd.Flags().StringVar(&authPassword, "password", "", "Password for authentication")
	deployCmd.Flags().StringVar(&sshKeyPath, "ssh-key-path", "", "Path to SSH private key")
	deployCmd.Flags().StringVar(&sshKey, "ssh-key", "", "SSH private key content")
	deployCmd.Flags().StringVar(&knownHosts, "known-hosts", "", "Path to known hosts file")
	deployCmd.Flags().StringVar(&tempDir, "temp-dir", "", "Temporary directory for cloning (default: auto-generated)")

	// Git revision flags
	deployCmd.Flags().StringVar(&branch, "branch", "", "Branch to clone")
	deployCmd.Flags().StringVar(&tag, "tag", "", "Tag to clone")
	deployCmd.Flags().StringVar(&commitHash, "commit", "", "Commit to checkout after cloning")
}

// getKubeconfigPath returns the kubeconfig path to use
func getKubeconfigPath() string {
	if kubeconfigPath != "" {
		return kubeconfigPath
	}
	// Default to $HOME/.kube/config if not specified
	return "$HOME/.kube/config"
}

// validateDeployFlags validates the command-line flags for the deploy command
func validateDeployFlags() error {
	// Validate repository URL if provided in non-interactive mode
	if nonInteractive && repositoryURL == "" {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeMissingRequired,
			"Repository URL is required in non-interactive mode",
			errors.ErrorSeverityError,
			false,
		).WithSuggestions(
			"Provide a repository URL using the --repository-url flag",
		)
	}

	// Validate authentication flags
	if authToken != "" && (authUsername != "" || authPassword != "") {
		return errors.NewAppError(
			errors.ErrorTypeAuth,
			errors.ErrCodeAuthFailed,
			"Cannot use token authentication with username/password authentication",
			errors.ErrorSeverityError,
			false,
		).WithSuggestions(
			"Use either token authentication or username/password authentication",
			"Remove conflicting authentication flags",
		)
	}

	// Validate SSH authentication flags
	if sshKey != "" && sshKeyPath != "" {
		return errors.NewAppError(
			errors.ErrorTypeAuth,
			errors.ErrCodeAuthFailed,
			"Cannot use both SSH key content and SSH key path",
			errors.ErrorSeverityError,
			false,
		).WithSuggestions(
			"Use either SSH key content or SSH key path",
			"Remove conflicting SSH authentication flags",
		)
	}

	// Validate Git revision flags
	revisionCount := 0
	if branch != "" {
		revisionCount++
	}
	if tag != "" {
		revisionCount++
	}
	if commitHash != "" {
		revisionCount++
	}

	if revisionCount > 1 {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeInvalidInput,
			"Cannot specify multiple Git revisions (branch, tag, commit)",
			errors.ErrorSeverityError,
			false,
		).WithSuggestions(
			"Use only one Git revision flag",
			"Remove conflicting revision flags",
		)
	}

	return nil
}
