package testhelper

import (
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
	"action:swapNode":                    "binary config expect expectFail genesisOverlay on onEach purpose reason save",
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
	"assertion:contractChecksum":         "address bytecode compare data delta expected on onEach tol",
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
	"assertion:validators":               "compare delta expected on onEach tol",
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
	check := func(kind, name string, args map[string]any, skip ...string) {
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
	}
	action := func(entry map[string]any) {
		name := dsl.ActionName(entry)
		check("action", name, dsl.ArgsOf(entry[name]))
	}
	assertion := func(entry map[string]any) {
		name, _ := entry["assert"].(string)
		check("assertion", name, entry, "assert")
	}
	for _, entry := range s.PreActions {
		action(entry)
	}
	if len(s.Sequence) > 0 {
		for _, st := range s.Sequence {
			if st.Do != "" {
				action(dsl.StatementStep(st))
			} else {
				assertion(dsl.StatementAssertion(st))
			}
		}
	} else {
		for _, entry := range s.Steps {
			action(entry)
		}
		for _, entry := range s.Assertions {
			assertion(entry)
		}
	}
	for _, entry := range s.PostActions {
		action(entry)
	}
	for _, entry := range s.OnFailActions {
		action(entry)
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
