package resource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"golang.org/x/exp/maps"
)

type stringInMapValidator struct {
	Valid []string
}

func stringInMap[T any](v map[string]T) stringInMapValidator {
	return stringInMapValidator{
		Valid: maps.Keys(v),
	}
}

func (v stringInMapValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("string must must be present in: %v", v.Valid)
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v stringInMapValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("string must must be present in: %v", v.Valid)
}

func (v stringInMapValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the value is unknown or null, there is nothing to validate.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	s := req.ConfigValue.ValueString()

	for _, str := range v.Valid {
		if s == str {
			return
		}
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Unknown Value",
		"Unknown value",
	)
}
