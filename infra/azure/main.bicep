// HomeCoin Azure stack: ACR + API + Worker Container Apps + PostgreSQL.
// Microservices: API (public HTTPS) communicates with Worker (internal) over HTTP.

@description('Application name prefix for Azure resources')
param appName string = 'homecoin'

@description('Azure region for all resources')
param location string = resourceGroup().location

@description('PostgreSQL administrator username')
param postgresAdminUser string = 'homecoin'

@description('PostgreSQL database name')
param postgresDatabaseName string = 'homecoin'

@secure()
@description('PostgreSQL administrator password')
param postgresAdminPassword string

@secure()
@description('JWT signing secret for the REST API')
param jwtSecret string

@secure()
@description('Superkit session secret (32+ characters)')
param superkitSecret string

@secure()
@description('Shared token for API → Worker internal calls')
param workerInternalToken string

@description('Container App minimum replicas (0 = scale to zero)')
param minReplicas int = 0

@description('Container App maximum replicas')
param maxReplicas int = 2

@description('Container image tag when usePlaceholderImage is false')
param imageTag string = 'latest'

@description('Use Microsoft placeholder image until the first CD pipeline run')
param usePlaceholderImage bool = true

var uniqueSuffix = uniqueString(resourceGroup().id)
var acrName = replace('${appName}acr${uniqueSuffix}', '-', '')
var apiAppName = '${appName}-api'
var workerAppName = '${appName}-worker'
var containerEnvName = '${appName}-env'
var postgresServerName = take(replace('${appName}-pg-${uniqueSuffix}', '-', ''), 63)
var logAnalyticsName = '${appName}-logs'
var apiImageName = '${appName}-api'
var workerImageName = '${appName}-worker'

resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2023-09-01' = {
  name: logAnalyticsName
  location: location
  properties: {
    sku: {
      name: 'PerGB2018'
    }
    retentionInDays: 30
  }
}

resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: acrName
  location: location
  sku: {
    name: 'Basic'
  }
  properties: {
    adminUserEnabled: false
    publicNetworkAccess: 'Enabled'
  }
}

resource containerEnv 'Microsoft.App/managedEnvironments@2024-03-01' = {
  name: containerEnvName
  location: location
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

resource postgres 'Microsoft.DBforPostgreSQL/flexibleServers@2023-12-01-preview' = {
  name: postgresServerName
  location: location
  sku: {
    name: 'Standard_B1ms'
    tier: 'Burstable'
  }
  properties: {
    version: '16'
    administratorLogin: postgresAdminUser
    administratorLoginPassword: postgresAdminPassword
    storage: {
      storageSizeGB: 32
    }
    backup: {
      backupRetentionDays: 7
      geoRedundantBackup: 'Disabled'
    }
    highAvailability: {
      mode: 'Disabled'
    }
  }
}

resource postgresDb 'Microsoft.DBforPostgreSQL/flexibleServers/databases@2023-12-01-preview' = {
  parent: postgres
  name: postgresDatabaseName
}

resource postgresFirewallAzure 'Microsoft.DBforPostgreSQL/flexibleServers/firewallRules@2023-12-01-preview' = {
  parent: postgres
  name: 'AllowAzureServices'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
  }
}

var databaseUrl = 'postgres://${postgresAdminUser}:${uriComponent(postgresAdminPassword)}@${postgres.properties.fullyQualifiedDomainName}:5432/${postgresDatabaseName}?sslmode=require'

var placeholderImage = 'mcr.microsoft.com/k8se/quickstart:latest'
var apiAcrImage = '${acr.properties.loginServer}/${apiImageName}:${imageTag}'
var workerAcrImage = '${acr.properties.loginServer}/${workerImageName}:${imageTag}'
var apiImage = usePlaceholderImage ? placeholderImage : apiAcrImage
var workerImage = usePlaceholderImage ? placeholderImage : workerAcrImage

