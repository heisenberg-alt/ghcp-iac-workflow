# gh-iac

GitHub CLI Extension for IaC Governance - Powered by GitHub Copilot SDK

## Installation

```bash
# Build the extension
cd gh-iac
go build -o gh-iac .

# Install as gh extension
gh extension install .
```

## Usage

```bash
# Show help
gh iac help

# Check agent status
gh iac status

# Run governance commands
gh iac policy 'resource "azurerm_storage_account" "sa" { name = "test" }'
gh iac cost < main.tf
gh iac security < infrastructure/main.tf
gh iac compliance < main.tf

# Full governance check
gh iac check < main.tf

# Operations
gh iac drift myResourceGroup
gh iac impact "Adding new VM"
gh iac deploy status
gh iac modules search networking

# Pipe from files
cat main.tf | gh iac policy
cat main.tf | gh iac cost

# Use with GitHub Copilot
gh copilot suggest "create secure storage account" | gh iac security
```

## Commands

### Governance
| Command | Aliases | Description |
|---------|---------|-------------|
| `check` | `full`, `governance` | Full governance check (orchestrates all agents) |
| `policy` | `p` | Check against organization policies |
| `cost` | `pricing`, `c` | Estimate Azure resource costs |
| `security` | `scan`, `sec` | Security vulnerability scan |
| `compliance` | `audit` | Compliance audit (CIS, NIST, SOC2) |

### Operations
| Command | Aliases | Description |
|---------|---------|-------------|
| `drift` | `d` | Detect infrastructure drift |
| `impact` | `blast` | Blast radius analysis |
| `deploy` | `promote` | Deployment promotion |
| `modules` | `registry` | Search approved modules |
| `notify` | `alerts` | Notification management |

### Utility
| Command | Aliases | Description |
|---------|---------|-------------|
| `status` | `s` | Show all agent status |
| `help` | `h`, `-h` | Show help |
| `version` | `v`, `-v` | Show version |

## Agent Architecture

This extension connects to 10 specialized governance agents:

| Port | Agent | Purpose |
|------|-------|---------|
| 8081 | Policy Checker | Organization policy validation |
| 8082 | Cost Estimator | Azure pricing estimation |
| 8083 | Drift Detector | Infrastructure drift detection |
| 8084 | Security Scanner | Vulnerability scanning |
| 8085 | Compliance Auditor | Regulatory compliance |
| 8086 | Module Registry | Approved module discovery |
| 8087 | Impact Analyzer | Change impact analysis |
| 8088 | Deploy Promoter | Deployment orchestration |
| 8089 | Notification Manager | Alert management |
| 8090 | Orchestrator | Multi-agent coordination |

## GitHub Copilot Integration

The agents use the GitHub Copilot Extensions protocol (SSE streaming with OpenAI-compatible format), making them compatible with the Copilot ecosystem:

```bash
# Copilot generates code, gh-iac validates it
gh copilot suggest "terraform for azure vm" | gh iac security

# Use in CI/CD with Copilot
gh copilot explain main.tf | gh iac compliance
```

## Requirements

- GitHub CLI (`gh`) installed
- All governance agents running (ports 8081-8090)
- Go 1.21+ (for building)

## License

MIT
