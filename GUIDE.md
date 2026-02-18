# GHCP IaC — Deployment & Usage Guide

GitHub Copilot Extension for AI-powered Infrastructure as Code governance (Terraform & Bicep) on Azure.

---

## Deployment

### Prerequisites

- Go 1.22+
- Docker 20+ (for containerized deployment)
- Azure CLI 2.50+ (for Azure deployment)
- Terraform 1.5+ (if using Terraform-based deployment)

### Option 1: Local / Binary

```bash
make build-server
ENVIRONMENT=dev ENABLE_LLM=true go run ./cmd/server
# → http://localhost:8080
```

Verify:

```bash
curl http://localhost:8080/health
```

### Option 2: Docker

```bash
docker build -t ghcp-iac .
docker run -d -p 8080:8080 \
  -e ENVIRONMENT=prod \
  -e GITHUB_WEBHOOK_SECRET=<secret> \
  -e MODEL_NAME=gpt-4o \
  ghcp-iac:latest
```

Or with docker-compose:

```bash
docker-compose up -d
```

### Option 3: Azure Container Apps — Terraform

```bash
cd infra/terraform
terraform init
terraform plan -var-file=envs/dev.tfvars    # dev | test | prod
terraform apply -var-file=envs/dev.tfvars
terraform output container_app_fqdn
```

### Option 4: Azure Container Apps — Bicep

```bash
az group create --name rg-ghcp-iac-dev --location eastus
az deployment group create \
  --resource-group rg-ghcp-iac-dev \
  --template-file infra/bicep/main.bicep \
  --parameters infra/bicep/envs/dev.bicepparam   # dev | test | prod
```

### Option 5: CI/CD — GitHub Actions

Push-based pipeline: `develop` → dev → test (auto) → prod (manual approval via GitHub Release).

Required GitHub secrets:

| Secret | Purpose |
|--------|---------|
| `AZURE_CLIENT_ID` | Service principal app ID |
| `AZURE_TENANT_ID` | Azure AD tenant |
| `AZURE_SUBSCRIPTION_ID` | Subscription |
| `GITHUB_WEBHOOK_SECRET` | Copilot webhook verification |

Create GitHub environments (`dev`, `test`, `production`) and set `production` to require reviewer approval.

### Register as Copilot Extension

1. **GitHub** → Settings → Developer settings → GitHub Apps → New
2. Set Webhook URL to `https://<your-fqdn>/agent`
3. Set Webhook secret to match `GITHUB_WEBHOOK_SECRET`
4. Enable **Copilot Chat: Read & Write**
5. Set Agent URL to `https://<your-fqdn>/agent`
6. Install on your org/repos
7. Invoke in Copilot Chat: `@ghcp-iac <your request>`

---

## Features & Usage

### 1. IaC Analysis

Scans Terraform and Bicep code against 17 built-in rules across three categories:

| Category | Rules | Examples |
|----------|-------|---------|
| **Policy** | 6 | HTTPS enforcement, AKS RBAC, TLS 1.2, no public blob, Key Vault soft delete / purge protection |
| **Security** | 5 | Hardcoded secrets, public network access, encryption at rest, overly permissive NSGs |
| **Compliance** | 6 | CIS Azure (storage HTTPS, Key Vault recovery), NIST 800-53 (network boundaries, encryption), SOC 2 (access controls, TLS) |

Each finding includes severity (Critical / High / Medium / Low), blast radius score, and remediation guidance.

**Usage:**

```
@ghcp-iac Analyze this terraform:
```hcl
resource "azurerm_storage_account" "main" {
  enable_https_traffic_only = false
  min_tls_version           = "TLS1_0"
}
```
```

Other prompts: `"check this bicep"`, `"scan for security issues"`, `"audit compliance"`

### 2. Cost Estimation

Estimates monthly Azure costs using the Azure Retail Prices API. Returns per-resource breakdown and optimization suggestions.

**Usage:**

```
@ghcp-iac How much will a Standard_D4s_v3 VM cost per month in eastus?
@ghcp-iac Estimate cost for this infrastructure: <paste code>
```

### 3. Infrastructure Ops

Drift detection, environment promotion (dev → staging → prod), and optional Teams/Slack notifications.

**Usage:**

```
@ghcp-iac Deploy my app to staging
@ghcp-iac Check for drift in production
@ghcp-iac Show environment status
```

### 4. Status

Reports agent version, build info, environment, and feature toggles.

```
@ghcp-iac status
```

### 5. Help

Lists all capabilities with example prompts.

```
@ghcp-iac help
```

---

## Environment Variables

| Variable | Default | Notes |
|----------|---------|-------|
| `PORT` | `8080` | Server port |
| `ENVIRONMENT` | `dev` | `dev` / `test` / `prod` |
| `GITHUB_WEBHOOK_SECRET` | — | Required in prod |
| `MODEL_NAME` | `gpt-4o-mini` | `gpt-4o` in prod |
| `MODEL_ENDPOINT` | `https://models.inference.ai.azure.com` | GitHub Models API |
| `ENABLE_LLM` | `true` | AI-enhanced analysis |
| `ENABLE_COST_API` | `true` | Live Azure pricing |
| `ENABLE_NOTIFICATIONS` | `false` | Teams/Slack webhooks |
| `TEAMS_WEBHOOK_URL` | — | Teams webhook |
| `SLACK_WEBHOOK_URL` | — | Slack webhook |
| `AZURE_SUBSCRIPTION_ID` | — | For cost API / drift |
| `AZURE_TENANT_ID` | — | Azure AD tenant |
| `AZURE_CLIENT_ID` | — | Service principal |
| `AZURE_CLIENT_SECRET` | — | Service principal secret |

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/agent` | Copilot Extension endpoint (SSE stream) |
| `GET` | `/health` | Health check (JSON) |

---

## Build Commands

```bash
make build          # Server + CLI
make build-server   # Server only
make dev            # Run locally (go run)
make test           # Unit tests with race detector
make test-cover     # Tests + coverage
make docker         # Build Docker image
make clean          # Remove artifacts
```
