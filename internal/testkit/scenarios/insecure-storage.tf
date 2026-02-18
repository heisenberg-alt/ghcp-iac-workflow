resource "azurerm_storage_account" "insecure" {
  name                      = "insecurestorage"
  resource_group_name       = azurerm_resource_group.main.name
  location                  = azurerm_resource_group.main.location
  account_tier              = "Standard"
  account_replication_type  = "LRS"

  enable_https_traffic_only = false
  min_tls_version           = "TLS1_0"
  allow_blob_public_access  = true

  public_network_access_enabled = true
}
