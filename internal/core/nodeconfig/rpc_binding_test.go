package nodeconfig

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Both endpoints must retain the configured binding when --http/--ws reset
// binary defaults, including a handoff relaunch without its TOML file.
func TestArgvRPCBindingMatchesConfiguration(t *testing.T) {
	for _, family := range []struct{ name, dialect string }{
		{"stablenet", "geth114"}, {"wbft", "geth114"}, {"wemix", "geth110-wemix"},
	} {
		for _, config := range []string{"/owned/node.toml", ""} {
			for _, host := range []string{"", "0.0.0.0", "127.0.0.1"} {
				t.Run(fmt.Sprintf("%s/config=%t/host=%s", family.name, config != "", host), func(t *testing.T) {
					spec := Spec{Chain: Chain{Dialect: family.dialect}, Role: node.RoleEN, DataDir: "/owned/node", ConfigPath: config, HTTPHost: host, Ports: node.Endpoints{P2P: 30301, HTTP: 8601, WS: 8701}}
					want := host
					if want == "" {
						want = "0.0.0.0"
					}
					text := string(TOML(spec))
					for _, field := range []string{"HTTPHost", "WSHost"} {
						if !strings.Contains(text, fmt.Sprintf("%s = %q", field, want)) {
							t.Fatalf("configuration lost %s binding %q", field, want)
						}
					}
					args, err := Argv(spec)
					if err != nil {
						t.Fatal(err)
					}
					for _, flag := range []string{"--http.addr", "--ws.addr"} {
						if got := rpcBindingValue(t, args, flag); got != want {
							t.Errorf("%s=%q, want configured %q", flag, got, want)
						}
					}
				})
			}
		}
	}
}

// The published UI knob must affect both transports through the real consumer.
func TestConfigHTTPHostChoiceReachesBothLaunchEndpoints(t *testing.T) {
	for _, host := range []string{"0.0.0.0", "127.0.0.1"} {
		t.Run(host, func(t *testing.T) {
			spec := Spec{Chain: Chain{Dialect: "geth114"}, DataDir: "/owned/node", Ports: node.Endpoints{P2P: 30301, HTTP: 8601, WS: 8701}}
			if err := ApplyConfigOverride(&spec, "httpHost", host); err != nil {
				t.Fatal(err)
			}
			args, err := Argv(spec)
			if err != nil {
				t.Fatal(err)
			}
			for _, flag := range []string{"--http.addr", "--ws.addr"} {
				if got := rpcBindingValue(t, args, flag); got != host {
					t.Errorf("published choice %q produced %s=%q", host, flag, got)
				}
			}
		})
	}
}

// Operator overrides remain independent and replace a default exactly once.
func TestRPCBindingCommandOverridesKeepPrecedence(t *testing.T) {
	spec := Spec{Chain: Chain{Dialect: "geth114"}, DataDir: "/owned/node", HTTPHost: "0.0.0.0", Ports: node.Endpoints{P2P: 30301, HTTP: 8601, WS: 8701}}
	args, err := Argv(spec, Override{Key: KeyHTTPAddr, Value: "127.0.0.1"}, Override{Key: KeyWSAddr, Value: "127.0.0.2"})
	if err != nil {
		t.Fatal(err)
	}
	for flag, want := range map[string]string{"--http.addr": "127.0.0.1", "--ws.addr": "127.0.0.2"} {
		if got := rpcBindingValue(t, args, flag); got != want {
			t.Errorf("operator override lost: %s=%q, want %q", flag, got, want)
		}
	}
}

// Require one valued flag so duplicate or missing bindings cannot look correct.
func rpcBindingValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	index := slices.Index(args, flag)
	if index < 0 || index+1 >= len(args) {
		t.Errorf("missing explicit %s in %v", flag, args)
		return ""
	}
	if slices.Contains(args[index+1:], flag) {
		t.Errorf("duplicate %s in %v", flag, args)
	}
	return args[index+1]
}
