package session

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

// CapturedSession contains only evidence files. It never includes keys, node
// databases, binaries or files reached through links in the artifact tree.
type CapturedSession struct {
	Result Result            `json:"result"`
	Files  map[string]string `json:"files"`
	Gaps   []string          `json:"gaps"`
}

// Capture reads a bounded evidence snapshot from a legacy or grouped session.
// The caller owns persistence and registered-secret redaction of this copy.
func Capture(root, id string) (CapturedSession, error) {
	out := CapturedSession{Files: map[string]string{}, Gaps: []string{}}
	parts := strings.Split(id, "/")
	if len(parts) > 2 {
		return out, errors.New("invalid session reference")
	}
	for _, part := range parts {
		if !fs.ValidPath(part) || part == "." || strings.Contains(part, `\`) {
			return out, errors.New("invalid session reference")
		}
	}
	base, err := os.OpenRoot(root)
	if err != nil {
		return out, err
	}
	defer func() { _ = base.Close() }()
	for i := range parts {
		info, err := base.Lstat(path.Join(parts[:i+1]...))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return out, errors.New("session must be an owned directory without links")
		}
	}
	run, err := base.OpenRoot(id)
	if err != nil {
		return out, err
	}
	defer func() { _ = run.Close() }()
	total := 0
	err = fs.WalkDir(run.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			out.Gaps = append(out.Gaps, "Linked artifact excluded: "+name)
			return nil
		}
		if entry.IsDir() {
			if evidenceDirectory(name) {
				return nil
			}
			return fs.SkipDir
		}
		if !evidenceFile(name) {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("invalid evidence file")
		}
		if info.Size() > 2<<20 || total+int(info.Size()) > 8<<20 {
			out.Gaps = append(out.Gaps, "Oversized artifact excluded: "+name)
			return nil
		}
		f, err := run.Open(name)
		if err != nil {
			return err
		}
		opened, statErr := f.Stat()
		if statErr != nil || !os.SameFile(info, opened) {
			_ = f.Close()
			return errors.New("artifact changed during capture")
		}
		b, readErr := io.ReadAll(io.LimitReader(f, (2<<20)+1))
		_ = f.Close()
		if readErr != nil || len(b) > 2<<20 || total+len(b) > 8<<20 {
			return errors.New("artifact exceeds capture limit")
		}
		if !validEvidenceJSON(name, b) {
			out.Gaps = append(out.Gaps, "Malformed JSON artifact excluded: "+name)
			return nil
		}
		if name == fileSession {
			if err = json.Unmarshal(b, &out.Result); err != nil {
				return err
			}
		}
		total += len(b)
		out.Files[name] = string(b)
		return nil
	})
	if err != nil {
		return out, err
	}
	if _, found := out.Files[fileSession]; !found || out.Result.ID == "" {
		return out, errors.New("session verdict missing")
	}
	return out, nil
}

func validEvidenceJSON(name string, b []byte) bool {
	if strings.HasSuffix(name, ".json") {
		return json.Valid(b)
	}
	if strings.HasSuffix(name, ".jsonl") {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.TrimSpace(line) != "" && !json.Valid([]byte(line)) {
				return false
			}
		}
	}
	return true
}

func evidenceDirectory(name string) bool {
	p := strings.Split(name, "/")
	return name == "." || name == dirTests || name == dirEnvironments ||
		(len(p) == 2 && (p[0] == dirTests || p[0] == dirEnvironments)) ||
		(len(p) == 3 && p[0] == dirEnvironments && p[2] == dirChainstate)
}

func evidenceFile(name string) bool {
	p := strings.Split(name, "/")
	if name == fileSession {
		return true
	}
	if len(p) == 3 && p[0] == dirTests {
		switch p[2] {
		case fileSpec, fileSteps, fileAssert, fileStatus, filePostAction, fileArtifacts, fileEnvRef:
			return true
		}
	}
	return (len(p) == 3 && p[0] == dirEnvironments && p[2] == fileEnv) ||
		(len(p) == 4 && p[0] == dirEnvironments && p[2] == dirChainstate && strings.HasSuffix(p[3], ".jsonl"))
}
