package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TxWait is what every "did my transaction land" question goes through, and it
// had no coverage. Its failure mode is not a crash: a pending transaction that
// is mistaken for a mined one returns a ZERO receipt, whose Status is "" and so
// is not "0x1" -- so Succeeded() says false and the caller reports that the
// transaction failed, when the truth is that it had not happened yet.
//
// Two places stop that. The client maps a JSON null receipt to nil bytes, and
// TxWait treats nil as "keep waiting". Either could change alone, so the
// property is pinned here, where a caller meets it.

// receiptServer serves eth_getTransactionReceipt: null for the first pending
// calls, then body. It counts the calls it answered.
func receiptServer(t *testing.T, pending int, body string) (string, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "eth_getTransactionReceipt" {
			t.Errorf("unexpected method %q", req.Method)
		}
		n := atomic.AddInt32(&calls, 1)
		result := "null"
		if int(n) > pending {
			result = body
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%d,"result":%s}`, req.ID, result)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &calls
}

const minedReceipt = `{"status":"0x1","blockNumber":"0x2a","gasUsed":"0x5208"}`

func TestTxWait_RefusesWithoutItsInputs(t *testing.T) {
	for _, c := range []TxWaitIn{{Hash: "0xabc"}, {RPC: "http://x"}, {}} {
		if _, err := TxWait(context.Background(), Deps{}, c); err == nil ||
			!strings.Contains(err.Error(), "endpoint and a transaction hash are required") {
			t.Errorf("%+v should be refused by name: %v", c, err)
		}
	}
}

// TestTxWait_ReturnsTheReceiptOnceMined: and the receipt has to be the mined
// one, not a zero value that happens to unmarshal.
func TestTxWait_ReturnsTheReceiptOnceMined(t *testing.T) {
	url, _ := receiptServer(t, 0, minedReceipt)
	got, err := TxWait(context.Background(), Deps{}, TxWaitIn{RPC: url, Hash: "0xabc", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("TxWait: %v", err)
	}
	if got.Status != "0x1" || got.BlockNumber != "0x2a" || got.GasUsed != "0x5208" {
		t.Fatalf("receipt = %+v, want the mined one", got)
	}
	if !got.Succeeded() {
		t.Error("a status 0x1 receipt should report success")
	}
}

// TestTxWait_KeepsWaitingWhileTheReceiptIsNull is the property that matters. A
// null receipt means "not yet", and taking it for an answer would hand back a
// zero receipt whose Status is empty — which reads as a FAILED transaction
// rather than a pending one.
func TestTxWait_KeepsWaitingWhileTheReceiptIsNull(t *testing.T) {
	url, calls := receiptServer(t, 1, minedReceipt)
	got, err := TxWait(context.Background(), Deps{}, TxWaitIn{RPC: url, Hash: "0xabc", Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("TxWait: %v", err)
	}
	if got.Status != "0x1" {
		t.Fatalf("a null receipt was taken as the answer: got %+v", got)
	}
	if n := atomic.LoadInt32(calls); n < 2 {
		t.Errorf("polled %d time(s); a pending receipt must not end the wait", n)
	}
}

// TestTxWait_TimesOutSayingWhichTransaction: the wait that runs out must fail,
// and name the hash and the window. Returning a zero receipt with no error is
// the shape this rules out.
func TestTxWait_TimesOutSayingWhichTransaction(t *testing.T) {
	url, _ := receiptServer(t, 1<<30, minedReceipt) // never mined
	got, err := TxWait(context.Background(), Deps{}, TxWaitIn{RPC: url, Hash: "0xfeed", Timeout: time.Millisecond})
	if err == nil {
		t.Fatalf("a transaction that never landed returned %+v with no error", got)
	}
	for _, want := range []string{"timed out", "0xfeed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the timeout should say %q: %v", want, err)
		}
	}
	if got.Status != "" {
		t.Errorf("a failed wait returned a receipt: %+v", got)
	}
}

// TestTxWait_StopsWhenTheContextIsCancelled: a cancelled run must not hold the
// caller for the rest of the timeout.
func TestTxWait_StopsWhenTheContextIsCancelled(t *testing.T) {
	url, _ := receiptServer(t, 1<<30, minedReceipt)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	if _, err := TxWait(ctx, Deps{}, TxWaitIn{RPC: url, Hash: "0xabc", Timeout: time.Minute}); err == nil {
		t.Fatal("a cancelled wait returned no error")
	}
	if d := time.Since(start); d > 30*time.Second {
		t.Errorf("cancellation took %s; it should end the wait promptly", d)
	}
}

// TestTxWait_HonoursAShortTimeout: the wait must end inside the window it was
// given. The poll slept a fixed second regardless of the budget, so a 100ms
// timeout took about a second — which makes a timeout used as an assertion
// ("this must land within X") weaker than it reads.
func TestTxWait_HonoursAShortTimeout(t *testing.T) {
	url, _ := receiptServer(t, 1<<30, minedReceipt) // never mined
	start := time.Now()
	if _, err := TxWait(context.Background(), Deps{}, TxWaitIn{
		RPC: url, Hash: "0xabc", Timeout: 100 * time.Millisecond,
	}); err == nil {
		t.Fatal("a transaction that never lands must time out")
	}
	// Generous, but far below the second the fixed sleep cost.
	if d := time.Since(start); d > 600*time.Millisecond {
		t.Errorf("a 100ms wait took %s; the poll is not bounded by the budget", d)
	}
}
