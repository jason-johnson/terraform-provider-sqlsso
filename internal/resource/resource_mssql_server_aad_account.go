package resource

import (
	"context"
	"database/sql"
	"fmt"

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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &mssqlResource{}
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
	ID          types.String `tfsdk:"id"`
	SqlServer   types.String `tfsdk:"sql_server_dns"`
	Database    types.String `tfsdk:"database"`
	Account     types.String `tfsdk:"account_name"`
	Port        types.Int64  `tfsdk:"port"`
	ObjectId    types.String `tfsdk:"object_id"`
	AccountType types.String `tfsdk:"account_type"`
	Role        types.String `tfsdk:"role"`
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				Computed:    true,
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
				Computed:    true,
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
				Computed:    true,
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

func (d *mssqlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mssqlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Could read status from the database and update the state

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *mssqlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mssqlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn := createSQLConnection(plan)
	conn.createAccount(ctx, plan, &resp.Diagnostics)

	if !resp.Diagnostics.HasError() {
		id := fmt.Sprint(plan.SqlServer, ":", plan.Database, ":", plan.Port, "/", plan.Account)
		plan.ID = types.StringValue(id)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (d *mssqlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Noop (any change requires delete and create)
}

func (d *mssqlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mssqlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	account := state.Account

	conn := createSQLConnection(state)
	conn.dropAccount(ctx, account.ValueString(), &resp.Diagnostics)
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

func createSQLConnection(config mssqlResourceModel) sqlConnection {
	return sqlConnection{
		connectionString: fmt.Sprintf("Server=%v;Database=%v;Port=%v;", config.SqlServer, config.Database, config.Port),
	}
}

func (c sqlConnection) createAccount(ctx context.Context, config mssqlResourceModel, diags *diag.Diagnostics) {
	account := config.Account
	objectId := config.ObjectId
	accountType, accOk := accountTypeMap[config.AccountType.ValueString()]
	role, roleOk := roleMap[config.Role.ValueString()]

	if !accOk {
		diags.AddError("internal error", fmt.Sprintf("Invalid account type %q", accountType))
	}

	if !roleOk {
		diags.AddError("internal error", fmt.Sprintf("Invalid role %q", role))
	}

	if !accOk || !roleOk {
		return
	}

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

	c.Execute(ctx, diags, cmd,
		sql.Named("account", account),
		sql.Named("objectId", objectId),
		sql.Named("accountType", accountType),
		sql.Named("role", role),
	)
}

func (c sqlConnection) dropAccount(ctx context.Context, account string, diags *diag.Diagnostics) {

	cmd := `DECLARE @sql nvarchar(max)
			SET @sql = 'DROP USER ' + QuoteName(@account)
			EXEC (@sql)`

	c.Execute(ctx, diags, cmd, sql.Named("account", account))
}

func (c sqlConnection) Execute(ctx context.Context, diags *diag.Diagnostics, command string, args ...interface{}) {
	tokenProvider, err := getTokenProvider()
	if err != nil {
		diags.AddError("error getting azcli token", err.Error())
		return
	}

	connector, err := mssql.NewAccessTokenConnector(c.connectionString, tokenProvider)
	if err != nil {
		diags.AddError("error", err.Error())
		return
	}

	conn := sql.OpenDB(connector)
	defer conn.Close()

	tflog.Debug(ctx, fmt.Sprintf("Executing command %q..", command))

	_, err = conn.ExecContext(ctx, command, args...)
	if err != nil {
		diags.AddError("statement error", fmt.Sprintf("error executing statement (%s) (%s): %s", command, c.connectionString, err))
	}
}
