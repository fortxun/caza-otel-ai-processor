FROM golang:1.23 AS builder

WORKDIR /build

# Install required dependencies
RUN apt-get update && apt-get install -y build-essential

# Copy only dependency files first to leverage Docker cache
COPY go.mod ./

# Force recreate go.sum and download all dependencies
RUN go mod download && go mod tidy

# Copy the rest of the code
COPY . .

# Run go mod tidy again with all the source code present
RUN go mod tidy

# Explicitly download all dependencies
RUN go get -v github.com/hashicorp/golang-lru/v2
RUN go get -v github.com/wasmerio/wasmer-go/wasmer
RUN go get -v go.uber.org/zap
RUN go get -v go.opentelemetry.io/collector/component
RUN go get -v go.opentelemetry.io/collector/consumer
RUN go get -v go.opentelemetry.io/collector/pdata
RUN go get -v go.opentelemetry.io/collector/processor
RUN go get -v go.opentelemetry.io/collector/processor/processorhelper
RUN go get -v go.opentelemetry.io/collector/confmap
RUN go get -v go.opentelemetry.io/collector/exporter
RUN go get -v go.opentelemetry.io/collector/exporter/otlpexporter
RUN go get -v go.opentelemetry.io/collector/otelcol
RUN go get -v go.opentelemetry.io/collector/receiver
RUN go get -v go.opentelemetry.io/collector/receiver/otlpreceiver

# Final dependency resolution
RUN go mod tidy

# Build the processor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o /bin/otel-ai-processor ./cmd/processor

# Final lightweight image
FROM alpine:3.17

# Install dependencies
RUN apk --no-cache add ca-certificates

# Create directory for models
RUN mkdir -p /models

# Copy the binary
COPY --from=builder /bin/otel-ai-processor /bin/otel-ai-processor
# Copy the models
COPY models/* /models/

# Set up a non-root user
RUN adduser -D -u 10001 otel
USER 10001

# Run the processor
ENTRYPOINT ["/bin/otel-ai-processor"]