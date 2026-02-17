# GHCP IaC — GitHub Copilot Extension for IaC Governance

A production-ready **GitHub Copilot Extension** that provides AI-powered Infrastructure as Code governance for Azure. It combines 17 deterministic policy, security, and compliance rules with LLM-enhanced analysis via [GitHub Models](https://docs.github.com/en/github-models).

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Clone & Build](#clone--build)
  - [Run Locally](#run-locally)
  - [Try It Out](#try-it-out)
- [Configuration](#configuration)
- [Testing](#testing)
- [Deployment](#deployment)
  - [Option A — Docker (any host)](#option-a--docker-any-host)
  - [Option B — Azure Container Apps with Terraform](#option-b--azure-container-apps-with-terraform)
  - [Option C — Azure Container Apps with Bicep](#option-c--azure-container-apps-with-bicep)
  - [Option D — CI/CD via GitHub Actions (recommended)](#option-d--cicd-via-github-actions-recommended)
- [Register as a GitHub Copilot Extension](#register-as-a-github-copilot-extension)
- [Analysis Rules](#analysis-rules)
- [Copilot Extension SSE Protocol](#copilot-extension-sse-protocol)
- [License](#license)

---

## Features

| Capability | Description |
|-----------|-------------|
| **IaC Analysis** | Security scanning, policy checking, compliance auditing (CIS, NIST, SOC2) for Terraform & Bicep |
| **Cost Estimation** | Azure resource cost estimation with optimization recommendations |
| **Infrastructure Ops** | Drift detection, environment promotion (dev → staging → prod), notifications |
| **LLM Enhancement** | AI-powered analysis via GitHub Models (`gpt-4o` / `gpt-4o-mini`) |
| **Blast Radius** | Risk-weighted impact analysis for infrastructure changes |
| **Intent Router** | LLM + keyword scoring to classify user requests into analyze / cost / ops / status / help |

## Architecture

Single Go binary with 3 specialized analyzers and an LLM-powered intent router:

```
User → GitHub Copilot Chat → GHCP IaC Extension (POST /agent)
                                     │
                           ┌─────────┼─────────┐
                           ▼         ▼         ▼
                     IaC Analyzer  Cost Est.  Infra Ops
                     (17 rules)    (Azure $)  (drift/deploy)
                           │         │         │
                           └─────────┼─────────┘
                                     ▼
                             GitHub Models (LLM)
```

**Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/agent` | Main Copilot Extension endpoint (SSE response) |
| `GET` | `/health` | Health check — returns JSON with status, environment, model info |

## Project Structure

```
ghcp-iac-workflow/
├── cmd/server/              # Entry point (main.go)
├── internal/
│   ├── analyzer/            # IaC analysis engine (17 rules: policy, security, compliance)
│   ├── auth/                # HMAC-SHA256 webhook signature verification
│   ├── config/              # Environment-based configuration loader
│   ├── costestimator/       # Azure cost estimation + Azure Retail Prices API
│   ├── infraops/            # Drift detection, deployment promotion, notifications
│   ├── llm/                 # GitHub Models API client (streaming + non-streaming)
│   ├── parser/              # Terraform HCL & Bicep parser
│   ├── router/              # LLM-powered intent classification with keyword fallback
│   └── server/              # HTTP server, SSE writer, middleware (panic recovery, logging)
├── infra/
│   ├── terraform/           # Azure Container Apps — Terraform modules + env tfvars
│   │   ├── main.tf, variables.tf, resources.tf, outputs.tf
│   │   └── envs/            # dev.tfvars, test.tfvars, prod.tfvars
│   └── bicep/               # Azure Container Apps — Bicep module + env params
│       ├── main.bicep
│       └── envs/            # dev.bicepparam, test.bicepparam, prod.bicepparam
├── .github/workflows/       # CI/CD pipelines
│   ├── ci.yml               # Lint → Test → Build → Docker image
│   ├── deploy-dev.yml       # Auto-deploy on push to develop
│   ├── deploy-test.yml      # Auto-promote after dev, runs smoke tests
│   └── deploy-prod.yml      # Manual approval gate, deploy on GitHub Release
├── Dockerfile               # Multi-stage build (golang:1.22-alpine → alpine:3.19)
├── docker-compose.yml       # Local development with all env vars
├── Makefile                 # Build automation (build, test, lint, dev, docker, clean)
└── go.mod                   # Go module (only external dep: google/uuid)
```

---

## Getting Started

### Prerequisites

| Tool | Version | Required |
|------|---------|----------|
| **Go** | 1.22+ | Yes |
| **Docker** | 20+ | For containerized runs |
| **Terraform** | 1.5+ | For Azure deployment (Option B) |
| **Azure CLI** | 2.50+ | For Azure deployment (Options B/C) |
| **Git** | any | Yes |

### Clone & Build

```bash
# Clone the repository
git clone https://github.com/<your-org>/ghcp-iac-workflow.git
cd ghcp-iac-workflow

# Download dependencies
go mod download

# Build the server binary
make build-server
# Output: bin/ghcp-iac-server

# Or build everything (server + CLI)
make build
```

### Run Locally

```bash
# Start in dev mode (no webhook secret required, uses gpt-4o-mini)
make dev
# Server starts on http://localhost:8080

# Verify it's running
curl http://localhost:8080/health
# {"status":"healthy","environment":"dev","llm_enabled":true,"model":"gpt-4o-mini","service":"ghcp-iac"}
```

### Try It Out

Send a Terraform snippet for analysis:

```bash
curl -s -X POST http://localhost:8080/agent \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "Analyze this terraform:\n```hcl\nresource \"azurerm_storage_account\" \"example\" {\n  name                      = \"mystorage\"\n  resource_group_name       = \"rg-test\"\n  location                  = \"eastus\"\n  account_tier              = \"Standard\"\n  account_replication_type  = \"LRS\"\n  enable_https_traffic_only = false\n  min_tls_version           = \"TLS1_0\"\n}\n```"
    }]
  }'
```

The response is a stream of [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events) with findings organized by category (Policy, Security, Compliance), severity counts, and blast radius.

**Other example prompts you can try:**

```bash
# Cost estimation
"How much will a Standard_D4s_v3 VM cost per month in eastus?"

# Infrastructure ops
"Deploy my app to staging"

# Drift detection
"Check for drift in production"

# Help
"What can you do?"
```

---

## Configuration

All configuration is via environment variables. The server uses sensible defaults for local development.

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `ENVIRONMENT` | `dev` | Environment name: `dev`, `test`, or `prod` |
| `GITHUB_WEBHOOK_SECRET` | — | HMAC secret for Copilot webhook signature verification. **Required in prod** — requests are rejected without it |
| `MODEL_NAME` | `gpt-4o-mini` | GitHub Models LLM model. Auto-overridden to `gpt-4o` in prod |
| `MODEL_ENDPOINT` | `https://models.inference.ai.azure.com` | GitHub Models API endpoint |
| `ENABLE_LLM` | `true` | Enable AI-enhanced analysis and intent routing |
| `ENABLE_COST_API` | `true` | Enable live Azure Retail Prices API for cost estimation |
| `ENABLE_NOTIFICATIONS` | `false` | Enable Teams/Slack notification webhooks |
| `TEAMS_WEBHOOK_URL` | — | Microsoft Teams incoming webhook URL |
| `SLACK_WEBHOOK_URL` | — | Slack incoming webhook URL |
| `AZURE_SUBSCRIPTION_ID` | — | Azure subscription (for cost API and drift detection) |
| `AZURE_TENANT_ID` | — | Azure AD tenant ID |
| `AZURE_CLIENT_ID` | — | Azure service principal client ID |
| `AZURE_CLIENT_SECRET` | — | Azure service principal client secret |
| `LOG_LEVEL` | `debug` | Log verbosity |

---

## Testing

```bash
# Run all unit tests (79 tests across 6 packages)
make test

# Run with coverage report
make test-cover

# Run with HTML coverage report (opens in browser)
make test-cover-html

# Static analysis
make vet           # go vet
make lint          # golangci-lint (install separately)
make fmt           # format all Go files
```

---

## Deployment

### Option A — Docker (any host)

The simplest way to deploy anywhere (Azure, AWS, GCP, on-prem).

```bash
# 1. Build the Docker image
make docker
# or: docker build -t ghcp-iac .

# 2. Run with environment variables
docker run -d \
  --name ghcp-iac \
  -p 8080:8080 \
  -e ENVIRONMENT=prod \
  -e GITHUB_WEBHOOK_SECRET=your-webhook-secret \
  -e MODEL_NAME=gpt-4o \
  ghcp-iac:latest

# 3. Verify
curl http://localhost:8080/health
```

**Using docker-compose (recommended for local/dev):**

```bash
# Copy and edit environment variables
cp .env.example .env  # or set vars inline

# Start
docker-compose up -d

# View logs
docker-compose logs -f
```

The Docker image is a minimal 2-layer build:
- **Build stage:** `golang:1.22-alpine` — compiles a static binary with version info
- **Runtime stage:** `alpine:3.19` (~7 MB) — non-root user, healthcheck included

---

### Option B — Azure Container Apps with Terraform

Provisions: Resource Group, Log Analytics, Azure Container Registry (ACR), Container App Environment, and Container App.

```bash
# 1. Navigate to the Terraform directory
cd infra/terraform

# 2. Initialize with your backend (Azure Storage for state)
terraform init \
  -backend-config="resource_group_name=rg-terraform-state" \
  -backend-config="storage_account_name=stterraformstate" \
  -backend-config="container_name=tfstate" \
  -backend-config="key=ghcp-iac-dev.tfstate"

# 3. Plan — review what will be created
terraform plan -var-file=envs/dev.tfvars
# For test: terraform plan -var-file=envs/test.tfvars
# For prod: terraform plan -var-file=envs/prod.tfvars

# 4. Apply
terraform apply -var-file=envs/dev.tfvars

# 5. Get the app URL from outputs
terraform output container_app_fqdn
```

**Environment-specific sizing:**

| Setting | Dev | Test | Prod |
|---------|-----|------|------|
| CPU | 0.25 | 0.5 | 1.0 |
| Memory | 0.5 Gi | 1 Gi | 2 Gi |
| Min replicas | 0 | 1 | 2 |
| Max replicas | 1 | 3 | 5 |
| Model | gpt-4o-mini | gpt-4o-mini | gpt-4o |

---

### Option C — Azure Container Apps with Bicep

Same infrastructure as Terraform, using Azure native Bicep templates.

```bash
# 1. Create the resource group (if it doesn't exist)
az group create --name rg-ghcp-iac-dev --location eastus

# 2. Deploy with environment-specific parameters
az deployment group create \
  --resource-group rg-ghcp-iac-dev \
  --template-file infra/bicep/main.bicep \
  --parameters infra/bicep/envs/dev.bicepparam

# For test:
az deployment group create \
  --resource-group rg-ghcp-iac-test \
  --template-file infra/bicep/main.bicep \
  --parameters infra/bicep/envs/test.bicepparam

# For prod:
az deployment group create \
  --resource-group rg-ghcp-iac-prod \
  --template-file infra/bicep/main.bicep \
  --parameters infra/bicep/envs/prod.bicepparam
```

---

### Option D — CI/CD via GitHub Actions (recommended)

The repo includes 4 GitHub Actions workflows that form a complete promotion pipeline:

```
┌──────────────┐     ┌───────────┐     ┌────────────┐     ┌──────────────┐
│   CI         │────▶│ Deploy    │────▶│ Deploy     │     │ Deploy       │
│ (lint/test/  │     │ Dev       │     │ Test       │     │ Prod         │
│  build/docker)│     │ (auto)    │     │ (auto +    │     │ (approval +  │
│              │     │           │     │  smoke test)│     │  health check)│
└──────────────┘     └───────────┘     └────────────┘     └──────────────┘
  push to            push to           after dev            GitHub Release
  main/develop       develop           succeeds             + manual approval
```

#### Step 1 — Provision Azure infrastructure

Use Terraform (Option B) or Bicep (Option C) to create the Container App infrastructure for each environment (dev, test, prod).

#### Step 2 — Set up Azure OIDC authentication

Create an Azure service principal with federated identity credentials for GitHub Actions:

```bash
# Create service principal
az ad sp create-for-rbac --name "ghcp-iac-github" --role contributor \
  --scopes /subscriptions/<SUBSCRIPTION_ID> --sdk-auth

# Add federated credentials for each environment
az ad app federated-credential create --id <APP_ID> --parameters '{
  "name": "github-develop",
  "issuer": "https://token.actions.githubusercontent.com",
  "subject": "repo:<your-org>/ghcp-iac-workflow:ref:refs/heads/develop",
  "audiences": ["api://AzureADTokenExchange"]
}'
```

#### Step 3 — Configure GitHub repository secrets

Go to **Settings → Secrets and variables → Actions** and add:

| Secret | Description |
|--------|-------------|
| `AZURE_CLIENT_ID` | Service principal application (client) ID |
| `AZURE_TENANT_ID` | Azure AD tenant ID |
| `AZURE_SUBSCRIPTION_ID` | Azure subscription ID |
| `GITHUB_WEBHOOK_SECRET` | Webhook secret for Copilot signature verification (used in test/prod) |

#### Step 4 — Create GitHub environments

Go to **Settings → Environments** and create:

| Environment | Protection Rules |
|-------------|-----------------|
| `dev` | None — auto-deploys on push to `develop` |
| `test` | None — auto-promotes after successful dev deployment |
| `production` | **Required reviewers** — manual approval before prod deploy |

#### Step 5 — Deploy

| Target | Trigger | What happens |
|--------|---------|--------------|
| **Dev** | Push to `develop` branch | CI runs → Docker image built → pushed to ACR → deployed to Container App → health check |
| **Test** | Automatic after dev succeeds | Same build → deploy to test environment → smoke tests (health + 405 check) |
| **Prod** | Create a GitHub Release | CI runs → approval gate waits for reviewer → deploy → health check with auto-rollback on failure |

```bash
# Deploy to dev — just push to develop
git checkout develop
git push origin develop

# Deploy to prod — create a release
gh release create v1.0.0 --title "v1.0.0" --notes "Production release"
# Then approve in the GitHub Actions UI
```

---

## Register as a GitHub Copilot Extension

After deploying the server to a public URL, register it as a Copilot Extension:

1. Go to **GitHub → Settings → Developer settings → GitHub Apps → New GitHub App**
2. Fill in:
   - **App name:** `GHCP IaC` (or your preferred name)
   - **Homepage URL:** your deployment URL
   - **Webhook URL:** `https://<your-container-app-fqdn>/agent`
   - **Webhook secret:** same value as `GITHUB_WEBHOOK_SECRET`
3. Under **Permissions**, enable:
   - **Copilot Chat:** Read & Write
4. Under **Copilot**, set:
   - **Agent URL:** `https://<your-container-app-fqdn>/agent`
   - **Agent description:** "AI-powered IaC governance for Azure (Terraform & Bicep)"
5. Install the app on your organization/repositories
6. In Copilot Chat, invoke with `@ghcp-iac` followed by your request

---

## Analysis Rules

### Policy (6 rules)
| Rule | Check |
|------|-------|
| POL-001 | Storage account HTTPS enforcement |
| POL-002 | AKS RBAC enabled |
| POL-003 | TLS 1.2 minimum version |
| POL-004 | No public blob access |
| POL-005 | Key Vault soft delete enabled |
| POL-006 | Key Vault purge protection enabled |

### Security (5 rules)
| Rule | Check |
|------|-------|
| SEC-001 | Hardcoded secrets detection (API keys, passwords, connection strings) |
| SEC-002 | Public network access disabled |
| SEC-003 | HTTPS traffic enforcement |
| SEC-004 | Encryption at rest (customer-managed keys) |
| SEC-005 | Overly permissive NSG rules (0.0.0.0/0) |

### Compliance (6 rules)
| Rule | Framework | Check |
|------|-----------|-------|
| CIS-4.1 | CIS Azure | Storage HTTPS |
| CIS-8.1 | CIS Azure | Key Vault recovery |
| NIST-SC7 | NIST 800-53 | Network boundary protection |
| NIST-SC28 | NIST 800-53 | Infrastructure encryption at rest |
| SOC2-CC6.1 | SOC 2 | Logical access controls (no public blob) |
| SOC2-CC6.6 | SOC 2 | Encryption in transit (TLS 1.2) |

---

## Copilot Extension SSE Protocol

The `/agent` endpoint responds with [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events) following the GitHub Copilot Extension protocol:

```
event: copilot_message
data: {"choices":[{"index":0,"delta":{"role":"assistant","content":"..."}}]}

event: copilot_references
data: [{"title":"CIS Azure Benchmark","url":"https://www.cisecurity.org/benchmark/azure"}]

event: copilot_confirmation
data: {"title":"Deploy?","message":"Ready to promote to staging"}

event: copilot_done
data: {}
```

Messages are streamed incrementally — each `copilot_message` event contains a chunk of the response. The stream ends with `copilot_done`.

---

## Makefile Reference

```bash
make build          # Build server + CLI binaries
make build-server   # Build server binary only
make dev            # Run server locally (go run)
make test           # Unit tests with race detector
make test-cover     # Tests + coverage report
make test-cover-html # Tests + open HTML coverage in browser
make vet            # go vet
make lint           # golangci-lint
make fmt            # gofmt all files
make docker         # Build Docker image
make docker-run     # Run Docker image (requires .env file)
make clean          # Remove build artifacts
```

## License

MIT
