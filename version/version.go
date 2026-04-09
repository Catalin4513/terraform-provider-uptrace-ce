// Package version exposes the provider version string. Override at build
// time with:
//
//	-ldflags "-X github.com/catalin4513/terraform-provider-uptrace-ce/version.ProviderVersion=v1.2.3"
package version

var ProviderVersion = "dev"
