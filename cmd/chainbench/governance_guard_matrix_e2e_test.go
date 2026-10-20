//go:build e2e

package main

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

type guardState struct {
	Staked, Delegated string
	Active            bool
}
type guardCase struct {
	Name, Input, Sender, Value, Reason, Subject string
	Before, After                               guardState
	Failure                                     minedStakingFailure
}

func verifyGovernanceGuards(cases []guardCase, minimum *big.Int, emergency bool) error {
	if minimum == nil || minimum.Sign() <= 0 {
		return fmt.Errorf("invalid minimum")
	}
	wantCount := 1
	staked := new(big.Int).Mul(minimum, big.NewInt(2))
	reason := "amount must equal balance to deactivate staker"
	if emergency {
		wantCount = 3
		staked.Set(minimum)
		reason = "operation not permitted by council"
	}
	if len(cases) != wantCount {
		return fmt.Errorf("incomplete operation matrix")
	}
	seen := map[string]bool{}
	for _, c := range cases {
		if c.Subject == "" || c.Subject != cases[0].Subject || c.Sender == "" || seen[c.Name] || c.Reason != reason || c.Before != c.After || !c.Before.Active || c.Before.Staked != staked.String() {
			return fmt.Errorf("wrong state or guard context")
		}
		seen[c.Name] = true
		input := ""
		value := big.NewInt(0)
		if !emergency {
			if c.Name != "unstake" {
				return fmt.Errorf("not original minimum unstake")
			}
			input = accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(new(big.Int).Add(minimum, big.NewInt(1))))
		} else {
			switch c.Name {
			case "stake":
				input = accounts.EncodeCallArgs("stake(uint256)", accounts.Uint(minimum))
				value.Set(minimum)
			case "unstake":
				input = accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(big.NewInt(1)))
			case "delegate":
				input = accounts.EncodeCallArgs("delegate(address,uint256)", accounts.Address(c.Subject), accounts.Uint(minimum))
				value.Set(minimum)
			default:
				return fmt.Errorf("unknown emergency operation")
			}
		}
		if c.Input != input || c.Value != value.String() {
			return fmt.Errorf("wrong actual operation input")
		}
		if err := validateMinedStakingFailure(c.Failure, c.Sender, input, value, reason); err != nil {
			return err
		}
	}
	return nil
}

func guardFixture(emergency bool) []guardCase {
	minimum := big.NewInt(100)
	subject := "0x1111111111111111111111111111111111111111"
	names := []string{"unstake"}
	if emergency {
		names = []string{"stake", "unstake", "delegate"}
	}
	result := []guardCase{}
	for _, name := range names {
		c := guardCase{Name: name, Sender: "sender", Subject: subject, Value: "0", Reason: "amount must equal balance to deactivate staker", Before: guardState{Staked: "200", Delegated: "0", Active: true}}
		c.Input = accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(big.NewInt(101)))
		if emergency {
			c.Reason = "operation not permitted by council"
			c.Before.Staked = "100"
			switch name {
			case "stake":
				c.Input = accounts.EncodeCallArgs("stake(uint256)", accounts.Uint(minimum))
				c.Value = "100"
			case "unstake":
				c.Input = accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(big.NewInt(1)))
			case "delegate":
				c.Input = accounts.EncodeCallArgs("delegate(address,uint256)", accounts.Address(subject), accounts.Uint(minimum))
				c.Value = "100"
			}
		}
		c.After = c.Before
		value, _ := new(big.Int).SetString(c.Value, 10)
		c.Failure = minedStakingFailure{Hash: name, Transaction: map[string]any{"hash": name, "blockHash": "block", "from": c.Sender, "to": e2eGovStaking, "input": c.Input, "value": "0x" + value.Text(16), "gas": "0x7a120"}, Receipt: map[string]any{"transactionHash": name, "blockHash": "block", "blockNumber": "0x5", "status": "0x0", "gasUsed": "0x8000"}, Trace: stakingFailureTrace{Failed: true, ReturnValue: accounts.EncodeCallArgs("Error(string)", accounts.StringArg(c.Reason))}}
		result = append(result, c)
	}
	return result
}
func TestGovernanceGuardsRejectEstimateAndIncompleteMatrix(t *testing.T) {
	for _, emergency := range []bool{false, true} {
		if err := verifyGovernanceGuards(guardFixture(emergency), big.NewInt(100), emergency); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name   string
		change func([]guardCase) []guardCase
	}{
		{"estimate failure only", func(p []guardCase) []guardCase { p[0].Failure.Receipt = nil; p[0].Failure.Transaction = nil; return p }},
		{"missing emergency action", func(p []guardCase) []guardCase { return p[:2] }},
		{"duplicate action", func(p []guardCase) []guardCase { p[2] = p[1]; return p }},
		{"wrong reason", func(p []guardCase) []guardCase { p[0].Reason = "HTTP error"; return p }},
		{"state changed", func(p []guardCase) []guardCase { p[0].After.Staked = "101"; return p }},
		{"wrong operation input", func(p []guardCase) []guardCase { p[0].Input = "0x00"; return p }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.change(guardFixture(true))
			if err := verifyGovernanceGuards(p, big.NewInt(100), true); err == nil {
				t.Fatal("unproven guard matrix accepted")
			}
		})
	}
	p := guardFixture(false)
	p[0].Before.Staked = "100"
	p[0].After = p[0].Before
	if err := verifyGovernanceGuards(p, big.NewInt(100), false); err == nil {
		t.Fatal("wrong original double-minimum state accepted")
	}
}

func readGovernanceGuardState(t *testing.T, c *rpc.Client, staker string) guardState {
	t.Helper()
	return guardState{Staked: stakingUint(t, c, "getStakerAmount(address)", staker).String(), Delegated: stakingUint(t, c, "getDelegatedAmount(address)", staker).String(), Active: stakingIsStaker(t, c, staker)}
}

// The guard caller must not receive consensus income during gas/refund accounting.
// This committed test-only operator key is distinct from the validator keys.
func governanceGuardOverlay(t *testing.T, emergency bool) (map[string]any, []byte) {
	t.Helper()
	key := mustHexBytes(t, scenarioOpKeys["op2"])
	provider, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatal(err)
	}
	address, err := provider.AddressForKey(key)
	if err != nil {
		t.Fatal(err)
	}
	overlay := map[string]any{"alloc": map[string]any{address: map[string]any{"balance": "0x33b2e3c9fd0803ce8000000"}}}
	if emergency {
		overlay["config"] = map[string]any{"croissant": map[string]any{"wBFT": map[string]any{"useNCP": true}}}
	}
	return overlay, key
}
func assertGuardCallerOutsideValidators(t *testing.T, c *rpc.Client, caller string) {
	t.Helper()
	var validators []string
	if err := c.Call(t.Context(), "istanbul_getValidators", &validators, "latest"); err != nil || len(validators) == 0 {
		t.Fatal("missing actual validators", err)
	}
	for _, address := range validators {
		if strings.EqualFold(address, caller) {
			t.Fatal("guard sender also receives consensus income")
		}
	}
}
