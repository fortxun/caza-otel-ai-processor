package tests

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// MockWasmRuntime is a mock implementation of the WasmRuntime
type MockWasmRuntime struct {
	ClassifyErrorCalled   bool
	ClassifyErrorInput    map[string]interface{}
	ClassifyErrorOutput   map[string]interface{}
	ClassifyErrorError    error
	
	SampleTelemetryCalled   bool
	SampleTelemetryInput    map[string]interface{}
	SampleTelemetryOutput   map[string]interface{}
	SampleTelemetryError    error
	
	ExtractEntitiesCalled   bool
	ExtractEntitiesInput    map[string]interface{}
	ExtractEntitiesOutput   map[string]interface{}
	ExtractEntitiesError    error
}

// ClassifyError is a mock implementation
func (m *MockWasmRuntime) ClassifyError(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	m.ClassifyErrorCalled = true
	m.ClassifyErrorInput = input
	return m.ClassifyErrorOutput, m.ClassifyErrorError
}

// SampleTelemetry is a mock implementation
func (m *MockWasmRuntime) SampleTelemetry(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	m.SampleTelemetryCalled = true
	m.SampleTelemetryInput = input
	return m.SampleTelemetryOutput, m.SampleTelemetryError
}

// ExtractEntities is a mock implementation
func (m *MockWasmRuntime) ExtractEntities(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	m.ExtractEntitiesCalled = true
	m.ExtractEntitiesInput = input
	return m.ExtractEntitiesOutput, m.ExtractEntitiesError
}

// Close is a mock implementation
func (m *MockWasmRuntime) Close() error {
	return nil
}

// NewMockWasmRuntime creates a new MockWasmRuntime with default values
func NewMockWasmRuntime() *MockWasmRuntime {
	return &MockWasmRuntime{
		ClassifyErrorOutput: map[string]interface{}{
			"category":   "database_error",
			"owner":      "database-team",
			"severity":   "high",
			"confidence": 0.9,
		},
		SampleTelemetryOutput: map[string]interface{}{
			"importance": 0.8,
			"keep":       true,
		},
		ExtractEntitiesOutput: map[string]interface{}{
			"services":     []string{"user-service", "api-gateway"},
			"dependencies": []string{"postgres", "redis"},
			"confidence":   0.85,
		},
	}
}

// MockConsumer is a mock implementation for testing
type MockConsumer struct {
	// MockTraceConsumer implementation
	TracesCalled bool
	TracesInput  ptrace.Traces
	TracesError  error
	
	// MockMetricsConsumer implementation
	MetricsCalled bool
	MetricsInput  pmetric.Metrics
	MetricsError  error
	
	// MockLogConsumer implementation
	LogsCalled bool
	LogsInput  plog.Logs
	LogsError  error
}

// MockTracesConsumer is a mock implementation of the traces consumer
type MockTracesConsumer struct {
	TracesCalled   bool
	TracesInput    ptrace.Traces
	TracesError    error
	ConsumedTraces []ptrace.Traces
}

// ConsumeTraces is a mock implementation
func (m *MockTracesConsumer) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	m.TracesCalled = true
	m.TracesInput = traces
	m.ConsumedTraces = append(m.ConsumedTraces, traces)
	return m.TracesError
}

// Capabilities returns the consumer capabilities
func (m *MockTracesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// MockMetricsConsumer is a mock implementation of the metrics consumer
type MockMetricsConsumer struct {
	MetricsCalled    bool
	MetricsInput     pmetric.Metrics
	MetricsError     error
	ConsumedMetrics  []pmetric.Metrics
}

// ConsumeMetrics is a mock implementation
func (m *MockMetricsConsumer) ConsumeMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	m.MetricsCalled = true
	m.MetricsInput = metrics
	m.ConsumedMetrics = append(m.ConsumedMetrics, metrics)
	return m.MetricsError
}

// Capabilities returns the consumer capabilities
func (m *MockMetricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// MockLogsConsumer is a mock implementation of the logs consumer
type MockLogsConsumer struct {
	LogsCalled   bool
	LogsInput    plog.Logs
	LogsError    error
	ConsumedLogs []plog.Logs
}

// ConsumeLogs is a mock implementation
func (m *MockLogsConsumer) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	m.LogsCalled = true
	m.LogsInput = logs
	m.ConsumedLogs = append(m.ConsumedLogs, logs)
	return m.LogsError
}

// Capabilities returns the consumer capabilities
func (m *MockLogsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeTraces is a mock implementation
func (m *MockConsumer) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	m.TracesCalled = true
	m.TracesInput = traces
	return m.TracesError
}

// ConsumeMetrics is a mock implementation
func (m *MockConsumer) ConsumeMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	m.MetricsCalled = true
	m.MetricsInput = metrics
	return m.MetricsError
}

