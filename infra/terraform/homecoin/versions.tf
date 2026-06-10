terraform {
  required_version = ">= 1.5.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }
}

provider "azurerm" {
  features {}

  # GitHub SP has Contributor only on RG — cannot register providers at subscription scope.
  # Required providers are registered locally via infra/azure/register-providers.sh
  # and verified in the Azure Infrastructure workflow before terraform apply.
  resource_provider_registrations = "none"
}
