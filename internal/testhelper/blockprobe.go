package testhelper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// Block-observation assertion names. blockHalt is the negation of blockAdvance
// (used to confirm a quorum-deficient network stops sealing); blockInterval
// checks the average block time over a recent window.
const (
	assertBlockHalt     = "blockHalt"
	assertBlockInterval = "blockInterval"
)

// Defaults for blockHalt: how long to watch, and how many blocks may still slip
// through (an in-flight block can seal the instant the nodes go down).
const (
	defaultBlockHaltWindow = 10 * time.Second
	defaultBlockHaltMaxAdv = 1
	defaultIntervalSamples = 20
	defaultIntervalMaxSecs = 60
	defaultIntervalMinSecs = 0
)

// blockHaltAssertion passes when the head advances by at most "maxAdvance"
// blocks over a "within" window — the network has stopped (or all but stopped)
// producing. It is the opposite of blockAdvance: rather than returning as soon
// as the head moves, it waits the whole window and checks how far it moved.
// A quorum-deficient BFT network (e.g. 2 of 4 validators down) cannot seal, so
// its head holds; one in-flight block may still land, hence maxAdvance defaults
// to 1.
//
// Args: on, within (watch window; default 10s), maxAdvance (default 1).
type blockHaltAssertion struct{}

func (blockHaltAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertBlockHalt, Provenance: ac.Spec}
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: blockHalt: no target node RPC URL")
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

	maxAdv := uint64(defaultBlockHaltMaxAdv)
	if v, ok := uintArg(ac.Spec["maxAdvance"]); ok {
		maxAdv = v
	}
	window := durationArg(ac.Spec, "within", defaultBlockHaltWindow)
	res.Expected = fmt.Sprintf("head advances <= %d over %s", maxAdv, window)

	// Watch the whole window: the point is to confirm the head does NOT climb,
	// which can only be shown by waiting, not by an early return.
	select {
	case <-ctx.Done():
		res.Actual = ctx.Err().Error()
		return res, ctx.Err()
	case <-time.After(window):
	}

	end, err := c.BlockNumber(ctx)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	advanced := end - start
	if end < start {
		advanced = 0
	}
	res.Actual = fmt.Sprintf("advanced %d (head %d -> %d)", advanced, start, end)
	res.Pass = advanced <= maxAdv
	if !res.Pass {
		res.Source = "head kept advancing — network did not halt"
	}
	return res, nil
}

// blockIntervalAssertion samples the timestamps of the last "blocks" blocks and
// passes when the average interval between them is within [minSeconds,
// maxSeconds]. It ports the block-time sanity check: a healthy chain seals on a
// bounded, positive cadence.
//
// Args: on, blocks (sample count; default 20), maxSeconds (default 60),
// minSeconds (default 0).
type blockIntervalAssertion struct{}

func (blockIntervalAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertBlockInterval, Provenance: ac.Spec}
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: blockInterval: no target node RPC URL")
		res.Actual = err.Error()
		return res, err
	}
	c, err := clientFor(ac.Deps, targets[0].url)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	head, err := c.BlockNumber(ctx)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}

	samples := uint64(defaultIntervalSamples)
	if v, ok := uintArg(ac.Spec["blocks"]); ok && v > 0 {
		samples = v
	}
	if samples > head {
		samples = head
	}
	if samples < 2 {
		res.Actual = fmt.Sprintf("only %d block(s) available, need 2", head)
		res.Source = "not enough blocks to measure an interval"
		return res, nil
	}

	first := head - samples + 1
	timestamps := make([]uint64, 0, samples)
	for n := first; n <= head; n++ {
		blk, err := c.BlockByNumber(ctx, "0x"+strconv.FormatUint(n, 16))
		if err != nil {
			res.Actual = err.Error()
			return res, err
		}
		timestamps = append(timestamps, blk.Timestamp)
	}
	avg, err := averageInterval(timestamps)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}

	maxSecs := float64(defaultIntervalMaxSecs)
	if v, ok := uintArg(ac.Spec["maxSeconds"]); ok {
		maxSecs = float64(v)
	}
	minSecs := float64(defaultIntervalMinSecs)
	if v, ok := uintArg(ac.Spec["minSeconds"]); ok {
		minSecs = float64(v)
	}
	res.Expected = fmt.Sprintf("avg interval in [%.0f, %.0f]s over %d blocks", minSecs, maxSecs, samples)
	res.Actual = fmt.Sprintf("%.2fs", avg)
	res.Pass = avg >= minSecs && avg <= maxSecs
	if !res.Pass {
		res.Source = "average block interval out of range"
	}
	return res, nil
}

// averageInterval returns the mean gap between consecutive timestamps (seconds).
// It requires at least two samples and rejects a non-monotonic series (a later
// block older than an earlier one is a bad read, not a valid cadence).
func averageInterval(timestamps []uint64) (float64, error) {
	if len(timestamps) < 2 {
		return 0, fmt.Errorf("dsl: blockInterval: need at least 2 timestamps, got %d", len(timestamps))
	}
	var total uint64
	for i := 1; i < len(timestamps); i++ {
		if timestamps[i] < timestamps[i-1] {
			return 0, fmt.Errorf("dsl: blockInterval: block timestamps are not monotonic (%d < %d)",
				timestamps[i], timestamps[i-1])
		}
		total += timestamps[i] - timestamps[i-1]
	}
	return float64(total) / float64(len(timestamps)-1), nil
}

// compile-time assertion that the RPC client exposes what these assertions read.
var _ = func(c *rpc.Client) { _, _ = c.BlockByNumber, c.BlockNumber }
