# =============================================================================
# Level 3.2: State Management - Challenge
# =============================================================================
# TODO: Configure remote state backend for this Terraform configuration
# =============================================================================

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }

  # TODO: Add backend "azurerm" block with:
  # - resource_group_name
  # - storage_account_name
  # - container_name
  # - key (state file name)
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

# TODO: Create a resource group


# TODO: Create a storage account for application data


# TODO: Create outputs for resource IDs
