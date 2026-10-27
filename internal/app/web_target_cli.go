package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// webTargetReader is an adapter to typed command observations. No transport
// or process driver is opened in the dashboard; each call has private inputs.
type webTargetReader struct {
	read  func(context.Context, ResourceReadIn) (ResourceReadOut, error)
	check func(context.Context, string) error
}

// observationSnapshot reuses successful identity and file observations within
// one read-only response. A new response gets a new snapshot; execution never
// uses it. Process and log reads remain live, and every reuse rechecks access.
func (r *webTargetReader) observationSnapshot() *webTargetReader {
	if r == nil || r.read == nil {
		return r
	}
	var mu sync.Mutex
	cache := map[string]ResourceReadOut{}
	clone := func(out ResourceReadOut) ResourceReadOut {
		out.Exists = maps.Clone(out.Exists)
		out.Checksums = maps.Clone(out.Checksums)
		if out.Target != nil {
			copy := *out.Target
			out.Target = &copy
		}
		return out
	}
	return &webTargetReader{check: r.check, read: func(ctx context.Context, in ResourceReadIn) (ResourceReadOut, error) {
		if err := r.CheckServer(ctx, in.Target.Server); err != nil {
			return ResourceReadOut{}, err
		}
		if in.Process != nil || in.Log != nil {
			return r.query(ctx, in)
		}
		raw, err := json.Marshal(in)
		if err != nil {
			return ResourceReadOut{}, err
		}
		mu.Lock()
		defer mu.Unlock()
		if err := r.CheckServer(ctx, in.Target.Server); err != nil {
			return ResourceReadOut{}, err
		}
		if out, ok := cache[string(raw)]; ok {
			return clone(out), nil
		}
		out, err := r.query(ctx, in)
		if err == nil {
			cache[string(raw)] = clone(out)
		}
		return out, err
	}}
}

func (e *WebChainEngine) inspectTarget(ctx context.Context, a DeploymentActor, p webChainPayload, spec resource.Spec) (resource.Inspection, error) {
	r, err := e.targetReader(ctx, a, p)
	if err != nil {
		return resource.Inspection{}, err
	}
	return r.Inspect(ctx, spec)
}

func (e *WebChainEngine) targetReader(ctx context.Context, a DeploymentActor, p webChainPayload) (*webTargetReader, error) {
	if e.cli == nil {
		return nil, errNoChainCLI
	}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings)
	if err != nil {
		return nil, err
	}
	r := &webTargetReader{}
	r.check = func(ctx context.Context, server string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := e.allowed(a); err != nil {
			return err
		}
		if server == "" {
			return nil
		}
		c, err := lookup(server)
		if err != nil {
			return err
		}
		if c.BeforeDial != nil {
			return c.BeforeDial()
		}
		return nil
	}
	r.read = func(ctx context.Context, in ResourceReadIn) (out ResourceReadOut, err error) {
		if err = resource.ValidateNamedRead(in.Target); err != nil {
			return out, err
		}
		guard := func() error { return r.check(ctx, in.Target.Server) }
		if err = guard(); err != nil {
			return out, err
		}
		dir, err := os.MkdirTemp(e.root, "target-read-")
		if err != nil {
			return out, err
		}
		defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
		logins, err := e.documents.writeJobServerSet(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings, filepath.Join(dir, "secrets"))
		if err != nil {
			return out, err
		}
		in.ServerSet = logins
		// A docker pool is reached through the localmap written beside logins.
		in.Docker = webDocker(p.Set.DeploymentDocumentInput)
		raw, err := json.Marshal(in)
		if err != nil {
			return out, err
		}
		path := filepath.Join(dir, "request.json")
		if err = (filestore.Local{}).Write(ctx, path, raw, 0600); err != nil {
			return out, err
		}
		args := []string{"resource", "inspect", "--json", "--request", path}
		if logins != "" {
			args = append(args, "--server-logins", logins)
		}
		stdout, err := runCLIWithAccessGuard(ctx, e.cli, guard, args)
		if err != nil {
			return out, err
		}
		if err = json.Unmarshal(stdout, &out); err != nil {
			return out, fmt.Errorf("chainbench target read: invalid output: %w", err)
		}
		return out, nil
	}
	return r, nil
}

