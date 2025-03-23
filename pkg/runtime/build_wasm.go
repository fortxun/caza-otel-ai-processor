//go:build fullwasm
// +build fullwasm

// This file contains build tags to enable the real WASM runtime implementation

package runtime

// This empty file ensures the real wasm_runtime.go is only used when 
// building with the fullwasm tag