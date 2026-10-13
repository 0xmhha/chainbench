package feature_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/feature"
)

func registered(t *testing.T, name string) feature.Registration {
	t.Helper()
	value, ok := feature.Lookup(name)
	if !ok {
		t.Fatalf("missing registration %s", name)
	}
	return value
}

func invokeJSON(t *testing.T, name string, args any) any {
	t.Helper()
	value := registered(t, name)
	input := value.Input()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, input); err != nil {
		t.Fatal(err)
	}
	out, err := value.Invoke(context.Background(), app.Deps{}, input)
	if err != nil {
		t.Fatalf("invoke %s: %v", name, err)
	}
	return out
}

func TestRegisteredOperationsRejectUnrelatedInputs(t *testing.T) {
	for _, value := range feature.Registered() {
		if strings.HasPrefix(value.Name, "probe.") {
			continue
		}
		t.Run(value.Name, func(t *testing.T) {
			if value.Summary == "" || value.Stage == "" {
				t.Fatal("missing operation metadata")
			}
			one, two := reflect.ValueOf(value.Input()), reflect.ValueOf(value.Input())
			if one.Kind() != reflect.Pointer || one.Type() != two.Type() {
				t.Fatal("input is not consistently typed")
			}
			if one.Elem().Type().Size() > 0 && one.Pointer() == two.Pointer() {
				t.Fatal("input aliases another caller's state")
			}
			if _, err := value.Invoke(context.Background(), app.Deps{}, "unrelated input"); err == nil {
				t.Fatal("untyped input reached an operation")
			}
		})
	}
}

func TestSecretAndArbitraryRPCOperationsAreNotQueries(t *testing.T) {
	for _, name := range []string{"key.resolve", "key.save", "key.generate", "key-set.generate", "keyring.export", "node.rpc", "network.node-call", "session.gc"} {
		if registered(t, name).ReadOnly {
			t.Errorf("%s was advertised as a public read-only query", name)
		}
	}
}

func TestRegisteredKeyringPublicReadsOmitExportedSecret(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "keys")
	newRing := registered(t, "keyring.new")
	created, err := newRing.Invoke(context.Background(), app.Deps{}, &app.RingCreateIn{Ring: app.RingRef{Dir: directory}, Count: 1, WithBLS: true})
	if err != nil {
		t.Fatal(err)
	}
	if entries := created.(app.RingOut).Entries; len(entries) != 1 || entries[0].PrivateKey != "" {
		t.Fatal("create result lost its identity or exposed a secret")
	}
	show := registered(t, "keyring.show")
	public, err := show.Invoke(context.Background(), app.Deps{}, &app.RingEntryIn{Ring: app.RingRef{Dir: directory}, Label: "node1"})
	if err != nil {
		t.Fatal(err)
	}
	export := registered(t, "keyring.export")
	secret, err := export.Invoke(context.Background(), app.Deps{}, &app.RingEntryIn{Ring: app.RingRef{Dir: directory}, Label: "node1"})
	if err != nil {
		t.Fatal(err)
	}
	pub, priv := public.(app.EntryOut), secret.(app.EntryOut)
	if pub.PrivateKey != "" || priv.PrivateKey == "" || pub.Address != priv.Address || pub.BLSPubKey == "" {
		t.Fatal("public and secret views do not preserve their separate contracts")
	}
}

func TestRegisteredNetworkHelpersRoundTripTheNamedEndpoints(t *testing.T) {
	directory := t.TempDir()
	set := app.NodeSet{Network: "registered-fixture", Chain: "stablenet", Nodes: []app.Node{{Index: 1, Role: app.RoleEN, RPCURL: "http://127.0.0.1:18545"}}}
	invokeJSON(t, "network.attach", map[string]any{"StateDir": directory, "Network": set})
	got := invokeJSON(t, "network.info", map[string]any{"StateDir": directory, "Name": set.Network}).(app.NodeSet)
	if got.Network != set.Network || len(got.Nodes) != 1 || got.Nodes[0].RPCURL != set.Nodes[0].RPCURL {
		t.Fatal("attached record did not retain its name and endpoints")
	}
	if len(invokeJSON(t, "network.list", map[string]any{"StateDir": directory}).([]app.NodeSet)) != 1 {
		t.Fatal("attached record is absent from the list")
	}
	invokeJSON(t, "network.detach", map[string]any{"StateDir": directory, "Name": set.Network})
	if len(invokeJSON(t, "network.list", map[string]any{"StateDir": directory}).([]app.NodeSet)) != 0 {
		t.Fatal("detached record is still present")
	}
}

func TestRegisteredRPCHelpersPreserveEndpointsAndReadings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		var result any
		switch request.Method {
		case "eth_blockNumber":
			result = "0x9"
		case "eth_chainId":
			result = "0x1"
		case "net_peerCount":
			result = "0x2"
		case "eth_syncing":
			result = false
		default:
			t.Errorf("unexpected method %s", request.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}))
	defer server.Close()
	if got := invokeJSON(t, "node.peers", map[string]any{"RPC": server.URL}).(uint64); got != 2 {
		t.Fatalf("peers = %d", got)
	}
	got := invokeJSON(t, "node.status", map[string]any{"RPC": server.URL}).(app.NodeReadings)
	if got.BlockNumber != 9 || got.ChainID != 1 || got.PeerCount != 2 || got.Syncing {
		t.Fatalf("wrong readings: %+v", got)
	}
}

func TestRegisteredCapabilityLookupPreservesNotFound(t *testing.T) {
	out := invokeJSON(t, "capability.lookup", map[string]any{"Name": "nonexistent.nonexistent.nonexistent"})
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var found struct {
		Found bool `json:"found"`
	}
	if err := json.Unmarshal(raw, &found); err != nil {
		t.Fatal(err)
	}
	if found.Found {
		t.Fatal("missing capability was represented as found")
	}
}
