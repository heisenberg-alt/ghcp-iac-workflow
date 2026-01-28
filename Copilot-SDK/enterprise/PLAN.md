# Enterprise IaC Governance Platform - Implementation Plan

## Executive Summary

Build a team of 10 coordinated AI agents that manage Infrastructure-as-Code end-to-end with enterprise governance, real-time monitoring, and automated notifications. The platform integrates with GitHub Copilot Chat to provide conversational IaC management.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        GitHub Copilot Chat Interface                        │
│                    "Run governance check on my Terraform"                   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ORCHESTRATOR AGENT (8090)                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Request   │  │  Workflow   │  │   Parallel  │  │   Result    │        │
│  │   Router    │→ │  Selector   │→ │  Executor   │→ │ Aggregator  │        │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────────────────────┘
          │           │           │           │           │           │
          ▼           ▼           ▼           ▼           ▼           ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│   Policy    │ │    Cost     │ │    Drift    │ │  Security   │ │ Compliance  │
│   Checker   │ │  Estimator  │ │  Detector   │ │   Scanner   │ │   Auditor   │
│   (8081)    │ │   (8082)    │ │   (8083)    │ │   (8084)    │ │   (8085)    │
│             │ │             │ │             │ │             │ │             │
│ ✓ Policies  │ │ ✓ Azure API │ │ ✓ Resource  │ │ ✓ CVE Check │ │ ✓ CIS       │
│ ✓ Terraform │ │ ✓ SKU Maps  │ │   Graph     │ │ ✓ Secrets   │ │ ✓ NIST      │
│ ✓ Bicep     │ │ ✓ Budgets   │ │ ✓ State     │ │ ✓ Exposure  │ │ ✓ SOC2      │
└─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘
          │           │           │           │           │
          ▼           ▼           ▼           ▼           ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────────┐
│   Module    │ │   Impact    │ │   Deploy    │ │    NOTIFICATION MANAGER     │
│  Registry   │ │  Analyzer   │ │  Promoter   │ │          (8089)             │
│   (8086)    │ │   (8087)    │ │   (8088)    │ │                             │
│             │ │             │ │             │ │  ┌───────┐ ┌───────┐        │
│ ✓ Approved  │ │ ✓ Blast     │ │ ✓ Approvals │ │  │ Teams │ │ Slack │        │
│   Modules   │ │   Radius    │ │ ✓ Gates     │ │  └───────┘ └───────┘        │
│ ✓ Versions  │ │ ✓ Deps      │ │ ✓ Promote   │ │  ┌───────┐ ┌───────┐        │
│ ✓ Registry  │ │ ✓ Rollback  │ │   Envs      │ │  │ Email │ │Webhook│        │
└─────────────┘ └─────────────┘ └─────────────┘ │  └───────┘ └───────┘        │
                                                └─────────────────────────────┘
```

---

## Agent Specifications

### 1. Orchestrator Agent (Port 8090)
**Status:** 🔴 Not Started

**Purpose:** Central coordinator that routes requests to appropriate agents and manages multi-agent workflows.

**Capabilities:**
- Intent detection from natural language
- Workflow selection (full-check, pre-deploy, promote, drift-scan)
- Parallel agent execution with timeout handling
- Result aggregation and unified reporting
- Conversation context management

**Key Endpoints:**
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/agent` | POST | Main Copilot Extension endpoint |
| `/workflow/full-check` | POST | Run all governance checks |
| `/workflow/pre-deploy` | POST | Pre-deployment validation |
| `/workflow/promote` | POST | Environment promotion |
| `/health` | GET | Health check |

---

### 2. Policy Checker Agent (Port 8081)
**Status:** ✅ Implemented

**Purpose:** Validates IaC against organization policy rules.

**Capabilities:**
- Terraform HCL parsing
- Bicep template parsing (with property extraction)
- Configurable policy rules (JSON)
- Severity levels (CRITICAL, HIGH, MEDIUM, LOW)
- Remediation suggestions

