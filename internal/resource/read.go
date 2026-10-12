package resource

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/process"
)

// ReadRequest describes target observations without shell commands or writes.
type ReadRequest struct {
	Target     Spec         `json:"target"`
	Inspect    bool         `json:"inspect,omitempty"`
	Exists     []string     `json:"exists,omitempty"`
	Checksums  []string     `json:"checksums,omitempty"`
	Regular    []string     `json:"regular,omitempty"`
	Executable []string     `json:"executable,omitempty"`
	Process    *ProcessRead `json:"process,omitempty"`
	Log        *LogRead     `json:"log,omitempty"`
}

// ValidateNamedRead permits a service to use local targets and named SSH
// logins, while refusing direct remote addresses that could bypass bindings.
func ValidateNamedRead(spec Spec) error {
	if err := spec.Validate(); err != nil {
		return err
	}
	if spec.IsRemote() && spec.Server == "" {
		return errors.New("resource: remote observations require a named server login")
	}
	return nil
}

// ProcessRead checks a recorded launch or vacancy through the ownership core.
type ProcessRead struct {
	PID        int      `json:"pid"`
	Binary     string   `json:"binary"`
	Args       []string `json:"args"`
	DataDir    string   `json:"dataDir,omitempty"`
	ConfigPath string   `json:"configPath,omitempty"`
}

// LogRead asks for a bounded snapshot; identity and bytes come from one read.
type LogRead struct {
	Path   string `json:"path"`
	Offset int64  `json:"offset"`
	Limit  int    `json:"limit"`
}

// LogSnapshot identifies a log across truncation, rotation and inode reuse.
type LogSnapshot struct {
	ID   uint64 `json:"id"`
	Size int64  `json:"size"`
	Head []byte `json:"head"`
	Data []byte `json:"data"`
}

// ReadResult contains only requested observations, never credentials.
type ReadResult struct {
	Target     *Inspection            `json:"target,omitempty"`
	Exists     map[string]bool        `json:"exists,omitempty"`
	Checksums  map[string]string      `json:"checksums,omitempty"`
	Process    *process.LaunchVerdict `json:"process,omitempty"`
	Log        *LogSnapshot           `json:"log,omitempty"`
	LogMissing bool                   `json:"logMissing,omitempty"`
}

func validateRead(r ReadRequest) error {
	if err := r.Target.Validate(); err != nil {
		return err
	}
	paths := append(append(append(append([]string{}, r.Exists...), r.Checksums...), r.Regular...), r.Executable...)
	if r.Log != nil {
		if r.Log.Offset < 0 || r.Log.Offset == math.MaxInt64 || r.Log.Limit < 1 || r.Log.Limit > 1<<20 {
			return errors.New("resource: log offset or limit is out of range")
		}
		paths = append(paths, r.Log.Path)
	}
	if len(paths) > 256 {
		return errors.New("resource: too many requested paths")
	}
	for _, p := range paths {
		if !filepath.IsAbs(p) || len(p) > 4096 || strings.ContainsAny(p, "\x00\r\n") {
			return errors.New("resource: reads require absolute, bounded file paths")
		}
	}
	if r.Process != nil {
		if r.Process.PID < 0 || !filepath.IsAbs(r.Process.Binary) || len(r.Process.Args) > 1024 {
			return errors.New("resource: invalid recorded launch")
		}
		for _, a := range r.Process.Args {
			if len(a) > 16384 || strings.ContainsRune(a, 0) {
				return errors.New("resource: invalid launch argument")
			}
		}
	}
	if !r.Inspect && len(paths) == 0 && r.Process == nil {
		return errors.New("resource: select an observation")
	}
	return nil
}

// Read observes target identity, files, logs or processes through one opener.
// It never opens a mutable composition, seeds a ledger or signals a process.
func (o Opener) Read(ctx context.Context, r ReadRequest) (ReadResult, error) {
	out := ReadResult{Exists: map[string]bool{}, Checksums: map[string]string{}}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if err := validateRead(r); err != nil {
		return out, err
	}
	access, err := o.Open(r.Target)
	if err != nil {
		return out, err
	}
	if r.Inspect {
		t, err := o.Inspect(ctx, r.Target)
		if err != nil {
			return out, err
		}
		out.Target = &t
	}
	for _, p := range r.Exists {
		b, err := access.Files.Exists(ctx, p)
		if err != nil {
			return out, err
		}
		out.Exists[p] = b
	}
	for _, p := range r.Regular {
		if err := access.VerifyRegularFile(ctx, p); err != nil {
			return out, err
		}
	}
	for _, p := range r.Executable {
		if err := access.VerifyExecutable(ctx, p); err != nil {
			return out, err
		}
	}
	for _, p := range r.Checksums {
		sum, err := access.Files.Checksum(ctx, p)
		if err != nil {
			return out, err
		}
		out.Checksums[p] = sum
	}
	if r.Process != nil {
		p := r.Process
		table := process.ProcessTableOf(ctx, access.Driver)
		var v process.LaunchVerdict
		if p.PID > 0 {
			v = process.ObserveLaunch(ctx, table, p.PID, p.Binary, p.Args)
		} else {
			v = process.ObserveVacancy(ctx, table, p.Binary, p.Args, p.DataDir, p.ConfigPath)
		}
		out.Process = &v
	}
	if r.Log != nil {
		s, err := readLogSnapshot(ctx, access, *r.Log)
		if errors.Is(err, ErrLogMissing) {
			out.LogMissing = true
		} else if err != nil {
			return out, err
		} else {
			out.Log = &s
		}
	}
	if err := ctx.Err(); err != nil {
		return out, fmt.Errorf("resource: read cancelled: %w", err)
	}
	return out, nil
}
