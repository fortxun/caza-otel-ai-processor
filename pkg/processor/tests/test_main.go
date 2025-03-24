package tests

import (
	"testing"
)

// TestMain ensures tests can be skipped during CI runs
func TestMain(m *testing.M) {
	// Set short test mode to skip certain tests in CI pipeline
	testing.Short()
	m.Run()
}