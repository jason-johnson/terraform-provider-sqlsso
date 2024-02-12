package resource

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &mssqlResource{}
	_ resource.ResourceWithConfigure = &mssqlResource{}
)

var accountTypeMap = map[string]string{"user": "E", "group": "X"}
var roleMap = map[string]string{"owner": "db_owner", "reader": "db_datareader", "writer": "db_datawriter"}

// New is a helper function to simplify the provider implementation.
func NewMssql() resource.Resource {
	return &mssqlResource{}
}

type mssqlResource struct {
}

type mssqlResourceModel struct {
	ID      types.String `tfsdk:"id"`
	DNS     types.String `tfsdk:"sql_server_dns"`
	DB      types.String `tfsdk:"database"`
	AccName types.String `tfsdk:"account_name"`
	Port    types.Int64  `tfsdk:"port"`
	OID     types.String `tfsdk:"object_id"`
	AccType types.String `tfsdk:"account_type"`
	Role    types.String `tfsdk:"role"`
}

func (d *mssqlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mssql_server_aad_account"
}

// Schema defines the schema for the resource.
func (d *mssqlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "`sqlsso_mssql_server_aad_account` enables AAD authentication for an Azure MS SQL server.\n\nFor this to work terraform should be run for the configured **Active Directory Admin** account, not the SQL Server Admin as AD users can only be administered with the AD Admin account. ",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			sqlServerDnsProp: schema.StringAttribute{
				Description: "The DNS name of the SQL server to add the account.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			databaseProp: schema.StringAttribute{
				Description: "The name of the database to add the account.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			accountNameProp: schema.StringAttribute{
				Description: "The name of the account to add to the database.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			portProp: schema.Int64Attribute{
				Description: "Port to connect to the database server.",
				Optional:    true,
				Default:     int64default.StaticInt64(1433),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			objectIdProp: schema.StringAttribute{
				Description: "Azure AD object ID for the account.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			accountTypeProp: schema.StringAttribute{
				Description: "Type of account to create: either a single user or an AAD group.",
				Optional:    true,
				Default:     stringdefault.StaticString("user"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringInMap(accountTypeMap),
				},
			},
			roleProp: schema.StringAttribute{
				Description: "The role the account should get (e.g. owner, reader, etc.).",
				Optional:    true,
				Default:     stringdefault.StaticString("reader"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringInMap(roleMap),
				},
			},
		}}
}

func (d *mssqlResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
}

func (d *mssqlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

func (d *mssqlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var config mssqlResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *mssqlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config mssqlResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *mssqlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var config mssqlResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
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
