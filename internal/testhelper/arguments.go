package testhelper

import (
	"slices"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// builtinArguments is what each registration reads, in runtime spelling ("is"
// is lowered to "expected"). It includes what the interpreter consumes on the
// builtin's behalf: "save" only where an action reports a value, and "onEach"
// only where every selected node is visited — an action is run once per node,
// and an assertion that checks just the first would drop the rest.
//
// The list is declared rather than computed so a reader can see it, and
// TestBuiltinArgumentsMatchImplementations derives it from the code so it
// cannot drift. Everything else is refused, by the editor and by validate:
// an argument nothing reads has no effect, and accepting it silently lets a
// case claim a timeout, a node or a saved value it never had.
var builtinArguments = map[string]string{
	"action:crossFork":                   "timeout",
	"action:deployContract":              "bytecode data from gas gasPrice key maxFeePerGas maxPriorityFeePerGas nonce on onEach pollInterval save timeout value",
	"action:faucet":                      "amount from gas gasPrice maxFeePerGas maxPriorityFeePerGas nonce on onEach pollInterval save timeout to",
	"action:healPartition":               "groups method",
	"action:load":                        "blocks fillPercent from gas gasPrice maxFeePerGas maxPriorityFeePerGas nonce on onEach pollInterval save timeout",
	"action:newAccount":                  "save saveKey",
	"action:partition":                   "groups method",
	"action:read":                        "on onEach save source",
	"action:readNodeLog":                 "maxBytes on onEach save",
	"action:registerContract":            "data from gas gasPrice maxFeePerGas maxPriorityFeePerGas nonce on onEach pollInterval save timeout to value",
	"action:resetNode":                   "on onEach",
	"action:restartNode":                 "on onEach",
	"action:sendRawTampered":             "feePayerKey on onEach reason save senderKey to value which",
	"action:sendSetCode":                 "authorityKey delegate key on onEach save",
	"action:sendTx":                      "accessList blocks data expect expectReject expectRevert feePayerKey from gas gasPrice key maxFeePerGas maxPriorityFeePerGas nonce on onEach pollInterval reason save timeout to value wait",
	"action:signAuthorization":           "authorityKey delegate on onEach save",
	"action:startNode":                   "expect expectFail on onEach reason save",
	"action:stopNode":                    "on onEach",
	"action:swapNode":                    "args binary config expect expectFail genesisOverlay on onEach purpose reason save",
	"action:waitBlock":                   "on onEach pollInterval target timeout",
	"action:waitFor":                     "compare delta expected on onEach pollInterval save source timeout tol",
	"action:wsOpen":                      "address event on onEach params save topics",
	"assertion:balanceAt":                "address compare delta expected on onEach tol",
	"assertion:baseFee":                  "compare delta expected on onEach tol",
	"assertion:blockAdvance":             "on onEach pollInterval timeout",
	"assertion:blockHalt":                "maxAdvance on onEach within",
	"assertion:blockInterval":            "blocks maxMillis maxSeconds minMillis minSeconds on",
	"assertion:blockNumber":              "compare delta expected on onEach tol",
	"assertion:blockStalled":             "on pollInterval timeout",
	"assertion:call":                     "compare data delta expected on onEach to tol",
	"assertion:callError":                "data on reason to",
	"assertion:chainId":                  "compare delta expected on onEach tol",
	"assertion:codeAt":                   "address compare delta expected on onEach tol",
	"assertion:contractChecksum":         "address bytecode compare data expected on onEach",
	"assertion:createAddress":            "compare delta deployer expected from nonce on onEach tol",
	"assertion:derive":                   "compare delta expected format index of on onEach op selector tol",
	"assertion:estimateGas":              "compare data delta expected from on onEach to tol",
	"assertion:gasPrice":                 "compare delta expected on onEach tol",
	"assertion:gasPriceIsBaseFeePlusTip": "on",
	"assertion:logs":                     "address compare delta expected fromBlock index on onEach select toBlock tol topics",
	"assertion:methodPresent":            "method on params",
	"assertion:metric":                   "compare expected name on onEach",
	"assertion:nonceAt":                  "address compare delta expected on onEach tol",
	"assertion:peerCount":                "compare delta expected on onEach tol",
	"assertion:receiptLog":               "address compare delta expected hash index on onEach select tol topic topic0",
	"assertion:rpcCall":                  "compare delta expected method on onEach params select tol",
	"assertion:rpcError":                 "method on params reason",
	"assertion:sameBlockHash":            "block on onEach",
	"assertion:txMined":                  "expected hash on",
	"assertion:txStatus":                 "compare delta expected hash on onEach tol",
	"assertion:validators":               "compare expected on onEach",
	"assertion:wsCollected":              "count sub timeout",
	"assertion:wsSubscribe":              "count event expected on params timeout",
	"reader:balanceAt":                   "address",
	"reader:baseFee":                     "",
	"reader:blockNumber":                 "",
	"reader:call":                        "data to",
	"reader:chainId":                     "",
	"reader:codeAt":                      "address",
	"reader:contractChecksum":            "address bytecode data",
	"reader:createAddress":               "deployer from nonce",
	"reader:derive":                      "format index of op selector",
	"reader:estimateGas":                 "data from to",
	"reader:gasPrice":                    "",
	"reader:logs":                        "address fromBlock index select toBlock topics",
	"reader:nonceAt":                     "address",
	"reader:peerCount":                   "",
	"reader:receiptLog":                  "address hash index select topic topic0",
	"reader:rpcCall":                     "method params select",
	"reader:txMined":                     "hash",
	"reader:txStatus":                    "hash",
	"reader:validators":                  "",
}

// builtinArgumentSet returns the arguments a registration reads, and whether
// the registration is known.
func builtinArgumentSet(kind, name string) (map[string]bool, bool) {
	fields, ok := builtinArguments[kind+":"+name]
	if !ok {
		return nil, false
	}
	set := map[string]bool{}
	for _, field := range strings.Fields(fields) {
		set[field] = true
	}
	return set, true
}

// IgnoredArguments returns the arguments a spec gives a builtin that the
// builtin does not read, as "<name>.<argument>", sorted and de-duplicated.
// Unknown builtin names are left to interp.Unresolved.
func IgnoredArguments(s dsl.Spec) []string {
	seen := map[string]bool{}
	check := func(kind, name string, args map[string]any, outcomes []string, skip ...string) {
		allowed, ok := builtinArgumentSet(kind, name)
		if !ok {
			return
		}
		if name == interp.ActionRead || name == interp.ActionWaitFor {
			source, _ := args["source"].(string)
			if fields, ok := builtinArgumentSet("reader", source); ok {
				for field := range fields {
					allowed[field] = true
				}
			}
		}
		for _, field := range skip {
			allowed[field] = true
		}
		for field := range args {
			if !allowed[field] {
				seen[name+"."+field] = true
			}
		}
		source, _ := args["source"].(string)
		for _, why := range unreadOnThisPath(kind, name, source, args, outcomes) {
			seen[name+"."+why] = true
		}
	}
	action := func(entry map[string]any, outcomes []string) {
		name := dsl.ActionName(entry)
		check("action", name, dsl.ArgsOf(entry[name]), outcomes)
	}
	assertion := func(entry map[string]any) {
		name, _ := entry["assert"].(string)
		check("assertion", name, entry, nil, "assert")
	}
	for _, entry := range s.PreActions {
		action(entry, nil)
	}
	if len(s.Sequence) > 0 {
		for _, st := range s.Sequence {
			if st.Do != "" {
				action(dsl.StatementStep(st), st.Outcomes)
			} else {
				assertion(dsl.StatementAssertion(st))
			}
		}
	} else {
		for _, entry := range s.Steps {
			action(entry, nil)
		}
		for _, entry := range s.Assertions {
			assertion(entry)
		}
	}
	for _, entry := range s.PostActions {
		action(entry, nil)
	}
	for _, entry := range s.OnFailActions {
		action(entry, nil)
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// firstRead lists arguments of which an implementation reads the first one
// present and never looks at the rest.
var firstRead = map[string][]string{
	"deployContract":   {"bytecode", "data"},
	"contractChecksum": {"bytecode", "data", "address"},
	"createAddress":    {"deployer", "from"},
	"load":             {"gas", "fillPercent"},
}

// nonNumeric lists readers whose value is never a number, so InDelta (and
// its delta and tol) can never hold for them.
var nonNumeric = map[string]bool{assertContractChecksum: true, assertValidators: true}

// localSigning lists what a step signed with "key" leaves out: the harness
// builds that transaction itself.
var localSigning = map[string][]string{
	"sendTx":         {"from"},
	"deployContract": {"from", "gas", "gasPrice", "maxFeePerGas", "maxPriorityFeePerGas", "nonce"},
}

// unreadOnThisPath returns the arguments a statement gives that its
// implementation reads only on a path this statement does not take, each with
// the reason. A value that is a binding is decided at run time, so a rule that
// depends on one is not applied.
func unreadOnThisPath(kind, name, source string, args map[string]any, outcomes []string) []string {
	var out []string
	has := func(field string) bool { _, ok := args[field]; return ok }
	text := func(field string) (string, bool) {
		v, ok := args[field].(string)
		return v, ok && !strings.HasPrefix(v, "$")
	}
	refuse := func(field, why string) {
		if has(field) {
			out = append(out, field+" ("+why+")")
		}
	}
	if has("on") && has("onEach") {
		refuse("onEach", "on and onEach select the same thing; write one")
	}
	builtin := name
	if name == "read" || name == "waitFor" {
		builtin = source
	}
	if fields, ok := firstRead[builtin]; ok {
		for i, first := range fields {
			if has(first) {
				for _, later := range fields[i+1:] {
					refuse(later, first+" is read instead")
				}
				break
			}
		}
	}
	if has("key") {
		for _, field := range localSigning[name] {
			refuse(field, "a step signed with key builds its own transaction")
		}
	}
	if op, ok := text("compare"); ok && op == "InDelta" && nonNumeric[builtin] {
		refuse("compare", "InDelta needs a number and "+builtin+" is not one")
	}
	if allowed, _ := builtinArgumentSet(kind, name); allowed["delta"] {
		op, given := text("compare")
		if !given {
			op = defaultCompare(name)
		}
		if op != "InDelta" && !strings.HasPrefix(op, "$") {
			refuse("delta", "read only with compare InDelta")
			refuse("tol", "read only with compare InDelta")
		} else if has("delta") {
			refuse("tol", "delta is read instead")
		}
	}
	switch name {
	case actionSendTx:
		if has("expect") && (has("expectRevert") || has("expectReject")) {
			refuse("expect", "expectRevert and expectReject decide the outcome instead")
		}
		reject, keptOut := expects(args, outcomes, "reject"), expects(args, outcomes, "keptOut")
		if b, ok := args["expectReject"].(bool); ok {
			reject = b
		}
		if !reject && !keptOut {
			refuse("reason", "read only when the step expects reject or keptOut")
		}
		if !keptOut {
			refuse("blocks", "read only with expect keptOut")
		}
	case actionPartition, actionHealPartition:
		if method, ok := text("method"); ok && method != partitionByPeers && method != partitionByFirewall {
			refuse("method", "peers or firewall; any other value falls back to peers")
		}
	case actionSwapNode:
		if !has("config") {
			refuse("purpose", "recorded only with a config change")
		}
	}
	switch name {
	case actionStartNode, actionSwapNode:
		if has("expect") && has("expectFail") {
			refuse("expect", "expectFail decides the outcome instead")
		}
		fail := expects(args, outcomes, "fail")
		if b, ok := args["expectFail"].(bool); ok {
			fail = b
		}
		if !fail {
			refuse("reason", "read only when the node is expected to fail")
		}
	}
	switch builtin {
	case assertBlockInterval:
		if has("maxMillis") {
			refuse("maxSeconds", "maxMillis is read instead")
		}
		if has("minMillis") {
			refuse("minSeconds", "minMillis is read instead")
		}
	case assertDerive:
		if op, ok := text("op"); ok {
			if op != "abiCall" {
				refuse("selector", "read only with op abiCall")
			}
			if op != "word" {
				refuse("index", "read only with op word")
			}
			if op != "sum" && op != "diff" && op != "mul" {
				refuse("format", "read only with op sum, diff or mul")
			}
		}
	case assertLogs:
		if selected, ok := text("select"); !has("select") || ok && selected == logSelectCount {
			refuse("index", "a count reads every matching log")
		}
	case assertReceiptLog:
		if selected, ok := text("select"); ok && selected != "data" {
			refuse("select", "only data is read; leave it out to read a topic")
		} else if ok {
			refuse("topic", "select data reads the data instead")
		}
	}
	return out
}

// expects reports whether the step expects outcome on this chain or, when it
// declares per-chain outcomes, on any chain.
func expects(args map[string]any, outcomes []string, outcome string) bool {
	if v, _ := args["expect"].(string); v == outcome {
		return true
	}
	return slices.Contains(outcomes, outcome)
}

// defaultCompare is the comparison a statement uses when it names none.
func defaultCompare(name string) string {
	for _, builtin := range builtinAssertions() {
		if builtin.name == name {
			return builtin.defaultOp
		}
	}
	return "Equal"
}
