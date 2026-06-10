locals {
  database_url = "postgres://${var.postgres_admin_user}:${urlencode(var.postgres_admin_password)}@${azurerm_postgresql_flexible_server.main.fqdn}:5432/${var.postgres_database_name}?sslmode=require&connect_timeout=15"
  api_image    = "${azurerm_container_registry.main.login_server}/${var.app_name}-api:${var.image_tag}"
  worker_image = "${azurerm_container_registry.main.login_server}/${var.app_name}-worker:${var.image_tag}"

  http_probe = {
    path             = "/health"
    port             = 8080
    transport        = "HTTP"
    interval_seconds = 10
  }
}

resource "azurerm_container_app" "worker" {
  count                        = var.deploy_apps ? 1 : 0
  name                         = local.worker_app_name
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = var.resource_group_name
  revision_mode                = "Single"

  secret {
    name  = "database-url"
    value = local.database_url
  }
  secret {
    name  = "worker-internal-token"
    value = var.worker_internal_token
  }
  secret {
    name  = "acr-password"
    value = var.acr_password
  }

  registry {
    server               = azurerm_container_registry.main.login_server
    username             = var.acr_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = false
    target_port      = 8080
    transport        = "auto"

    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = var.min_replicas
    max_replicas = var.max_replicas

    container {
      name   = "worker"
      image  = local.worker_image
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "PORT"
        value = "8080"
      }
      env {
        name        = "DATABASE_URL"
        secret_name = "database-url"
      }
      env {
        name        = "WORKER_INTERNAL_TOKEN"
        secret_name = "worker-internal-token"
      }
      env {
        name  = "LOG_LEVEL"
        value = "info"
      }

      liveness_probe {
        transport        = "HTTP"
        port             = 8080
        path             = "/health"
        interval_seconds = 30
      }

      startup_probe {
        transport        = "HTTP"
        port             = 8080
        path             = "/health"
        interval_seconds = 10
        failure_count_threshold = 18
      }
    }
  }
}

locals {
  worker_internal_url = var.deploy_apps ? "https://${azurerm_container_app.worker[0].ingress[0].fqdn}" : ""
}

resource "azurerm_container_app" "api" {
  count                        = var.deploy_apps ? 1 : 0
  name                         = local.api_app_name
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = var.resource_group_name
  revision_mode                = "Single"

  secret {
    name  = "database-url"
    value = local.database_url
  }
  secret {
    name  = "jwt-secret"
    value = var.jwt_secret
  }
  secret {
    name  = "superkit-secret"
    value = var.superkit_secret
  }
  secret {
    name  = "worker-internal-token"
    value = var.worker_internal_token
  }
  secret {
    name  = "acr-password"
    value = var.acr_password
  }

  registry {
    server               = azurerm_container_registry.main.login_server
    username             = var.acr_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = true
    target_port      = 8080
    transport        = "auto"

    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = var.min_replicas
    max_replicas = var.max_replicas

    container {
      name   = "api"
      image  = local.api_image
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "PORT"
        value = "8080"
      }
      env {
        name        = "DATABASE_URL"
        secret_name = "database-url"
      }
      env {
        name        = "JWT_SECRET"
        secret_name = "jwt-secret"
      }
      env {
        name        = "SUPERKIT_SECRET"
        secret_name = "superkit-secret"
      }
      env {
        name  = "WORKER_URL"
        value = local.worker_internal_url
      }
      env {
        name        = "WORKER_INTERNAL_TOKEN"
        secret_name = "worker-internal-token"
      }
      env {
        name  = "SUPERKIT_ENV"
        value = "production"
      }
      env {
        name  = "TLS_BEHIND_PROXY"
        value = "true"
      }
      env {
        name  = "AUTO_MIGRATE"
        value = "true"
      }
      env {
        name  = "LOG_LEVEL"
        value = "info"
      }

      liveness_probe {
        transport        = "HTTP"
        port             = 8080
        path             = "/health"
        interval_seconds = 30
      }

      startup_probe {
        transport        = "HTTP"
        port             = 8080
        path             = "/health"
        interval_seconds = 10
        failure_count_threshold = 18
      }
    }
  }

  depends_on = [azurerm_container_app.worker]
}
