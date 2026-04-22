# Network Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the `network/` Go module skeleton, author three JSON Schemas (network, command, event) with runtime validation tests, wire up a `go generate` pipeline to produce Go types from those schemas, and produce two inventory documents (adapter contract + binary-name hardcoding audit) that future work depends on.

**Architecture:** A new Go module rooted at `network/` (inside the existing repo, not a git submodule) hosts a cobra-based CLI binary `chainbench-net`. Schemas live in `network/schema/*.json` as the single source of truth; Go types in `network/internal/types/` are generated from them via `go generate`. Runtime validation uses `santhosh-tekuri/jsonschema/v5`. Inventory scripts in `scripts/inventory/` produce deterministic outputs consumed by markdown docs.

**Tech Stack:** Go 1.23 (already installed), `github.com/spf13/cobra`, `github.com/santhosh-tekuri/jsonschema/v5`, `github.com/atombender/go-jsonschema` (generator, dev-only via `tools.go` build tag).

**Spec reference:** `docs/VISION_AND_ROADMAP.md` §5.15 (event catalog), §5.16 S1/S8, §5.17.1 (project structure), §6 Sprint 1.

---

## File Structure

**New files:**
- `network/go.mod`, `network/go.sum`
- `network/.gitignore`
- `network/README.md`
- `network/cmd/chainbench-net/main.go` — cobra root + `version` subcommand
- `network/cmd/chainbench-net/main_test.go`
- `network/schema/network.json`, `command.json`, `event.json`
- `network/schema/fixtures/network-local.json`, `network-remote.json`, `network-hybrid.json`
- `network/schema/fixtures/command-network-load.json`, `command-node-rpc.json`
- `network/schema/fixtures/event-node-started.json`, `event-chain-block.json`, `event-progress.json`, `event-result-ok.json`, `event-result-error.json`
- `network/schema/schema.go` — embeds schemas for runtime validation
- `network/schema/schema_test.go` — validates each fixture against its schema
- `network/internal/types/doc.go` — package doc + `go:generate` directives
- `network/internal/types/network_gen.go` — generated (committed)
- `network/internal/types/command_gen.go` — generated
- `network/internal/types/event_gen.go` — generated
- `network/internal/types/roundtrip_test.go`
- `network/tools.go` — build-tag-gated generator dependency pin
- `scripts/inventory/list-adapter-functions.sh`
- `scripts/inventory/scan-binary-hardcoding.sh`
- `docs/ADAPTER_CONTRACT.md`
- `docs/HARDCODING_AUDIT.md`

**Modified files:** none (Sprint 1 is additive).

---

## Task 1: Initialize Go module skeleton

**Files:**
- Create: `network/go.mod`
- Create: `network/.gitignore`
- Create: `network/README.md`

- [ ] **Step 1.1: Create the directory and module**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
mkdir -p network/cmd/chainbench-net network/schema/fixtures network/internal/types
cd network
go mod init github.com/0xmhha/chainbench/network
cd ..
```

Expected: `network/go.mod` exists and first line is `module github.com/0xmhha/chainbench/network`, second line is `go 1.23` (the installed toolchain).

- [ ] **Step 1.2: Write `network/.gitignore`**

```
# binaries
/bin/
/chainbench-net

# go
/vendor/
*.test
*.out
coverage.html
```

- [ ] **Step 1.3: Write `network/README.md`**

```markdown
# chainbench-net

Network abstraction layer for chainbench. Provides a uniform command/event
interface over local, remote, and (future) ssh-remote chain nodes. Invoked as a
subprocess by the chainbench CLI and MCP server.

See `docs/VISION_AND_ROADMAP.md` §5.15–5.17 for the design.

## Build

    go build -o bin/chainbench-net ./cmd/chainbench-net

## Develop

    go generate ./...
    go test ./...
```

- [ ] **Step 1.4: Verify module resolves**

Run: `cd network && go mod tidy && cd ..`
Expected: exits 0, `go.sum` not created yet (no deps). No errors about Go version.

- [ ] **Step 1.5: Commit**

```bash
git add network/go.mod network/.gitignore network/README.md
git commit -m "network: initialize Go module skeleton"
```

---

## Task 2: Minimal cobra CLI with `version` subcommand

**Files:**
- Create: `network/cmd/chainbench-net/main.go`
- Create: `network/cmd/chainbench-net/main_test.go`
- Modify: `network/go.mod` (cobra dep added)

- [ ] **Step 2.1: Add cobra dependency**

Run:
```bash
cd network && go get github.com/spf13/cobra@latest && cd ..
```

Expected: `network/go.mod` now contains `require github.com/spf13/cobra`; `network/go.sum` is created.

- [ ] **Step 2.2: Write the failing test**

Create `network/cmd/chainbench-net/main_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand_PrintsSemver(t *testing.T) {
	var buf bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "chainbench-net ") {
		t.Fatalf("want prefix %q, got %q", "chainbench-net ", out)
	}
	if !strings.Contains(out, "\n") {
		t.Fatalf("want trailing newline, got %q", out)
	}
}
```

- [ ] **Step 2.3: Run test — expect compile failure**

Run: `cd network && go test ./cmd/chainbench-net/... && cd ..`
Expected: FAIL — `newRootCmd` is undefined.

- [ ] **Step 2.4: Write minimal `main.go`**

Create `network/cmd/chainbench-net/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.0.0-dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "chainbench-net",
		Short:         "Network abstraction layer for chainbench",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the binary version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "chainbench-net %s\n", version)
			return err
		},
	}
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2.5: Run test — expect pass**

