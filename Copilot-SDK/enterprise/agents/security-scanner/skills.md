# Security Scanner Agent Skills

## Overview

The Security Scanner Agent provides comprehensive security analysis for Infrastructure as Code, detecting vulnerabilities, misconfigurations, and compliance issues.

## Skills

### 1. Security Scan (`/agent`)

**Endpoint:** `POST /agent`

**Description:** Scans Terraform or Bicep code for security vulnerabilities, hardcoded secrets, and misconfigurations.

**Request Format:**
```json
{
  "messages": [
    {
      "role": "user",
      "content": "Scan this Terraform:\n\n```terraform\nresource \"azurerm_storage_account\" \"example\" {\n  name = \"storage\"\n  allow_blob_public_access = true\n}\n```"
    }
  ]
}
```

**Response:** Server-Sent Events (SSE) stream with security findings

**Capabilities:**
- 🔑 **Secret Detection:** Identifies hardcoded passwords, API keys, connection strings, and private keys
- 🌐 **Network Security:** Checks for public access, NSG misconfigurations, and network exposure
- 🔐 **Encryption:** Validates TLS versions, HTTPS enforcement, and data encryption
- 👤 **Access Control:** Checks RBAC, managed identity usage, and authentication settings
- 📝 **Logging:** Validates diagnostic settings and audit logging

### 2. List Security Rules (`/rules`)

**Endpoint:** `GET /rules`

**Description:** Returns all configured security rules.

**Response:**
```json
{
  "rules": [
    {
      "id": "SEC001",
      "title": "Hardcoded Secrets Detected",
      "severity": "critical",
      "category": "secrets",
      "description": "...",
      "remediation": "..."
    }
  ],
  "count": 20
}
```

### 3. Health Check (`/health`)

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "service": "security-scanner-agent",
  "security_rules": 20
}
```

## Security Categories

| Category | Icon | Description |
|----------|------|-------------|
| secrets | 🔑 | Hardcoded credentials and sensitive data |
| network | 🌐 | Network exposure and firewall rules |
| encryption | 🔐 | Data encryption and TLS settings |
| access | 👤 | IAM, RBAC, and authentication |
| logging | 📝 | Audit logging and monitoring |

## Severity Levels

| Severity | Icon | Action Required |
|----------|------|-----------------|
| critical | 🔴 | Immediate fix required - blocks deployment |
| high | 🟠 | Fix before production deployment |
| medium | 🟡 | Should be addressed in sprint |
| low | 🟢 | Nice to have improvement |

## CWE Mappings

The agent maps findings to Common Weakness Enumeration (CWE) identifiers:

- **CWE-798:** Hardcoded Credentials
- **CWE-284:** Improper Access Control
- **CWE-311:** Missing Encryption
- **CWE-319:** Cleartext Transmission
- **CWE-326:** Inadequate Encryption Strength
- **CWE-269:** Improper Privilege Management
- **CWE-778:** Insufficient Logging
- **CWE-693:** Protection Mechanism Failure

## Integration Examples

### GitHub Actions

```yaml
- name: Security Scan
  run: |
    curl -X POST http://localhost:8084/agent \
      -H "Content-Type: application/json" \
      -d '{"messages":[{"role":"user","content":"Scan: '"$(cat main.tf)"'"}]}'
```

### VS Code Copilot

```
@security-scanner Scan this Terraform code for vulnerabilities
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8084 | Server port |
| DEBUG | false | Enable debug logging |
| GITHUB_WEBHOOK_SECRET | - | For webhook signature validation |

### Custom Rules

Add rules to `data/security-rules.json`:

```json
{
  "id": "CUSTOM001",
  "title": "Custom Security Check",
  "severity": "high",
  "category": "access",
  "resourceTypes": ["azurerm_resource_type"],
  "properties": [
    {"property": "property_name", "operator": "equals", "value": true}
  ],
  "remediation": "How to fix this issue",
  "documentation": "https://docs.example.com"
}
```

## Running the Agent

```bash
# Build
go build -o security-scanner.exe .

# Run
$env:PORT="8084"; .\security-scanner.exe
```
