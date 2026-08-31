package testhelper

import "github.com/0xmhha/chainbench/internal/dsl/interp"

// testhelperRegistry is a registry with the built-ins on it — what
// testhelper.Registry() used to hand out before the vocabulary moved here.
func testhelperRegistry() interp.Registry {
	r := interp.NewRegistry()
	Register(r)
	return r
}
