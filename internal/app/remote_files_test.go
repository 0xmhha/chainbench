package app_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/driver"
)

// shippingDriver stands in for a host reached over SSH: it records what was
// shipped to it and never touches this machine's disk.
type shippingDriver struct{ shipped map[string][]byte }

func newShippingDriver() *shippingDriver {
	return &shippingDriver{shipped: map[string][]byte{}}
}

func (d *shippingDriver) Provision(context.Context, driver.NodeSpec) error { return nil }

func (d *shippingDriver) Launch(_ context.Context, s driver.NodeSpec) (driver.Handle, error) {
	return driver.Handle{PID: 1000 + s.Index}, nil
}

func (d *shippingDriver) Stop(context.Context, driver.Handle) error { return nil }

func (d *shippingDriver) ProvisionFile(_ context.Context, path string, b []byte, _ fs.FileMode) error {
	d.shipped[path] = b
	return nil
}

// names returns the shipped files' base names, sorted.
func (d *shippingDriver) names() []string {
	out := make([]string, 0, len(d.shipped))
	for p := range d.shipped {
		out = append(out, filepath.Base(p))
	}
	sort.Strings(out)
	return out
}

// TestRemoteProvision_LeavesNothingOnThisMachine is a regression.
//
// A network provisioned onto another host used to write its genesis and every
// per-node config to the operator's own disk: the driver shipped the identities
// because the launcher asks it to directly, but the file seam was never given a
// store, so it defaulted to this filesystem. The remote node then started
// against a datadir with no genesis in it.
func TestRemoteProvision_LeavesNothingOnThisMachine(t *testing.T) {
	root := t.TempDir()
	host := newShippingDriver()

	if _, err := app.NetworkProvision(context.Background(), app.Deps{
		Driver: func() (driver.Driver, error) { return host, nil },
	}, app.NetworkProvisionIn{
		Spec:    app.NetworkSpecIn{Chain: "stablenet", DataDir: root},
		KeysDir: presetDir,
	}); err != nil {
		t.Fatalf("NetworkProvision: %v", err)
	}

	if left := filesUnder(t, root); len(left) > 0 {
		t.Errorf("a remote provision wrote to the local disk: %v", left)
	}

	shipped := host.names()
	for _, want := range []string{"genesis.json", "config_node1.toml"} {
		if !contains(shipped, want) {
			t.Errorf("%s was not shipped to the host; shipped: %v", want, shipped)
		}
	}
}

// TestLocalProvision_StaysLocal is the other half: with no driver named, the
// same use case writes here, and nothing changed for the common case.
func TestLocalProvision_StaysLocal(t *testing.T) {
	root := t.TempDir()

	if _, err := app.NetworkProvision(context.Background(), app.Deps{}, app.NetworkProvisionIn{
		Spec:    app.NetworkSpecIn{Chain: "stablenet", DataDir: root},
		KeysDir: presetDir,
	}); err != nil {
		t.Fatalf("NetworkProvision: %v", err)
	}

	written := filesUnder(t, root)
	if !contains(written, "genesis.json") {
		t.Errorf("a local provision wrote no genesis; wrote: %v", written)
	}
}

// TestSaveTopology_FollowsTheNetwork keeps the layout file beside the genesis
// and configs it describes, rather than on whichever machine ran the command.
func TestSaveTopology_FollowsTheNetwork(t *testing.T) {
	root := t.TempDir()
	topo := filepath.Join(t.TempDir(), "topology.yaml")
	layout := "chain: stablenet\nnodes:\n  - index: 1\n    role: validator\n"
	if err := os.WriteFile(topo, []byte(layout), 0o644); err != nil {
		t.Fatal(err)
	}
	host := newShippingDriver()

	if _, err := app.NetworkProvision(context.Background(), app.Deps{
		Driver: func() (driver.Driver, error) { return host, nil },
	}, app.NetworkProvisionIn{
		Spec:    app.NetworkSpecIn{Chain: "stablenet", DataDir: root, TopologyPath: topo},
		KeysDir: presetDir,
	}); err != nil {
		t.Fatalf("NetworkProvision: %v", err)
	}

	if !contains(host.names(), "topology.yaml") {
		t.Errorf("the layout stayed on this machine; shipped: %v", host.names())
	}
	if left := filesUnder(t, root); len(left) > 0 {
		t.Errorf("a remote provision wrote to the local disk: %v", left)
	}
}

// filesUnder lists the files below root, relative to it.
func filesUnder(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(out)
	return out
}

func contains(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle || strings.HasSuffix(s, "/"+needle) {
			return true
		}
	}
	return false
}
