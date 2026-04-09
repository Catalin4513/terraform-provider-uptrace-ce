package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/catalin4513/terraform-provider-uptrace-ce/internal/services/org"
)

// ServiceRegistration is implemented by every service package under
// internal/services. Add a new service by appending to the services slice.
type ServiceRegistration interface {
	Name() string
	Resources() []func() resource.Resource
	DataSources() []func() datasource.DataSource
}

var services = []ServiceRegistration{
	org.Registration{},
}
