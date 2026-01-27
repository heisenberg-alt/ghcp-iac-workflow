# Level 4.1: Multi-Region Deployment

Deploy applications across multiple Azure regions with Azure Front Door for global load balancing.

## Learning Objectives

- Deploy resources to multiple regions using loops
- Configure Azure Front Door for global distribution
- Implement traffic routing policies
- Handle regional failover

## Challenge

Create a Bicep template that deploys:
1. Web apps in multiple regions (eastus, westeurope)
2. Azure Front Door for global load balancing
3. Health probes for automatic failover

## Copilot Prompts to Try

```
Create a Bicep template for multi-region web app deployment with Azure Front Door
```

```
How do I configure Azure Front Door origins and routes in Bicep?
```

```
Add health probes and failover configuration to Front Door
```

## Files

- `challenge/main.bicep` - Starter template with TODOs
- `solution/main.bicep` - Complete implementation

## Deployment

```bash
# Deploy the solution
az deployment sub create \
  --location eastus \
  --template-file solution/main.bicep \
  --parameters environment=dev
```

## Validation

After deployment, verify:
1. Web apps are running in both regions
2. Front Door endpoint is accessible
3. Traffic routes to healthy origins
