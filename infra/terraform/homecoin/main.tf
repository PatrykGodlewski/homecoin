data "azurerm_resource_group" "main" {
  name = var.resource_group_name
}

locals {
  unique_suffix      = substr(sha1(data.azurerm_resource_group.main.id), 0, 8)
  acr_name           = replace("${var.app_name}acr${local.unique_suffix}", "-", "")
  container_env_name = "${var.app_name}-env"
  postgres_name      = substr(replace("${var.app_name}-pg-${local.unique_suffix}", "-", ""), 0, 63)
  log_analytics_name = "${var.app_name}-logs"
  api_app_name       = "${var.app_name}-api"
  worker_app_name    = "${var.app_name}-worker"
}

resource "azurerm_log_analytics_workspace" "main" {
  name                = local.log_analytics_name
  location            = var.location
  resource_group_name = var.resource_group_name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

resource "azurerm_container_registry" "main" {
  name                = local.acr_name
  location            = var.location
  resource_group_name = var.resource_group_name
  sku                 = "Basic"
  admin_enabled       = true
}

resource "azurerm_container_app_environment" "main" {
  name                       = local.container_env_name
  location                   = var.location
  resource_group_name        = var.resource_group_name
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
}

resource "azurerm_postgresql_flexible_server" "main" {
  name                   = local.postgres_name
  location               = var.location
  resource_group_name    = var.resource_group_name
  version                = "16"
  administrator_login    = var.postgres_admin_user
  administrator_password = var.postgres_admin_password
  storage_mb             = 32768
  backup_retention_days  = 7
  geo_redundant_backup_enabled = false

  sku_name = "B_Standard_B1ms"

  lifecycle {
    ignore_changes = [zone]
  }
}

resource "azurerm_postgresql_flexible_server_database" "main" {
  name      = var.postgres_database_name
  server_id = azurerm_postgresql_flexible_server.main.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

resource "azurerm_postgresql_flexible_server_firewall_rule" "azure_services" {
  name             = "AllowAzureServices"
  server_id        = azurerm_postgresql_flexible_server.main.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0"
}

resource "azurerm_postgresql_flexible_server_configuration" "pgcrypto" {
  name      = "azure.extensions"
  server_id = azurerm_postgresql_flexible_server.main.id
  value     = "PGCRYPTO"
}
