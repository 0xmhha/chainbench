package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ManifestFiles owns Web manifest declarations, audit records and selection pins.
// Node databases are still written exclusively by the composition engine.
type ManifestFiles struct{ root string }

func OpenManifestFiles(root string) (*ManifestFiles, error) {
	if err := os.MkdirAll(filepath.Join(root, "manifests"), 0700); err != nil {
		return nil, err
	}
	return &ManifestFiles{root: root}, nil
}

func (s *ManifestFiles) Read(id string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.root, "manifests", id+".json"))
}
func (s *ManifestFiles) IDs() ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(s.root, "manifests", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	ids := []string{}
	for _, path := range paths {
		base := filepath.Base(path)
		ids = append(ids, base[:len(base)-5])
	}
	return ids, nil
}
func (s *ManifestFiles) Save(id string, raw []byte) error {
	dir := filepath.Join(s.root, "manifests")
	tmp, err := os.CreateTemp(dir, ".manifest-")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err = tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, id+".json"))
}
func (s *ManifestFiles) NewSetupDir() (string, error) {
	return os.MkdirTemp(filepath.Join(s.root, "manifests"), "setup-")
}
func (s *ManifestFiles) Pin(dir string, manifest, template []byte) (string, string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", "", err
	}
	mp, tp := filepath.Join(dir, "selected-manifest.json"), filepath.Join(dir, "selected-template.json")
	if err := os.WriteFile(mp, manifest, 0600); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(tp, template, 0600); err != nil {
		return "", "", err
	}
	return mp, tp, nil
}
func (s *ManifestFiles) Audit(actor, operation, target string, status int) error {
	f, err := os.OpenFile(filepath.Join(s.root, "manifest-audit.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	record := map[string]any{"actorId": actor, "operation": operation, "target": target, "status": status, "at": time.Now().UTC()}
	if err = json.NewEncoder(f).Encode(record); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