// runCLIWithAccessGuard cancels a command whose account or credential was
// revoked. The runner waits for the command to exit before secret cleanup.
func runCLIWithAccessGuard(ctx context.Context, cli ChainCLI, guard func() error, args []string) ([]byte, error) {
	if err := guard(); err != nil {
		return nil, err
	}
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	checked := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				checked <- nil
				return
			case <-child.Done():
				checked <- nil
				return
			case <-ticker.C:
				if err := guard(); err != nil {
					cancel()
					checked <- err
					return
				}
			}
		}
	}()
	raw, err := cli.Run(child, args...)
	close(done)
	err = errors.Join(err, <-checked, guard())
	return raw, err
}

func (r *webTargetReader) CheckServer(ctx context.Context, server string) error {
	if r == nil || r.check == nil {
		return errNoChainCLI
	}
	return r.check(ctx, server)
}

func (r *webTargetReader) query(ctx context.Context, in ResourceReadIn) (ResourceReadOut, error) {
	if r == nil || r.read == nil {
		return ResourceReadOut{}, errNoChainCLI
	}
	return r.read(ctx, in)
}

func (r *webTargetReader) Inspect(ctx context.Context, spec resource.Spec) (resource.Inspection, error) {
	out, err := r.query(ctx, ResourceReadIn{ReadRequest: resource.ReadRequest{Target: spec, Inspect: true}})
	if err != nil {
		return resource.Inspection{}, err
	}
	if out.Target == nil || out.Target.HostIdentity == "" {
		return resource.Inspection{}, errors.New("chainbench target read: missing identity")
	}
	return *out.Target, nil
}

type webTargetAccess struct {
	reader *webTargetReader
	spec   resource.Spec
	Files  webTargetFiles
}

func (r *webTargetReader) Open(spec resource.Spec) (*webTargetAccess, error) {
	if r == nil || r.read == nil {
		return nil, errNoChainCLI
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return &webTargetAccess{reader: r, spec: spec, Files: webTargetFiles{r, spec}}, nil
}

func (a *webTargetAccess) VerifyRegularFile(ctx context.Context, path string) error {
	_, err := a.reader.query(ctx, ResourceReadIn{ReadRequest: resource.ReadRequest{Target: a.spec, Regular: []string{path}}})
	return err
}

func (a *webTargetAccess) VerifyExecutable(ctx context.Context, path string) error {
	_, err := a.reader.query(ctx, ResourceReadIn{ReadRequest: resource.ReadRequest{Target: a.spec, Executable: []string{path}}})
	return err
}

func (a *webTargetAccess) Observe(ctx context.Context, in resource.ProcessRead) (process.LaunchVerdict, error) {
	out, err := a.reader.query(ctx, ResourceReadIn{ReadRequest: resource.ReadRequest{Target: a.spec, Process: &in}})
	if err != nil {
		return process.LaunchVerdict{}, err
	}
	if out.Process == nil {
		return process.LaunchVerdict{}, errors.New("chainbench target read: missing process observation")
	}
	return *out.Process, nil
}

type webTargetFiles struct {
	reader *webTargetReader
	spec   resource.Spec
}

func (f webTargetFiles) Exists(ctx context.Context, path string) (bool, error) {
	out, err := f.reader.query(ctx, ResourceReadIn{ReadRequest: resource.ReadRequest{Target: f.spec, Exists: []string{path}}})
	if err != nil {
		return false, err
	}
	value, ok := out.Exists[path]
	if !ok {
		return false, errors.New("chainbench target read: missing existence observation")
	}
	return value, nil
}

func (f webTargetFiles) Checksum(ctx context.Context, path string) (string, error) {
	out, err := f.reader.query(ctx, ResourceReadIn{ReadRequest: resource.ReadRequest{Target: f.spec, Checksums: []string{path}}})
	if err != nil {
		return "", err
	}
	value, ok := out.Checksums[path]
	if !ok || value == "" {
		return "", errors.New("chainbench target read: missing checksum")
	}
	return value, nil
}

type webCLILogs struct {
	reader *webTargetReader
	spec   resource.Spec
}

func (l webCLILogs) Snapshot(ctx context.Context, path string, offset int64, limit int) (webLogSnapshot, error) {
	out, err := l.reader.query(ctx, ResourceReadIn{ReadRequest: resource.ReadRequest{Target: l.spec, Log: &resource.LogRead{Path: path, Offset: offset, Limit: limit}}})
	if err != nil {
		return webLogSnapshot{}, err
	}
	if out.LogMissing {
		return webLogSnapshot{}, os.ErrNotExist
	}
	if out.Log == nil {
		return webLogSnapshot{}, errors.New("chainbench target read: missing log snapshot")
	}
	s := out.Log
	return webLogSnapshot{ID: s.ID, Size: s.Size, Head: s.Head, Data: s.Data}, nil
}
