// Package session owns the .chainbench/<session>/ artifact tree (DDD context
// C5): it is the single owner of all derived paths, of environment reuse by
// fingerprint, and of per-test records. Other layers write artifacts only
// through this package, never by building paths themselves.
//
// Status: interface freeze only (T0.1). Implementation lands in T1.3/T3.4.
package session
