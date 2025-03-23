package processor

import (
	"context"
	"testing"

	"github.com/fortxun/caza-otel-ai-processor/pkg/processor/tests"
	"github.com/fortxun/caza-otel-ai-processor/pkg/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

func TestFactory_Type(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, "ai_processor", factory.Type())
}

func TestFactory_CreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	
	assert.NotNil(t, cfg)
	assert.IsType(t, &Config{}, cfg)
	
	// Verify it's properly cast
	pCfg, ok := cfg.(*Config)
	assert.True(t, ok)
	
	// Check a few default values
	assert.True(t, pCfg.Features.ErrorClassification)
	assert.True(t, pCfg.Features.SmartSampling)
	assert.False(t, pCfg.Features.EntityExtraction)
	assert.False(t, pCfg.Features.ContextLinking)
}

func TestFactory_CreateTracesProcessor(t *testing.T) {
	// Create a factory
	factory := NewFactory()
	
	// Create a test logger
	logger, _ := zap.NewDevelopment()
	
	// Create processor settings
	settings := component.ProcessorCreateSettings{
		Logger: logger,
	}
	
	// Get the default config
	cfg := factory.CreateDefaultConfig()
	
	// Create a mock consumer
	consumer := &tests.MockTracesConsumer{}
	
	// Mock the runtime creation
	originalNewWasmRuntime := runtime.NewWasmRuntime
	defer func() { runtime.NewWasmRuntime = originalNewWasmRuntime }()
	
	runtime.NewWasmRuntime = func(logger *zap.Logger, config *runtime.WasmRuntimeConfig) (*runtime.WasmRuntime, error) {
		return &runtime.WasmRuntime{}, nil
	}
	
	// Create the processor
	processor, err := factory.CreateTracesProcessor(context.Background(), settings, cfg, consumer)
	
	// Verify
	require.NoError(t, err)
	assert.NotNil(t, processor)
}

func TestFactory_CreateMetricsProcessor(t *testing.T) {
	// Create a factory
	factory := NewFactory()
	
	// Create a test logger
	logger, _ := zap.NewDevelopment()
	
	// Create processor settings
	settings := component.ProcessorCreateSettings{
		Logger: logger,
	}
	
	// Get the default config
	cfg := factory.CreateDefaultConfig()
	
	// Create a mock consumer
	consumer := &tests.MockMetricsConsumer{}
	
	// Mock the runtime creation
	originalNewWasmRuntime := runtime.NewWasmRuntime
	defer func() { runtime.NewWasmRuntime = originalNewWasmRuntime }()
	
	runtime.NewWasmRuntime = func(logger *zap.Logger, config *runtime.WasmRuntimeConfig) (*runtime.WasmRuntime, error) {
		return &runtime.WasmRuntime{}, nil
	}
	
	// Create the processor
	processor, err := factory.CreateMetricsProcessor(context.Background(), settings, cfg, consumer)
	
	// Verify
	require.NoError(t, err)
	assert.NotNil(t, processor)
}

func TestFactory_CreateLogsProcessor(t *testing.T) {
	// Create a factory
	factory := NewFactory()
	
	// Create a test logger
	logger, _ := zap.NewDevelopment()
	
	// Create processor settings
	settings := component.ProcessorCreateSettings{
		Logger: logger,
	}
	
	// Get the default config
	cfg := factory.CreateDefaultConfig()
	
	// Create a mock consumer
	consumer := &tests.MockLogsConsumer{}
	
	// Mock the runtime creation
	originalNewWasmRuntime := runtime.NewWasmRuntime
	defer func() { runtime.NewWasmRuntime = originalNewWasmRuntime }()
	
	runtime.NewWasmRuntime = func(logger *zap.Logger, config *runtime.WasmRuntimeConfig) (*runtime.WasmRuntime, error) {
		return &runtime.WasmRuntime{}, nil
	}
	
	// Create the processor
	processor, err := factory.CreateLogsProcessor(context.Background(), settings, cfg, consumer)
	
	// Verify
	require.NoError(t, err)
	assert.NotNil(t, processor)
}

func TestFactory_InvalidConfig(t *testing.T) {
	// Create a factory
	factory := NewFactory()
	
	// Create a test logger
	logger, _ := zap.NewDevelopment()
	
	// Create processor settings
	settings := component.ProcessorCreateSettings{
		Logger: logger,
	}
	
	// Create an invalid config (wrong type)
	invalidCfg := "invalid config"
	
	// Create a mock consumer
	consumer := &tests.MockTracesConsumer{}
	
	// Try to create the processor
	processor, err := factory.CreateTracesProcessor(context.Background(), settings, invalidCfg, consumer)
	
	// Verify
	assert.Error(t, err)
	assert.Nil(t, processor)
}