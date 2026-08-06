// Package assert holds the type-aware comparison primitives the interpreter
// uses to check assertions (DDD context C1). Each returns (pass, detail) where
// detail explains a mismatch for provenance. Numeric comparisons are value-
// based (7, "7", "0x7", and wei hex strings compare equal), so RPC results and
// spec literals of different concrete types still match.
package assert
