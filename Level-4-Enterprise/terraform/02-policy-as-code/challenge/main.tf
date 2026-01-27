# =============================================================================
# Level 4.2: Policy as Code - Challenge
# =============================================================================
# TODO: Implement Azure Policy definitions and assignments using Terraform
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
  default     = "policy"
}

variable "allowed_locations" {
  description = "List of allowed Azure regions"
  type        = list(string)
  default     = ["eastus", "eastus2", "westus2"]
}

variable "required_tags" {
  description = "Tags that must be present on resources"
  type        = list(string)
  default     = ["environment", "project", "cost_center"]
}

locals {
  common_tags = {
    environment = var.environment
    project     = var.workload
    managed_by  = "terraform"
  }
}

# TODO: Create a resource group for policy testing


# TODO: Create a custom policy definition to enforce allowed locations


# TODO: Create a custom policy definition to require specific tags


# TODO: Create a policy initiative (policy set) combining both policies


# TODO: Assign the policy initiative to the resource group


# TODO: Add outputs for:
#   - Policy definition IDs
#   - Policy initiative ID
#   - Policy assignment ID
