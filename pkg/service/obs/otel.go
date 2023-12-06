package obs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/service/svcmeta"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
)

type OtelProvider struct {
	shutdownFuncs []func(context.Context) error
}

var interval = 20 * time.Second

func Init(ctx context.Context, info svcmeta.Info) (*OtelProvider, error) {
	var shutdownFuncs []func(context.Context) error
	var err error

	// sutdown funcs
	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// error handler
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// otel resource
	res, err := newResource(info.Name, info.Version)
	if err != nil {
		handleErr(err)
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Set up trace provider.
	if env.GetAsBool("obs.trace.stdout") {
		tracerProvider, err := newStdoutTraceProvider(res)
		if err != nil {
			handleErr(err)
			return nil, err
		}
		shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
		otel.SetTracerProvider(tracerProvider)
		// Set up meter provider.
		meterProvider, err := newStdoutMeterProvider(res)
		if err != nil {
			handleErr(err)
			return nil, err
		}
		shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
		otel.SetMeterProvider(meterProvider)

		metricProvider, err := newMetric(res)
		if err != nil {
			handleErr(err)
			return nil, err
		}
		shutdownFuncs = append(shutdownFuncs, metricProvider.Shutdown)
		otel.SetMeterProvider(metricProvider)

		meterProviderForHost, err := newStdoutMeterProvider(res)
		if err != nil {
			handleErr(err)
			return nil, err
		}
		shutdownFuncs = append(shutdownFuncs, meterProviderForHost.Shutdown)
		err = host.Start(host.WithMeterProvider(meterProviderForHost))
		if err != nil {
			handleErr(err)
			return nil, err
		}
	}

	if env.GetAsBool("obs.trace.promotheus") {
		exporter, err := prometheus.New()
		if err != nil {
			log.Fatal(err)
		}

		provider := metric.NewMeterProvider(metric.WithReader(exporter))
		_ = provider.Meter(info.Name)
		otel.SetMeterProvider(provider)
	}

	// Test: otelHandler := otelhttp.NewHandler(http.HandlerFunc(helloHandler), "Hello")

	return &OtelProvider{
		shutdownFuncs: shutdownFuncs,
	}, nil
}

func newResource(serviceName, serviceVersion string) (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		))
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newStdoutTraceProvider(res *resource.Resource) (*trace.TracerProvider, error) {
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	return newTraceProvider(traceExporter, res)
}

func newMetric(res *resource.Resource) (*metric.MeterProvider, error) {
	exp, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}
	return newMetricProvider(exp, res)

}

func newMetricProvider(exp metric.Exporter, res *resource.Resource) (*metric.MeterProvider, error) {
	read := metric.NewPeriodicReader(exp, metric.WithInterval(interval))
	provider := metric.NewMeterProvider(metric.WithResource(res), metric.WithReader(read))
	return provider, nil
}

func newTraceProvider(traceExporter *stdouttrace.Exporter, res *resource.Resource) (*trace.TracerProvider, error) {
	traceProvider := trace.NewTracerProvider(
		// In a production application, use sdktrace.ProbabilitySampler with a desired probability.
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithBatcher(traceExporter,

			trace.WithBatchTimeout(interval)),
		trace.WithResource(res),
	)
	return traceProvider, nil
}

func newStdoutMeterProvider(res *resource.Resource) (*metric.MeterProvider, error) {
	metricExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}
	return newMeterProvider(res, metricExporter)
}

func newMeterProvider(res *resource.Resource, metricExporter metric.Exporter) (*metric.MeterProvider, error) {
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter,

			metric.WithInterval(interval))),
	)
	return meterProvider, nil
}

func Use(r *mux.Router) {
	r.Use(otelmux.Middleware("my-server"))
}

func StatsForOtel() grpc.ServerOption {
	x := grpc.StatsHandler(otelgrpc.NewServerHandler())
	return x
}

func StartRuntime() error {
	return runtime.Start(runtime.WithMinimumReadMemStatsInterval(interval))
}

func serveMetrics() {
	log.Printf("serving metrics at localhost:2223/metrics")
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":2223", nil) //nolint:gosec // Ignoring G114: Use of net/http serve function that has no support for setting timeouts.
	if err != nil {
		fmt.Printf("error serving http: %v", err)
		return
	}
}
