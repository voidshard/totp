package totp

import (
	"context"

	"github.com/honeycombio/otel-config-go/otelconfig"
)

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func setupOTelSDK(ctx context.Context) (func(), error) {
	return otelconfig.ConfigureOpenTelemetry()
}
