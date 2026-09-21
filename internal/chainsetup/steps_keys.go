package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/preset"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/core/blueprint"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// The keys step: the key set a composition runs on.
//
// It resolves what the declaration asked for — a preset used as-is, or a set
// generated here — and checks the node table's key references before anything
// is written, because a reference that only fails at launch has already cost a
// provisioning round trip.

// The kinds of failure this step has. They are what a caller branches on: the
// messages below name the node, the file and what to do about it, which is what
// makes them worth reading, and none of that is something a caller can switch
// on. A kind added in front of the message would say the same thing twice, so
// the kind rides alongside the message instead.
var (
	// errKeySourceUnknown: the source named is not one of the three.
	errKeySourceUnknown = errors.New("key source unknown")
	// errKeyCountShort: the declaration holds fewer identities than the
	// network has nodes.
	errKeyCountShort = errors.New("declared identities are fewer than the nodes")
	// errKeyRefNotLocal: a node named its key as something other than a local
	// file — inline material, or a path on another machine.
	errKeyRefNotLocal = errors.New("node key reference is not a local file")
	// errKeyUnreadable: the local file a node named cannot be read as a key.
	errKeyUnreadable = errors.New("node key file cannot be read as a key")
)

// keyFail is an error of this step with its kind attached.
//
// Error is the message the step already wrote, so nothing a person reads
// changes. Unwrap reports both, so errors.Is finds the kind and the cause.
type keyFail struct {
	kind error
	err  error
}

func (f keyFail) Error() string   { return f.err.Error() }
func (f keyFail) Unwrap() []error { return []error{f.kind, f.err} }

// ofKind marks err as being of this kind. A nil err stays nil so it can wrap a
// return site without a branch around it.
func ofKind(kind error, err error) error {
	if err == nil {
		return nil
	}
	return keyFail{kind: kind, err: err}
}

// KeysDone is what the keys step did: the line it recorded, and which of the
// three sources the identities actually came from.
//
// The source is reported rather than inferred from the request because the
// request does not decide it on its own. A node table that names per-node keys
// is the source whatever the request said, and a handler reading the request
// would call that a preset.
type KeysDone struct {
	Detail string
	Source lifecycle.Status
}

func (w *Workspace) plugin() (registry.ChainPlugin, error) {
	if w.state.Chain == "" && w.state.ManifestPath == "" {
		return nil, fmt.Errorf("chainsetup: no chain set — run `chain new` first")
	}
	return external.ResolveChain(w.state.Chain, w.state.ManifestPath, w.state.TemplatePath)
}

// KeysOpts selects where node identities come from (algorithm steps 2-3).
type KeysOpts struct {
	// Source is "keyPreset" (default), "generate", or "declared".
	Source string
	// Blueprint is the declaration the keys come from when Source is
	// "declared". Its nodes carry their own nodekeys, which is what lets a
	// network be composed with no preset directory anywhere (N3).
	Blueprint *blueprint.Blueprint
	// Nodes is how many identities the set must cover; <=0 uses the node table
	// length, falling back to the validator count.
	Nodes int
	// Validators is how many identities join the validator set (generate only);
	// <=0 means all.
	Validators int
}

