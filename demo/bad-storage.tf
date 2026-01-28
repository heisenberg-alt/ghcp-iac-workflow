# Bad Storage Account - Multiple Issues
# Used for demo purposes

resource "azurerm_storage_account" "bad" {
  name                      = "badstorage"
  resource_group_name       = "rg-demo"
  location                  = "eastus"
  account_tier              = "Premium"
  account_replication_type  = "LRS"
  
  # Security Issues
  enable_https_traffic_only = false
  min_tls_version           = "TLS1_0"
  allow_blob_public_access  = true
  
  # Missing tags (policy violation)
}
