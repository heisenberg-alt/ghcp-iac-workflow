# 💰 Demo 5: Cost Estimator Agent

> **Difficulty:** ⭐⭐⭐ Advanced | **Time:** 30 minutes

Build a Copilot Agent that estimates Azure resource costs from your Infrastructure as Code using the real **Azure Retail Prices API**.

```
  ┌──────────────────────────────────────────────────────────────┐
  │                   Cost Estimator Agent                        │
  ├──────────────────────────────────────────────────────────────┤
  │                                                               │
  │   User: "Estimate costs for my Terraform AKS cluster"        │
  │         │                                                     │
  │         ▼                                                     │
  │   ┌─────────────────────────────────────────────────────┐    │
  │   │                  Agent Handler                       │    │
  │   │                                                      │    │
  │   │  1. Parse Terraform/Bicep code                      │    │
  │   │  2. Extract resource configurations                 │    │
  │   │  3. Map to Azure pricing SKUs                       │    │
  │   │  4. Query Azure Retail Prices API                   │    │
  │   │  5. Calculate monthly estimates                     │    │
  │   │  6. Stream breakdown to user                        │    │
  │   │                                                      │    │
  │   └──────────────────┬──────────────────────────────────┘    │
  │                      │                                        │
  │                      ▼                                        │
  │   ┌─────────────────────────────────────────────────────┐    │
  │   │           Azure Retail Prices API                    │    │
  │   │       (https://prices.azure.com/api/retail)         │    │
  │   │                                                      │    │
  │   │  • No authentication required!                      │    │
  │   │  • Real-time pricing data                           │    │
  │   │  • All Azure regions and SKUs                       │    │
  │   │                                                      │    │
  │   └─────────────────────────────────────────────────────┘    │
  │                                                               │
  │   Response:                                                   │
  │   ┌─────────────────────────────────────────────────────┐    │
  │   │ 💰 Estimated Monthly Cost: $847.23                  │    │
  │   │                                                      │    │
  │   │ Breakdown:                                           │    │
  │   │ • AKS Cluster (3x Standard_D2s_v3): $312.48        │    │
  │   │ • Storage Account (100GB): $2.30                    │    │
  │   │ • Load Balancer: $18.25                             │    │
  │   │ • Bandwidth (100GB): $8.70                          │    │
  │   │ • ...                                                │    │
  │   └─────────────────────────────────────────────────────┘    │
  │                                                               │
  └──────────────────────────────────────────────────────────────┘
```

## 📋 What You'll Learn

- Azure Retail Prices API (no auth required!)
- Parsing IaC for resource specifications
- SKU mapping for cost calculation
- Building cost breakdown reports
- Streaming detailed results

---

## 🎯 Objectives

1. ✅ Understand Azure Retail Prices API
2. ✅ Parse IaC for cost-relevant properties
3. ✅ Map resources to Azure pricing meters
4. ✅ Calculate monthly cost estimates
5. ✅ Stream itemized cost breakdown

---

## 📚 Azure Retail Prices API

The Azure Retail Prices API is **free and requires no authentication**!

**Base URL:** `https://prices.azure.com/api/retail/prices`

### Example Query

```bash
# Get pricing for Standard_D2s_v3 VMs in East US
curl "https://prices.azure.com/api/retail/prices?\$filter=serviceName eq 'Virtual Machines' and armSkuName eq 'Standard_D2s_v3' and armRegionName eq 'eastus' and priceType eq 'Consumption'"
```

### Response Structure

```json
{
  "Items": [
    {
      "currencyCode": "USD",
      "retailPrice": 0.096,
      "unitOfMeasure": "1 Hour",
      "armRegionName": "eastus",
      "productName": "Virtual Machines Dsv3 Series",
      "skuName": "D2s v3",
      "serviceName": "Virtual Machines"
    }
  ]
}
```

---

## 🛠️ Prerequisites

- Go 1.21+
- Internet access (for Azure Pricing API)
- ngrok for exposing the agent
- GitHub App configured for Agent type

---

## 📂 Project Structure