**Existing Code:** `Copilot-SDK/03-policy-agent/`

---

### 3. Cost Estimator Agent (Port 8082)
**Status:** ✅ Implemented

**Purpose:** Estimates Azure resource costs using Retail Prices API.

**Capabilities:**
- Azure Retail Prices API integration
- SKU mapping for resource types
- Monthly cost projections
- Budget threshold alerts

**Existing Code:** `Copilot-SDK/04-cost-estimator/`

---

### 4. Drift Detector Agent (Port 8083)
**Status:** 🔴 Not Started

**Purpose:** Detects configuration drift between IaC definitions and actual Azure resources.

**Capabilities:**
- Azure Resource Graph queries
- Terraform state comparison
- Property-level diff detection
- Drift severity classification
- Scheduled drift scans (cron)

**Key Endpoints:**
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/agent` | POST | Detect drift from IaC |
| `/scan` | POST | Full subscription scan |
| `/report` | GET | Latest drift report |

**Azure APIs Required:**
- Azure Resource Graph API
- Azure Management API

---

### 5. Security Scanner Agent (Port 8084)
**Status:** 🔴 Not Started

**Purpose:** Scans IaC for security vulnerabilities and misconfigurations.

**Capabilities:**
- Secret detection (API keys, passwords, tokens)
- Public exposure checks (public IPs, open ports)
- Encryption validation (at-rest, in-transit)
- Network security rule analysis
- CVE checking for container images

**Security Rules Categories:**
- Secrets exposure
- Network security
- Encryption settings
- Identity & access
- Logging & monitoring

---

### 6. Compliance Auditor Agent (Port 8085)
**Status:** 🔴 Not Started

**Purpose:** Audits IaC against regulatory compliance frameworks.

**Supported Frameworks:**
- CIS Azure Foundations Benchmark v2.0
- NIST 800-53
- SOC 2 Type II
- ISO 27001
- PCI-DSS (future)
- HIPAA (future)

**Capabilities:**
- Framework-specific control mapping
- Compliance score calculation
- Gap analysis reports
- Evidence collection for audits

---

### 7. Module Registry Agent (Port 8086)
**Status:** 🔴 Not Started

**Purpose:** Manages approved IaC modules and enforces module usage policies.

**Capabilities:**
- Approved module catalog
- Version management
- Module usage validation
- Deprecated module detection
- Internal registry integration

**Key Endpoints:**
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/modules` | GET | List approved modules |
| `/modules/{name}` | GET | Get module details |
| `/validate` | POST | Validate module usage |

---

### 8. Impact Analyzer Agent (Port 8087)
**Status:** 🔴 Not Started

**Purpose:** Analyzes the blast radius and downstream impacts of IaC changes.

**Capabilities:**
- Dependency graph construction
- Blast radius calculation
- Affected resource identification
- Rollback planning
- Change risk scoring

**Analysis Types:**
- Direct resource impacts
- Dependent service impacts
- Data flow impacts
- Network topology impacts

---

### 9. Deploy Promoter Agent (Port 8088)
**Status:** 🔴 Not Started

**Purpose:** Manages controlled promotion of IaC across environments.

**Capabilities:**
- Environment gates (dev → staging → prod)
- Approval workflow integration
- GitHub PR automation
- Deployment history tracking
- Rollback triggers

**Promotion Pipeline:**
```
dev → [Tests Pass] → staging → [Approvals] → prod
        ↓                         ↓
    Auto-promote           Manual approval
                           (min 2 reviewers)
```

---

### 10. Notification Manager Agent (Port 8089)
**Status:** 🔴 Not Started

**Purpose:** Sends governance alerts and reports via multiple channels.

**Supported Channels:**
- Microsoft Teams (Adaptive Cards)
- Slack (Block Kit)
- Email (SMTP)
- Webhooks (generic)