// Keys ensures the workspace's key set exists and covers the requested node
// count, through the same KeySource boundary `chainbench run` uses.
func (w *Workspace) Keys(ctx context.Context, opts KeysOpts) (KeysDone, error) {
	if _, err := w.plugin(); err != nil {
		return KeysDone{}, err
	}
	n := opts.Nodes
	if n <= 0 {
		n = len(w.state.Nodes)
	}
	if n <= 0 {
		n = opts.Validators
	}
	if n <= 0 {
		return KeysDone{}, fmt.Errorf("chainsetup: keys: node count unknown — run `chain place` first or pass --nodes")
	}
	// A key set named on a server is downloaded to a local directory first, so
	// the rest of this step reads it the one local way.
	if err := w.materializeKeyring(ctx); err != nil {
		return KeysDone{}, err
	}

	// A node table that names any per-node key builds the set from the table:
	// declared where a key is given, generated where not. This fixes each
	// producer's genesis validator address to its key, and hands a non-producer
	// its nodekey and enode without making it a validator. It takes precedence
	// over the source string because a table that declares keys has said where
	// its identities come from.
	var src store.KeySource
	// from is the state this step ends in, set where the source is chosen so
	// the two cannot drift apart.
	var from lifecycle.Status
	if set, pinned, keyed, kerr := w.nodeTableKeys(ctx, n); kerr != nil {
		return KeysDone{}, kerr
	} else if keyed {
		src = store.DeclaredKeys{Path: w.state.KeysDir, Set: set, Pinned: pinned}
		from = lifecycle.ChainEnsureKeysFromBlueprint
	} else {
		switch opts.Source {
		case "", "keyPreset":
			src = store.PresetKeys{Path: w.state.KeysDir}
			from = lifecycle.ChainEnsureKeysFromPreset
		case "declared":
			// The declaration is the origin, and the ring is materialised from it
			// so that the genesis source, the launcher and provision keep reading
			// keys the one way they already do.
			if opts.Blueprint == nil {
				return KeysDone{}, ofKind(errKeySourceUnknown,
					fmt.Errorf("chainsetup: keys: source %q needs a blueprint to take the keys from", opts.Source))
			}
			set, err := w.declaredKeys(*opts.Blueprint, n)
			if err != nil {
				return KeysDone{}, err
			}
			// A blueprint names a key for every node it declares (a node without
			// one is an error there), so every index is pinned: if the ring on
			// disk holds a different identity, the document and the disk
			// disagree and the operator has to say which is right.
			pinned := make([]int, 0, len(set.Nodes))
			for _, e := range set.Nodes {
				pinned = append(pinned, e.Index)
			}
			sort.Ints(pinned)
			src = store.DeclaredKeys{Path: w.state.KeysDir, Set: set, Pinned: pinned}
			from = lifecycle.ChainEnsureKeysFromBlueprint
		case "generate":
			// A generated set must declare exactly the topology's validators, not
			// make every node one: a network with endpoints (4 bp + 11 en) whose key
			// set claims 15 validators fails genesis, where the governance contract
			// requires members and validators to match. The allocated count is the
			// authority; an explicit opts.Validators still wins.
			validators := opts.Validators
			if validators <= 0 {
				validators = w.state.BPCount
			}
			src = store.GeneratedKeys{Path: w.state.KeysDir, Validators: validators}
			from = lifecycle.ChainEnsureKeysGenerated
		default:
			return KeysDone{}, ofKind(errKeySourceUnknown,
				fmt.Errorf("chainsetup: keys: unknown source %q (want keyPreset, generate or declared)", opts.Source))
		}
	}
	ks, err := src.Ensure(ctx, n)
	if err != nil {
		return KeysDone{}, err
	}
	detail := fmt.Sprintf("%s: %d identities, %d declared validators",
		src.Describe(), len(ks.Nodes), len(ks.Network.Validators))
	w.markStep("keys", detail)
	return KeysDone{Detail: detail, Source: from}, nil
}

// declaredKeys derives the ring a blueprint declares, for the network the
// placement has already decided.
//
// It resolves against the node table this workspace allocated rather than
// against the document alone: the identity that matters is the node's index,
// which is what its datadir, its keyring entry and its enode are all named
// from. Deriving against a different table would produce keys that are correct
// in isolation and attached to the wrong nodes.
func (w *Workspace) declaredKeys(bp blueprint.Blueprint, n int) (preset.Key, error) {
	placed, err := w.Netmap()
	if err != nil {
		return preset.Key{}, fmt.Errorf("chainsetup: keys: %w — run `chain place` first", err)
	}
	r, err := blueprint.Resolve(bp, blueprint.Inputs{
		Placed: placed.Placements(),
		Chain:  blueprint.ChainFacts{ID: w.state.Chain, Binary: w.state.Binary},
		Layout: node.Layout{Root: w.state.Target.DataRoot},
	})
	if err != nil {
		return preset.Key{}, err
	}
	// BLS material is derived for every family, which is what the generated
	// source already does. Only wbft reads it, and asking the family instead
	// would be the better answer, but there is no method that says so today and
	// inventing one here would put the question in two places. Recorded as N3
	// debt rather than guessed at.
	set, err := blueprint.PresetFrom(r, derive.WithBLS, localKeyReader)
	if err != nil {
		return preset.Key{}, err
	}
	if len(set.Nodes) < n {
		return preset.Key{}, ofKind(errKeyCountShort,
			fmt.Errorf("chainsetup: keys: the blueprint declares %d identities and the network has %d nodes", len(set.Nodes), n))
	}
	return set, nil
}

