// =============================================================================
// Web App Module for Multi-Region Deployment
// =============================================================================

@description('Web app name')
param appName string

@description('Location')
param location string

@description('Tags')
param tags object = {}

// App Service Plan
resource appServicePlan 'Microsoft.Web/serverfarms@2023-12-01' = {
  name: 'asp-${appName}'
  location: location
  sku: {
    name: 'B1'
    tier: 'Basic'
  }
  kind: 'linux'
  properties: {
    reserved: true
  }
  tags: tags
}

// Web App
resource webApp 'Microsoft.Web/sites@2023-12-01' = {
  name: appName
  location: location
  properties: {
    serverFarmId: appServicePlan.id
    siteConfig: {
      linuxFxVersion: 'NODE|18-lts'
      alwaysOn: true
      ftpsState: 'Disabled'
      minTlsVersion: '1.2'
    }
    httpsOnly: true
  }
  tags: tags
}

output id string = webApp.id
output name string = webApp.name
output defaultHostname string = webApp.properties.defaultHostName
