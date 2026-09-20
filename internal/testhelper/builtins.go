package testhelper

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/dsl/interp"

	"github.com/0xmhha/chainbench/internal/accounts"
)

// The builtin vocabulary: the names every action and assertion is registered
// under, and the registry that holds them.
//
// The name tables stay here with Register. Splitting a registration from the
// table it reads is how a name ends up registered twice, or not at all.
//
// Sending a transaction is in sendtx.go, and judging what one did — plus the
// argument shapes a step declares — in outcome.go.

// on). Kept as typed-free string constants so the registry and specs share one
// source of truth.
const (
	actionSendTx          = "sendTx"
	actionWaitBlock       = "waitBlock"
	actionWaitFor         = interp.ActionWaitFor
	actionNewAccount      = "newAccount"
	actionSendRawTampered = "sendRawTampered"
	actionSendSetCode     = "sendSetCode"
	actionSignAuth        = "signAuthorization"
	actionLoad            = "load"

	assertChainID          = "chainId"
	assertBlockNumber      = "blockNumber"
	assertPeerCount        = "peerCount"
	assertBalanceAt        = "balanceAt"
	assertCodeAt           = "codeAt"
	assertNonceAt          = "nonceAt"
	assertCall             = "call"
	assertTxStatus         = "txStatus"
	assertReceiptLog       = "receiptLog"
	assertBlockAdvance     = "blockAdvance"
	assertBlockStalled     = "blockStalled"
	assertSameBlockHash    = "sameBlockHash"
	assertBaseFee          = "baseFee"
	assertEstimateGas      = "estimateGas"
	assertTxMined          = "txMined"
	assertCallError        = "callError"
	assertMethodPresent    = "methodPresent"
	assertCreateAddress    = "createAddress"
	assertContractChecksum = "contractChecksum"
)

// Defaults for the sendTx wait loop, overridable per action via args.
const (
	defaultTxTimeout      = 30 * time.Second
	defaultTxPollInterval = 500 * time.Millisecond
)

// Defaults for the waitBlock poll loop, overridable per action via args.
const (
	defaultWaitBlockTimeout = 60 * time.Second
	defaultWaitBlockPoll    = 500 * time.Millisecond
)

// Defaults for the blockAdvance assertion poll loop, overridable per spec.
const (
	defaultBlockAdvanceTimeout = 30 * time.Second
	defaultBlockAdvancePoll    = 500 * time.Millisecond
)

// Registry returns a fresh registry with the built-ins registered — the
// default vocabulary a run or `validate` resolves against.
func Registry() interp.Registry {
	r := interp.NewRegistry()
	Register(r)
	return r
}

// Register puts the built-in vocabulary on r: the actions a spec can do, the
// assertions it can check, and the readers "read" and "waitFor" draw from.
// The grammar and interpreter (dsl) know none of these — a run wires
// them by calling this, and a test that wants a narrower vocabulary registers
// its own.
func Register(r interp.Registry) {
	r.RegisterAction(actionSendTx, sendTxAction{})
	r.RegisterAction(actionWaitBlock, waitBlockAction{})
	r.RegisterAction(actionWaitFor, waitForAction{})
	r.RegisterAction(interp.ActionRead, readAction{})
	r.RegisterAction(actionNewAccount, newAccountAction{})
	r.RegisterAction(actionSendRawTampered, sendRawTamperedAction{})
	r.RegisterAction(actionSendSetCode, sendSetCodeAction{})
	r.RegisterAction(actionSignAuth, signAuthorizationAction{})
	r.RegisterAction(actionLoad, loadAction{})
	seedFaultBuiltins(r)
	seedCrossForkBuiltins(r)
	seedAssetBuiltins(r)
	seedDerivedBuiltins(r)
	r.RegisterAssertion(assertBlockAdvance, blockAdvanceAssertion{})
	r.RegisterAssertion(assertBlockStalled, blockStalledAssertion{})
	r.RegisterAssertion(assertBlockHalt, blockHaltAssertion{})
	r.RegisterAssertion(assertBlockInterval, blockIntervalAssertion{})
	r.RegisterAssertion(assertSameBlockHash, sameBlockHashAssertion{})
	r.RegisterAssertion(assertMetric, metricAssertion{})
	r.RegisterAssertion(assertCallError, callErrorAssertion{})
	r.RegisterAssertion(assertMethodPresent, methodPresentAssertion{})
	for _, a := range builtinAssertions() {
		r.RegisterAssertion(a.name, a)
		r.RegisterReader(a.name, interp.Reader(a.read))
	}
	r.RegisterReader(assertTxMined, interp.Reader(readTxMined))
	r.RegisterAssertion(assertTxMined, txMinedAssertion{})
}

// waitBlockAction blocks until the target node's height reaches "target" (a
// number) or the timeout elapses. Args: target, on, timeout, pollInterval.
type waitBlockAction struct{}

func (waitBlockAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	target, ok := uintArg(ac.Args["target"])
	if !ok {
		return fmt.Errorf("dsl: waitBlock requires a numeric \"target\"")
	}
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, durationArg(ac.Args, "timeout", defaultWaitBlockTimeout))
	defer cancel()
	t := time.NewTicker(durationArg(ac.Args, "pollInterval", defaultWaitBlockPoll))
	defer t.Stop()
	for {
		if bn, err := c.BlockNumber(ctx); err == nil && bn >= target {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("dsl: waitBlock: height %d not reached: %w", target, ctx.Err())
		case <-t.C:
		}
	}
}

// sendTxAction submits a node-signed transaction and, unless wait:false, polls
// for its receipt before returning.

type newAccountAction struct{}

// Do generates a key pair and records the address as the step value plus the
// private key under the "saveKey" binding. Args: save (address binding, the
// usual step "save"), saveKey (private-key binding name, required).
func (newAccountAction) Do(_ context.Context, ac *interp.ActionCtx) error {
	keyName, _ := ac.Args["saveKey"].(string)
	if keyName == "" {
		return fmt.Errorf("dsl: newAccount requires \"saveKey\"")
	}
	priv, addr, err := accounts.GenerateKey()
	if err != nil {
		return fmt.Errorf("dsl: newAccount: %w", err)
	}
	ac.Value = addr
	if ac.Extra == nil {
		ac.Extra = map[string]any{}
	}
	ac.Extra[keyName] = hex.EncodeToString(priv)
	return nil
}

// checkTxOutcome enforces a tx step's declared expectation (F11 — a tx step is
// atomically successful only when its expectation is met). By default the
// transaction must not revert. With expectRevert (or expect:"revert") the
// transaction MUST revert: a success then fails the step, so negative cases are
// expressed declaratively.
