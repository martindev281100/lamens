package k8s

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/your-org/kargo-bootstrap/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// NamespaceExists checks if a namespace exists in the cluster
func (c *Client) NamespaceExists(ctx context.Context, namespaceName string) (bool, error) {
	if namespaceName == "" {
		return false, errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeMissingRequired,
			"namespace name cannot be empty",
			errors.ErrorSeverityError,
			false,
		).WithField("namespace", namespaceName)
	}

	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(2).Infof("Namespace %s does not exist", namespaceName)
			return false, nil
		}
		return false, errors.NewAppError(
			errors.ErrorTypeKubernetes,
			errors.ErrCodeNamespaceNotFound,
			fmt.Sprintf("failed to check if namespace %s exists", namespaceName),
			errors.ErrorSeverityError,
			true, // Network issues might be retryable
		).WithCause(err).WithResource(namespaceName).WithNamespace(namespaceName)
	}

	klog.V(2).Infof("Namespace %s exists", namespaceName)
	return true, nil
}

// CheckNamespaceExists is an enhanced version with better error handling
func (c *Client) CheckNamespaceExists(ctx context.Context, namespaceName string) (bool, error) {
	if err := ValidateNamespaceName(namespaceName); err != nil {
		return false, errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeInvalidFormat,
			"invalid namespace name",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("namespace", namespaceName)
	}

	return c.NamespaceExists(ctx, namespaceName)
}

// CreateNamespace creates a namespace if it doesn't exist
func (c *Client) CreateNamespace(ctx context.Context, namespaceName string, labels map[string]string) error {
	if namespaceName == "" {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeMissingRequired,
			"namespace name cannot be empty",
			errors.ErrorSeverityError,
			false,
		).WithField("namespace", namespaceName)
	}

	// Check if namespace already exists
	exists, err := c.NamespaceExists(ctx, namespaceName)
	if err != nil {
		return err
	}

	if exists {
		klog.V(1).Infof("Namespace %s already exists, skipping creation", namespaceName)
		return nil
	}

	// Create the namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespaceName,
			Labels: labels,
		},
	}

	_, err = c.clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return errors.NewAppError(
				errors.ErrorTypeNamespace,
				errors.ErrCodeResourceExists,
				fmt.Sprintf("namespace %s already exists", namespaceName),
				errors.ErrorSeverityWarning,
				false,
			).WithCause(err).WithResource(namespaceName).WithNamespace(namespaceName)
		}
		if apierrors.IsForbidden(err) {
			return errors.NewAppError(
				errors.ErrorTypePermission,
				errors.ErrCodePermissionDenied,
				fmt.Sprintf("permission denied: cannot create namespace %s", namespaceName),
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithResource(namespaceName).WithNamespace(namespaceName).WithSuggestions(
				"Check RBAC permissions",
				"Ensure you have namespace creation privileges",
				"Contact cluster administrator",
			)
		}
		return errors.NewAppError(
			errors.ErrorTypeNamespace,
			errors.ErrCodeResourceNotFound,
			fmt.Sprintf("failed to create namespace %s", namespaceName),
			errors.ErrorSeverityError,
			true, // Some errors might be retryable
		).WithCause(err).WithResource(namespaceName).WithNamespace(namespaceName)
	}

	klog.Infof("Successfully created namespace %s", namespaceName)
	return nil
}

// CreateNamespaceWithLabels creates a namespace with specific labels and annotations
func (c *Client) CreateNamespaceWithLabels(ctx context.Context, namespaceName string, labels, annotations map[string]string) error {
	if err := ValidateNamespaceName(namespaceName); err != nil {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeInvalidFormat,
			"invalid namespace name",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("namespace", namespaceName)
	}

	// Check if namespace already exists
	exists, err := c.NamespaceExists(ctx, namespaceName)
	if err != nil {
		return errors.NewAppError(
			errors.ErrorTypeNamespace,
			errors.ErrCodeNamespaceNotFound,
			"failed to check if namespace exists",
			errors.ErrorSeverityError,
			true,
		).WithCause(err).WithNamespace(namespaceName)
	}

	if exists {
		klog.V(1).Infof("Namespace %s already exists, skipping creation", namespaceName)
		return errors.NewAppError(
			errors.ErrorTypeNamespace,
			errors.ErrCodeResourceExists,
			fmt.Sprintf("namespace %s already exists", namespaceName),
			errors.ErrorSeverityWarning,
			false,
		).WithResource(namespaceName).WithNamespace(namespaceName)
	}

	// Create the namespace with labels and annotations
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        namespaceName,
			Labels:      labels,
			Annotations: annotations,
		},
	}

	_, err = c.clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return errors.NewAppError(
				errors.ErrorTypeNamespace,
				errors.ErrCodeResourceExists,
				fmt.Sprintf("namespace %s already exists", namespaceName),
				errors.ErrorSeverityWarning,
				false,
			).WithCause(err).WithResource(namespaceName).WithNamespace(namespaceName)
		}
		if apierrors.IsForbidden(err) {
			return errors.NewAppError(
				errors.ErrorTypePermission,
				errors.ErrCodePermissionDenied,
				fmt.Sprintf("permission denied: cannot create namespace %s", namespaceName),
				errors.ErrorSeverityError,
				false,
			).WithCause(err).WithResource(namespaceName).WithNamespace(namespaceName).WithSuggestions(
				"Check RBAC permissions",
				"Ensure you have namespace creation privileges",
				"Contact cluster administrator",
			)
		}
		return errors.NewAppError(
			errors.ErrorTypeNamespace,
			errors.ErrCodeResourceNotFound,
			fmt.Sprintf("failed to create namespace %s", namespaceName),
			errors.ErrorSeverityError,
			true,
		).WithCause(err).WithResource(namespaceName).WithNamespace(namespaceName)
	}

	klog.Infof("Successfully created namespace %s with labels and annotations", namespaceName)
	return nil
}

