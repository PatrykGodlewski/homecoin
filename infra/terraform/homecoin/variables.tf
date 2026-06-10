variable "app_name" {
  type    = string
  default = "homecoin"
}

variable "resource_group_name" {
  type = string
}

variable "location" {
  type = string
}

variable "postgres_admin_user" {
  type    = string
  default = "homecoin"
}

variable "postgres_database_name" {
  type    = string
  default = "homecoin"
}

variable "postgres_admin_password" {
  type      = string
  sensitive = true
}

# Container Apps deployment (apps.tf) — leave image_tag empty for infra-only apply.
variable "deploy_apps" {
  type        = bool
  description = "Deploy API + Worker Container Apps (requires images in ACR)"
  default     = false
}

variable "image_tag" {
  type    = string
  default = "latest"
}

variable "jwt_secret" {
  type      = string
  sensitive = true
  default   = ""
}

variable "superkit_secret" {
  type      = string
  sensitive = true
  default   = ""
}

variable "worker_internal_token" {
  type      = string
  sensitive = true
  default   = ""
}

variable "acr_username" {
  type      = string
  sensitive = true
  default   = ""
}

variable "acr_password" {
  type      = string
  sensitive = true
  default   = ""
}

variable "min_replicas" {
  type    = number
  default = 1
}

variable "max_replicas" {
  type    = number
  default = 2
}
