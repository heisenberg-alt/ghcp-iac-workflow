resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'insecurestorage'
  location: resourceGroup().location
  kind: 'StorageV2'
  sku: {
    name: 'Standard_LRS'
  }
  properties: {
    supportsHttpsTrafficOnly: false
    minimumTlsVersion: 'TLS1_0'
    allowBlobPublicAccess: true
  }
}
