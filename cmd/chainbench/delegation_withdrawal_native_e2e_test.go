//go:build e2e

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func unbondingPeriodOverlay(alloc map[string]any, delegator, staker uint64) map[string]any {
	overlay := map[string]any{"config": map[string]any{"croissant": map[string]any{"govContracts": map[string]any{"govConfig": map[string]any{"params": map[string]any{"unbondingPeriodDelegator": fmt.Sprint(delegator), "unbondingPeriodStaker": fmt.Sprint(staker)}}}}}}
	if len(alloc) > 0 {
		overlay["alloc"] = alloc
	}
	return overlay
}

func captureDelegationWithdrawal(t *testing.T, c *rpc.Client, caller string, index uint64, hash string) credentialWithdrawal {
	t.Helper()
	w := credentialWithdrawal{Hash: hash, Index: index}
	if err := c.Call(t.Context(), "eth_getTransactionByHash", &w.Transaction, hash); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(t.Context(), "eth_getTransactionReceipt", &w.Receipt, hash); err != nil {
		t.Fatal(err)
	}
	block, err := c.BlockByNumber(t.Context(), fmt.Sprint(w.Receipt["blockNumber"]))
	if err != nil {
		t.Fatal(err)
	}
	w.Timestamp = block.Timestamp
	w.Credentials = []withdrawalCredential{credentialAt(t, c, caller, index)}
	w.CredentialIndex, w.WithdrawalIndex = credentialPointers(t, c, caller)
	logs, err := json.Marshal(feeReceipt(t, c, hash))
	if err != nil {
		t.Fatal(err)
	}
	w.LogsJSON = string(logs)
	if err := c.Call(t.Context(), "debug_traceTransaction", &w.Diff, hash, map[string]any{"tracer": "prestateTracer", "tracerConfig": map[string]any{"diffMode": true}}); err != nil {
		t.Fatal(err)
	}
	return w
}

// The original72h/7d configuration is deployed and read, without pretending
// to have waited those periods. The separate lifecycle uses approved30s/70s.
func TestWemixGovernanceOriginalUnbondingConfigurationE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	c := rpc.Dial(runGovHandoffOverlay(t, from, to, unbondingPeriodOverlay(nil, 259200, 604800)))
	delegator, staker := govConfigUint(t, c, "unbondingPeriodDelegator()"), govConfigUint(t, c, "unbondingPeriodStaker()")
	if delegator.Cmp(big.NewInt(259200)) != 0 || staker.Cmp(big.NewInt(604800)) != 0 {
		t.Fatal("original GOV-011 configured periods differ", delegator, staker)
	}
	var raw [2]string
	for i, slot := range []string{"0x3", "0x2"} {
		if err := c.Call(t.Context(), "eth_getStorageAt", &raw[i], e2eGovConfig, slot, "latest"); err != nil {
			t.Fatal(err)
		}
	}
	if raw[0] != "0x"+fmt.Sprintf("%064x", 259200) || raw[1] != "0x"+fmt.Sprintf("%064x", 604800) {
		t.Fatal("original period storage differs", raw)
	}
	logGovernanceProof(t, "GOV-011-original-period-configuration", map[string]any{"delegator": delegator.String(), "staker": staker.String(), "storage": raw, "literal72hAnd7dWaitPerformed": false, "scope": "Explicit original-number fixture configuration; not a measurement of mainnet deployment parameters."})
}

