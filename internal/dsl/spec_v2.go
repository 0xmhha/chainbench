package dsl

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
)

// The v2 grammar: the types a declaration parses into, and the checks that are
// about a type rather than about parsing.
//
// The embedded schema is the field-level source of truth the parser, the docs
// and external tooling share; the strict parser is what enforces it. Parsing
// and env resolution are in parse_v2.go, and the v1 lowering in lower_v1.go.

// SchemaV2 is the canonical v2 grammar (schema/v2.schema.json). The strict
// parser is the enforcement; this document is the field-level source of truth
// the parser, docs, and external tooling share.
//
//go:embed schema/v2.schema.json
var SchemaV2 []byte

// DSL v2 (docs/dev/dsl-v2-proposal.md). v2 separates the
// declaration (env — the fingerprint/reuse unit) from the scenario (case), and
// projects every predicate through one statement form:
//
//	{ "do": "<action>", ...complement, ...adjunct }
//	{ "expect": "<source>", "is": <expected>, ...adjunct }
//
// v2 is not a second runtime: a case LOWERS onto the v1 Spec the engine
// already executes, and v1 files desugar into the same unified statement
// sequence — one execution path, two grammars. Unknown fields are errors
// (strict): v1 let typos flow through map[string]any to the runtime.

// KindEnv and KindCase are the v2 file kinds.
const (
	KindEnv  = "env"
	KindCase = "case"
)

// schemaVersionV2 is the v2 grammar version.
const schemaVersionV2 = "2"

// Statement is one unified v2 statement: exactly one of Do or Expect names the
// head; Args carry complement and adjuncts (on/save/timeout/compare/...).
type Statement struct {
	// Do is the action name ("" when this is an expect statement).
	Do string
	// Expect is the assertion/source name ("" when this is a do statement).
	Expect string
	// Args are the statement's remaining fields, in the runtime's (v1) arg
	// vocabulary — lowering already renamed "is" to "expected".
	Args map[string]any
}

// EnvV2 is the v2 environment declaration — the reuse unit.
type EnvV2 struct {
	SchemaVersion string `json:"schemaVersion"`
	Kind          string `json:"kind"`
	ID            string `json:"id"`
	// Description says what this environment is for, in prose. It carries no
	// execution semantics; it exists so a declaration can explain itself where
	// it is read, rather than in a file beside it.
	Description string `json:"description,omitempty"`
	Target      string `json:"target,omitempty"`
	Chain       string `json:"chain"`
	// Manifest is an external, project-supplied chain manifest JSON, run on the
	// built-in family named by Chain; GenesisTemplate is its genesis template.
	// They are the DSL equivalent of the CLI's --manifest/--genesis-template.
	Manifest        string                 `json:"manifest,omitempty"`
	GenesisTemplate string                 `json:"genesisTemplate,omitempty"`
	Binaries        map[string]BinaryRefV2 `json:"binaries,omitempty"`
	Keys            *KeysV2                `json:"keys,omitempty"`
	// Blueprint is a network declaration file — the layout AND the node keys in
	// one document. With it, no topology or key set is needed; it is the DSL
	// equivalent of the CLI's --blueprint.
	Blueprint    string                    `json:"blueprint,omitempty"`
	Genesis      *GenesisV2                `json:"genesis,omitempty"`
	Topology     map[string]any            `json:"topology,omitempty"`
	Hardforks    map[string]int            `json:"hardforks,omitempty"`
	Launch       map[string]map[string]any `json:"launch,omitempty"`
	Config       map[string]map[string]any `json:"config,omitempty"`
	Capabilities []string                  `json:"capabilities,omitempty"`
	// Accounts declares test accounts by name, created and funded when the
	// network comes up. They are not in the genesis on purpose: an account
	// funded at run time is one the genesis never has to mention, so preparing
	// a test account stops meaning editing the genesis and re-deriving
	// everything downstream of it.
	Accounts map[string]AccountV2 `json:"accounts,omitempty"`
	// Upgrade declares a mixed-binary handoff: the network starts on the
	// producer's binary and forks to the validators'. With it, Binaries names
	// the two roles ("producer", "validator") rather than a default.
	Upgrade *UpgradeV2 `json:"upgrade,omitempty"`
	// Attach says this environment does not compose a network: it runs against
	// one that is already up.
	//
	// Every other field above describes a network to build. Without this one a
	// case could only ever say "compose this", and attaching existed solely as
	// command-line flags — so a case that is only meaningful against a network
	// somebody else set up had no way to say so, and `validate` could not tell
	// the two apart.
	//
	// It is exclusive with the composition fields. A declaration that both
	// builds a network and attaches to one has not said which network its
	// assertions are about.
	Attach *AttachV2 `json:"attach,omitempty"`
}