Run: `cd network && go test ./cmd/chainbench-net/... && cd ..`
Expected: PASS.

- [ ] **Step 2.6: Smoke-test the binary**

Run:
```bash
cd network && go run ./cmd/chainbench-net version && cd ..
```
Expected stdout: `chainbench-net 0.0.0-dev`.

- [ ] **Step 2.7: Commit**

```bash
git add network/go.mod network/go.sum network/cmd/chainbench-net/main.go network/cmd/chainbench-net/main_test.go
git commit -m "network: add cobra CLI entry with version subcommand"
```

---

## Task 3: `network.json` schema + fixtures + validation test

**Files:**
- Create: `network/schema/network.json`
- Create: `network/schema/fixtures/network-local.json`
- Create: `network/schema/fixtures/network-remote.json`
- Create: `network/schema/fixtures/network-hybrid.json`
- Create: `network/schema/schema.go`
- Create: `network/schema/schema_test.go`
- Modify: `network/go.mod` (jsonschema dep)

- [ ] **Step 3.1: Add validator dependency**

```bash
cd network && go get github.com/santhosh-tekuri/jsonschema/v5 && cd ..
```

- [ ] **Step 3.2: Write the failing test**

Create `network/schema/schema_test.go`:

```go
package schema

import (
	"path/filepath"
	"testing"
)

func TestNetworkSchema_AcceptsValidFixtures(t *testing.T) {
	fixtures := []string{
		"network-local.json",
		"network-remote.json",
		"network-hybrid.json",
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx, func(t *testing.T) {
			path := filepath.Join("fixtures", fx)
			if err := ValidateFile("network", path); err != nil {
				t.Fatalf("fixture %s must validate: %v", fx, err)
			}
		})
	}
}

func TestNetworkSchema_RejectsMissingChainType(t *testing.T) {
	doc := []byte(`{
		"name": "no-chain-type",
		"chain_id": 1,
		"nodes": [{"id":"n1","provider":"local","http":"http://127.0.0.1:8545"}]
	}`)
	if err := ValidateBytes("network", doc); err == nil {
		t.Fatal("expected validation error for missing chain_type")
	}
}
```

- [ ] **Step 3.3: Run test — expect compile failure**

Run: `cd network && go test ./schema/... && cd ..`
Expected: FAIL — `ValidateFile`, `ValidateBytes`, package `schema` not defined.

- [ ] **Step 3.4: Write `network.json`**

