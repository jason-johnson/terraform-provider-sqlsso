provider "azurerm" {
  features {}
}

provider "sqlsso" {}

data "azurerm_client_config" "current" {}

resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "West Europe"
}

resource "azurerm_mssql_server" "example" {
  name                = "example-sqlserver"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
  version             = "12.0"
  minimum_tls_version = "1.2"

  azuread_administrator {
    login_username              = "AzureAD Admin"
    object_id                   = data.azurerm_client_config.current.object_id
    azuread_authentication_only = true
  }
}

resource "azurerm_mssql_database" "example" {
  name      = "example-db"
  server_id = azurerm_mssql_server.example.id
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

resource "sqlsso_mssql_server_aad_account" "example" {
  sql_server_dns = azurerm_mssql_server.example.fully_qualified_domain_name
  database       = azurerm_mssql_database.example.name
  account_name   = azurerm_linux_web_app.example.name
  object_id      = data.azuread_service_principal.example.application_id
  role           = "owner"
}