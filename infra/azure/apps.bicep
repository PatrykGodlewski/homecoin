// Deploy HomeCoin API + Worker Container Apps (run after images are pushed to ACR).

param appName string = 'homecoin'
param location string = resourceGroup().location
param acrName string
param containerEnvName string
param postgresServerName string
param postgresAdminUser string = 'homecoin'
param postgresDatabaseName string = 'homecoin'

@secure()
param postgresAdminPassword string

@secure()
param jwtSecret string

@secure()
param superkitSecret string

@secure()
param workerInternalToken string

param imageTag string
param minReplicas int = 0
param maxReplicas int = 2

var apiAppName = '${appName}-api'
var workerAppName = '${appName}-worker'
var apiImageName = '${appName}-api'
var workerImageName = '${appName}-worker'

resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' existing = {
  name: acrName
}

resource containerEnv 'Microsoft.App/managedEnvironments@2024-03-01' existing = {
  name: containerEnvName
}

resource postgres 'Microsoft.DBforPostgreSQL/flexibleServers@2023-12-01-preview' existing = {
  name: postgresServerName
}

var databaseUrl = 'postgres://${postgresAdminUser}:${uriComponent(postgresAdminPassword)}@${postgres.properties.fullyQualifiedDomainName}:5432/${postgresDatabaseName}?sslmode=require'
var apiImage = '${acr.properties.loginServer}/${apiImageName}:${imageTag}'
var workerImage = '${acr.properties.loginServer}/${workerImageName}:${imageTag}'

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

output containerAppFqdn string = apiApp.properties.configuration.ingress.fqdn
output workerAppFqdn string = workerApp.properties.configuration.ingress.fqdn
