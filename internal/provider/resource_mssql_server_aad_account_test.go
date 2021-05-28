package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccresourceMsSlqServerAadAccount(t *testing.T) {
	t.Skip("test not yet implemented, see github issue #3")

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
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
