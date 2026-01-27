// =============================================================================
// Level 4.1: Multi-Region Deployment - Solution
// =============================================================================

targetScope = 'subscription'

@description('Environment')
param environment string = 'prod'

@description('Workload name')
param workload string = 'multiregion'

@description('Regions for deployment')
param regions array = ['eastus', 'westeurope']

// Variables
var commonTags = {
  environment: environment
  project: workload
  managed_by: 'bicep'
}

// Global Resource Group (for Front Door)
resource rgGlobal 'Microsoft.Resources/resourceGroups@2024-03-01' = {
  name: 'rg-${workload}-global-${environment}'
  location: regions[0]
  tags: commonTags
}

// Regional Resource Groups
resource rgRegional 'Microsoft.Resources/resourceGroups@2024-03-01' = [for region in regions: {
  name: 'rg-${workload}-${region}-${environment}'
  location: region
  tags: commonTags
}]

// Regional Web Apps
module webApps '../modules/webapp.bicep' = [for (region, i) in regions: {
  scope: rgRegional[i]
  name: 'webapp-${region}'
  params: {
    appName: 'app-${workload}-${region}-${environment}'
    location: region
    tags: commonTags
  }
}]

// Front Door
module frontDoor '../modules/frontdoor.bicep' = {
  scope: rgGlobal
  name: 'frontdoor'
  params: {
    frontDoorName: 'fd-${workload}-${environment}'
    webAppHostnames: [for (region, i) in regions: webApps[i].outputs.defaultHostname]
    tags: commonTags
  }
}

// Outputs
output frontDoorEndpoint string = frontDoor.outputs.endpoint
output regionalWebAppUrls array = [for (region, i) in regions: 'https://${webApps[i].outputs.defaultHostname}']
