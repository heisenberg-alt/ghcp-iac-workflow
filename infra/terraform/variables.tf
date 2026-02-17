variable "project" {
  description = "Project name"
  type        = string
  default     = "ghcp-iac"
}

variable "environment" {
  description = "Deployment environment (dev, test, prod)"
  type        = string
  validation {
    condition     = contains(["dev", "test", "prod"], var.environment)
    error_message = "Environment must be dev, test, or prod."
  }
}

variable "location" {
  description = "Azure region"
  type        = string
  default     = "eastus"
}

variable "container_image" {
  description = "Container image URI"
  type        = string
}

variable "container_image_tag" {
  description = "Container image tag"
  type        = string
  default     = "latest"
}

variable "cpu" {
  description = "Container CPU cores"
  type        = number
  default     = 0.5
}

variable "memory" {
  description = "Container memory in Gi"
  type        = string
  default     = "1Gi"
}

variable "min_replicas" {
  description = "Minimum number of replicas"
  type        = number
  default     = 0
}

variable "max_replicas" {
  description = "Maximum number of replicas"
  type        = number
  default     = 3
}

variable "github_webhook_secret" {
  description = "GitHub webhook secret for signature verification"
  type        = string
  sensitive   = true
  default     = ""
}

variable "model_name" {
  description = "LLM model name"
  type        = string
  default     = "gpt-4o-mini"
}

variable "enable_llm" {
  description = "Enable LLM-powered analysis"
  type        = bool
  default     = true
}

variable "enable_notifications" {
  description = "Enable Teams/Slack notifications"
  type        = bool
  default     = false
}

variable "teams_webhook_url" {
  description = "Microsoft Teams webhook URL"
  type        = string
  sensitive   = true
  default     = ""
}

variable "slack_webhook_url" {
  description = "Slack webhook URL"
  type        = string
  sensitive   = true
  default     = ""
}

variable "tags" {
  description = "Additional resource tags"
  type        = map(string)
  default     = {}
}
