resource "azurerm_storage_account" "secure" {
  name                      = "securestorage"
  resource_group_name       = azurerm_resource_group.main.name
  location                  = azurerm_resource_group.main.location
  account_tier              = "Standard"
  account_replication_type  = "LRS"

  enable_https_traffic_only = true
  min_tls_version           = "TLS1_2"
  allow_blob_public_access  = false

  infrastructure_encryption_enabled = true

  network_rules {
    default_action = "Deny"
  }
}
