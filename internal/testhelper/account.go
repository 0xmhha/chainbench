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
		// A contract name reaching an account position is a wiring mistake, not
		// a typo, and the two need different fixes. Saying "unknown account"
		// about a name the chain does declare sends the reader to look for a
		// missing key; this says which position accepted it and which did not.
		if _, isContract := d.Contracts[ref]; isContract {
			return Account{}, fmt.Errorf("dsl: %q is one of this chain's contracts, not an account — it has no key, so it cannot stand where one signs", ref)
		}
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

// ResolveAddress turns what a spec wrote in an address position into an
// address, from either of the two things that have one.
//
//	0x...       an address written out
//	<label>     an account in the key set
//	<name>      a contract the chain declares
//
// Contracts are named rather than written out because the address does not
// identify one: 0x…1001 is govValidator on stablenet and govStaking on wbft, so
// a spec that writes it calls a different contract the moment it runs anywhere
// else. Naming it makes that a refusal instead.
//
// An account wins a name collision. Key-set labels are chosen per run and a
// contract table is fixed by the chain, so a run that introduces a clash meant
// the thing it just created.
func ResolveAddress(d *interp.Deps, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("dsl: an address is required (an account label, a contract name, or a 0x address)")
	}
	if addressLiteral.MatchString(ref) {
		return ref, nil
	}
	if d != nil && d.Keys != nil {
		if _, ok := d.Keys.Get(keyring.Label(ref)); ok || ref == FaucetLabel {
			acct, err := ResolveAccount(d, ref)
			return acct.Address, err
		}
	}
	if d != nil {
		if addr, ok := d.Contracts[ref]; ok {
			return addr, nil
		}
	}
	return "", fmt.Errorf("dsl: unknown address %q; the key set holds %s, and this chain declares %s",
		ref, strings.Join(knownLabels(d), ", "), knownContracts(d))
}

// knownContracts lists the chain's contract names for an error message, so a
// reader learns whether the chain declares none at all or just not this one.
func knownContracts(d *interp.Deps) string {
	if d == nil || len(d.Contracts) == 0 {
		return "no contracts (it deploys them at run time, or this build does not know it)"
	}
	names := make([]string, 0, len(d.Contracts))
	for n := range d.Contracts {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// knownLabels lists the key set's labels for an error message, with the
// reserved name included so a reader learns it exists at the moment they need
// it.
func knownLabels(d *interp.Deps) []string {
	// A run with no key set is ordinary (attach mode, a unit test); building the
	// error message must not be the thing that crashes. ResolveAddress already
	// guards the lookup itself, so only this half was left dereferencing.
	if d == nil || d.Keys == nil {
		return []string{"no key set for this run"}
	}
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
var addressArgs = []string{"address", "to"}

// signerArgs are the argument names that hold an account that will SIGN.
//
// They resolve against the key set only. A contract has no key, so letting a
// contract name stand here would hand a node an address it cannot sign for and
// the failure would come back as "unknown account" from the node, about the
// address rather than about the name that produced it.
var signerArgs = []string{"from", "deployer", "funder"}

// addressListArgs are the arguments that hold a LIST of values, some of which
// may be addresses.
//
// "of" is the readers' value list: derive sums numbers with it, packs ABI words
// with it, and slices a hex blob with it. An address in that list is packed as a
// 32-byte word, which is the one place a spec still had to paste hex to say
// "this account" — the selector takes an address, and the label mechanism
// stopped at the argument names above.
//
// An element that does not resolve is left exactly as written. The list
// legitimately carries numbers, hex blobs and already-substituted bindings, and
// whatever consumes it reports a bad element with the message that knows what it
// expected. Resolving here can only turn what was an error into an address.
var addressListArgs = []string{"of"}

// valueArgs are the arguments that hold a value a spec COMPARES against, and
// the raw JSON-RPC argument list.
//
// They take the same rule as "of" — resolve a name the key set knows, leave
// everything else exactly as written — for the same reason: they legitimately
// carry numbers, hex, booleans, block tags and already-substituted bindings, so
// a failure to resolve is not a mistake here.
//
// It is the rule and not the position that does the work. A raw params list
// puts the address wherever the method wants it (eth_getBalance first,
// eth_createAccessList inside the transaction object), so nothing here knows
// which slot to look at — and nothing has to. "node1" cannot be a block tag, a
// quantity or a hash, so a value that resolves was meant as an account.
//
// Walking into nested lists and objects is what reaches the transaction object
// eth_createAccessList takes.
var valueArgs = []string{"expected", "params"}

// resolveAddressArgs returns spec with every address-shaped argument resolved,
// leaving everything else untouched. The input map is not modified: a spec is
// read more than once (an assertion runs against each target node), and
// rewriting it in place would resolve against a spec that had already changed.
func resolveAddressArgs(d *interp.Deps, spec map[string]any) (map[string]any, error) {
	var out map[string]any
	copyOnce := func() {
		if out != nil {
			return
		}
		out = make(map[string]any, len(spec))
		for k, v := range spec {
			out[k] = v
		}
	}
	for _, key := range signerArgs {
		ref, ok := spec[key].(string)
		if !ok || ref == "" || addressLiteral.MatchString(ref) {
			continue
		}
		acct, err := ResolveAccount(d, ref)
		if err != nil {
			return nil, err
		}
		copyOnce()
		out[key] = acct.Address
	}
	for _, key := range addressArgs {
		ref, ok := spec[key].(string)
		if !ok || ref == "" || addressLiteral.MatchString(ref) {
			continue
		}
		addr, err := ResolveAddress(d, ref)
		if err != nil {
			return nil, err
		}
		copyOnce()
		out[key] = addr
	}
	for _, key := range append(addressListArgs, valueArgs...) {
		v, ok := spec[key]
		if !ok {
			continue
		}
		resolved, changed := resolveNames(d, v)
		if !changed {
			continue
		}
		copyOnce()
		out[key] = resolved
	}
	if out == nil {
		return spec, nil
	}
	return out, nil
}

// resolveNames returns v with every name the key set or the chain's contract
// table knows replaced by its address, reporting whether anything changed so an
// unchanged value is not copied.
//
// It walks lists and objects because the address is not always at the top: a
// params list holds it where the method wants it, and eth_createAccessList puts
// it inside a transaction object. Nothing here decides which position is an
// address — only whether the string resolves.
//
// A failure to resolve is not an error: see addressListArgs and valueArgs. The
// input is never modified; a spec is read more than once.
func resolveNames(d *interp.Deps, v any) (any, bool) {
	switch t := v.(type) {
	case string:
		if t == "" || addressLiteral.MatchString(t) {
			return v, false
		}
		addr, err := ResolveAddress(d, t)
		if err != nil {
			return v, false
		}
		return addr, true
	case []any:
		var out []any
		for i, e := range t {
			r, changed := resolveNames(d, e)
			if !changed {
				continue
			}
			if out == nil {
				out = append(out, t...)
			}
			out[i] = r
		}
		if out == nil {
			return v, false
		}
		return out, true
	case map[string]any:
		var out map[string]any
		for k, e := range t {
			r, changed := resolveNames(d, e)
			if !changed {
				continue
			}
			if out == nil {
				out = make(map[string]any, len(t))
				for kk, vv := range t {
					out[kk] = vv
				}
			}
			out[k] = r
		}
		if out == nil {
			return v, false
		}
		return out, true
	default:
		return v, false
	}
}
