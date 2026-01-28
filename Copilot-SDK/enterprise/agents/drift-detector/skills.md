# Drift Detector Agent Skills

## Overview

The Drift Detector Agent compares Infrastructure as Code definitions with live Azure resources to identify configuration drift, missing resources, and manual changes.

## Skills

### 1. Drift Detection (`/agent`)

**Endpoint:** `POST /agent`

**Description:** Compares Terraform or Bicep code with actual Azure resource configurations to detect drift.

**Request Format:**
```json
{
  "messages": [
    {
      "role": "user",
      "content": "Check drift for:\n\n```terraform\nresource \"azurerm_storage_account\" \"example\" {\n  name = \"storage\"\n  enable_https_traffic_only = true\n}\n```"
    }
  ]
}
```

**Response:** Server-Sent Events (SSE) stream with drift analysis

**Capabilities:**
- 🔄 **Property Drift:** Detect when Azure resource properties differ from IaC definitions
- ❌ **Missing Resources:** Find resources defined in IaC but not in Azure
- ➕ **Unmanaged Resources:** Identify resources in Azure not tracked by IaC
- 🏷️ **Tag Drift:** Compare tag values between IaC and Azure

### 2. Full Subscription Scan (`/scan`)

**Endpoint:** `POST /scan`

**Description:** Initiates a full subscription drift scan (requires Azure credentials).

**Request Format:**
```json
{
  "subscription_id": "xxx-xxx-xxx",
  "resource_groups": ["rg-prod", "rg-staging"]
}
```

### 3. Get Latest Report (`/report`)

**Endpoint:** `GET /report`

**Description:** Returns the most recent drift report.

### 4. Health Check (`/health`)

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "service": "drift-detector-agent",
  "azure_configured": true,
  "subscription_id": "xxx-xxx-xxx"
}
```

## Drift Status Types

| Status | Icon | Description |
|--------|------|-------------|
| in_sync | ✅ | IaC and Azure match |
| drifted | 🔄 | Properties differ between IaC and Azure |
| missing_in_azure | ❌ | Resource in IaC but not found in Azure |
| missing_in_iac | ➕ | Resource in Azure but not defined in IaC |

## Drift Severity

| Severity | Icon | Meaning |
|----------|------|---------|
| critical | 🔴 | Security-related drift requiring immediate action |
| high | 🟠 | Important configuration drift |
| medium | 🟡 | Non-critical property drift |
| low | 🟢 | Minor differences (tags, descriptions) |

## Properties Monitored

### Storage Accounts
- `enable_https_traffic_only`
- `min_tls_version`
- `allow_blob_public_access`
- `public_network_access_enabled`

### Kubernetes Clusters
- `role_based_access_control_enabled`
- `kubernetes_version`
- `network_policy`

### Key Vaults
- `soft_delete_retention_days`
- `purge_protection_enabled`
- `enable_rbac_authorization`

## Azure Integration

### Required Permissions
- `Reader` on target subscriptions
- `Azure Resource Graph Reader`

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| PORT | No | Server port (default: 8083) |
| AZURE_SUBSCRIPTION_ID | Yes | Target Azure subscription |
| AZURE_TENANT_ID | Yes | Azure AD tenant |
| AZURE_CLIENT_ID | Conditional | For service principal auth |
| AZURE_CLIENT_SECRET | Conditional | For service principal auth |

## Example Drift Report

```markdown
## Drift Detection Report

### 🔄 azurerm_storage_account.example

| Property | Expected (IaC) | Actual (Azure) | Severity |
|----------|----------------|----------------|----------|
| enable_https_traffic_only | true | false | 🟠 high |
| min_tls_version | TLS1_2 | TLS1_0 | 🟠 high |

**Remediation:** Run `terraform apply` to sync
```

## VS Code Copilot Usage

```
@drift-detector Check if my Terraform matches Azure
```

## Running the Agent

```bash
# Build
go build -o drift-detector.exe .

# Run with Azure credentials
$env:PORT="8083"
$env:AZURE_SUBSCRIPTION_ID="your-subscription-id"
.\drift-detector.exe
```

## Scheduled Drift Detection

For continuous drift monitoring, run the agent with a cron schedule:

```yaml
# Kubernetes CronJob example
apiVersion: batch/v1
kind: CronJob
spec:
  schedule: "0 */6 * * *"  # Every 6 hours
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: drift-scanner
              command: ["./drift-detector", "--scan-all"]
```
