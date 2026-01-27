// =============================================================================
// Level 3.3: AKS Cluster - Challenge
// =============================================================================
// TODO: Create an Azure Kubernetes Service cluster
// =============================================================================

@description('Environment')
param environment string = 'dev'

@description('Location')
param location string = 'eastus'

@description('Workload name')
param workload string = 'aks'

@description('Kubernetes version')
param kubernetesVersion string = '1.28'

@description('System node count')
@minValue(1)
@maxValue(10)
param systemNodeCount int = 2

@description('System node VM size')
param systemNodeVmSize string = 'Standard_D2s_v3'

// Variables
var aksName = 'aks-${workload}-${environment}'
var commonTags = {
  environment: environment
  project: workload
  managed_by: 'bicep'
}

// TODO: Create a Log Analytics workspace for AKS monitoring


// TODO: Create an Azure Container Registry (ACR)


// TODO: Create the AKS cluster with:
// - System-assigned managed identity
// - System node pool configuration
// - Azure CNI networking
// - Log Analytics integration
// - Azure Policy addon


// TODO: Create role assignment to allow AKS to pull from ACR


// TODO: Add outputs for:
// - aksClusterName
// - aksClusterId
// - acrLoginServer
