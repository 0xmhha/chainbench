package testhelper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestDeployContractRecordsObservedNonceAndBytecode(t *testing.T) {
	const deployer = "0x6ac7ea33f8831ea9dcc53393aaa88b25a785dbf0"
	address, err := accounts.CreateAddress(deployer, 1)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		var result any
		switch request.Method {
		case "eth_coinbase":
			result = deployer
		case "eth_sendTransaction":
			result = "0xtxhash"
		case "eth_getTransactionReceipt":
			result = map[string]any{"status": "0x1", "contractAddress": address, "blockNumber": "0x3", "transactionHash": "0xtxhash"}
		case "eth_getTransactionByHash":
			result = map[string]any{"from": deployer, "nonce": "0x1", "input": "0x6080604052", "hash": "0xtxhash"}
		default:
			t.Errorf("unexpected RPC %s", request.Method)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}))
	defer server.Close()
	dependency := deps()
	environment := envWithNode(t, server.URL)
	action, _ := dependency.Actions.Action(actionDeployContract)
	ctx := &interp.ActionCtx{Env: environment, Deps: &dependency, Args: map[string]any{"bytecode": "0x6080604052", "save": "contract"}}
	if err := action.Do(context.Background(), ctx); err != nil {
		t.Fatal(err)
	}
	p := ctx.Deployment
	if p == nil || p.Nonce == nil || *p.Nonce != 1 || p.Deployer != deployer || p.ExpectedAddress != address || p.ActualAddress != address || p.ObservationError != "" {
		t.Fatalf("deployment provenance missing or invented: %+v", p)
	}
	bytes, _ := hex.DecodeString("6080604052")
	want := sha256.Sum256(bytes)
	if p.BytecodeSHA256 != hex.EncodeToString(want[:]) || ctx.Receipt["blockNumber"] != "0x3" || ctx.Hash != "0xtxhash" {
		t.Fatal("deployment evidence disconnected from actual transaction")
	}
}

func TestDeploymentUnavailableObservationNeverInventsNonceZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID any `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": nil})
	}))
	defer server.Close()
	ctx := &interp.ActionCtx{Hash: "0xtx", Value: "0xactual"}
	recordContractDeployment(context.Background(), ctx, rpc.Dial(server.URL), "0x6000")
	if ctx.Deployment == nil || ctx.Deployment.Nonce != nil || ctx.Deployment.ObservationError != "transaction_metadata_unavailable" || ctx.Deployment.BytecodeSHA256 == "" {
		t.Fatal("missing observation was represented as a successful nonce-zero deployment")
	}
}
