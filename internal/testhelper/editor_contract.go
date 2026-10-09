package testhelper

import (
	"fmt"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// ParameterSchema belongs beside the builtin implementations. The Web editor
// and its validator consume this same contract; they do not infer argv or RPCs.
type ParameterSchema = map[string]any

type VocabularyEntry struct {
	Name          string   `json:"name"`
	Kind          string   `json:"kind"`
	ContractID    string   `json:"contractId"`
	SchemaRef     string   `json:"schemaRef"`
	Prerequisites []string `json:"prerequisites,omitempty"`
}

type contractRegistry struct {
	interp.Registry
	entries []VocabularyEntry
}

func (r *contractRegistry) add(kind, name string) {
	r.entries = append(r.entries, VocabularyEntry{Name: name, Kind: kind, ContractID: "dsl", SchemaRef: "#/$defs/" + kind + "-" + name})
}
func (r *contractRegistry) RegisterAction(name string, a interp.Action) {
	r.Registry.RegisterAction(name, a)
	r.add("action", name)
}
func (r *contractRegistry) RegisterAssertion(name string, a interp.Assertion) {
	r.Registry.RegisterAssertion(name, a)
	r.add("assertion", name)
}
func (r *contractRegistry) RegisterReader(name string, rd interp.Reader) {
	r.Registry.RegisterReader(name, rd)
	r.add("reader", name)
}

// EditorVocabulary enumerates actual registrations, so a missing descriptor
// fails closed instead of silently hiding an engine-supported builtin.
func EditorVocabulary() ([]VocabularyEntry, map[string]ParameterSchema, error) {
	r := &contractRegistry{Registry: interp.NewRegistry()}
	Register(r)
	sort.Slice(r.entries, func(i, j int) bool { a, b := r.entries[i], r.entries[j]; return a.Kind+":"+a.Name < b.Kind+":"+b.Name })
	schemas := map[string]ParameterSchema{}
	readers := []string{}
	for _, e := range r.entries {
		if e.Kind == "reader" {
			readers = append(readers, e.Name)
		}
	}
	for i, e := range r.entries {
		// read and waitFor name the registration as an action; the reader's
		// own arguments are merged per source by the app.
		read, ok := builtinArgumentSet(e.Kind, e.Name)
		if !ok {
			return nil, nil, fmt.Errorf("builtin %s has no argument contract", e.Name)
		}
		props := map[string]any{}
		for field := range read {
			switch field {
			case "expected":
				// The document writes "is"; lowering renames it. waitFor also
				// takes the runtime spelling, which most cases use.
				props["is"] = map[string]any{}
				props["isPerChain"] = map[string]any{"type": "object"}
				if e.Kind == "action" {
					props["expected"] = editorParameter(field)
				}
			case "expect":
				props["expect"] = map[string]any{"enum": []string{"receipt", "revert", "reject", "keptOut", "fail"}}
				props["expectPerChain"] = map[string]any{"type": "object", "additionalProperties": props["expect"]}
			case "method":
				props[field] = editorParameter(field)
				if e.Name == actionPartition || e.Name == actionHealPartition {
					props[field] = map[string]any{"enum": []string{partitionByPeers, partitionByFirewall}}
				}
			default:
				props[field] = editorParameter(field)
			}
		}
		switch e.Kind {
		case "action":
			props["do"] = map[string]any{"const": e.Name}
		case "assertion":
			props["expect"] = map[string]any{"const": e.Name}
		}
		if e.Name == "read" || e.Name == "waitFor" {
			props["source"] = map[string]any{"enum": readers}
		}
		required := append([]string{}, editorRequired[e.Name]...)
		if e.Kind == "action" {
			required = append(required, "do")
		}
		if e.Kind == "assertion" {
			required = append(required, "expect")
		}
		for field, value := range editorDefaults(e.Name) {
			if parameter, ok := props[field].(map[string]any); ok {
				parameter["default"] = value
			}
		}
		if e.Kind == "assertion" {
			for _, builtin := range builtinAssertions() {
				if builtin.name == e.Name {
					props["compare"].(map[string]any)["default"] = builtin.defaultOp
				}
			}
		}
		parameterSchema := ParameterSchema{"type": "object", "properties": props, "required": required, "additionalProperties": false}
		for _, field := range editorRequired[e.Name] {
			if property, ok := props[field].(map[string]any); ok && property["type"] == "string" {
				property["minLength"] = 1
			}
		}
		if _, both := props["expected"]; both {
			// Lowering lets "is" overwrite "expected"; one of them would be ignored.
			parameterSchema["not"] = map[string]any{"required": []string{"is", "expected"}}
		}
		switch e.Name {
		case actionDeployContract:
			parameterSchema["anyOf"] = editorAlternatives("bytecode", "data")
		case assertContractChecksum:
			parameterSchema["anyOf"] = editorAlternatives("bytecode", "data", "address")
		case assertCreateAddress:
			parameterSchema["anyOf"] = editorAlternatives("deployer", "from")
		case actionLoad:
			parameterSchema["anyOf"] = editorAlternatives("gas", "fillPercent")
		case actionSwapNode:
			parameterSchema["anyOf"] = editorAlternatives("binary", "config", "genesisOverlay")
		case assertDerive:
			parameterSchema["anyOf"] = []any{map[string]any{"required": []string{"selector"}, "properties": map[string]any{"op": map[string]any{"const": "abiCall"}}}, map[string]any{"required": []string{"of"}}}
		}
		schemas[e.Kind+"-"+e.Name] = parameterSchema
		if strings.Contains(" stopNode startNode restartNode resetNode swapNode crossFork readNodeLog ", " "+e.Name+" ") {
			r.entries[i].Prerequisites = []string{"owned-node-control"}
		}
	}
	return r.entries, schemas, nil
}

var editorRequired = map[string][]string{
	"waitBlock": {"target"}, "read": {"source"}, "waitFor": {"source"}, "newAccount": {"saveKey"},
	"sendRawTampered": {"which", "senderKey", "feePayerKey", "to"}, "sendSetCode": {"key", "authorityKey", "delegate"}, "signAuthorization": {"authorityKey", "delegate"},
	"stopNode": {"on"}, "startNode": {"on"}, "restartNode": {"on"}, "resetNode": {"on"}, "swapNode": {"on"}, "readNodeLog": {"on"}, "partition": {"groups"},
	"faucet": {"to", "amount"}, "registerContract": {"to", "data"}, "metric": {"name"},
	"balanceAt": {"address"}, "codeAt": {"address"}, "nonceAt": {"address"}, "call": {"to", "data"}, "estimateGas": {"to", "data"},
	"txStatus": {"hash"}, "receiptLog": {"hash"}, "txMined": {"hash"}, "rpcCall": {"method"}, "rpcError": {"method"}, "methodPresent": {"method"}, "derive": {"op"}, "wsCollected": {"sub"},
}

func editorParameter(field string) map[string]any {
	s := map[string]any{"type": "string"}
	if field == "timeout" || field == "pollInterval" || field == "within" {
		s["format"] = "duration"
	}
	switch field {
	case "key", "feePayerKey", "senderKey", "authorityKey":
		return map[string]any{"type": "string", "pattern": `^\$(?:[A-Za-z_][A-Za-z0-9_]*|\{[A-Za-z_][A-Za-z0-9_]*\})$`, "description": "Private key binding from an earlier step. Literal private key material cannot be shared through the Web document API."}
	case "is", "expected", "delta", "tol":
		return map[string]any{}
	case "params", "of", "accessList", "topics":
		// A list may also come whole from an earlier step's saved value.
		return map[string]any{"anyOf": []any{map[string]any{"type": "array", "items": map[string]any{}}, map[string]any{"type": "string", "pattern": `^\$(?:[A-Za-z_][A-Za-z0-9_]*|\{[A-Za-z_][A-Za-z0-9_]*\})$`}}}
	case "groups":
		return map[string]any{"type": "array", "items": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}
	case "onEach":
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	case "genesisOverlay":
		return map[string]any{"type": "object"}
	case "config":
		return map[string]any{"anyOf": []any{map[string]any{"type": "object"}, map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}}
	case "wait", "expectRevert", "expectReject", "expectFail":
		s["type"] = "boolean"
	case "target", "value", "gas", "gasPrice", "maxFeePerGas", "maxPriorityFeePerGas", "nonce", "amount", "blocks", "count", "index", "topic", "maxBytes", "maxAdvance", "maxMillis", "maxSeconds", "minMillis", "minSeconds", "fillPercent":
		return map[string]any{"anyOf": []any{map[string]any{"type": "number", "minimum": 0}, map[string]any{"type": "string", "pattern": "^(\\$[A-Za-z_][A-Za-z0-9_]*|.*\\$\\{[A-Za-z_][A-Za-z0-9_]*\\}.*|0x[0-9a-fA-F]+|[0-9]+)$"}}}
	case "op":
		return map[string]any{"enum": []string{"sum", "diff", "mul", "abiCall", "word", "quorum"}}
	case "format":
		return map[string]any{"enum": []string{"dec", "hex"}}
	case "which":
		return map[string]any{"enum": []string{"sender", "feepayer"}}
	case "compare":
		return map[string]any{"enum": []string{"Equal", "NotEqual", "EqualCI", "Len", "Greater", "GreaterOrEqual", "Less", "LessOrEqual", "True", "False", "Nil", "NotNil", "Contains", "NotContains", "ElementsMatch", "Regexp", "In", "InDelta"}}
	}
	return s
}

func editorDefaults(name string) map[string]any {
	switch name {
	case actionSendTx, actionFaucet, actionDeployContract, actionRegisterContract, actionLoad:
		return map[string]any{"timeout": defaultTxTimeout.String(), "pollInterval": defaultTxPollInterval.String()}
	case actionWaitBlock:
		return map[string]any{"timeout": defaultWaitBlockTimeout.String(), "pollInterval": defaultWaitBlockPoll.String()}
	case actionWaitFor:
		return map[string]any{"timeout": defaultWaitForTimeout.String(), "pollInterval": defaultWaitForPoll.String(), "compare": "Equal"}
	case assertBlockAdvance, assertBlockStalled:
		return map[string]any{"timeout": defaultBlockAdvanceTimeout.String(), "pollInterval": defaultBlockAdvancePoll.String()}
	case assertWSSubscribe, assertWSCollected:
		return map[string]any{"timeout": defaultSubscribeTimeout.String(), "count": 1}
	case actionReadNodeLog:
		return map[string]any{"maxBytes": nodeLogDefaultMaxBytes}
	}
	return nil
}

func editorAlternatives(fields ...string) []any {
	alternatives := []any{}
	for _, field := range fields {
		alternatives = append(alternatives, map[string]any{"required": []string{field}})
	}
	return alternatives
}
