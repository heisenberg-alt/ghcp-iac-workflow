// =============================================================================
// Level 4.2: Policy as Code - Solution
// =============================================================================

targetScope = 'subscription'

@description('Environment')
param environment string = 'prod'

@description('Allowed Azure locations')
param allowedLocations array = ['eastus', 'eastus2', 'westus2']

@description('Required tags')
param requiredTags array = ['environment', 'project', 'cost_center']

// Variables
var commonTags = {
  environment: environment
  project: 'policy'
  managed_by: 'bicep'
}

// Build policy definitions array for initiative
var locationPolicyRef = [
  {
    policyDefinitionId: policyAllowedLocations.id
    parameters: {
      allowedLocations: {
        value: '[parameters(\'allowedLocations\')]'
      }
    }
  }
]

var tagPolicyRefs = [for tag in requiredTags: {
  policyDefinitionId: policyRequireTag.id
  policyDefinitionReferenceId: 'require-tag-${tag}'
  parameters: {
    tagName: {
      value: tag
    }
  }
}]

// Policy Definition: Allowed Locations
resource policyAllowedLocations 'Microsoft.Authorization/policyDefinitions@2023-04-01' = {
  name: 'policy-allowed-locations-${environment}'
  properties: {
    displayName: 'Allowed Locations for Resources'
    description: 'Restricts resource deployment to specific Azure regions'
    policyType: 'Custom'
    mode: 'Indexed'
    metadata: {
      category: 'General'
      version: '1.0.0'
    }
    parameters: {
      allowedLocations: {
        type: 'Array'
        metadata: {
          description: 'The list of allowed locations'
          displayName: 'Allowed Locations'
        }
      }
    }
    policyRule: {
      if: {
        not: {
          field: 'location'
          in: '[parameters(\'allowedLocations\')]'
        }
      }
      then: {
        effect: 'deny'
      }
    }
  }
}

// Policy Definition: Require Tag
resource policyRequireTag 'Microsoft.Authorization/policyDefinitions@2023-04-01' = {
  name: 'policy-require-tag-${environment}'
  properties: {
    displayName: 'Require Tag on Resources'
    description: 'Ensures resources have a specific tag'
    policyType: 'Custom'
    mode: 'Indexed'
    metadata: {
      category: 'Tags'
      version: '1.0.0'
    }
    parameters: {
      tagName: {
        type: 'String'
        metadata: {
          description: 'Name of the required tag'
          displayName: 'Tag Name'
        }
      }
    }
    policyRule: {
      if: {
        field: '[concat(\'tags[\', parameters(\'tagName\'), \']\')]'
        exists: 'false'
      }
      then: {
        effect: 'deny'
      }
    }
  }
}

// Policy Initiative
resource policyInitiative 'Microsoft.Authorization/policySetDefinitions@2023-04-01' = {
  name: 'initiative-governance-${environment}'
  properties: {
    displayName: 'Governance Initiative'
    description: 'Initiative combining location and tag policies'
    policyType: 'Custom'
    metadata: {
      category: 'Governance'
      version: '1.0.0'
    }
    parameters: {
      allowedLocations: {
        type: 'Array'
        metadata: {
          description: 'Allowed Azure locations'
          displayName: 'Allowed Locations'
        }
        defaultValue: allowedLocations
      }
    }
    policyDefinitions: concat(locationPolicyRef, tagPolicyRefs)
  }
}

// Resource Group for testing
resource rg 'Microsoft.Resources/resourceGroups@2024-03-01' = {
  name: 'rg-policy-test-${environment}'
  location: allowedLocations[0]
  tags: commonTags
}

// Policy Assignment (assigned at subscription scope)
resource policyAssignment 'Microsoft.Authorization/policyAssignments@2023-04-01' = {
  name: 'assign-governance-${environment}'
  properties: {
    displayName: 'Governance Policy Assignment'
    description: 'Governance policies for ${environment}'
    policyDefinitionId: policyInitiative.id
    parameters: {
      allowedLocations: {
        value: allowedLocations
      }
    }
  }
}

// Outputs
output policyAllowedLocationsId string = policyAllowedLocations.id
output policyRequireTagId string = policyRequireTag.id
output policyInitiativeId string = policyInitiative.id
output policyAssignmentId string = policyAssignment.id
