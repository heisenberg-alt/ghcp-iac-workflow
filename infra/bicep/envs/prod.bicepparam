using './main.bicep'

param environment = 'prod'
param containerImage = 'ghcpiacprod.azurecr.io/ghcp-iac'
param containerImageTag = 'v1.0.0'
param cpu = '1.0'
param memory = '2Gi'
param minReplicas = 2
param maxReplicas = 5
param modelName = 'gpt-4.1'
param enableLlm = true
param enableNotifications = true
