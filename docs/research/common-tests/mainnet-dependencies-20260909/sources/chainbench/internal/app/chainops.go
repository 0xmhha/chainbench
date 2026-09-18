package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// The on-chain verbs: sending a transaction, deploying and calling a contract,
// funding an account, reading its state.
//
// They live here rather than in each surface because until 2026-09-05 they were
// implemented three times over — once in the CLI, once in the MCP tools, once
// in the DSL's built-in actions — each assembling accounts and core/rpc its own
// way. Three copies of "what --value means" is three chances to disagree, and
// `faucet` reaching the CLI and MCP but never the DSL is what that costs.
//
// These wrap the accounts module and the RPC client; they decide only what
// belongs to the verb itself, such as how long to wait for a receipt.

// ChainRef says which chain's rules apply and where to reach it.
type ChainRef struct {
	// Chain is the registry chain id; empty means stablenet.
	Chain string
	// Manifest selects an external, project-supplied chain instead of an
	// embedded one, and Template is its genesis template.
	Manifest string
	Template string
	// RPC is the node endpoint.
	RPC string
}

// provider resolves the account rules for this reference: an external
// manifest's borrowed protocol when one is named, otherwise the embedded
// chain's.
func (r ChainRef) provider() (accounts.AccountProvider, error) {
	if r.RPC == "" {
		return nil, fmt.Errorf("a node endpoint is required")
	}
	if r.Manifest != "" {
		p, err := ResolveChain(r.Chain, r.Manifest, r.Template)
		if err != nil {
			return nil, err
		}
		return accounts.New(p.Protocol()), nil
	}
	chain := r.Chain
	if chain == "" {
		chain = "stablenet"
	}
	return accounts.ForChain(chain)
}

// TxSendIn is one signed transaction: who sends it, to whom, with what.
type TxSendIn struct {
	Chain ChainRef
	// FromKey is the sender's private key (0x-hex or bare).
	FromKey string
	// To is the recipient or contract address.
	To string
	// Data is calldata (0x-hex); empty for a plain transfer.
	Data string
	// Value is the amount in wei, decimal. Empty is zero.
	Value string
}

// TxSend signs and sends a transaction, returning its hash.
func TxSend(ctx context.Context, _ Deps, in TxSendIn) (string, error) {
	key, err := HexBytes(in.FromKey)
	if err != nil {
		return "", fmt.Errorf("bad sender key: %w", err)
	}
	data, err := HexBytes(in.Data)
	if err != nil {
		return "", fmt.Errorf("bad calldata: %w", err)
	}
	wei, err := Wei(in.Value)
	if err != nil {
		return "", err
	}
	ap, err := in.Chain.provider()
	if err != nil {
		return "", err
	}
	w, err := ap.OpenWallet(ctx, key, in.Chain.RPC)
	if err != nil {
		return "", err
	}
	return w.Execute(ctx, in.To, data, wei)
}

// TxWaitIn asks for a transaction's receipt, waiting for inclusion.
type TxWaitIn struct {
	RPC  string
	Hash string
	// Timeout bounds the wait; zero is 30s.
	Timeout time.Duration
}

// TxReceipt is what a receipt says once a transaction is in a block.
type TxReceipt struct {
	Status      string `json:"status"`
	BlockNumber string `json:"blockNumber"`
	GasUsed     string `json:"gasUsed"`
	Contract    string `json:"contractAddress,omitempty"`
}

// Succeeded reports whether the transaction did what it was sent to do. A
// receipt exists for a reverted transaction too, so its presence is not the
// answer.
func (r TxReceipt) Succeeded() bool { return r.Status == "0x1" }

