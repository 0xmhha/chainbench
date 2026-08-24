// Package app is the use-case layer between the two user surfaces (CLI, MCP)
// and the orchestration/core packages.
//
// One use case = one function. Inputs and outputs are plain structs that both
// cobra flag binding and MCP JSON-schema binding can target, so the two
// surfaces call the same function and cannot drift apart. Nothing here knows
// cobra or MCP types, and nothing here formats output — rendering (tabwriter,
// JSON, MCP text) is the surface's job.
package app

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/provision"
)

// Deps are the collaborators the use cases share, injected once at the surface
// boundary. No package-level state.
type Deps struct {
	// Clock supplies timestamps (workspace step stamps, session age); nil uses
	// time.Now.
	Clock func() time.Time
	// Env reads environment variables (remote-target credentials at resolve
	// time); nil uses os.Getenv. Injected so tests never depend on the process
	// environment.
	Env func(string) string
	// Driver resolves the transport used to control node processes of an
	// already-launched network; nil uses the local driver. Injected so the use
	// cases can be tested without spawning processes, and so a surface that
	// targets a remote host routes the same use case over SSH.
	Driver func() (driver.Driver, error)
	// Files resolves where a network's on-disk material lands; nil uses this
	// machine's filesystem.
	//
	// It is separate from Driver because the two can differ — a launch driven
	// over SSH still reads its keys from here — and because leaving it implicit
	// is how a remote provision came to write its genesis and configs to the
	// operator's own disk while shipping only the identities.
	Files func() (provision.FileStore, error)
}

// now reports the current time through the injected clock.
func (d Deps) now() time.Time {
	if d.Clock == nil {
		return time.Now()
	}
	return d.Clock()
}

// nodeDriver resolves the transport for node-process control, defaulting to
// this machine.
func (d Deps) nodeDriver() (driver.Driver, error) {
	if d.Driver == nil {
		return driver.NewLocalDriver(), nil
	}
	return d.Driver()
}

// files resolves where on-disk material lands, defaulting to this machine.
//
// When a driver ships files to another host but no store was named, that
// driver is the store: a caller that routed the processes to a host and said
// nothing about the files meant the files to follow.
func (d Deps) files() (provision.FileStore, error) {
	if d.Files != nil {
		return d.Files()
	}
	if d.Driver == nil {
		return provision.LocalFileStore{}, nil
	}
	drv, err := d.Driver()
	if err != nil {
		return nil, err
	}
	if fp, ok := drv.(driver.FileProvisioner); ok {
		return driverStore{fp}, nil
	}
	return provision.LocalFileStore{}, nil
}

// driverStore adapts a driver that ships files into a provision.FileStore.
//
// A driver knows how to put a file on the host it controls but not how to ask
// whether one is there or read it back, so those two answer for the shape of
// the seam rather than for the transport: a file is never assumed present, and
// reading is not offered.
type driverStore struct{ fp driver.FileProvisioner }

// Exists reports false: a driver has no probe, and claiming a file is present
// would make provisioning skip a write that never happened.
func (driverStore) Exists(context.Context, string) (bool, error) { return false, nil }

// Read is not available over a bare file provisioner.
func (driverStore) Read(_ context.Context, path string) ([]byte, error) {
	return nil, fmt.Errorf("app: this target can ship files but not read them back (%s)", path)
}

// Write ships the file to the host the driver controls.
func (s driverStore) Write(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	return s.fp.ProvisionFile(ctx, path, content, mode)
}
