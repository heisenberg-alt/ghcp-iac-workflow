# GHCP IaC — GitHub Copilot Extension for IaC Governance

A production-ready **GitHub Copilot Extension** that provides AI-powered Infrastructure as Code governance for Azure. Built as a **multi-agent host** with 10 specialized agents, 12 deterministic analysis rules, and two transports (HTTP/SSE + MCP stdio). Powered by [GitHub Models](https://docs.github.com/en/github-models).

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Agents](#agents)
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
- [Transports & Protocols](#transports--protocols)
- [License](#license)

---

## Features

| Capability | Description |
|-----------|-------------|
| **Multi-Agent Architecture** | 10 specialized agents coordinated by an orchestrator with intent-based routing |
| **IaC Analysis** | Policy, security, and compliance scanning (12 rules) for Terraform & Bicep |
| **Cost Estimation** | Azure resource cost estimation with optimization recommendations |
| **Infrastructure Ops** | Drift detection, environment promotion (dev → staging → prod), notifications |
| **LLM Enhancement** | AI-powered analysis via GitHub Models (`gpt-4.1` / `gpt-4.1-mini`) |
| **Blast Radius** | Risk-weighted impact analysis for infrastructure changes |
| **Dual Transport** | HTTP/SSE for GitHub Copilot Chat + MCP stdio (JSON-RPC 2.0) for IDE integration |

## Architecture

Multi-agent host with an orchestrator that classifies intent and dispatches to specialized agents. Supports two transports: HTTP (SSE) for GitHub Copilot Chat and MCP stdio (JSON-RPC 2.0) for IDE tool integration.

```
                        ┌─── HTTP/SSE ───┐    ┌── MCP stdio ──┐
                        │ POST /agent     │    │ JSON-RPC 2.0  │
                        │ POST /agent/{id}│    │ stdin/stdout  │
                        └────────┬────────┘    └──────┬────────┘
                                 │                    │
                                 ▼                    ▼
                         ┌──────────────────────────────┐
                         │     Host  (Registry +        │
                         │     Dispatcher + Enricher)    │
                         └──────────────┬───────────────┘
                                        ▼
                         ┌──────────────────────────────┐
                         │       Orchestrator Agent      │
                         │   (intent classification)     │
                         └──────────────┬───────────────┘
                                        │
          ┌──────────┬──────────┬───────┼───────┬──────────┬──────────┐
          ▼          ▼          ▼       ▼       ▼          ▼          ▼
       Policy    Security  Compliance  Cost   Drift     Deploy   Notification
       (6 rules) (4 rules) (2 rules) (Azure$)(state)  (promote)  (Teams/Slack)
                                                │
                                          ┌─────┴─────┐
                                          ▼           ▼
                                       Impact      Module
                                    (blast radius) (registry)
```

**HTTP Endpoints:**

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/agent` | Orchestrator endpoint — classifies intent and routes to agents (SSE) |
| `POST` | `/agent/{id}` | Direct agent endpoint — invoke a specific agent by ID (SSE) |
| `GET`  | `/agents` | List all registered agents (JSON) |
| `GET`  | `/health` | Health check — returns status, version, environment, agent count |

## Agents

The orchestrator classifies each request and dispatches to the appropriate agents:

| Agent | ID | Trigger Intents | Description |
|-------|----|-----------------|-------------|
| **Policy** | `policy` | analyze | 6 deterministic rules (HTTPS, RBAC, TLS, blob access, soft-delete, purge protection) |
| **Security** | `security` | analyze | 4 rules (hardcoded secrets, public access, encryption, NSG) |
| **Compliance** | `compliance` | analyze | 2 rules (NIST-SC7 network boundaries, NIST-SC28 encryption at rest) |
| **Impact** | `impact` | analyze | Blast radius and risk-weighted change analysis |
| **Cost** | `cost` | cost | Azure resource cost estimation via Retail Prices API |
| **Drift** | `drift` | ops | Infrastructure state drift detection |
| **Deploy** | `deploy` | ops | Environment promotion (dev → staging → prod) |
| **Notification** | `notification` | ops | Teams/Slack webhook notifications |
| **Module** | `module` | help | Terraform module registry lookups |
| **Orchestrator** | `orchestrator` | (default) | Intent classification + multi-agent coordination |

## Project Structure

```
ghcp-iac-workflow/
├── cmd/
│   └── agent-host/          # Entry point — multi-agent host (HTTP + MCP stdio)
├── agents/                  # Specialized agent packages
│   ├── policy/              # Policy analysis agent (6 rules)
│   ├── security/            # Security scanning agent (4 rules)
│   ├── compliance/          # Compliance auditing agent (NIST)
│   ├── cost/                # Cost estimation agent
│   ├── drift/               # Drift detection agent
│   ├── deploy/              # Deployment promotion agent
│   ├── notification/        # Teams/Slack notification agent
│   ├── impact/              # Blast radius analysis agent
│   ├── module/              # Terraform module registry agent
│   └── orchestrator/        # Intent classification + multi-agent coordination
├── internal/
│   ├── protocol/            # Agent interface, Emitter interface, shared types
│   ├── host/                # Agent registry, dispatcher, request enrichment
│   ├── transport/
│   │   └── mcpstdio/        # MCP stdio adapter (JSON-RPC 2.0 over stdin/stdout)
│   ├── analyzer/            # IaC analysis engine (12 rules: policy, security, compliance)
│   ├── config/              # Environment-based configuration loader
│   ├── llm/                 # GitHub Models API client (streaming + non-streaming)
│   ├── parser/              # Terraform HCL & Bicep parser
│   ├── server/              # HTTP server, SSE writer, middleware
│   └── testkit/             # Test fixtures and characterization tests
├── infra/
│   ├── terraform/           # Azure Container Apps — Terraform modules + env tfvars
│   │   ├── main.tf, variables.tf, resources.tf, outputs.tf
│   │   └── envs/            # dev.tfvars, test.tfvars, prod.tfvars
│   └── bicep/               # Azure Container Apps — Bicep module + env params
│       ├── main.bicep
│       └── envs/            # dev.bicepparam, test.bicepparam, prod.bicepparam
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

# Build the agent-host binary
make build
# Output: ./ghcp-iac

# Run locally
make dev
```

### Run Locally

```bash
# Start in HTTP mode (SSE transport for GitHub Copilot Chat)
make dev
# Agent host starts on http://localhost:8080 with 10 agents registered

# Start in MCP stdio mode (JSON-RPC 2.0 for IDE integration)
make dev-mcp

# Verify it's running (HTTP mode)
curl http://localhost:8080/health
# {"status":"ok","service":"ghcp-iac-agent-host","version":"dev","environment":"dev","agents":10}

# List registered agents
curl http://localhost:8080/agents
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
| `MODEL_NAME` | `gpt-4.1-mini` | GitHub Models LLM model. Auto-overridden to `gpt-4.1` in prod |
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
# Run all tests (175 tests across 20 packages)
make test

# Run only agent tests
make test-agents

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
  -e MODEL_NAME=gpt-4.1 \
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
| Model | gpt-4.1-mini | gpt-4.1-mini | gpt-4.1 |

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

12 deterministic rules organized by category:

### Policy (6 rules)
| Rule | Check |
|------|-------|
| POL-001 | Storage account HTTPS enforcement |
| POL-002 | AKS RBAC enabled |
| POL-003 | TLS 1.2 minimum version |
| POL-004 | No public blob access |
| POL-005 | Key Vault soft delete enabled |
| POL-006 | Key Vault purge protection enabled |

### Security (4 rules)
| Rule | Check |
|------|-------|
| SEC-001 | Hardcoded secrets detection (API keys, passwords, connection strings) |
| SEC-002 | Public network access disabled |
| SEC-004 | Encryption at rest (customer-managed keys) |
| SEC-005 | Overly permissive NSG rules (0.0.0.0/0) |

### Compliance (2 rules)
| Rule | Framework | Check |
|------|-----------|-------|
| NIST-SC7 | NIST 800-53 | Network boundary protection |
| NIST-SC28 | NIST 800-53 | Infrastructure encryption at rest |

---

## Transports & Protocols

### HTTP/SSE (GitHub Copilot Chat)

The `/agent` and `/agent/{id}` endpoints respond with [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events) following the GitHub Copilot Extension protocol:

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

### MCP stdio (JSON-RPC 2.0)

For IDE integration, the agent host supports the [Model Context Protocol](https://modelcontextprotocol.io/) over stdin/stdout:

```bash
# Start in MCP mode
./bin/ghcp-iac-server --transport=stdio
```

Supported JSON-RPC methods:

| Method | Description |
|--------|-------------|
| `initialize` | Returns protocol version and server capabilities |
| `tools/list` | Lists all registered agents as MCP tools |
| `tools/call` | Invokes an agent by name with the given arguments |

---

## Makefile Reference

```bash
make build          # Build agent-host binary to bin/
make dev            # Run agent-host locally (HTTP mode)
make dev-mcp        # Run agent-host locally (MCP stdio mode)
make test           # All tests with race detector
make test-agents    # Agent package tests only
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
