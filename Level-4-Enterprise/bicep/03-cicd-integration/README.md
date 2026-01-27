# Level 4.3: CI/CD Integration

Implement CI/CD pipelines for Bicep deployments using GitHub Actions and Azure DevOps.

## Learning Objectives

- Configure GitHub Actions for Bicep
- Implement what-if deployments for PRs
- Set up OIDC authentication
- Create deployment workflows

## Challenge

Create:
1. A Bicep template for a web application
2. A GitHub Actions workflow that:
   - Validates Bicep on PRs
   - Runs what-if on PRs
   - Deploys on merge to main

## Copilot Prompts to Try

```
Create a GitHub Actions workflow for Bicep with OIDC authentication to Azure
```

```
Add a what-if step to the Bicep deployment workflow
```

```
How do I configure federated credentials for GitHub Actions?
```

## Files

- `challenge/main.bicep` - Application template
- `challenge/.github/workflows/` - Workflow to complete
- `solution/` - Complete implementation

## GitHub Actions Setup

### 1. Create Azure AD App Registration
```bash
az ad app create --display-name "github-actions-bicep"
```

### 2. Configure Federated Credentials
```bash
az ad app federated-credential create \
  --id <app-id> \
  --parameters @credential.json
```

### 3. Assign RBAC Role
```bash
az role assignment create \
  --assignee <app-id> \
  --role "Contributor" \
  --scope /subscriptions/<sub-id>
```

## Workflow Triggers

| Event | Action |
|-------|--------|
| Pull Request | Validate + What-If |
| Push to main | Deploy to Dev |
| Release | Deploy to Prod |
