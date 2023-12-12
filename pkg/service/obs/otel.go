package obs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/service/svcmeta"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OtelProvider struct {
	enabled       bool
	traceProvider *sdktrace.TracerProvider
	meterProvider *metric.MeterProvider
}

func New(ctx context.Context, info svcmeta.Info) (*OtelProvider, error) {
	if !env.GetAsBool("obs.enabled") {
		return &OtelProvider{enabled: false}, nil
	}

	logger := zerolog.Ctx(ctx)

	interval := *env.GetAsDuration("obs.interval", 20*time.Second)

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
	res, err := newResource(ctx, info.Name, info.Version)
	if err != nil {
		return nil, err
	}

	// Configuring Otel Signals:  Metrics(meters), traces, baggage (logs not implemented by otel)
	otel.SetTextMapPropagator(newPropagator())

	// Set exporters
	var batches []sdktrace.SpanExporter
	var exporters []metric.Exporter

	//// Set stdout exporter
	if env.GetAsBool("obs.exporter.stdout") {
		// trace
		b, err := stdouttrace.New(
			stdouttrace.WithPrettyPrint())
		if err != nil {
			handleErr(err)
			return nil, err
		}
		batches = append(batches, b)

		// metrics
		e, err := stdoutmetric.New()
		if err != nil {
			handleErr(err)
			return nil, err
		}
		exporters = append(exporters, e)
	}

	//// Set oltp grpc exporters
	traceAddrGrpc := env.GetOrNil("obs.exporter.trace.grpc.host")
	traceAddrHttp := env.GetOrNil("obs.exporter.trace.http.host")
	metricAddrGrpc := env.GetOrNil("obs.exporter.metrics.grpc.host")
	metricAddrHttp := env.GetOrNil("obs.exporter.metrics.http.host")
	var meterProvider *metric.MeterProvider
	var traceProvider *sdktrace.TracerProvider

	if env.GetAsBool("obs.exporter.trace") || traceAddrGrpc != nil || traceAddrHttp != nil {
		if traceAddrGrpc == nil && traceAddrHttp == nil { // in case trace is enabled but addrs are not configured
			logger.Warn().Msg("trace exporter not enabled because hosts are not configured using obs.exporter.trace.grpc.host or obs.exporter.trace.http.host")
		} else {
			if traceAddrGrpc != nil {
				ctx, done := context.WithTimeout(ctx, 5*time.Second)
				conn, err := grpc.DialContext(ctx, *traceAddrGrpc,
					grpc.WithTransportCredentials(insecure.NewCredentials()),
					grpc.WithBlock(),
				)
				done()
				if err != nil {
					handleErr(err)
					return nil, fmt.Errorf("failed to create trace exporter: %w", err)
				}

				// trace
				e, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
				if err != nil {
					handleErr(err)
					return nil, fmt.Errorf("failed to create trace exporter: %w", err)
				}
				batches = append(batches, e)
			}
			if traceAddrHttp != nil {
				e, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(*traceAddrHttp))
				if err != nil {
					handleErr(err)
					return nil, err
				}
				batches = append(batches, e)
			}

			// TODO: In a production application, use sdktrace.ProbabilitySampler with a desired probability.
			traceProvider, err = newTraceProviderArr(batches, sdktrace.AlwaysSample(), res, interval)
			if err != nil {
				handleErr(err)
				return nil, err
			}
			otel.SetTracerProvider(traceProvider)

		}
	}
	// metrics
	if env.GetAsBool("obs.exporter.metrics") || metricAddrGrpc != nil || metricAddrHttp != nil {
		if metricAddrGrpc == nil && metricAddrHttp == nil { // in case metric is enabled but addrs are not configured
			logger.Warn().Msg("meter exporter not enabled because hosts are not configured using obs.exporter.metrics.grpc.host or obs.exporter.metrics.http.host")
		} else {
			if metricAddrGrpc != nil {
				ctx, done := context.WithTimeout(ctx, 5*time.Second)
				conn, err := grpc.DialContext(ctx, *metricAddrGrpc,
					grpc.WithTransportCredentials(insecure.NewCredentials()),
					grpc.WithBlock(),
				)
				done()
				if err != nil {
					handleErr(err)
					return nil, fmt.Errorf("failed to create neter exporter: %w", err)
				}
				m, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
				if err != nil {
					handleErr(err)
					return nil, err
				}
				exporters = append(exporters, m)
			}
			if metricAddrHttp != nil {
				var ops []otlpmetrichttp.Option
				ops = append(ops, otlpmetrichttp.WithInsecure(), otlpmetrichttp.WithEndpoint(*metricAddrHttp))
				p := env.GetOrNil("obs.exporter.metrics.host.path")
				if p != nil {
					ops = append(ops, otlpmetrichttp.WithURLPath(*p))
				}
				m, err := otlpmetrichttp.New(ctx, ops...)
				if err != nil {
					handleErr(err)
					return nil, err
				}
				exporters = append(exporters, m)
			}

			meterProvider, err = newMeterProviderArr(res, exporters, interval)
			if err != nil {
				if traceProvider != nil {
					traceProvider.Shutdown(ctx)
				}
				return nil, err
			}
			otel.SetMeterProvider(meterProvider)

		}
	}

	if env.GetAsBool("obs.metrics.host") {
		err = host.Start()
		if err != nil {
			traceProvider.Shutdown(ctx)
			if meterProvider != nil {
				meterProvider.Shutdown(ctx)
			}
			return nil, err
		}
	}

	//runtime telemetry
	if env.GetAsBool("obs.metrics.runtime") {
		err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(interval))
		if err != nil {
			traceProvider.Shutdown(ctx)
			if meterProvider != nil {
				meterProvider.Shutdown(ctx)
			}
			return nil, err
		}
	}

	// TODO: Test: otelHandler := otelhttp.NewHandler(http.HandlerFunc(helloHandler), "Hello")
	return &OtelProvider{
		enabled:       true,
		traceProvider: traceProvider,
		meterProvider: meterProvider,
	}, nil
}

