package resource_test

import (
	"regexp"
	"terraform-provider-sqlsso/internal/acctest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccresourceMsSlqServerAadAccount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccresourceMsSlqServerAadAccount,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"sqlsso_mssql_server_aad_account.foo", "sql_server_dns", regexp.MustCompile("^my")),
				),
			},
		},
	})
}

const testAccresourceMsSlqServerAadAccount = `
resource "sqlsso_mssql_server_aad_account" "foo" {
  sql_server_dns = "my.database.com"
	database = "mydb"
	account_name = "user"
	object_id = "0x111"
	account_type = "user"
	role = "owner"
}
`
