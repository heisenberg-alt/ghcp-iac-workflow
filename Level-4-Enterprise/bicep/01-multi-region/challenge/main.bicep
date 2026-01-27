// =============================================================================
// Level 4.1: Multi-Region Deployment - Challenge
// =============================================================================
// TODO: Deploy web apps to multiple regions with Azure Front Door
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

// TODO: Create a resource group for global resources (Front Door)


// TODO: Create resource groups for each region using a loop


// TODO: Create App Service Plans in each region using a module


// TODO: Create Web Apps in each region using a module


// TODO: Create Azure Front Door profile


// TODO: Create Front Door endpoint


// TODO: Create Front Door origin group with health probes


// TODO: Create Front Door origins for each regional web app


// TODO: Create Front Door route


// TODO: Add outputs for:
// - frontDoorEndpoint
// - regionalWebAppUrls
