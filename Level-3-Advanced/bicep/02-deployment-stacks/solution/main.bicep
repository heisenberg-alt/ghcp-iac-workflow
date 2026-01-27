// =============================================================================
// Level 3.2: Deployment Stacks - Solution
// =============================================================================

targetScope = 'subscription'

@description('Environment')
param environment string = 'dev'

@description('Location')
param location string = 'eastus'

@description('Workload name')
param workload string = 'stack'

// Variables
var resourceGroupName = 'rg-${workload}-${environment}'
var commonTags = {
  environment: environment
  project: workload
  managed_by: 'bicep-deployment-stack'
}

// Resource Group
resource rg 'Microsoft.Resources/resourceGroups@2024-03-01' = {
  name: resourceGroupName
  location: location
  tags: commonTags
}

// Storage Account Module
module storage '../modules/storage.bicep' = {
  scope: rg
  name: 'storageDeployment'
  params: {
    storageAccountName: 'st${workload}${environment}001'
    location: location
    tags: commonTags
  }
}

// Key Vault Module
module keyVault '../modules/keyvault.bicep' = {
  scope: rg
  name: 'keyVaultDeployment'
  params: {
    keyVaultName: 'kv-${workload}-${environment}'
    location: location
    tags: commonTags
  }
}

// Outputs
output resourceGroupId string = rg.id
output storageAccountId string = storage.outputs.id
output keyVaultId string = keyVault.outputs.id