// ConsumeLogs is a mock implementation
func (m *MockConsumer) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	m.LogsCalled = true
	m.LogsInput = logs
	return m.LogsError
}

// Capabilities returns the consumer capabilities
func (m *MockConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// CreateTestTraces creates test trace data for testing
func CreateTestTraces(numTraces int, errorStatus bool) ptrace.Traces {
	traces := ptrace.NewTraces()
	
	rs := traces.ResourceSpans().AppendEmpty()
	resource := rs.Resource()
	resource.Attributes().PutStr("service.name", "test-service")
	resource.Attributes().PutStr("deployment.environment", "production")
	
	ss := rs.ScopeSpans().AppendEmpty()
	scope := ss.Scope()
	scope.SetName("test-scope")
	scope.SetVersion("v1.0.0")
	
	for i := 0; i < numTraces; i++ {
		span := ss.Spans().AppendEmpty()
		span.SetName("test-span")
		span.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
		span.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
		span.SetStartTimestamp(pcommon.Timestamp(1000000000))
		span.SetEndTimestamp(pcommon.Timestamp(2000000000))
		
		if errorStatus {
			span.Status().SetCode(ptrace.StatusCodeError)
			span.Status().SetMessage("Test error")
		} else {
			span.Status().SetCode(ptrace.StatusCodeOk)
		}
		
		// Add some attributes
		span.Attributes().PutStr("operation", "test")
		span.Attributes().PutBool("internal", true)
		span.Attributes().PutInt("retry_count", 3)
	}
	
	return traces
}

// CreateTestMetrics creates test metric data for testing
func CreateTestMetrics(numMetrics int) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	rm := metrics.ResourceMetrics().AppendEmpty()
	resource := rm.Resource()
	resource.Attributes().PutStr("service.name", "test-service")
	resource.Attributes().PutStr("deployment.environment", "production")
	
	sm := rm.ScopeMetrics().AppendEmpty()
	scope := sm.Scope()
	scope.SetName("test-scope")
	scope.SetVersion("v1.0.0")
	
	for i := 0; i < numMetrics; i++ {
		metric := sm.Metrics().AppendEmpty()
		metric.SetName("test-metric")
		metric.SetDescription("A test metric")
		
		gauge := metric.SetEmptyGauge()
		dp := gauge.DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.Timestamp(1000000000))
		dp.SetDoubleValue(42.0)
		
		// Add some attributes
		dp.Attributes().PutStr("operation", "test")
		dp.Attributes().PutBool("internal", true)
		dp.Attributes().PutInt("instance_id", 3)
	}
	
	return metrics
}

// CreateTestLogs creates test log data for testing
func CreateTestLogs(numLogs int, errorSeverity bool) plog.Logs {
	logs := plog.NewLogs()
	
	rl := logs.ResourceLogs().AppendEmpty()
	resource := rl.Resource()
	resource.Attributes().PutStr("service.name", "test-service")
	resource.Attributes().PutStr("deployment.environment", "production")
	
	sl := rl.ScopeLogs().AppendEmpty()
	scope := sl.Scope()
	scope.SetName("test-scope")
	scope.SetVersion("v1.0.0")
	
	for i := 0; i < numLogs; i++ {
		log := sl.LogRecords().AppendEmpty()
		log.SetTimestamp(pcommon.Timestamp(1000000000))
		log.Body().SetStr("This is a test log message")
		
		if errorSeverity {
			log.SetSeverityNumber(plog.SeverityNumberError)
			log.SetSeverityText("ERROR")
		} else {
			log.SetSeverityNumber(plog.SeverityNumberInfo)
			log.SetSeverityText("INFO")
		}
		
		// Add some attributes
		log.Attributes().PutStr("operation", "test")
		log.Attributes().PutBool("internal", true)
		log.Attributes().PutInt("instance_id", 3)
	}
	
	return logs
}

// TestData is a utility struct for creating test data
type TestData struct{}

