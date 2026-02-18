using './main.bicep'

param environment = 'dev'
param containerImage = 'ghcpiacdev.azurecr.io/ghcp-iac'
param containerImageTag = 'latest'
param cpu = '0.25'
param memory = '0.5Gi'
param minReplicas = 0
param maxReplicas = 1
param modelName = 'gpt-4.1-mini'
param enableLlm = true
param enableNotifications = false
