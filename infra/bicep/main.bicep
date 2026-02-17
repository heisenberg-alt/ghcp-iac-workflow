targetScope = 'resourceGroup'

@description('Project name')
param project string = 'ghcp-iac'

@description('Deployment environment')
@allowed(['dev', 'test', 'prod'])
param environment string

@description('Azure region')
param location string = resourceGroup().location

@description('Container image URI')
param containerImage string

@description('Container image tag')
param containerImageTag string = 'latest'

@description('Container CPU cores')
param cpu string = '0.5'

@description('Container memory')
param memory string = '1Gi'

@description('Minimum replicas')
param minReplicas int = 0

@description('Maximum replicas')
param maxReplicas int = 3

@description('LLM model name')
param modelName string = 'gpt-4o-mini'

@description('Enable LLM analysis')
param enableLlm bool = true

@description('Enable notifications')
param enableNotifications bool = false

@secure()
@description('GitHub webhook secret')
param githubWebhookSecret string = ''

@secure()
@description('Teams webhook URL')
param teamsWebhookUrl string = ''

@secure()
@description('Slack webhook URL')
param slackWebhookUrl string = ''

var namePrefix = '${project}-${environment}'
var tags = {
  project: project
  environment: environment
  managedBy: 'bicep'
}

// Log Analytics Workspace
resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {
  name: 'log-${namePrefix}'
  location: location
  tags: tags
  properties: {
    sku: {
      name: 'PerGB2018'
    }
    retentionInDays: environment == 'prod' ? 90 : 30
  }
}

// Container Registry
resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: replace('acr${namePrefix}', '-', '')
  location: location
  tags: tags
  sku: {
    name: environment == 'prod' ? 'Standard' : 'Basic'
  }
  properties: {
    adminUserEnabled: true
  }
}

// Container App Environment
resource containerAppEnv 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: 'cae-${namePrefix}'
  location: location
  tags: tags
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: logAnalytics.properties.customerId
        sharedKey: logAnalytics.listKeys().primarySharedKey
      }
    }
  }
}

// Container App
resource containerApp 'Microsoft.App/containerApps@2023-05-01' = {
  name: 'ca-${namePrefix}'
  location: location
  tags: tags
  properties: {
    managedEnvironmentId: containerAppEnv.id
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
        transport: 'http'
        traffic: [
          {
            weight: 100
            latestRevision: true
          }
        ]
      }
      registries: [
        {
          server: acr.properties.loginServer
          username: acr.listCredentials().username
          passwordSecretRef: 'registry-password'
        }
      ]
      secrets: [
        {
          name: 'registry-password'
          value: acr.listCredentials().passwords[0].value
        }
        {
          name: 'webhook-secret'
          value: githubWebhookSecret
        }
      ]
    }
    template: {
      containers: [
        {
          name: 'ghcp-iac'
          image: '${containerImage}:${containerImageTag}'
          resources: {
            cpu: json(cpu)
            memory: memory
          }
          env: [
            { name: 'PORT', value: '8080' }
            { name: 'ENVIRONMENT', value: environment }
            { name: 'MODEL_NAME', value: modelName }
            { name: 'ENABLE_LLM', value: string(enableLlm) }
            { name: 'ENABLE_NOTIFICATIONS', value: string(enableNotifications) }
            { name: 'GITHUB_WEBHOOK_SECRET', secretRef: 'webhook-secret' }
          ]
          probes: [
            {
              type: 'Liveness'
              httpGet: { path: '/health', port: 8080 }
            }
            {
              type: 'Readiness'
              httpGet: { path: '/health', port: 8080 }
            }
            {
              type: 'Startup'
              httpGet: { path: '/health', port: 8080 }
            }
          ]
        }
      ]
      scale: {
        minReplicas: minReplicas
        maxReplicas: maxReplicas
        rules: [
          {
            name: 'http-scaling'
            http: {
              metadata: {
                concurrentRequests: '50'
              }
            }
          }
        ]
      }
    }
  }
}

@description('Container App URL')
output containerAppUrl string = 'https://${containerApp.properties.configuration.ingress.fqdn}'

@description('Container Registry login server')
output acrLoginServer string = acr.properties.loginServer

@description('Container Registry name')
output acrName string = acr.name

@description('Resource group name')
output resourceGroupName string = resourceGroup().name