```
04-cost-estimator/
├── go.mod
├── go.sum
├── main.go                # HTTP server & agent handler
├── agent/
│   ├── handler.go         # Copilot event handling
│   └── parser.go          # IaC parsing
├── pricing/
│   ├── client.go          # Azure Retail Prices API client
│   ├── calculator.go      # Cost calculation logic
│   └── skumap.go          # SKU mapping helpers
└── data/
    └── sku-mappings.json  # Resource to SKU mappings
```

---

## 📝 Running the Agent

```bash
cd Copilot-SDK/04-cost-estimator

# Build and run
go mod tidy
go run .

# Server starts on :8080
```

In another terminal:

```bash
ngrok http 8080
```

---

## 🔧 Supported Resources

| Resource Type | Pricing Factors |
|---------------|-----------------|
| AKS Cluster | Node count, VM size, region |
| Virtual Machines | VM size, OS, region |
| Storage Account | Tier, redundancy, capacity |
| SQL Database | DTU/vCore, storage |
| App Service | Plan tier, instance count |
| Functions | Execution count, memory |
| Key Vault | Operations, secrets |
| Virtual Network | Peering, gateway |
| Load Balancer | Rules, data processed |
| Container Registry | SKU, storage |

---

## 🎮 Usage Examples

### In Copilot Chat

```
@cost-estimator Estimate the cost of this Terraform:
resource "azurerm_kubernetes_cluster" "aks" {
  name                = "myaks"
  location            = "eastus"
  default_node_pool {
    name       = "system"
    node_count = 3
    vm_size    = "Standard_D2s_v3"
  }
}

@cost-estimator How much will 5 Standard_D4s_v3 VMs cost per month in West Europe?

@cost-estimator Compare costs between Standard_LRS and Standard_GRS storage
```

---

## 📊 Cost Calculation Logic

```
┌─────────────────────────────────────────────────────────┐
│               Cost Calculation Flow                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  1. Parse IaC                                           │
│     └── Extract: resource type, SKU, count, region      │
│                                                          │
│  2. Map to Azure Meters                                 │
│     └── resource_type + SKU → API query parameters     │
│                                                          │
│  3. Query Pricing API                                   │
│     └── GET retail/prices with filters                 │
│                                                          │
│  4. Calculate Costs                                     │
│     └── hourly_price × 730 hours × count               │
│                                                          │
│  5. Apply Adjustments                                   │
│     └── Storage: per GB/month                          │
│     └── Bandwidth: per GB transferred                  │
│     └── Operations: per 10K transactions               │
│                                                          │
│  6. Generate Report                                     │
│     └── Itemized breakdown + total                     │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 💡 Pricing Notes

- **VM pricing** is per hour (multiply by 730 for monthly)
- **Storage** is per GB per month
- **Bandwidth** outbound is charged, inbound is free
- **Reserved instances** offer 30-70% savings
- **Prices vary by region**

---

## 🧪 Testing

```bash
# Test the pricing API directly
curl "https://prices.azure.com/api/retail/prices?\$filter=serviceName eq 'Storage' and armSkuName eq 'Standard_LRS' and armRegionName eq 'eastus'"

# Test the agent endpoint
curl -X POST http://localhost:8080/agent \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "Estimate cost for 3 Standard_D2s_v3 VMs in eastus"
    }]
  }'
```

---

## ✅ Completion Checklist

- [ ] Implemented Azure Retail Prices API client
- [ ] Created resource-to-SKU mapping
- [ ] Built cost calculation logic
- [ ] Implemented streaming agent handler
- [ ] Tested with various resource types
- [ ] Verified pricing accuracy

---

## 🏆 All Demos Complete!

Congratulations! You've completed all 5 Copilot SDK demos:

| Demo | Status | Achievement |
|------|--------|-------------|
| 1. GitHub MCP Server | ✅ | 🟢 MCP Rookie |
| 2. IaC Validator MCP | ✅ | 🔵 Server Builder |
| 3. IaC Skillset | ✅ | 🟣 Skillset Creator |
| 4. Policy Agent | ✅ | 🟠 Agent Developer |
| 5. Cost Estimator | ✅ | 🏅 **Full Stack AI** |

---

<div align="center">

**🏆 Achievement Unlocked: Full Stack AI 🏅**

[← Back to Copilot SDK](../README.md) | [Back to Main Lab →](../../README.md)

</div>
