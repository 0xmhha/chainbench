package mcp

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/0xmhha/chainbench/internal/app"
)

// Keyring tools expose key material to an agent.
//
// There is deliberately no export tool. Exporting prints a private key, and an
// agent's transcript is a place secrets should never reach — the CLI keeps that
// verb behind an explicit confirmation, and a surface with no human at the
// keyboard has no equivalent. An agent that needs to sign asks a signing tool
// to do it, rather than asking for the key.
func keyringTools() []Tool {
	return []Tool{
		keyringNewTool(),
		keyringAddTool(),
		keyringListTool(),
		keyringShowTool(),
		keyringImportTool(),
	}
}

// ringSchema is the argument every keyring tool shares.
func ringSchema(extra map[string]any) map[string]any {
	props := map[string]any{
		"keyringDir": map[string]any{
			"type": "string",
			"description": "ring directory; a plain path is the operator machine, srv://<server>/path " +
				"places the ring on that server; omit for " + app.DefaultRingDir +
				" or the " + app.RingEnv + " environment variable",
		},
		"serverSet": map[string]any{"type": "string", "description": "server-set file for srv:// paths (which servers exist and how to reach them)"},
		"docker":    map[string]any{"type": "boolean", "description": "the server is a local docker container: translate dials via the localmap next to the server set"},
	}
	maps.Copy(props, extra)
	return map[string]any{"type": "object", "properties": props}
}

// ringRef reads the shared arguments.
func ringRef(args map[string]any) app.RingRef {
	return app.RingRef{
		Dir:       argString(args, "keyringDir", ""),
		ServerSet: argString(args, "serverSet", ""),
		Docker:    argBool(args, "docker", false),
	}
}

// makeArgs are the arguments the two creating tools share.
var makeArgs = map[string]any{
	"count":      map[string]any{"type": "number", "description": "how many identities to create"},
	"validators": map[string]any{"type": "number", "description": "how many join the validator set; omit for all on new and none on add, 0 to declare none"},
	"withBls":    map[string]any{"type": "boolean", "description": "also derive BLS material (required for stablenet and wbft)"},
	"password":   map[string]any{"type": "string", "description": "password for the generated keystores (default 1)"},
	"balance":    map[string]any{"type": "string", "description": "genesis balance per identity, 0x-hex wei"},
}

// createIn reads the creating tools' arguments. Whether validators was supplied
// is carried as a pointer, because absent and zero mean opposite things.
func createIn(args map[string]any) app.RingCreateIn {
	in := app.RingCreateIn{
		Ring:     ringRef(args),
		Count:    argInt(args, "count", 0),
		WithBLS:  argBool(args, "withBls", false),
		Password: argString(args, "password", "1"),
		Balance:  argString(args, "balance", defaultRingBalance),
	}
	if _, ok := args["validators"]; ok {
		v := argInt(args, "validators", 0)
		in.Validators = &v
	}
	return in
}

// defaultRingBalance matches the CLI's default so a ring made through either
// surface funds its identities the same way.
const defaultRingBalance = "0x200000000000000000000000000000000000000000000000000000000000000"

func keyringNewTool() Tool {
	return Tool{
		Name: "chainbench_keyring_new",
		Description: "Create a ring of node identities: keys, derived addresses, optional BLS material, " +
			"keystores, and the index the harness reads. Runs no chain binary.",
		InputSchema: ringSchema(makeArgs),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return renderRing(app.KeyringNew(ctx, app.Deps{}, createIn(args)))
		},
	}
}

func keyringAddTool() Tool {
	return Tool{
		Name: "chainbench_keyring_add",
		Description: "Add identities to an existing ring, keeping the ones already there. " +
			"New identities do not join the validator set unless asked.",
		InputSchema: ringSchema(makeArgs),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return renderRing(app.KeyringAdd(ctx, app.Deps{}, createIn(args)))
		},
	}
}

