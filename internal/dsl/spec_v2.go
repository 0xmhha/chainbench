package dsl

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
)

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

// UpgradeV2 declares a handoff composition: which golden profile shapes it
// and which genesis template the producer's binary generates from. It is a
// declaration only; the composer that runs it lives above the grammar.
type UpgradeV2 struct {
	// Preset names a hardfork preset under presets/hardfork, without the
	// directory or the extension.
	Preset string `json:"preset,omitempty"`
	// Profile is a hardfork preset by path, for one that is not under
	// presets/hardfork. Preset names one that is.
	Profile string `json:"profile,omitempty"`
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
		if u.Preset != "" || u.Profile != "" {
			return fmt.Errorf("dsl: case %s: a %s hardfork names no preset — a preset describes a handover between two chains, and this is one chain crossing its own fork", caseID, UpgradeRestart)
		}
		if u.Fork == "" || u.At == nil {
			return fmt.Errorf("dsl: case %s: a %s hardfork says which fork it crosses and at which block (\"fork\" and \"at\")", caseID, UpgradeRestart)
		}
	} else if u.Preset == "" && u.Profile == "" {
		return fmt.Errorf("dsl: case %s: upgrade needs a \"preset\" or a \"profile\"", caseID)
	}
	if u.Preset != "" && u.Profile != "" {
		return fmt.Errorf("dsl: case %s: upgrade names both a preset (%s) and a profile (%s) — name one", caseID, u.Preset, u.Profile)
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
func IsV2(raw []byte) bool {
	var s sniff
	return json.Unmarshal(raw, &s) == nil && s.SchemaVersion == schemaVersionV2
}

// parseStrict decodes into out rejecting unknown fields.
func parseStrict(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

// ParseV2 parses a v2 case (env inline) and lowers it onto the executable
// Spec. An env file alone is not runnable and is rejected with guidance.
func ParseV2(raw []byte) (Spec, error) {
	var s sniff
	if err := json.Unmarshal(raw, &s); err != nil {
		return Spec{}, fmt.Errorf("dsl: parse v2: %w", err)
	}
	switch s.Kind {
	case KindCase:
		var c CaseV2
		if err := parseStrict(raw, &c); err != nil {
			return Spec{}, fmt.Errorf("dsl: parse v2 case: %w", err)
		}
		return lowerCase(c)
	case KindEnv:
		return Spec{}, fmt.Errorf("dsl: an env declaration is not runnable — reference it from a case (\"env\": \"<id>\")")
	default:
		return Spec{}, fmt.Errorf("dsl: v2 spec needs \"kind\": %q or %q", KindCase, KindEnv)
	}
}

// IsEnv reports whether raw is a v2 env declaration: a file that is not
// runnable on its own but is what cases reference.
func IsEnv(raw []byte) bool {
	var s sniff
	return json.Unmarshal(raw, &s) == nil && s.SchemaVersion == schemaVersionV2 && s.Kind == KindEnv
}

// ParseEnv parses a v2 env declaration strictly, so a declaration can be
// validated on its own before any case references it.
func ParseEnv(raw []byte) (EnvV2, error) {
	var env EnvV2
	if err := parseStrict(raw, &env); err != nil {
		return EnvV2{}, fmt.Errorf("dsl: env: %w", err)
	}
	if env.Kind != KindEnv {
		return EnvV2{}, fmt.Errorf("dsl: env kind is %q, want %q", env.Kind, KindEnv)
	}
	if env.ID == "" || env.Chain == "" {
		return EnvV2{}, fmt.Errorf("dsl: env needs \"id\" and \"chain\"")
	}
	return env, nil
}

// InlineEnv resolves a case's env reference through lookup and rewrites the
// case with the env object inlined. Three forms are accepted:
//
//	"env": "<id>"                              — the canonical env, verbatim.
//	"env": {"extends": "<id>", "topology": …}  — the canonical env with the
//	                                             named fields overridden.
//	"env": { …a full env object… }             — inline, no lookup.
//
// The override form is a deep merge: an object meets an object by key, a null
// removes the key, and anything else replaces — so a case declares what differs
// and keeps the rest of the shared env. See mergeEnv for why each rule is what
// it is. A case with an inline env (or a v1 spec) passes through untouched.
// lookup receives the env id and returns the env file's bytes; the caller owns
// where env files live.
func InlineEnv(raw []byte, lookup func(id string) ([]byte, error)) ([]byte, error) {
	if !IsV2(raw) {
		return raw, nil
	}
	var probe struct {
		Kind string          `json:"kind"`
		Env  json.RawMessage `json:"env"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("dsl: inline env: %w", err)
	}
	if probe.Kind != KindCase || len(probe.Env) == 0 {
		return raw, nil
	}
	// "env": "<id>" — inline the canonical env verbatim.
	var id string
	if json.Unmarshal(probe.Env, &id) == nil && id != "" {
		envRaw, err := resolveEnv(id, lookup)
		if err != nil {
			return nil, err
		}
		return replaceEnv(raw, envRaw)
	}
	// "env": {"extends": "<id>", …} — inline the canonical env with overrides.
	var envObj map[string]json.RawMessage
	if json.Unmarshal(probe.Env, &envObj) == nil {
		if extRaw, ok := envObj["extends"]; ok {
			var baseID string
			if json.Unmarshal(extRaw, &baseID) != nil || baseID == "" {
				return nil, fmt.Errorf("dsl: env.extends must be an env id string")
			}
			baseRaw, err := resolveEnv(baseID, lookup)
			if err != nil {
				return nil, err
			}
			base, err := objectOf(baseRaw, baseID)
			if err != nil {
				return nil, err
			}
			if _, chained := base["extends"]; chained {
				return nil, fmt.Errorf("dsl: env %q extends another env; a shared env is the base, not a step in a chain", baseID)
			}
			over, err := objectOf(probe.Env, "the case's env")
			if err != nil {
				return nil, err
			}
			delete(over, "extends")
			merged, err := json.Marshal(mergeEnv(base, over))
			if err != nil {
				return nil, fmt.Errorf("dsl: merge env %q: %w", baseID, err)
			}
			return replaceEnv(raw, merged)
		}
	}
	return raw, nil // inline env object (or malformed — ParseV2 reports it)
}

// UseEnv rewrites a case so it runs on the env named envID, keeping whatever
// the case itself overrode.
//
// It is how one case runs on more than one chain without being edited. A case
// names its network, and that name is the last mainnet-specific thing left in
// the common cases; replacing it at read time is what lets the same steps meet
// a different chain.
//
// The two reference forms are rewritten; an inline env object is refused. An
// inline object IS the case's declaration, and swapping it would discard what
// the case asked for with no way to tell which parts mattered. Such a case is
// converted to the extends form first, which says out loud what it keeps.
//
// A non-case document, or a case with no env, passes through untouched.
func UseEnv(raw []byte, envID string) ([]byte, error) {
	if envID == "" || !IsV2(raw) {
		return raw, nil
	}
	var probe struct {
		Kind string          `json:"kind"`
		ID   string          `json:"id"`
		Env  json.RawMessage `json:"env"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("dsl: use env: %w", err)
	}
	if probe.Kind != KindCase || len(probe.Env) == 0 {
		return raw, nil
	}
	var id string
	if json.Unmarshal(probe.Env, &id) == nil && id != "" {
		return replaceEnv(raw, mustQuote(envID))
	}
	over, err := objectOf(probe.Env, "the case's env")
	if err != nil {
		return nil, err
	}
	if _, ok := over["extends"]; !ok {
		return nil, fmt.Errorf("dsl: case %s declares its env inline, so it cannot be moved onto env %q — give it \"extends\" and keep only what differs", probe.ID, envID)
	}
	over["extends"] = envID
	merged, err := json.Marshal(over)
	if err != nil {
		return nil, fmt.Errorf("dsl: use env %q: %w", envID, err)
	}
	return replaceEnv(raw, merged)
}

// mustQuote renders a string as a JSON scalar. The input is an env id that has
// already round-tripped through the resolver, so encoding cannot fail.
func mustQuote(s string) []byte {
	b, _ := json.Marshal(s)
	return b
}

// resolveEnv looks up a canonical env by id, requiring a resolver.
func resolveEnv(id string, lookup func(id string) ([]byte, error)) ([]byte, error) {
	if lookup == nil {
		return nil, fmt.Errorf("dsl: case references env %q but no env resolver is available", id)
	}
	envRaw, err := lookup(id)
	if err != nil {
		return nil, fmt.Errorf("dsl: resolve env %q: %w", id, err)
	}
	return envRaw, nil
}

// replaceEnv rewrites the case document with env set to the resolved object.
func replaceEnv(raw, envRaw []byte) ([]byte, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("dsl: inline env: %w", err)
	}
	doc["env"] = envRaw
	return json.Marshal(doc)
}

// lowerCase lowers a v2 case (env inline) onto the executable Spec.
func lowerCase(c CaseV2) (Spec, error) {
	if c.ID == "" {
		return Spec{}, fmt.Errorf("dsl: v2 case needs \"id\"")
	}
	if len(c.Env) == 0 {
		return Spec{}, fmt.Errorf("dsl: v2 case %s needs \"env\" (an env id or an inline env object)", c.ID)
	}
	var envID string
	if json.Unmarshal(c.Env, &envID) == nil {
		return Spec{}, fmt.Errorf("dsl: case %s references env %q — resolve it with InlineEnv before parsing", c.ID, envID)
	}
	var env EnvV2
	if err := parseStrict(c.Env, &env); err != nil {
		return Spec{}, fmt.Errorf("dsl: case %s: env: %w", c.ID, err)
	}
	if env.Kind != "" && env.Kind != KindEnv {
		return Spec{}, fmt.Errorf("dsl: case %s: env kind is %q, want %q", c.ID, env.Kind, KindEnv)
	}
	if env.Chain == "" {
		return Spec{}, fmt.Errorf("dsl: case %s: env needs \"chain\"", c.ID)
	}
	if err := checkAttach(c.ID, env); err != nil {
		return Spec{}, err
	}
	// Timeout values are durations; reject an unparsable one here so a typo
	// fails at parse time rather than being silently ignored at run time.
	//
	// The KEY is checked too, and for the same reason. Only case (alias test) is
	// consumed — the interpreter bounds the whole case with it — so any other key
	// was validated as a duration and then ignored, which is the worst of both:
	// the spec looks like it set a budget and no budget exists. A name outside the
	// vocabulary is a typo, not a feature request.
	for name, v := range c.Timeouts {
		if !timeoutKeys[name] {
			return Spec{}, fmt.Errorf("dsl: case %s: timeouts.%s is not a known timeout (want %s)", c.ID, name, timeoutKeyList())
		}
		if _, err := time.ParseDuration(v); err != nil {
			return Spec{}, fmt.Errorf("dsl: case %s: timeouts.%s %q is not a duration: %w", c.ID, name, v, err)
		}
	}

	spec := Spec{
		SchemaVersion:    supportedSchemaVersion, // lowered form IS the executable v1 shape
		ID:               c.ID,
		ApplicableChains: c.ApplicableChains,
		Requires:         c.Requires,
		Chain:            ChainSpec{Name: env.Chain},
		Topology:         env.Topology,
		Hardforks:        env.Hardforks,
		Placement:        env.Target,
		DefaultOn:        c.On,
		Timeouts:         c.Timeouts,
		EnvAttach:        env.Attach,
	}
	// The env's capabilities and the case's requires are both gating inputs, so
	// they union — a case that lists its own requires must not lose the ones the
	// env declares. Duplicates are dropped, case order first.
	if len(env.Capabilities) > 0 {
		seen := make(map[string]bool, len(spec.Requires))
		for _, r := range spec.Requires {
			seen[r] = true
		}
		for _, c := range env.Capabilities {
			if !seen[c] {
				seen[c] = true
				spec.Requires = append(spec.Requires, c)
			}
		}
	}

	// An external manifest (and its genesis template) runs on the family named
	// by chain; both travel on the chain spec for the composer to thread.
	spec.Chain.ManifestPath = env.Manifest
	spec.Chain.TemplatePath = env.GenesisTemplate

	// A definition names a binary; it does not place one. A path here is a
	// fact about one machine, and a case that carries it runs nowhere else.
	names := map[string]string{}
	for key, ref := range env.Binaries {
		if err := binaryRefIsAName(ref.Binary); err != nil {
			return Spec{}, fmt.Errorf("dsl: case %s: binaries.%s %q %w", c.ID, key, ref.Binary, err)
		}
		names[key] = ref.Binary
		if ref.Chain == "" {
			continue
		}
		if spec.Chain.BinaryChains == nil {
			spec.Chain.BinaryChains = map[string]string{}
		}
		spec.Chain.BinaryChains[key] = ref.Chain
	}

	// Binaries: "default" is every node's binary; other keys are per-role.
	if b, ok := names[BinaryDefault]; ok && len(names) == 1 {
		spec.Chain.Binary = b
	} else if len(names) > 0 {
		spec.Chain.Binaries = names
	}

	// An upgrade names its two binaries by which side of the fork each seals,
	// and nothing else: a default would mean every node runs one binary, which
	// is not a handoff.
	if u := env.Upgrade; u != nil {
		if err := checkUpgrade(c.ID, u, env); err != nil {
			return Spec{}, err
		}
		spec.EnvUpgrade = u
	}

	// Genesis: template(+overlay/set) is the runtime's proven path; the other
	// declared modes have no support yet and are rejected by name (G2 partial).
	if g := env.Genesis; g != nil {
		switch g.Mode {
		case "", "template":
			overlay := map[string]any{}
			maps.Copy(overlay, g.Overlay)
			for path, v := range g.Set {
				mergeDotPath(overlay, path, v)
			}
			if len(overlay) > 0 {
				spec.Chain.GenesisOverlay = overlay
			}
			spec.Chain.GenesisProvides = g.Provides
			spec.Chain.GenesisHaltsAt = g.HaltsAt
			for name, side := range g.PerBinary {
				if _, ok := env.Binaries[name]; !ok {
					return Spec{}, fmt.Errorf("dsl: case %s: genesis.perBinary names %q, which binaries does not declare", c.ID, name)
				}
				one := map[string]any{}
				maps.Copy(one, side.Overlay)
				for path, v := range side.Set {
					mergeDotPath(one, path, v)
				}
				if len(one) == 0 {
					return Spec{}, fmt.Errorf("dsl: case %s: genesis.perBinary.%s says nothing — give it a set or an overlay, or drop it", c.ID, name)
				}
				if spec.Chain.GenesisPerBinary == nil {
					spec.Chain.GenesisPerBinary = map[string]map[string]any{}
				}
				spec.Chain.GenesisPerBinary[name] = one
			}
		case "existing":
			// A finished genesis is used verbatim, so set/overlay — which edit a
			// built one — have nothing to act on and are refused rather than
			// silently ignored.
			if g.Ref == "" {
				return Spec{}, fmt.Errorf("dsl: case %s: genesis mode existing needs a ref", c.ID)
			}
			if len(g.Set) > 0 || len(g.Overlay) > 0 {
				return Spec{}, fmt.Errorf("dsl: case %s: genesis mode existing uses the file verbatim — set/overlay do not apply", c.ID)
			}
			spec.Chain.GenesisExisting = g.Ref
		default:
			return Spec{}, fmt.Errorf("dsl: case %s: genesis mode %q has no runtime boundary yet (supported: template, existing)", c.ID, g.Mode)
		}
	}

	// Keys/launch declarations carry through for the surface (cmd run) to fold
	// into the engine's construction boundaries.
	if len(env.Accounts) > 0 {
		spec.EnvAccounts = env.Accounts
	}
	if env.Keys != nil && env.Keys.NodeKeys != nil {
		spec.EnvKeys = env.Keys.NodeKeys
	}
	spec.EnvBlueprint = env.Blueprint
	if len(env.Launch) > 0 {
		spec.EnvLaunch = map[string][]string{}
		for scope, kvs := range env.Launch {
			if !node.ValidScope(scope) {
				return Spec{}, fmt.Errorf("dsl: case %s: launch scope %q must be %s", c.ID, scope, node.ScopeWords())
			}
			for k, v := range kvs {
				spec.EnvLaunch[scope] = append(spec.EnvLaunch[scope], fmt.Sprintf("%s=%v", k, v))
			}
		}
	}
	if len(env.Config) > 0 {
		spec.EnvConfig = map[string][]string{}
		for scope, kvs := range env.Config {
			if !node.ValidScope(scope) {
				return Spec{}, fmt.Errorf("dsl: case %s: config scope %q must be %s", c.ID, scope, node.ScopeWords())
			}
			for k, v := range kvs {
				spec.EnvConfig[scope] = append(spec.EnvConfig[scope], fmt.Sprintf("%s=%v", k, v))
			}
		}
	}

	// Hooks.
	if h := c.Hooks; h != nil {
		var err error
		if spec.PreActions, err = lowerHookActions(c.ID, "pre", h.Pre); err != nil {
			return Spec{}, err
		}
		if spec.PostActions, err = lowerHookActions(c.ID, "post", h.Post); err != nil {
			return Spec{}, err
		}
		if spec.OnFailActions, err = lowerHookActions(c.ID, "onFail", h.OnFail); err != nil {
			return Spec{}, err
		}
	}

	// Statements.
	if len(c.Steps) == 0 {
		return Spec{}, fmt.Errorf("dsl: case %s has no steps", c.ID)
	}
	expects := 0
	for i, raw := range c.Steps {
		st, err := lowerStatement(raw)
		if err != nil {
			return Spec{}, fmt.Errorf("dsl: case %s: step %d: %w", c.ID, i+1, err)
		}
		if st.Expect != "" {
			expects++
			spec.Assertions = append(spec.Assertions, StatementAssertion(st))
		}
		spec.Sequence = append(spec.Sequence, st)
	}
	if expects == 0 {
		return Spec{}, fmt.Errorf("dsl: case %s verifies nothing — at least one expect statement is required", c.ID)
	}
	if err := checkCrossFork(c.ID, spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

// ActionCrossFork is the step that crosses the hardfork a network is composed
// to cross.
//
// The name lives in the grammar because the grammar checks it and because the
// composer reads a case's steps for it: a case that names the step crosses the
// fork itself, and one that does not gets a network already past it. Three
// places have to mean the same word, so there is one.
const ActionCrossFork = "crossFork"

// checkCrossFork holds a case to what it said about crossing the fork.
//
// A case that names the step on a network with no hardfork is refused. The step
// would fail at run time with the same reason, but by then the network is up
// and the case has usually done something first — and the likeliest cause is an
// env that was meant to declare an upgrade and does not.
//
// Naming it twice is refused for the same kind of reason. The step is
// idempotent, so the second one reports "already crossed" and the case passes
// while reading as though it crossed twice.
func checkCrossFork(caseID string, spec Spec) error {
	crossings := 0
	for _, st := range spec.Sequence {
		if st.Do == ActionCrossFork {
			crossings++
		}
	}
	if crossings == 0 {
		return nil
	}
	if spec.EnvUpgrade == nil {
		return fmt.Errorf("dsl: case %s names the %s step, but its env declares no hardfork to cross — declare one under env.upgrade", caseID, ActionCrossFork)
	}
	if crossings > 1 {
		return fmt.Errorf("dsl: case %s names the %s step %d times, and a network crosses its fork once", caseID, ActionCrossFork, crossings)
	}
	return nil
}

// expectAliases maps proposal-vocabulary source names onto registered
// assertion names.
var expectAliases = map[string]string{"rpc": "rpcCall"}

// timeoutKeys is the timeout vocabulary a case's "timeouts" may name. Only these
// are read — interp.caseTimeout bounds the whole case with case (or its alias
// test) — so the set is the parser's promise that a declared budget takes effect.
// Widening it means giving the new key a consumer in the same change.
var timeoutKeys = map[string]bool{"case": true, "test": true}

// timeoutKeyList renders the vocabulary for an error message, sorted so the text
// does not depend on map order.
func timeoutKeyList() string {
	keys := make([]string, 0, len(timeoutKeys))
	for k := range timeoutKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// expectAdjuncts is the outcome vocabulary a do step's "expect" may name:
// receipt (the default — the tx must be mined), revert (mined with status 0x0),
// reject (the submit itself must fail), and fail (a launched node must not come
// up). A value outside this set is a typo that must be refused, not treated as
// the default success.
var expectAdjuncts = map[string]bool{"receipt": true, "revert": true, "reject": true, "fail": true}

// lowerStatement lowers one v2 statement map onto the runtime vocabulary.
func lowerStatement(m map[string]any) (Statement, error) {
	doName, hasDo := m["do"].(string)
	exName, hasEx := m["expect"].(string)
	// A do statement may carry expect as an ADJUNCT ("expect":"receipt"|"revert"
	// on sendTx). It is a statement head only when "do" is absent.
	if !hasDo && !hasEx {
		return Statement{}, fmt.Errorf("statement needs \"do\" or \"expect\"")
	}
	// A do step's expect names an outcome, not an assertion, so it is checked
	// here rather than by Unresolved (which resolves head expects as assertion
	// names). A value outside the outcome vocabulary would otherwise fall through
	// to the default "must succeed", silently turning a negative case positive.
	if hasDo && hasEx && !expectAdjuncts[strings.ToLower(exName)] {
		return Statement{}, fmt.Errorf("do step's expect adjunct %q is not a known outcome (want receipt, revert, reject, or fail)", exName)
	}
	if _, isOverride := m["override"]; isOverride {
		return Statement{}, fmt.Errorf("override hooks (G5) have no execution semantics yet and are not accepted")
	}
	args := make(map[string]any, len(m))
	for k, v := range m {
		switch k {
		case "do":
			// head, not an arg
		case "is":
			args["expected"] = v
		default:
			args[k] = v
		}
	}
	if hasDo {
		return Statement{Do: doName, Args: args}, nil
	}
	delete(args, "expect")
	if alias, ok := expectAliases[exName]; ok {
		exName = alias
	}
	return Statement{Expect: exName, Args: args}, nil
}

// StatementAssertion renders an expect statement in the v1 assertion map shape.
func StatementAssertion(st Statement) map[string]any {
	out := make(map[string]any, len(st.Args)+1)
	maps.Copy(out, st.Args)
	out["assert"] = st.Expect
	return out
}

// StatementStep renders a do statement in the v1 step map shape.
func StatementStep(st Statement) map[string]any {
	return map[string]any{st.Do: st.Args}
}

// mergeDotPath sets a dot-path value inside a nested map, creating levels.
func mergeDotPath(m map[string]any, path string, v any) {
	parts := strings.Split(path, ".")
	cur := m
	for _, p := range parts[:len(parts)-1] {
		next, ok := cur[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
	cur[parts[len(parts)-1]] = v
}

// lowerHookActions lowers hook statements (do form) onto v1 action maps.
func lowerHookActions(caseID, hook string, stmts []map[string]any) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(stmts))
	for i, raw := range stmts {
		st, err := lowerStatement(raw)
		if err != nil {
			return nil, fmt.Errorf("dsl: case %s: hooks.%s[%d]: %w", caseID, hook, i, err)
		}
		if st.Do == "" {
			return nil, fmt.Errorf("dsl: case %s: hooks.%s[%d]: hooks take do statements", caseID, hook, i)
		}
		out = append(out, StatementStep(st))
	}
	return out, nil
}

// envDefaultRE matches the ${VAR:-default} form and captures the default, which
// is the only part of an expansion this file can judge.
var envDefaultRE = regexp.MustCompile(`^\$\{[A-Za-z_][A-Za-z0-9_]*:-(.*)\}$`)

// binaryRefIsAName reports why a declared binary reference is not one.
//
// A definition says WHICH binary; a workspace-config says WHERE binaries live
// on the target (dataRoot plus paths.binaries) and binaryAliases says which
// file this environment calls that name. Writing the path in the definition
// says both at once, in the document that is supposed to travel: five specs
// carried /data/chainbench/bin/... and ran on one docker environment and
// nowhere else, while the environment file beside them already produced the
// same path from the name.
//
// An expansion is judged by its default, because that is the part the
// definition wrote. ${GWBFT_BIN:-gwbft} is a name with a machine-local escape
// hatch; ${GWBFT_BIN:-/opt/gwbft} is the path problem wearing a variable. A
// bare $VAR names nothing this file can see, so it passes and placeBinary
// judges what it expands to.
func binaryRefIsAName(ref string) error {
	if strings.TrimSpace(ref) == "" {
		return errors.New("is empty — name the binary, or leave it out and let the chain name it")
	}
	lit := ref
	if m := envDefaultRE.FindStringSubmatch(ref); m != nil {
		lit = m[1]
	} else if strings.Contains(ref, "$") {
		return nil // an expansion with no default: only the machine knows
	}
	switch {
	case strings.HasPrefix(lit, "~"):
		return errors.New("starts at a home directory, which is a fact about one machine")
	case strings.ContainsRune(lit, '/'):
		return errors.New("is a path — name the binary, and let a workspace-config say where binaries live on the target")
	}
	return nil
}

// objectOf decodes a JSON object, naming what failed. A null decodes into a nil
// map without error, and a caller that then assigns into it panics — so it is
// refused here with the same words a non-object gets.
func objectOf(raw []byte, what string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("dsl: %s is not an object: %w", what, err)
	}
	if m == nil {
		return nil, fmt.Errorf("dsl: %s is not an object: it is null", what)
	}
	return m, nil
}

// mergeEnv lays a case's overrides over a shared env and returns the result.
//
// Three rules, and each exists because the alternative loses something the
// author wrote:
//
//   - An object meets an object by KEY, recursively. Replacing the whole object
//     was the old rule, and under it a case that wanted node2 to differ wrote
//     config.node2 and silently dropped the shared env's "all" scope and every
//     other node's — the case ran, and only the result was different.
//   - null REMOVES the key. Deep merge alone can only add and overwrite, so
//     there would be no way to switch off something the shared env turned on.
//     That is not hypothetical: the commonest difference between the 205 inline
//     envs and the shape they share is a capability list that is absent.
//   - An array REPLACES. Unioning reads as generous and takes away the only way
//     to drop an entry, which is the same hole as having no delete.
//
// The merged object is parsed strictly afterwards, so a key neither side should
// have is refused there rather than checked twice here.
func mergeEnv(base, over map[string]any) map[string]any {
	for k, v := range over {
		if v == nil {
			delete(base, k)
			continue
		}
		bo, baseIsObject := base[k].(map[string]any)
		oo, overIsObject := v.(map[string]any)
		if baseIsObject && overIsObject {
			base[k] = mergeEnv(bo, oo)
			continue
		}
		base[k] = v
	}
	return base
}

// checkAttach holds an attaching env to saying only what attaching needs.
//
// The two forms are exclusive on purpose. Every composition field describes a
// network to build, and a declaration that both builds one and attaches to
// another has not said which network its assertions are about — it would build
// a network, run nothing against it, and report on a different one. That is the
// kind of wrong answer that reads as a pass.
//
// "chain" stays required. The direction recorded for this work was to read the
// chain's identity over RPC after attaching, and that is still right, but it
// cannot be done first: a spec that names a contract ("govMinter") needs the
// chain's table before the first call, and the table comes from the manifest.
func checkAttach(caseID string, env EnvV2) error {
	if env.Attach == nil {
		return nil
	}
	if len(env.Attach.RPC) == 0 {
		return fmt.Errorf("dsl: case %s: env.attach needs \"rpc\" (the endpoints of the network to run against)", caseID)
	}
	for i, u := range env.Attach.RPC {
		if strings.TrimSpace(u) == "" {
			return fmt.Errorf("dsl: case %s: env.attach.rpc[%d] is empty", caseID, i)
		}
	}
	var composing []string
	for name, set := range map[string]bool{
		"binaries":        len(env.Binaries) > 0,
		"keys":            env.Keys != nil,
		"blueprint":       env.Blueprint != "",
		"genesis":         env.Genesis != nil,
		"topology":        len(env.Topology) > 0,
		"hardforks":       len(env.Hardforks) > 0,
		"launch":          len(env.Launch) > 0,
		"config":          len(env.Config) > 0,
		"accounts":        len(env.Accounts) > 0,
		"upgrade":         env.Upgrade != nil,
		"target":          env.Target != "",
		"manifest":        env.Manifest != "",
		"genesisTemplate": env.GenesisTemplate != "",
	} {
		if set {
			composing = append(composing, name)
		}
	}
	if len(composing) > 0 {
		sort.Strings(composing)
		return fmt.Errorf("dsl: case %s: env.attach runs against a network that is already up, so it cannot also compose one — drop %s",
			caseID, strings.Join(composing, ", "))
	}
	return nil
}
