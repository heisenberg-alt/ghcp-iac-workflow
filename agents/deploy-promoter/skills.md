# Deploy Promoter Agent Skills

## Overview

The Deploy Promoter Agent manages environment promotion workflows, ensuring safe progression of infrastructure changes through dev → staging → production.

## Skills

### 1. Promotion Management (`/agent`)

**Endpoint:** `POST /agent`

**Description:** Process promotion requests between environments.

**Commands:**
- `promote dev to staging` - Promote from dev to staging
- `promote staging to prod` - Promote to production
- `status` - Show deployment status across environments
- `rollback [env]` - Initiate rollback procedure

### 2. Health Check (`/health`)

**Response:**
```json
{
  "status": "healthy",
  "service": "deploy-promoter-agent"
}
```

### 3. Deployment Status (`/status`)

Returns current deployment versions across all environments.

## Environment Levels

| Environment | Level | Requires Approval | Approvers |
|-------------|-------|-------------------|-----------|
| dev | 1 | No | - |
| staging | 2 | Yes | lead |
| prod | 3 | Yes | lead, manager |

## Promotion Rules

1. **Sequential Only:** Must promote through each level (no skipping)
2. **Source Required:** Must have deployed version in source environment
3. **Approval Gates:** Higher environments require explicit approval
4. **Version Tracking:** All promotions are version-tracked

## Pre-flight Checks

| Check | Description |
|-------|-------------|
| Source Deployment | Verifies source has deployable version |
| Environment Order | Ensures promotion goes to higher level |
| Approval Required | Lists required approvers |
| Sequential Promotion | Prevents environment skipping |

## Running the Agent

```bash
$env:PORT="8088"
$env:GITHUB_TOKEN="ghp_xxx"
.\deploy-promoter.exe
```
