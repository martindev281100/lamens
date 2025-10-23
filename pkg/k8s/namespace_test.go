package k8s_test

import (
	"context"
	"testing"

	"github.com/your-org/kargo-bootstrap/mocks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespaceExists(t *testing.T) {
	tests := []struct {
		name          string
		namespaceName string
		setupMock     func(*mocks.MockK8sClient)
		expected      bool
		expectedError bool
	}{
		{
			name:          "namespace exists",
			namespaceName: "test-namespace",
			setupMock: func(mock *mocks.MockK8sClient) {
				mock.AddTestNamespace(&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				})
			},
			expected:      true,
			expectedError: false,
		},
		{
			name:          "namespace does not exist",
			namespaceName: "non-existent",
			setupMock:     func(mock *mocks.MockK8sClient) {},
			expected:      false,
			expectedError: false,
		},
		{
			name:          "empty namespace name",
			namespaceName: "",
			setupMock:     func(mock *mocks.MockK8sClient) {},
			expected:      false,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockK8sClient()
			tt.setupMock(mock)

			ctx := context.Background()
			exists, err := mock.NamespaceExists(ctx, tt.namespaceName)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if exists != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, exists)
				}
			}
		})
	}
}

func TestCreateNamespace(t *testing.T) {
	tests := []struct {
		name          string
		namespaceName string
		labels        map[string]string
		setupMock     func(*mocks.MockK8sClient)
		expectedError bool
	}{
		{
			name:          "create namespace successfully",
			namespaceName: "test-namespace",
			labels:        map[string]string{"app": "test"},
			setupMock:     func(mock *mocks.MockK8sClient) {},
			expectedError: false,
		},
		{
			name:          "create namespace that already exists",
			namespaceName: "existing-namespace",
			labels:        map[string]string{"app": "test"},
			setupMock: func(mock *mocks.MockK8sClient) {
				mock.AddTestNamespace(&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-namespace",
					},
				})
			},
			expectedError: false,
		},
		{
			name:          "empty namespace name",
			namespaceName: "",
			labels:        map[string]string{"app": "test"},
			setupMock:     func(mock *mocks.MockK8sClient) {},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockK8sClient()
			tt.setupMock(mock)

			ctx := context.Background()
			err := mock.CreateNamespace(ctx, tt.namespaceName, tt.labels)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify namespace was created
				exists, err := mock.NamespaceExists(ctx, tt.namespaceName)
				if err != nil {
					t.Errorf("failed to verify namespace exists: %v", err)
				}
				if !exists {
					t.Errorf("expected namespace to exist")
				}
			}
		})
	}
}

func TestGetNamespace(t *testing.T) {
	tests := []struct {
		name          string
		namespaceName string
		setupMock     func(*mocks.MockK8sClient)
		expectedError bool
		expectedName  string
	}{
		{
			name:          "get existing namespace",
			namespaceName: "test-namespace",
			setupMock: func(mock *mocks.MockK8sClient) {
				mock.AddTestNamespace(&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{"app": "test"},
					},
				})
			},
			expectedError: false,
			expectedName:  "test-namespace",
		},
		{
			name:          "get non-existent namespace",
			namespaceName: "non-existent",
			setupMock:     func(mock *mocks.MockK8sClient) {},
			expectedError: true,
		},
		{
			name:          "empty namespace name",
			namespaceName: "",
			setupMock:     func(mock *mocks.MockK8sClient) {},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockK8sClient()
			tt.setupMock(mock)

			ctx := context.Background()
			namespace, err := mock.GetNamespace(ctx, tt.namespaceName)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if namespace != nil {
					t.Errorf("expected nil namespace but got %v", namespace)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if namespace == nil {
					t.Errorf("expected namespace but got nil")
				} else if namespace.Name != tt.expectedName {
					t.Errorf("expected name %s, got %s", tt.expectedName, namespace.Name)
				}
			}
		})
	}
}

func TestEnsureNamespace(t *testing.T) {
	tests := []struct {
		name          string
		namespaceName string
		labels        map[string]string
		setupMock     func(*mocks.MockK8sClient)
		expectedError bool
	}{
		{
			name:          "ensure namespace that doesn't exist",
			namespaceName: "new-namespace",
			labels:        map[string]string{"app": "test"},
			setupMock:     func(mock *mocks.MockK8sClient) {},
			expectedError: false,
		},
		{
			name:          "ensure namespace that already exists",
			namespaceName: "existing-namespace",
			labels:        map[string]string{"app": "test"},
			setupMock: func(mock *mocks.MockK8sClient) {
				mock.AddTestNamespace(&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-namespace",
					},
				})
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockK8sClient()
			tt.setupMock(mock)

			ctx := context.Background()
			err := mock.EnsureNamespace(ctx, tt.namespaceName, tt.labels)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify namespace exists
				exists, err := mock.NamespaceExists(ctx, tt.namespaceName)
				if err != nil {
					t.Errorf("failed to verify namespace exists: %v", err)
				}
				if !exists {
					t.Errorf("expected namespace to exist")
				}
			}
		})
	}
}

func TestValidateNamespaceName(t *testing.T) {
	tests := []struct {
		name          string
		namespaceName string
		expectedError bool
	}{
		{
			name:          "valid namespace name",
			namespaceName: "test-namespace",
			expectedError: false,
		},
		{
			name:          "valid namespace name with numbers",
			namespaceName: "test-123",
			expectedError: false,
		},
		{
			name:          "empty namespace name",
			namespaceName: "",
			expectedError: true,
		},
		{
			name:          "namespace name too long",
			namespaceName: "this-is-a-very-long-namespace-name-that-exceeds-the-sixty-three-character-limit",
			expectedError: true,
		},
		{
			name:          "namespace name with uppercase letters",
			namespaceName: "Test-Namespace",
			expectedError: true,
		},
		{
			name:          "namespace name with invalid characters",
			namespaceName: "test_namespace",
			expectedError: true,
		},
		{
			name:          "namespace name starting with hyphen",
			namespaceName: "-test-namespace",
			expectedError: true,
		},
		{
			name:          "namespace name ending with hyphen",
			namespaceName: "test-namespace-",
			expectedError: true,
		},
		{
			name:          "reserved namespace name",
			namespaceName: "kube-system",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockK8sClient()
			err := mock.ValidateNamespaceName(tt.namespaceName)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateNamespaceName(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		environment string
		expected    string
		expectError bool
	}{
		{
			name:        "generate namespace name with valid inputs",
			appName:     "myapp",
			environment: "production",
			expected:    "myapp-production",
			expectError: false,
		},
		{
			name:        "generate namespace name with underscores",
			appName:     "my_app",
			environment: "prod_env",
			expected:    "myapp-prodenv",
			expectError: false,
		},
		{
			name:        "generate namespace name with uppercase",
			appName:     "MyApp",
			environment: "Production",
			expected:    "myapp-production",
			expectError: false,
		},
		{
			name:        "empty app name",
			appName:     "",
			environment: "production",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty environment",
			appName:     "myapp",
			environment: "",
			expected:    "myapp-default",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.NewMockK8sClient()
			result, err := mock.GenerateNamespaceName(tt.appName, tt.environment)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}