func TestWemixGovernanceDelegationWithdrawalE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	provider, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatal(err)
	}
	keys := [][]byte{mustHexBytes(t, scenarioOpKeys["op1"]), mustHexBytes(t, scenarioOpKeys["op2"])}
	alloc := map[string]any{}
	for _, key := range keys {
		address, err := provider.AddressForKey(key)
		if err != nil {
			t.Fatal(err)
		}
		alloc[address] = map[string]any{"balance": "0x33b2e3c9fd0803ce8000000"}
	}
	url := runGovHandoffOverlay(t, from, to, unbondingPeriodOverlay(alloc, 30, 70))
	c := rpc.Dial(url)
	wallets := []accounts.Wallet{}
	for _, key := range keys {
		w, err := provider.OpenWallet(t.Context(), key, url)
		if err != nil {
			t.Fatal(err)
		}
		wallets = append(wallets, w)
		assertGuardCallerOutsideValidators(t, c, w.Address())
	}
	operator, delegator := wallets[0], wallets[1]
	staker := presetNodeAddr(t, 1)
	pk, sig := presetNodeBLS(t, 1)
	minimum := govConfigUint(t, c, "minimumStaking()")
	stakingRegister(t, c, operator, staker, pk, sig, minimum)
	p := delegationWithdrawalProof{Staker: staker, Operator: operator.Address(), StakerAmount: stakingUint(t, c, "getStakerAmount(address)", staker).String(), DelegatorDelay: govConfigUint(t, c, "unbondingPeriodDelegator()").String(), StakerDelay: govConfigUint(t, c, "unbondingPeriodStaker()").String()}
	caller := delegator.Address()
	index, oldest := credentialPointers(t, c, caller)
	n, err := strconv.ParseUint(index, 10, 64)
	if err != nil || index != oldest {
		t.Fatal("unexpected preexisting credential", index, oldest, err)
	}
	p.Withdrawal = autoWithdrawalProof{Ledger: credentialExpiryProof{Caller: caller, Contract: e2eGovStaking, Amount: minimum.String(), N: n}, Delay: 30}
	p.DelegatedBefore = stakingUint(t, c, "getDelegatedAmount(address)", staker).String()
	p.PowerBefore = stakingInfoWord(t, c, staker, 5).String()
	input := accounts.EncodeCallArgs("delegate(address,uint256)", accounts.Address(staker), accounts.Uint(minimum))
	p.Delegate = captureDelegationWithdrawal(t, c, caller, n, stakingSendOK(t, c, delegator, input, minimum))
	p.DelegatedAfter = stakingUint(t, c, "getDelegatedAmount(address)", staker).String()
	p.PowerAfter = stakingInfoWord(t, c, staker, 5).String()
	input = accounts.EncodeCallArgs("undelegate(address,uint256)", accounts.Address(staker), accounts.Uint(minimum))
	p.Undelegate = captureDelegationWithdrawal(t, c, caller, n, stakingSendOK(t, c, delegator, input, nil))
	p.DelegatedFinal = stakingUint(t, c, "getDelegatedAmount(address)", staker).String()
	p.PowerFinal = stakingInfoWord(t, c, staker, 5).String()
	if !stakingIsStaker(t, c, staker) {
		t.Fatal("active staker removed by undelegation")
	}
	p.Withdrawal.Created = credentialAt(t, c, caller, n)
	if p.Withdrawal.Created.RequestTime != fmt.Sprint(p.Undelegate.Timestamp) {
		t.Fatal("credential was not created in actual undelegation block")
	}
	input = accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(new(big.Int)))
	p.Withdrawal.Early = captureDelegationWithdrawal(t, c, caller, n, stakingSendOK(t, c, delegator, input, nil))
	p.Withdrawal.AfterEarly = credentialAt(t, c, caller, n)
	h, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(p.Withdrawal.Early.Receipt["blockNumber"]), "0x"), 16, 64)
	if err != nil || h == 0 {
		t.Fatal("invalid early block", err)
	}
	if err := c.Call(t.Context(), "eth_getBalance", &p.Withdrawal.EarlyContractBefore, e2eGovStaking, fmt.Sprintf("0x%x", h-1)); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(t.Context(), "eth_getBalance", &p.Withdrawal.EarlyContractAfter, e2eGovStaking, fmt.Sprintf("0x%x", h)); err != nil {
		t.Fatal(err)
	}
	mature, err := strconv.ParseUint(p.Withdrawal.Created.WithdrawableTime, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	waitCredentialTime(t, c, mature)
	p.Withdrawal.Success = captureDelegationWithdrawal(t, c, caller, n, stakingSendOK(t, c, delegator, input, nil))
	logGovernanceProof(t, "GOV-011-delegation-withdrawal", p)
	if err := verifyDelegationWithdrawal(p); err != nil {
		t.Fatal(err)
	}
}
