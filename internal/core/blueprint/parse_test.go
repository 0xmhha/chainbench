package blueprint_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/blueprint"
)

// full is the document design §3.1 shows, which is the widest shape this
// package has to carry.
const full = `version: 1
chain: wemix
binaries:
  node: /path/to/gwemix
  bootnode: /path/to/bootnode
  overrides:
    - nodes: [bp5, bp6]
      node: /path/to/gwbft
nodes:
  - name: bp1
    role: bp
    server: local
    ports: {p2p: 8589, http: 8588}
    nodekey: {file: ./keys/bp1.nodekey}
    account:
      keystore: ./keys/bp1.json
      password: ./keys/password
    syncmode: full
    launch: {maxtxsperblock: 1000}
  - {name: bp5, role: bp}
  - {name: bp6, role: bp}
  - {name: en1, role: en, server: srv2}
peering: mesh
validators:
  from: role
  stake: 1500000000000000000000000
alloc:
  - {account: bp1, balance: 200000000000000000000000}
  - {address: "0x000000000000000000000000000000000000dEaD", balance: 1000000000000000000}
genesis:
  chainId: 8285
  overrides: {bohoBlock: 10}
  overlay: ./overlay.json
governance:
  roles: {staker: bp1, ecosystem: bp1}
  env: {blockCreationTime: 1000}
`

// TestParse_RoundTripsWhatItReads is the generator's guarantee.
//
// `net blueprint --from-preset` writes a document a person then edits, so a
// field this package can read but not write is a field the generator silently
// drops. Comparing the re-read value rather than the bytes is deliberate:
// formatting is the encoder's business, meaning is this package's.
func TestParse_RoundTripsWhatItReads(t *testing.T) {
	for name, doc := range map[string]string{
		"the whole shape": full,
		"empty":           "",
		"a comment only":  "# nothing yet\n",
		"chain alone":     "chain: wbft\n",
		"four bare producers": `chain: wbft
nodes: [{role: bp}, {role: bp}, {role: bp}, {role: bp}]
`,
		"an amount that does not fit in an int64": "validators: {stake: 1500000000000000000000000}\n",
		"a hex nodekey": `nodes:
  - {name: bp1, nodekey: {hex: "0x` + strings.Repeat("ab", 32) + `"}}
`,
		"a manifest instead of a chain": "manifest: ./my-chain.json\n",
		"ports pinned one at a time":    "nodes: [{name: bp1, ports: {p2p: 30303}}]\n",
	} {
		t.Run(name, func(t *testing.T) {
			first, err := blueprint.Parse([]byte(doc))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			out, err := blueprint.Marshal(first)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			second, err := blueprint.Parse(out)
			if err != nil {
				t.Fatalf("re-parse of\n%s\n: %v", out, err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Errorf("round trip changed the blueprint\n first: %+v\nsecond: %+v\nwrote:\n%s", first, second, out)
			}
		})
	}
}

// TestParse_KeepsTheDigitsOfALargeAmount: a stake read as a number and written
// back as one loses its low digits, and a genesis built from it funds the wrong
// balance. Nothing downstream could notice.
func TestParse_KeepsTheDigitsOfALargeAmount(t *testing.T) {
	const stake = "1500000000000000000000000"
	bp, err := blueprint.Parse([]byte("validators: {stake: " + stake + "}\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := string(bp.Validators.Stake); got != stake {
		t.Fatalf("stake = %q, want %q", got, stake)
	}
	out, err := blueprint.Marshal(bp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), stake) {
		t.Errorf("the written document lost the digits:\n%s", out)
	}
}

// TestParse_RefusesAnUnknownField is the reason for KnownFields.
//
// A misspelled key has no symptom: the field is absent, the value is resolved
// from elsewhere, and the network comes up looking right and configured
// differently from what the document says.
func TestParse_RefusesAnUnknownField(t *testing.T) {
	for name, doc := range map[string]string{
		"at the top":      "chian: wbft\n",
		"inside a node":   "nodes: [{name: bp1, rol: bp}]\n",
		"inside ports":    "nodes: [{name: bp1, ports: {p2P: 30303}}]\n",
		"inside genesis":  "genesis: {chainID: 5}\n",
		"inside binaries": "binaries: {nodes: /x}\n",
		"inside an alloc": "alloc: [{acount: bp1}]\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := blueprint.Parse([]byte(doc)); err == nil {
				t.Fatalf("%q was accepted", doc)
			}
		})
	}
}

// TestParse_RefusesASecondDocument: a stream with two documents would have the
// second one silently dropped, and a person who wrote it meant it.
func TestParse_RefusesASecondDocument(t *testing.T) {
	if _, err := blueprint.Parse([]byte("chain: wbft\n---\nchain: wemix\n")); err == nil {
		t.Fatal("a two-document stream was accepted")
	}
}

