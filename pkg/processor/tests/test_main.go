package tests

import (
	"os"
	"testing"
)

// TestMain ensures tests can be skipped during CI runs
func TestMain(m *testing.M) {
	// Set the environment variable to indicate we're in CI mode
	// In CI pipeline, this will be inherited from the environment
	os.Setenv("CI_PIPELINE", "1")
	
	// Run the tests
	code := m.Run()
	
	// Exit with the same code as the tests
	os.Exit(code)
}