package sql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type SqlConnection interface {
	getConnectionString() string
	createConnection() (*sql.DB, error)
}

func Execute(ctx context.Context, c SqlConnection, diags *diag.Diagnostics, command string, args ...interface{}) {
	conn, err := c.createConnection()
	if err != nil {
		diags.AddError("error", err.Error())
		return
	}
	defer conn.Close()

	tflog.Debug(ctx, fmt.Sprintf("Executing command %q..", command))

	_, err = conn.ExecContext(ctx, command, args...)
	if err != nil {
		diags.AddError("statement error", fmt.Sprintf("error executing statement (%s) (%s): %s", command, c.getConnectionString(), err))
	}
}
