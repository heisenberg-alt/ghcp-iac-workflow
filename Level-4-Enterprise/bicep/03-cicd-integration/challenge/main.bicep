// =============================================================================
// Level 4.3: CI/CD Integration - Challenge
// =============================================================================
// TODO: Create a Bicep template suitable for CI/CD deployment
// =============================================================================

@description('Environment name')
param environment string

@description('Location')
param location string = 'eastus'

@description('Workload name')
param workload string = 'cicd'

// Variables
var namePrefix = '${workload}-${environment}'
var commonTags = {
  environment: environment
  project: workload
  managed_by: 'bicep'
  deployed_by: 'github-actions'
}

// TODO: Create a storage account


// TODO: Create an App Service Plan (Linux, B1)


// TODO: Create a Web App with:
// - Node.js 18 runtime
// - Environment app settings
// - HTTPS only


// TODO: Add outputs for CI/CD validation:
// - resourceGroupName (use resourceGroup().name)
// - storageAccountName
// - webAppName
// - webAppUrl
