output "resource_group_name" {
  description = "The resource group name"
  value       = azurerm_resource_group.main.name
}

output "container_registry_login_server" {
  description = "The Container Registry login server"
  value       = azurerm_container_registry.main.login_server
}

output "container_registry_name" {
  description = "The Container Registry name"
  value       = azurerm_container_registry.main.name
}

output "container_app_url" {
  description = "The Container App FQDN"
  value       = "https://${azurerm_container_app.ghcp_iac.ingress[0].fqdn}"
}

output "container_app_environment_id" {
  description = "The Container App Environment ID"
  value       = azurerm_container_app_environment.main.id
}

output "log_analytics_workspace_id" {
  description = "The Log Analytics Workspace ID"
  value       = azurerm_log_analytics_workspace.main.id
}
