resource "azurerm_storage_account" "data" {
  name                      = "datastorage"
  resource_group_name       = azurerm_resource_group.main.name
  location                  = azurerm_resource_group.main.location
  account_tier              = "Standard"
  account_replication_type  = "GRS"

  enable_https_traffic_only = false
  min_tls_version           = "TLS1_0"
  allow_blob_public_access  = true
}

resource "azurerm_key_vault" "main" {
  name                = "app-keyvault"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  tenant_id           = data.azurerm_client_config.current.tenant_id
  sku_name            = "standard"

  soft_delete_enabled       = false
  purge_protection_enabled  = false
}

resource "azurerm_network_security_group" "open" {
  name                = "open-nsg"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location

  security_rule {
    name                       = "AllowAll"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}