**Notification Types:**
- Policy violations (real-time)
- Cost threshold alerts
- Drift detection alerts
- Compliance failures
- Deployment status updates
- Scheduled reports

---

## Implementation Phases

### Phase 1: Foundation (Week 1-2)
- [x] Policy Checker Agent
- [x] Cost Estimator Agent
- [ ] Shared framework package (`pkg/`)
  - HTTP client utilities
  - SSE response helpers
  - Logging & metrics
  - Health check handlers
- [ ] Orchestrator Agent (basic routing)

### Phase 2: Security & Compliance (Week 3-4)
- [ ] Security Scanner Agent
- [ ] Compliance Auditor Agent
- [ ] Notification Manager Agent (Teams, Slack)
- [ ] Orchestrator workflows (full-check)

### Phase 3: Operations (Week 5-6)
- [ ] Drift Detector Agent
- [ ] Impact Analyzer Agent
- [ ] Module Registry Agent
- [ ] Notification Manager (Email, Webhooks)

### Phase 4: Deployment & CI/CD (Week 7-8)
- [ ] Deploy Promoter Agent
- [ ] GitHub Actions integration
- [ ] Docker Compose deployment
- [ ] Kubernetes Helm charts

### Phase 5: Production Hardening (Week 9-10)
- [ ] Azure Container Apps deployment
- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Alert rules
- [ ] Documentation & training

---

## Directory Structure

```
Copilot-SDK/enterprise/
├── PLAN.md                          # This file
├── DEPLOYMENT-GUIDE.md              # Deployment documentation
├── pkg/                             # Shared Go packages
│   ├── agent/                       # Agent base framework
│   │   ├── agent.go                 # Base agent struct
│   │   ├── sse.go                   # SSE response helpers
│   │   └── health.go                # Health check handler
│   ├── parser/                      # IaC parsers
│   │   ├── terraform.go             # Terraform HCL parser
│   │   └── bicep.go                 # Bicep parser
│   ├── azure/                       # Azure API clients
│   │   ├── pricing.go               # Retail Prices API
│   │   ├── graph.go                 # Resource Graph API
│   │   └── management.go            # Management API
│   └── notify/                      # Notification clients
│       ├── teams.go                 # Teams webhook
│       ├── slack.go                 # Slack webhook
│       └── email.go                 # SMTP client
├── agents/
│   ├── orchestrator/                # Orchestrator Agent
│   │   ├── main.go
│   │   ├── router.go
│   │   ├── workflows.go
│   │   └── go.mod
│   ├── policy-checker/              # Policy Checker (existing)
│   │   ├── main.go
│   │   ├── data/rules.json
│   │   └── go.mod
│   ├── cost-estimator/              # Cost Estimator (existing)
│   │   ├── main.go
│   │   ├── data/sku-mappings.json
│   │   └── go.mod
│   ├── drift-detector/              # Drift Detector
│   │   ├── main.go
│   │   ├── graph.go
│   │   └── go.mod
│   ├── security-scanner/            # Security Scanner
│   │   ├── main.go
│   │   ├── data/security-rules.json
│   │   └── go.mod
│   ├── compliance-auditor/          # Compliance Auditor
│   │   ├── main.go
│   │   ├── data/cis-azure.json
│   │   ├── data/nist-800-53.json
│   │   └── go.mod
│   ├── module-registry/             # Module Registry
│   │   ├── main.go
│   │   ├── data/approved-modules.json
│   │   └── go.mod
│   ├── impact-analyzer/             # Impact Analyzer
│   │   ├── main.go
│   │   └── go.mod
│   ├── deploy-promoter/             # Deploy Promoter
│   │   ├── main.go
│   │   └── go.mod
│   └── notification-manager/        # Notification Manager
│       ├── main.go
│       └── go.mod
├── config/                          # Configuration files
│   ├── policies/                    # Policy rules
│   ├── compliance/                  # Compliance frameworks
│   ├── security/                    # Security rules
│   └── modules/                     # Approved modules
├── deploy/                          # Deployment configs
│   ├── docker-compose.yml
│   ├── helm/
│   │   └── iac-governance/
│   └── azure-container-apps/
├── monitoring/                      # Monitoring configs
│   ├── prometheus/
│   └── grafana/
├── scripts/                         # Build & utility scripts
│   ├── build-all.sh
│   ├── build-all.ps1
│   ├── start-local.ps1
│   └── test-all.sh
└── .github/
    └── workflows/
        └── iac-governance.yml       # GitHub Actions workflow
```

