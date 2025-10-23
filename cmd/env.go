package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/kargo-bootstrap/pkg/config"
	"github.com/your-org/kargo-bootstrap/pkg/summary"
)

var (
	envConfigFile  string
	envAutoApprove bool
)

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment configurations",
	Long: `Manage environment configurations for kargo-bootstrap.

This command provides subcommands to list, configure, and validate
environment configurations used for deployments.`,
}

// envListCmd represents the env list command
var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available environments",
	Long: `List all available environment configurations.

This command displays all configured environments with their
properties such as type, namespace, and resource settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize environment manager
		envManager := config.NewEnvironmentManager()

		// Load default environments
		envManager.LoadDefaultEnvironments()

		// Load custom environments from config file if specified
		if envConfigFile != "" {
			if err := envManager.LoadFromFile(envConfigFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error loading environment config file: %v\n", err)
				os.Exit(1)
			}
		}

		// Get all environments
		envs := envManager.ListEnvironments()

		if len(envs) == 0 {
			fmt.Println("No environments configured")
			return
		}

		// Display environment summary
		fmt.Println(envManager.GetEnvironmentSummary())
	},
}

// envConfigCmd represents the env config command
var envConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure environments",
	Long: `Configure environment settings.

This command allows you to add, update, or remove environment
configurations. You can also save configurations to a file.`,
}

// envConfigAddCmd represents the env config add command
var envConfigAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new environment",
	Long: `Add a new environment configuration.

This command creates a new environment with the specified name
and prompts for configuration details.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		// Initialize environment manager
		envManager := config.NewEnvironmentManager()
		envManager.LoadDefaultEnvironments()

		// Load custom environments from config file if specified
		if envConfigFile != "" {
			if err := envManager.LoadFromFile(envConfigFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error loading environment config file: %v\n", err)
				os.Exit(1)
			}
		}

		// Create new environment config
		envConfig := &config.EnvironmentConfig{
			Name:         name,
			Type:         "custom",
			Description:  "Custom environment",
			Namespace:    name,
			AutoSync:     true,
			Prune:        true,
			SelfHeal:     true,
			ReplicaCount: 1,
			Values:       make(map[string]interface{}),
			Labels:       make(map[string]string),
			Annotations:  make(map[string]string),
		}

		// Build environment operation summary
		summaryBuilder := summary.NewSummaryBuilder()
		summaryBuilder.WithMetadata(false, envAutoApprove)

		// Create a simple summary for the environment operation
		envSummary := &summary.EnvOperationSummary{
			Metadata: summary.SummaryMetadata{
				GeneratedAt: time.Now(),
				DryRun:      false,
				AutoApprove: envAutoApprove,
			},
			Operation: summary.EnvOperationAdd,
			Environments: []summary.EnvConfigSummary{
				{
					Name:         envConfig.Name,
					Type:         envConfig.Type,
					Namespace:    envConfig.Namespace,
					Description:  envConfig.Description,
					ReplicaCount: envConfig.ReplicaCount,
					AutoSync:     envConfig.AutoSync,
					Prune:        envConfig.Prune,
					SelfHeal:     envConfig.SelfHeal,
				},
			},
		}

		// Display the summary
		fmt.Println(summary.FormatEnvOperationSummary(envSummary))

		// Confirm operation unless in auto-approve mode
		if !envAutoApprove {
			confirmOptions := &summary.ConfirmationOptions{
				Message:     fmt.Sprintf("Do you want to add environment '%s'?", name),
				Default:     false,
				ShowHelp:    true,
				AllowModify: false,
				AllowBack:   false,
				AutoApprove: envAutoApprove,
				DryRun:      false,
			}

			result, err := summary.ConfirmEnvOperation(envSummary, confirmOptions)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting confirmation: %v\n", err)
				os.Exit(1)
			}

			if !result.Confirmed || result.Cancelled {
				fmt.Println("Environment addition cancelled by user")
				os.Exit(0)
			}
		} else {
			fmt.Println("Auto-approving environment addition due to --auto-approve flag")
		}

		// Add the environment
		if err := envManager.AddEnvironment(envConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding environment: %v\n", err)
			os.Exit(1)
		}

		// Save to config file if specified
		if envConfigFile != "" {
			if err := envManager.SaveToFile(envConfigFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving environment config file: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Printf("✅ Successfully added environment '%s'\n", name)
	},
}

