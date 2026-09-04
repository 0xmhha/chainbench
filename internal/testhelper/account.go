package testhelper

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// FaucetLabel is the reserved name for the network's funding account.
//
// It is a name for a decision, not a new identity: the account that starts with
// a genesis balance is the first producer's, so that is what the label resolves
// to. Tests refer to the role rather than to whichever address happens to hold
// the money, which is the whole reason labels exist — the address changes with
// the key set, the role does not.
const FaucetLabel = "faucet"

// nodeLabel matches the identities a chain runs on: node1, node2, ...
var nodeLabel = regexp.MustCompile(`^node[1-9][0-9]*$`)

// addressLiteral matches a spec writing an address rather than a name.
//
// The test is the prefix, not the length. A label is a name — node1, dev1 —
// and no name starts with 0x, so the prefix settles which of the two a caller
// meant without needing either to be declared. Whether the hex that follows is
// a well-formed address is the node's judgement, not this package's: pretending
// to validate it here would produce a second, weaker answer to a question the
// chain already answers exactly.
var addressLiteral = regexp.MustCompile(`^0[xX]`)

// Account is what a label names: an address, and how a transaction from it is
// signed.
//
// The signing half is not an implementation detail the caller can ignore,
// because the two kinds of account are signed in different places. A node's
// account is unlocked in that node's keystore and the node signs for it; an
// account that only this harness holds has to be signed here and submitted as
// a raw transaction. A label that answered only "what address" would leave
// every caller to work that out, and they would not all reach the same answer.
type Account struct {
	// Label is the name that was resolved, or empty for a literal address.
	Label string
	// Address is the 0x-prefixed account address.
	Address string
	// Key is the private key, present only when this harness signs for the
	// account. A node's account is signed by the node and carries none here.
	Key []byte
}

// SignsLocally reports whether a transaction from this account is signed here
// and submitted raw, rather than handed to a node to sign.
func (a Account) SignsLocally() bool { return len(a.Key) > 0 }

// ResolveAccount turns what a spec wrote in an address position into an
// account.
//
// Three forms, told apart by shape so nothing has to be declared:
//
//	0x...       an address written out; no key, and no claim about who holds it
//	node<N>     a node's own account, signed by that node
//	<label>     any other identity in the key set, signed here
//
// A name that is not in the key set is an error that lists what is, never a
// silent fall-through to the zero address: a typo has to fail the step rather
// than send value nowhere. This follows the rule bindings already keep for
// "$name".
func ResolveAccount(d *interp.Deps, ref string) (Account, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Account{}, fmt.Errorf("dsl: an account is required (a label such as node1 or dev1, or a 0x address)")
	}
	if addressLiteral.MatchString(ref) {
		return Account{Address: ref}, nil
	}
	if d == nil || d.Keys == nil {
		return Account{}, fmt.Errorf("dsl: %q is a label but this run has no key set to resolve it against", ref)
	}
	label := ref
	if label == FaucetLabel {
		label = "node1"
	}
	entry, ok := d.Keys.Get(keyring.Label(label))
	if !ok {
		return Account{}, fmt.Errorf("dsl: unknown account %q; the key set holds %s",
			ref, strings.Join(knownLabels(d), ", "))
	}
	acct := Account{Label: ref, Address: entry.Address}
	if !nodeLabel.MatchString(label) {
		// Only this harness holds the key, so only this harness can sign for it.
		acct.Key = entry.Nodekey.Bytes()
	}
	return acct, nil
}

// knownLabels lists the key set's labels for an error message, with the
// reserved name included so a reader learns it exists at the moment they need
// it.
func knownLabels(d *interp.Deps) []string {
	labels := d.Keys.Labels()
	out := make([]string, 0, len(labels)+1)
	for _, l := range labels {
		out = append(out, string(l))
	}
	sort.Strings(out)
	if _, ok := d.Keys.Get("node1"); ok {
		out = append(out, FaucetLabel+" (= node1)")
	}
	return out
}

// addressArgs are the argument names that hold an account address, and so the
// places a label may stand in for one.
//
// The list lives here rather than in the interpreter because these are chain
// words: the grammar knows a step has arguments, not that "deployer" is an
// account. Naming them in one place is what keeps a label usable everywhere an
// address is — a label that worked in "from" but not in "address" would be a
// half-usable feature, and a spec would go back to pasting hex for the half it
// could not name.
var addressArgs = []string{"address", "from", "to", "deployer", "funder"}

// resolveAddressArgs returns spec with every address-shaped argument resolved,
// leaving everything else untouched. The input map is not modified: a spec is
// read more than once (an assertion runs against each target node), and
// rewriting it in place would resolve against a spec that had already changed.
func resolveAddressArgs(d *interp.Deps, spec map[string]any) (map[string]any, error) {
	var out map[string]any
	for _, key := range addressArgs {
		ref, ok := spec[key].(string)
		if !ok || ref == "" || addressLiteral.MatchString(ref) {
			continue
		}
		acct, err := ResolveAccount(d, ref)
		if err != nil {
			return nil, err
		}
		if out == nil {
			out = make(map[string]any, len(spec))
			for k, v := range spec {
				out[k] = v
			}
		}
		out[key] = acct.Address
	}
	if out == nil {
		return spec, nil
	}
	return out, nil
}
