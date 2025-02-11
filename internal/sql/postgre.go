package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	_ "github.com/lib/pq"
)

type postgreConnection struct {
	sqlServer string
	database  string
	port      int64
	user      string
	account   string
	role      string
}

func CreatePostgreConnection(sqlServer string, database string, port int64, user string, account string, role string) postgreConnection {
	return postgreConnection{
		sqlServer: sqlServer,
		database:  database,
		port:      port,
		user:      user,
		account:   account,
		role:      role,
	}
}

func (c postgreConnection) getConnectionString() string {
	return fmt.Sprintf("postgres://%v:{password}@%v:%v/%v?sslmode=require", c.user, c.sqlServer, c.port, c.database)
}

func (c postgreConnection) createConnection(ctx context.Context) (*sql.DB, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{"https://ossrdbms-aad.database.windows.net"}})

	if err != nil {
		return nil, err
	}

	connStr := strings.Replace(c.getConnectionString(), "{password}", token.Token, 1)

	return sql.Open("postgres", connStr)
}

func (c postgreConnection) CreateAccount(ctx context.Context, diags *diag.Diagnostics) {

	// Create account has to run on postgres database
	targetDatabase := c.database
	c.database = "postgres"

	ctx = tflog.SetField(ctx, "account", c.account)
	tflog.Debug(ctx, "Creating account..")
	cmd := fmt.Sprintf(`select * from pg_catalog.pgaadauth_create_principal('%s', false, false);`, c.account)
	Execute(ctx, c, diags, cmd)

	if diags.HasError() {
		return
	}

	tflog.Debug(ctx, "Account created, creating role..")
	cmd = fmt.Sprintf(`GRANT %s ON DATABASE %s TO "%s";`, c.role, targetDatabase, c.account)
	Execute(ctx, c, diags, cmd)
}

func (c postgreConnection) DropAccount(ctx context.Context, diags *diag.Diagnostics) {

	targetDatabase := c.database
	c.database = "postgres"

	tflog.Debug(ctx, "Revoking role..")
	cmd := fmt.Sprintf(`REVOKE %s ON DATABASE %s FROM "%s";`, c.role, targetDatabase, c.account)
	Execute(ctx, c, diags, cmd)

	if diags.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "account", c.account)
	tflog.Debug(ctx, "dropping account..")
	cmd = fmt.Sprintf(`drop user "%s";`, c.account)
	Execute(ctx, c, diags, cmd)
}

func (c postgreConnection) Id() string {
	return fmt.Sprint(c.sqlServer, ":", c.database, ":", c.port, "/", c.account)
}