// AttachV2 names the running network an env attaches to.
type AttachV2 struct {
	// RPC are the endpoints, in the order a spec's node selectors address them.
	// "${VAR:-default}" works here as it does for a binary, which is what keeps
	// a machine's address out of a committed case.
	RPC []string `json:"rpc"`
	// KeysDir is the key set the running network was composed from. Without it
	// a spec attached to a network cannot turn "node1" into an address — the
	// run holds no key set of its own.
	KeysDir string `json:"keysDir,omitempty"`
	// Provides is what the running network offers, for capability-gated cases.
	//
	// Nothing composed this network, so nothing advertised anything about it;
	// the operator who set it up is the only one who knows. It is "provides"
	// and not "capabilities" for the reason genesis.provides is: the env's
	// "capabilities" is a REQUIREMENT, and two opposite meanings under one word
	// is how six cases came to skip forever.
	Provides []string `json:"provides,omitempty"`
}

// AccountV2 is one declared test account.
//
// An empty declaration is deliberate and useful: an account with no balance is
// what a test needs to exercise the paths that fail for want of gas.
type AccountV2 struct {
	// Fund is the balance to send it once the chain is up, in wei (decimal or
	// 0x-hex). Empty leaves the account at zero.
	Fund string `json:"fund,omitempty"`
}

// The two binaries an upgrade env names, by the key it names them under.
//
// They are named for the fork rather than for a role, because that is what
// tells them apart. Both binaries run nodes of several roles; what differs is
// which side of the fork each one seals.
//
// They used to be spelled "producer" and "validator". Both words were wrong in
// the same way: "producer" reads as the bp role, and "validator" names what a
// bp does while another bp proposes — neither describes a binary. The names
// also disagreed with the rest of the handoff, which already says From and To
// in the code (upgrade.HandoffInputs) and on the command line
// (--from-binary/--to-binary, --from-genesis/--to-chain).
//
// Concretely, on the handoff this harness runs: BinaryFrom is go-wemix, which
// seals under poa up to the fork block; BinaryTo is go-wbft, which syncs those
// blocks as an endpoint until the fork and produces under the new consensus
// after it.
const (
	// BinaryFrom seals up to the fork.
	BinaryFrom = "from"
	// BinaryTo takes over after it.
	BinaryTo = "to"
	// BinaryDefault is what a node that names no binary runs. A declaration of
	// one binary calls it that, and a declaration of several says which of them
	// the rest of the network runs.
	BinaryDefault = "default"
)

// BinaryRefV2 is one entry of an env's "binaries": which binary the name means
// and, when it differs from the environment's, which chain that binary runs.
//
// A bare string is the shorthand and means "this environment's chain", which is
// every declaration that runs one build. Chain is for a network that runs two
// that are not the same chain — a handoff across a fork is the case — because
// the chain is what says the binary's flag vocabulary, its RPC namespace and
// what its consensus asks of a launch. A per-node binary without it left every
// node assembling argv against the other build's answers.
type BinaryRefV2 struct {
	Binary string `json:"binary"`
	Chain  string `json:"chain,omitempty"`
}

// UnmarshalJSON accepts both forms: "gwbft" and {"binary":"gwbft","chain":"wbft"}.
func (b *BinaryRefV2) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		b.Binary = name
		return nil
	}
	type ref BinaryRefV2
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var r ref
	if err := dec.Decode(&r); err != nil {
		return fmt.Errorf("dsl: a binaries entry is a name or {binary, chain}: %w", err)
	}
	if r.Binary == "" {
		return fmt.Errorf("dsl: a binaries entry needs a binary")
	}
	*b = BinaryRefV2(r)
	return nil
}

