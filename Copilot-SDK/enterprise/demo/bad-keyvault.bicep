// Bad Key Vault - Security Issues
// Used for demo purposes

resource kv 'Microsoft.KeyVault/vaults@2023-02-01' = {
  name: 'badkeyvault'
  location: resourceGroup().location
  properties: {
    sku: {
      family: 'A'
      name: 'standard'
    }
    tenantId: subscription().tenantId
    
    // Security Issues
    enableSoftDelete: false
    enablePurgeProtection: false
    publicNetworkAccess: 'Enabled'
    
    accessPolicies: []
  }
  // Missing tags
}