// nodeTableKeys builds the key set a node table with per-node keys asks for:
// a node that names a key uses it (case a), and a node that names none is
// generated (case b). A producer's key fixes its genesis validator address; a
// non-producer takes the key as its nodekey and enode without becoming a
// validator (case c). The second result is false when no node names a key, so
// the caller falls back to the source string.
//
// The generated identities come from store.Generate — the one module that
// creates keys, with the entropy, BLS derivation, keystores and password the
// rest of the system expects — rather than being hand-rolled here. Only their
// key material is taken; DeclaredKeys re-writes the ring (keystores, password,
// metadata) at the workspace's key dir, the way the declared source already does.
func (w *Workspace) nodeTableKeys(ctx context.Context, n int) (preset.Key, []int, bool, error) {
	byIndex := make(map[int]node.Record, len(w.state.Nodes))
	// pinned are the nodes whose key the table actually names. They are the only
	// ones an existing ring can contradict — the rest are filled with fresh
	// entropy, so they differ on every call by construction.
	var pinned []int
	for _, r := range w.state.Nodes {
		byIndex[r.Index] = r
		if r.Key != "" {
			pinned = append(pinned, r.Index)
		}
	}
	if len(pinned) == 0 {
		return preset.Key{}, nil, false, nil
	}
	sort.Ints(pinned)

	// Generate a full set once, to a throwaway dir, for the entropy of the nodes
	// that name no key. Its keystores are not reused — DeclaredKeys re-writes
	// them from the key material below — so the dir is temporary.
	tmp, err := os.MkdirTemp("", "cb-nodekeys-")
	if err != nil {
		return preset.Key{}, nil, true, fmt.Errorf("chainsetup: keys: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	gen, err := store.GenerateAt(ctx, store.GenerateOpts{Nodes: n, Out: tmp, Derive: derive.WithBLS}, nil)
	if err != nil {
		return preset.Key{}, nil, true, fmt.Errorf("chainsetup: keys: generate node identities: %w", err)
	}

	var set preset.Key
	for i := 1; i <= n; i++ {
		r, ok := byIndex[i]
		if !ok {
			return preset.Key{}, nil, true, fmt.Errorf("chainsetup: keys: node table has no node%d", i)
		}
		var key derive.PrivateKey
		if r.Key != "" {
			// parseNodeKey names the node itself: its refusal for inline key
			// material must not be wrapped in anything that quotes the value.
			key, err = parseNodeKey(i, r.Key)
			if err != nil {
				return preset.Key{}, nil, true, err
			}
		} else {
			key = gen.Nodes[i-1].Nodekey
		}
		id, derr := derive.Derive(key, derive.WithBLS)
		if derr != nil {
			return preset.Key{}, nil, true, fmt.Errorf("chainsetup: keys: node%d: %w", i, derr)
		}
		set.Nodes = append(set.Nodes, keyring.Entry{
			Label:    keyring.Label(node.LabelFor(i)),
			Index:    i,
			Nodekey:  key,
			Identity: id,
		})
	}
	// The validator set is the producers, in index order — the same rule the
	// genesis source applies. A non-producer holds a key but is not listed.
	for i := 1; i <= n; i++ {
		if node.Is(node.Role(byIndex[i].Role), node.RoleBP) {
			set.Network.Validators = append(set.Network.Validators, set.Nodes[i-1].Address)
		}
	}
	return set, pinned, true, nil
}

// parseNodeKey reads a node's declared key from the local file the node table
// names. Inline key material is refused — see the reasoning on the function.
// localKeyReader is the file reader a blueprint's declared keys are read
// through. Like parseNodeKey, it refuses a server reference: a private key a
// blueprint names is a local file or inline hex, never a secret pulled off a
// server onto this machine.
func localKeyReader(path string) ([]byte, error) {
	if strings.HasPrefix(path, "srv://") {
		return nil, fmt.Errorf("key reference %q is on a server: a private key is not read across machines — use inline hex or a local key file", path)
	}
	return os.ReadFile(path)
}

// checkNodeKeyRef decides whether a node table's key reference may be carried
// any further, and says why not without ever quoting key material.
//
// The distinction it draws is between a reference and a secret. A server path
// and a mistyped file path are both references: naming them is what tells the
// operator which line to fix, and neither is a secret. Key material is not, so
// a refusal that echoed it would move the key from the document into the error
// — and from there into stderr, the session record and the --json report, which
// is the leak the whole rule exists to prevent. The node is named by its index
// instead; the operator has the document.
func checkNodeKeyRef(index int, ref string) error {
	if ref == "" {
		return nil
	}
	if looksLikeKeyMaterial(ref) {
		return ofKind(errKeyRefNotLocal, fmt.Errorf(
			"chainsetup: keys: node%d declares a private key inline: a node key is named by a local file path, never written inline — an inline key would be stored in this workspace's state in cleartext (the value is withheld here for the same reason)",
			index))
	}
	if strings.HasPrefix(ref, "srv://") {
		return ofKind(errKeyRefNotLocal,
			fmt.Errorf("chainsetup: keys: node%d: key reference %q is on a server: a private key is not read across machines — point it at a local key file", index, ref))
	}
	if _, err := os.Stat(ref); err != nil {
		return ofKind(errKeyRefNotLocal,
			fmt.Errorf("chainsetup: keys: node%d: key reference %q is not a readable file: a node key is named by a local file path", index, ref))
	}
	return nil
}

// checkTopologyKeyRefs applies checkNodeKeyRef to every node a topology
// declares. A nil topology declares nothing.
func checkTopologyKeyRefs(t *node.Topology) error {
	if t == nil {
		return nil
	}
	for _, n := range t.Sorted() {
		if err := checkNodeKeyRef(n.Index, n.Key); err != nil {
			return err
		}
	}
	return nil
}

// looksLikeKeyMaterial reports whether a key reference is the key itself rather
// than a path to one.
//
// A secp256k1 private key is 32 bytes, written as 64 hex digits with or without
// the 0x prefix. The test is deliberately wider than that — any run of hex at
// least 32 digits long — because the cost of the two mistakes is not symmetric:
// treating a path as a key withholds it from one message, while treating a key
// as a path prints it. A real path carries a separator or a dot long before it
// carries thirty-two hex digits and nothing else.
func looksLikeKeyMaterial(ref string) bool {
	h := strings.TrimSpace(ref)
	if strings.HasPrefix(h, "0x") || strings.HasPrefix(h, "0X") {
		h = h[2:]
	}
	if len(h) < 32 {
		return false
	}
	for _, c := range h {
		switch {
		case '0' <= c && c <= '9', 'a' <= c && c <= 'f', 'A' <= c && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func parseNodeKey(index int, ref string) (derive.PrivateKey, error) {
	// A node table's key is a local FILE, and nothing else.
	//
	// Not a server reference: reading a secret off a server onto this machine is
	// what the key contract forbids (a prepared key is verified on its own
	// server, and only its public identity leaves it).
	//
	// And not inline hex either. The node table is copied into the workspace's
	// own state — node records and the recorded request both keep this string —
	// and that state is ordinary JSON a person reads, diffs and copies. A key
	// written inline would sit there in cleartext, which is the same reason the
	// blueprint generator references keys by path and never inlines them
	// (blueprint.FromPreset). A path in state names a secret; it is not one.
	//
	// place already refused everything this rejects, before the string reached
	// the node record. The check stands here too because this function is what
	// opens the file: a reader that trusts an earlier caller to have validated
	// its input is one refactor away from reading whatever it is handed.
	if err := checkNodeKeyRef(index, ref); err != nil {
		return derive.PrivateKey{}, err
	}
	b, rerr := os.ReadFile(ref)
	if rerr != nil {
		return derive.PrivateKey{}, ofKind(errKeyUnreadable,
			fmt.Errorf("chainsetup: keys: node%d: read key file %q: %w", index, ref, rerr))
	}
	// The parse failure is not wrapped with the bytes: a file that is nearly a
	// key is still a key someone meant to keep.
	key, perr := derive.ParsePrivateKey(string(b))
	if perr != nil {
		return derive.PrivateKey{}, ofKind(errKeyUnreadable,
			fmt.Errorf("chainsetup: keys: node%d: key file %q does not hold a private key", index, ref))
	}
	return key, nil
}

// AllocateOpts sizes the network.