// CreateTestTraces creates test traces with resource and span attributes
func (td *TestData) CreateTestTraces(resourceAttrs map[string]interface{}, spanAttrs map[string]interface{}, statusCode ptrace.StatusCode) ptrace.Traces {
	traces := ptrace.NewTraces()
	
	rs := traces.ResourceSpans().AppendEmpty()
	resource := rs.Resource()
	
	// Set resource attributes if provided
	if resourceAttrs != nil {
		for k, v := range resourceAttrs {
			switch val := v.(type) {
			case string:
				resource.Attributes().PutStr(k, val)
			case bool:
				resource.Attributes().PutBool(k, val)
			case int:
				resource.Attributes().PutInt(k, int64(val))
			case float64:
				resource.Attributes().PutDouble(k, val)
			}
		}
	} else {
		// Default attributes
		resource.Attributes().PutStr("service.name", "test-service")
		resource.Attributes().PutStr("deployment.environment", "production")
	}
	
	ss := rs.ScopeSpans().AppendEmpty()
	scope := ss.Scope()
	scope.SetName("test-scope")
	scope.SetVersion("v1.0.0")
	
	span := ss.Spans().AppendEmpty()
	span.SetName("test-operation")
	span.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	span.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	
	startTime := pcommon.Timestamp(1000000000)
	span.SetStartTimestamp(startTime)
	span.SetEndTimestamp(startTime + 500000000) // 500ms duration
	
	span.Status().SetCode(statusCode)
	if statusCode == ptrace.StatusCodeError {
		span.Status().SetMessage("Test error message")
	}
	
	// Set span attributes if provided
	if spanAttrs != nil {
		for k, v := range spanAttrs {
			switch val := v.(type) {
			case string:
				span.Attributes().PutStr(k, val)
			case bool:
				span.Attributes().PutBool(k, val)
			case int:
				span.Attributes().PutInt(k, int64(val))
			case float64:
				span.Attributes().PutDouble(k, val)
			}
		}
	} else {
		// Default attributes
		span.Attributes().PutStr("operation.type", "test")
		span.Attributes().PutBool("internal", true)
	}
	
	return traces
}

// CreateTestLogs creates test logs with resource attributes and a log message
func (td *TestData) CreateTestLogs(resourceAttrs map[string]interface{}, severityNumber plog.SeverityNumber, message string) plog.Logs {
	logs := plog.NewLogs()
	
	rl := logs.ResourceLogs().AppendEmpty()
	resource := rl.Resource()
	
	// Set resource attributes if provided
	if resourceAttrs != nil {
		for k, v := range resourceAttrs {
			switch val := v.(type) {
			case string:
				resource.Attributes().PutStr(k, val)
			case bool:
				resource.Attributes().PutBool(k, val)
			case int:
				resource.Attributes().PutInt(k, int64(val))
			case float64:
				resource.Attributes().PutDouble(k, val)
			}
		}
	} else {
		// Default attributes
		resource.Attributes().PutStr("service.name", "test-service")
		resource.Attributes().PutStr("deployment.environment", "production")
	}
	
	sl := rl.ScopeLogs().AppendEmpty()
	scope := sl.Scope()
	scope.SetName("test-scope")
	scope.SetVersion("v1.0.0")
	
	log := sl.LogRecords().AppendEmpty()
	log.SetTimestamp(pcommon.Timestamp(1000000000))
	
	if message != "" {
		log.Body().SetStr(message)
	} else {
		log.Body().SetStr("This is a test log message")
	}
	
	log.SetSeverityNumber(severityNumber)
	if severityNumber == plog.SeverityNumberError {
		log.SetSeverityText("ERROR")
	} else if severityNumber == plog.SeverityNumberWarn {
		log.SetSeverityText("WARN")
	} else {
		log.SetSeverityText("INFO")
	}
	
	// Add default attributes
	log.Attributes().PutStr("component", "test")
	log.Attributes().PutBool("internal", true)
	log.Attributes().PutInt("attempt", 1)
	
	return logs
}

// CreateTestMetrics creates test metrics with resource attributes and a metric value
func (td *TestData) CreateTestMetrics(resourceAttrs map[string]interface{}, metricName string, value float64) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	
	rm := metrics.ResourceMetrics().AppendEmpty()
	resource := rm.Resource()
	
	// Set resource attributes if provided
	if resourceAttrs != nil {
		for k, v := range resourceAttrs {
			switch val := v.(type) {
			case string:
				resource.Attributes().PutStr(k, val)
			case bool:
				resource.Attributes().PutBool(k, val)
			case int:
				resource.Attributes().PutInt(k, int64(val))
			case float64:
				resource.Attributes().PutDouble(k, val)
			}
		}
	} else {
		// Default attributes
		resource.Attributes().PutStr("service.name", "test-service")
		resource.Attributes().PutStr("deployment.environment", "production")
	}
	
	sm := rm.ScopeMetrics().AppendEmpty()
	scope := sm.Scope()
	scope.SetName("test-scope")
	scope.SetVersion("v1.0.0")
	
	metric := sm.Metrics().AppendEmpty()
	if metricName != "" {
		metric.SetName(metricName)
	} else {
		metric.SetName("test.metric")
	}
	metric.SetDescription("Test metric")
	
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.Timestamp(1000000000))
	dp.SetDoubleValue(value)
	
	// Add default attributes
	dp.Attributes().PutStr("operation", "test")
	dp.Attributes().PutBool("internal", true)
	
	return metrics
}