// Package session owns the .chainbench/<session>/ artifact tree (DDD context
// C5): it is the single owner of all derived paths, of environment reuse by
// fingerprint, and of per-test records. Other layers write artifacts only
// through this package, never by building paths themselves.
package session