// UpgradeV2 declares a handoff composition: which hardfork preset shapes it
// and which genesis template the producer's binary generates from. It is a
// declaration only; the composer that runs it lives above the grammar.
type UpgradeV2 struct {
	// Preset names a hardfork preset under presets/chain, without the
	// directory or the extension.
	//
	// It used to have a sibling, "profile", that named the same document by
	// path. Two names for one thing, mutually exclusive, and no declaration in
	// the repository ever used the second — so a reader met a word the code
	// spelled two ways and the schema documented under a third directory.
	Preset string `json:"preset,omitempty"`
	// Template named the producer chain's own genesis template, for the handoff
	// composer that generated the pre-fork genesis by running that binary
	// against it. The ordinary path builds the genesis from the chain plugin's
	// template, so a declaration naming one is refused (see checkUpgrade)
	// rather than quietly composed from a different document.
	//
	// Deprecated: has no effect, and naming it is an error.
	Template string `json:"template,omitempty"`
	// Fork is the hardfork's name and At is the block it activates on. Given,
	// they are checked against the preset rather than replacing it: a case
	// saying which fork it tests and being wrong about it is worse than a case
	// that does not say.
	Fork string `json:"fork,omitempty"`
	At   *int64 `json:"at,omitempty"`
	// From and To name the binaries — the keys of "binaries" — that handle
	// before and after the fork. They default to "from" and "to".
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	// Style is how the network crosses the fork.
	//
	// UpgradeConcurrent is the unusual one and the one this harness runs: both
	// binaries are up from genesis and the nodes that seal change at the fork.
	// UpgradeRestart is the ordinary hardfork — every node runs the pre-fork
	// binary, then is stopped and relaunched on the post-fork one. Empty is
	// concurrent, which is what every upgrade declaration written so far means.
	Style string `json:"style,omitempty"`
	// Carry is which file brings the fork's configuration to the nodes that
	// need it: CarryGenesis (the default) gives the network two genesis
	// documents and one shape of config, CarryConfig one genesis document and
	// two shapes of config.
	//
	// It is a property of the network under test, not of the harness: a chain
	// team ships a hardfork one way or the other, and a case that pins the way
	// is testing the thing they will actually do.
	Carry string `json:"carry,omitempty"`
}

// Which file a fork's configuration travels in.
const (
	// CarryGenesis writes the fork's section into a second genesis document,
	// read by the nodes running the post-fork binary.
	CarryGenesis = "genesis"
	// CarryConfig leaves the genesis alone and writes the whole genesis, the
	// fork's section included, into those nodes' config file.
	CarryConfig = "config"
)

// How a network crosses a hardfork.
const (
	// UpgradeConcurrent runs both binaries from genesis; the sealing set
	// changes at the fork.
	UpgradeConcurrent = "concurrent"
	// UpgradeRestart is the ordinary hardfork of ONE chain: the nodes run the
	// pre-fork build, and at the fork every one of them is relaunched on the
	// build that knows what happens there.
	//
	// Nothing about the network changes but the executable. The chain is the
	// same chain, the nodes keep their databases and their work, and the fork's
	// own configuration is in the genesis from block 0 — a build that does not
	// know the fork simply never acts on it, which is why the build has to
	// change.
	UpgradeRestart = "restart"
)

// KeysV2 declares where node identities come from (background 1.4/1.5,
// algorithm steps 2-3 — gap G1's grammar side).
type KeysV2 struct {
	NodeKeys *KeySourceV2 `json:"nodekeys,omitempty"`
}

// KeySourceV2 is one key-material source declaration.
type KeySourceV2 struct {
	// Source is preset (default) | generate.
	Source string `json:"source"`
	// Ref is the key-set directory.
	Ref string `json:"ref,omitempty"`
	// Validators is how many of a generated set join the validator set (0 =
	// all). It has effect only with source "generate": it composes a network
	// whose key set has more identities than producers, the DSL equivalent of
	// the keys step's --validators.
	Validators int `json:"validators,omitempty"`
	// Bootnode named the external BLS-deriving binary. It is accepted so that
	// existing specs keep parsing, and ignored: BLS material is now derived in
	// process (derive.Derive).
	//
	// Deprecated: has no effect.
	Bootnode string `json:"bootnode,omitempty"`
}

