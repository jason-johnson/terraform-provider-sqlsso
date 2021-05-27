package provider

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func stringInStringMapKeys(m map[string]string) schema.SchemaValidateDiagFunc {
	mapKeys := make([]string, 0, len(m))
	for k := range m {
		mapKeys = append(mapKeys, k)
	}

	return stringInSlice(mapKeys)
}

func stringInSlice(valid []string) schema.SchemaValidateDiagFunc {
	f := validation.StringInSlice(valid, false)

	return func(v interface{}, path cty.Path) (diags diag.Diagnostics) {

		warnings, errors := f(v, fmt.Sprintf("%s", path))

		for _, warn := range warnings {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  warn,
			})
		}

		for _, error := range errors {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  error.Error(),
			})
		}

		return diags
	}
}

func stringFromMap(d *schema.ResourceData, k string, m map[string]string, diags diag.Diagnostics) (string, diag.Diagnostics) {
	v, dok := d.Get(k).(string)

	if !dok {
		diags = appendError(diags, fmt.Sprintf("%q not set", k))
	}

	result, mok := m[v]

	if !mok {
		diags = appendError(diags, fmt.Sprintf("%q not found in map %v", v, m))
	}

	return result, diags
}

func appendError(diags diag.Diagnostics, message string) diag.Diagnostics {
	diags = append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  message,
		Detail:   message,
	})
	return diags
}
