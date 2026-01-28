# Module Registry Agent Skills

## Overview

The Module Registry Agent manages the organization's approved IaC module catalog, validates module usage, and recommends standardized modules for common infrastructure patterns.

## Skills

### 1. Module Validation (`/agent`)

**Endpoint:** `POST /agent`

**Description:** Validates Terraform module usage against the approved catalog.

**Request Format:**
```json
{
  "messages": [
    {
      "role": "user",
      "content": "Check modules:\n\n```terraform\nmodule \"storage\" {\n  source  = \"registry.terraform.io/Azure/storage/azurerm\"\n  version = \"3.0.0\"\n}\n```"
    }
  ]
}
```

**Capabilities:**
- ✅ Verify modules are from approved sources
- 📌 Check version requirements
- 🚫 Flag deprecated modules
- 💡 Suggest approved alternatives

### 2. List Modules (`/modules`)

**Endpoint:** `GET /modules`

**Description:** Returns all approved modules in the catalog.

**Response:**
```json
{
  "modules": [
    {
      "name": "storage-account",
      "source": "registry.terraform.io/Azure/storage/azurerm",
      "version": "3.0.0",
      "description": "Azure Storage Account with security best practices",
      "category": "storage",
      "approved": true
    }
  ],
  "count": 5
}
```

### 3. Search Modules (`/modules/search`)

**Endpoint:** `GET /modules/search?q=storage&category=storage`

**Parameters:**
- `q` - Search query
- `category` - Filter by category

### 4. Get Recommendations (`/modules/recommend`)

**Endpoint:** `POST /modules/recommend`

**Request:**
```json
{
  "resource_type": "azurerm_storage_account",
  "use_case": "blob storage"
}
```

### 5. Health Check (`/health`)

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "service": "module-registry-agent",
  "total_modules": 10,
  "approved_modules": 8
}
```

## Module Status Types

| Status | Icon | Description |
|--------|------|-------------|
| approved | ✅ | Module is approved for use |
| not_approved | ⚠️ | Module not in approved catalog |
| deprecated | 🚫 | Module is deprecated |
| version_mismatch | 📌 | Version below minimum required |
| unknown_source | ❓ | Source not from approved registry |

## Module Categories

- **storage** - Storage accounts, blob, files
- **containers** - AKS, container instances
- **networking** - VNets, subnets, NSGs
- **security** - Key Vault, managed identities
- **compute** - VMs, scale sets
- **databases** - SQL, Cosmos DB, PostgreSQL

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8086 | Server port |
| ALLOWED_MODULE_SOURCES | registry.terraform.io,github.com/yourorg | Approved sources |
| MODULE_REGISTRY_URL | - | Custom registry URL |

## Custom Module Catalog

Add modules to `data/modules.json`:

```json
{
  "modules": [
    {
      "name": "custom-vnet",
      "source": "github.com/yourorg/terraform-azurerm-vnet",
      "version": "2.0.0",
      "description": "Standard VNet configuration",
      "provider": "azurerm",
      "category": "networking",
      "approved": true,
      "min_version": "1.5.0",
      "inputs": [
        {"name": "address_space", "type": "list(string)", "required": true}
      ],
      "outputs": [
        {"name": "vnet_id", "description": "VNet resource ID"}
      ]
    }
  ]
}
```

## VS Code Copilot Usage

```
@module-registry Check if my modules are approved

@module-registry Recommend a module for AKS

@module-registry Find storage modules
```

## Example Output

```markdown
## Module Validation Results

**Summary:**
- ✅ Approved: 2
- ⚠️ Issues: 1

### ⚠️ `legacy-storage`

- Source: `github.com/old-org/terraform-azure-storage`
- Version: `1.0.0`
- Status: **deprecated**
- Module is deprecated
- 💡 Migrate to: storage-account

**Recommended Replacement:**
```hcl
module "storage-account" {
  source  = "registry.terraform.io/Azure/storage/azurerm"
  version = "3.0.0"
}
```

## Running the Agent

```bash
# Build
go build -o module-registry.exe .

# Run
$env:PORT="8086"
$env:ALLOWED_MODULE_SOURCES="registry.terraform.io,github.com/yourorg"
.\module-registry.exe
```

## Integration with CI/CD

```yaml
- name: Validate Modules
  run: |
    response=$(curl -s -X POST http://localhost:8086/agent \
      -H "Content-Type: application/json" \
      -d '{"messages":[{"role":"user","content":"Check: '"$(cat main.tf)"'"}]}')
    
    if echo "$response" | grep -q "not_approved\|deprecated"; then
      echo "Unapproved or deprecated modules detected!"
      exit 1
    fi
```
