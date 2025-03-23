//go:build !fullwasm
// +build !fullwasm

// This file contains build tags to disable the real WASM runtime and use the stub instead

package runtime

// This empty file exists to make sure the stub version is used for building
// when the fullwasm build tag is not provided