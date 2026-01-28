# Enterprise IaC Governance Platform - Deployment Guide

This guide covers deploying and operating the Enterprise IaC Governance Platform, a multi-agent system for automated infrastructure-as-code validation, cost estimation, compliance checking, and deployment promotion.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start (Local Development)](#quick-start-local-development)
4. [Production Deployment](#production-deployment)
5. [Configuration Reference](#configuration-reference)
6. [Agent Endpoints](#agent-endpoints)
7. [Usage Examples](#usage-examples)
8. [GitHub Actions Integration](#github-actions-integration)
9. [Monitoring & Alerts](#monitoring--alerts)
10. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        GitHub Copilot Chat Interface                        │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Orchestrator Agent (8090)                           │
│  • Request routing & workflow coordination                                  │
│  • Parallel agent execution                                                 │
│  • Result aggregation & reporting                                           │
└─────────────────────────────────────────────────────────────────────────────┘
          │           │           │           │           │           │
          ▼           ▼           ▼           ▼           ▼           ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│   Policy    │ │    Cost     │ │    Drift    │ │  Security   │ │ Compliance  │
│   Checker   │ │  Estimator  │ │  Detector   │ │   Scanner   │ │   Auditor   │
│   (8081)    │ │   (8082)    │ │   (8083)    │ │   (8084)    │ │   (8085)    │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘
          │           │           │           │           │
          ▼           ▼           ▼           ▼           ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────────┐
│   Module    │ │   Impact    │ │   Deploy    │ │    Notification Manager     │
│  Registry   │ │  Analyzer   │ │  Promoter   │ │          (8089)             │
│   (8086)    │ │   (8087)    │ │   (8088)    │ │  Teams/Slack/Email/Webhook  │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────────────────────┘
```

### Agent Responsibilities

| Agent | Port | Purpose |
|-------|------|---------|
| **Orchestrator** | 8090 | Routes requests, coordinates workflows, aggregates results |
| **Policy Checker** | 8081 | Validates IaC against organization policies |
| **Cost Estimator** | 8082 | Estimates Azure resource costs using Retail Prices API |
| **Drift Detector** | 8083 | Compares IaC state with actual Azure resources |
| **Security Scanner** | 8084 | Checks for security misconfigurations and vulnerabilities |
| **Compliance Auditor** | 8085 | Validates against regulatory frameworks (CIS, NIST, SOC2) |
| **Module Registry** | 8086 | Manages approved IaC modules and versions |
| **Impact Analyzer** | 8087 | Analyzes blast radius and dependency impacts |
| **Deploy Promoter** | 8088 | Manages environment promotions (dev→staging→prod) |
| **Notification Manager** | 8089 | Sends alerts via Teams, Slack, Email, Webhooks |

---

## Prerequisites

### Required Software

| Software | Minimum Version | Purpose |
|----------|-----------------|---------|
| Go | 1.21+ | Build agents |
| Docker | 24.0+ | Container deployment |
| kubectl | 1.28+ | Kubernetes deployment |
| Helm | 3.12+ | Kubernetes package management |
| Azure CLI | 2.50+ | Azure authentication |

### Azure Requirements

```bash
# Required Azure permissions
- Reader access to target subscriptions (for drift detection)
- Azure Resource Graph Reader (for resource queries)
- Key Vault Secrets User (for secret management)

# Required Azure resources
- Azure Container Registry (for container images)
- Azure Key Vault (for secrets)
- Log Analytics Workspace (for monitoring)
- Application Insights (for APM)
```

### Environment Variables

```bash
# Core Configuration
export AZURE_SUBSCRIPTION_ID="your-subscription-id"
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-client-id"
export AZURE_CLIENT_SECRET="your-client-secret"  # Or use managed identity

# Notification Configuration
export TEAMS_WEBHOOK_URL="https://outlook.office.com/webhook/..."
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
export SMTP_HOST="smtp.office365.com"
export SMTP_PORT="587"
export SMTP_USERNAME="alerts@yourdomain.com"
export SMTP_PASSWORD="your-smtp-password"
export ALERT_EMAIL_RECIPIENTS="team@yourdomain.com"

# GitHub Integration
export GITHUB_TOKEN="ghp_..."
export GITHUB_WEBHOOK_SECRET="your-webhook-secret"
```

---

## Quick Start (Local Development)

### 1. Clone and Build

```bash
# Clone repository
git clone https://github.com/your-org/copilot-iac.git
cd copilot-iac/Copilot-SDK/enterprise

# Build all agents
./scripts/build-all.sh

# Or build individually
cd agents/orchestrator && go build -o orchestrator.exe .
cd ../policy-checker && go build -o policy-checker.exe .
cd ../cost-estimator && go build -o cost-estimator.exe .
# ... repeat for other agents
```

### 2. Start Agents

```bash
# Start all agents (PowerShell)
.\scripts\start-local.ps1

# Or start individually in separate terminals
$env:PORT="8090"; .\agents\orchestrator\orchestrator.exe
$env:PORT="8081"; .\agents\policy-checker\policy-checker.exe
$env:PORT="8082"; .\agents\cost-estimator\cost-estimator.exe
$env:PORT="8083"; .\agents\drift-detector\drift-detector.exe
$env:PORT="8084"; .\agents\security-scanner\security-scanner.exe
$env:PORT="8085"; .\agents\compliance-auditor\compliance-auditor.exe
$env:PORT="8086"; .\agents\module-registry\module-registry.exe
$env:PORT="8087"; .\agents\impact-analyzer\impact-analyzer.exe
$env:PORT="8088"; .\agents\deploy-promoter\deploy-promoter.exe
$env:PORT="8089"; .\agents\notification-manager\notification-manager.exe
```

### 3. Verify Health

```bash
# Check all agents
curl http://localhost:8090/health  # Orchestrator
curl http://localhost:8081/health  # Policy Checker
curl http://localhost:8082/health  # Cost Estimator
# ... check all ports 8081-8090
```

### 4. Test the Platform

```bash
# Full governance check via Orchestrator
curl -X POST http://localhost:8090/agent \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "Run full governance check on this Terraform:\n\nresource \"azurerm_storage_account\" \"example\" {\n  name                     = \"examplestorage\"\n  resource_group_name      = \"example-rg\"\n  location                 = \"eastus\"\n  account_tier             = \"Standard\"\n  account_replication_type = \"GRS\"\n}"
    }]
  }'
```

---

## Production Deployment

### Option 1: Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  orchestrator:
    build: ./agents/orchestrator
    ports:
      - "8090:8090"
    environment:
      - PORT=8090
      - POLICY_AGENT_URL=http://policy-checker:8081
      - COST_AGENT_URL=http://cost-estimator:8082
      - DRIFT_AGENT_URL=http://drift-detector:8083
      - SECURITY_AGENT_URL=http://security-scanner:8084
      - COMPLIANCE_AGENT_URL=http://compliance-auditor:8085
      - MODULE_AGENT_URL=http://module-registry:8086
      - IMPACT_AGENT_URL=http://impact-analyzer:8087
      - DEPLOY_AGENT_URL=http://deploy-promoter:8088
      - NOTIFICATION_AGENT_URL=http://notification-manager:8089
    depends_on:
      - policy-checker
      - cost-estimator
      - drift-detector
      - security-scanner
      - compliance-auditor
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8090/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  policy-checker:
    build: ./agents/policy-checker
    ports:
      - "8081:8081"
    environment:
      - PORT=8081
    volumes:
      - ./config/policies:/app/config/policies:ro
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  cost-estimator:
    build: ./agents/cost-estimator
    ports:
      - "8082:8082"
    environment:
      - PORT=8082
    volumes:
      - ./config/sku-mappings:/app/data:ro
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  drift-detector:
    build: ./agents/drift-detector
    ports:
      - "8083:8083"
    environment:
      - PORT=8083
      - AZURE_SUBSCRIPTION_ID=${AZURE_SUBSCRIPTION_ID}
      - AZURE_TENANT_ID=${AZURE_TENANT_ID}
      - AZURE_CLIENT_ID=${AZURE_CLIENT_ID}
      - AZURE_CLIENT_SECRET=${AZURE_CLIENT_SECRET}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8083/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  security-scanner:
    build: ./agents/security-scanner
    ports:
      - "8084:8084"
    environment:
      - PORT=8084
    volumes:
      - ./config/security-rules:/app/config/rules:ro
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8084/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  compliance-auditor:
    build: ./agents/compliance-auditor
    ports:
      - "8085:8085"
    environment:
      - PORT=8085
      - COMPLIANCE_FRAMEWORKS=CIS,NIST,SOC2
    volumes:
      - ./config/compliance:/app/config/compliance:ro
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8085/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  module-registry:
    build: ./agents/module-registry
    ports:
      - "8086:8086"
    environment:
      - PORT=8086
    volumes:
      - ./config/approved-modules:/app/config/modules:ro
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8086/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  impact-analyzer:
    build: ./agents/impact-analyzer
    ports:
      - "8087:8087"
    environment:
      - PORT=8087
      - AZURE_SUBSCRIPTION_ID=${AZURE_SUBSCRIPTION_ID}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8087/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  deploy-promoter:
    build: ./agents/deploy-promoter
    ports:
      - "8088:8088"
    environment:
      - PORT=8088
      - GITHUB_TOKEN=${GITHUB_TOKEN}
      - REQUIRE_APPROVALS=true
      - MIN_APPROVERS=2
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8088/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  notification-manager:
    build: ./agents/notification-manager
    ports:
      - "8089:8089"
    environment:
      - PORT=8089
      - TEAMS_WEBHOOK_URL=${TEAMS_WEBHOOK_URL}
      - SLACK_WEBHOOK_URL=${SLACK_WEBHOOK_URL}
      - SMTP_HOST=${SMTP_HOST}
      - SMTP_PORT=${SMTP_PORT}
      - SMTP_USERNAME=${SMTP_USERNAME}
      - SMTP_PASSWORD=${SMTP_PASSWORD}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8089/health"]
      interval: 30s
      timeout: 10s
      retries: 3

networks:
  default:
    name: iac-governance
```

**Deploy with Docker Compose:**

```bash
# Create .env file with secrets
cat > .env << EOF
AZURE_SUBSCRIPTION_ID=your-sub-id
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret
TEAMS_WEBHOOK_URL=your-teams-webhook
SLACK_WEBHOOK_URL=your-slack-webhook
GITHUB_TOKEN=your-github-token
SMTP_HOST=smtp.office365.com
SMTP_PORT=587
SMTP_USERNAME=alerts@yourdomain.com
SMTP_PASSWORD=your-smtp-password
EOF

# Deploy
docker-compose up -d

# Check status
docker-compose ps
docker-compose logs -f orchestrator
```

### Option 2: Kubernetes with Helm

```bash
# Add Helm repository (if published)
helm repo add iac-governance https://your-org.github.io/iac-governance-charts
helm repo update

# Install with custom values
helm install iac-governance iac-governance/enterprise \
  --namespace iac-governance \
  --create-namespace \
  --values values-production.yaml \
  --set azure.subscriptionId=$AZURE_SUBSCRIPTION_ID \
  --set azure.tenantId=$AZURE_TENANT_ID \
  --set notifications.teams.webhookUrl=$TEAMS_WEBHOOK_URL
```

**values-production.yaml:**

```yaml
# Helm values for production deployment
global:
  image:
    registry: youracr.azurecr.io
    pullPolicy: Always
  
  azure:
    subscriptionId: ""  # Set via --set
    tenantId: ""        # Set via --set
    useManagedIdentity: true

orchestrator:
  replicaCount: 2
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilization: 70

policyChecker:
  replicaCount: 2
  config:
    policiesConfigMap: iac-policies
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

costEstimator:
  replicaCount: 2
  config:
    skuMappingsConfigMap: sku-mappings
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

driftDetector:
  replicaCount: 1
  schedule: "0 */6 * * *"  # Run every 6 hours
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 1Gi

securityScanner:
  replicaCount: 2
  config:
    rulesConfigMap: security-rules
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

complianceAuditor:
  replicaCount: 2
  config:
    frameworks:
      - CIS
      - NIST
      - SOC2
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

moduleRegistry:
  replicaCount: 1
  persistence:
    enabled: true
    size: 10Gi
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

impactAnalyzer:
  replicaCount: 1
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 1Gi

deployPromoter:
  replicaCount: 1
  config:
    requireApprovals: true
    minApprovers: 2
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

notificationManager:
  replicaCount: 1
  secrets:
    teamsWebhookUrl: ""  # Set via --set or external secret
    slackWebhookUrl: ""
    smtp:
      host: smtp.office365.com
      port: 587
      username: ""
      password: ""
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: iac-governance.yourdomain.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: iac-governance-tls
      hosts:
        - iac-governance.yourdomain.com

monitoring:
  serviceMonitor:
    enabled: true
  dashboards:
    enabled: true
```

### Option 3: Azure Container Apps

```bash
# Create Container Apps Environment
az containerapp env create \
  --name iac-governance-env \
  --resource-group iac-governance-rg \
  --location eastus

# Deploy Orchestrator
az containerapp create \
  --name orchestrator \
  --resource-group iac-governance-rg \
  --environment iac-governance-env \
  --image youracr.azurecr.io/iac-governance/orchestrator:latest \
  --target-port 8090 \
  --ingress external \
  --min-replicas 1 \
  --max-replicas 5 \
  --cpu 0.5 \
  --memory 1.0Gi \
  --env-vars "POLICY_AGENT_URL=http://policy-checker" \
             "COST_AGENT_URL=http://cost-estimator"

# Deploy other agents similarly (internal ingress for backend agents)
az containerapp create \
  --name policy-checker \
  --resource-group iac-governance-rg \
  --environment iac-governance-env \
  --image youracr.azurecr.io/iac-governance/policy-checker:latest \
  --target-port 8081 \
  --ingress internal \
  --min-replicas 1 \
  --max-replicas 3
```

---

## Configuration Reference

### Policy Rules (config/policies/rules.json)

```json
{
  "rules": [
    {
      "id": "STORAGE_HTTPS",
      "name": "Storage Account HTTPS Required",
      "description": "Storage accounts must enforce HTTPS traffic only",
      "severity": "HIGH",
      "resource_type": "azurerm_storage_account",
      "property": "enable_https_traffic_only",
      "expected_value": true,
      "remediation": "Set enable_https_traffic_only = true"
    },
    {
      "id": "STORAGE_TLS",
      "name": "Storage Account Minimum TLS Version",
      "description": "Storage accounts must use TLS 1.2 or higher",
      "severity": "HIGH",
      "resource_type": "azurerm_storage_account",
      "property": "min_tls_version",
      "expected_value": "TLS1_2",
      "remediation": "Set min_tls_version = \"TLS1_2\""
    },
    {
      "id": "STORAGE_PUBLIC_ACCESS",
      "name": "Storage Account Public Access Disabled",
      "description": "Storage accounts must disable public blob access",
      "severity": "CRITICAL",
      "resource_type": "azurerm_storage_account",
      "property": "allow_blob_public_access",
      "expected_value": false,
      "remediation": "Set allow_blob_public_access = false"
    },
    {
      "id": "KV_SOFT_DELETE",
      "name": "Key Vault Soft Delete",
      "description": "Key Vaults must have soft delete enabled",
      "severity": "HIGH",
      "resource_type": "azurerm_key_vault",
      "property": "soft_delete_retention_days",
      "condition": "greater_than",
      "expected_value": 7,
      "remediation": "Set soft_delete_retention_days >= 7"
    }
  ]
}
```

### SKU Mappings (config/sku-mappings/mappings.json)

```json
{
  "azurerm_storage_account": {
    "serviceName": "Storage",
    "skuMappings": {
      "Standard_LRS": "Standard LRS",
      "Standard_GRS": "Standard GRS",
      "Standard_RAGRS": "Standard RA-GRS",
      "Premium_LRS": "Premium LRS"
    },
    "meterCategory": "Storage"
  },
  "azurerm_container_registry": {
    "serviceName": "Container Registry",
    "skuMappings": {
      "Basic": "Basic",
      "Standard": "Standard",
      "Premium": "Premium"
    }
  },
  "azurerm_service_plan": {
    "serviceName": "Azure App Service",
    "skuMappings": {
      "B1": "B1",
      "B2": "B2",
      "B3": "B3",
      "S1": "S1",
      "S2": "S2",
      "S3": "S3",
      "P1v2": "P1 v2",
      "P2v2": "P2 v2",
      "P3v2": "P3 v2",
      "P1v3": "P1 v3",
      "P2v3": "P2 v3",
      "P3v3": "P3 v3"
    }
  }
}
```

### Compliance Frameworks (config/compliance/)

```json
// cis-azure.json
{
  "framework": "CIS Azure Foundations Benchmark",
  "version": "2.0",
  "controls": [
    {
      "id": "CIS-2.1",
      "title": "Ensure that Microsoft Defender for Cloud is set to On",
      "check_type": "subscription_setting",
      "expected": "enabled"
    },
    {
      "id": "CIS-3.1",
      "title": "Ensure that 'Secure transfer required' is set to 'Enabled'",
      "resource_type": "azurerm_storage_account",
      "property": "enable_https_traffic_only",
      "expected": true
    }
  ]
}
```

---

## Agent Endpoints

### Orchestrator (8090)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/agent` | POST | Main entry point - routes to appropriate agents |
| `/workflow/full-check` | POST | Run all governance checks |
| `/workflow/pre-deploy` | POST | Pre-deployment validation |
| `/workflow/promote` | POST | Environment promotion workflow |

### Policy Checker (8081)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/agent` | POST | Check IaC against policy rules |
| `/rules` | GET | List all policy rules |
| `/rules/{id}` | GET | Get specific rule details |

### Cost Estimator (8082)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/agent` | POST | Estimate costs for IaC |
| `/prices/refresh` | POST | Refresh price cache |

### Drift Detector (8083)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/agent` | POST | Detect drift between IaC and actual resources |
| `/scan` | POST | Full subscription drift scan |
| `/report` | GET | Get latest drift report |

### Security Scanner (8084)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/agent` | POST | Scan IaC for security issues |
| `/rules` | GET | List security rules |

### Compliance Auditor (8085)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/agent` | POST | Audit IaC against compliance frameworks |
| `/frameworks` | GET | List supported frameworks |
| `/report` | POST | Generate compliance report |

### Notification Manager (8089)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/notify` | POST | Send notification |
| `/notify/teams` | POST | Send Teams notification |
| `/notify/slack` | POST | Send Slack notification |
| `/notify/email` | POST | Send email notification |

---

## Usage Examples

### 1. Full Governance Check

```bash
curl -X POST http://localhost:8090/agent \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "Run full governance check:\n\nresource \"azurerm_storage_account\" \"main\" {\n  name                     = \"mystorage\"\n  resource_group_name      = \"my-rg\"\n  location                 = \"eastus\"\n  account_tier             = \"Standard\"\n  account_replication_type = \"GRS\"\n  enable_https_traffic_only = true\n  min_tls_version          = \"TLS1_2\"\n}"
    }]
  }'
```

### 2. Cost Estimation Only

```bash
curl -X POST http://localhost:8082/agent \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "Estimate costs:\n\nresource \"azurerm_container_registry\" \"acr\" {\n  name                = \"myacr\"\n  resource_group_name = \"my-rg\"\n  location            = \"eastus\"\n  sku                 = \"Premium\"\n}"
    }]
  }'
```

### 3. Pre-Deployment Validation

```bash
curl -X POST http://localhost:8090/workflow/pre-deploy \
  -H "Content-Type: application/json" \
  -d '{
    "repository": "org/my-iac-repo",
    "branch": "feature/new-resources",
    "files": ["main.tf", "variables.tf"],
    "environment": "staging"
  }'
```

### 4. Send Notification

```bash
curl -X POST http://localhost:8089/notify \
  -H "Content-Type: application/json" \
  -d '{
    "channels": ["teams", "slack"],
    "severity": "warning",
    "title": "Policy Violations Detected",
    "message": "3 policy violations found in PR #42",
    "details": {
      "repository": "org/my-iac-repo",
      "pr_number": 42,
      "violations": [
        {"rule": "STORAGE_HTTPS", "severity": "HIGH"},
        {"rule": "STORAGE_TLS", "severity": "HIGH"},
        {"rule": "STORAGE_PUBLIC_ACCESS", "severity": "CRITICAL"}
      ]
    }
  }'
```

---

## GitHub Actions Integration

### .github/workflows/iac-governance.yml

```yaml
name: IaC Governance

on:
  pull_request:
    paths:
      - '**/*.tf'
      - '**/*.bicep'
      - '**/*.bicepparam'

env:
  GOVERNANCE_URL: https://iac-governance.yourdomain.com

jobs:
  governance-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Collect IaC Files
        id: collect
        run: |
          files=$(git diff --name-only ${{ github.event.pull_request.base.sha }} ${{ github.sha }} | grep -E '\.(tf|bicep)$' | tr '\n' ' ')
          echo "files=$files" >> $GITHUB_OUTPUT

      - name: Run Policy Check
        id: policy
        run: |
          for file in ${{ steps.collect.outputs.files }}; do
            content=$(cat "$file" | jq -Rs .)
            response=$(curl -s -X POST "${{ env.GOVERNANCE_URL }}/agent" \
              -H "Content-Type: application/json" \
              -d "{\"messages\":[{\"role\":\"user\",\"content\":\"Check policies:\\n\\n$content\"}]}")
            echo "$response"
          done

      - name: Run Cost Estimation
        id: cost
        run: |
          for file in ${{ steps.collect.outputs.files }}; do
            content=$(cat "$file" | jq -Rs .)
            response=$(curl -s -X POST "${{ env.GOVERNANCE_URL }}:8082/agent" \
              -H "Content-Type: application/json" \
              -d "{\"messages\":[{\"role\":\"user\",\"content\":\"Estimate costs:\\n\\n$content\"}]}")
            echo "$response"
          done

      - name: Run Security Scan
        id: security
        run: |
          for file in ${{ steps.collect.outputs.files }}; do
            content=$(cat "$file" | jq -Rs .)
            response=$(curl -s -X POST "${{ env.GOVERNANCE_URL }}:8084/agent" \
              -H "Content-Type: application/json" \
              -d "{\"messages\":[{\"role\":\"user\",\"content\":\"Scan security:\\n\\n$content\"}]}")
            echo "$response"
          done

      - name: Post PR Comment
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `## 🔒 IaC Governance Report
              
              ### Policy Check
              ${{ steps.policy.outputs.result }}
              
              ### Cost Estimation
              ${{ steps.cost.outputs.result }}
              
              ### Security Scan
              ${{ steps.security.outputs.result }}
              `
            })
```

---

## Monitoring & Alerts

### Prometheus Metrics

Each agent exposes metrics at `/metrics`:

```
# Policy Checker
iac_policy_checks_total{status="pass|fail"}
iac_policy_violations_total{rule_id="...",severity="..."}

# Cost Estimator
iac_cost_estimates_total
iac_estimated_monthly_cost{resource_type="..."}

# Drift Detector
iac_drift_detections_total{status="drift|no_drift"}
iac_drifted_resources_total{resource_type="..."}

# General
iac_agent_requests_total{agent="...",status="..."}
iac_agent_request_duration_seconds{agent="..."}
```

### Grafana Dashboard

Import the provided dashboard from `monitoring/grafana/iac-governance-dashboard.json`:

- Overview panel with all agent health status
- Policy violations by severity over time
- Cost trends and budget tracking
- Drift detection history
- Request latency percentiles

### Alert Rules

```yaml
# monitoring/prometheus/alerts.yml
groups:
  - name: iac-governance
    rules:
      - alert: CriticalPolicyViolation
        expr: increase(iac_policy_violations_total{severity="CRITICAL"}[5m]) > 0
        for: 0m
        labels:
          severity: critical
        annotations:
          summary: Critical policy violation detected
          description: "{{ $labels.rule_id }} violation in {{ $labels.repository }}"

      - alert: HighCostEstimate
        expr: iac_estimated_monthly_cost > 10000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: High cost estimate detected
          description: "Estimated monthly cost: ${{ $value }}"

      - alert: DriftDetected
        expr: iac_drifted_resources_total > 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: Infrastructure drift detected
          description: "{{ $value }} resources have drifted from IaC"

      - alert: AgentDown
        expr: up{job=~"iac-.*"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: IaC Governance agent is down
          description: "{{ $labels.job }} has been down for more than 1 minute"
```

---

## Troubleshooting

### Common Issues

#### 1. Agent Not Responding

```bash
# Check if agent is running
curl http://localhost:8081/health

# Check logs
docker logs policy-checker

# Verify port is not in use
netstat -an | grep 8081
```

#### 2. Azure Authentication Errors

```bash
# Verify Azure CLI is logged in
az account show

# Test managed identity (if using)
curl -H "Metadata: true" "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/"

# Check environment variables
echo $AZURE_SUBSCRIPTION_ID
echo $AZURE_TENANT_ID
```

#### 3. Policy Rules Not Loading

```bash
# Verify rules file exists and is valid JSON
cat config/policies/rules.json | jq .

# Check agent logs for parsing errors
docker logs policy-checker 2>&1 | grep -i "error"
```

#### 4. Cost Estimation Returns Zero

```bash
# Check SKU mappings
cat config/sku-mappings/mappings.json | jq .

# Verify Azure Retail Prices API is accessible
curl "https://prices.azure.com/api/retail/prices?api-version=2023-01-01-preview&\$filter=serviceName eq 'Storage'"

# Check for rate limiting
curl -I "https://prices.azure.com/api/retail/prices"
```

#### 5. Notifications Not Sending

```bash
# Test Teams webhook directly
curl -X POST "$TEAMS_WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d '{"text": "Test message"}'

# Test Slack webhook
curl -X POST "$SLACK_WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d '{"text": "Test message"}'

# Check notification manager logs
docker logs notification-manager
```

### Debug Mode

Enable debug logging by setting environment variable:

```bash
export LOG_LEVEL=debug
```

### Support

For issues and feature requests:
- GitHub Issues: https://github.com/your-org/copilot-iac/issues
- Documentation: https://your-org.github.io/copilot-iac/
- Teams Channel: #iac-governance-support
