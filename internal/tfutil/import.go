package tfutil

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// ImportStateCompoundID parses a ":"-separated compound import ID and writes
// each segment to a matching state attribute. The last field is mapped to
// "id" (Terraform primary-key convention); earlier fields are written to
// attributes whose name equals the field name.
func ImportStateCompoundID(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
	fields ...string,
) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != len(fields) {
		resp.Diagnostics.AddError(
			"invalid import ID",
			fmt.Sprintf("expected format <%s>, got %q", strings.Join(fields, ">:<"), req.ID),
		)
		return
	}
	for i, name := range fields {
		if parts[i] == "" {
			resp.Diagnostics.AddError(
				"invalid import ID",
				fmt.Sprintf("%s segment must not be empty", name),
			)
			return
		}
	}
	for i, name := range fields {
		attr := name
		if i == len(fields)-1 {
			attr = "id"
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr), parts[i])...)
	}
}
