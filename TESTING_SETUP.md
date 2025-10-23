# Testing Setup for kargo-bootstrap

This document describes the testing infrastructure that has been set up for the kargo-bootstrap project.

## Directory Structure

```
.
├── tests/                 # Test utilities and integration tests
│   └── utils.go          # Test utilities and helper functions
├── testdata/             # Test fixtures
│   ├── charts/           # Test Helm charts
│   │   └── test-chart/   # A sample test chart
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       └── templates/
│   │           └── deployment.yaml
│   ├── repos/            # Test Git repositories
│   │   └── README.md
│   ├── yaml/             # Test YAML files
│   │   ├── application.yaml
│   │   └── project.yaml
│   └── config/           # Test configuration files
│       └── test-config.yaml
└── mocks/                # Mock implementations
    ├── mock_k8s.go       # Mock Kubernetes client
    ├── mock_argocd.go     # Mock ArgoCD client
    ├── mock_git.go        # Mock Git client
    └── mock_prompt.go     # Mock prompt client
```

## Mock Implementations

### MockK8sClient

The `MockK8sClient` provides a mock implementation of the Kubernetes client for testing. It includes:

- Namespace management (create, get, list, delete)
- Label and annotation management
- Namespace validation
- Namespace generation

### MockArgoCDClient

The `MockArgoCDClient` provides a mock implementation of the ArgoCD client for testing. It includes:

- Project management (create, get, list, delete)
- Application management (create, get, list, delete, sync)
- Status checking
- Validation

### MockGitClient

The `MockGitClient` provides a mock implementation of the Git client for testing. It includes:

- Repository cloning
- Branch and tag management
- File operations in repositories
- Chart discovery

### MockPromptClient

The `MockPromptClient` provides a mock implementation of the prompt client for testing. It includes:

- Input prompts
- Selection prompts
- Confirmation prompts
- Predefined response handling

## Test Utilities

The `tests/utils.go` file provides utility functions for testing, including:

- Test setup and cleanup
- Mock client creation
- Assertion helpers
- Test data management

## Test Fixtures

### Test Helm Chart

A sample Helm chart is provided in `testdata/charts/test-chart/` for testing chart-related functionality. It includes:

- Chart.yaml with basic metadata
- values.yaml with default values
- A deployment template

### Test YAML Files

Sample YAML files are provided in `testdata/yaml/` for testing YAML-related functionality:

- `application.yaml`: A sample ArgoCD Application
- `project.yaml`: A sample ArgoCD Project

### Test Configuration

A sample configuration file is provided in `testdata/config/test-config.yaml` for testing configuration-related functionality.

## Makefile Targets

The Makefile includes numerous targets for testing:

- `test`: Run all tests
- `test-unit`: Run unit tests
- `test-integration`: Run integration tests
- `test-coverage`: Run tests with coverage
- `test-[package]`: Run tests for specific packages
- Various other testing options

## Running Tests

To run all tests:

```bash
make test
```

To run unit tests:

```bash
make test-unit
```

To run tests with coverage:

```bash
make test-coverage
```

To run tests for a specific package:

```bash
make test-k8s
make test-argocd
make test-git
make test-yaml
make test-config
make test-errors
make test-prompt
```

## Writing New Tests

When writing new tests:

1. Use the mock implementations for external dependencies
2. Use the test utilities for common test operations
3. Use the test fixtures for sample data
4. Follow the naming conventions: `function_test.go` for unit tests
5. Use table-driven tests for multiple test cases

## Example Test

Here's an example of how to use the testing infrastructure:

```go
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
```

## Next Steps

The remaining tasks are to create unit tests for each package:

1. pkg/k8s functions (namespace.go)
2. pkg/argocd functions (projects.go, application.go)
3. pkg/git functions (clone.go, discovery.go)
4. pkg/yaml functions (application.go)
5. pkg/config functions (env.go)
6. pkg/errors functions (validation.go)
7. Integration tests (deploy_test.go, argocd_test.go, git_test.go, env_test.go)

These tests should follow the patterns established in this setup.