var sharedSecrets = [
  {
    name: 'database-url'
    value: databaseUrl
  }
  {
    name: 'jwt-secret'
    value: jwtSecret
  }
  {
    name: 'superkit-secret'
    value: superkitSecret
  }
  {
    name: 'worker-internal-token'
    value: workerInternalToken
  }
]

resource workerApp 'Microsoft.App/containerApps@2024-03-01' = {
  name: workerAppName
  location: location
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    managedEnvironmentId: containerEnv.id
    configuration: {
      ingress: {
        external: false
        targetPort: 8080
        transport: 'auto'
        allowInsecure: false
      }
      registries: [
        {
          server: acr.properties.loginServer
          identity: 'system'
        }
      ]
      secrets: sharedSecrets
    }
    template: {
      containers: [
        {
          name: 'worker'
          image: workerImage
          resources: {
            cpu: json('0.25')
            memory: '0.5Gi'
          }
          env: [
            {
              name: 'PORT'
              value: '8080'
            }
            {
              name: 'DATABASE_URL'
              secretRef: 'database-url'
            }
            {
              name: 'WORKER_INTERNAL_TOKEN'
              secretRef: 'worker-internal-token'
            }
            {
              name: 'LOG_LEVEL'
              value: 'info'
            }
          ]
        }
      ]
      scale: {
        minReplicas: minReplicas
        maxReplicas: maxReplicas
      }
    }
  }
}

var workerInternalURL = 'https://${workerApp.properties.configuration.ingress.fqdn}'

resource apiApp 'Microsoft.App/containerApps@2024-03-01' = {
  name: apiAppName
  location: location
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    managedEnvironmentId: containerEnv.id
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
        transport: 'auto'
        allowInsecure: false
      }
      registries: [
        {
          server: acr.properties.loginServer
          identity: 'system'
        }
      ]
      secrets: sharedSecrets
    }
    template: {
      containers: [
        {
          name: 'api'
          image: apiImage
          resources: {
            cpu: json('0.5')
            memory: '1Gi'
          }
          env: [
            {
              name: 'PORT'
              value: '8080'
            }
            {
              name: 'DATABASE_URL'
              secretRef: 'database-url'
            }
            {
              name: 'JWT_SECRET'
              secretRef: 'jwt-secret'
            }
            {
              name: 'SUPERKIT_SECRET'
              secretRef: 'superkit-secret'
            }
            {
              name: 'WORKER_URL'
              value: workerInternalURL
            }
            {
              name: 'WORKER_INTERNAL_TOKEN'
              secretRef: 'worker-internal-token'
            }
            {
              name: 'SUPERKIT_ENV'
              value: 'production'
            }
            {
              name: 'TLS_BEHIND_PROXY'
              value: 'true'
            }
            {
              name: 'AUTO_MIGRATE'
              value: 'true'
            }
            {
              name: 'LOG_LEVEL'
              value: 'info'
            }
          ]
        }
      ]
      scale: {
        minReplicas: minReplicas
        maxReplicas: maxReplicas
      }
    }
  }
}

resource acrPullRoleApi 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(acr.id, apiApp.id, 'AcrPull')
  scope: acr
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '7f951dda-4ed3-4680-a7f8-43da3d0d369c')
    principalId: apiApp.identity.principalId
    principalType: 'ServicePrincipal'
  }
}

resource acrPullRoleWorker 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(acr.id, workerApp.id, 'AcrPull')
  scope: acr
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '7f951dda-4ed3-4680-a7f8-43da3d0d369c')
    principalId: workerApp.identity.principalId
    principalType: 'ServicePrincipal'
  }
}

output acrName string = acr.name
output acrLoginServer string = acr.properties.loginServer
output containerAppName string = apiApp.name
output workerAppName string = workerApp.name
output containerAppFqdn string = apiApp.properties.configuration.ingress.fqdn
output workerAppFqdn string = workerApp.properties.configuration.ingress.fqdn
output postgresFqdn string = postgres.properties.fullyQualifiedDomainName
