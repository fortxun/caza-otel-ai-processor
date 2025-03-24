package processor

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
)

func TestFactory_Type(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, "ai_processor", factory.Type().String())
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