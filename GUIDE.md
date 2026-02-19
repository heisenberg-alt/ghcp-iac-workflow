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
make build
ENVIRONMENT=dev ENABLE_LLM=true go run ./cmd/agent-host
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
  -e MODEL_NAME=gpt-4.1 \
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

Scans Terraform and Bicep code against 12 built-in rules across three categories:

| Category | Rules | Examples |
|----------|-------|---------|
| **Policy** | 6 | HTTPS enforcement, AKS RBAC, TLS 1.2, no public blob, Key Vault soft delete / purge protection |
| **Security** | 4 | Hardcoded secrets, public network access, encryption at rest, overly permissive NSGs |
| **Compliance** | 2 | NIST 800-53 (network boundaries SC-7, encryption at rest SC-28) |

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
```

### 4. Help

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
| `MODEL_NAME` | `gpt-4.1-mini` | `gpt-4.1` in prod |
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
| `POST` | `/agent` | Orchestrator endpoint — SSE stream response |
| `POST` | `/agent/{id}` | Direct agent endpoint — invoke specific agent by ID |
| `GET` | `/agents` | List all registered agents (JSON) |
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

---

## GitHub Copilot Agent Workflow

This project uses **GitHub Copilot agent workflow** (`.github/agents/`) — a feature that lets you define custom AI agents that run directly in your repository via Copilot. These agents can read code, run commands, make edits, and execute tasks autonomously within the repo context.

### How It Works

Agent definitions live in `.github/agents/` as Markdown files with YAML frontmatter:

```
.github/
  agents/
    test.agent.md     # Test & validation agent
```

Each `.agent.md` file defines:

| Field | Purpose |
|-------|---------|
| `name` | Agent identifier (used to invoke it) |
| `description` | What the agent does — Copilot uses this for routing |
| `tools` | Permitted tools: `shell`, `read`, `search`, `edit`, `task`, `skill`, `web_search`, `web_fetch`, `ask_user` |
| Body (Markdown) | Detailed instructions, context, and examples the agent follows |

### Available Agents

#### `test` — Test & Validation Agent

**File:** `.github/agents/test.agent.md`

Runs the full test and validation suite for the ghcp-iac extension:

- **Unit tests** — `go test ./... -v -race`
- **Build verification** — `go build ./...`
- **Lint checks** — `go vet` + `gofmt`
- **Server smoke test** — Starts the server, hits `/health`, sends a sample Terraform analysis request to `/agent`
- **Rule validation** — Verifies known-insecure IaC triggers the expected findings

**Invoke in Copilot Chat:**

```
@test run the full test suite
@test verify the analysis rules work correctly
@test smoke test the server
```

### Creating New Agents

To add a new agent, create a `.agent.md` file in `.github/agents/`:

```markdown
---
description: <what the agent does — be specific>
name: <agent-name>
tools: ['shell', 'read', 'search', 'edit']
---

# Instructions

<Detailed instructions for the agent>
```

**Example agents you could add:**

| Agent | Purpose |
|-------|---------|
| `review` | Code review agent — runs linters, checks for security issues, validates rule coverage |
| `deploy` | Deployment agent — builds Docker image, runs smoke tests, triggers deploy workflows |
| `docs` | Documentation agent — keeps README, GUIDE, and inline docs in sync with code changes |

### Agent vs. Copilot Extension

| | Agent Workflow (`.github/agents/`) | Copilot Extension (`/agent` endpoint) |
|---|---|---|
| **Runs where** | In the repo via GitHub Copilot | On your deployed server |
| **Access** | Repo code, shell, file editing | User's chat messages + references |
| **Use case** | Dev workflow automation (test, review, deploy) | End-user IaC governance (analyze, cost, ops) |
| **Auth** | GitHub permissions | X-GitHub-Token + webhook signature |
| **Invoke** | `@agent-name <prompt>` in repo | `@ghcp-iac <prompt>` in Copilot Chat |

Both work together: the **agent workflow** handles development tasks (testing, reviewing, deploying), while the **Copilot Extension** serves end users with IaC analysis, cost estimation, and infrastructure ops.
