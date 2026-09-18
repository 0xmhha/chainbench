package app

import (
	"github.com/0xmhha/chainbench/internal/dsl"
)

// Spec migration: reading a v1 test spec and writing it in the v2 grammar.

// SpecIsV2 reports whether a spec is already in the current grammar.
func SpecIsV2(raw []byte) bool { return dsl.IsV2(raw) }

// MigrateSpec converts a v1 spec to v2.
//
// The conversion is one reading of the old grammar. Two surfaces converting
// separately would be two answers to what a v1 spec meant, and the answer is
// written to disk.
func MigrateSpec(raw []byte) ([]byte, error) { return dsl.MigrateV1(raw) }
