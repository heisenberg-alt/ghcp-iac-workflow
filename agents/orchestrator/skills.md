# Orchestrator Agent Skills

## Overview

The Orchestrator Agent is the central coordinator for the Enterprise IaC Governance Platform. It routes requests to specialized agents, orchestrates multi-agent workflows, and aggregates results.

## Skills

### 1. Orchestration (`/agent`)

**Endpoint:** `POST /agent`

**Commands:**
- `status` - Check all agent connectivity
- `code review` - Parallel code review workflow
- `full analysis` - Comprehensive IaC analysis
- `deploy check` - Pre-deployment validation
- `[agent] [request]` - Route to specific agent

### 2. Agent List (`/agents`)

**Endpoint:** `GET /agents`

Returns status of all registered agents.

### 3. Health Check (`/health`)

## Registered Agents

| Agent | Port | Description |
|-------|------|-------------|
| policy | 8081 | Governance policy validation |
| cost | 8082 | Azure cost estimation |
| drift | 8083 | Configuration drift detection |
| security | 8084 | Security vulnerability scanning |
| compliance | 8085 | Compliance framework auditing |
| module | 8086 | Module registry management |
| impact | 8087 | Blast radius analysis |
| deploy | 8088 | Environment promotion |
| notification | 8089 | Multi-channel notifications |

## Workflows

### Code Review
Runs parallel analysis across:
- Policy Agent (governance)
- Security Agent (vulnerabilities)
- Cost Agent (estimation)
- Module Agent (validation)

### Full Analysis
Sequential comprehensive analysis:
1. Security Scan
2. Policy Check
3. Compliance Audit
4. Cost Estimation
5. Impact Analysis
6. Module Validation

### Deployment Check
Pre-flight validation for deployments:
- Security (no critical issues)
- Policy (all pass)
- Cost (within budget)
- Impact (acceptable)

## Environment Variables

```bash
PORT=8090
POLICY_AGENT_URL=http://localhost:8081
COST_AGENT_URL=http://localhost:8082
DRIFT_AGENT_URL=http://localhost:8083
SECURITY_AGENT_URL=http://localhost:8084
COMPLIANCE_AGENT_URL=http://localhost:8085
MODULE_AGENT_URL=http://localhost:8086
IMPACT_AGENT_URL=http://localhost:8087
DEPLOY_AGENT_URL=http://localhost:8088
NOTIFICATION_AGENT_URL=http://localhost:8089
```

## Running the Agent

```bash
$env:PORT="8090"
.\orchestrator.exe
```

## Architecture

```
                    ┌─────────────────┐
                    │   Orchestrator  │
                    │     (8090)      │
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
   ┌────┴────┐          ┌────┴────┐          ┌────┴────┐
   │ Policy  │          │Security │          │  Cost   │
   │ (8081)  │          │ (8084)  │          │ (8082)  │
   └─────────┘          └─────────┘          └─────────┘
        │                    │                    │
   ┌────┴────┐          ┌────┴────┐          ┌────┴────┐
   │Compliance│         │  Drift  │          │ Impact  │
   │ (8085)  │          │ (8083)  │          │ (8087)  │
   └─────────┘          └─────────┘          └─────────┘
        │                    │                    │
   ┌────┴────┐          ┌────┴────┐          ┌────┴────┐
   │ Module  │          │ Deploy  │          │ Notify  │
   │ (8086)  │          │ (8088)  │          │ (8089)  │
   └─────────┘          └─────────┘          └─────────┘
```