---

## Workflows

### 1. Full Governance Check

Triggered by: `"Run full governance check on this Terraform"`

```
┌──────────────┐
│ Orchestrator │
└──────┬───────┘
       │ parallel
       ├─────────────────┬─────────────────┬─────────────────┬─────────────────┐
       ▼                 ▼                 ▼                 ▼                 ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│    Policy    │  │     Cost     │  │   Security   │  │  Compliance  │  │    Module    │
│    Checker   │  │   Estimator  │  │   Scanner    │  │   Auditor    │  │   Registry   │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │                 │                 │
       └─────────────────┴─────────────────┴─────────────────┴─────────────────┘
                                           │
                                           ▼
                                  ┌──────────────────┐
                                  │  Aggregate Results │
                                  │  • Pass/Fail      │
                                  │  • Violations     │
                                  │  • Cost Estimate  │
                                  │  • Compliance %   │
                                  └────────┬─────────┘
                                           │
                                           ▼
                                  ┌──────────────────┐
                                  │   Notification   │
                                  │    Manager       │
                                  └──────────────────┘
```

### 2. Pre-Deployment Validation

Triggered by: Pull Request to `main` branch

```
PR Opened → Policy Check → Security Scan → Cost Check → Compliance Audit
                ↓              ↓              ↓              ↓
           [PASS/FAIL]    [PASS/FAIL]    [< Budget?]   [PASS/FAIL]
                                              ↓
                                        Impact Analysis
                                              ↓
                                    Post PR Comment with Results
```

### 3. Environment Promotion

Triggered by: `"Promote staging to production"`

```
Request → Validate All Checks Pass → Get Approvals → Create PR → Deploy
              ↓                          ↓              ↓
         Policy ✓                  2+ Approvers     Auto-merge
         Security ✓                                     ↓
         Compliance ✓                              Notify Teams
```

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Policy check latency | < 5 seconds |
| Cost estimation latency | < 10 seconds |
| Full governance check latency | < 30 seconds |
| Agent availability | 99.9% |
| False positive rate | < 5% |
| Developer adoption | > 80% of IaC PRs |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Agent timeout | Circuit breaker pattern, graceful degradation |
| Azure API rate limits | Caching, request batching |
| False positives | Configurable rule thresholds, exception lists |
| Secret exposure | No secrets in logs, Key Vault integration |
| Network partition | Retry with backoff, health checks |

---

## Next Steps

1. **Immediate:** Create shared `pkg/` framework
2. **This Week:** Implement Orchestrator Agent
3. **Next Week:** Build Security Scanner and Notification Manager
4. **Ongoing:** Refine policies based on team feedback

---

## References

- [GitHub Copilot Extensions Documentation](https://docs.github.com/en/copilot/building-copilot-extensions)
- [@copilot-extensions/preview-sdk](https://github.com/copilot-extensions/preview-sdk.js) v5.0.1
- [Azure Retail Prices API](https://learn.microsoft.com/en-us/rest/api/cost-management/retail-prices)
- [Azure Resource Graph](https://learn.microsoft.com/en-us/azure/governance/resource-graph/)
- [CIS Azure Foundations Benchmark](https://www.cisecurity.org/benchmark/azure)
