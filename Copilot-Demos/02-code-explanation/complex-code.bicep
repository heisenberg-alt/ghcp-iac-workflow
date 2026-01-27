// =============================================================================
// Complex Code for Explanation Demo (Bicep)
// =============================================================================
// Select sections of this code and ask Copilot to explain them!
// =============================================================================

@description('Network security rules configuration')
param networkRules array = []

@description('Subnet configurations')
param subnets array = [
  {
    name: 'web'
    addressPrefix: '10.0.1.0/24'
    serviceEndpoints: ['Microsoft.Storage', 'Microsoft.KeyVault']
    delegation: null
  }
  {
    name: 'app'
    addressPrefix: '10.0.2.0/24'
    serviceEndpoints: []
    delegation: {
      name: 'appServiceDelegation'
      serviceName: 'Microsoft.Web/serverFarms'
      actions: ['Microsoft.Network/virtualNetworks/subnets/action']
    }
  }
]

// Complex loop with conditionals - Select this and ask Copilot to explain
resource nsg 'Microsoft.Network/networkSecurityGroups@2024-01-01' = {
  name: 'nsg-example'
  location: 'eastus'
  properties: {
    securityRules: [for (rule, i) in networkRules: {
      name: rule.name
      properties: {
        priority: rule.priority
        direction: rule.direction
        access: rule.access
        protocol: rule.protocol
        sourcePortRange: rule.sourcePortRange
        destinationPortRange: rule.destinationPortRange
        sourceAddressPrefix: rule.sourceAddressPrefix
        destinationAddressPrefix: rule.destinationAddressPrefix
      }
    }]
  }
}

// Complex subnet creation with optional delegation - Select this for explanation
// Note: Service endpoints are simplified for demo purposes
resource vnet 'Microsoft.Network/virtualNetworks@2024-01-01' = {
  name: 'vnet-example'
  location: 'eastus'
  properties: {
    addressSpace: {
      addressPrefixes: ['10.0.0.0/16']
    }
    subnets: [for subnet in subnets: {
      name: 'snet-${subnet.name}'
      properties: {
        addressPrefix: subnet.addressPrefix
        delegations: subnet.delegation != null ? [
          {
            name: subnet.delegation.name
            properties: {
              serviceName: subnet.delegation.serviceName
            }
          }
        ] : []
      }
    }]
  }
}

// Complex expression with filtering - Select this for explanation
var webSubnets = filter(subnets, s => contains(s.serviceEndpoints, 'Microsoft.Storage'))

// Union and intersection operations
var allEndpoints = union(subnets[0].serviceEndpoints, subnets[1].serviceEndpoints)

output webSubnetCount int = length(webSubnets)
output allServiceEndpoints array = allEndpoints
