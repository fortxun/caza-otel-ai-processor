# Integration Examples

This guide provides examples of integrating the AI-Enhanced Telemetry Processor with various systems and tools.

## Integration with Distributed Tracing Systems

### Jaeger Integration

#### Configuration

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  ai_processor:
    models:
      error_classifier:
        path: "/models/error-classifier.wasm"
        memory_limit_mb: 100
        timeout_ms: 50
      importance_sampler:
        path: "/models/importance-sampler.wasm"
        memory_limit_mb: 80
        timeout_ms: 30
      entity_extractor:
        path: "/models/entity-extractor.wasm"
        memory_limit_mb: 150
        timeout_ms: 50
    
    processing:
      batch_size: 50
      concurrency: 4
    
    features:
      error_classification: true
      smart_sampling: true
      entity_extraction: true
    
    sampling:
      error_events: 1.0
      slow_spans: 1.0
      normal_spans: 0.1
      threshold_ms: 500
    
    output:
      attribute_namespace: "ai."
      include_confidence_scores: true

exporters:
  jaeger:
    endpoint: jaeger-collector:14250
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [ai_processor]
      exporters: [jaeger]
```

#### With Docker Compose

```yaml
version: '3'
services:
  # Jaeger
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # UI
      - "14250:14250"  # gRPC collector
    environment:
      - COLLECTOR_OTLP_ENABLED=true
  
  # OpenTelemetry Collector with AI Processor
  otel-collector:
    image: fortxun/caza-otel-ai-processor:latest
    volumes:
      - ./config:/config
      - ./models:/models
    ports:
      - "4317:4317"
      - "4318:4318"
    command: ["--config=/config/config.yaml"]
    depends_on:
      - jaeger
```

### Zipkin Integration

#### Configuration

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  ai_processor:
    # Similar configuration as above...

exporters:
  zipkin:
    endpoint: "http://zipkin:9411/api/v2/spans"

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [ai_processor]
      exporters: [zipkin]
```

## Integration with Log Management Systems

### Elasticsearch Integration

#### Configuration

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  ai_processor:
    # AI processor configuration...
  
  batch:
    timeout: 1s
    send_batch_size: 1000

exporters:
  elasticsearch:
    endpoints: ["https://elasticsearch:9200"]
    username: "${ELASTICSEARCH_USERNAME}"
    password: "${ELASTICSEARCH_PASSWORD}"
    mapping:
      mode: "index"
      index: "otel-logs-%{yyyy.MM.dd}"
      pipeline: "otel-pipeline"

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [ai_processor, batch]
      exporters: [elasticsearch]
```

#### Example Elasticsearch Mapping for AI Attributes

```json
{
  "template": {
    "mappings": {
      "properties": {
        "ai.error.category": {
          "type": "keyword"
        },
        "ai.error.system": {
          "type": "keyword"
        },
        "ai.error.owner": {
          "type": "keyword"
        },
        "ai.error.severity": {
          "type": "keyword"
        },
        "ai.error.impact": {
          "type": "keyword"
        },
        "ai.error.confidence": {
          "type": "float"
        },
        "ai.entity.services": {
          "type": "keyword"
        },
        "ai.entity.dependencies": {
          "type": "keyword"
        },
        "ai.entity.operations": {
          "type": "keyword"
        },
        "ai.importance": {
          "type": "float"
        }
      }
    }
  }
}
```

### Loki Integration

#### Configuration

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  ai_processor:
    # AI processor configuration...

exporters:
  loki:
    endpoint: "http://loki:3100/loki/api/v1/push"
    tenant_id: "tenant1"
    labels:
      resource:
        service.name: "service_name"
        service.namespace: "service_namespace"
      attributes:
        ai.error.category: "error_category"
        ai.error.owner: "error_owner"
        ai.error.severity: "error_severity"

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [ai_processor]
      exporters: [loki]
```

## Integration with Metrics Systems

### Prometheus Integration

#### Configuration

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  ai_processor:
    # AI processor configuration...

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"
    namespace: "otel"
    send_timestamps: true
    metric_expiration: 180m
    resource_to_telemetry_conversion:
      enabled: true

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [ai_processor]
      exporters: [prometheus]