Create `network/schema/network.json`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://chainbench.io/schema/network.json",
  "title": "Network",
  "type": "object",
  "required": ["name", "chain_type", "chain_id", "nodes"],
  "additionalProperties": false,
  "properties": {
    "name":       { "type": "string", "pattern": "^[a-z0-9][a-z0-9_-]*$" },
    "chain_type": { "type": "string", "enum": ["stablenet", "wbft", "wemix", "ethereum"] },
    "chain_id":   { "type": "integer", "minimum": 1 },
    "nodes": {
      "type": "array",
      "minItems": 1,
      "items": { "$ref": "#/$defs/Node" }
    }
  },
  "$defs": {
    "Node": {
      "type": "object",
      "required": ["id", "provider", "http"],
      "additionalProperties": false,
      "properties": {
        "id":            { "type": "string", "minLength": 1 },
        "role":          { "type": "string", "enum": ["validator", "endpoint", "observer"] },
        "provider":      { "type": "string", "enum": ["local", "remote", "ssh-remote"] },
        "http":          { "type": "string", "format": "uri" },
        "ws":            { "type": "string", "format": "uri" },
        "auth":          { "$ref": "#/$defs/Auth" },
        "provider_meta": { "type": "object" }
      }
    },
    "Auth": {
      "type": "object",
      "required": ["type"],
      "oneOf": [
        {
          "properties": {
            "type":   { "const": "api-key" },
            "header": { "type": "string", "default": "Authorization" },
            "env":    { "type": "string", "description": "env var holding the key" }
          },
          "required": ["type", "env"]
        },
        {
          "properties": {
            "type": { "const": "jwt" },
            "env":  { "type": "string" }
          },
          "required": ["type", "env"]
        },
        {
          "properties": {
            "type": { "const": "ssh-password" },
            "user": { "type": "string" },
            "host": { "type": "string" },
            "port": { "type": "integer", "default": 22 }
          },
          "required": ["type", "user", "host"]
        }
      ]
    }
  }
}
```

- [ ] **Step 3.5: Write three fixtures**

Create `network/schema/fixtures/network-local.json`:

```json
{
  "name": "my-local",
  "chain_type": "stablenet",
  "chain_id": 8283,
  "nodes": [
    {
      "id": "node1",
      "role": "validator",
      "provider": "local",
      "http": "http://127.0.0.1:8501",
      "ws":   "ws://127.0.0.1:9501",
      "provider_meta": { "pid_key": "node1" }
    }
  ]
}
```

Create `network/schema/fixtures/network-remote.json`:

```json
{
  "name": "devnet",
  "chain_type": "stablenet",
  "chain_id": 8283,
  "nodes": [
    {
      "id": "endpoint-0",
      "role": "endpoint",
      "provider": "remote",
      "http": "https://devnet.example.com",
      "ws":   "wss://devnet.example.com/ws",
      "auth": { "type": "api-key", "env": "CHAINBENCH_DEVNET_KEY", "header": "X-API-Key" }
    }
  ]
}
```

Create `network/schema/fixtures/network-hybrid.json`:

```json
{
  "name": "mixed",
  "chain_type": "ethereum",
  "chain_id": 1337,
  "nodes": [
    { "id": "v1",  "role": "validator", "provider": "local",  "http": "http://127.0.0.1:8545" },
    { "id": "v2",  "role": "validator", "provider": "local",  "http": "http://127.0.0.1:8546" },
    { "id": "v3",  "role": "validator", "provider": "local",  "http": "http://127.0.0.1:8547" },
    { "id": "rpc", "role": "endpoint",  "provider": "remote", "http": "https://devnet.example.com" }
  ]
}
```

- [ ] **Step 3.6: Write `schema.go`**

Create `network/schema/schema.go`:

```go
// Package schema embeds the JSON Schemas that define the chainbench-net
// command/event/network contracts and exposes runtime validation helpers.
package schema

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed network.json command.json event.json
var schemaFS embed.FS

func loadSchema(name string) (*jsonschema.Schema, error) {
	path := name + ".json"
	data, err := schemaFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(path, bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("add %s: %w", path, err)
	}
	return compiler.Compile(path)
}

// ValidateBytes validates the given JSON document against the named schema
// ("network" | "command" | "event").
func ValidateBytes(name string, doc []byte) error {
	sch, err := loadSchema(name)
	if err != nil {
		return err
	}
	var v any
	if err := json.Unmarshal(doc, &v); err != nil {
		return fmt.Errorf("parse document: %w", err)
	}
	return sch.Validate(v)
}

// ValidateFile reads a JSON file and validates it against the named schema.
func ValidateFile(name, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	return ValidateBytes(name, data)
}
```

- [ ] **Step 3.7: Create placeholder `command.json` and `event.json` so `embed` compiles**

These will be filled out in Tasks 4 and 5. For now, minimal valid schemas:

`network/schema/command.json`:
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://chainbench.io/schema/command.json",
  "title": "Command",
  "type": "object"
}
```

`network/schema/event.json`:
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://chainbench.io/schema/event.json",
  "title": "StreamMessage",
  "type": "object"
}
```

- [ ] **Step 3.8: Run test — expect pass**

Run: `cd network && go test ./schema/... && cd ..`
Expected: PASS (two tests).

- [ ] **Step 3.9: Commit**

```bash
git add network/schema/ network/go.mod network/go.sum
git commit -m "network: add network JSON schema with fixtures and validator"
```

---

## Task 4: `command.json` schema + fixtures + test

**Files:**
- Modify: `network/schema/command.json` (replace placeholder)
- Create: `network/schema/fixtures/command-network-load.json`
- Create: `network/schema/fixtures/command-node-rpc.json`
- Modify: `network/schema/schema_test.go` (add command tests)

- [ ] **Step 4.1: Write the failing test**

Append to `network/schema/schema_test.go`:

```go
func TestCommandSchema_AcceptsValidFixtures(t *testing.T) {
	fixtures := []string{
		"command-network-load.json",
		"command-node-rpc.json",
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx, func(t *testing.T) {
			path := filepath.Join("fixtures", fx)
			if err := ValidateFile("command", path); err != nil {
				t.Fatalf("fixture %s must validate: %v", fx, err)
			}
		})
	}
}