// TestValidate_RefusesWhatTheDocumentAloneCanDecide walks the rules that need
// nothing but the document. Anything needing the registry, the disk or the
// network is Resolve's question and deliberately absent here.
func TestValidate_RefusesWhatTheDocumentAloneCanDecide(t *testing.T) {
	for name, c := range map[string]struct{ doc, wants string }{
		"a future schema version":     {"version: 2\n", "version 2"},
		"a chain and a manifest":      {"chain: wbft\nmanifest: ./x.json\n", "keep one"},
		"an unknown peering":          {"peering: star\n", "peering"},
		"an unknown role":             {"nodes: [{name: bp1, role: miner}]\n", "miner"},
		"a name that is not a path":   {"nodes: [{name: BP1}]\n", "lower-case"},
		"two nodes with one name":     {"nodes: [{name: bp1}, {name: bp1}]\n", "two nodes"},
		"an endpoint with an account": {"nodes: [{name: en1, role: en, account: {keystore: ./k.json}}]\n", "does not seal"},
		"an account with no keystore": {"nodes: [{name: bp1, role: bp, account: {password: ./p}}]\n", "keystore"},
		"a nodekey naming nothing":    {"nodes: [{name: bp1, nodekey: {}}]\n", "file or a hex"},
		"a nodekey naming both":       {"nodes: [{name: bp1, nodekey: {file: ./k, hex: \"0x00\"}}]\n", "not both"},
		"a nodekey of the wrong size": {"nodes: [{name: bp1, nodekey: {hex: \"0xabcd\"}}]\n", "32 bytes"},
		"a port no socket can carry":  {"nodes: [{name: bp1, ports: {p2p: 70000}}]\n", "65535"},
		"an override with no nodes":   {"binaries: {overrides: [{node: /x}]}\n", "no nodes"},
		"an override with no binary":  {"nodes: [{name: bp1}]\nbinaries: {overrides: [{nodes: [bp1]}]}\n", "no binary"},
		"an override naming a stranger": {"nodes: [{name: bp1}]\nbinaries: {overrides: [{nodes: [bp9], node: /x}]}\n",
			"declares bp1"},
		"validators derived and listed":         {"nodes: [{name: bp1}]\nvalidators: {from: role, explicit: [bp1]}\n", "not both"},
		"validators from a rule we do not have": {"validators: {from: stake}\n", "stake"},
		"validators naming a stranger":          {"nodes: [{name: bp1}]\nvalidators: {explicit: [bp9]}\n", "declares bp1"},
		"an alloc naming nothing":               {"alloc: [{balance: 1}]\n", "neither"},
		"an alloc naming both":                  {"alloc: [{account: bp1, address: \"0x00\"}]\n", "keep one"},
		"an alloc with a short address":         {"alloc: [{address: \"0xdead\"}]\n", "20-byte"},
		"an alloc naming a stranger":            {"nodes: [{name: bp1}]\nalloc: [{account: bp9}]\n", "declares bp1"},
		"an amount that is a list":              {"validators: {stake: [1, 2]}\n", "not a list"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := blueprint.Parse([]byte(c.doc))
			if err == nil {
				t.Fatalf("%q was accepted", c.doc)
			}
			if !strings.Contains(err.Error(), c.wants) {
				t.Errorf("error %q does not say %q, so it does not tell the writer what to fix", err, c.wants)
			}
		})
	}
}

// FuzzParse: a declaration is read from a file a person edits, so malformed
// input is the normal case and has to come back as an error.
//
// It also asserts the round trip on everything that parses, which is the part a
// hand-written table cannot cover: the shapes that survive are exactly the ones
// the generator has to be able to write back.
func FuzzParse(f *testing.F) {
	f.Add(full)
	for _, s := range []string{
		"", "chain: wbft\n", "nodes: [{role: bp}]\n", "\x00", "- - -", "!!binary x",
		"a: &x [*x]", strings.Repeat("nodes:\n", 50),
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, doc string) {
		bp, err := blueprint.Parse([]byte(doc))
		if err != nil {
			return
		}
		out, err := blueprint.Marshal(bp)
		if err != nil {
			t.Fatalf("a blueprint that parsed cannot be written back: %v (from %q)", err, doc)
		}
		again, err := blueprint.Parse(out)
		if err != nil {
			t.Fatalf("a written blueprint cannot be read back: %v\nwrote:\n%s\nfrom %q", err, out, doc)
		}
		if !reflect.DeepEqual(bp, again) {
			t.Fatalf("round trip changed the blueprint\nfrom %q\n first: %+v\nsecond: %+v", doc, bp, again)
		}
	})
}
