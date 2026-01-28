# Impact Analyzer Agent Skills

## Overview

The Impact Analyzer Agent evaluates the blast radius and dependency impact of infrastructure changes, helping teams understand risks before deployment.

## Skills

### 1. Impact Analysis (`/agent`)

**Endpoint:** `POST /agent`

**Description:** Analyzes blast radius and dependencies for IaC changes.

**Capabilities:**
- 💥 **Blast Radius:** Calculate how many resources are affected
- 🔗 **Dependencies:** Map resource relationships
- ⚠️ **Risk Assessment:** Evaluate change severity
- ⏱️ **Downtime Prediction:** Identify potential service disruption

### 2. Health Check (`/health`)

**Response:**
```json
{
  "status": "healthy",
  "service": "impact-analyzer-agent"
}
```

## Risk Levels

| Level | Icon | Description |
|-------|------|-------------|
| critical | 🔴 | Cluster/DB changes, high blast radius |
| high | 🟠 | Storage, network, security changes |
| medium | 🟡 | Compute, app service changes |
| low | 🟢 | Tags, outputs, low-impact changes |

## Resource Risk Classification

| Resource Type | Risk | Downtime | Data Loss |
|--------------|------|----------|-----------|
| AKS Cluster | Critical | Yes | No |
| SQL Database | High | Yes | Yes |
| Storage Account | High | No | Yes |
| Virtual Network | High | Yes | No |
| Key Vault | High | No | No |
| App Service | Medium | Yes | No |

## Running the Agent

```bash
$env:PORT="8087"
.\impact-analyzer.exe
```
