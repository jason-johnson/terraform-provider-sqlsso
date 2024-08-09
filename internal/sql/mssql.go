package sql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mssql "github.com/microsoft/go-mssqldb"
)

type mssqlConnection struct {
	connectionString string
}

func CreateMssqlConnection(sqlServer string, database string, port int64) mssqlConnection {
	return mssqlConnection{
		connectionString: fmt.Sprintf("Server=%v;Database=%v;Port=%v;", sqlServer, database, port),
	}
}

func (c mssqlConnection) getConnectionString() string {
	return c.connectionString
}

func getTokenProvider() (func() (string, error), error) {
	token, err := cli.GetTokenFromCLI("https://database.windows.net/")

	return func() (string, error) {
		return token.AccessToken, nil
	}, err
}

func (c mssqlConnection) createConnection() (*sql.DB, error) {
	tokenProvider, err := getTokenProvider()
	if err != nil {
		return nil, err
	}

	connector, err := mssql.NewAccessTokenConnector(c.connectionString, tokenProvider)
	if err != nil {
		return nil, err
	}

	return sql.OpenDB(connector), nil
}

func (c mssqlConnection) CreateMssqlAccount(ctx context.Context, account string, objectId string, accountType string, role string, diags *diag.Diagnostics) {

	ctx = tflog.SetField(ctx, "account", account)
	ctx = tflog.SetField(ctx, "objectId", objectId)
	ctx = tflog.SetField(ctx, "accountType", accountType)
	ctx = tflog.SetField(ctx, "role", role)
	tflog.Debug(ctx, "Creating account..")

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'CREATE USER ' + QuoteName(@account) + ' WITH SID=' + CONVERT(varchar(64), CAST(CAST(@objectId AS UNIQUEIDENTIFIER) AS VARBINARY(16)), 1) + ', TYPE=' + @accountType
			EXEC (@sql)
			SET @sql = 'ALTER ROLE ' + @role + ' ADD MEMBER ' + QuoteName(@account)
			EXEC (@sql)`

	Execute(ctx, c, diags, cmd,
		sql.Named("account", account),
		sql.Named("objectId", objectId),
		sql.Named("accountType", accountType),
		sql.Named("role", role),
	)
}

func (c mssqlConnection) DropMssqlAccount(ctx context.Context, account string, diags *diag.Diagnostics) {

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'DROP USER ' + QuoteName(@account)
			EXEC (@sql)`

	Execute(ctx, c, diags, cmd, sql.Named("account", account))
}
