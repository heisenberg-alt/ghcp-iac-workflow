# Compliance Auditor Agent Skills

## Overview

The Compliance Auditor Agent validates Infrastructure as Code against industry regulatory compliance frameworks including CIS, NIST, SOC2, HIPAA, and PCI-DSS.

## Skills

### 1. Compliance Audit (`/agent`)

**Endpoint:** `POST /agent`

**Description:** Audits Terraform or Bicep code against multiple compliance frameworks simultaneously.

**Request Format:**
```json
{
  "messages": [
    {
      "role": "user",
      "content": "Audit compliance for:\n\n```terraform\nresource \"azurerm_storage_account\" \"example\" {\n  name = \"storage\"\n  enable_https_traffic_only = false\n}\n```"
    }
  ]
}
```

**Response:** Server-Sent Events (SSE) stream with compliance findings

### 2. List Frameworks (`/frameworks`)

**Endpoint:** `GET /frameworks`

**Description:** Returns all supported compliance frameworks.

**Response:**
```json
{
  "frameworks": [
    {
      "id": "CIS",
      "name": "CIS Azure Foundations Benchmark",
      "version": "2.0",
      "controls": 15
    },
    {
      "id": "NIST",
      "name": "NIST SP 800-53",
      "version": "Rev 5",
      "controls": 12
    }
  ]
}
```

### 3. Generate Report (`/report`)

**Endpoint:** `POST /report`

**Description:** Generates a detailed compliance report in various formats.

### 4. Health Check (`/health`)

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "service": "compliance-auditor-agent",
  "frameworks": ["CIS", "NIST", "SOC2"],
  "controls": 30
}
```

## Supported Frameworks

### CIS Azure Foundations Benchmark v2.0

| Control ID | Title | Category |
|------------|-------|----------|
| CIS-3.1 | Ensure secure transfer required is enabled | Storage |
| CIS-3.7 | Ensure public access level is disabled | Storage |
| CIS-8.1 | Ensure RBAC is enabled on AKS clusters | Containers |

### NIST SP 800-53 Rev 5

| Control ID | Title | Category |
|------------|-------|----------|
| NIST-SC-8 | Transmission Confidentiality and Integrity | System/Comms |
| NIST-SC-28 | Protection of Information at Rest | System/Comms |
| NIST-AC-6 | Least Privilege | Access Control |

### SOC 2 Type II

| Control ID | Title | Category |
|------------|-------|----------|
| SOC2-CC6.1 | Logical Access Security | Common Criteria |
| SOC2-CC6.7 | Transmission Security | Common Criteria |

## Finding Status

| Status | Meaning |
|--------|---------|
| pass | Control requirements met |
| fail | Control requirements not met |
| not_applicable | Control doesn't apply to resource |

## Compliance Score

The agent calculates a compliance score:

```
Score = (Passing Controls / Total Controls) × 100%
```

Example output:
```
Compliance Score: 85.7% (6/7 controls)
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8085 | Server port |
| COMPLIANCE_FRAMEWORKS | CIS,NIST,SOC2 | Enabled frameworks (comma-separated) |
| DEBUG | false | Enable debug logging |

## Custom Frameworks

Add custom frameworks to `data/frameworks.json`:

```json
{
  "frameworks": [
    {
      "id": "CUSTOM",
      "name": "Company Security Policy",
      "version": "1.0",
      "controls": [
        {
          "id": "CUSTOM-001",
          "title": "All storage must use TLS 1.2",
          "severity": "high",
          "resourceTypes": ["azurerm_storage_account"],
          "checks": [
            {"property": "min_tls_version", "operator": "equals", "value": "TLS1_2"}
          ],
          "remediation": "Set min_tls_version = \"TLS1_2\""
        }
      ]
    }
  ]
}
```

## VS Code Copilot Usage

```
@compliance-auditor Check if my Terraform is CIS compliant

@compliance-auditor Audit against SOC2 requirements
```

## Example Compliance Report

```markdown
## Compliance Audit Report

**Compliance Score: 66.7%** (4/6 controls)

### 📋 CIS Violations

1. 🟠 **[CIS-3.1] Ensure secure transfer required is enabled**
   - Resource: `azurerm_storage_account.example`
   - Category: Storage
   - Property 'enable_https_traffic_only' does not meet compliance requirement
   - 💡 Fix: Set enable_https_traffic_only = true
   - 📖 [Documentation](https://www.cisecurity.org/benchmark/azure)
```

## Running the Agent

```bash
# Build
go build -o compliance-auditor.exe .

# Run with specific frameworks
$env:PORT="8085"
$env:COMPLIANCE_FRAMEWORKS="CIS,NIST,SOC2,HIPAA"
.\compliance-auditor.exe
```

## Integration with CI/CD

```yaml
- name: Compliance Audit
  run: |
    result=$(curl -s -X POST http://localhost:8085/agent \
      -H "Content-Type: application/json" \
      -d '{"messages":[{"role":"user","content":"Audit: '"$(cat main.tf)"'"}]}')
    
    if echo "$result" | grep -q "violation"; then
      echo "Compliance violations found!"
      exit 1
    fi
```
