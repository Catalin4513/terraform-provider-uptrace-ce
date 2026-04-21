package tfutil

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// FromProviderData unwraps the value the provider stashes in ProviderData
// during Configure. Returns nil during pre-configure (ProviderData is still
// nil) without emitting diagnostics; emits a diagnostic error and returns nil
// for an unexpected type.
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
