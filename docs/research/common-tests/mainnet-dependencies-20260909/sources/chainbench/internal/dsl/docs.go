// Package dsl parses and interprets the JSON test-definition DSL (DDD
// context C1, the core domain): it validates specs, derives the reuse
// fingerprint from declared values, and runs pre-actions, steps, assertions,
// and post-actions as atomic steps with recorded provenance. It is the one way
// tests are expressed; the Go-function testkit it replaced was retired in R5.
package dsl
