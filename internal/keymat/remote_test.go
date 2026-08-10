package keymat_test

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/internal/keymat"
)

func TestRemoteFileSource_InjectedReader(t *testing.T) {
	// raw hex remote file
	a, _ := keymat.RandomSource{}.Resolve(context.Background())
	rawHex := "0x" + hex.EncodeToString(a.PrivateKeyBytes())
	src := keymat.RemoteFileSource{
		Path: "/k",
		Read: func(context.Context) ([]byte, error) { return []byte(rawHex), nil },
	}
	got, err := src.Resolve(context.Background())
	if err != nil {
		t.Fatalf("resolve raw: %v", err)
	}
	if got.Address().Hex() != a.Address().Hex() {
		t.Fatalf("raw remote import mismatch")
	}

	// keystore JSON remote file (needs password)
	ksPath, _ := (keymat.KeystoreStore{ScryptN: 1 << 12, ScryptP: 1}).Save(t.TempDir(), "k", a, keymat.StaticPassword("pw"))
	ksData := mustRead(t, ksPath)
	ksSrc := keymat.RemoteFileSource{
		Path: "/k.json", Password: keymat.StaticPassword("pw"),
		Read: func(context.Context) ([]byte, error) { return ksData, nil },
	}
	got2, err := ksSrc.Resolve(context.Background())
	if err != nil || got2.Address().Hex() != a.Address().Hex() {
		t.Fatalf("keystore remote import: %v", err)
	}

	// read error propagates
	bad := keymat.RemoteFileSource{Path: "/k", Read: func(context.Context) ([]byte, error) { return nil, errors.New("ssh down") }}
	if _, err := bad.Resolve(context.Background()); err == nil {
		t.Fatal("expected read error to propagate")
	}
}
