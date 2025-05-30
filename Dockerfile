FROM golang:1.18-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum* ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/idop cmd/idop-server/main.go

# Create a minimal production image
FROM alpine:latest

# Add CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/bin/idop /usr/local/bin/idop

# Copy the configuration
COPY --from=builder /app/config /etc/idop/config

# Set the working directory
WORKDIR /etc/idop

# Set the entrypoint
ENTRYPOINT ["idop", "--config", "/etc/idop/config/config.yaml"]
