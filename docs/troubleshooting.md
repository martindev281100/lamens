# Troubleshooting

This document provides solutions to common issues you might encounter when using kargo-bootstrap.

## Connection Issues

### ArgoCD Connection Failed

**Error**: `Error connecting to ArgoCD`

**Solution**:

1. Verify ArgoCD is installed:

   ```bash
   kubectl get pods -n argocd
   ```

2. Check if ArgoCD is in a different namespace:

   ```bash
   kubectl get pods --all-namespaces | grep argocd
   ```

3. Test with the correct namespace:

   ```bash
   kargo-bootstrap argocd test --argocd-namespace YOUR_ARGOCD_NAMESPACE
   ```

4. Verify ArgoCD API is accessible:
   ```bash
   kubectl get deployment argocd-server -n argocd
   ```

### Kubernetes Connection Failed

**Error**: `Error creating Kubernetes client`

**Solution**:

1. Verify your kubeconfig file:

   ```bash
   kubectl config view
   ```

2. Test cluster connectivity:

   ```bash
   kubectl cluster-info
   ```

3. Check current context:

   ```bash
   kubectl config current-context
   ```

4. Use a custom kubeconfig:
   ```bash
   kargo-bootstrap argocd test --kubeconfig /path/to/kubeconfig
   ```

## Git Issues

### Git Repository Access Denied

**Error**: `authentication failed` or `repository not found`

**Solution**:

1. Verify repository URL is correct:

   ```bash
   kargo-bootstrap git clone --url YOUR_REPO_URL --dry-run
   ```

2. Test with token authentication:

   ```bash
   kargo-bootstrap git clone \
     --url YOUR_REPO_URL \
     --token YOUR_TOKEN
   ```

3. Test with username/password:

   ```bash
   kargo-bootstrap git clone \
     --url YOUR_REPO_URL \
     --username YOUR_USERNAME \
     --password YOUR_PASSWORD
   ```

4. Test with SSH key:
   ```bash
   kargo-bootstrap git clone \
     --url YOUR_REPO_URL \
     --ssh-key-path ~/.ssh/id_rsa
   ```

### No Helm Charts Found

**Error**: `No Helm charts found in repository`

**Solution**:

1. Verify the repository structure:

   ```bash
   kargo-bootstrap git clone --url YOUR_REPO_URL
   # Check for Chart.yaml files
   find . -name "Chart.yaml" -type f
   ```

2. List all charts in the repository:

   ```bash
   kargo-bootstrap git list-charts --url YOUR_REPO_URL
   ```

3. Check if charts are in a subdirectory:
   ```bash
   kargo-bootstrap git list-charts \
     --url YOUR_REPO_URL \
     --branch YOUR_BRANCH
   ```

## Deployment Issues

### Application Not Syncing

**Error**: Application is in `OutOfSync` status

**Solution**:

1. Check application status:

   ```bash
   kargo-bootstrap argocd status --name YOUR_APP_NAME
   ```

2. Manually sync the application:

   ```bash
   kargo-bootstrap argocd sync --name YOUR_APP_NAME
   ```

3. Force sync if needed:

   ```bash
   kargo-bootstrap argocd sync --name YOUR_APP_NAME --force
   ```

4. Check application logs:
   ```bash
   kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller
   ```

### Namespace Creation Failed

**Error**: `Error creating namespace`

**Solution**:

1. Check if you have permission to create namespaces:

   ```bash
   kubectl auth can-i create namespace
   ```

2. Verify the namespace doesn't already exist:

   ```bash
   kubectl get namespace YOUR_NAMESPACE
   ```

3. Create the namespace manually:
   ```bash
   kubectl create namespace YOUR_NAMESPACE
   ```

### Application Health Check Failed

**Error**: Application is in `Degraded` health status

**Solution**:

1. Check application status:

   ```bash
   kargo-bootstrap argocd status --name YOUR_APP_NAME
   ```

2. Check pods in the application namespace:

   ```bash
   kubectl get pods -n YOUR_APP_NAMESPACE
   ```

3. Check pod logs:

   ```bash
   kubectl logs -n YOUR_APP_NAMESPACE -l app=YOUR_APP_NAME
   ```

4. Describe problematic pods:
   ```bash
   kubectl describe pod -n YOUR_APP_NAMESPACE POD_NAME
   ```

## Environment Issues

### Environment Validation Failed

**Error**: `Environment validation failed`

**Solution**:

1. Check environment configuration:

   ```bash
   kargo-bootstrap env list
   ```

