package poa

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/registry"
)

// Reading the validator set of a running wemix (poa) chain over RPC.
//
// A wemix chain keeps its producers in a governance contract the first node
// deploys, not in a JSON-RPC method — there is no <ns>_getValidators. So the
// set is read the way the chain itself reads it: find the governance contract
// (admin.wemixInfo reports its address), then eth_call its member enumeration.
// Everything here is plain HTTP RPC, so a node on another machine (docker,
// remote) is reached the same way a local one is — no IPC attach.

const (
	// govGetMemberLength is the selector for Gov.getMemberLength() -> uint256.
	govGetMemberLength = "0xd965ea00"
	// govGetMember is the selector for Gov.getMember(uint256) -> address. The
	// governance member index is 1-based; index 0 is the zero address.
	govGetMember = "0xab3545e5"
	zeroAddr40   = "0000000000000000000000000000000000000000"
	// maxGovMembers bounds how many members this will enumerate. A wemix
	// producer set is tens of nodes; a reply above this is malformed or hostile,
	// and following it would mean one eth_call per claimed member.
	maxGovMembers = 1000
)

// RuntimeValidators asks a running wemix node which validators its governance
// recognizes, satisfying registry.RuntimeValidatorReader. It returns the member
// addresses (0x-hex), which are exactly the addresses the producer node keys
// must derive to sign — a mismatch is why a wrong key set stalls the chain.
func (Family) RuntimeValidators(ctx context.Context, c registry.Caller) ([]string, error) {
	gov, err := governanceAddress(ctx, c)
	if err != nil {
		return nil, err
	}
	lenHex, err := ethCall(ctx, c, gov, govGetMemberLength)
	if err != nil {
		return nil, fmt.Errorf("poa: getMemberLength: %w", err)
	}
	count, err := parseMemberCount(lenHex)
	if err != nil {
		return nil, fmt.Errorf("poa: getMemberLength: %w", err)
	}
	out := make([]string, 0, count)
	for i := int64(1); i <= int64(count); i++ { // members are 1-based
		res, err := ethCall(ctx, c, gov, govGetMember+pad32(i))
		if err != nil {
			return nil, fmt.Errorf("poa: getMember(%d): %w", i, err)
		}
		addr := lastAddress(res)
		if addr == "0x"+zeroAddr40 {
			continue
		}
		out = append(out, addr)
	}
	return out, nil
}

// parseMemberCount reads an ABI uint256 member count out of an eth_call reply.
//
// The value comes from whatever the node answers, so it is a trust boundary,
// not a fact. It is validated before anything is sized from it: big.Int accepts
// a leading sign, so "0x-1" used to parse as -1 and reach make([]string, 0, -1),
// and a full-width uint256 truncates to a negative int64 the same way — both
// panic in makeslice. A merely huge value is no better: the enumeration below is
// one eth_call per member, so an unbounded count is an RPC flood.
func parseMemberCount(raw string) (int, error) {
	h := strings.TrimPrefix(strings.TrimSpace(raw), "0x")
	if h == "" {
		return 0, fmt.Errorf("reply %q carries no value", raw)
	}
	// An ABI word is 32 bytes; anything longer is not one. Digits are checked
	// explicitly because big.Int would happily accept "-1" or "+1".
	if len(h) > 64 {
		return 0, fmt.Errorf("reply %q is longer than a uint256 word", raw)
	}
	for _, c := range h {
		if !isHexDigit(c) {
			return 0, fmt.Errorf("reply %q is not a hex uint256", raw)
		}
	}
	n, ok := new(big.Int).SetString(h, 16)
	if !ok {
		return 0, fmt.Errorf("reply %q is not a hex uint256", raw)
	}
	if !n.IsInt64() || n.Int64() > maxGovMembers {
		return 0, fmt.Errorf("reply %q claims more than %d governance members", raw, maxGovMembers)
	}
	return int(n.Int64()), nil
}

// isHexDigit reports whether c is a hex digit. It is spelled out rather than
// left to big.Int, which accepts a leading sign an ABI word never carries.
func isHexDigit(c rune) bool {
	switch {
	case '0' <= c && c <= '9', 'a' <= c && c <= 'f', 'A' <= c && c <= 'F':
		return true
	default:
		return false
	}
}

// governanceAddress reads the governance contract's address from admin.wemixInfo
// (an admin-namespace RPC that answers over HTTP once governance is deployed).
func governanceAddress(ctx context.Context, c registry.Caller) (string, error) {
	var info struct {
		Governance string `json:"governance"`
	}
	if err := c.Call(ctx, "admin_wemixInfo", &info); err != nil {
		return "", fmt.Errorf("poa: admin_wemixInfo: %w", err)
	}
	if info.Governance == "" || info.Governance == "0x"+zeroAddr40 {
		return "", fmt.Errorf("poa: governance is not deployed yet (admin_wemixInfo reports none) — the chain is not ready to check")
	}
	return info.Governance, nil
}

// ethCall makes a read-only call to a contract and returns the 0x-hex result.
func ethCall(ctx context.Context, c registry.Caller, to, data string) (string, error) {
	var out string
	err := c.Call(ctx, "eth_call", &out, map[string]string{"to": to, "data": data}, "latest")
	return out, err
}

// pad32 renders a non-negative int as a 32-byte (64-hex) big-endian word.
func pad32(i int64) string {
	return fmt.Sprintf("%064x", i)
}

// lastAddress takes the trailing 20 bytes of an ABI word as a 0x-hex address.
func lastAddress(word string) string {
	h := strings.TrimPrefix(word, "0x")
	if len(h) < 40 {
		return "0x" + zeroAddr40
	}
	return "0x" + h[len(h)-40:]
}
