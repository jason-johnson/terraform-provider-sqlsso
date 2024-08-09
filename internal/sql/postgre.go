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
	token, err := cli.GetTokenFromCLI("https://database.windows.net/")
	if err != nil {
		return nil, err
	}

	connStr := strings.Replace(c.connectionString, "{password}", token.AccessToken, 1)

	return sql.Open("postgres", connStr)
}

func (c postgreConnection) CreatePostgreAccount(ctx context.Context, account string, objectId string, accountType string, role string, diags *diag.Diagnostics) {

	ctx = tflog.SetField(ctx, "account", account)
	ctx = tflog.SetField(ctx, "objectId", objectId)
	ctx = tflog.SetField(ctx, "accountType", accountType)
	ctx = tflog.SetField(ctx, "role", role)
	tflog.Debug(ctx, "Creating account..")

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'CREATE USER ' + QuoteName(@account) + ' IN ROLE azure_ad_user'
			EXEC (@sql)
			SET @sql = 'GRANT ' + @role + ' TO ' + QuoteName(@account) + ' WITH INHERIT TRUE'
			EXEC (@sql)`

	Execute(ctx, c, diags, cmd,
		sql.Named("account", account),
		sql.Named("objectId", objectId),
		sql.Named("accountType", accountType),
		sql.Named("role", role),
	)
}

func (c postgreConnection) DropPostgreAccount(ctx context.Context, account string, diags *diag.Diagnostics) {

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM ' + QuoteName(@account)
			EXEC (@sql)
			SET @sql = 'REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM ' + QuoteName(@account)
			EXEC (@sql)
			SET @sql = 'REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public FROM ' + QuoteName(@account)
			EXEC (@sql)
			SET @sql = 'DROP USER ' + QuoteName(@account)
			EXEC (@sql)`

	Execute(ctx, c, diags, cmd, sql.Named("account", account))
}
