package monitor

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *MonitorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg monitorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.Type.IsUnknown() || cfg.Type.IsNull() {
		return
	}
	typeName := cfg.Type.ValueString()

	validateExactlyOneParamsBlock(&cfg, typeName, &resp.Diagnostics)
	if typeName == "metric" && cfg.ParamsMetric != nil {
		validateDetector(cfg.ParamsMetric, &resp.Diagnostics)
	}
}

func validateExactlyOneParamsBlock(cfg *monitorModel, typeName string, diags *diag.Diagnostics) {
	blocks := []struct {
		name  string
		isSet bool
	}{
		{"params_error", cfg.ParamsError != nil},
		{"params_metric", cfg.ParamsMetric != nil},
	}
	expected := "params_" + typeName
	for _, b := range blocks {
		switch {
		case b.name == expected && !b.isSet:
			diags.AddAttributeError(
				path.Root(b.name),
				"missing params block",
				fmt.Sprintf("%s is required when type is %q.", b.name, typeName),
			)
		case b.name != expected && b.isSet:
			diags.AddAttributeError(
				path.Root(b.name),
				"unexpected params block",
				fmt.Sprintf("%s must not be configured when type is %q.", b.name, typeName),
			)
		}
	}
}

func validateDetector(m *metricParamsModel, diags *diag.Diagnostics) {
	if m.Detector == nil {
		diags.AddAttributeError(
			path.Root("params_metric").AtName("detector"),
			"detector is required",
			"params_metric.detector must be set for a metric monitor.",
		)
		return
	}
	manualSet := m.Detector.Manual != nil
	autoSet := m.Detector.Auto != nil
	switch {
	case manualSet && autoSet:
		diags.AddAttributeError(
			path.Root("params_metric").AtName("detector"),
			"detector must set exactly one kind",
			"detector must set exactly one of `manual` or `auto`, not both.",
		)
	case !manualSet && !autoSet:
		diags.AddAttributeError(
			path.Root("params_metric").AtName("detector"),
			"detector must set exactly one kind",
			"detector must set exactly one of `manual` or `auto`.",
		)
	}
}
