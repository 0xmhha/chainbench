package session

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes b to path through a temp file and a rename, so a
// concurrent reader never sees a half-written artifact and a failed write leaves
// any prior file intact. It is the one write primitive every session artifact
// goes through — session.json, env.json, the per-test records, the network
// registry, and the run report — so atomicity is not re-implemented per call
// site.
func WriteFileAtomic(path string, b []byte, perm fs.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// writeJSON marshals v indented and writes it atomically to path, scrubbing
// secrets first — it is the seam every JSON evidence artifact (the per-test
// records, env.json, session.json) goes through, so redaction is not
// re-implemented per record. The functional files a node reads do not go
// through here: workspace.json is written straight through WriteFileAtomic, and
// genesis/config land in the node's datadir via the filestore.
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal %s: %w", filepath.Base(path), err)
	}
	if err := WriteFileAtomic(path, Scrub(b), 0o644); err != nil {
		return fmt.Errorf("session: write %s: %w", filepath.Base(path), err)
	}
	return nil
}
