# =============================================================================
# Level 4.3: CI/CD Integration - Challenge
# =============================================================================
# TODO: Create Terraform configuration for CI/CD pipeline testing
# =============================================================================

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }

  # TODO: Configure remote backend for CI/CD state management
}

provider "azurerm" {
  features {}
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "location" {
  description = "Azure region"
  type        = string
  default     = "eastus"
}

variable "workload" {
  description = "Workload name"
  type        = string
  default     = "cicd"
}

locals {
  name_prefix = "${var.workload}-${var.environment}"
  common_tags = {
    environment = var.environment
    project     = var.workload
    managed_by  = "terraform"
    deployed_by = "github-actions"
  }
}

# TODO: Create a resource group


# TODO: Create a storage account


# TODO: Create an App Service Plan


# TODO: Create a Web App


# TODO: Add outputs for deployment validation:
#   - resource_group_name
#   - storage_account_name
#   - web_app_name
#   - web_app_url
