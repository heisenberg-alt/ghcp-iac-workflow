// =============================================================================
// Key Vault Module for Deployment Stacks
// =============================================================================

@description('Key Vault name')
param keyVaultName string

@description('Location')
param location string

@description('Tags')
param tags object = {}

resource keyVault 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: keyVaultName
  location: location
  properties: {
    sku: {
      family: 'A'
      name: 'standard'
    }
    tenantId: subscription().tenantId
    enableRbacAuthorization: true
    enableSoftDelete: true
    softDeleteRetentionInDays: 7
  }
  tags: tags
}

output id string = keyVault.id
output name string = keyVault.name
output vaultUri string = keyVault.properties.vaultUri
