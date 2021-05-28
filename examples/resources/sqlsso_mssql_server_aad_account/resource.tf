resource "sqlsso_mssql_server_aad_account" "example" {
  sql_server_dns = "my.database.com"
  database       = "mydb"
  account_name   = "myuser"
  object_id      = var.myuser_objectid
  account_type   = "user"
  role           = "owner"
}