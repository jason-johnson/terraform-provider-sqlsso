package provider

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	mssql "github.com/denisenkom/go-mssqldb"
)

var accountTypeMap = map[string]string{"user": "E", "group": "X"}
var roleMap = map[string]string{"owner": "db_owner", "reader": "db_datareader", "writer": "db_datawriter"}

func resourceMsSlqServerAadAccount() *schema.Resource {

	return &schema.Resource{
		Description: "`sqlsso_mssql_server_aad_account` enables AAD authentication for an Azure MS SQL server.\n\n" +
			"For this to work terraform should be run for the configured **Active Directory Admin** account, not the SQL " +
			"Server Admin as AD users can only be administered with the AD Admin account. ",

		CreateContext: resourceMsSlqServerAadAccountCreate,
		ReadContext:   schema.NoopContext,
		DeleteContext: resourceMsSlqServerAadAccountDelete,

		Schema: map[string]*schema.Schema{
			sqlServerDnsProp: {
				Type:        schema.TypeString,
				Description: "The DNS name of the SQL server to add the account.",
				Required:    true,
				ForceNew:    true,
			},
			databaseProp: {
				Type:        schema.TypeString,
				Description: "The name of the database to add the account.",
				Required:    true,
				ForceNew:    true,
			},
			accountNameProp: {
				Type:        schema.TypeString,
				Description: "The name of the account to add to the database.",
				Required:    true,
				ForceNew:    true,
			},
			portProp: {
				Type:        schema.TypeInt,
				Description: "Port to connect to the database server.",
				Optional:    true,
				Default:     1433,
				ForceNew:    true,
			},
			objectIdProp: {
				Type:        schema.TypeString,
				Description: "Azure AD object ID for the account.",
				Required:    true,
				ForceNew:    true,
			},
			accountTypeProp: {
				Type:             schema.TypeString,
				Description:      "Type of account to create: either a single user or an AAD group.",
				Optional:         true,
				Default:          "user",
				ValidateDiagFunc: stringInStringMapKeys(accountTypeMap),
				ForceNew:         true,
			},
			roleProp: {
				Type:             schema.TypeString,
				Description:      "The role the account should get (e.g. owner, reader, etc.).",
				Optional:         true,
				Default:          "reader",
				ValidateDiagFunc: stringInStringMapKeys(roleMap),
				ForceNew:         true,
			},
		},
	}
}

func resourceMsSlqServerAadAccountCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn := createSQLConnection(d)
	diag := conn.createAccount(ctx, d)

	if !diag.HasError() {
		server := d.Get(sqlServerDnsProp).(string)
		database := d.Get(databaseProp).(string)
		port := d.Get(portProp).(int)
		account := d.Get(accountNameProp).(string)

		id := fmt.Sprint(server, ":", database, ":", port, "/", account)
		d.SetId(id)
	}

	return diag
}

func resourceMsSlqServerAadAccountDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	account := d.Get(accountNameProp).(string)

	conn := createSQLConnection(d)
	diag := conn.dropAccount(ctx, account)

	return diag
}

func getTokenProvider() (func() (string, error), error) {
	token, err := cli.GetTokenFromCLI("https://database.windows.net/")

	return func() (string, error) {
		return token.AccessToken, nil
	}, err
}

type sqlConnection struct {
	connectionString string
}

func createSQLConnection(d *schema.ResourceData) sqlConnection {
	server := d.Get(sqlServerDnsProp).(string)
	database := d.Get(databaseProp).(string)
	port := d.Get(portProp).(int)

	return sqlConnection{
		connectionString: fmt.Sprintf("Server=%v;Database=%v;Port=%v;", server, database, port),
	}
}

func (c sqlConnection) createAccount(ctx context.Context, d *schema.ResourceData) (diags diag.Diagnostics) {
	account := d.Get(accountNameProp).(string)
	objectId := d.Get(objectIdProp).(string)
	accountType, diags := stringFromMap(d, accountTypeProp, accountTypeMap, diags)
	role, diags := stringFromMap(d, roleProp, roleMap, diags)

	debugLog("[DEBUG] Setting account to %q..", account)
	debugLog("[DEBUG] Setting object_id to %q..", objectId)
	debugLog("[DEBUG] Setting accountType to %q..", accountType)
	debugLog("[DEBUG] Setting role to %q..", role)

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'CREATE USER ' + QuoteName(@account) + ' WITH SID=' + CONVERT(varchar(64), CAST(CAST(@objectId AS UNIQUEIDENTIFIER) AS VARBINARY(16)), 1) + ', TYPE=' + @accountType
			EXEC (@sql)
			SET @sql = 'ALTER ROLE ' + @role + ' ADD MEMBER ' + QuoteName(@account)
			EXEC (@sql)`

	return c.Execute(ctx, diags, cmd,
		sql.Named("account", account),
		sql.Named("objectId", objectId),
		sql.Named("accountType", accountType),
		sql.Named("role", role),
	)
}

func (c sqlConnection) dropAccount(ctx context.Context, account string) (diags diag.Diagnostics) {

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'DROP USER ' + QuoteName(@account)
			EXEC (@sql)`

	return c.Execute(ctx, diags, cmd, sql.Named("account", account))
}

func (c sqlConnection) Execute(ctx context.Context, diags diag.Diagnostics, command string, args ...interface{}) diag.Diagnostics {
	tokenProvider, err := getTokenProvider()
	if err != nil {
		return diag.FromErr(err)
	}

	connector, err := mssql.NewAccessTokenConnector(c.connectionString, tokenProvider)
	if err != nil {
		return diag.FromErr(err)
	}

	conn := sql.OpenDB(connector)
	defer conn.Close()

	debugLog("[DEBUG] Executing command %q..", command)

	_, err = conn.ExecContext(ctx, command, args...)
	if err != nil {
		return diag.Errorf("error executing statement (%s) (%s): %s", command, c.connectionString, err)
	}

	return diags
}

func debugLog(f string, v ...interface{}) {
	if os.Getenv("TF_LOG") == "" {
		return
	}

	if os.Getenv("TF_ACC") != "" {
		return
	}

	log.Printf(f, v...)
}
