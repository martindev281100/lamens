package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

// Config holds the configuration for connecting to a Kubernetes cluster.
type Config struct {
	// KubeconfigPath is the path to the kubeconfig file.
	// If empty, defaults to $HOME/.kube/config.
	KubeconfigPath string
}

// Client provides methods for interacting with a Kubernetes cluster.
type Client struct {
	// clientset is the Kubernetes clientset.
	clientset *kubernetes.Clientset
	// config is the Kubernetes configuration.
	config *rest.Config
}

// NewClient creates a new Kubernetes client with the provided configuration.
//
// Parameters:
//   - config: The Kubernetes configuration.
//
// Returns:
//   - *Client: A new Kubernetes client.
//   - error: An error if the client could not be created.
func NewClient(config *Config) (*Client, error) {
	var kubeconfig string
	if config.KubeconfigPath != "" {
		kubeconfig = config.KubeconfigPath
	} else {
		// Default to $HOME/.kube/config
		if home := homeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// Use the current context in kubeconfig
	var restConfig *rest.Config
	var err error

	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		// Kubeconfig doesn't exist, try in-cluster config
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		// Use kubeconfig
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig %s: %w", kubeconfig, err)
		}
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    restConfig,
	}, nil
}

// TestConnection tests the connection to the Kubernetes cluster.
//
// Returns:
//   - error: An error if the connection test fails.
func (c *Client) TestConnection() error {
	// Test the connection by getting the server version
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to get server version: %w", err)
	}

	return nil
}

// EnsureNamespace ensures a namespace exists, creating it if necessary.
//
// Parameters:
//   - ctx: The context for the operation.
//   - namespace: The name of the namespace to ensure.
//   - labels: Labels to apply to the namespace.
//
// Returns:
//   - error: An error if the namespace could not be ensured.
func (c *Client) EnsureNamespace(ctx context.Context, namespace string, labels map[string]string) error {
	// Check if namespace already exists
	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		// Namespace already exists
		klog.V(2).Infof("Namespace %s already exists", namespace)
		return nil
	}

	if !errors.IsNotFound(err) {
		// Some other error occurred
		return fmt.Errorf("failed to get namespace %s: %w", namespace, err)
	}

	// Create the namespace
	ns := &metav1.ObjectMeta{
		Name:   namespace,
		Labels: labels,
	}

	_, err = c.clientset.CoreV1().Namespaces().Create(ctx, &metav1.Namespace{
		ObjectMeta: *ns,
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}

	klog.V(2).Infof("Created namespace %s", namespace)
	return nil
}

// CreateNamespaceWithLabels creates a namespace with the specified labels and annotations.
//
// Parameters:
//   - ctx: The context for the operation.
//   - namespace: The name of the namespace to create.
//   - labels: Labels to apply to the namespace.
//   - annotations: Annotations to apply to the namespace.
//
// Returns:
//   - error: An error if the namespace could not be created.
func (c *Client) CreateNamespaceWithLabels(ctx context.Context, namespace string, labels map[string]string, annotations map[string]string) error {
	ns := &metav1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}

	_, err := c.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}

	klog.V(2).Infof("Created namespace %s with labels and annotations", namespace)
	return nil
}

// AddLabelsToNamespace adds labels to an existing namespace.
//
// Parameters:
//   - ctx: The context for the operation.
//   - namespace: The name of the namespace.
//   - labels: Labels to add to the namespace.
//
// Returns:
//   - error: An error if the labels could not be added.
func (c *Client) AddLabelsToNamespace(ctx context.Context, namespace string, labels map[string]string) error {
	// Get the current namespace
	ns, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", namespace, err)
	}

	// Add new labels
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	for k, v := range labels {
		ns.Labels[k] = v
	}

	// Update the namespace
	_, err = c.clientset.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update namespace %s: %w", namespace, err)
	}

	klog.V(2).Infof("Added labels to namespace %s", namespace)
	return nil
}

