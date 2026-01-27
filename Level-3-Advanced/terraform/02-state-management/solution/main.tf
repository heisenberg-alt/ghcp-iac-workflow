# =============================================================================
# Level 3.2: State Management - Solution
# =============================================================================

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }

  # Remote state configuration
  backend "azurerm" {
    resource_group_name  = "rg-terraform-state"
    storage_account_name = "stterraformstate"
    container_name       = "tfstate"
    key                  = "statemgmt.terraform.tfstate"
  }
}

provider "azurerm" {
  features {}
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}

variable "location" {
  description = "Azure region"
  type        = string
  default     = "eastus"
}

variable "workload" {
  description = "Workload name"
  type        = string
  default     = "statemgmt"
}

locals {
  name_prefix = "${var.workload}-${var.environment}"
  common_tags = {
    environment = var.environment
    project     = var.workload
    managed_by  = "terraform"
  }
}

# Resource Group
resource "azurerm_resource_group" "main" {
  name     = "rg-${local.name_prefix}"
  location = var.location
  tags     = local.common_tags
}

# Storage Account for application data
resource "azurerm_storage_account" "main" {
  name                     = "st${replace(local.name_prefix, "-", "")}001"
  resource_group_name      = azurerm_resource_group.main.name
  location                 = azurerm_resource_group.main.location
  account_tier             = "Standard"
  account_replication_type = "LRS"

  blob_properties {
    versioning_enabled = true
  }

  tags = local.common_tags
}

# Outputs
output "resource_group_id" {
  description = "Resource Group ID"
  value       = azurerm_resource_group.main.id
}

output "storage_account_id" {
  description = "Storage Account ID"
  value       = azurerm_storage_account.main.id
}

output "storage_account_name" {
  description = "Storage Account name"
  value       = azurerm_storage_account.main.name
}