func keyringListTool() Tool {
	return Tool{
		Name:        "chainbench_keyring_list",
		Description: "List a ring's identities: label, address, whether it validates, and whether it has BLS material.",
		InputSchema: ringSchema(map[string]any{
			"verify": map[string]any{
				"type":        "boolean",
				"description": "re-derive every identity from its own key and fail on a mismatch",
			},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return renderRing(app.KeyringList(ctx, app.Deps{}, app.RingListIn{
				Ring: ringRef(args), Verify: argBool(args, "verify", false),
			}))
		},
	}
}

func keyringShowTool() Tool {
	return Tool{
		Name: "chainbench_keyring_show",
		Description: "Show one identity's public material: address, devp2p public key, and BLS key with its " +
			"proof of possession. Never includes the private key.",
		InputSchema: ringSchema(map[string]any{
			"name": map[string]any{"type": "string", "description": "identity label, e.g. node1"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			e, err := app.KeyringShow(ctx, app.Deps{}, app.RingEntryIn{
				Ring: ringRef(args), Label: argString(args, "name", ""),
			})
			if err != nil {
				return "", err
			}
			return asJSON(e)
		},
	}
}

func keyringImportTool() Tool {
	return Tool{
		Name: "chainbench_keyring_import",
		Description: "Bring an existing key into a ring, from a path here or on another host. " +
			"Prefer srv://<server>/path, which keeps the host address in the operator's server set " +
			"rather than in this conversation.",
		InputSchema: ringSchema(map[string]any{
			"name": map[string]any{"type": "string", "description": "label to store the identity under"},
			"from": map[string]any{
				"type":        "string",
				"description": "key file path: /local/path | srv://<server>/path | [user@]host:path | ssh://user@host:port/path",
			},
			"password":   map[string]any{"type": "string", "description": "password for a keystore named by from"},
			"mnemonic":   map[string]any{"type": "string", "description": "derive the key from a BIP-39 mnemonic (alternative to from)"},
			"passphrase": map[string]any{"type": "string", "description": "optional BIP-39 passphrase (with mnemonic)"},
			"hdCoinType": map[string]any{"type": "number", "description": "BIP-44 coin type for mnemonic (default 60)"},
			"hdAccount":  map[string]any{"type": "number", "description": "BIP-44 account index for mnemonic"},
			"hdChange":   map[string]any{"type": "number", "description": "BIP-44 change level for mnemonic (0 external, 1 internal)"},
			"hdIndex":    map[string]any{"type": "number", "description": "BIP-44 address index for mnemonic"},
			"withBls":    map[string]any{"type": "boolean", "description": "also derive BLS material"},
			"expectAddress": map[string]any{
				"type":        "string",
				"description": "refuse the import unless the key derives exactly this address",
			},
			"fromRing": map[string]any{
				"type": "string",
				"description": "clone a whole ring instead of one key (same path syntax as keyringDir); " +
					"labels and the validator declaration are copied, every entry verified against the source index",
			},
			"serverSet": map[string]any{"type": "string", "description": "server-set file for an srv:// source"},
			"docker":    map[string]any{"type": "boolean", "description": "the server is a local docker container: translate this dial via the localmap next to the server set"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			in := app.RingImportIn{
				Ring: ringRef(args), Label: argString(args, "name", ""),
				From: argString(args, "from", ""), Password: argString(args, "password", ""),
				Mnemonic:      argString(args, "mnemonic", ""),
				Passphrase:    argString(args, "passphrase", ""),
				HDCoinType:    uint32(argInt(args, "hdCoinType", 0)),
				HDAccount:     uint32(argInt(args, "hdAccount", 0)),
				HDChange:      uint32(argInt(args, "hdChange", 0)),
				HDIndex:       uint32(argInt(args, "hdIndex", 0)),
				WithBLS:       argBool(args, "withBls", false),
				Docker:        argBool(args, "docker", false),
				ExpectAddress: argString(args, "expectAddress", ""),
				FromRing:      argString(args, "fromRing", ""),
			}
			if in.FromRing != "" {
				return renderRing(app.KeyringImportRing(ctx, app.Deps{}, in))
			}
			e, err := app.KeyringImport(ctx, app.Deps{}, in)
			if err != nil {
				return "", err
			}
			return asJSON(e)
		},
	}
}

// renderRing reports a whole ring, keeping which directory was used and why —
// an agent that cannot see the resolved path cannot tell two rings apart.
func renderRing(r app.RingOut, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return asJSON(struct {
		Keyring    string         `json:"keyring"`
		Source     string         `json:"source"`
		Validators int            `json:"validators"`
		Entries    []app.EntryOut `json:"entries"`
	}{r.Dir, r.Source, r.Validators, r.Entries})
}

// asJSON renders a result for an agent.
func asJSON(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
