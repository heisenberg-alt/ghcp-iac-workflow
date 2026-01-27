# Level 4.2: Policy as Code

Implement Azure Policy definitions and assignments using Bicep for governance at scale.

## Learning Objectives

- Create custom policy definitions
- Build policy initiatives (policy sets)
- Assign policies to scopes
- Implement governance guardrails

## Challenge

Create Bicep templates that:
1. Define a custom policy for allowed locations
2. Define a custom policy for required tags
3. Create a policy initiative combining both
4. Assign the initiative to a resource group

## Copilot Prompts to Try

```
Create a custom Azure Policy definition in Bicep to restrict allowed locations
```

```
How do I create a policy initiative (policy set) in Bicep?
```

```
Assign a policy initiative to a resource group scope
```

## Files

- `challenge/main.bicep` - Starter template with TODOs
- `solution/main.bicep` - Complete implementation

## Deployment

```bash
# Deploy at subscription scope
az deployment sub create \
  --location eastus \
  --template-file solution/main.bicep \
  --parameters environment=prod
```

## Policy Effects

| Effect | Description |
|--------|-------------|
| Deny | Block non-compliant resources |
| Audit | Log non-compliance |
| Modify | Auto-remediate resources |
| DeployIfNotExists | Deploy missing resources |
