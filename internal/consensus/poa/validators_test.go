package poa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// govRPCServer answers the RPC calls RuntimeValidators makes: admin_wemixInfo
// for the governance address, then eth_call getMemberLength / getMember. Members
// are 1-based; member[0] is the zero address, as the real contract returns.
func govRPCServer(t *testing.T, gov string, members []string) *httptest.Server {
	t.Helper()
	word := func(hex40 string) string {
		return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(hex40, "0x")
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		reply := func(result any) {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
		}
		switch req.Method {
		case "admin_wemixInfo":
			reply(map[string]any{"governance": gov})
		case "eth_call":
			var p []json.RawMessage
			_ = json.Unmarshal(req.Params, &p)
			var call struct {
				To   string `json:"to"`
				Data string `json:"data"`
			}
			_ = json.Unmarshal(p[0], &call)
			switch {
			case call.Data == govGetMemberLength:
				reply(fmt.Sprintf("0x%064x", len(members)))
			case strings.HasPrefix(call.Data, govGetMember):
				// index is the last hex of the selector+arg
				idxHex := strings.TrimPrefix(call.Data, govGetMember)
				var idx int
				fmt.Sscanf(idxHex, "%064x", &idx)
				if idx >= 1 && idx <= len(members) {
					reply(word(members[idx-1]))
				} else {
					reply(word(zeroAddr40))
				}
			default:
				reply("0x")
			}
		default:
			reply(nil)
		}
	}))
}

func TestRuntimeValidators_ReadsGovernanceMembers(t *testing.T) {
	members := []string{
		"0xc17d493883eaa3b4cceb0f214b273392d562f9d8",
		"0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c",
		"0x8c4a10b9108d49b9d23f764464090831d9c17764",
	}
	srv := govRPCServer(t, "0x269330264fb2510dc374e08cf79fc08b805ecf0c", members)
	defer srv.Close()

	got, err := Family{}.RuntimeValidators(context.Background(), rpc.Dial(srv.URL))
	if err != nil {
		t.Fatalf("RuntimeValidators: %v", err)
	}
	if len(got) != len(members) {
		t.Fatalf("got %d members, want %d: %v", len(got), len(members), got)
	}
	for i := range members {
		if !strings.EqualFold(got[i], members[i]) {
			t.Fatalf("member %d = %s, want %s", i, got[i], members[i])
		}
	}
}

// TestParseMemberCount covers the member count as what it is — a value the node
// supplies, not a fact. Anything that cannot size an enumeration must come back
// as an error, never as a slice capacity.
func TestParseMemberCount(t *testing.T) {
	ok := []struct {
		raw  string
		want int
	}{
		{"0x0000000000000000000000000000000000000000000000000000000000000004", 4},
		{"0x4", 4},
		{"0x0", 0},
	}
	for _, tc := range ok {
		got, err := parseMemberCount(tc.raw)
		if err != nil || got != tc.want {
			t.Fatalf("parseMemberCount(%q) = %d, %v; want %d, nil", tc.raw, got, err, tc.want)
		}
	}

	bad := []struct {
		name string
		raw  string
	}{
		{"negative notation", "0x-1"},
		{"positive sign", "0x+1"},
		{"uint256 max truncates to -1 as int64", "0x" + strings.Repeat("f", 64)},
		{"beyond int64 but positive", "0x8000000000000000"},
		{"over the member cap", "0x10000"},
		{"longer than a word", "0x" + strings.Repeat("1", 65)},
		{"not hex", "0xzz"},
		{"empty", "0x"},
		{"blank", ""},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseMemberCount(tc.raw); err == nil {
				t.Fatalf("parseMemberCount(%q) must be refused", tc.raw)
			}
		})
	}
}

// TestRuntimeValidators_MalformedMemberCountIsAnError drives the whole reader
// with a node that answers a bad length: the command must report it, not panic.
func TestRuntimeValidators_MalformedMemberCountIsAnError(t *testing.T) {
	for _, lenReply := range []string{"0x-1", "0x" + strings.Repeat("f", 64), "0x10000"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				ID     any    `json:"id"`
				Method string `json:"method"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			result := any(nil)
			switch req.Method {
			case "admin_wemixInfo":
				result = map[string]any{"governance": "0x269330264fb2510dc374e08cf79fc08b805ecf0c"}
			case "eth_call":
				result = lenReply
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
		}))
		f := Family{}
		_, err := f.RuntimeValidators(context.Background(), rpc.Dial(srv.URL))
		srv.Close()
		if err == nil {
			t.Fatalf("member length %q must be refused", lenReply)
		}
	}
}

func TestRuntimeValidators_GovernanceNotDeployed(t *testing.T) {
	srv := govRPCServer(t, "0x"+zeroAddr40, nil)
	defer srv.Close()
	f := Family{}
	if _, err := f.RuntimeValidators(context.Background(), rpc.Dial(srv.URL)); err == nil {
		t.Fatal("a chain with no governance deployed must error, not report an empty set")
	}
}
