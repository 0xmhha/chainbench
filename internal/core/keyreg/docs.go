// Package keyreg is the session key registry (DDD context C2): it owns every
// node-identity and signing key under .chainbench/<session>/keys/<name>/,
// sourced by random generation, local file, or remote download, and exposes
// them by name. BLS/PoP derivation is delegated to an injected BLSDeriver
// because chainbench has no native BLS crypto.
//
// Status: interface freeze only (T0.1). Implementation lands in T1.6.
package keyreg