func TestCommandSchema_RejectsUnknownCommand(t *testing.T) {
	doc := []byte(`{"command":"not.a.real.command","args":{}}`)
	if err := ValidateBytes("command", doc); err == nil {
		t.Fatal("expected validation error for unknown command name")
	}
}

func TestCommandSchema_RejectsMissingArgs(t *testing.T) {
	doc := []byte(`{"command":"network.load"}`)
	if err := ValidateBytes("command", doc); err == nil {
		t.Fatal("expected validation error for missing args")
	}
}
```

- [ ] **Step 4.2: Run test — expect fail**

Run: `cd network && go test ./schema/... && cd ..`
Expected: FAIL — two fixtures missing and placeholder schema accepts unknown commands.

- [ ] **Step 4.3: Write `command.json`**

Replace `network/schema/command.json`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://chainbench.io/schema/command.json",
  "title": "Command",
  "type": "object",
  "required": ["command", "args"],
  "additionalProperties": false,
  "properties": {
    "command": {
      "type": "string",
      "enum": [
        "network.load",
        "network.probe",
        "network.capabilities",
        "node.rpc",
        "node.start",
        "node.stop",
        "node.restart",
        "node.tail_log",
        "tx.send",
        "subscription.open"
      ]
    },
    "args": { "type": "object" },
    "env":  { "type": "object", "description": "optional env overrides passed through to the process" }
  }
}
```

- [ ] **Step 4.4: Write fixtures**

`network/schema/fixtures/command-network-load.json`:
```json
{ "command": "network.load", "args": { "name": "my-local" } }
```

`network/schema/fixtures/command-node-rpc.json`:
```json
{
  "command": "node.rpc",
  "args": {
    "node_id": "node1",
    "method":  "eth_blockNumber",
    "params":  []
  }
}
```

- [ ] **Step 4.5: Run test — expect pass**

Run: `cd network && go test ./schema/... && cd ..`
Expected: PASS (now 5 tests across command + network).

- [ ] **Step 4.6: Commit**

```bash
git add network/schema/command.json network/schema/fixtures/command-*.json network/schema/schema_test.go
git commit -m "network: add command JSON schema with fixtures"
```

---

## Task 5: `event.json` schema + event catalog + fixtures + test

**Files:**
- Modify: `network/schema/event.json` (replace placeholder)
- Create: `network/schema/fixtures/event-node-started.json`
- Create: `network/schema/fixtures/event-chain-block.json`
- Create: `network/schema/fixtures/event-progress.json`
- Create: `network/schema/fixtures/event-result-ok.json`
- Create: `network/schema/fixtures/event-result-error.json`
- Modify: `network/schema/schema_test.go` (add event tests)

- [ ] **Step 5.1: Write the failing test**

Append to `network/schema/schema_test.go`:

```go
func TestEventSchema_AcceptsValidFixtures(t *testing.T) {
	fixtures := []string{
		"event-node-started.json",
		"event-chain-block.json",
		"event-progress.json",
		"event-result-ok.json",
		"event-result-error.json",
	}
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx, func(t *testing.T) {
			path := filepath.Join("fixtures", fx)
			if err := ValidateFile("event", path); err != nil {
				t.Fatalf("fixture %s must validate: %v", fx, err)
			}
		})
	}
}

func TestEventSchema_RejectsUnknownEventName(t *testing.T) {
	doc := []byte(`{"type":"event","name":"not.a.real.event","ts":"2026-04-20T10:00:00Z"}`)
	if err := ValidateBytes("event", doc); err == nil {
		t.Fatal("expected validation error for unknown event name")
	}
}

func TestEventSchema_RejectsResultWithoutOk(t *testing.T) {
	doc := []byte(`{"type":"result"}`)
	if err := ValidateBytes("event", doc); err == nil {
		t.Fatal("expected validation error for result without ok field")
	}
}
```

- [ ] **Step 5.2: Run test — expect fail**

Run: `cd network && go test ./schema/... && cd ..`
Expected: FAIL — fixtures missing, placeholder schema accepts everything.

- [ ] **Step 5.3: Write `event.json`**

