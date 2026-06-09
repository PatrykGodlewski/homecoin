// Deploy HomeCoin API + Worker Container Apps (run after images are pushed to ACR).
// Uses ACR admin credentials for image pull (works on Azure for Students without AcrPull RBAC in Bicep).

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

@secure()
param acrUsername string

@secure()
param acrPassword string

param imageTag string
param minReplicas int = 1
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

var appSecrets = [
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
  {
    name: 'acr-password'
    value: acrPassword
  }
]

var registryConfig = [
  {
    server: acr.properties.loginServer
    username: acrUsername
    passwordSecretRef: 'acr-password'
  }
]

var startupProbe = [
  {
    type: 'Startup'
    httpGet: {
      path: '/health'
      port: 8080
      scheme: 'HTTP'
    }
    periodSeconds: 10
    failureThreshold: 18
  }
  {
    type: 'Liveness'
    httpGet: {
      path: '/health'
      port: 8080
      scheme: 'HTTP'
    }
    periodSeconds: 30
    failureThreshold: 3
  }
]

resource workerApp 'Microsoft.App/containerApps@2024-03-01' = {
  name: workerAppName
  location: location
  properties: {
    managedEnvironmentId: containerEnv.id
    configuration: {
      ingress: {
        external: false
        targetPort: 8080
        transport: 'auto'
        allowInsecure: false
      }
      registries: registryConfig
      secrets: appSecrets
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
          probes: startupProbe
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
  properties: {
    managedEnvironmentId: containerEnv.id
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
        transport: 'auto'
        allowInsecure: false
      }
      registries: registryConfig
      secrets: appSecrets
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
          probes: startupProbe
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

output containerAppFqdn string = apiApp.properties.configuration.ingress.fqdn
output workerAppFqdn string = workerApp.properties.configuration.ingress.fqdn
