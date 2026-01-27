// =============================================================================
// Hardcoded Values - Refactoring Demo (Bicep)
// =============================================================================
// This code has many hardcoded values.
// Ask Copilot to refactor it to use parameters and variables!
// =============================================================================

// Hardcoded storage account
resource storageAccount 'Microsoft.Storage/storageAccounts@2023-05-01' = {
  name: 'stmyappprod001'
  location: 'eastus'
  kind: 'StorageV2'
  sku: {
    name: 'Standard_GRS'
  }
  properties: {
    accessTier: 'Hot'
    supportsHttpsTrafficOnly: true
    minimumTlsVersion: 'TLS1_2'
  }
  tags: {
    environment: 'production'
    project: 'myapp'
    cost_center: '12345'
    owner: 'platform-team'
  }
}

// Hardcoded virtual network
resource vnet 'Microsoft.Network/virtualNetworks@2024-01-01' = {
  name: 'vnet-myapp-production-eastus'
  location: 'eastus'
  properties: {
    addressSpace: {
      addressPrefixes: ['10.0.0.0/16']
    }
    subnets: [
      {
        name: 'snet-web'
        properties: {
          addressPrefix: '10.0.1.0/24'
        }
      }
      {
        name: 'snet-app'
        properties: {
          addressPrefix: '10.0.2.0/24'
        }
      }
      {
        name: 'snet-data'
        properties: {
          addressPrefix: '10.0.3.0/24'
        }
      }
    ]
  }
  tags: {
    environment: 'production'
    project: 'myapp'
    cost_center: '12345'
    owner: 'platform-team'
  }
}

// Hardcoded Key Vault
resource keyVault 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: 'kv-myapp-prod-001'
  location: 'eastus'
  properties: {
    sku: {
      family: 'A'
      name: 'standard'
    }
    tenantId: subscription().tenantId
    enableRbacAuthorization: true
    enableSoftDelete: true
    softDeleteRetentionInDays: 90
  }
  tags: {
    environment: 'production'
    project: 'myapp'
    cost_center: '12345'
    owner: 'platform-team'
  }
}
