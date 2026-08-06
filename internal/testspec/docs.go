// Package testspec parses and interprets the JSON test-definition DSL (DDD
// context C1, the core domain): it validates specs, derives the reuse
// fingerprint from declared values, and runs pre-actions, steps, assertions,
// and post-actions as atomic steps with recorded provenance. It replaces the
// Go-function testkit as the way tests are expressed.
//
// Status: interface freeze only (T0.1). Parse/Fingerprint/interpreter bodies
// land in T1.1 and T3/T4.
package testspec
