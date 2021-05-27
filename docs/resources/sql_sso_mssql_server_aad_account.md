---
page_title: "sql_sso_mssql_server_aad_account Resource - terraform-provider-sql-sso"
subcategory: ""
description: |-
  
---

# Resource `sql_sso_mssql_server_aad_account`





## Schema

### Required

- **account_name** (String) The name of the account to add to the database
- **database** (String) The name of the database to add the account
- **object_id** (String) Azure AD object ID for the account
- **sql_server_dns** (String) The DNS name of the SQL server to add account

### Optional

- **account_type** (String) Type of account to create: either a single user or an AAD group
- **id** (String) The ID of this resource.
- **port** (Number) Port to connect to the database server
- **role** (String) The role the account should get (e.g. owner, reader, etc.)


