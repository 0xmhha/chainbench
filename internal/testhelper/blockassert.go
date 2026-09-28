package testhelper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/0xmhha/chainbench/internal/dsl/interp"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/assert"
)

// Assertions about how a chain is moving: the head advancing, standing still,
// or every node agreeing on the block at a height.
//
// They are together because they share one shape — sample the chain twice and
// compare — and that shape is where a flaky consensus test usually goes wrong.

// sameBlockHashAssertion passes when every target node reports the same hash for
// a block — a cross-node no-fork / same-chain check. Spec: block (tag, default
// "latest"; use "0x0" for genesis agreement), on/onEach (default: every node).
// A node that does not answer fails the check; a node that answers without the
// block is skipped, and it fails if none has it.
type sameBlockHashAssertion struct{}

// Check compares the targets' hashes for the block.
//
// Without "on"/"onEach" it compares every node of the network. It used to fall
// back to the primary node like the single-node assertions do, which made it a
// comparison of one hash with itself: every case that asked "do the nodes
// agree" without naming them passed whatever the other nodes held.
//
// "latest" is read as the lowest head among the nodes, not each node's own
// latest: nodes a block apart hold different latest blocks, and comparing those
// failed a network that agreed on everything it had in common.
func (sameBlockHashAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertSameBlockHash, Provenance: ac.Spec}
	targets := assertTargets(ac)
	if len(ac.On) == 0 {
		targets = allTargets(ac)
	}
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: sameBlockHash: no target node RPC URL")
		res.Actual = err.Error()
		return res, err
	}
	tag, _ := ac.Spec["block"].(string)
	if tag == "" || tag == "latest" {
		lowest, err := lowestHead(ctx, ac.Deps, targets)
		if err != nil {
			res.Actual = err.Error()
			return res, err
		}
		tag = fmt.Sprintf("0x%x", lowest)
	}
	res.Expected = fmt.Sprintf("all %d node(s) agree on block %s hash", len(targets), tag)

	var hashes []string
	perNode := make(map[string]any, len(targets))
	for _, tgt := range targets {
		c, err := clientFor(ac.Deps, tgt.url)
		if err != nil {
			res.Actual = err.Error()
			return res, err
		}
		b, err := c.BlockByNumber(ctx, tag)
		if err != nil {
			res.Actual = err.Error()
			return res, err
		}
		if b.Hash == "" {
			continue // node has not produced this block yet
		}
		hashes = append(hashes, b.Hash)
		perNode[tgt.name] = b.Hash
	}
	res.Actual = perNode
	if len(hashes) == 0 {
		res.Pass, res.Source = false, "no node returned block "+tag
		return res, nil
	}
	pass, detail := assert.HashesEqual(hashes)
	res.Pass = pass
	if !pass {
		res.Source = detail
	}
	return res, nil
}

// lowestHead is the smallest head block number among targets: the newest block
// every one of them holds.
func lowestHead(ctx context.Context, deps *interp.Deps, targets []assertTarget) (uint64, error) {
	var lowest uint64
	for i, tgt := range targets {
		c, err := clientFor(deps, tgt.url)
		if err != nil {
			return 0, err
		}
		n, err := c.BlockNumber(ctx)
		if err != nil {
			return 0, fmt.Errorf("dsl: sameBlockHash: %s head: %w", tgt.name, err)
		}
		if i == 0 || n < lowest {
			lowest = n
		}
	}
	return lowest, nil
}

// blockStalledAssertion is the negation of blockAdvance: it passes when the
// target node's head does NOT move for the whole window. Spec: timeout,
// pollInterval, on.
//
// It exists because "the chain must stop here" is a real expectation — a
// genesis the node cannot commit halts consensus at that block — and asserting
// it with blockAdvance inverted is not possible: an assertion that fails is a
// failed test, not a satisfied negative. Waiting the full window is the point,
// so unlike blockAdvance this one cannot return early on success.
type blockStalledAssertion struct{}

func (blockStalledAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertBlockStalled, Provenance: ac.Spec}
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: blockStalled: no target node RPC URL")
		res.Actual = err.Error()
		return res, err
	}
	c, err := clientFor(ac.Deps, targets[0].url)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	start, err := c.BlockNumber(ctx)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	res.Expected = "head stays at " + strconv.FormatUint(start, 10)

	timeout := durationArg(ac.Spec, "timeout", defaultBlockAdvanceTimeout)
	pctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	t := time.NewTicker(durationArg(ac.Spec, "pollInterval", defaultBlockAdvancePoll))
	defer t.Stop()
	for {
		select {
		case <-pctx.Done():
			// The window closed with the head where it started: stalled.
			res.Pass, res.Actual = true, start
			return res, nil
		case <-t.C:
			cur, err := c.BlockNumber(pctx)
			if err != nil {
				// An unreachable node is not a moving chain; keep waiting.
				continue
			}
			if cur > start {
				res.Actual = cur
				res.Source = "head advanced to " + strconv.FormatUint(cur, 10)
				return res, nil
			}
		}
	}
}

// blockAdvanceAssertion passes when the target node's head advances within the
// poll window — proof the network is producing blocks. Spec: timeout,
// pollInterval, on. It reads the head once, then polls until a higher head or
// the timeout.
type blockAdvanceAssertion struct{}

func (blockAdvanceAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertBlockAdvance, Provenance: ac.Spec}
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: blockAdvance: no target node RPC URL")
		res.Actual = err.Error()
		return res, err
	}
	c, err := clientFor(ac.Deps, targets[0].url)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	start, err := c.BlockNumber(ctx)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	res.Expected = "head > " + strconv.FormatUint(start, 10)

	timeout := durationArg(ac.Spec, "timeout", defaultBlockAdvanceTimeout)
	pctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	t := time.NewTicker(durationArg(ac.Spec, "pollInterval", defaultBlockAdvancePoll))
	defer t.Stop()
	for {
		if cur, err := c.BlockNumber(pctx); err == nil && cur > start {
			res.Pass, res.Actual = true, cur
			return res, nil
		}
		select {
		case <-pctx.Done():
			res.Pass, res.Actual = false, start
			res.Source = "head did not advance within " + timeout.String()
			return res, nil
		case <-t.C:
		}
	}
}

// readAction reads one RPC value and, with "save", binds it for later steps and
// assertions to reference as "$name" (design §3.2b). It is how a spec compares
// two on-chain reads to each other — read the first, then assert the second
// against "$name" — which a single-shot assertion cannot express.
//
// Args: source (one of the RPC-reading assertion names), save, on, plus whatever
// that source needs (to/data for "call", address for "balanceAt", ...).
type readAction struct{}
