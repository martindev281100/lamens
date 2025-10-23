package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/your-org/kargo-bootstrap/pkg/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
)

// Config holds the configuration for the Kubernetes client
type Config struct {
	KubeconfigPath string
}

// Client holds the Kubernetes clients
type Client struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
	config        *rest.Config
}

// NewClient creates a new Kubernetes client with the given configuration
func NewClient(cfg *Config) (*Client, error) {
	var restConfig *rest.Config
	var err error

	// Try to load kubeconfig from the provided path, default location, or in-cluster config
	if cfg.KubeconfigPath != "" {
		// Use the provided kubeconfig path
		klog.V(2).Infof("Using kubeconfig from path: %s", cfg.KubeconfigPath)
		restConfig, err = clientcmd.BuildConfigFromFlags("", cfg.KubeconfigPath)
		if err != nil {
			return nil, errors.NewAppError(
				errors.ErrorTypeKubernetes,
				errors.ErrCodeK8sConfig,
				fmt.Sprintf("failed to build config from kubeconfig path %s", cfg.KubeconfigPath),
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithField("kubeconfig", cfg.KubeconfigPath).WithSuggestions(
				"Check if the kubeconfig file exists",
				"Verify the kubeconfig file is valid YAML",
				"Check file permissions",
			)
		}
	} else {
		// Try in-cluster config first
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig location
			klog.V(2).Info("In-cluster config failed, trying default kubeconfig location")
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, errors.NewAppError(
					errors.ErrorTypeConfig,
					errors.ErrCodeInvalidInput,
					"failed to get user home directory",
					errors.ErrorSeverityError,
					false,
				).WithCause(err).WithSuggestions(
					"Check if HOME environment variable is set",
					"Verify user directory permissions",
				)
			}
			defaultKubeconfig := filepath.Join(homeDir, ".kube", "config")

			// Check if the default kubeconfig exists
			if _, err := os.Stat(defaultKubeconfig); os.IsNotExist(err) {
				return nil, errors.NewAppError(
					errors.ErrorTypeKubernetes,
					errors.ErrCodeK8sConfig,
					fmt.Sprintf("no kubeconfig found at %s and not running in a cluster", defaultKubeconfig),
					errors.ErrorSeverityError,
					false,
				).WithField("kubeconfig", defaultKubeconfig).WithSuggestions(
					"Create a kubeconfig file at the default location",
					"Use --kubeconfig flag to specify a custom path",
					"Ensure running inside a Kubernetes cluster with service account",
				)
			}

			restConfig, err = clientcmd.BuildConfigFromFlags("", defaultKubeconfig)
			if err != nil {
				return nil, errors.NewAppError(
					errors.ErrorTypeKubernetes,
					errors.ErrCodeK8sConfig,
					fmt.Sprintf("failed to build config from default kubeconfig %s", defaultKubeconfig),
					errors.ErrorSeverityError,
					false,
				).WithCause(err).WithField("kubeconfig", defaultKubeconfig).WithSuggestions(
					"Check if the kubeconfig file is valid",
					"Verify Kubernetes cluster is accessible",
					"Check authentication credentials",
				)
			}
		} else {
			klog.V(2).Info("Using in-cluster configuration")
		}
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeKubernetes,
			errors.ErrCodeK8sConnection,
			"failed to create clientset",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the Kubernetes configuration is valid",
			"Verify network connectivity to the cluster",
			"Check authentication credentials",
		)
	}

	// Create the dynamic client
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.NewAppError(
			errors.ErrorTypeKubernetes,
			errors.ErrCodeK8sConnection,
			"failed to create dynamic client",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithSuggestions(
			"Check if the Kubernetes configuration is valid",
			"Verify network connectivity to the cluster",
			"Check authentication credentials",
		)
	}

	return &Client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		config:        restConfig,
	}, nil
}

// GetClientset returns the Kubernetes clientset
func (c *Client) GetClientset() kubernetes.Interface {
	return c.clientset
}

// GetDynamicClient returns the Kubernetes dynamic client
func (c *Client) GetDynamicClient() dynamic.Interface {
	return c.dynamicClient
}

// GetConfig returns the REST config
func (c *Client) GetConfig() *rest.Config {
	return c.config
}

// GetCurrentContext returns the current context from the kubeconfig
func GetCurrentContext(kubeconfigPath string) (string, error) {
	var config *api.Config
	var err error

	if kubeconfigPath != "" {
		config, err = clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return "", errors.NewAppError(
				errors.ErrorTypeKubernetes,
				errors.ErrCodeK8sConfig,
				fmt.Sprintf("failed to load kubeconfig from %s", kubeconfigPath),
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithField("kubeconfig", kubeconfigPath).WithSuggestions(
				"Check if the kubeconfig file exists",
				"Verify the kubeconfig file is valid YAML",
				"Check file permissions",
			)
		}
	} else {
		// Try to get the default kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		config, err = loadingRules.Load()
		if err != nil {
			return "", errors.NewAppError(
				errors.ErrorTypeKubernetes,
				errors.ErrCodeK8sConfig,
				"failed to load default kubeconfig",
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithSuggestions(
				"Check if the default kubeconfig file exists",
				"Verify the kubeconfig file is valid",
				"Use --kubeconfig flag to specify a custom path",
			)
		}
	}

	return config.CurrentContext, nil
}

// TestConnection tests the connection to the Kubernetes API server
func (c *Client) TestConnection() error {
	// Try to access the Kubernetes API server version
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeKubernetes,
			errors.ErrCodeK8sConnection,
			"failed to connect to Kubernetes API server",
			errors.ErrorSeverityError,
			true, // Connection issues are often retryable
		).WithCause(err).WithSuggestions(
			"Check if the Kubernetes cluster is accessible",
			"Verify network connectivity",
			"Check authentication credentials",
			"Ensure the cluster is running",
		)
	}

	klog.V(1).Info("Successfully connected to Kubernetes API server")
	return nil
}
