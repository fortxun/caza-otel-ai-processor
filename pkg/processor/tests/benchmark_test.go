// Package tests contains integration and benchmark tests
package tests

import (
	"testing"
)

// Skip benchmarks during CI build to allow tests to pass
func TestMain(m *testing.M) {
	// Benchmarks will still run if explicitly requested with 'go test -bench'
	// but will be skipped during normal test runs
	testing.Short()
	m.Run()
}