2. Validate specific environments:

   ```bash
   kargo-bootstrap env validate ENVIRONMENT_NAME
   ```

3. Check for required fields in your environment configuration

### Custom Environment Not Found

**Error**: `Environment not found`

**Solution**:

1. Verify the environment configuration file:

   ```bash
   kargo-bootstrap env list --config YOUR_CONFIG_FILE
   ```

2. Check if the environment is defined in the configuration

3. Validate the configuration file syntax

## Configuration Issues

### Invalid Configuration File

**Error**: `Error loading environment config file`

**Solution**:

1. Validate YAML syntax:

   ```bash
   yamllint YOUR_CONFIG_FILE
   ```

2. Check file permissions:

   ```bash
   ls -la YOUR_CONFIG_FILE
   ```

3. Verify file path is correct

### Required Flag Missing

**Error**: `required flag(s) not set`

**Solution**:

1. Check command help for required flags:

   ```bash
   kargo-bootstrap deploy --help
   ```

2. Provide all required flags or use interactive mode

3. Use configuration files to set default values

## Performance Issues

### Slow Git Operations

**Solution**:

1. Use shallow clones:

   ```bash
   kargo-bootstrap git clone \
     --url YOUR_REPO_URL \
     --shallow true
   ```

2. Use a closer Git mirror if available

3. Check network connectivity to the Git repository

### Slow ArgoCD Operations

**Solution**:

1. Check ArgoCD server resources:

   ```bash
   kubectl top pods -n argocd
   ```

2. Increase ArgoCD server resources if needed

3. Check for network issues between kargo-bootstrap and ArgoCD

## Debugging Tips

### Enable Verbose Logging

Use the `--verbose` flag to get detailed output:

```bash
kargo-bootstrap deploy --verbose
```

### Use Dry-Run Mode

Preview changes without applying them:

```bash
kargo-bootstrap deploy --dry-run
```

### Validate Configuration

Validate your configuration before deploying:

```bash
kargo-bootstrap render application \
  --name YOUR_APP_NAME \
  --project YOUR_PROJECT \
  --repo YOUR_REPO_URL \
  --path YOUR_CHART_PATH \
  --validate
```

### Test Individual Components

Test Git operations:

```bash
kargo-bootstrap git clone --url YOUR_REPO_URL
kargo-bootstrap git list-charts --url YOUR_REPO_URL
```

Test ArgoCD operations:

```bash
kargo-bootstrap argocd test
kargo-bootstrap argocd list
```

## Common Error Messages

### `Error: repository URL is required in non-interactive mode`

**Cause**: Using non-interactive mode without providing a repository URL

**Solution**: Provide the `--repository-url` flag or use interactive mode

### `Error: Cannot use token authentication with username/password authentication`

**Cause**: Providing both token and username/password authentication

**Solution**: Use only one authentication method

### `Error: Cannot specify multiple Git revisions (branch, tag, commit)`

**Cause**: Providing multiple revision flags

**Solution**: Use only one revision flag (branch, tag, or commit)

### `Error: No ArgoCD projects found`

**Cause**: No ArgoCD projects exist in the specified namespace

**Solution**: Create an ArgoCD project or use a different namespace

### `Error: Chart validation warning`

**Cause**: The Helm chart has validation issues

**Solution**: Fix the chart issues or continue with the warning

## Getting Help

If you're still experiencing issues:

1. Check the [GitHub Issues](https://github.com/your-org/kargo-bootstrap/issues) page
2. Create a new issue with:

   - The exact error message
   - The command you ran
   - The `--verbose` output
   - Your environment details (OS, kargo-bootstrap version, etc.)

3. Join our [Discord Community](https://discord.gg/kargo-bootstrap) for live support

## Reporting Bugs

When reporting bugs, please include:

1. **kargo-bootstrap version**: `kargo-bootstrap version`
2. **Kubernetes version**: `kubectl version`
3. **ArgoCD version**: `argocd version`
4. **Operating System**: `uname -a`
5. **Command that failed**: Include all flags and parameters
6. **Verbose output**: Use the `--verbose` flag
7. **Steps to reproduce**: Detailed steps to reproduce the issue
8. **Expected behavior**: What you expected to happen
9. **Actual behavior**: What actually happened
10. **Additional context**: Any other relevant information

## Feature Requests

For feature requests, please:

1. Check existing issues to avoid duplicates
2. Provide a clear description of the feature
3. Explain the use case and why the feature is needed
4. Suggest how the feature should work
