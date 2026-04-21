package tfutil

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// FromProviderData unwraps the provider's ProviderData. Nil ProviderData
// (pre-Configure) returns nil silently; a wrong type emits a diagnostic.
func FromProviderData[T any](providerData any, diags *diag.Diagnostics) *T {
	if providerData == nil {
		return nil
	}
	c, ok := providerData.(*T)
	if !ok {
		diags.AddError(
			"unexpected provider data type",
			fmt.Sprintf("expected *%T, got %T", new(T), providerData),
		)
		return nil
	}
	return c
}

// ImportStateCompoundID parses "a:b:...:id" and writes each segment to the
// matching attribute. The last field is mapped to "id".
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
