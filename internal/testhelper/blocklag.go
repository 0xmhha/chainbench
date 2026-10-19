package testhelper

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

const defaultBlockLagMax uint64 = 2

// blockLagAssertion polls fresh heads together. Without reference it measures
// the spread of all selected nodes. With reference it measures how far targets
// trail that node; a target ahead of the reference is allowed.
type blockLagAssertion struct{}

type lagSample struct {
	ObservedAt string            `json:"observedAt"`
	Heads      map[string]uint64 `json:"heads"`
	Errors     map[string]string `json:"errors,omitempty"`
	Lag        string            `json:"lag,omitempty"`
	Pass       bool              `json:"pass"`
}

func (blockLagAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertBlockLag, Provenance: ac.Spec}
	fail := func(err error) (session.AssertResult, error) { res.Actual = err.Error(); return res, err }
	for _, field := range []string{"timeout", "pollInterval"} {
		if err := validateDurationControl(assertBlockLag, field, ac.Spec); err != nil {
			return fail(err)
		}
	}
	if err := validateUnsignedControl("assertion", assertBlockLag, "maxLag", ac.Spec); err != nil {
		return fail(err)
	}
	maximum := defaultBlockLagMax
	if value, ok := ac.Spec["maxLag"]; ok {
		maximum, _ = uintArg(value)
	}
	reference := ""
	if value, present := ac.Spec["reference"]; present {
		var ok bool
		reference, ok = value.(string)
		if !ok || reference == "" {
			return fail(fmt.Errorf("dsl: blockLag: bad reference selector"))
		}
	}
	targets, reference, err := lagTargets(ac, reference)
	if err != nil {
		return fail(err)
	}
	res.Expected = map[string]any{"maxLag": maximum, "reference": reference}
	timeout := durationArg(ac.Spec, "timeout", defaultBlockAdvanceTimeout)
	poll := durationArg(ac.Spec, "pollInterval", defaultBlockAdvancePoll)
	pctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	started := time.Now()
	samples := []lagSample{}
	attempts := 0
	dropped := 0
	actual := func() map[string]any {
		return map[string]any{"reference": reference, "attempts": attempts, "samples": samples, "droppedSamples": dropped, "elapsedMillis": time.Since(started).Milliseconds()}
	}
	for {
		if pctx.Err() != nil {
			res.Actual = actual()
			return res, fmt.Errorf("dsl: blockLag: convergence not reached within %s: %w", timeout, pctx.Err())
		}
		sample := sampleLagHeads(pctx, ac, targets, reference, maximum)
		attempts++
		samples = append(samples, sample)
		// Bound diagnostic retention. Original90/120s recovery cases poll every2s,
		// so their complete trajectories fit; faster custom cases expose truncation.
		if len(samples) > 128 {
			samples = samples[1:]
			dropped++
		}
		res.Actual = actual()
		if pctx.Err() == nil && sample.Pass {
			res.Pass = true
			return res, nil
		}
		select {
		case <-pctx.Done():
		case <-ticker.C:
		}
	}
}

func lagTargets(ac *interp.AssertCtx, reference string) ([]assertTarget, string, error) {
	selected := *ac
	if len(ac.On) == 0 && ac.Env != nil {
		selected.On = ac.Env.Nodes()
	}
	targets := assertTargets(&selected)
	if reference != "" {
		if ac.Env == nil {
			return nil, "", fmt.Errorf("dsl: blockLag: reference requires a node table")
		}
		n, err := ac.Env.Resolve(reference)
		if err != nil {
			return nil, "", fmt.Errorf("dsl: blockLag: reference: %w", err)
		}
		reference = fmt.Sprintf("node%d", n.Index)
		found := false
		for _, target := range targets {
			found = found || target.name == reference
		}
		if !found {
			targets = append([]assertTarget{{name: reference, url: n.RPCURL}}, targets...)
		}
	}
	if len(targets) < 2 {
		return nil, "", fmt.Errorf("dsl: blockLag: requires at least two distinct nodes")
	}
	names := map[string]bool{}
	urls := map[string]bool{}
	for _, target := range targets {
		if target.url == "" || names[target.name] || urls[target.url] {
			return nil, "", fmt.Errorf("dsl: blockLag: missing or duplicate endpoint for %s", target.name)
		}
		names[target.name] = true
		urls[target.url] = true
	}
	return targets, reference, nil
}

func sampleLagHeads(ctx context.Context, ac *interp.AssertCtx, targets []assertTarget, reference string, maximum uint64) lagSample {
	sample := lagSample{Heads: map[string]uint64{}, Errors: map[string]string{}}
	for _, target := range targets {
		c, err := clientFor(ac.Deps, target.url)
		if err != nil {
			sample.Errors[target.name] = err.Error()
			continue
		}
		head, err := c.BlockNumber(ctx)
		if err != nil {
			sample.Errors[target.name] = err.Error()
			continue
		}
		sample.Heads[target.name] = head
		if head == 0 {
			sample.Errors[target.name] = "head is zero"
		}
	}
	sample.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if len(sample.Errors) > 0 {
		return sample
	}
	var lag *big.Int
	if reference != "" {
		for name, head := range sample.Heads {
			if name == reference {
				continue
			}
			difference := new(big.Int).Sub(new(big.Int).SetUint64(sample.Heads[reference]), new(big.Int).SetUint64(head))
			if lag == nil || difference.Cmp(lag) > 0 {
				lag = difference
			}
		}
	} else {
		low := ^uint64(0)
		high := uint64(0)
		for _, head := range sample.Heads {
			low = min(low, head)
			high = max(high, head)
		}
		lag = new(big.Int).SetUint64(high - low)
	}
	if lag != nil {
		sample.Lag = lag.String()
		sample.Pass = lag.Cmp(new(big.Int).SetUint64(maximum)) <= 0
	}
	return sample
}
