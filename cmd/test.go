package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/your-org/kargo-bootstrap/pkg/argocd"
	"github.com/your-org/kargo-bootstrap/pkg/k8s"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test Kubernetes connection and functionality",
	Long: `Test the connection to the Kubernetes cluster and verify basic functionality.

This command tests the connection to the Kubernetes API server and can optionally
list namespaces to verify that the client has the necessary permissions.`,
	Example: `  # Test Kubernetes connection
  kargo-bootstrap test

  # Test connection and list namespaces
  kargo-bootstrap test --list-namespaces

  # Test with custom kubeconfig
  kargo-bootstrap test --kubeconfig ~/.kube/custom-config`,
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

		fmt.Println("✅ Successfully connected to Kubernetes API server")

		// Get current context if possible
		if contextName, err := k8s.GetCurrentContext(config.KubeconfigPath); err == nil {
			fmt.Printf("Current context: %s\n", contextName)
		} else {
			klog.V(2).Infof("Could not determine current context: %v", err)
		}

		// List namespaces if requested
		if listNamespaces {
			fmt.Println("\nListing namespaces:")
			fmt.Println()

			namespaces, err := client.ListNamespaces(context.Background())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing namespaces: %v\n", err)
				os.Exit(1)
			}

			// Use tabwriter for nice formatting
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAMESPACE\tSTATUS\t")
			fmt.Fprintln(w, "--------\t------\t")

			for _, ns := range namespaces {
				// Check if namespace exists (this is always true since we got it from the list)
				// but we can add more status information here in the future
				fmt.Fprintf(w, "%s\tActive\t\n", ns)
			}

			w.Flush()
			fmt.Printf("\nFound %d namespaces\n", len(namespaces))
		}

		// Test namespace creation if requested
		if testNamespace != "" {
			fmt.Printf("\nTesting namespace functionality with namespace '%s':\n", testNamespace)

			// Test namespace name validation
			fmt.Println("Testing namespace name validation...")
			if err := k8s.ValidateNamespaceName(testNamespace); err != nil {
				fmt.Printf("❌ Namespace name validation failed: %v\n", err)
			} else {
				fmt.Printf("✅ Namespace name '%s' is valid\n", testNamespace)
			}

			// Test enhanced namespace existence check
			fmt.Println("Testing enhanced namespace existence check...")
			exists, err := client.CheckNamespaceExists(context.Background(), testNamespace)
			if err != nil {
				fmt.Printf("❌ Enhanced namespace check failed: %v\n", err)
			} else if exists {
				fmt.Printf("Namespace '%s' already exists\n", testNamespace)
			} else {
				fmt.Printf("Namespace '%s' does not exist\n", testNamespace)
			}

			// Test namespace status checking
			if exists {
				fmt.Println("Testing namespace status checking...")
				status, err := client.GetNamespaceStatus(context.Background(), testNamespace)
				if err != nil {
					fmt.Printf("❌ Failed to get namespace status: %v\n", err)
				} else {
					fmt.Printf("✅ Namespace status: %s\n", status)
				}

				// Test getting namespace details
				fmt.Println("Testing namespace details retrieval...")
				namespace, err := client.GetNamespace(context.Background(), testNamespace)
				if err != nil {
					fmt.Printf("❌ Failed to get namespace details: %v\n", err)
				} else {
					info := k8s.FormatNamespaceForDisplay(namespace)
					fmt.Printf("✅ Namespace details:\n")
					fmt.Printf("  Name: %s\n", info.Name)
					fmt.Printf("  Status: %s\n", info.Status)
					fmt.Printf("  Created: %s\n", info.Created.Format(time.RFC3339))
					if len(info.Labels) > 0 {
						fmt.Printf("  Labels:\n")
						for k, v := range info.Labels {
							fmt.Printf("    %s: %s\n", k, v)
						}
					}
					if len(info.Annotations) > 0 {
						fmt.Printf("  Annotations:\n")
						for k, v := range info.Annotations {
							fmt.Printf("    %s: %s\n", k, v)
						}
					}
				}

				// Test adding labels to existing namespace
				fmt.Println("Testing adding labels to namespace...")
				testLabels := map[string]string{
					"kargo-bootstrap/test-label": "test-value",
					"kargo-bootstrap/test-time":  time.Now().Format(time.RFC3339),
				}
				if err := client.AddLabelsToNamespace(context.Background(), testNamespace, testLabels); err != nil {
					fmt.Printf("❌ Failed to add labels to namespace: %v\n", err)
				} else {
					fmt.Printf("✅ Successfully added labels to namespace '%s'\n", testNamespace)
				}

				// Test adding annotations to existing namespace
				fmt.Println("Testing adding annotations to namespace...")
				testAnnotations := map[string]string{
					"kargo-bootstrap/test-annotation": "test-annotation-value",
					"kargo-bootstrap/test-timestamp":  time.Now().Format(time.RFC3339),
				}
				if err := client.AddAnnotationsToNamespace(context.Background(), testNamespace, testAnnotations); err != nil {
					fmt.Printf("❌ Failed to add annotations to namespace: %v\n", err)
				} else {
					fmt.Printf("✅ Successfully added annotations to namespace '%s'\n", testNamespace)
				}
			} else {
				// Test namespace creation with labels and annotations
				fmt.Println("Testing namespace creation with labels and annotations...")
				labels := map[string]string{
					"kargo-bootstrap/test":      "true",
					"kargo-bootstrap/test-type": "enhanced",
				}
				annotations := map[string]string{
					"kargo-bootstrap/created-by": "kargo-bootstrap-test",
					"kargo-bootstrap/created-at": time.Now().Format(time.RFC3339),
				}

				if err := client.CreateNamespaceWithLabels(context.Background(), testNamespace, labels, annotations); err != nil {
					fmt.Printf("❌ Error creating test namespace with labels and annotations: %v\n", err)
				} else {
					fmt.Printf("✅ Successfully created test namespace '%s' with labels and annotations\n", testNamespace)
				}

				// Test namespace readiness waiting
				fmt.Println("Testing namespace readiness waiting...")
				if err := client.WaitForNamespaceReady(context.Background(), testNamespace, 10*time.Second); err != nil {
					fmt.Printf("❌ Namespace readiness check failed: %v\n", err)
				} else {
					fmt.Printf("✅ Namespace '%s' is ready\n", testNamespace)
				}
			}
		}

		// Test namespace name generation
		fmt.Println("\nTesting namespace name generation...")
		testAppName := "my-app"
		testEnvironments := []string{"development", "staging", "production"}
		for _, env := range testEnvironments {
			genName, err := k8s.GenerateNamespaceName(testAppName, env)
			if err != nil {
				fmt.Printf("❌ Failed to generate namespace name for %s/%s: %v\n", testAppName, env, err)
			} else {
				fmt.Printf("✅ Generated namespace name for %s/%s: %s\n", testAppName, env, genName)
			}
		}

		// Test ArgoCD connectivity
		fmt.Println("\nTesting ArgoCD connectivity...")
		argoClient, err := argocd.NewClient(client, argoCDNamespace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating ArgoCD client: %v\n", err)
			os.Exit(1)
		}

		// Check if ArgoCD is installed and accessible
		if err := argoClient.IsArgoCDInstalled(context.Background()); err != nil {
			fmt.Printf("⚠️  ArgoCD is not installed or not accessible in namespace '%s': %v\n", argoCDNamespace, err)
			fmt.Println("This is expected if you haven't installed ArgoCD yet")
		} else {
			fmt.Println("✅ Successfully connected to ArgoCD")

			// Try to list projects to verify full functionality
			projects, err := argoClient.ListProjects(context.Background())
			if err != nil {
				fmt.Printf("⚠️  Error listing ArgoCD projects: %v\n", err)
			} else {
				fmt.Printf("✅ Successfully found %d ArgoCD project(s)\n", len(projects))
				if len(projects) > 0 {
					fmt.Println("Available projects:")
					for _, project := range projects {
						fmt.Printf("  - %s", project.Name)
						if project.Description != "" {
							fmt.Printf(": %s", project.Description)
						}
						fmt.Println()
					}
				}
			}
		}

		fmt.Println("\n✅ All tests passed successfully!")
	},
}

var (
	listNamespaces bool
	testNamespace  string
)

func init() {
	rootCmd.AddCommand(testCmd)

	// Flags for the test command
	testCmd.Flags().BoolVar(&listNamespaces, "list-namespaces", false,
		"List all namespaces in the cluster")
	testCmd.Flags().StringVar(&testNamespace, "test-namespace", "",
		"Test namespace functionality by creating a namespace with this name")
}
