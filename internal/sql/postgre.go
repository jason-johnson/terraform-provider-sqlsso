package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure/cli"
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

func (c postgreConnection) createConnection() (*sql.DB, error) {
	token, err := cli.GetTokenFromCLI("https://ossrdbms-aad.database.windows.net")
	if err != nil {
		return nil, err
	}

	connStr := strings.Replace(c.getConnectionString(), "{password}", token.AccessToken, 1)

	return sql.Open("postgres", connStr)
}

func (c postgreConnection) CreateAccount(ctx context.Context, diags *diag.Diagnostics) {

	ctx = tflog.SetField(ctx, "account", c.account)
	tflog.Debug(ctx, "Creating account..")
	cmd := fmt.Sprintf(`pg_catalog.pgaadauth_create_principal("%s", false, false);`, c.account)
	Execute(ctx, c, diags, cmd)

	tflog.Debug(ctx, "Account created, creating role..")
	cmd = fmt.Sprintf(`GRANT %s TO "%s" WITH INHERIT TRUE;`, c.role, c.account)
	Execute(ctx, c, diags, cmd)
}

func (c postgreConnection) DropAccount(ctx context.Context, diags *diag.Diagnostics) {

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM ' + QUOTE_IDENT(@account)
			EXEC (@sql)
			SET @sql = 'REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM ' + QUOTE_IDENT(@account)
			EXEC (@sql)
			SET @sql = 'REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public FROM ' + QUOTE_IDENT(@account)
			EXEC (@sql)
			SET @sql = 'DROP USER ' + QUOTE_IDENT(@account)
			EXEC (@sql)`

	Execute(ctx, c, diags, cmd, sql.Named("account", c.account))
}

func (c postgreConnection) Id() string {
	return fmt.Sprint(c.sqlServer, ":", c.database, ":", c.port, "/", c.account)
}