// EnsureNamespace ensures a namespace exists, creating it if necessary
func (c *Client) EnsureNamespace(ctx context.Context, namespaceName string, labels map[string]string) error {
	if namespaceName == "" {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeMissingRequired,
			"namespace name cannot be empty",
			errors.ErrorSeverityError,
			false,
		).WithField("namespace", namespaceName)
	}

	// Check if namespace already exists
	exists, err := c.NamespaceExists(ctx, namespaceName)
	if err != nil {
		return err
	}

	if exists {
		klog.V(1).Infof("Namespace %s already exists", namespaceName)
		return nil
	}

	// Create the namespace
	return c.CreateNamespace(ctx, namespaceName, labels)
}

// GetNamespaceStatus checks the status of a namespace
func (c *Client) GetNamespaceStatus(ctx context.Context, namespaceName string) (corev1.NamespacePhase, error) {
	if err := ValidateNamespaceName(namespaceName); err != nil {
		return "", errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeInvalidFormat,
			"invalid namespace name",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("namespace", namespaceName)
	}

	namespace, err := c.GetNamespace(ctx, namespaceName)
	if err != nil {
		return "", errors.NewAppError(
			errors.ErrorTypeNamespace,
			errors.ErrCodeNamespaceNotFound,
			"failed to get namespace status",
			errors.ErrorSeverityError,
			true,
		).WithCause(err).WithNamespace(namespaceName)
	}

	return namespace.Status.Phase, nil
}

// ValidateNamespaceName validates namespace names against Kubernetes requirements
func ValidateNamespaceName(name string) error {
	if name == "" {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeMissingRequired,
			"namespace name cannot be empty",
			errors.ErrorSeverityError,
			false,
		).WithField("namespace", name)
	}

	// Kubernetes namespace name requirements:
	// - Max 63 characters
	// - Lowercase letters, numbers, and hyphens
	// - Must start and end with a letter or number
	// - Cannot be formatted as IP address
	if len(name) > 63 {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeInvalidFormat,
			"namespace name cannot be longer than 63 characters",
			errors.ErrorSeverityError,
			false,
		).WithField("namespace", name).WithSuggestions(
			"Use a shorter namespace name",
			"Consider abbreviating the name",
		)
	}

	// Check for valid pattern
	validPattern := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validPattern.MatchString(name) {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeInvalidFormat,
			"namespace name must consist of lowercase letters, numbers, and hyphens, and must start and end with a letter or number",
			errors.ErrorSeverityError,
			false,
		).WithField("namespace", name).WithSuggestions(
			"Use only lowercase letters, numbers, and hyphens",
			"Start and end with a letter or number",
		)
	}

	// Check if it looks like an IP address
	ipPattern := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	if ipPattern.MatchString(name) {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeInvalidFormat,
			"namespace name cannot be formatted as an IP address",
			errors.ErrorSeverityError,
			false,
		).WithField("namespace", name).WithSuggestions(
			"Use a descriptive name instead of an IP address",
		)
	}

	// Reserved namespaces that shouldn't be used
	reservedNamespaces := map[string]bool{
		"default":         true,
		"kube-system":     true,
		"kube-public":     true,
		"kube-node-lease": true,
	}

	if reservedNamespaces[name] {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeInvalidFormat,
			fmt.Sprintf("namespace '%s' is a reserved system namespace", name),
			errors.ErrorSeverityError,
			false,
		).WithField("namespace", name).WithSuggestions(
			"Use a different namespace name",
			"Consider adding a prefix like 'app-' or 'team-'",
		)
	}

	return nil
}

// AddLabelsToNamespace adds or updates labels on an existing namespace
func (c *Client) AddLabelsToNamespace(ctx context.Context, namespaceName string, labels map[string]string) error {
	if err := ValidateNamespaceName(namespaceName); err != nil {
		return errors.NewAppError(
			errors.ErrorTypeValidation,
			errors.ErrCodeInvalidFormat,
			"invalid namespace name",
			errors.ErrorSeverityError,
			false,
		).WithCause(err).WithField("namespace", namespaceName)
	}

	if len(labels) == 0 {
		return fmt.Errorf("no labels provided")
	}

	// Get the current namespace
	namespace, err := c.GetNamespace(ctx, namespaceName)
	if err != nil {
		return fmt.Errorf("failed to get namespace: %w", err)
	}

	// Update labels
	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}

	for key, value := range labels {
		namespace.Labels[key] = value
	}

	// Update the namespace
	_, err = c.clientset.CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update namespace labels: %w", err)
	}

	klog.V(2).Infof("Successfully added labels to namespace %s", namespaceName)
	return nil
}

