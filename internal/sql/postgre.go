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
	connectionString string
}

func CreatePostgreConnection(sqlServer string, database string, port int64, user string) postgreConnection {
	return postgreConnection{
		connectionString: fmt.Sprintf("postgres://%v:{password}@%v:%v/%v?sslmode=require", user, sqlServer, port, database),
	}
}

func (c postgreConnection) getConnectionString() string {
	return c.connectionString
}

func (c postgreConnection) createConnection() (*sql.DB, error) {
	token, err := cli.GetTokenFromCLI("https://ossrdbms-aad.database.windows.net")
	if err != nil {
		return nil, err
	}

	connStr := strings.Replace(c.connectionString, "{password}", token.AccessToken, 1)

	return sql.Open("postgres", connStr)
}

func (c postgreConnection) CreatePostgreAccount(ctx context.Context, account string, role string, diags *diag.Diagnostics) {

	ctx = tflog.SetField(ctx, "account", account)
	tflog.Debug(ctx, "Creating account..")

	cmd := fmt.Sprintf(`CREATE USER "%s" IN ROLE azure_ad_user;
						GRANT %s TO "%s" WITH INHERIT TRUE;`, account, role, account)

	Execute(ctx, c, diags, cmd)
}

func (c postgreConnection) DropPostgreAccount(ctx context.Context, account string, diags *diag.Diagnostics) {

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM ' + QUOTE_IDENT(@account)
			EXEC (@sql)
			SET @sql = 'REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM ' + QUOTE_IDENT(@account)
			EXEC (@sql)
			SET @sql = 'REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public FROM ' + QUOTE_IDENT(@account)
			EXEC (@sql)
			SET @sql = 'DROP USER ' + QUOTE_IDENT(@account)
			EXEC (@sql)`

	Execute(ctx, c, diags, cmd, sql.Named("account", account))
}
