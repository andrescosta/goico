Config:

  ## Obs configs
obs.enabled=true
obs.exporter.stdout=false
obs.exporter.trace=true
obs.exporter.trace.grpc.host=localhost:4317
#obs.exporter.trace.http.host=
obs.exporter.metrics=true
obs.exporter.metrics.http.host=localhost:9090
obs.exporter.metrics.host.path=/api/v1/otlp/v1/metrics
#obs.exporter.metrics.grpc.host
obs.interval = 10s
obs.metrics.host=true
obs.metrics.runtime=true


https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/google.golang.org/grpc/otelgrpc/example/server/main.go

https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main


Query for prometheus: {job='ctl'}

https://promlabs.com/blog/2020/12/17/promql-queries-for-exploring-your-metrics/
https://medium.com/jaegertracing/introducing-native-support-for-opentelemetry-in-jaeger-eb661be8183c
https://opentelemetry.io/docs/specs/otel/logs/


docker run --name jaeger `
  -e COLLECTOR_OTLP_ENABLED=true `
  -p 16686:16686 `
  -p 4317:4317 `
  -p 4318:4318 `
  jaegertracing/all-in-one:1.35

docker run `
    -p 9090:9090 `
    -v C:\Users\Andres\projects\go\jobico\obs\prom:/prometheus `
    prom/prometheus --enable-feature=otlp-write-receiver

https://github.com/open-telemetry/opentelemetry-demo/blob/main/docker-compose.yml

prometheus.yml

# my global config
global:
  scrape_interval: 15s # Set the scrape interval to every 15 seconds. Default is every 1 minute.
  evaluation_interval: 15s # Evaluate rules every 15 seconds. The default is every 1 minute.
  # scrape_timeout is set to the global default (10s).

# Alertmanager configuration
alerting:
  alertmanagers:
    - static_configs:
        - targets:
          # - alertmanager:9093

# Load rules once and periodically evaluate them according to the global 'evaluation_interval'.
rule_files:
  # - "first_rules.yml"
  # - "second_rules.yml"

# A scrape configuration containing exactly one endpoint to scrape:
# Here it's Prometheus itself.
scrape_configs:
  # The job name is added as a label `job=<job_name>` to any timeseries scraped from this config.
  - job_name: "prometheus"

    # metrics_path defaults to '/metrics'
    # scheme defaults to 'http'.

    static_configs:
      - targets: ["localhost:9090"]


#  - job_name: "node-exporter"
#    static_configs:
#      - targets: ["localhost:9100"]    