// =============================================================================
// Level 3.1: Bicep Modules - Challenge
// =============================================================================
// TODO: Use the provided modules to deploy networking and storage resources
// =============================================================================

@description('Environment')
param environment string = 'dev'

@description('Location')
param location string = 'eastus'

@description('Workload name')
param workload string = 'modular'

// Variables
var commonTags = {
  environment: environment
  project: workload
  managed_by: 'bicep'
}

// TODO: Use the networking module from '../modules/networking.bicep'
// Pass the following parameters:
// - vnetName: constructed from workload and environment
// - location: from parameter
// - addressSpace: ['10.0.0.0/16']
// - subnets: array with web (10.0.1.0/24) and app (10.0.2.0/24) subnets
// - tags: commonTags


// TODO: Use the storage module from '../modules/storage.bicep'
// Pass the following parameters:
// - storageAccountName: constructed from workload and environment
// - location: from parameter
// - sku: 'Standard_LRS'
// - containerNames: ['data', 'logs']
// - tags: commonTags


// TODO: Add outputs for:
// - vnetId from networking module
// - storageAccountName from storage module
