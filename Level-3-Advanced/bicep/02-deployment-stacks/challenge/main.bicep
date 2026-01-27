// =============================================================================
// Level 3.2: Deployment Stacks - Challenge
// =============================================================================
// TODO: Create a Bicep template suitable for Azure Deployment Stacks
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

// TODO: Create a resource group


// TODO: Create a module deployment for a storage account within the resource group


// TODO: Create a module deployment for a key vault within the resource group


// TODO: Add outputs for:
// - resourceGroupId
// - storageAccountId
// - keyVaultId
