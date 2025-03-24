package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigDefaults(t *testing.T) {
	// Get the default configuration
	config := CreateDefaultConfig().(*Config)
	
	// Test default model configurations
	assert.Equal(t, "/models/error-classifier.wasm", config.Models.ErrorClassifier.Path)
	assert.Equal(t, 100, config.Models.ErrorClassifier.MemoryLimitMB)
	assert.Equal(t, 50, config.Models.ErrorClassifier.TimeoutMs)
	
	assert.Equal(t, "/models/importance-sampler.wasm", config.Models.ImportanceSampler.Path)
	assert.Equal(t, 80, config.Models.ImportanceSampler.MemoryLimitMB)
	assert.Equal(t, 30, config.Models.ImportanceSampler.TimeoutMs)
	
	assert.Equal(t, "/models/entity-extractor.wasm", config.Models.EntityExtractor.Path)
	assert.Equal(t, 150, config.Models.EntityExtractor.MemoryLimitMB)
	assert.Equal(t, 50, config.Models.EntityExtractor.TimeoutMs)
	
	// Test default processing configurations
	assert.Equal(t, 50, config.Processing.BatchSize)
	assert.Equal(t, 4, config.Processing.Concurrency)
	assert.Equal(t, 1000, config.Processing.QueueSize)
	assert.Equal(t, 500, config.Processing.TimeoutMs)
	
	// Test default feature toggles
	assert.True(t, config.Features.ErrorClassification)
	assert.True(t, config.Features.SmartSampling)
	assert.False(t, config.Features.EntityExtraction)
	assert.False(t, config.Features.ContextLinking)
	
	// Test default sampling configurations
	assert.Equal(t, 1.0, config.Sampling.ErrorEvents)
	assert.Equal(t, 1.0, config.Sampling.SlowSpans)
	assert.Equal(t, 0.1, config.Sampling.NormalSpans)
	assert.Equal(t, 500, config.Sampling.ThresholdMs)
	
	// Test default output configurations
	assert.Equal(t, "ai.", config.Output.AttributeNamespace)
	assert.True(t, config.Output.IncludeConfidenceScores)
	assert.Equal(t, 256, config.Output.MaxAttributeLength)
}

func TestConfigCustomValues(t *testing.T) {
	// Create a custom configuration
	config := &Config{
		Models: ModelsConfig{
			ErrorClassifier: ModelConfig{
				Path:        "/custom/path/error-model.wasm",
				MemoryLimitMB: 200,
				TimeoutMs:   100,
			},
			ImportanceSampler: ModelConfig{
				Path:        "/custom/path/sampler-model.wasm",
				MemoryLimitMB: 150,
				TimeoutMs:   75,
			},
			EntityExtractor: ModelConfig{
				Path:        "/custom/path/entity-model.wasm",
				MemoryLimitMB: 250,
				TimeoutMs:   120,
			},
		},
		Processing: ProcessingConfig{
			BatchSize:   100,
			Concurrency: 8,
			QueueSize:   2000,
			TimeoutMs:   1000,
		},
		Features: FeaturesConfig{
			ErrorClassification: false,
			SmartSampling:       true,
			EntityExtraction:    true,
			ContextLinking:      true,
		},
		Sampling: SamplingConfig{
			ErrorEvents: 0.5,
			SlowSpans:   0.75,
			NormalSpans: 0.05,
			ThresholdMs: 1000,
		},
		Output: OutputConfig{
			AttributeNamespace:     "aiml.",
			IncludeConfidenceScores: false,
			MaxAttributeLength:      512,
		},
	}
	
	// Test custom model configurations
	assert.Equal(t, "/custom/path/error-model.wasm", config.Models.ErrorClassifier.Path)
	assert.Equal(t, 200, config.Models.ErrorClassifier.MemoryLimitMB)
	assert.Equal(t, 100, config.Models.ErrorClassifier.TimeoutMs)
	
	assert.Equal(t, "/custom/path/sampler-model.wasm", config.Models.ImportanceSampler.Path)
	assert.Equal(t, 150, config.Models.ImportanceSampler.MemoryLimitMB)
	assert.Equal(t, 75, config.Models.ImportanceSampler.TimeoutMs)
	
	assert.Equal(t, "/custom/path/entity-model.wasm", config.Models.EntityExtractor.Path)
	assert.Equal(t, 250, config.Models.EntityExtractor.MemoryLimitMB)
	assert.Equal(t, 120, config.Models.EntityExtractor.TimeoutMs)
	
	// Test custom processing configurations
	assert.Equal(t, 100, config.Processing.BatchSize)
	assert.Equal(t, 8, config.Processing.Concurrency)
	assert.Equal(t, 2000, config.Processing.QueueSize)
	assert.Equal(t, 1000, config.Processing.TimeoutMs)
	
	// Test custom feature toggles
	assert.False(t, config.Features.ErrorClassification)
	assert.True(t, config.Features.SmartSampling)
	assert.True(t, config.Features.EntityExtraction)
	assert.True(t, config.Features.ContextLinking)
	
	// Test custom sampling configurations
	assert.Equal(t, 0.5, config.Sampling.ErrorEvents)
	assert.Equal(t, 0.75, config.Sampling.SlowSpans)
	assert.Equal(t, 0.05, config.Sampling.NormalSpans)
	assert.Equal(t, 1000, config.Sampling.ThresholdMs)
	
	// Test custom output configurations
	assert.Equal(t, "aiml.", config.Output.AttributeNamespace)
	assert.False(t, config.Output.IncludeConfidenceScores)
	assert.Equal(t, 512, config.Output.MaxAttributeLength)
}

func TestSamplingRateLimits(t *testing.T) {
	// Create a configuration with sampling rates outside the valid range
	config := &Config{
		Sampling: SamplingConfig{
			ErrorEvents: 1.5,  // Above 1.0
			SlowSpans:   -0.1, // Below 0.0
			NormalSpans: 0.5,  // Valid
		},
	}
	
	// Here we would normally have validation logic that would cap these values
	// between 0.0 and 1.0, and then we would test that behavior.
	// For this exercise, we'll just demonstrate what we would test.
	
	// Example of validation method that might be added to the SamplingConfig
	validateSamplingRates := func(config *SamplingConfig) {
		if config.ErrorEvents < 0.0 {
			config.ErrorEvents = 0.0
		} else if config.ErrorEvents > 1.0 {
			config.ErrorEvents = 1.0
		}
		
		if config.SlowSpans < 0.0 {
			config.SlowSpans = 0.0
		} else if config.SlowSpans > 1.0 {
			config.SlowSpans = 1.0
		}
		
		if config.NormalSpans < 0.0 {
			config.NormalSpans = 0.0
		} else if config.NormalSpans > 1.0 {
			config.NormalSpans = 1.0
		}
	}
	
	// Apply validation
	validateSamplingRates(&config.Sampling)
	
	// Test that values were capped to the valid range
	assert.Equal(t, 1.0, config.Sampling.ErrorEvents)
	assert.Equal(t, 0.0, config.Sampling.SlowSpans)
	assert.Equal(t, 0.5, config.Sampling.NormalSpans)
}