// AddAnnotationsToNamespace adds or updates annotations on an existing namespace
func (c *Client) AddAnnotationsToNamespace(ctx context.Context, namespaceName string, annotations map[string]string) error {
	if err := ValidateNamespaceName(namespaceName); err != nil {
		return fmt.Errorf("invalid namespace name: %w", err)
	}

	if len(annotations) == 0 {
		return fmt.Errorf("no annotations provided")
	}

	// Get the current namespace
	namespace, err := c.GetNamespace(ctx, namespaceName)
	if err != nil {
		return fmt.Errorf("failed to get namespace: %w", err)
	}

	// Update annotations
	if namespace.Annotations == nil {
		namespace.Annotations = make(map[string]string)
	}

	for key, value := range annotations {
		namespace.Annotations[key] = value
	}

	// Update the namespace
	_, err = c.clientset.CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update namespace annotations: %w", err)
	}

	klog.V(2).Infof("Successfully added annotations to namespace %s", namespaceName)
	return nil
}

// ListNamespaces lists all namespaces in the cluster
func (c *Client) ListNamespaces(ctx context.Context) ([]string, error) {
	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var namespaceNames []string
	for _, ns := range namespaces.Items {
		namespaceNames = append(namespaceNames, ns.Name)
	}

	return namespaceNames, nil
}

// GetNamespace gets a namespace by name
func (c *Client) GetNamespace(ctx context.Context, namespaceName string) (*corev1.Namespace, error) {
	if namespaceName == "" {
		return nil, fmt.Errorf("namespace name cannot be empty")
	}

	namespace, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %s: %w", namespaceName, err)
	}

	return namespace, nil
}

// GenerateNamespaceName generates namespace names based on app and environment
func GenerateNamespaceName(appName, environment string) (string, error) {
	if appName == "" {
		return "", fmt.Errorf("application name cannot be empty")
	}

	if environment == "" {
		environment = "default"
	}

	// Clean and normalize inputs
	cleanAppName := strings.ToLower(strings.ReplaceAll(appName, "_", "-"))
	cleanAppName = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(cleanAppName, "")

	cleanEnv := strings.ToLower(strings.ReplaceAll(environment, "_", "-"))
	cleanEnv = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(cleanEnv, "")

	// Generate namespace name
	namespaceName := fmt.Sprintf("%s-%s", cleanAppName, cleanEnv)

	// Ensure it's not too long
	if len(namespaceName) > 63 {
		// Truncate the app name part, keep the environment
		maxAppLen := 63 - len(cleanEnv) - 1 // -1 for the hyphen
		if maxAppLen < 1 {
			return "", fmt.Errorf("environment name too long to generate a valid namespace")
		}
		namespaceName = fmt.Sprintf("%s-%s", cleanAppName[:maxAppLen], cleanEnv)
	}

	return namespaceName, nil
}

// NamespaceInfo contains formatted information about a namespace
type NamespaceInfo struct {
	Name        string
	Status      corev1.NamespacePhase
	Labels      map[string]string
	Annotations map[string]string
	Created     time.Time
}

// FormatNamespaceForDisplay formats namespace information for display
func FormatNamespaceForDisplay(namespace *corev1.Namespace) NamespaceInfo {
	info := NamespaceInfo{
		Name:    namespace.Name,
		Status:  namespace.Status.Phase,
		Created: namespace.CreationTimestamp.Time,
	}

	if namespace.Labels != nil {
		info.Labels = make(map[string]string)
		for k, v := range namespace.Labels {
			info.Labels[k] = v
		}
	}

	if namespace.Annotations != nil {
		info.Annotations = make(map[string]string)
		for k, v := range namespace.Annotations {
			info.Annotations[k] = v
		}
	}

	return info
}

// WaitForNamespaceReady waits for a namespace to be ready (Active)
func (c *Client) WaitForNamespaceReady(ctx context.Context, namespaceName string, timeout time.Duration) error {
	if err := ValidateNamespaceName(namespaceName); err != nil {
		return fmt.Errorf("invalid namespace name: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	klog.V(2).Infof("Waiting for namespace %s to be ready", namespaceName)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for namespace %s to become ready", namespaceName)
		default:
			status, err := c.GetNamespaceStatus(ctx, namespaceName)
			if err != nil {
				return fmt.Errorf("failed to get namespace status: %w", err)
			}

			if status == corev1.NamespaceActive {
				klog.V(2).Infof("Namespace %s is ready", namespaceName)
				return nil
			}

			// Wait before checking again
			time.Sleep(1 * time.Second)
		}
	}
}
