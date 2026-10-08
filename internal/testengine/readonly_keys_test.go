package testengine

import (
	"bytes"
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/session"
)

func TestReadOnlyAttachKeysPreserveAcceptedSource(t *testing.T) {
	dir := t.TempDir()
	if _, err := store.Generate(store.GenerateOpts{Out: dir, Nodes: 2, Password: "fixture", Derive: derive.WithBLS}, nil); err != nil {
		t.Fatal(err)
	}
	before, err := session.CaptureKeys(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	ring, err := ringForOutput(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []keyring.Label{"node1", "node2"} {
		entry, ok := ring.Get(label)
		if !ok || entry.Nodekey == (derive.PrivateKey{}) || entry.BLS == nil {
			t.Fatal("immutable inputs lost usable signing identities")
		}
	}
	for range 2 {
		if _, err = NewAttachEngine(AttachConfig{Chain: "stablenet", RPCURLs: []string{"http://127.0.0.1:1"}, KeysDir: dir, ReadOnlyKeys: true, ArtifactRoot: t.TempDir()}); err != nil {
			t.Fatal(err)
		}
		after, err := session.CaptureKeys(context.Background(), dir)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("reading test accounts mutated the accepted key inputs", err)
		}
	}
}