// envConfigRemoveCmd represents the env config remove command
var envConfigRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove an environment",
	Long: `Remove an existing environment configuration.

This command removes the specified environment from the configuration.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		// Initialize environment manager
		envManager := config.NewEnvironmentManager()
		envManager.LoadDefaultEnvironments()

		// Load custom environments from config file if specified
		if envConfigFile != "" {
			if err := envManager.LoadFromFile(envConfigFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error loading environment config file: %v\n", err)
				os.Exit(1)
			}
		}

		// Get the environment config to show in the summary
		envConfig, err := envManager.GetEnvironment(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting environment config: %v\n", err)
			os.Exit(1)
		}

		// Build environment operation summary
		summaryBuilder := summary.NewSummaryBuilder()
		summaryBuilder.WithMetadata(false, envAutoApprove)

		// Create a simple summary for the environment operation
		envSummary := &summary.EnvOperationSummary{
			Metadata: summary.SummaryMetadata{
				GeneratedAt: time.Now(),
				DryRun:      false,
				AutoApprove: envAutoApprove,
			},
			Operation: summary.EnvOperationRemove,
			Environments: []summary.EnvConfigSummary{
				{
					Name:         envConfig.Name,
					Type:         envConfig.Type,
					Namespace:    envConfig.Namespace,
					Description:  envConfig.Description,
					ReplicaCount: envConfig.ReplicaCount,
					AutoSync:     envConfig.AutoSync,
					Prune:        envConfig.Prune,
					SelfHeal:     envConfig.SelfHeal,
				},
			},
		}

		// Display the summary
		fmt.Println(summary.FormatEnvOperationSummary(envSummary))

		// Confirm operation unless in auto-approve mode
		if !envAutoApprove {
			confirmOptions := &summary.ConfirmationOptions{
				Message:     fmt.Sprintf("Do you want to remove environment '%s'?", name),
				Default:     false,
				ShowHelp:    true,
				AllowModify: false,
				AllowBack:   false,
				AutoApprove: envAutoApprove,
				DryRun:      false,
			}

			result, err := summary.ConfirmEnvOperation(envSummary, confirmOptions)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting confirmation: %v\n", err)
				os.Exit(1)
			}

			if !result.Confirmed || result.Cancelled {
				fmt.Println("Environment removal cancelled by user")
				os.Exit(0)
			}
		} else {
			fmt.Println("Auto-approving environment removal due to --auto-approve flag")
		}

		// Remove the environment
		if err := envManager.RemoveEnvironment(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing environment: %v\n", err)
			os.Exit(1)
		}

		// Save to config file if specified
		if envConfigFile != "" {
			if err := envManager.SaveToFile(envConfigFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving environment config file: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Printf("✅ Successfully removed environment '%s'\n", name)
	},
}

// envValidateCmd represents the env validate command
var envValidateCmd = &cobra.Command{
	Use:   "validate [environments...]",
	Short: "Validate environment configurations",
	Long: `Validate environment configurations.

This command validates the specified environments and checks for
potential conflicts or issues.`,
	Run: func(cmd *cobra.Command, args []string) {
		var environments []string

		if len(args) > 0 {
			environments = args
		} else {
			// If no environments specified, validate all
			envManager := config.NewEnvironmentManager()
			envManager.LoadDefaultEnvironments()

			if envConfigFile != "" {
				if err := envManager.LoadFromFile(envConfigFile); err != nil {
					fmt.Fprintf(os.Stderr, "Error loading environment config file: %v\n", err)
					os.Exit(1)
				}
			}

			environments = envManager.ListEnvironmentNames()
		}

		if len(environments) == 0 {
			fmt.Println("No environments to validate")
			return
		}

		// Initialize environment manager
		envManager := config.NewEnvironmentManager()
		envManager.LoadDefaultEnvironments()

		// Load custom environments from config file if specified
		if envConfigFile != "" {
			if err := envManager.LoadFromFile(envConfigFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error loading environment config file: %v\n", err)
				os.Exit(1)
			}
		}

		// Validate environments
		result := envManager.ValidateEnvironmentSetup(environments)

		// Display validation results
		if result.Valid {
			fmt.Printf("✅ Validation successful for %d environment(s): %s\n",
				len(environments), strings.Join(environments, ", "))
		} else {
			fmt.Printf("❌ Validation failed for %d environment(s):\n", len(environments))
			for _, errMsg := range result.Errors {
				fmt.Printf("  - %s\n", errMsg)
			}
		}

		// Display warnings if any
		if len(result.Warnings) > 0 {
			fmt.Printf("\n⚠️  Validation warnings:\n")
			for _, warning := range result.Warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}
	},
}

// envDefaultsCmd represents the env defaults command
var envDefaultsCmd = &cobra.Command{
	Use:   "defaults",
	Short: "Show default environment configurations",
	Long: `Show the default environment configurations.

This command displays the built-in default environment configurations
that are loaded when no custom configuration is provided.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize environment manager
		envManager := config.NewEnvironmentManager()
		envManager.LoadDefaultEnvironments()

		// Display default environments
		fmt.Println("Default Environment Configurations:")
		fmt.Println(envManager.GetEnvironmentSummary())
	},
}

func init() {
	rootCmd.AddCommand(envCmd)

	// Add subcommands
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envConfigCmd)
	envCmd.AddCommand(envValidateCmd)
	envCmd.AddCommand(envDefaultsCmd)

	// Add config subcommands
	envConfigCmd.AddCommand(envConfigAddCmd)
	envConfigCmd.AddCommand(envConfigRemoveCmd)

	// Add flags
	envListCmd.Flags().StringVar(&envConfigFile, "config", "",
		"Path to environment configuration file")

	envConfigCmd.PersistentFlags().StringVar(&envConfigFile, "config", "",
		"Path to environment configuration file")

	envValidateCmd.Flags().StringVar(&envConfigFile, "config", "",
		"Path to environment configuration file")

	// Add auto-approve flag to config commands
	envConfigCmd.PersistentFlags().BoolVar(&envAutoApprove, "auto-approve", false,
		"Automatically approve environment operations without confirmation")
}
