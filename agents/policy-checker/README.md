# 🛡️ Demo 4: Policy Checker Agent

> **Difficulty:** ⭐⭐⭐ Advanced | **Time:** 30 minutes

Build a full Copilot Agent that checks your Infrastructure as Code against Azure Policy definitions and security best practices using real Azure APIs.

```
  ┌──────────────────────────────────────────────────────────────┐
  │                   Policy Checker Agent                        │
  ├──────────────────────────────────────────────────────────────┤
  │                                                               │
  │   User: "Check if my Terraform follows Azure policies"       │
  │         │                                                     │
  │         ▼                                                     │
  │   ┌─────────────────────────────────────────────────────┐    │
  │   │                  Agent Handler                       │    │
  │   │                                                      │    │
  │   │  1. Parse IaC code                                  │    │
  │   │  2. Extract resource configurations                 │    │
  │   │  3. Query Azure Policy API                         │    │
  │   │  4. Check against built-in policies                │    │
  │   │  5. Stream results back to user                    │    │
  │   │                                                      │    │
  │   └──────────────────┬──────────────────────────────────┘    │
  │                      │                                        │
  │                      ▼                                        │
  │   ┌─────────────────────────────────────────────────────┐    │
  │   │               Azure Policy API                       │    │
  │   │                                                      │    │
  │   │  • Get policy definitions                           │    │
  │   │  • List built-in policies                           │    │
  │   │  • Check compliance requirements                    │    │
  │   │                                                      │    │
  │   └─────────────────────────────────────────────────────┘    │
  │                                                               │
  │   Response: "Found 3 policy violations:                      │
  │              1. Storage: HTTPS not enforced                  │
  │              2. VM: No managed identity                      │
  │              3. Network: Public IP allowed"                  │
  │                                                               │
  └──────────────────────────────────────────────────────────────┘
```

## 📋 What You'll Learn

- Full Copilot Agent architecture
- Server-Sent Events (SSE) for streaming responses
- Azure Policy REST API integration
- Parsing and analyzing IaC configurations
- Security best practices validation

---

## 🎯 Objectives

1. ✅ Understand Agent vs Skillset architecture
2. ✅ Implement event stream handling
3. ✅ Integrate with Azure Policy API
4. ✅ Parse Terraform/Bicep for policy compliance
5. ✅ Stream real-time results to Copilot

---

## 📚 Agent Architecture

Agents provide full control over the conversation flow:

```
┌───────────────────────────────────────────────────────────────┐
│                    Agent Event Flow                            │
├───────────────────────────────────────────────────────────────┤
│                                                                │
│   GitHub Copilot                                               │
│        │                                                       │
│        │  POST /agent (SSE)                                   │
│        ▼                                                       │
│   ┌─────────────┐                                             │
│   │   Agent     │───▶ Parse messages                          │
│   │   Handler   │───▶ Process request                         │
│   │             │───▶ Call Azure APIs                         │
│   │             │───▶ Stream response events                  │
│   └─────────────┘                                             │
│        │                                                       │
│        │  SSE Events:                                         │
│        │  • copilot_message (content)                         │
│        │  • copilot_confirmation (actions)                    │
│        │  • copilot_references (links)                        │
│        ▼                                                       │
│   User sees streaming response                                 │
│                                                                │
└───────────────────────────────────────────────────────────────┘
```

---

## 🛠️ Prerequisites

- Go 1.21+
- Azure CLI (logged in)
- Azure subscription with Policy access
- ngrok for exposing the agent
- GitHub App configured for Agent type

---

## 📂 Project Structure

```
03-policy-agent/
├── go.mod
├── go.sum
├── main.go                # HTTP server & agent handler
├── agent/
│   ├── handler.go         # Copilot event handling
│   ├── parser.go          # IaC parsing
│   └── policy.go          # Policy validation
├── azure/
│   ├── client.go          # Azure API client
│   └── policy.go          # Policy API integration
├── policies/
│   ├── builtin.go         # Built-in policy checks
│   └── rules.json         # Custom policy rules
└── sse/
    └── writer.go          # SSE event writer
```

