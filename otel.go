package totp

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	tracer trace.Tracer
)

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func setupOTelSDK(ctx context.Context) (func(), error) {
	// Configure a new OTLP exporter
	client := otlptracegrpc.NewClient()
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, err
	}

	// Create a new tracer provider with a batch span processor and the otlp exporter
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp))

	// Register the global Tracer provider
	otel.SetTracerProvider(tp)

	tracer = otel.Tracer("github.com/voidshard/totp")

	// Register the W3C trace context and baggage propagators so data is propagated across services/processes
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return func() {
		_ = exp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
	}, err
}

// otelWrapHandler wraps an HTTP handler with OpenTelemetry instrumentation.
func otelWrapHandler(h http.Handler, name string) http.Handler {
	return otelhttp.NewHandler(h, name)
}
