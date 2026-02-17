using './main.bicep'

param environment = 'test'
param containerImage = 'ghcpiactest.azurecr.io/ghcp-iac'
param containerImageTag = 'latest'
param cpu = '0.5'
param memory = '1Gi'
param minReplicas = 1
param maxReplicas = 2
param modelName = 'gpt-4o-mini'
param enableLlm = true
param enableNotifications = false