```

#### Example Prometheus Queries Utilizing AI Attributes

```promql
# Count of errors by category
sum by (ai_error_category) (count_over_time({ai_error_category=~".+"}[15m]))

# Error rate by owner
sum by (ai_error_owner) (rate({ai_error_owner=~".+"}[5m]))

# High severity errors
sum(count_over_time({ai_error_severity="high"}[15m]))
```

## Integration with Alerting Systems

### Alertmanager Integration

```yaml
# Prometheus Alerting Rules Using AI Attributes
groups:
- name: ai-enhanced-alerts
  rules:
  - alert: HighSeverityErrors
    expr: sum(count_over_time({ai_error_severity="high"}[5m])) > 10
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "High number of severe errors"
      description: "There have been {{ $value }} high severity errors in the last 5 minutes"
  
  - alert: DatabaseErrors
    expr: sum(count_over_time({ai_error_category="database_error"}[5m])) > 5
    for: 2m
    labels:
      severity: warning
      team: "{{ $labels.ai_error_owner }}"
    annotations:
      summary: "Database errors detected"
      description: "There have been {{ $value }} database errors in the last 5 minutes"
  
  - alert: ServiceDegradation
    expr: sum(rate({ai_importance>0.8}[5m])) / sum(rate({job="application"}[5m])) > 0.2
    for: 3m
    labels:
      severity: warning
    annotations:
      summary: "Service degradation detected"
      description: "More than 20% of high importance telemetry items indicate issues"
```

## Integration with Dashboarding Systems

### Grafana Dashboard Example

Create a JSON file called `ai-telemetry-dashboard.json`:

```json
{
  "title": "AI-Enhanced Telemetry Dashboard",
  "panels": [
    {
      "title": "Errors by Category",
      "type": "piechart",
      "datasource": "Loki",
      "targets": [
        {
          "expr": "sum by(ai_error_category) (count_over_time({job=\"application\"} | json | ai_error_category != \"\" [1h]))",
          "refId": "A"
        }
      ]
    },
    {
      "title": "Error Owner Distribution",
      "type": "barchart",
      "datasource": "Loki",
      "targets": [
        {
          "expr": "sum by(ai_error_owner) (count_over_time({job=\"application\"} | json | ai_error_owner != \"\" [1h]))",
          "refId": "A"
        }
      ]
    },
    {
      "title": "Services Detected in Telemetry",
      "type": "table",
      "datasource": "Loki",
      "targets": [
        {
          "expr": "sum by(ai_entity_services) (count_over_time({job=\"application\"} | json | ai_entity_services != \"\" [1h]))",
          "refId": "A"
        }
      ]
    }
  ]
}
```

## Integration with CI/CD Systems

### GitHub Actions Example

Create a file called `.github/workflows/otel-ai-processor.yml`:

```yaml
name: Deploy OpenTelemetry AI Processor

on:
  push:
    branches: [ main ]
  workflow_dispatch:

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2
      
      - name: Login to Docker Hub
        uses: docker/login-action@v2
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}
      
      - name: Build and push Docker image
        uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: fortxun/caza-otel-ai-processor:latest
      
      - name: Deploy to Kubernetes
        uses: actions-hub/kubectl@master
        env:
          KUBE_CONFIG: ${{ secrets.KUBE_CONFIG }}
        with:
          args: apply -f kubernetes/otel-ai-processor.yaml
```

## Integration with Service Mesh

### Istio Integration

Create a file called `kubernetes/otel-ai-processor.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: otel-collector
  template:
    metadata:
      labels:
        app: otel-collector
        sidecar.istio.io/inject: "false"  # Disable Istio sidecar
    spec:
      containers:
      - name: otel-collector
        image: fortxun/caza-otel-ai-processor:latest
        ports:
        - containerPort: 4317  # OTLP/gRPC
        - containerPort: 4318  # OTLP/HTTP
        volumeMounts:
        - name: config
          mountPath: /config
        - name: models
          mountPath: /models
        args:
        - --config=/config/config.yaml
      volumes:
      - name: config
        configMap:
          name: otel-collector-config
      - name: models
        persistentVolumeClaim:
          claimName: otel-models-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: otel-collector
  namespace: monitoring
