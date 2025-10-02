package tracing

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// InitTracer инициализирует OpenTelemetry трейсер с Jaeger экспортером
func InitTracer(serviceName string) (*tracesdk.TracerProvider, error) {
	jaegerHost := getEnv("JAEGER_AGENT_HOST", "localhost")
	jaegerPort := getEnv("JAEGER_AGENT_PORT", "6831")

	exp, err := jaeger.New(
		jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(jaegerHost),
			jaeger.WithAgentPort(jaegerPort),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Printf("[Tracing] Initialized for service '%s' at %s:%s", serviceName, jaegerHost, jaegerPort)

	return tp, nil
}

// Shutdown корректно завершает работу трейсера
func Shutdown(ctx context.Context, tp *tracesdk.TracerProvider) {
	if tp == nil {
		return
	}
	if err := tp.Shutdown(ctx); err != nil {
		log.Printf("[Tracing] Error shut down tracer: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