// AddAnnotationsToNamespace adds annotations to an existing namespace.
//
// Parameters:
//   - ctx: The context for the operation.
//   - namespace: The name of the namespace.
//   - annotations: Annotations to add to the namespace.
//
// Returns:
//   - error: An error if the annotations could not be added.
func (c *Client) AddAnnotationsToNamespace(ctx context.Context, namespace string, annotations map[string]string) error {
	// Get the current namespace
	ns, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", namespace, err)
	}

	// Add new annotations
	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	for k, v := range annotations {
		ns.Annotations[k] = v
	}

	// Update the namespace
	_, err = c.clientset.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update namespace %s: %w", namespace, err)
	}

	klog.V(2).Infof("Added annotations to namespace %s", namespace)
	return nil
}

// WaitForNamespaceReady waits for a namespace to be ready.
//
// Parameters:
//   - ctx: The context for the operation.
//   - namespace: The name of the namespace.
//   - timeout: The timeout for the operation.
//
// Returns:
//   - error: An error if the namespace is not ready within the timeout.
func (c *Client) WaitForNamespaceReady(ctx context.Context, namespace string, timeout time.Duration) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Poll for namespace to be ready
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for namespace %s to be ready", namespace)
		case <-ticker.C:
			// Check if namespace exists
			_, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			if err == nil {
				// Namespace exists and is ready
				klog.V(2).Infof("Namespace %s is ready", namespace)
				return nil
			}

			if !errors.IsNotFound(err) {
				// Some other error occurred
				return fmt.Errorf("failed to get namespace %s: %w", namespace, err)
			}

			// Namespace doesn't exist yet, continue polling
			klog.V(2).Infof("Waiting for namespace %s to be created...", namespace)
		}
	}
}

// GenerateNamespaceName generates a namespace name based on the application name and environment.
//
// Parameters:
//   - appName: The name of the application.
//   - environment: The environment name.
//
// Returns:
//   - string: The generated namespace name.
//   - error: An error if the namespace name could not be generated.
func GenerateNamespaceName(appName, environment string) (string, error) {
	if appName == "" {
		return "", fmt.Errorf("application name cannot be empty")
	}
	if environment == "" {
		return "", fmt.Errorf("environment cannot be empty")
	}

	// Sanitize the application name and environment
	sanitizedAppName := strings.ToLower(strings.ReplaceAll(appName, "_", "-"))
	sanitizedEnv := strings.ToLower(strings.ReplaceAll(environment, "_", "-"))

	// Generate the namespace name
	namespaceName := fmt.Sprintf("%s-%s", sanitizedAppName, sanitizedEnv)

	// Validate the namespace name
	if err := validateNamespaceName(namespaceName); err != nil {
		return "", fmt.Errorf("invalid namespace name %s: %w", namespaceName, err)
	}

	return namespaceName, nil
}

// validateNamespaceName validates a namespace name according to Kubernetes requirements.
//
// Parameters:
//   - name: The namespace name to validate.
//
// Returns:
//   - error: An error if the namespace name is invalid.
func validateNamespaceName(name string) error {
	if len(name) == 0 || len(name) > 63 {
		return fmt.Errorf("namespace name must be between 1 and 63 characters")
	}

	// Check for invalid characters
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("namespace name can only contain lowercase alphanumeric characters and hyphens")
		}
	}

	// Check if it starts or ends with a hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("namespace name cannot start or end with a hyphen")
	}

	return nil
}

// GetCurrentContext gets the current context from the kubeconfig file.
//
// Parameters:
//   - kubeconfigPath: The path to the kubeconfig file.
//
// Returns:
//   - string: The current context name.
//   - error: An error if the current context could not be retrieved.
func GetCurrentContext(kubeconfigPath string) (string, error) {
	if kubeconfigPath == "" {
		if home := homeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	// Load the kubeconfig
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
	}

	// Get the current context
	currentContext := config.CurrentContext
	if currentContext == "" {
		return "", fmt.Errorf("no current context set in kubeconfig")
	}

	return currentContext, nil
}

// homeDir gets the user's home directory.
//
// Returns:
//   - string: The user's home directory.
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}
