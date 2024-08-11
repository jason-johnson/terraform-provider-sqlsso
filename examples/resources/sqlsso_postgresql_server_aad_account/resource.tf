provider "azurerm" {
  features {}
}

provider "sqlsso" {}

data "azurerm_client_config" "current" {}

resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "West Europe"
}

resource "azurerm_postgresql_flexible_server" "example" {
  name                          = "example-postgresql-server"
  resource_group_name           = azurerm_resource_group.example.name
  location                      = azurerm_resource_group.example.location
  version                       = "16"
  public_network_access_enabled = true
  authentication {
    active_directory_auth_enabled = true
    password_auth_enabled         = false
    tenant_id                     = data.azurerm_client_config.current.tenant_id
  }

  sku_name = "B_Standard_B1ms"

  lifecycle {
    ignore_changes = [
      zone
    ]
  }
}

resource "azurerm_postgresql_flexible_server_active_directory_administrator" "example" {
  server_name         = azurerm_postgresql_flexible_server.example.name
  resource_group_name = azurerm_resource_group.example.name
  tenant_id           = data.azurerm_client_config.current.tenant_id
  object_id           = data.azurerm_client_config.current.object_id
  principal_name      = data.azurerm_client_config.current.display_name
  principal_type      = "User"
}

resource "azurerm_postgresql_flexible_server_database" "example" {
  name      = "example-db"
  server_id = azurerm_postgresql_flexible_server.example.id
  collation = "en_US.utf8"
  charset   = "utf8"
}

resource "azurerm_service_plan" "example" {
  name                = "example-plan"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
  os_type             = "Linux"
  sku_name            = "P1v2"
}

resource "azurerm_linux_web_app" "example" {
  name                = "example-linux-web-app"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_service_plan.example.location
  service_plan_id     = azurerm_service_plan.example.id

  site_config {}

  identity {
    type = "SystemAssigned"
  }
}

# This will require the right permissions, see azuread_service_principal
data "azuread_service_principal" "example" {
  object_id = azurerm_linux_web_app.example.identity[0].principal_id
}

resource "sqlsso_postgresql_server_aad_account" "example" {
  sql_server_dns = azurerm_postgresql_flexible_server.example.fully_qualified_domain_name
  database       = azurerm_postgresql_flexible_server_database.example.name
  user_name      = data.azurerm_client_config.current.display_name
  account_name   = azurerm_linux_web_app.example.name
  role           = "owner"
}
