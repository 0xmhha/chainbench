package testhelper

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// burnInitCode is a contract-creation payload whose constructor spins in a tight
// loop while gasleft() > 100000, then returns empty runtime code. The
// transaction always succeeds (it RETURNs), and it consumes essentially its
// whole gas limit, so one such tx with gas = p% of the block gas limit fills the
// block it lands in to ~p%. This is the gas-burner the Anzeon baseFee tests use
// to drive block utilization above 20% (raise the base fee) — leaving blocks
// empty afterwards drives it back down.
//
//	PUSH1 0 | JUMPDEST | PUSH1 1 ADD | PUSH3 100000 | GAS GT | PUSH1 2 JUMPI | PUSH1 0 PUSH1 0 RETURN
const burnInitCode = "0x60005b600101620186a05a1160025760006000f3"

// burnReserveGas is the gasleft the burner stops at; the per-tx gas limit must
// sit comfortably above it or the loop never runs and no gas is burned.
const burnReserveGas = 100_000

// loadDefaultBlocks is how many consecutive blocks a load fills when "blocks" is
// omitted.
const loadDefaultBlocks = 1

// loadAction drives block gas usage by deploying the gas-burner (burnInitCode)
// once per block for "blocks" consecutive blocks. Each burn tx's gas limit is
// "fillPercent"% of the current block gas limit (or an explicit "gas"), so the
// block it lands in is filled to ~fillPercent%. Sending one burn per block and
// waiting for each receipt paces the load to one block apiece; sustaining it
// across blocks is what moves the Anzeon base fee monotonically, which a single
// tx cannot do (the DSL has no loop). Stopping the load and letting empty blocks
// pass (a plain waitBlock) lets the base fee fall again.
//
// Args:
//   - from        — funded sender (node-signed eth_sendTransaction); default: the node coinbase
//   - fillPercent — integer percent (1-100) of the block gas limit to consume per tx
//   - gas         — explicit per-tx gas limit (0x-hex or decimal); overrides fillPercent
//   - blocks      — number of consecutive blocks to fill (default 1)
//   - on          — node selector
//   - timeout     — per-burn receipt deadline (default defaultTxTimeout)
type loadAction struct{}

func (loadAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	from, err := funder(ctx, c, ac.Deps, ac.Args)
	if err != nil {
		return fmt.Errorf("dsl: load: %w", err)
	}

	blocks := loadDefaultBlocks
	if n, ok := uintArg(ac.Args["blocks"]); ok && n > 0 {
		blocks = int(n)
	}

	var lastHash string
	for i := 0; i < blocks; i++ {
		gasHex, err := loadGasLimit(ctx, c, ac.Args)
		if err != nil {
			return err
		}
		args := rpc.SendTxArgs{From: from, Data: burnInitCode, Gas: gasHex}
		if err := applyFeeArgs(&args, ac.Args); err != nil {
			return err
		}
		receipt, hash, err := sendAndConfirm(ctx, c, args, ac.Args)
		if err != nil {
			return fmt.Errorf("dsl: load: burn %d/%d: %w", i+1, blocks, err)
		}
		if statusReverted(receipt) {
			return fmt.Errorf("dsl: load: burn %d/%d reverted (%s)", i+1, blocks, hash)
		}
		lastHash = hash
	}
	ac.Hash = lastHash
	return nil
}

// loadGasLimit returns the per-tx gas limit as a 0x-hex quantity: an explicit
// "gas" argument if present, else "fillPercent"% of the latest block's gas
// limit. Reading the limit each call keeps the load correct if the chain adjusts
// its block gas limit over the run.
func loadGasLimit(ctx context.Context, c *rpc.Client, args map[string]any) (string, error) {
	if g, ok := hexQuantity(args["gas"]); ok {
		return g, nil
	}
	pct, ok := uintArg(args["fillPercent"])
	if !ok {
		return "", fmt.Errorf("dsl: load requires \"fillPercent\" (1-100) or an explicit \"gas\"")
	}
	blk, err := c.BlockByNumber(ctx, "latest")
	if err != nil {
		return "", fmt.Errorf("dsl: load: block gas limit: %w", err)
	}
	gas, err := burnGas(blk.GasLimit, pct)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%x", gas), nil
}

// burnGas is the pure gas-sizing rule: fillPercent% of the block gas limit,
// rejected when the percent is out of range or the result would not leave the
// burner enough gas to actually spin (so the block would not fill).
func burnGas(blockGasLimit, fillPercent uint64) (uint64, error) {
	if fillPercent == 0 || fillPercent > 100 {
		return 0, fmt.Errorf("dsl: load: fillPercent must be 1-100, got %d", fillPercent)
	}
	if blockGasLimit == 0 {
		return 0, fmt.Errorf("dsl: load: node reports a zero block gas limit")
	}
	gas := blockGasLimit / 100 * fillPercent
	if gas <= burnReserveGas {
		return 0, fmt.Errorf("dsl: load: fillPercent %d%% of block gas limit %d is below the burner floor (%d)",
			fillPercent, blockGasLimit, burnReserveGas)
	}
	return gas, nil
}
