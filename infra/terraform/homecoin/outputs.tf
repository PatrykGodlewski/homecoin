output "acr_name" {
  value = azurerm_container_registry.main.name
}

output "acr_login_server" {
  value = azurerm_container_registry.main.login_server
}

output "container_env_name" {
  value = azurerm_container_app_environment.main.name
}

output "postgres_server_name" {
  value = azurerm_postgresql_flexible_server.main.name
}

output "postgres_database_name" {
  value = var.postgres_database_name
}

output "postgres_admin_user" {
  value = var.postgres_admin_user
}

output "postgres_fqdn" {
  value = azurerm_postgresql_flexible_server.main.fqdn
}

output "container_app_name" {
  value = local.api_app_name
}

output "worker_app_name" {
  value = local.worker_app_name
}

output "container_app_fqdn" {
  value = var.deploy_apps ? azurerm_container_app.api[0].ingress[0].fqdn : ""
}

output "worker_app_fqdn" {
  value = var.deploy_apps ? azurerm_container_app.worker[0].ingress[0].fqdn : ""
}
