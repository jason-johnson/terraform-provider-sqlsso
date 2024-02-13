package provider

import (
	"context"

	sqlsso "terraform-provider-sqlsso/internal/resource"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ provider.Provider = &sqlssoProvider{}
)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &sqlssoProvider{
			version: version,
		}
	}
}

type sqlssoProvider struct {
	version string
}

type sqlssoProviderModel struct {
}

func (p *sqlssoProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sqlsso"
	resp.Version = p.version
}

func (p *sqlssoProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{},
	}
}

func (p *sqlssoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config sqlssoProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (p *sqlssoProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *sqlssoProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		sqlsso.NewMssql,
	}
}
