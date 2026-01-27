# =============================================================================
# Level 4.2: Policy as Code - Solution
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

data "azurerm_subscription" "current" {}

# Resource Group for policy testing
resource "azurerm_resource_group" "main" {
  name     = "rg-${var.workload}-${var.environment}"
  location = "eastus"
  tags     = local.common_tags
}

# Custom Policy: Allowed Locations
resource "azurerm_policy_definition" "allowed_locations" {
  name         = "policy-allowed-locations-${var.environment}"
  policy_type  = "Custom"
  mode         = "Indexed"
  display_name = "Allowed Locations for Resources"
  description  = "Restricts resource deployment to specific Azure regions"

  metadata = jsonencode({
    category = "General"
    version  = "1.0.0"
  })

  policy_rule = jsonencode({
    if = {
      not = {
        field = "location"
        in    = "[parameters('allowedLocations')]"
      }
    }
    then = {
      effect = "deny"
    }
  })

  parameters = jsonencode({
    allowedLocations = {
      type = "Array"
      metadata = {
        description = "The list of allowed locations"
        displayName = "Allowed Locations"
      }
    }
  })
}

# Custom Policy: Require Tags
resource "azurerm_policy_definition" "require_tags" {
  name         = "policy-require-tags-${var.environment}"
  policy_type  = "Custom"
  mode         = "Indexed"
  display_name = "Require Specific Tags on Resources"
  description  = "Ensures resources have required tags"

  metadata = jsonencode({
    category = "Tags"
    version  = "1.0.0"
  })

  policy_rule = jsonencode({
    if = {
      field  = "[concat('tags[', parameters('tagName'), ']')]"
      exists = "false"
    }
    then = {
      effect = "deny"
    }
  })

  parameters = jsonencode({
    tagName = {
      type = "String"
      metadata = {
        description = "Name of the required tag"
        displayName = "Tag Name"
      }
    }
  })
}

# Policy Initiative (Policy Set)
resource "azurerm_policy_set_definition" "governance" {
  name         = "initiative-governance-${var.environment}"
  policy_type  = "Custom"
  display_name = "Governance Initiative"
  description  = "Initiative combining location and tag policies"

  metadata = jsonencode({
    category = "Governance"
    version  = "1.0.0"
  })

  parameters = jsonencode({
    allowedLocations = {
      type = "Array"
      metadata = {
        description = "Allowed Azure locations"
        displayName = "Allowed Locations"
      }
      defaultValue = var.allowed_locations
    }
  })

  policy_definition_reference {
    policy_definition_id = azurerm_policy_definition.allowed_locations.id
    parameter_values = jsonencode({
      allowedLocations = { value = "[parameters('allowedLocations')]" }
    })
  }

  dynamic "policy_definition_reference" {
    for_each = var.required_tags
    content {
      policy_definition_id = azurerm_policy_definition.require_tags.id
      reference_id         = "require-tag-${policy_definition_reference.value}"
      parameter_values = jsonencode({
        tagName = { value = policy_definition_reference.value }
      })
    }
  }
}

# Policy Assignment
resource "azurerm_resource_group_policy_assignment" "governance" {
  name                 = "assign-governance-${var.environment}"
  resource_group_id    = azurerm_resource_group.main.id
  policy_definition_id = azurerm_policy_set_definition.governance.id
  description          = "Governance policies for ${var.environment}"
  display_name         = "Governance Policy Assignment"

  parameters = jsonencode({
    allowedLocations = { value = var.allowed_locations }
  })
}

# Outputs
output "policy_allowed_locations_id" {
  description = "ID of allowed locations policy"
  value       = azurerm_policy_definition.allowed_locations.id
}

output "policy_require_tags_id" {
  description = "ID of require tags policy"
  value       = azurerm_policy_definition.require_tags.id
}

output "policy_initiative_id" {
  description = "ID of governance initiative"
  value       = azurerm_policy_set_definition.governance.id
}

output "policy_assignment_id" {
  description = "ID of policy assignment"
  value       = azurerm_resource_group_policy_assignment.governance.id
}
