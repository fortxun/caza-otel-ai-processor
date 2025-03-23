// This file contains the implementation of parallel processing

package processor

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Worker pool for parallel processing of telemetry items
type workerPool struct {
	numWorkers int
	taskChan   chan task
	wg         sync.WaitGroup
}

// Task to be executed by a worker
type task struct {
	ctx      context.Context
	fn       func(context.Context)
	callback func()
}

// Create a new worker pool
func newWorkerPool(numWorkers int) *workerPool {
	pool := &workerPool{
		numWorkers: numWorkers,
		taskChan:   make(chan task, numWorkers*10), // Buffer tasks to avoid blocking
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		go pool.worker()
	}

	return pool
}

// Worker goroutine that processes tasks
func (p *workerPool) worker() {
	for task := range p.taskChan {
		// Execute the task
		task.fn(task.ctx)
		
		// Execute callback if provided
		if task.callback != nil {
			task.callback()
		}
		
		// Mark task as done
		p.wg.Done()
	}
}

// Submit a task to the worker pool
func (p *workerPool) submit(ctx context.Context, fn func(context.Context), callback func()) {
	p.wg.Add(1)
	p.taskChan <- task{ctx, fn, callback}
}

// Wait for all tasks to complete
func (p *workerPool) wait() {
	p.wg.Wait()
}

// Close the worker pool
func (p *workerPool) close() {
	close(p.taskChan)
}

// Process spans in parallel
func processSpansInParallel(
	ctx context.Context,
	pool *workerPool,
	spans ptrace.SpanSlice,
	resource pcommon.Resource,
	processor func(context.Context, ptrace.Span, pcommon.Resource),
) {
	// Submit each span for processing
	for i := 0; i < spans.Len(); i++ {
		span := spans.At(i)
		
		pool.submit(ctx, func(ctx context.Context) {
			processor(ctx, span, resource)
		}, nil)
	}
}

// Process logs in parallel
func processLogsInParallel(
	ctx context.Context,
	pool *workerPool,
	logs plog.LogRecordSlice,
	resource pcommon.Resource,
	processor func(context.Context, plog.LogRecord, pcommon.Resource),
) {
	// Submit each log for processing
	for i := 0; i < logs.Len(); i++ {
		log := logs.At(i)
		
		pool.submit(ctx, func(ctx context.Context) {
			processor(ctx, log, resource)
		}, nil)
	}
}

// Process metrics in parallel
func processMetricsInParallel(
	ctx context.Context,
	pool *workerPool,
	metrics pmetric.MetricSlice,
	resource pcommon.Resource,
	processor func(context.Context, pmetric.Metric, pcommon.Resource),
) {
	// Submit each metric for processing
	for i := 0; i < metrics.Len(); i++ {
		metric := metrics.At(i)
		
		pool.submit(ctx, func(ctx context.Context) {
			processor(ctx, metric, resource)
		}, nil)
	}
}