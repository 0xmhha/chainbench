package arch

import (
	"sort"
	"testing"
)

// nameShared holds names that more than one package declares on purpose.
//
// The line drawn here is between a STRUCTURAL role and DOMAIN vocabulary. A
// structural name says what a declaration is to its own package ("this
// package's dependencies", "this call's options"), and it only means anything
// with the package in front of it, so `collector.Options` and `health.Options`
// are two packages each naming their own thing correctly. Domain vocabulary
// names something in the chain, the node, the key or the resource, and there
// [[layers]] §5b applies: one concept keeps one name, and a second concept has
// to find another. Those go in nameCollisionDebt below.
//
// Verbs sit here for a different reason. `dsl.Parse` and `resource.Parse` read
// as English at the call site the way `json.Marshal` and `xml.Marshal` do; the
// package is part of the sentence rather than a disambiguator bolted on.
var nameShared = map[string]string{
	// Verbs. The package name always precedes them at the call site and the two together read as a sentence.
	"Build":            "building a genesis, and building a report",
	"Compose":          "assembling a genesis, and the preflight step that says nothing has been assembled yet",
	"DefaultKeySetDir": "passed straight through from app via operation to store",
	"Generate":         "generating a key set, and generating a report",
	"List":             "what a key set holds, and which sessions an artifact root has",
	"Load":             "reading a topology, an external plugin, and a validator roster",
	"Parse":            "reading a test spec, and reading a server selector",
	"Register":         "registering a chain plugin, and registering DSL vocabulary",
	"Resolve":          "layering configuration, and settling the network id",

	// Structural role names. The package completes the meaning, so sharing them is
	// the expected shape rather than a collision.
	"Config":  "every package has its own configuration struct",
	"Deps":    "every package declares what it takes at its boundary. This is what having no package-global state looks like",
	"Inputs":  "every package declares the inputs of its own call",
	"Options": "every package declares the optional arguments of its own call",
	"Request": "every package has its own request struct",
	"Result":  "every package has its own result struct",

	// The same constant offered again as it crosses a layer.
	"KeySetEnv":     "app and operation mirror store's environment variable name for surface help",
	"GenesisParams": "registry mirrors wbft's parameters so core need not import the family. The comment there says so",
}

// nameCollisionDebt holds domain words that two concepts are currently sharing.
// Each entry names what the two things actually are, and where one can, the
// item that renames one of them. It may only shrink: a name that stops
// colliding must leave this map, so the list tracks the code rather than
// drifting above it.
//
// This is the A7 measurement, taken 2026-09-07 with [Collisions].
var nameCollisionDebt = map[string]string{
	"Runner":      "poa means running a command, process means running a remote shell",
	"Handler":     "S track -- registry and mcp declare the same signature separately. They merge when the MCP schema shrinks",
	"NewServer":   "S track -- dashboard and mcp each build a server",
	"Server":      "S track -- the two above, plus resource's machine a node actually runs on: three",
	"Fingerprint": "B2 -- session's type and interp's constructor. Only the function remains once the type goes back to a string",
	"Network":     "app reads an attached network, keyring means a preset's chain parameters",
	"Chain":       "app resolves a plugin, nodeconfig is a configuration struct",
	"GenerateKey": "the accounts one is a throwaway for tests and must not carry this name",
	"Identity":    "derive means a derived identity safe to show, nodeconfig means who a node is",
	"Node":        "app aliases core/node, while preflight's is the subject of can this fill the role asked for",
	"NodeSwap":    "chainsetup is the act of swapping a node, hardfork is the struct recording that swap",
	"Step":        "chainsetup aliases session.Step, while poa's is one act of the bootstrap",
	"Entry":       "arch means a registered capability, keyring a key entry, node a boot entry",
	"Account":     "poa means a genesis-prefunded account, validatorset an account with a role, testhelper an account the DSL uses",
	"Label":       "core/node's comment already calls it a different concept that sometimes shares a spelling",
	"Spec":        "one node's configuration, a test definition, a server selector. Three strangers",
	"Plan":        "a handover plan, a hardfork plan, a launch plan, and resource's port placement function",
	"Report":      "app's lookup, health's verification result, report's session report",
	"Kind":        "collector means a kind of event, resource means how a server runs",
	"Phase":       "collector means a pipeline stage, registry a group of actions that must finish first",
	"Source":      "genesis means what offers extraData, keyring what offers a key",
	"Store":       "collector means an event store, filestore a file store",
	"Host":        "inspector means what to knock on, resource a machine with an address",
	"Ports":       "inspector is the act of checking ports, resource is the set assigned. 08-25 named this a real signal too",
	"Auth":        "core/node and core/remote each declare the same map[string]any. They can be merged",
	"Opener":      "operation is the narrowed interface, resource's is the implementation",
	"Registry":    "interp is the vocabulary registry interface, testhelper the function that builds one",
	"Lookup":      "registry finds a capability, assert finds an assertion, resource's is the function type that finds a credential",
	"Defaults":    "nodeconfig is the function that makes defaults, resource is the defaults struct",
	"Inventory":   "chainsetup is the function that gathers an inventory, resource is that inventory",
	"Verdict":     "preflight means how much has to be rebuilt, nodemonitor what the gate does next",

	// For the duration of the HSM migration only. Machine went when the old
	// core/lifecycle machine was deleted; State waits on renaming the record.
	"State": "core/statemachine means a state object with Enter, Exit and Process; chainsetup means the record left on disk (app aliases it). Renaming the old one to record belongs to commit 15, which raises FormatVersion to 2",
}

// TestNamesDoNotCollide is A7: an exported name declared at package level in
// more than one package is reported unless something accounts for it.
//
// It counts declarations rather than every identifier because a method is
// namespaced by its receiver. Including methods put 577 more names in the tally
// and buried `RoleValidator` among `Stop` and `Save`, which is how a test stops
// being read.
func TestNamesDoNotCollide(t *testing.T) {
	seenShared, seenDebt := map[string]bool{}, map[string]bool{}
	var explained int

	for _, c := range Collisions(moduleRoot) {
		if why := c.Explained(); why != "" {
			explained++
			continue
		}
		switch {
		case nameShared[c.Name] != "":
			seenShared[c.Name] = true
		case nameCollisionDebt[c.Name] != "":
			seenDebt[c.Name] = true
		default:
			t.Errorf("%s is declared in %v — one concept keeps one name ([[layers]] §5b); rename one, or record why both keep it in nameShared", c.Name, c.Pkgs)
		}
	}

	// A ratchet has to hold in both directions. An entry whose collision is
	// gone is a claim about the code that is no longer true, and leaving it
	// means the next reader trusts a list that has drifted.
	for _, m := range []struct {
		name string
		list map[string]string
		seen map[string]bool
	}{{"nameShared", nameShared, seenShared}, {"nameCollisionDebt", nameCollisionDebt, seenDebt}} {
		for name := range m.list {
			if !m.seen[name] {
				t.Errorf("%s[%q] matches no collision — the name is now unique, so remove the entry", m.name, name)
			}
		}
	}

	t.Logf("%d shared names are accounted for by rule, %d are tolerated, %d are debt",
		explained, len(seenShared), len(seenDebt))
	if testing.Verbose() {
		var debt []string
		for n := range seenDebt {
			debt = append(debt, n)
		}
		sort.Strings(debt)
		for _, n := range debt {
			t.Logf("  %s: %s", n, nameCollisionDebt[n])
		}
	}
}
