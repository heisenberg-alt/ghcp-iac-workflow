# Notification Manager Agent Skills

## Overview

The Notification Manager Agent handles multi-channel notifications for IaC events including deployments, policy violations, drift detection, and security findings.

## Skills

### 1. Notification Management (`/agent`)

**Endpoint:** `POST /agent`

**Commands:**
- `channels` - List all notification channels
- `rules` - Show routing rules
- `history` - View recent notifications
- `test [channel]` - Send test notification
- `send [type] [severity]` - Send notification

### 2. Direct Notification (`/notify`)

**Endpoint:** `POST /notify`

Send notification directly via API:
```json
{
  "type": "security",
  "severity": "critical",
  "title": "Security Finding",
  "message": "Hardcoded secret detected",
  "resource": "storage.tf"
}
```

### 3. Channels (`/channels`)

**Endpoint:** `GET /channels`

Returns configured notification channels.

### 4. History (`/history`)

**Endpoint:** `GET /history`

Returns notification history.

### 5. Health Check (`/health`)

## Notification Channels

| Channel | Type | Description |
|---------|------|-------------|
| teams-alerts | teams | Microsoft Teams webhook |
| slack-devops | slack | Slack incoming webhook |
| email-admins | email | SMTP email |
| webhook-audit | webhook | Custom audit endpoint |

## Event Types

| Type | Description | Default Severity |
|------|-------------|------------------|
| deployment | Deploy events | info |
| drift | Config drift | warning |
| policy | Policy violations | warning |
| security | Security findings | error |
| cost | Cost alerts | warning |

## Routing Rules

Events are routed to channels based on type and severity:

- **security (all):** teams, email, webhook
- **deployment (error):** teams, slack, email
- **drift (all):** teams, slack
- **policy (error):** teams, email

## Environment Variables

```bash
PORT=8089
TEAMS_WEBHOOK_URL=https://outlook.office.com/webhook/...
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
SMTP_SERVER=smtp.example.com:587
```

## Running the Agent

```bash
$env:PORT="8089"
$env:TEAMS_WEBHOOK_URL="https://..."
.\notification-manager.exe
```
