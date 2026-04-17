package client

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringFromPtr maps *string → types.String; nil or "" → Null.
func StringFromPtr(s *string) types.String {
	if s == nil || *s == "" {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// PreserveIfEmpty returns the API value, falling back to prior when empty.
// Use for Sensitive fields the backend may redact on refresh.
func PreserveIfEmpty(api string, prior types.String) types.String {
	if api == "" {
		return prior
	}
	return types.StringValue(api)
}

// PreserveIfEmptyPtr is PreserveIfEmpty for *string API fields.
func PreserveIfEmptyPtr(api *string, prior types.String) types.String {
	if api == nil || *api == "" {
		return prior
	}
	return types.StringValue(*api)
}

// EnumToValue maps *Enum (string-backed) → types.String; nil or "" → Null.
func EnumToValue[T ~string](p *T) types.String {
	if p == nil || *p == "" {
		return types.StringNull()
	}
	return types.StringValue(string(*p))
}

// EnumToValueOrDefault is EnumToValue that falls back to a schema default
// instead of Null, so attributes with a Default stay consistent with plan.
func EnumToValueOrDefault[T ~string](p *T, fallback string) types.String {
	if p != nil && *p != "" {
		return types.StringValue(string(*p))
	}
	return types.StringValue(fallback)
}

// IntPtrToValue maps *int → types.Int64. Nil or zero with a null prior → Null;
// otherwise the API value surfaces so drift stays visible.
func IntPtrToValue(api *int, prior types.Int64) types.Int64 {
	if api == nil {
		return prior
	}
	if *api == 0 && prior.IsNull() {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*api))
}

// PriorString extracts a types.String field from a possibly-nil block pointer.
func PriorString[T any](prior *T, get func(*T) types.String) types.String {
	if prior == nil {
		return types.StringNull()
	}
	return get(prior)
}

// PriorInt64 is PriorString for types.Int64 fields.
func PriorInt64[T any](prior *T, get func(*T) types.Int64) types.Int64 {
	if prior == nil {
		return types.Int64Null()
	}
	return get(prior)
}
