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
	_ resource.Resource = &postgreResource{}
)

// New is a helper function to simplify the provider implementation.
func NewPostgre() resource.Resource {
	return &postgreResource{}
}

type postgreResource struct {
}

type postgreResourceModel struct {
	ID        types.String `tfsdk:"id"`
	SqlServer types.String `tfsdk:"sql_server_dns"`
	Database  types.String `tfsdk:"database"`
	UserName  types.String `tfsdk:"user_name"`
	Account   types.String `tfsdk:"account_name"`
	Port      types.Int64  `tfsdk:"port"`
	Role      types.String `tfsdk:"role"`
}

func (d *postgreResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresql_server_aad_account"
}

// Schema defines the schema for the resource.
func (d *postgreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			userNameProp: schema.StringAttribute{
				Description: "The name of the account that will log into the database (not currently infered from connection).",
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
				Default:     int64default.StaticInt64(5432),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
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

func (d *postgreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state postgreResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Could read status from the database and update the state

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *postgreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan postgreResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn := ssoSql.CreatePostgreConnection(plan.SqlServer.ValueString(), plan.Database.ValueString(), plan.Port.ValueInt64(), plan.UserName.ValueString())

	role, roleOk := roleMap[plan.Role.ValueString()]

	if !roleOk {
		resp.Diagnostics.AddError("internal error", fmt.Sprintf("Invalid role %q", role))
		return
	}

	conn.CreatePostgreAccount(ctx, plan.Account.ValueString(), &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	id := fmt.Sprint(plan.SqlServer.ValueString(), ":", plan.Database.ValueString(), ":", plan.Port, "/", plan.Account.ValueString())
	plan.ID = types.StringValue(id)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (d *postgreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Noop (any change requires delete and create)
}

func (d *postgreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state postgreResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	account := state.Account

	conn := ssoSql.CreatePostgreConnection(state.SqlServer.ValueString(), state.Database.ValueString(), state.Port.ValueInt64(), state.Account.ValueString())
	conn.DropPostgreAccount(ctx, account.ValueString(), &resp.Diagnostics)
}
