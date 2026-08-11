package session_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/keyreg"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// fixedTime is a deterministic start time for reproducible session ids.
var fixedTime = time.Date(2026, 8, 6, 14, 30, 5, 0, time.UTC)

func newSession(t *testing.T) session.Session {
	t.Helper()
	s, err := session.New(t.TempDir(), "chainbench test --suite gov", fixedTime, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestSessionID_Format(t *testing.T) {
	if got := session.SessionID(fixedTime); got != "UTC-20260806-143005" {
		t.Fatalf("SessionID = %q, want UTC-20260806-143005", got)
	}
}

func TestNew_CreatesLayout(t *testing.T) {
	s := newSession(t)
	if s.ID() != "UTC-20260806-143005" {
		t.Fatalf("ID = %q", s.ID())
	}
	if filepath.Base(s.Root()) != s.ID() {
		t.Fatalf("Root %q must end in ID %q", s.Root(), s.ID())
	}
	for _, sub := range []string{"keys", "environments", "tests"} {
		if fi, err := os.Stat(filepath.Join(s.Root(), sub)); err != nil || !fi.IsDir() {
			t.Fatalf("missing dir %s: %v", sub, err)
		}
	}
}

func TestEnvID_Derivation(t *testing.T) {
	s := newSession(t)
	fp := session.Fingerprint("a1b2c3d4e5f67890deadbeefcafef00d")
	env, err := s.NewEnvironment(fp)
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	if env.ID() != "env-a1b2c3d4e5f6" {
		t.Fatalf("env id = %q, want env-a1b2c3d4e5f6", env.ID())
	}
	if filepath.Base(env.Dir()) != env.ID() {
		t.Fatalf("env Dir %q must end in id", env.Dir())
	}
	for _, sub := range []string{"logs", "chainstate", "nodes"} {
		if fi, err := os.Stat(filepath.Join(env.Dir(), sub)); err != nil || !fi.IsDir() {
			t.Fatalf("env missing dir %s: %v", sub, err)
		}
	}
	if env.LogPath("bp1") != filepath.Join(env.Dir(), "logs", "bp1.log") {
		t.Fatalf("LogPath = %q", env.LogPath("bp1"))
	}
}

func TestEnvironment_Reuse(t *testing.T) {
	s := newSession(t)
	fpA := session.Fingerprint("aaaaaaaaaaaa1111")
	fpB := session.Fingerprint("bbbbbbbbbbbb2222")

	if _, ok := s.Environment(fpA); ok {
		t.Fatal("Environment must be absent before creation")
	}
	envA, err := s.NewEnvironment(fpA)
	if err != nil {
		t.Fatalf("NewEnvironment A: %v", err)
	}
	got, ok := s.Environment(fpA)
	if !ok || got.ID() != envA.ID() {
		t.Fatalf("Environment(fpA) reuse failed: ok=%v", ok)
	}
	if _, ok := s.Environment(fpB); ok {
		t.Fatal("different fingerprint must not resolve")
	}
	// Two different configs -> two env dirs (F1 AC-2).
	if _, err := s.NewEnvironment(fpB); err != nil {
		t.Fatalf("NewEnvironment B: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(s.Root(), "environments"))
	if len(entries) != 2 {
		t.Fatalf("environments dir has %d entries, want 2", len(entries))
	}
}

func TestTestRecord_PathAndStatus(t *testing.T) {
	s := newSession(t)
	rec := s.Test(1, "GOV-005")
	if filepath.Base(rec.Dir()) != "001_GOV-005" {
		t.Fatalf("test dir = %q, want 001_GOV-005", rec.Dir())
	}
	rec.SetEnvRef("env-a1b2c3d4e5f6")
	rec.Status(session.StatusPass)

	if b, err := os.ReadFile(filepath.Join(rec.Dir(), "env-ref")); err != nil || string(b) != "env-a1b2c3d4e5f6" {
		t.Fatalf("env-ref = %q err=%v", b, err)
	}
	var status map[string]any
	b, err := os.ReadFile(filepath.Join(rec.Dir(), "status.json"))
	if err != nil {
		t.Fatalf("read status.json: %v", err)
	}
	if err := json.Unmarshal(b, &status); err != nil {
		t.Fatalf("status.json invalid: %v", err)
	}
	if status["result"] != "pass" {
		t.Fatalf("status result = %v, want pass", status["result"])
	}
}

func TestEnvironment_Resolve(t *testing.T) {
	s := newSession(t)
	env, _ := s.NewEnvironment(session.Fingerprint("cccccccccccc3333"))
	env.PopulateNodeTable(node.NodeSet{Nodes: []node.Node{
		{Index: 1, Role: node.RoleValidator, RPCURL: "http://a"},
		{Index: 2, Role: node.RoleValidator, RPCURL: "http://b"},
		{Index: 3, Role: node.RoleEN, RPCURL: "http://c"},
	}})

	cases := []struct {
		sel  string
		want string // expected RPCURL
	}{
		{"bp1", "http://a"},
		{"bp2", "http://b"},
		{"bp:any", "http://a"},
		{"en:0", "http://c"},
		{"en1", "http://c"},
	}
	for _, c := range cases {
		n, err := env.Resolve(c.sel)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", c.sel, err)
		}
		if n.RPCURL != c.want {
			t.Fatalf("Resolve(%q) = %q, want %q", c.sel, n.RPCURL, c.want)
		}
	}
	if _, err := env.Resolve("bp9"); err == nil {
		t.Fatal("out-of-range selector must error")
	}
	if _, err := env.Resolve("xyz1"); err == nil {
		t.Fatal("unknown role must error")
	}
}

func TestSave_WritesJSON(t *testing.T) {
	s := newSession(t)
	env, _ := s.NewEnvironment(session.Fingerprint("dddddddddddd4444"))
	env.PopulateNodeTable(node.NodeSet{Chain: "wbft", Nodes: []node.Node{
		{Index: 1, Role: node.RoleValidator, Host: "127.0.0.1", RPCURL: "http://a"},
	}})
	if err := env.Save(); err != nil {
		t.Fatalf("env.Save: %v", err)
	}
	if err := s.Save(); err != nil {
		t.Fatalf("session.Save: %v", err)
	}

	var envDoc map[string]any
	b, err := os.ReadFile(filepath.Join(env.Dir(), "env.json"))
	if err != nil {
		t.Fatalf("read env.json: %v", err)
	}
	if err := json.Unmarshal(b, &envDoc); err != nil {
		t.Fatalf("env.json invalid: %v", err)
	}
	if envDoc["fingerprint"] != "dddddddddddd4444" {
		t.Fatalf("env.json fingerprint = %v", envDoc["fingerprint"])
	}

	sb, err := os.ReadFile(filepath.Join(s.Root(), "session.json"))
	if err != nil {
		t.Fatalf("read session.json: %v", err)
	}
	var sesDoc map[string]any
	if err := json.Unmarshal(sb, &sesDoc); err != nil {
		t.Fatalf("session.json invalid: %v", err)
	}
	if sesDoc["id"] != "UTC-20260806-143005" {
		t.Fatalf("session.json id = %v", sesDoc["id"])
	}
}

func TestNewWithKeys_RegistryIsRootedInTheSession(t *testing.T) {
	base := t.TempDir()
	s, err := session.NewWithKeys(base, "chainbench run", fixedTime, keyreg.Deps{
		Generate: func() ([]byte, string, error) { return []byte{0x01, 0x02}, "0xabc", nil },
	})
	if err != nil {
		t.Fatalf("NewWithKeys: %v", err)
	}
	reg := s.Keys()
	if reg == nil {
		t.Fatal("Keys() is nil — the session did not build a registry")
	}

	if _, err := reg.Ensure(context.Background(), "op1", keyreg.Random, "", keyreg.EnsureOpts{}); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	// The point of session-owned keys is that callers never derive the path:
	// material must land inside this session's tree.
	addr := filepath.Join(s.Root(), "keys", "op1", "address")
	if _, err := os.Stat(addr); err != nil {
		t.Fatalf("key was not persisted under the session: %v", err)
	}
}

func TestNew_WithoutKeysLeavesRegistryNil(t *testing.T) {
	s := newSession(t)
	if s.Keys() != nil {
		t.Error("New(..., nil) should leave the registry unset")
	}
}
