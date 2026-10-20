package testhelper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// recordContractDeployment observes the submitted transaction rather than
// guessing a nonce from the account's changing state or recording a secret key.
// Failure to observe is explicit and does not retry or resubmit a transaction.
func recordContractDeployment(ctx context.Context, ac *interp.ActionCtx, client *rpc.Client, requestedCode string) {
	observed := &session.ContractDeployment{}
	ac.Deployment = observed
	observed.ActualAddress, _ = ac.Receipt["contractAddress"].(string)
	bytes, err := hex.DecodeString(strings.TrimPrefix(requestedCode, "0x"))
	if err == nil {
		sha := sha256.Sum256(bytes)
		observed.BytecodeSHA256 = hex.EncodeToString(sha[:])
		observed.Artifact = "sha256:" + observed.BytecodeSHA256
	}
	var transaction *struct {
		From  string `json:"from"`
		Nonce string `json:"nonce"`
		Input string `json:"input"`
		Hash  string `json:"hash"`
	}
	if err := client.Call(ctx, "eth_getTransactionByHash", &transaction, ac.Hash); err != nil || transaction == nil {
		observed.ObservationError = "transaction_metadata_unavailable"
		return
	}
	if transaction.Hash != "" && transaction.Hash != ac.Hash {
		observed.ObservationError = "transaction_hash_mismatch"
		return
	}
	observed.Deployer = transaction.From
	if !strings.HasPrefix(transaction.Nonce, "0x") {
		observed.ObservationError = "transaction_nonce_unavailable"
		return
	}
	nonce, err := strconv.ParseUint(strings.TrimPrefix(transaction.Nonce, "0x"), 16, 64)
	if err != nil {
		observed.ObservationError = "transaction_nonce_invalid"
		return
	}
	observed.Nonce = &nonce
	expected, err := accounts.CreateAddress(transaction.From, nonce)
	if err != nil {
		observed.ObservationError = "transaction_deployer_invalid"
		return
	}
	observed.ExpectedAddress = expected
	deployed, err := hex.DecodeString(strings.TrimPrefix(transaction.Input, "0x"))
	if err != nil || transaction.Input == "" {
		observed.ObservationError = "transaction_bytecode_unavailable"
		return
	}
	sha := sha256.Sum256(deployed)
	observed.BytecodeSHA256 = hex.EncodeToString(sha[:])
	observed.Artifact = "sha256:" + observed.BytecodeSHA256
	if observed.ActualAddress == "" {
		observed.ObservationError = "deployment_address_unavailable"
		return
	}
	if !strings.EqualFold(expected, observed.ActualAddress) {
		observed.ObservationError = "deployment_address_mismatch"
	}
}
