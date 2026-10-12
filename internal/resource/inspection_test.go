package resource_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestInspectionResolvesPhysicalPathWithoutCreatingTarget(t *testing.T) {
	root, alias := t.TempDir(), filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	first, err := (resource.Opener{}).Inspect(context.Background(), resource.Spec{DataRoot: filepath.Join(root, "not-created", "nodes")})
	if err != nil {
		t.Fatal(err)
	}
	second, err := (resource.Opener{}).Inspect(context.Background(), resource.Spec{DataRoot: filepath.Join(alias, "not-created", "nodes")})
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.OS != runtime.GOOS || first.Architecture != runtime.GOARCH || first.Transport != "local" || !strings.HasPrefix(first.HostIdentity, "machine-sha256:") {
		t.Fatal("aliases or platform did not resolve consistently", first, second)
	}
	if _, err = os.Stat(filepath.Join(root, "not-created")); !os.IsNotExist(err) {
		t.Fatal("inspection wrote target", err)
	}
	if _, err = (resource.Opener{}).Inspect(context.Background(), resource.Spec{DataRoot: "relative"}); err == nil {
		t.Fatal("relative root accepted")
	}
}

func TestPrivateOpenerNeverFallsBackToSharedCredentials(t *testing.T) {
	set := fixture(t)
	denied := errors.New("private lease unavailable")
	o := resource.Opener{ServerSet: set, Lookup: func(name string) (remote.Credentials, error) {
		if name != "box1" {
			t.Fatal("wrong server", name)
		}
		return remote.Credentials{}, denied
	}}
	if _, err := o.OpenPath("srv://box1/data"); !errors.Is(err, denied) {
		t.Fatal("shared-file fallback bypassed private resolver", err)
	}
	// Cached handles must still reject revoked access at the first real read.
	o.Lookup = func(string) (remote.Credentials, error) {
		return remote.Credentials{Host: "192.0.2.99", User: "private", Password: "private-material", HostKey: remote.HostKeyPolicy{InsecureHostKey: true}, BeforeDial: func() error { return denied }}, nil
	}
	acc, err := o.OpenPath("srv://box1/data")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = acc.Files.Exists(context.Background(), "/data"); err == nil || !strings.Contains(err.Error(), "authorization denied") {
		t.Fatal("revoked cached handle dialed", err)
	}
}