---

## 📝 Azure Setup

### Get Azure Credentials

```bash
# Login to Azure
az login

# Get access token for Azure Management API
az account get-access-token --resource https://management.azure.com
```

### Required Permissions

Your Azure account needs:
- `Microsoft.Authorization/policyDefinitions/read`
- `Microsoft.Authorization/policyAssignments/read`

---

## 📝 GitHub App Setup (Agent Type)

1. Go to [GitHub Developer Settings](https://github.com/settings/apps)

2. Create new GitHub App:
   - **Name:** `Policy Checker Agent`
   - **Homepage URL:** Your repo URL
   - **Webhook:** Uncheck "Active"

3. Under **Copilot:**
   - Check **"Copilot Extension"**
   - Extension Type: **Agent**
   - Inference Description: "Checks IaC against Azure policies"

4. Set **Callback URL** to your ngrok URL: `https://xxx.ngrok.io/agent`

---

## 📝 Running the Agent

```bash
cd Copilot-SDK/03-policy-agent

# Set environment variables
export AZURE_SUBSCRIPTION_ID="your-subscription-id"
export GITHUB_WEBHOOK_SECRET="your-secret"  # Optional

# Build and run
go mod tidy
go run .

# Server starts on :8080
```

In another terminal:

```bash
ngrok http 8080
```

Update your GitHub App callback URL with the ngrok URL.

---

## 🔧 Policy Checks Included

### Storage Account Policies

| Policy | Description |
|--------|-------------|
| HTTPS Required | Storage accounts should use HTTPS |
| TLS 1.2 | Minimum TLS version should be 1.2 |
| Public Access | Blob public access should be disabled |
| Network Rules | Default network action should be Deny |

### Kubernetes (AKS) Policies

| Policy | Description |
|--------|-------------|
| RBAC | RBAC should be enabled |
| Network Policy | Network policy should be configured |
| Managed Identity | Should use managed identity |
| Private Cluster | API server should be private |

### Virtual Machine Policies

| Policy | Description |
|--------|-------------|
| Managed Disks | VMs should use managed disks |
| Encryption | Disk encryption should be enabled |
| Managed Identity | Should use managed identity |
| Extensions | Only approved extensions |

### Network Policies

| Policy | Description |
|--------|-------------|
| NSG | Subnets should have NSG |
| No Public IP | Resources shouldn't expose public IPs |
| DDoS Protection | VNets should have DDoS protection |

---

## 🎮 Usage Examples

### In Copilot Chat

```
@policy-checker Check this Terraform for policy compliance:
resource "azurerm_storage_account" "example" {
  name                     = "storageaccount"
  resource_group_name      = "rg-example"
  location                 = "eastus"
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

@policy-checker What Azure policies apply to AKS clusters?

@policy-checker Scan my current directory for policy violations
```

---

## 📋 SSE Event Types

The agent uses Server-Sent Events to stream responses:

```
event: copilot_message
data: {"content": "🔍 Analyzing your Terraform code..."}

event: copilot_message
data: {"content": "Found 2 policy violations:\n\n"}

event: copilot_references
data: {"references": [{"title": "Azure Policy Docs", "url": "https://..."}]}

event: copilot_confirmation
data: {"confirmation": {"title": "Apply fixes?", "message": "..."}}
```

---

## 🧪 Testing

```bash
# Test the agent endpoint
curl -X POST http://localhost:8080/agent \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "Check this Terraform:\nresource \"azurerm_storage_account\" \"test\" {\n  name = \"test\"\n}"
    }]
  }'
```

---

## ✅ Completion Checklist

- [ ] Configured Azure credentials
- [ ] Created GitHub App (Agent type)
- [ ] Implemented agent handler with SSE
- [ ] Integrated Azure Policy API
- [ ] Added IaC parsing for Terraform/Bicep
- [ ] Tested with Copilot Chat

---

<div align="center">

**🏆 Achievement Unlocked: Agent Developer 🟠**

[← Back to Copilot SDK](../README.md) | [Next: Cost Estimator Agent →](../04-cost-estimator/README.md)

</div>
