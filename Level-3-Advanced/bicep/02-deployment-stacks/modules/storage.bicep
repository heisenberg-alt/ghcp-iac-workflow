// =============================================================================
// Storage Module for Deployment Stacks
// =============================================================================

@description('Storage account name')
param storageAccountName string

@description('Location')
param location string

@description('Tags')
param tags object = {}

resource storageAccount 'Microsoft.Storage/storageAccounts@2023-05-01' = {
  name: storageAccountName
  location: location
  kind: 'StorageV2'
  sku: {
    name: 'Standard_LRS'
  }
  properties: {
    accessTier: 'Hot'
    supportsHttpsTrafficOnly: true
    minimumTlsVersion: 'TLS1_2'
  }
  tags: tags
}

output id string = storageAccount.id
output name string = storageAccount.name
