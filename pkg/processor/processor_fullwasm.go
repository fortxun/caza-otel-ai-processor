//go:build fullwasm
// +build fullwasm

// This file contains common definitions for fullwasm processor implementation

package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Common interfaces used by fullwasm implementation

// tracesProcessor processes trace data
type tracesProcessor interface {
	processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error)
	shutdown(ctx context.Context) error
}

// metricsProcessor processes metric data
type metricsProcessor interface {
	processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error)
	shutdown(ctx context.Context) error
}

// logsProcessor processes log data
type logsProcessor interface {
	processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error)
	shutdown(ctx context.Context) error
}

// Common wrappers for fullwasm processor implementations

// tracesProcessorWrapper implements processor.Traces
type tracesProcessorWrapper struct {
	processor tracesProcessor
	next      consumer.Traces
}

func (pw *tracesProcessorWrapper) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	processed, err := pw.processor.processTraces(ctx, td)
	if err != nil {
		return err
	}
	return pw.next.ConsumeTraces(ctx, processed)
}

func (pw *tracesProcessorWrapper) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (pw *tracesProcessorWrapper) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (pw *tracesProcessorWrapper) Shutdown(ctx context.Context) error {
	return pw.processor.shutdown(ctx)
}

// metricsProcessorWrapper implements processor.Metrics
type metricsProcessorWrapper struct {
	processor metricsProcessor
	next      consumer.Metrics
}

func (pw *metricsProcessorWrapper) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	processed, err := pw.processor.processMetrics(ctx, md)
	if err != nil {
		return err
	}
	return pw.next.ConsumeMetrics(ctx, processed)
}

func (pw *metricsProcessorWrapper) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (pw *metricsProcessorWrapper) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (pw *metricsProcessorWrapper) Shutdown(ctx context.Context) error {
	return pw.processor.shutdown(ctx)
}

// logsProcessorWrapper implements processor.Logs
type logsProcessorWrapper struct {
	processor logsProcessor
	next      consumer.Logs
}

func (pw *logsProcessorWrapper) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	processed, err := pw.processor.processLogs(ctx, ld)
	if err != nil {
		return err
	}
	return pw.next.ConsumeLogs(ctx, processed)
}

func (pw *logsProcessorWrapper) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (pw *logsProcessorWrapper) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (pw *logsProcessorWrapper) Shutdown(ctx context.Context) error {
	return pw.processor.shutdown(ctx)
}