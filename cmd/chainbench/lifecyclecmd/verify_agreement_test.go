package lifecyclecmd_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// verify prints whether the nodes agree on one chain beside whether the network
// is producing, because producing alone is true on every side of a split. The
// rendering has three outcomes -- agreed, not agreed, and not checked -- and the
// third is the one worth guarding: a comparison that could not be made must not
// read as agreement.
//
// These drive the command as an operator does, over mock JSON-RPC, because what
// is under test is the LINE an operator reads.

// chainNode answers the reads verify makes. hashAt decides which chain it claims
// to be on; two nodes with different tags are a split.
func chainNode(t *testing.T, height uint64, tag string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		reply := func(v string) { fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":%s}`, req.ID, v) }
		switch req.Method {
		case "eth_chainId":
			reply(`"0x2058"`)
		case "eth_blockNumber":
			reply(fmt.Sprintf(`"0x%x"`, height))
		case "net_peerCount":
			reply(`"0x3"`)
		case "eth_syncing":
			reply(`false`)
		case "eth_getBlockByNumber":
			n, _ := req.Params[0].(string)
			reply(fmt.Sprintf(`{"number":%q,"hash":"0x%s000000","miner":"0xabc"}`, n, tag))
		default:
			reply(`null`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func verifyOut(t *testing.T, urls ...string) string {
	t.Helper()
	args := []string{"verify", "--chain", "wbft", "--ready-timeout", "0", "--progress-delay", "1ms"}
	for _, u := range urls {
		args = append(args, "--rpc", u)
	}
	out, _ := run(t, args...)
	return out
}

// TestVerify_ReportsAgreementOnOneChain: the ordinary case names the block it
// compared, so a reader can tell the check happened.
func TestVerify_ReportsAgreementOnOneChain(t *testing.T) {
	out := verifyOut(t, chainNode(t, 20, "aaaa"), chainNode(t, 20, "aaaa"))
	if !strings.Contains(out, "agreement: yes") {
		t.Fatalf("two nodes on one chain should report agreement:\n%s", out)
	}
	if !strings.Contains(out, "block") {
		t.Errorf("the line should name the block compared:\n%s", out)
	}
}

// TestVerify_ReportsASplit is the case producing alone gets wrong. Both sides are
// up, in sync with themselves, and advancing.
func TestVerify_ReportsASplit(t *testing.T) {
	out := verifyOut(t, chainNode(t, 20, "aaaa"), chainNode(t, 20, "bbbb"))
	if !strings.Contains(out, "agreement: NO") {
		t.Fatalf("two nodes on different chains should not read as agreement:\n%s", out)
	}
	// The line has to say which nodes are where, or it cannot be acted on.
	if !strings.Contains(out, "different hashes") {
		t.Errorf("the line should say what differs:\n%s", out)
	}
}

// TestVerify_SaysWhenItCouldNotCheck: one node cannot agree with anyone, and the
// output must say the comparison did not happen rather than print agreement.
func TestVerify_SaysWhenItCouldNotCheck(t *testing.T) {
	out := verifyOut(t, chainNode(t, 20, "aaaa"))
	if !strings.Contains(out, "agreement: not checked") {
		t.Fatalf("a single node should report that agreement was not checked:\n%s", out)
	}
	if strings.Contains(out, "agreement: yes") {
		t.Fatal("an unchecked comparison printed as agreement")
	}
	if !strings.Contains(out, "at least two") {
		t.Errorf("the line should say why it could not check:\n%s", out)
	}
}
