variable "project_name" {
  type        = string
  description = "Prefix for Azure resource names"
  default     = "homecoin"
}

variable "resource_group_name" {
  type        = string
  description = "Existing or new Azure resource group name"
}

variable "location" {
  type        = string
  description = "Azure region"
  default     = "westeurope"
}

variable "create_resource_group" {
  type        = bool
  description = "Create the resource group (set false if it already exists)"
  default     = true
}

variable "vm_count" {
  type        = number
  description = "Number of HomeCoin application VMs"
  default     = 1
}

variable "vm_size" {
  type        = string
  description = "VM SKU — B2s recommended for Docker builds"
  default     = "Standard_B2s"
}

variable "admin_username" {
  type    = string
  default = "azureadmin"
}

variable "ssh_public_key" {
  type        = string
  description = "SSH public key for VM access (generated automatically if empty)"
  default     = ""
}

variable "allowed_ssh_cidr" {
  type        = string
  description = "CIDR allowed to SSH (use 0.0.0.0/0 for GitHub Actions / dynamic IPs)"
  default     = "0.0.0.0/0"
}

variable "tags" {
  type = map(string)
  default = {
    deployment  = "terraform"
    environment = "homecoin-vm"
    managed_by  = "homecoin-iac"
  }
}
