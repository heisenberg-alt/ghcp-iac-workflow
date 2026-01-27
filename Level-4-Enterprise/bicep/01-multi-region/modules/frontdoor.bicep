// =============================================================================
// Front Door Module for Multi-Region Deployment
// =============================================================================

@description('Front Door name')
param frontDoorName string

@description('Web app hostnames for origins')
param webAppHostnames array

@description('Tags')
param tags object = {}

// Front Door Profile
resource frontDoorProfile 'Microsoft.Cdn/profiles@2024-02-01' = {
  name: frontDoorName
  location: 'global'
  sku: {
    name: 'Standard_AzureFrontDoor'
  }
  tags: tags
}

// Front Door Endpoint
resource frontDoorEndpoint 'Microsoft.Cdn/profiles/afdEndpoints@2024-02-01' = {
  parent: frontDoorProfile
  name: 'endpoint-${frontDoorName}'
  location: 'global'
  properties: {
    enabledState: 'Enabled'
  }
}

// Origin Group
resource originGroup 'Microsoft.Cdn/profiles/originGroups@2024-02-01' = {
  parent: frontDoorProfile
  name: 'og-webapps'
  properties: {
    loadBalancingSettings: {
      sampleSize: 4
      successfulSamplesRequired: 3
      additionalLatencyInMilliseconds: 50
    }
    healthProbeSettings: {
      probePath: '/'
      probeRequestType: 'HEAD'
      probeProtocol: 'Https'
      probeIntervalInSeconds: 30
    }
    sessionAffinityState: 'Disabled'
  }
}

// Origins (one per web app)
resource origins 'Microsoft.Cdn/profiles/originGroups/origins@2024-02-01' = [for (hostname, i) in webAppHostnames: {
  parent: originGroup
  name: 'origin-${i}'
  properties: {
    hostName: hostname
    httpPort: 80
    httpsPort: 443
    originHostHeader: hostname
    priority: 1
    weight: 1000
    enabledState: 'Enabled'
  }
}]

// Route
resource route 'Microsoft.Cdn/profiles/afdEndpoints/routes@2024-02-01' = {
  parent: frontDoorEndpoint
  name: 'route-default'
  properties: {
    originGroup: {
      id: originGroup.id
    }
    supportedProtocols: ['Http', 'Https']
    patternsToMatch: ['/*']
    forwardingProtocol: 'HttpsOnly'
    linkToDefaultDomain: 'Enabled'
    httpsRedirect: 'Enabled'
  }
  dependsOn: [origins]
}

output endpoint string = 'https://${frontDoorEndpoint.properties.hostName}'
output profileId string = frontDoorProfile.id
