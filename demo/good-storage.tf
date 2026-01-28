# Good Storage Account - Compliant
# Used for demo purposes

resource "azurerm_storage_account" "good" {
  name                      = "goodstorage"
  resource_group_name       = "rg-demo"
  location                  = "eastus"
  account_tier              = "Standard"
  account_replication_type  = "GRS"
  
  # Security Best Practices
  enable_https_traffic_only = true
  min_tls_version           = "TLS1_2"
  allow_blob_public_access  = false
  
  # Required Tags
  tags = {
    environment = "production"
    project     = "demo"
    managed_by  = "terraform"
  }
}
