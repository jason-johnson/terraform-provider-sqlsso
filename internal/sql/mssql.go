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
	sqlServer   string
	database    string
	port        int64
	account     string
	objectId    string
	accountType string
	role        string
}

func CreateMssqlConnection(sqlServer string, database string, port int64, account string, objectId string, accountType string, role string) mssqlConnection {
	return mssqlConnection{
		sqlServer:   sqlServer,
		database:    database,
		port:        port,
		account:     account,
		objectId:    objectId,
		accountType: accountType,
		role:        role,
	}
}

func (c mssqlConnection) getConnectionString() string {
	return fmt.Sprintf("Server=%v;Database=%v;Port=%v;", c.sqlServer, c.database, c.port)
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

	connector, err := mssql.NewAccessTokenConnector(c.getConnectionString(), tokenProvider)
	if err != nil {
		return nil, err
	}

	return sql.OpenDB(connector), nil
}

func (c mssqlConnection) CreateAccount(ctx context.Context, diags *diag.Diagnostics) {

	ctx = tflog.SetField(ctx, "account", c.account)
	ctx = tflog.SetField(ctx, "objectId", c.objectId)
	ctx = tflog.SetField(ctx, "accountType", c.accountType)
	ctx = tflog.SetField(ctx, "role", c.role)
	tflog.Debug(ctx, "Creating account..")

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'CREATE USER ' + QuoteName(@account) + ' WITH SID=' + CONVERT(varchar(64), CAST(CAST(@objectId AS UNIQUEIDENTIFIER) AS VARBINARY(16)), 1) + ', TYPE=' + @accountType
			EXEC (@sql)
			SET @sql = 'ALTER ROLE ' + @role + ' ADD MEMBER ' + QuoteName(@account)
			EXEC (@sql)`

	Execute(ctx, c, diags, cmd,
		sql.Named("account", c.account),
		sql.Named("objectId", c.objectId),
		sql.Named("accountType", c.accountType),
		sql.Named("role", c.role),
	)
}

func (c mssqlConnection) DropAccount(ctx context.Context, diags *diag.Diagnostics) {

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'DROP USER ' + QuoteName(@account)
			EXEC (@sql)`

	Execute(ctx, c, diags, cmd, sql.Named("account", c.account))
}

func (c mssqlConnection) Id() string {
	return fmt.Sprint(c.sqlServer, ":", c.database, ":", c.port, "/", c.account)
}
