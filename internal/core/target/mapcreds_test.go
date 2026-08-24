package target

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// TestMapCredentials_ResolvesTheDefaultPortBeforeMapping pins the order the
// docker translation depends on. A directly named host (user@host:path) often
// carries no port; the dial would default it to 22 AFTER the map ran, so a map
// keyed on 22 never matched and the harness dialed 127.0.0.1:22 — this
// machine's own sshd — instead of the container. Caught live by the gated
// remote keyring test; pinned here so it cannot come back without the fleet.
func TestMapCredentials_ResolvesTheDefaultPortBeforeMapping(t *testing.T) {
	m := func(host string, port int) (string, int) {
		if host == "172.30.0.11" && port == remote.DefaultSSHPort {
			return "127.0.0.1", 2201
		}
		return host, port
	}

	got := mapCredentials(remote.Credentials{User: "root", Host: "172.30.0.11"}, m)
	if got.Host != "127.0.0.1" || got.Port != 2201 {
		t.Fatalf("port-less credentials mapped to %s:%d, want 127.0.0.1:2201", got.Host, got.Port)
	}

	// An explicit port is respected as given.
	got = mapCredentials(remote.Credentials{Host: "172.30.0.11", Port: 10022}, m)
	if got.Host != "172.30.0.11" || got.Port != 10022 {
		t.Fatalf("unmapped explicit port changed: %s:%d", got.Host, got.Port)
	}

	// No map, no change — including the port default, which the dial owns.
	got = mapCredentials(remote.Credentials{Host: "10.0.0.9"}, nil)
	if got.Host != "10.0.0.9" || got.Port != 0 {
		t.Fatalf("nil map altered credentials: %s:%d", got.Host, got.Port)
	}
}
