# =============================================================================
# Level 3.3: AKS Cluster - Challenge
# =============================================================================
# TODO: Create an Azure Kubernetes Service cluster with proper configuration
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
  default     = "aks"
}

variable "kubernetes_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.28"
}

variable "system_node_count" {
  description = "Number of system nodes"
  type        = number
  default     = 2
}

variable "system_node_vm_size" {
  description = "VM size for system nodes"
  type        = string
  default     = "Standard_D2s_v3"
}

locals {
  name_prefix = "${var.workload}-${var.environment}"
  common_tags = {
    environment = var.environment
    project     = var.workload
    managed_by  = "terraform"
  }
}

# TODO: Create a resource group for AKS


# TODO: Create an Azure Container Registry (ACR)


# TODO: Create a Log Analytics workspace for monitoring


# TODO: Create the AKS cluster with:
#   - System-assigned managed identity
#   - System node pool with the specified configuration
#   - Azure CNI networking
#   - Azure Policy addon
#   - Log Analytics integration


# TODO: Create role assignment to allow AKS to pull from ACR


# TODO: Add outputs for:
#   - AKS cluster name
#   - AKS cluster ID
#   - ACR login server
#   - Kube config (sensitive)