func (o *OtelProvider) Shutdown(ctx context.Context) error {
	var errf error
	if o.traceProvider != nil {
		if err := o.traceProvider.Shutdown(ctx); err != nil {
			errf = errors.Join(err)
		}
	}
	if o.meterProvider != nil {
		if err := o.meterProvider.Shutdown(ctx); err != nil {
			errf = errors.Join(err, errf)
		}
	}
	return errf
}

func newResource(ctx context.Context, serviceName, serviceVersion string) (*resource.Resource, error) {
	var attrs []attribute.KeyValue
	if serviceName != "" {
		attrs = append(attrs, semconv.ServiceName(serviceName))
	}
	if serviceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(serviceVersion))
	}
	r1 := resource.Default()
	r2 := resource.NewWithAttributes(semconv.SchemaURL, attrs...)
	res, err := resource.Merge(r1, r2)
	if err != nil {
		return nil, err
	}
	zerolog.Ctx(ctx).Debug().Msgf("Obs resource name: %s", res.String())
	return res, nil

}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProviderArr(exps []sdktrace.SpanExporter, s sdktrace.Sampler, res *resource.Resource, interval time.Duration) (*sdktrace.TracerProvider, error) {
	var opts []sdktrace.TracerProviderOption
	opts = append(opts, sdktrace.WithSampler(s))
	opts = append(opts, sdktrace.WithResource(res))
	for _, e := range exps {
		opts = append(opts, sdktrace.WithBatcher(e, sdktrace.WithBatchTimeout(interval)))
	}

	traceProvider := sdktrace.NewTracerProvider(opts...)
	return traceProvider, nil
}

func newMeterProviderArr(res *resource.Resource, exps []metric.Exporter, interval time.Duration) (*metric.MeterProvider, error) {
	var opts []metric.Option
	opts = append(opts, metric.WithResource(res))

	for _, e := range exps {
		opts = append(opts, metric.WithReader(
			metric.NewPeriodicReader(e, metric.WithInterval(interval))))
	}

	meterProvider := metric.NewMeterProvider(opts...)
	return meterProvider, nil
}

func (o *OtelProvider) InstrumentMuxRouter(name string, r *mux.Router) {
	if o.enabled {
		r.Use(otelmux.Middleware(name))
	}
}

func (o *OtelProvider) InstrumentGrpcServer() grpc.ServerOption {
	var x grpc.ServerOption
	x = grpc.EmptyServerOption{}
	if o.enabled {
		x = grpc.StatsHandler(otelgrpc.NewServerHandler())
	}
	return x
}