// GenesisV2 declares the genesis build (gap G2). The runtime's proven path is
// template + overlay; Set is dot-path sugar over the same overlay.
type GenesisV2 struct {
	// Mode is "template" (default) or "existing". "existing" uses a finished
	// genesis file verbatim (Ref); the other design modes (build | inherit)
	// have no runtime boundary yet and are rejected by name.
	Mode string `json:"mode,omitempty"`
	// Ref names the finished genesis file for mode "existing": a portable
	// reference (resolved under the workspace-config genesis directory), a
	// srv:// path, or a local absolute path. Required for "existing", refused
	// otherwise.
	Ref string `json:"ref,omitempty"`
	// Set applies dot-path single values (e.g. "config.chainId": 8284).
	Set map[string]any `json:"set,omitempty"`
	// Overlay deep-merges into the built genesis.
	Overlay map[string]any `json:"overlay,omitempty"`
	// Provides names what this genesis makes the network ABLE to do, for cases
	// that gate on it.
	//
	// It belongs here because the genesis is what makes it true: seeding three
	// accounts with Extra bits is what gives a network the account-extra
	// capability, and shortening a governance expiry is what gives it
	// short-expiry. A capability declared anywhere else would be a claim with
	// nothing behind it.
	//
	// It is NOT the env's "capabilities" field, which says what a case REQUIRES
	// — the two read alike and mean opposite things. Cases that wrote their
	// capability there ended up requiring something no network offered and
	// skipped forever.
	Provides []string `json:"provides,omitempty"`
	// HaltsAt is the block this genesis makes the network stop one short of,
	// and 0 for a genesis that keeps producing.
	//
	// A network is gated ready by "is it advancing", and for a chain that is
	// meant to stop that question has the wrong answer. Declaring an
	// unsupported system-contract version is exactly such a chain: it seals up
	// to the block before the fork, refuses that one, and the gate then waits
	// out its whole budget and reports a correct network as unfit — the case
	// could not start, which is not the same as failing.
	//
	// Saying it here makes standing at haltsAt-1 a ready network, the way a
	// hardfork's handover block already does.
	HaltsAt int64 `json:"haltsAt,omitempty"`
	// PerBinary is, per binary name (the keys "binaries" declares), what that
	// binary's own genesis needs on top of the network's, in the same two forms
	// the network's genesis takes.
	//
	// A network can run two builds that do not accept the same genesis. The
	// nodes running a named binary initialize from the network's genesis merged
	// with its entry here; every other node gets the network's unchanged.
	PerBinary map[string]GenesisSideV2 `json:"perBinary,omitempty"`
}

// GenesisSideV2 is the extra one binary's genesis needs, in the same two forms
// the network's genesis takes.
type GenesisSideV2 struct {
	// Set applies dot-path single values (e.g. "config.croissantBlock": 20).
	Set map[string]any `json:"set,omitempty"`
	// Overlay deep-merges into the built genesis.
	Overlay map[string]any `json:"overlay,omitempty"`
}