Replace `network/schema/event.json`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://chainbench.io/schema/event.json",
  "title": "StreamMessage",
  "description": "One JSON object per line on chainbench-net stdout. A stream always terminates with exactly one type=result message.",
  "oneOf": [
    { "$ref": "#/$defs/Event" },
    { "$ref": "#/$defs/Progress" },
    { "$ref": "#/$defs/Result" }
  ],
  "$defs": {
    "Event": {
      "type": "object",
      "required": ["type", "name", "ts"],
      "additionalProperties": false,
      "properties": {
        "type": { "const": "event" },
        "name": {
          "type": "string",
          "enum": [
            "node.started",
            "node.stopped",
            "node.health.changed",
            "chain.block",
            "chain.tx",
            "chain.log",
            "network.quorum",
            "error"
          ]
        },
        "data": { "type": "object" },
        "ts":   { "type": "string", "format": "date-time" }
      }
    },
    "Progress": {
      "type": "object",
      "required": ["type", "step"],
      "additionalProperties": false,
      "properties": {
        "type":  { "const": "progress" },
        "step":  { "type": "string" },
        "done":  { "type": "integer", "minimum": 0 },
        "total": { "type": "integer", "minimum": 1 }
      }
    },
    "Result": {
      "type": "object",
      "required": ["type", "ok"],
      "additionalProperties": false,
      "properties": {
        "type": { "const": "result" },
        "ok":   { "type": "boolean" },
        "data": { "type": "object" },
        "error": {
          "type": "object",
          "required": ["code", "message"],
          "properties": {
            "code":    { "type": "string", "enum": ["NOT_SUPPORTED", "PROTOCOL_ERROR", "UPSTREAM_ERROR", "INVALID_ARGS", "INTERNAL"] },
            "message": { "type": "string" }
          }
        }
      }
    }
  }
}
```

- [ ] **Step 5.4: Write fixtures**

`event-node-started.json`:
```json
{ "type": "event", "name": "node.started", "data": { "node_id": "node1", "pid": 12345 }, "ts": "2026-04-20T10:00:00Z" }
```

`event-chain-block.json`:
```json
{ "type": "event", "name": "chain.block", "data": { "height": 42, "hash": "0xabc", "miner": "0xc17d" }, "ts": "2026-04-20T10:00:01Z" }
```

`event-progress.json`:
```json
{ "type": "progress", "step": "init", "done": 2, "total": 4 }
```

`event-result-ok.json`:
```json
{ "type": "result", "ok": true, "data": { "blockNumber": "0x2a" } }
```

`event-result-error.json`:
```json
{ "type": "result", "ok": false, "error": { "code": "NOT_SUPPORTED", "message": "capability 'process' is unavailable in remote networks" } }
```

- [ ] **Step 5.5: Run test — expect pass**

Run: `cd network && go test ./schema/... && cd ..`
Expected: PASS (schema test count now 10).

- [ ] **Step 5.6: Commit**

```bash
git add network/schema/event.json network/schema/fixtures/event-*.json network/schema/schema_test.go
git commit -m "network: add event JSON schema with catalog and fixtures"
```

---

## Task 6: `go generate` pipeline producing Go types

**Files:**
- Create: `network/tools.go`
- Create: `network/internal/types/doc.go`
- Create: `network/internal/types/network_gen.go` (generated, committed)
- Create: `network/internal/types/command_gen.go` (generated, committed)
- Create: `network/internal/types/event_gen.go` (generated, committed)
- Create: `network/internal/types/roundtrip_test.go`

- [ ] **Step 6.1: Pin the generator tool via `tools.go`**

Create `network/tools.go`:

```go
//go:build tools
// +build tools

// Package tools pins development-only tool dependencies so they are recorded
// in go.mod but not compiled into the chainbench-net binary.
package tools

import (
	_ "github.com/atombender/go-jsonschema"
)
```

Run:
```bash
cd network && go mod tidy && cd ..
```
Expected: `github.com/atombender/go-jsonschema` appears in `go.mod`.

- [ ] **Step 6.2: Install the generator locally**

```bash
cd network && go install github.com/atombender/go-jsonschema@latest && cd ..
```

Verify: `which go-jsonschema` returns a path under `$HOME/go/bin` (or `$GOBIN`).

- [ ] **Step 6.3: Write `network/internal/types/doc.go` with `go:generate` directives**

```go
// Package types contains Go structs generated from the JSON Schemas under
// network/schema/. Run `go generate ./...` from the network/ module root after changing
// any schema file.
package types

//go:generate go-jsonschema --package types --output network_gen.go ../../schema/network.json
//go:generate go-jsonschema --package types --output command_gen.go ../../schema/command.json
//go:generate go-jsonschema --package types --output event_gen.go   ../../schema/event.json
```

- [ ] **Step 6.4: Run the generator**

```bash
cd network && go generate ./internal/types/... && cd ..
```

Expected: `network/internal/types/{network_gen.go,command_gen.go,event_gen.go}` appear, each beginning with `// Code generated by github.com/atombender/go-jsonschema, DO NOT EDIT.`

- [ ] **Step 6.5: Write the roundtrip test**

Create `network/internal/types/roundtrip_test.go`:

