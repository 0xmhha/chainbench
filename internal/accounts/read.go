package accounts

import (
	"context"
	"fmt"
	"math/big"
)

// EthCaller performs an eth_call to `to` with 0x-hex calldata, returning the
// 0x-hex result. rpc.Client.EthCall satisfies it directly, so a case can pass
// `rpc.Dial(url).EthCall`.
type EthCaller func(ctx context.Context, to, data string) (string, error)

// ReadUint calls to.methodSig(args...) via eth_call and decodes the returned
// 32-byte word as a uint256. args are raw big-endian words (an address from
// AddressArg, or a uint's minimal bytes). It composes EncodeCall + WordToBig so
// contract-read cases do not repeat the encode/decode boilerplate.
func ReadUint(ctx context.Context, call EthCaller, to, methodSig string, args ...[]byte) (*big.Int, error) {
	ret, err := call(ctx, to, EncodeCall(methodSig, args...))
	if err != nil {
		return nil, err
	}
	v, ok := WordToBig(ret)
	if !ok {
		return nil, fmt.Errorf("accounts: %s returned an undecodable word %q", methodSig, ret)
	}
	return v, nil
}
