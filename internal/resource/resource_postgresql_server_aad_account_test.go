package resource_test

import (
	"fmt"
	"os"
	"terraform-provider-sqlsso/internal/acctest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccresourcePostgreServerAadAccount(t *testing.T) {
	serverDns := os.Getenv("TF_SQLSSO_POSTGRE_SERVER_DNS")
	dbName := os.Getenv("TF_SQLSSO_DB_NAME")
	userName := os.Getenv("TF_SQLSSO_USER_NAME")
	accountName := os.Getenv("TF_SQLSSO_ACCOUNT_NAME")

	if len(serverDns) == 0 {
		t.Skip("TF_SQLSSO_POSTGRE_SERVER_DNS must be set to test MS SQL Server AAD Account")
	}
	if len(dbName) == 0 {
		t.Skip("TF_SQLSSO_DB_NAME must be set for acceptance tests")
	}
	if len(accountName) == 0 {
		t.Skip("TF_SQLSSO_ACCOUNT_NAME must be set for acceptance tests")
	}
	if len(userName) == 0 {
		t.Skip("TF_SQLSSO_USER_NAME must be set for acceptance tests")
	}

	config := fmt.Sprintf(testAccresourcePostgreServerAadAccount, serverDns, dbName, userName, accountName)
	expectedId := fmt.Sprint(serverDns, ":", dbName, ":5432", "/", accountName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("sqlsso_postgresql_server_aad_account.example", "id", expectedId),
				),
			},
		},
	})
}

const testAccresourcePostgreServerAadAccount = `
resource "sqlsso_postgresql_server_aad_account" "example" {
  sql_server_dns = "%s"
	database = "%s"
	user_name = "%s"
	account_name = "%s"
	role = "owner"
}
`
