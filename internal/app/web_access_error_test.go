package app

import (
	"errors"
	"testing"
)

// A plan names every SSH server the user has not bound a credential to,
// instead of failing later at the first dial.
func TestPlanNamesTheSSHAccessTheUserLacks(t *testing.T) {
	set := spreadSet(t, `{"name":"local","addr":"127.0.0.1"},{"name":"a","addr":"10.0.0.1"},{"name":"b","addr":"10.0.0.2"}`)
	servers, err := webPlanServers(set, webChainArguments{AllServers: true}, "chain.deploy")
	if err != nil {
		t.Fatal(err)
	}
	missing := webMissingAccess(servers, map[string]string{"a": "credential"})
	if len(missing) != 1 || missing[0] != "ssh:b" {
		t.Fatalf("missing = %v", missing)
	}
	if got := webMissingAccess(servers, map[string]string{"a": "c1", "b": "c2"}); len(got) != 0 {
		t.Fatalf("bound servers reported missing: %v", got)
	}
	err = &WebAccessError{Missing: missing}
	if errors.Is(err, ErrDeploymentForbidden) || err.Error() == "" {
		t.Fatalf("%v", err)
	}
}