```go
package types

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNetwork_RoundtripLocalFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "schema", "fixtures", "network-local.json"))
	if err != nil {
		t.Fatal(err)
	}
	var n Network
	if err := json.Unmarshal(data, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got, want map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}

	// Compare semantic equality via JSON re-marshal.
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("roundtrip lost data\n got:  %s\n want: %s", gotJSON, wantJSON)
	}
}
```

> **Note:** the generator names the root type after the schema's `title`. The schema we wrote has `"title": "Network"`, so the struct should be `Network`. If generation produces a different symbol, update this test to match — but first confirm the `title` is preserved verbatim across all three schemas.

- [ ] **Step 6.6: Run test**

Run: `cd network && go test ./internal/types/... && cd ..`
Expected: PASS.

- [ ] **Step 6.7: Verify whole-module test suite still green**

Run: `cd network && go test ./... && cd ..`
Expected: PASS. All packages.

- [ ] **Step 6.8: Commit**

```bash
git add network/tools.go network/internal/ network/go.mod network/go.sum
git commit -m "network: generate Go types from JSON schemas via go:generate"
```

---

## Task 7: Adapter function inventory

**Files:**
- Create: `scripts/inventory/list-adapter-functions.sh`
- Create: `docs/ADAPTER_CONTRACT.md`

- [ ] **Step 7.1: Write the inventory script**

Create `scripts/inventory/list-adapter-functions.sh`:

```bash
#!/usr/bin/env bash
# Prints every adapter_* function defined in lib/adapters/*.sh with its file,
# line, and implementation status (real vs. stub).
#
# Usage: scripts/inventory/list-adapter-functions.sh [--json]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ADAPTERS_DIR="${ROOT}/lib/adapters"

format="text"
[[ "${1:-}" == "--json" ]] && format="json"

declare -a rows=()
for f in "${ADAPTERS_DIR}"/*.sh; do
  chain="$(basename "$f" .sh)"
  while IFS= read -r line; do
    lineno="${line%%:*}"
    fn="${line#*:}"
    fn="${fn%%(*}"
    fn="${fn## }"; fn="${fn%% }"
    status="real"
    if grep -q "_cb_${chain}_not_implemented" "$f" && \
       grep -q "^${fn}()[[:space:]]*{[[:space:]]*_cb_${chain}_not_implemented" "$f"; then
      status="stub"
    fi
    rows+=("${chain}|${fn}|${f#${ROOT}/}:${lineno}|${status}")
  done < <(grep -n "^adapter_[a-zA-Z_]*()" "$f" || true)
done

if [[ "$format" == "json" ]]; then
  printf '[\n'
  first=1
  for r in "${rows[@]}"; do
    IFS='|' read -r chain fn loc status <<<"$r"
    [[ $first -eq 0 ]] && printf ',\n'
    first=0
    printf '  {"chain":"%s","function":"%s","location":"%s","status":"%s"}' "$chain" "$fn" "$loc" "$status"
  done
  printf '\n]\n'
else
  printf '%-10s %-40s %-50s %s\n' CHAIN FUNCTION LOCATION STATUS
  for r in "${rows[@]}"; do
    IFS='|' read -r chain fn loc status <<<"$r"
    printf '%-10s %-40s %-50s %s\n' "$chain" "$fn" "$loc" "$status"
  done
fi
```

- [ ] **Step 7.2: Make it executable and run**

```bash
chmod +x scripts/inventory/list-adapter-functions.sh
scripts/inventory/list-adapter-functions.sh
```

Expected output (text mode), lines like:
```
CHAIN      FUNCTION                                 LOCATION                                           STATUS
stablenet  adapter_generate_genesis                 lib/adapters/stablenet.sh:14                       real
stablenet  adapter_generate_toml                    lib/adapters/stablenet.sh:124                      real
stablenet  adapter_extra_start_flags                lib/adapters/stablenet.sh:203                      real
stablenet  adapter_consensus_rpc_namespace          lib/adapters/stablenet.sh:216                      real
wbft       adapter_generate_genesis                 lib/adapters/wbft.sh:<N>                           stub
... (and so on)
```

- [ ] **Step 7.3: Write `docs/ADAPTER_CONTRACT.md`**

