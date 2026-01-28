# gh iac - Enterprise IaC Governance Workflow

A complete Infrastructure as Code governance platform powered by GitHub Copilot SDK with 10 specialized agents and a custom `gh iac` CLI extension.

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    gh iac CLI Extension                          │
│         (GitHub CLI extension for IaC governance)               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Orchestrator (8090)                         │
│              Central coordinator for all agents                  │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│ Policy (8081) │    │  Cost (8082)  │    │ Drift (8083)  │
│   Checker     │    │  Estimator    │    │  Detector     │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│Security (8084)│    │Compliance(8085│    │ Module (8086) │
│   Scanner     │    │   Auditor     │    │  Registry     │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│Impact (8087)  │    │ Deploy (8088) │    │ Notify (8089) │
│  Analyzer     │    │  Promoter     │    │   Manager     │
└───────────────┘    └───────────────┘    └───────────────┘
```

## 📦 Components

| Component | Port | Description |
|-----------|------|-------------|
| **Policy Checker** | 8081 | Validates IaC against organization policies |
| **Cost Estimator** | 8082 | Estimates Azure resource costs |
| **Drift Detector** | 8083 | Detects infrastructure drift from IaC state |
| **Security Scanner** | 8084 | Scans for security vulnerabilities |
| **Compliance Auditor** | 8085 | Audits against CIS, NIST, SOC2 frameworks |
| **Module Registry** | 8086 | Manages approved IaC modules |
| **Impact Analyzer** | 8087 | Analyzes blast radius of changes |
| **Deploy Promoter** | 8088 | Manages environment promotions |
| **Notification Manager** | 8089 | Sends alerts via Teams/Slack/Email |
| **Orchestrator** | 8090 | Central coordinator for all agents |
| **Dashboard** | 3001 | Real-time status dashboard |

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- GitHub CLI (`gh`)

### 1. Build All Agents

```bash
# Windows
.\scripts\start-all-agents.ps1

# Linux/macOS
./scripts/start-all-agents.sh
```

### 2. Install gh iac CLI Extension

```bash
cd gh-iac
go build -o gh-iac .

# Install as gh extension
mkdir -p "$env:LOCALAPPDATA\GitHub CLI\extensions\gh-iac"
cp gh-iac.exe "$env:LOCALAPPDATA\GitHub CLI\extensions\gh-iac\"
```

### 3. Use the CLI

```bash
# Check all agents status
gh iac status

# Policy validation
gh iac policy "resource azurerm_storage_account..."
cat main.tf | gh iac policy

# Cost estimation
gh iac cost "resource azurerm_virtual_machine..."

# Security scan
gh iac security "resource azurerm_key_vault..."

# Full governance check (uses orchestrator)
gh iac check "your IaC code here"
```

## 📋 CLI Commands

| Command | Description |
|---------|-------------|
| `gh iac help` | Show all available commands |
| `gh iac status` | Check status of all 10 agents |
| `gh iac policy <code>` | Validate IaC against policies |
| `gh iac cost <code>` | Estimate Azure resource costs |
| `gh iac drift` | Detect infrastructure drift |
| `gh iac security <code>` | Scan for vulnerabilities |
| `gh iac compliance <code>` | CIS/NIST/SOC2 audit |
| `gh iac modules [search]` | Search approved modules |
| `gh iac impact <desc>` | Blast radius analysis |
| `gh iac deploy` | Environment promotion |
| `gh iac notify <msg>` | Send notifications |
| `gh iac check <code>` | Full governance check |

## 🔧 Project Structure

```
ghcp-iac-workflow/
├── agents/                    # 10 governance agents
│   ├── policy-checker/        # Port 8081
│   ├── cost-estimator/        # Port 8082
│   ├── drift-detector/        # Port 8083
│   ├── security-scanner/      # Port 8084
│   ├── compliance-auditor/    # Port 8085
│   ├── module-registry/       # Port 8086
│   ├── impact-analyzer/       # Port 8087
│   ├── deploy-promoter/       # Port 8088
│   ├── notification-manager/  # Port 8089
│   └── orchestrator/          # Port 8090
├── gh-iac/                    # GitHub CLI extension
├── dashboard/                 # Real-time status dashboard
├── demo/                      # Demo scripts and sample IaC
└── scripts/                   # Startup scripts
```

## 🎯 GitHub Copilot SDK Integration

All agents implement the GitHub Copilot SDK SSE (Server-Sent Events) protocol:

```go
// SSE Format
event: copilot_message
data: {"content": "Analysis result..."}

event: copilot_done
data: {}
```

This enables seamless integration with GitHub Copilot extensions and the `gh copilot` CLI.

## 📊 Dashboard

Access the real-time dashboard at `http://localhost:3001` to monitor:
- All 10 agent statuses (online/offline)
- Health check responses
- System overview

## 🎬 Demo

Run the demo script to see all agents in action:

```bash
cd demo
.\demo-script.ps1
```

## License

MIT
