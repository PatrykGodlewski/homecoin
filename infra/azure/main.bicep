// HomeCoin Azure base stack: ACR + Container Apps Environment + PostgreSQL.
// Container Apps (API + Worker) are deployed separately via apps.bicep after images are in ACR.

@description('Application name prefix for Azure resources')
param appName string = 'homecoin'

@description('Azure region for all resources')
param location string = resourceGroup().location

@description('PostgreSQL administrator username')
param postgresAdminUser string = 'homecoin'

@description('PostgreSQL database name')
param postgresDatabaseName string = 'homecoin'

@secure()
@minLength(8)
@description('PostgreSQL administrator password (min 8 chars)')
param postgresAdminPassword string

var uniqueSuffix = uniqueString(resourceGroup().id)
var acrName = replace('${appName}acr${uniqueSuffix}', '-', '')
var containerEnvName = '${appName}-env'
var postgresServerName = take(replace('${appName}-pg-${uniqueSuffix}', '-', ''), 63)
var logAnalyticsName = '${appName}-logs'
var apiAppName = '${appName}-api'
var workerAppName = '${appName}-worker'

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

// Required for migration 000001 (CREATE EXTENSION pgcrypto)
resource postgresExtensions 'Microsoft.DBforPostgreSQL/flexibleServers/configurations@2023-12-01-preview' = {
  parent: postgres
  name: 'azure.extensions'
  properties: {
    value: 'PGCRYPTO'
    source: 'user-override'
  }
}

output acrName string = acr.name
output acrLoginServer string = acr.properties.loginServer
output containerEnvName string = containerEnv.name
output postgresServerName string = postgres.name
output postgresDatabaseName string = postgresDatabaseName
output postgresAdminUser string = postgresAdminUser
output containerAppName string = apiAppName
output workerAppName string = workerAppName
output postgresFqdn string = postgres.properties.fullyQualifiedDomainName