// checkUpgrade refuses an upgrade declaration that contradicts itself or the
// environment around it.
//
// Every refusal here is one the runtime would otherwise meet as something else:
// a missing preset as a file-not-found, a misspelled side as a node running the
// wrong build, an unbuilt style as a handoff that quietly did the other thing.
func checkUpgrade(caseID string, u *UpgradeV2, env EnvV2) error {
	switch u.Style {
	case "", UpgradeConcurrent, UpgradeRestart:
	default:
		return fmt.Errorf("dsl: case %s: unknown upgrade style %q (want %s or %s)", caseID, u.Style, UpgradeConcurrent, UpgradeRestart)
	}
	switch u.Carry {
	case "", CarryGenesis, CarryConfig:
	default:
		return fmt.Errorf("dsl: case %s: unknown upgrade carry %q (want %s or %s)", caseID, u.Carry, CarryGenesis, CarryConfig)
	}
	// A preset describes a HANDOVER environment: which chain hands to which, at
	// which fork, and how many nodes stand on each side. A restart has no such
	// environment — one chain, one build at a time — so it says its fork and
	// its block itself, and there is no preset with anything to add.
	if u.Style == UpgradeRestart {
		if u.Preset != "" {
			return fmt.Errorf("dsl: case %s: a %s hardfork names no preset — a preset describes a handover between two chains, and this is one chain crossing its own fork", caseID, UpgradeRestart)
		}
		if u.Fork == "" || u.At == nil {
			return fmt.Errorf("dsl: case %s: a %s hardfork says which fork it crosses and at which block (\"fork\" and \"at\")", caseID, UpgradeRestart)
		}
	} else if u.Preset == "" {
		return fmt.Errorf("dsl: case %s: upgrade needs a \"preset\"", caseID)
	}
	// A hardfork says which build each node runs, and the node table is where it
	// says it. Without one nothing decides which side of the fork a node is on,
	// and the fork's own configuration is read out of the chain the post-fork
	// nodes run — so there is nothing to build it from either.
	if len(env.Topology) == 0 {
		return fmt.Errorf("dsl: case %s: upgrade needs a node table (topology.nodes[]) saying which build each node runs", caseID)
	}
	// The template was the handoff composer's: it generated the pre-fork genesis
	// by running the producer's binary against a template that binary ships.
	// The ordinary path builds it from the chain plugin's own template, so a
	// declaration naming one is describing a composer that no longer exists.
	// Refused rather than ignored, because a case that names a template and
	// gets another one is composing a network it did not ask for.
	if u.Template != "" {
		return fmt.Errorf("dsl: case %s: upgrade names a genesis template (%s), and the chain supplies its own — remove it", caseID, u.Template)
	}
	// The two sides, by the names the env's own binaries use.
	from, to := u.From, u.To
	if from == "" {
		from = BinaryFrom
	}
	if to == "" {
		to = BinaryTo
	}
	if from == to {
		return fmt.Errorf("dsl: case %s: upgrade names %q on both sides of the fork", caseID, from)
	}
	for _, name := range []string{from, to} {
		if env.Binaries[name].Binary == "" {
			return fmt.Errorf("dsl: case %s: upgrade runs %q across the fork but binaries.%s is missing", caseID, name, name)
		}
	}
	if len(env.Binaries) != 2 {
		return fmt.Errorf("dsl: case %s: an upgrade env names exactly the %s and %s binaries", caseID, from, to)
	}
	return nil
}

// HooksV2 are the case hooks. Override hooks (gap G5) are deliberately not
// accepted yet: parsing a declaration the runtime cannot execute would repeat
// the declared-but-never-emitted defect.
type HooksV2 struct {
	Pre    []map[string]any `json:"pre,omitempty"`
	Post   []map[string]any `json:"post,omitempty"`
	OnFail []map[string]any `json:"onFail,omitempty"`
}

// CaseV2 is the v2 scenario file.
type CaseV2 struct {
	SchemaVersion string `json:"schemaVersion"`
	Kind          string `json:"kind"`
	ID            string `json:"id"`
	// Description says what this case verifies, in prose. Strict parsing means
	// a case cannot carry a note unless the grammar has a place for one, and a
	// test that cannot say what it is for is read by opening its steps.
	Description      string            `json:"description,omitempty"`
	Env              json.RawMessage   `json:"env"`
	ApplicableChains string            `json:"applicableChains,omitempty"`
	Requires         []string          `json:"requires,omitempty"`
	On               string            `json:"on,omitempty"`
	Timeouts         map[string]string `json:"timeouts,omitempty"`
	Hooks            *HooksV2          `json:"hooks,omitempty"`
	Steps            []map[string]any  `json:"steps"`
}

// sniff reads just enough to route a raw spec to its grammar.
type sniff struct {
	SchemaVersion string `json:"schemaVersion"`
	Kind          string `json:"kind"`
}

// IsV2 reports whether raw declares the v2 grammar.
