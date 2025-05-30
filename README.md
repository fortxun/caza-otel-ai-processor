# Intelligent Database Observability Processor (IDOP)

IDOP is an automated monitoring system that enhances Percona PMM deployments with intelligent anomaly detection and root cause analysis for database environments.

## Overview

IDOP solves the critical problem of reactive database monitoring by transforming raw telemetry data into actionable insights using statistical analysis and SRE best practices.

## Features

- PMM Integration: Connect to Percona PMM to collect database metrics
- Anomaly Detection: Identify abnormal patterns in database metrics
- Root Cause Analysis: Determine the likely causes of detected anomalies
- Reporting: Generate human-readable reports with actionable recommendations

## Architecture

IDOP follows an OpenTelemetry-inspired architecture with receiver-processor-exporter components:

1. **PMM Data Receiver**: Interfaces with Percona PMM via PromQL API
2. **Core Processing Pipeline**: Processes metrics and detects anomalies
3. **Report Generation Exporter**: Produces human-readable incident reports

## Getting Started

### Prerequisites

- Go 1.18 or higher
- Access to a Percona PMM instance
- Docker (for containerized deployment)

### Installation

```bash
# Clone the repository
git clone https://github.com/fortxun/idop.git
cd idop

# Build the project
go build -o bin/idop cmd/idop-server/main.go

# Run the server
./bin/idop
```

### Configuration

IDOP uses a YAML configuration file located at `config/config.yaml`. See the example configuration for details.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

