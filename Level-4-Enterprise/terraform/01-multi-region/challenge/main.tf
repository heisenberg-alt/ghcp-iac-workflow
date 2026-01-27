# =============================================================================
# Level 4.1: Multi-Region Deployment - Challenge
# =============================================================================
# TODO: Create a multi-region infrastructure with Azure Front Door
# =============================================================================

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
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "prod"
}

variable "workload" {
  description = "Workload name"
  type        = string
  default     = "multiregion"
}

variable "regions" {
  description = "List of Azure regions for deployment"
  type        = list(string)
  default     = ["eastus", "westeurope"]
}

locals {
  common_tags = {
    environment = var.environment
    project     = var.workload
    managed_by  = "terraform"
  }
}

# TODO: Create resource groups for each region using for_each


# TODO: Create App Service Plans in each region (Linux, B1 SKU)


# TODO: Create Web Apps in each region


# TODO: Create Azure Front Door profile


# TODO: Create Front Door endpoint


# TODO: Create Front Door origin group


# TODO: Create Front Door origins pointing to each regional web app


# TODO: Create Front Door route


# TODO: Add outputs for:
#   - Front Door endpoint hostname
#   - Regional web app hostnames
