//go:build tools
// +build tools

// Package tools pins development-only tool dependencies so they are recorded
// in go.mod but not compiled into the chainbench-net binary.
package tools

import (
	_ "github.com/atombender/go-jsonschema/pkg/generator"
)