```markdown
# Adapter Contract

> **Status:** draft — inventory of current `lib/adapters/*.sh` interface, to be
> promoted to the Network Abstraction interface contract (§5.17 of VISION_AND_ROADMAP).
> Regenerate the tables below with:
>
>     scripts/inventory/list-adapter-functions.sh

## 1. Current adapter surface (stablenet — authoritative)

These are the four functions every chain adapter must provide today. Each is
called from `lib/cmd_*.sh` through the dispatcher in `lib/chain_adapter.sh`.

| Function | Arity | Purpose |
|---|---|---|
| `adapter_generate_genesis <profile_json> <template> <out> <meta> <num_validators> <base_p2p>` | 6 | Produce `genesis.json` and sidecar metadata for the chosen chain |
| `adapter_generate_toml <profile_json> <node_idx> <out>` | 3 | Produce the per-node TOML config the binary reads at startup |
| `adapter_extra_start_flags` | 0 | Chain-specific CLI flags appended to the node launch command |
| `adapter_consensus_rpc_namespace` | 0 | Name of the RPC namespace exposing validator/consensus methods (e.g. `istanbul`) |

## 2. Per-chain implementation status

<paste output of `scripts/inventory/list-adapter-functions.sh` below and keep in sync>

## 3. Gaps — functions the Network Abstraction contract will need but adapters do not expose today

Derived from §5.15 event catalog and §5.17 Network Abstraction interface.

| Proposed function | Why needed | First consumer |
|---|---|---|
| `adapter_binary_name` | Remove `gstable` hardcoding from `cmd_start/stop/node` (see `docs/HARDCODING_AUDIT.md`) | LocalDriver subprocess launch |
| `adapter_datadir_layout <node_idx>` | Compute per-node data dir path without shell assumptions | LocalDriver + log tail |
| `adapter_log_file_path <node_idx>` | Locate the appender-rotated file driver-side | `node.tail_log` |
| `adapter_consensus_validator_rpc_method` | Uniform access to `istanbul_getValidators` / `clique_getSigners` / `wemix_*` | `network.capabilities`, consensus tests |
| `adapter_supported_tx_types` | Gate chain-specific tx types (0x16, 0x04) for Layer 2 tx_builder | `tx.send` composite |
| `adapter_probe_markers` | RPC method names whose presence identifies the chain type | `network.probe` (§5.17 Q2) |

## 4. Migration guidance

Each gap becomes a named entry in the generated `network/schema/network.json`
capability flags and a Go interface method in `network/internal/adapters/`. See
§5.12 M0–M4 of the roadmap for the sequencing.
```

- [ ] **Step 7.4: Regenerate and paste the Section 2 output**

Run: `scripts/inventory/list-adapter-functions.sh >> /tmp/adapter-inventory.txt`

Manually copy the table from `/tmp/adapter-inventory.txt` into `docs/ADAPTER_CONTRACT.md` Section 2 (replace the `<paste output...>` line).

- [ ] **Step 7.5: Commit**

```bash
git add scripts/inventory/list-adapter-functions.sh docs/ADAPTER_CONTRACT.md
git commit -m "docs: add adapter contract with function inventory"
```

---

## Task 8: Chain binary hardcoding audit

**Files:**
- Create: `scripts/inventory/scan-binary-hardcoding.sh`
- Create: `docs/HARDCODING_AUDIT.md`

- [ ] **Step 8.1: Write the scan script**

Create `scripts/inventory/scan-binary-hardcoding.sh`:

```bash
#!/usr/bin/env bash
# Scans lib/cmd_*.sh for references to the current chain binary name
# ("gstable"). Output lists each hit with file:line:context.
#
# Usage: scripts/inventory/scan-binary-hardcoding.sh [--json]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CMD_DIR="${ROOT}/lib"

format="text"
[[ "${1:-}" == "--json" ]] && format="json"

declare -a rows=()
while IFS= read -r hit; do
  rows+=("$hit")
done < <(grep -n 'gstable' "${CMD_DIR}"/cmd_*.sh || true)

if [[ "$format" == "json" ]]; then
  printf '[\n'
  first=1
  for r in "${rows[@]}"; do
    file="${r%%:*}"; rest="${r#*:}"
    lineno="${rest%%:*}"; text="${rest#*:}"
    # JSON-escape the text (basic: backslashes and quotes)
    esc="${text//\\/\\\\}"; esc="${esc//\"/\\\"}"
    [[ $first -eq 0 ]] && printf ',\n'
    first=0
    printf '  {"file":"%s","line":%s,"text":"%s"}' "${file#${ROOT}/}" "$lineno" "$esc"
  done
  printf '\n]\n'
else
  for r in "${rows[@]}"; do
    printf '%s\n' "${r#${ROOT}/}"
  done
fi
```

- [ ] **Step 8.2: Run**

```bash
chmod +x scripts/inventory/scan-binary-hardcoding.sh
scripts/inventory/scan-binary-hardcoding.sh
```

Expected: 7 lines across `cmd_init.sh`, `cmd_start.sh`, `cmd_stop.sh`, `cmd_node.sh` (see `docs/VISION_AND_ROADMAP.md` §3 — "7곳 잔존").

- [ ] **Step 8.3: Write `docs/HARDCODING_AUDIT.md`**

```markdown
# Chain Binary Hardcoding Audit

> **Status:** baseline for the adapter-extraction work in `docs/VISION_AND_ROADMAP.md` §5.12 M4.
> Regenerate with:
>
>     scripts/inventory/scan-binary-hardcoding.sh

## What this is

Every site in `lib/cmd_*.sh` that names the current chain binary (`gstable`)
explicitly, rather than going through the adapter. Each of these has to be
refactored before any second chain (wbft, wemix, ethereum) can be supported.

## Findings

<paste output of scripts/inventory/scan-binary-hardcoding.sh here and keep in sync>

## Classification

| # | Site | Category | Proposed replacement |
|---|------|----------|----------------------|
| 1 | `cmd_init.sh:113` — `_CHAIN_TYPE="${CHAINBENCH_CHAIN_TYPE:-stablenet}"` | Default selection | Keep default (profile override already works) |
| 2 | `cmd_init.sh:183–192` — "Run gstable init" comment + error message | Log cosmetics | Reference `${_BINARY_NAME}` from adapter |
| 3 | `cmd_start.sh:220` — launch comment | Log cosmetics | Reference adapter binary name |
| 4 | `cmd_stop.sh:14` — `_BINARY_NAME="${CHAINBENCH_BINARY:-gstable}"` | Default name for `pkill` | Fetch from active network's adapter (`adapter_binary_name`) |
| 5 | `cmd_node.sh:231–234` — doc comments | Log cosmetics | Update after adapter name wired |
| 6 | `cmd_node.sh:262` — `binary = node.get("binary","gstable")` | Runtime fallback | Read from `state/pids.json` written by adapter-aware start |
| 7 | `cmd_node.sh:356` — doc comment | Log cosmetics | Update with adapter name |

## Exit criteria for this audit

This document is considered closed when:

1. `scripts/inventory/scan-binary-hardcoding.sh` returns zero lines
2. Every replacement in the table above has been implemented in a commit
3. `chainbench init --profile <non-stablenet>` succeeds end-to-end with a non-stub adapter
```

- [ ] **Step 8.4: Paste current scan output into the doc**

Manually copy the `scripts/inventory/scan-binary-hardcoding.sh` output into the `## Findings` section (replace the `<paste output...>` marker).

- [ ] **Step 8.5: Commit**

```bash
git add scripts/inventory/scan-binary-hardcoding.sh docs/HARDCODING_AUDIT.md
git commit -m "docs: add audit of chain binary name hardcoding"
```

---

## Final verification

- [ ] **Run the full Go suite from the Network module**

```bash
cd network && go test ./... && cd ..
```
Expected: PASS across `schema`, `cmd/chainbench-net`, `internal/types`.

- [ ] **Build the Network binary**

```bash
cd network && go build -o ../bin/chainbench-net ./cmd/chainbench-net && cd ..
./bin/chainbench-net version
```
Expected: prints `chainbench-net 0.0.0-dev`.

- [ ] **Run both inventory scripts once and confirm docs are in sync**

```bash
scripts/inventory/list-adapter-functions.sh       | diff -q - <(sed -n '/^## 2/,/^## 3/p' docs/ADAPTER_CONTRACT.md | head -n -1 | tail -n +3) || echo "Adapter doc out of sync — re-paste"
scripts/inventory/scan-binary-hardcoding.sh       | diff -q - <(sed -n '/^## Findings/,/^## Classification/p' docs/HARDCODING_AUDIT.md | head -n -1 | tail -n +3) || echo "Hardcoding doc out of sync — re-paste"
```
(These `diff -q` calls are best-effort sanity checks; manual paste is authoritative.)

- [ ] **Review commit list**

```bash
git log --oneline main..HEAD
```
Expected 8 commits, one per task:
1. `network: initialize Go module skeleton`
2. `network: add cobra CLI entry with version subcommand`
3. `network: add network JSON schema with fixtures and validator`
4. `network: add command JSON schema with fixtures`
5. `network: add event JSON schema with catalog and fixtures`
6. `network: generate Go types from JSON schemas via go:generate`
7. `docs: add adapter contract with function inventory`
8. `docs: add audit of chain binary name hardcoding`

---

## Out of scope (next plan)

The following belong to Sprint 2+ and are intentionally not implemented here:

- LocalDriver / RemoteDriver implementation
- Wire protocol (stdin command envelope + stdout NDJSON handshake) beyond the schemas
- bash `lib/network_client.sh` wrapper
- Any modification of existing `cmd_*.sh` files
- `install.sh` changes to build the Network binary
- MCP server refactor to consume Network Abstraction

Each of those gets its own plan once this foundation is merged.