spec:
  selector:
    app: otel-collector
  ports:
  - name: otlp-grpc
    port: 4317
    targetPort: 4317
  - name: otlp-http
    port: 4318
    targetPort: 4318
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: otel-collector-vs
  namespace: monitoring
spec:
  hosts:
  - "otel-collector.example.com"
  gateways:
  - monitoring-gateway
  http:
  - match:
    - uri:
        prefix: /v1/traces
    route:
    - destination:
        host: otel-collector
        port:
          number: 4318
```

## End-to-End Example: Microservices Monitoring

This example demonstrates a complete setup with multiple components:

### Docker Compose Setup

Create a file called `docker-compose-complete.yml`:

```yaml
version: '3'
services:
  # Sample application
  sample-app:
    image: otel/opentelemetry-demo:latest
    environment:
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
      - OTEL_SERVICE_NAME=demo-service
    depends_on:
      - otel-collector
  
  # OpenTelemetry Collector with AI Processor
  otel-collector:
    image: fortxun/caza-otel-ai-processor:latest
    volumes:
      - ./config:/config
      - ./models:/models
    ports:
      - "4317:4317"
      - "4318:4318"
      - "8889:8889"  # Prometheus exporter
    command: ["--config=/config/config.yaml"]
    depends_on:
      - jaeger
      - prometheus
      - elasticsearch
  
  # Jaeger for distributed tracing
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # UI
      - "14250:14250"  # gRPC collector
    environment:
      - COLLECTOR_OTLP_ENABLED=true
  
  # Prometheus for metrics
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./config/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
  
  # Elasticsearch for logs
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:7.16.2
    environment:
      - discovery.type=single-node
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
    ports:
      - "9200:9200"
  
  # Kibana for log visualization
  kibana:
    image: docker.elastic.co/kibana/kibana:7.16.2
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    ports:
      - "5601:5601"
    depends_on:
      - elasticsearch
  
  # Grafana for dashboards
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - ./config/grafana/provisioning:/etc/grafana/provisioning
      - ./config/grafana/dashboards:/var/lib/grafana/dashboards
    depends_on:
      - prometheus
      - elasticsearch
```

### Configuration for the End-to-End Example

Create a file called `config/config.yaml`:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  ai_processor:
    models:
      error_classifier:
        path: "/models/error-classifier.wasm"
        memory_limit_mb: 100
        timeout_ms: 50
      importance_sampler:
        path: "/models/importance-sampler.wasm"
        memory_limit_mb: 80
        timeout_ms: 30
      entity_extractor:
        path: "/models/entity-extractor.wasm"
        memory_limit_mb: 150
        timeout_ms: 50
    
    processing:
      batch_size: 50
      concurrency: 4
    
    features:
      error_classification: true
      smart_sampling: true
      entity_extraction: true
    
    sampling:
      error_events: 1.0
      slow_spans: 1.0
      normal_spans: 0.1
      threshold_ms: 500
    
    output:
      attribute_namespace: "ai."
      include_confidence_scores: true
  
  batch:
    timeout: 1s
    send_batch_size: 1000

exporters:
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true
  
  prometheus:
    endpoint: 0.0.0.0:8889
    namespace: otel
    send_timestamps: true
  
  elasticsearch:
    endpoints: ["http://elasticsearch:9200"]
    index: "otel-logs-%{yyyy.MM.dd}"

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [ai_processor, batch]
      exporters: [jaeger]
    
    metrics:
      receivers: [otlp]
      processors: [ai_processor, batch]
      exporters: [prometheus]
    
    logs:
      receivers: [otlp]
      processors: [ai_processor, batch]
      exporters: [elasticsearch]
```

### Running the End-to-End Example

1. Create all necessary configuration files
2. Start the system:
   ```bash
   docker-compose -f docker-compose-complete.yml up -d
   ```
3. Generate test telemetry:
   ```bash
   curl -X GET http://localhost:8080/checkout
   ```
4. View results:
   - Traces: http://localhost:16686 (Jaeger UI)
   - Metrics: http://localhost:9090 (Prometheus UI)
   - Logs: http://localhost:5601 (Kibana)
   - Dashboards: http://localhost:3000 (Grafana)