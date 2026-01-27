// =============================================================================
// Level 4.3: CI/CD Integration - Solution
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

// Storage Account
resource storageAccount 'Microsoft.Storage/storageAccounts@2023-05-01' = {
  name: 'st${replace(namePrefix, '-', '')}001'
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
  tags: commonTags
}

// App Service Plan
resource appServicePlan 'Microsoft.Web/serverfarms@2023-12-01' = {
  name: 'asp-${namePrefix}'
  location: location
  sku: {
    name: 'B1'
    tier: 'Basic'
  }
  kind: 'linux'
  properties: {
    reserved: true
  }
  tags: commonTags
}

// Web App
resource webApp 'Microsoft.Web/sites@2023-12-01' = {
  name: 'app-${namePrefix}'
  location: location
  properties: {
    serverFarmId: appServicePlan.id
    siteConfig: {
      linuxFxVersion: 'NODE|18-lts'
      alwaysOn: true
      ftpsState: 'Disabled'
      minTlsVersion: '1.2'
      appSettings: [
        {
          name: 'ENVIRONMENT'
          value: environment
        }
        {
          name: 'STORAGE_ACCOUNT_NAME'
          value: storageAccount.name
        }
        {
          name: 'WEBSITE_RUN_FROM_PACKAGE'
          value: '1'
        }
      ]
    }
    httpsOnly: true
  }
  tags: commonTags
}

// Outputs for CI/CD validation
output resourceGroupName string = resourceGroup().name
output storageAccountName string = storageAccount.name
output webAppName string = webApp.name
output webAppUrl string = 'https://${webApp.properties.defaultHostName}'
