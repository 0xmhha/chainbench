package testkit

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

// Client is the RPC surface a test case uses. *rpc.Client satisfies it; the
// runner may inject a fake for its own tests.
type Client interface {
	Call(ctx context.Context, method string, out any, params ...any) error
	BlockNumber(ctx context.Context) (uint64, error)
	ChainID(ctx context.Context) (uint64, error)
}

// ClientFactory builds a Client for an RPC URL.
type ClientFactory func(url string) Client

// DefaultFactory dials a real rpc.Client.
func DefaultFactory(url string) Client { return rpc.Dial(url) }

// failSentinel is panicked by Fatalf/Fatal and recovered by the runner.
type failSentinel struct{ msg string }

// skipSentinel is panicked by Skip and recovered by the runner, which records
// the case as skipped (not a pass or fail).
type skipSentinel struct{ msg string }

// T is the handle a Case uses to drive one NodeSet and assert on it. Assertions
// mirror the shape of the standard library's testing.T: Errorf records a
// failure and continues; Fatalf records and aborts the case.
type T struct {
	ctx       context.Context
	ns        node.NodeSet
	factory   ClientFactory
	fundedKey []byte
	failed    bool
	msgs      []string
}

// newT constructs a T (used by the runner).
func newT(ctx context.Context, ns node.NodeSet, f ClientFactory, fundedKey []byte) *T {
	if f == nil {
		f = DefaultFactory
	}
	return &T{ctx: ctx, ns: ns, factory: f, fundedKey: fundedKey}
}

// FundedKey returns the caller-supplied funded-account private key and whether
// one was configured (CHAINBENCH_FUNDED_KEY). Chain-agnostic write cases use it
// to act on an arbitrary chain — e.g. an external L2 supplied via --manifest —
// and Skip when it is absent. The key is never serialized: it flows only through
// the run, never through the NodeSet (which is written to nodeset.json).
func (t *T) FundedKey() ([]byte, bool) {
	return t.fundedKey, len(t.fundedKey) > 0
}

// Skip records the case as skipped with the given reason and aborts it. Use it
// when a case cannot apply to the current run for a reason not expressible as a
// static capability (e.g. no funded key configured).
func (t *T) Skip(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	panic(skipSentinel{msg: msg})
}

// Ctx returns the case context.
func (t *T) Ctx() context.Context { return t.ctx }

// NodeSet returns the network under test.
func (t *T) NodeSet() node.NodeSet { return t.ns }

// Primary returns a Client for the primary node.
func (t *T) Primary() Client {
	p, ok := t.ns.Primary()
	if !ok {
		t.Fatalf("no nodes in set")
	}
	return t.factory(p.RPCURL)
}

// Node returns a Client for the node with the given 1-based index.
func (t *T) Node(index int) Client {
	for _, n := range t.ns.Nodes {
		if n.Index == index {
			return t.factory(n.RPCURL)
		}
	}
	t.Fatalf("no node with index %d", index)
	return nil
}

// Errorf records a failure and continues.
func (t *T) Errorf(format string, args ...any) {
	t.failed = true
	t.msgs = append(t.msgs, fmt.Sprintf(format, args...))
}

// Fatalf records a failure and aborts the case.
func (t *T) Fatalf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	t.failed = true
	t.msgs = append(t.msgs, msg)
	panic(failSentinel{msg: msg})
}

// NoErr fails fatally if err is non-nil.
func (t *T) NoErr(err error, context string) {
	if err != nil {
		t.Fatalf("%s: %v", context, err)
	}
}

// Truef fails (non-fatal) if cond is false.
func (t *T) Truef(cond bool, format string, args ...any) {
	if !cond {
		t.Errorf(format, args...)
	}
}

// Equalf fails (non-fatal) if got != want.
func (t *T) Equalf(got, want any, format string, args ...any) {
	if got != want {
		t.Errorf("%s: got %v, want %v", fmt.Sprintf(format, args...), got, want)
	}
}

// WaitFor polls cond until it returns true or timeout elapses, sleeping
// interval between checks. It fails fatally on timeout.
func (t *T) WaitFor(cond func() bool, timeout, interval time.Duration, what string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("timed out after %s waiting for %s", timeout, what)
}

// message returns the joined failure messages.
func (t *T) message() string {
	if len(t.msgs) == 0 {
		return ""
	}
	out := t.msgs[0]
	for _, m := range t.msgs[1:] {
		out += "; " + m
	}
	return out
}
