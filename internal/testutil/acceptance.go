package testutil

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/provider"
)

// ProtoV6ProviderFactories returns the provider factories for acceptance tests.
// Used as ProtoV6ProviderFactories field in resource.TestCase.
var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"uptrace": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// PreCheck validates that required environment variables are set.
// Call from every acceptance test's PreCheck function.
func PreCheck(t *testing.T) {
	t.Helper()
	LoadEnv()
	if os.Getenv("UPTRACE_ENDPOINT") == "" {
		t.Fatal("UPTRACE_ENDPOINT must be set for acceptance tests")
	}
	if os.Getenv("UPTRACE_TOKEN") == "" {
		t.Fatal("UPTRACE_TOKEN must be set for acceptance tests")
	}
}
