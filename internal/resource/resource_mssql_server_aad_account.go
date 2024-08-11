package resource

import (
	"context"
	"fmt"

	ssoSql "terraform-provider-sqlsso/internal/sql"

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

	accountType, accOk := accountTypeMap[plan.AccountType.ValueString()]
	role, roleOk := roleMap[plan.Role.ValueString()]

	if !accOk {
		resp.Diagnostics.AddError("internal error", fmt.Sprintf("Invalid account type %q", accountType))
	}

	if !roleOk {
		resp.Diagnostics.AddError("internal error", fmt.Sprintf("Invalid role %q", role))
	}

	if !accOk || !roleOk {
		return
	}

	conn := ssoSql.CreateMssqlConnection(plan.SqlServer.ValueString(), plan.Database.ValueString(), plan.Port.ValueInt64(), plan.Account.ValueString(), plan.ObjectId.ValueString(), accountType, role)
	conn.CreateAccount(ctx, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	id := conn.Id()
	plan.ID = types.StringValue(id)

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

	conn := ssoSql.CreateMssqlConnection(state.SqlServer.ValueString(), state.Database.ValueString(), state.Port.ValueInt64(), state.Account.ValueString(), state.ObjectId.ValueString(), state.AccountType.ValueString(), state.Role.ValueString())
	conn.DropAccount(ctx, &resp.Diagnostics)
}
