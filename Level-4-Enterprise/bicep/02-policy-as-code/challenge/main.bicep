// =============================================================================
// Level 4.2: Policy as Code - Challenge
// =============================================================================
// TODO: Create Azure Policy definitions and assignments
// =============================================================================

targetScope = 'subscription'

@description('Environment')
param environment string = 'prod'

@description('Allowed Azure locations')
param allowedLocations array = ['eastus', 'eastus2', 'westus2']

@description('Required tags')
param requiredTags array = ['environment', 'project', 'cost_center']

// TODO: Create a custom policy definition for allowed locations
// - Use 'deny' effect
// - Accept allowedLocations as parameter


// TODO: Create a custom policy definition for required tags
// - Use 'deny' effect
// - Check if tag exists


// TODO: Create a policy initiative (policy set) that includes:
// - The allowed locations policy
// - The required tags policy (for each required tag)


// TODO: Create a resource group for testing policies


// TODO: Assign the policy initiative to the resource group


// TODO: Add outputs for:
// - policyDefinitionIds
// - policyInitiativeId
// - policyAssignmentId
