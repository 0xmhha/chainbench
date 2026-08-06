package accounts_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

// key 0x..01 -> address, from accounts conformance vectors (core.json).
const (
	testPrivKeyHex = "0000000000000000000000000000000000000000000000000000000000000001"
	testAddress    = "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf"
)

func mustKey(t *testing.T, h string) []byte {
	t.Helper()
	b, err := hex.DecodeString(h)
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	return b
}

// TestAddressForKey_Offline pins that the provider derives the golden address
// offline (no RPC) for every chain.
func TestAddressForKey_Offline(t *testing.T) {
	key := mustKey(t, testPrivKeyHex)
	for _, chain := range []string{"stablenet", "wbft", "wemix"} {
		ap, err := accounts.ForChain(chain)
		if err != nil {
			t.Fatalf("ForChain(%q): %v", chain, err)
		}
		got, err := ap.AddressForKey(key)
		if err != nil {
			t.Fatalf("%s AddressForKey: %v", chain, err)
		}
		if !strings.EqualFold(got, testAddress) {
			t.Errorf("%s: got %s, want %s", chain, got, testAddress)
		}
	}
}

// rpcMock is a minimal JSON-RPC server answering the calls wallet.SendCoin
// makes, so the faucet path can be exercised deterministically without a node.
type rpcMock struct {
	mu        sync.Mutex
	rawTxSeen string // hex of the last eth_sendRawTransaction param
}

func (m *rpcMock) handler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     int           `json:"id"`
		Method string        `json:"method"`
		Params []interface{} `json:"params"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	result := func(v interface{}) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0", "id": req.ID, "result": v,
		})
	}

	switch req.Method {
	case "eth_chainId":
		result("0x205b") // 8283
	case "eth_getProof":
		// stablenet Extra flags read; empty extra => not blacklisted.
		result(map[string]interface{}{"extra": "0x0"})
	case "eth_getTransactionCount":
		result("0x0")
	case "eth_maxPriorityFeePerGas":
		result("0x3b9aca00") // 1 gwei
	case "eth_gasPrice":
		result("0x3b9aca00")
	case "eth_sendRawTransaction":
		m.mu.Lock()
		if len(req.Params) > 0 {
			if s, ok := req.Params[0].(string); ok {
				m.rawTxSeen = s
			}
		}
		m.mu.Unlock()
		result("0x" + strings.Repeat("ab", 32)) // fake tx hash
	default:
		result(nil)
	}
}

// TestFaucet_RPCMock exercises the faucet path (OpenWallet -> SendCoin) end to
// end against a mock JSON-RPC node, proving the accounts SDK wiring: a funded
// key sends value and a raw signed tx reaches eth_sendRawTransaction.
func TestFaucet_RPCMock(t *testing.T) {
	mock := &rpcMock{}
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	ap, err := accounts.ForChain("stablenet")
	if err != nil {
		t.Fatal(err)
	}
	key := mustKey(t, testPrivKeyHex)

	hash, err := ap.Faucet(context.Background(), key, testAddress, big.NewInt(1_000_000), srv.URL)
	if err != nil {
		t.Fatalf("Faucet: %v", err)
	}
	if !strings.HasPrefix(hash, "0x") || len(hash) != 66 {
		t.Errorf("tx hash: got %q", hash)
	}
	mock.mu.Lock()
	raw := mock.rawTxSeen
	mock.mu.Unlock()
	if !strings.HasPrefix(raw, "0x02") { // EIP-1559 dynamic fee envelope
		t.Errorf("expected 0x02 dynamic-fee raw tx submitted, got %q", raw)
	}
}

// TestOpenWallet_BadKey surfaces key errors from the boundary.
func TestOpenWallet_BadKey(t *testing.T) {
	ap, _ := accounts.ForChain("wbft")
	if _, err := ap.AddressForKey([]byte{0x01, 0x02}); err == nil {
		t.Error("expected error for short key")
	}
}