// TxWait polls until the transaction is in a block, or the wait runs out.
//
// The poll is here rather than in each surface because "how long is long
// enough" is a property of the verb, not of who asked: a CLI that waited 30
// seconds and a tool that waited 5 would disagree about whether the same chain
// works.
func TxWait(ctx context.Context, _ Deps, in TxWaitIn) (TxReceipt, error) {
	if in.RPC == "" || in.Hash == "" {
		return TxReceipt{}, fmt.Errorf("an endpoint and a transaction hash are required")
	}
	timeout := in.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	c := rpc.Dial(in.RPC)
	deadline := time.Now().Add(timeout)
	for {
		raw, err := c.TxReceipt(ctx, in.Hash)
		if err != nil {
			return TxReceipt{}, err
		}
		if raw != nil {
			var r TxReceipt
			if err := json.Unmarshal(raw, &r); err != nil {
				return TxReceipt{}, fmt.Errorf("receipt for %s: %w", in.Hash, err)
			}
			return r, nil
		}
		if time.Now().After(deadline) {
			return TxReceipt{}, fmt.Errorf("timed out after %s waiting for tx %s", timeout, in.Hash)
		}
		select {
		case <-ctx.Done():
			return TxReceipt{}, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// ContractDeployIn is a contract to create from its bytecode.
type ContractDeployIn struct {
	Chain ChainRef
	// FromKey is the deployer's private key.
	FromKey string
	// Bytecode is the creation bytecode (0x-hex).
	Bytecode string
	// Value endows the new contract, in wei, decimal.
	Value string
}

// ContractDeployOut names the transaction and the address it created.
type ContractDeployOut struct {
	Tx      string `json:"tx"`
	Address string `json:"address"`
}

// ContractDeploy creates a contract and reports where it landed.
func ContractDeploy(ctx context.Context, _ Deps, in ContractDeployIn) (ContractDeployOut, error) {
	if in.Bytecode == "" {
		return ContractDeployOut{}, fmt.Errorf("contract creation bytecode is required")
	}
	key, err := HexBytes(in.FromKey)
	if err != nil {
		return ContractDeployOut{}, fmt.Errorf("bad deployer key: %w", err)
	}
	code, err := HexBytes(in.Bytecode)
	if err != nil {
		return ContractDeployOut{}, fmt.Errorf("bad bytecode: %w", err)
	}
	wei, err := Wei(in.Value)
	if err != nil {
		return ContractDeployOut{}, err
	}
	ap, err := in.Chain.provider()
	if err != nil {
		return ContractDeployOut{}, err
	}
	w, err := ap.OpenWallet(ctx, key, in.Chain.RPC)
	if err != nil {
		return ContractDeployOut{}, err
	}
	hash, addr, err := w.Deploy(ctx, code, wei)
	if err != nil {
		return ContractDeployOut{}, err
	}
	return ContractDeployOut{Tx: hash, Address: addr}, nil
}

// ContractCallIn is a read-only call: no key, no transaction, no state change.
type ContractCallIn struct {
	RPC  string
	To   string
	Data string
}

// ContractCall performs an eth_call and returns the 0x-hex result.
func ContractCall(ctx context.Context, _ Deps, in ContractCallIn) (string, error) {
	if in.RPC == "" || in.To == "" {
		return "", fmt.Errorf("an endpoint and a contract address are required")
	}
	return rpc.Dial(in.RPC).EthCall(ctx, in.To, in.Data)
}

// FaucetIn funds an address from a key that already holds a balance.
type FaucetIn struct {
	Chain ChainRef
	// FromKey holds the funds — typically a genesis-allocated identity.
	FromKey string
	// To is the address to fund; Amount is in wei, decimal.
	To     string
	Amount string
}

// Faucet sends funds and returns the transaction hash.
func Faucet(ctx context.Context, _ Deps, in FaucetIn) (string, error) {
	if in.To == "" {
		return "", fmt.Errorf("a recipient address is required")
	}
	key, err := HexBytes(in.FromKey)
	if err != nil {
		return "", fmt.Errorf("bad funding key: %w", err)
	}
	amount, err := Wei(in.Amount)
	if err != nil {
		return "", err
	}
	if amount.Sign() <= 0 {
		return "", fmt.Errorf("an amount above zero is required")
	}
	ap, err := in.Chain.provider()
	if err != nil {
		return "", err
	}
	return ap.Faucet(ctx, key, in.To, amount, in.Chain.RPC)
}

// FaucetKeyIn funds an address from a key the caller already resolved, which is
// how a surface funds from a key set entry rather than an inline hex key.
type FaucetKeyIn struct {
	Chain  ChainRef
	Key    PrivateKey
	To     string
	Amount string
}

// FaucetFromKey is Faucet with the key already in hand.
func FaucetFromKey(ctx context.Context, d Deps, in FaucetKeyIn) (string, error) {
	return Faucet(ctx, d, FaucetIn{
		Chain: in.Chain, FromKey: hex.EncodeToString(in.Key.Bytes()),
		To: in.To, Amount: in.Amount,
	})
}

// AccountStateIn asks what the chain knows about an address.
type AccountStateIn struct {
	RPC     string
	Address string
}

// AccountStateOut is an account as the chain holds it.
type AccountStateOut struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
	Nonce   uint64 `json:"nonce"`
	// Contract is true when the address holds code.
	Contract bool `json:"contract"`
}

// AccountState reads an address's balance, nonce, and whether it holds code.
func AccountState(ctx context.Context, _ Deps, in AccountStateIn) (AccountStateOut, error) {
	if in.RPC == "" || in.Address == "" {
		return AccountStateOut{}, fmt.Errorf("an endpoint and an address are required")
	}
	c := rpc.Dial(in.RPC)
	bal, err := c.BalanceAt(ctx, in.Address)
	if err != nil {
		return AccountStateOut{}, err
	}
	nonce, err := c.NonceAt(ctx, in.Address)
	if err != nil {
		return AccountStateOut{}, err
	}
	code, err := c.CodeAt(ctx, in.Address)
	if err != nil {
		return AccountStateOut{}, err
	}
	return AccountStateOut{
		Address: in.Address, Balance: bal.String(), Nonce: nonce,
		Contract: hasCode(code),
	}, nil
}

// hasCode reports whether an eth_getCode result is a real contract. An account
// with no code answers with several spellings of nothing, and reading any of
// them as code would call every plain account a contract.
func hasCode(code string) bool {
	switch code {
	case "", "0x", "0x0":
		return false
	}
	return true
}

// HexBytes decodes a 0x-prefixed or bare hex string; empty yields no bytes.
//
// It is exported because every surface takes hex from an operator and they must
// all read it the same way — one accepting a bare string while another demands
// the prefix is a difference nobody would think to test.
func HexBytes(s string) ([]byte, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	if s == "" {
		return nil, nil
	}
	return hex.DecodeString(s)
}

// Wei parses a decimal wei amount; empty is zero.
//
// Decimal, not hex: an amount silently read in the other base moves a different
// sum than the operator asked for.
func Wei(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0), nil
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("bad amount %q (decimal wei expected)", s)
	}
	return v, nil
}
