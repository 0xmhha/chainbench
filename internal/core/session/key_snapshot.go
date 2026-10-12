package session

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const keyFileLimit = 1 << 20
const keyTreeLimit = 16 << 20

// keyCapture is a bounded capture used only for private execution inputs.
// It is never a shared document, API response or history artifact.
type keyCapture struct {
	Files map[string][]byte `json:"files"`
}

// CaptureKeys refuses links and special files rather than silently excluding
// material the engine might later read. OpenRoot prevents traversal outside it.
func CaptureKeys(ctx context.Context, directory string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("key source must be a directory without links")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	out := keyCapture{Files: map[string][]byte{}}
	total, entries := 0, 0
	err = fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		entries++
		if entries > 2048 {
			return errors.New("key source has too many entries")
		}
		if entry.Type()&os.ModeSymlink != 0 || strings.Contains(name, `\`) {
			return errors.New("key source contains a link or invalid path")
		}
		if entry.IsDir() {
			return nil
		}
		info, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > keyFileLimit || len(out.Files) >= 1024 {
			return errors.New("key source contains a special or oversized file")
		}
		file, err := root.Open(name)
		if err != nil {
			return err
		}
		opened, err := file.Stat()
		if err != nil || !os.SameFile(info, opened) {
			_ = file.Close()
			return errors.New("key file changed during capture")
		}
		data, readErr := io.ReadAll(io.LimitReader(file, keyFileLimit+1))
		closeErr := file.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if len(data) > keyFileLimit || total+len(data) > keyTreeLimit {
			return errors.New("key source exceeds capture limit")
		}
		total += len(data)
		out.Files[name] = data
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out.Files) == 0 {
		return nil, errors.New("key source is empty")
	}
	return json.Marshal(out)
}

// KeySnapshotFiles owns encrypted snapshots and the private material the
// existing engine reads. Accepted digests never name browser-provided paths.
type KeySnapshotFiles struct{ root string }

func OpenKeySnapshotFiles(root string) (*KeySnapshotFiles, error) {
	for _, dir := range []string{"key-snapshots", "key-material"} {
		name := filepath.Join(root, dir)
		if err := os.MkdirAll(name, 0700); err != nil {
			return nil, err
		}
		info, err := os.Lstat(name)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			return nil, errors.New("key snapshot directories must be private and unlinked")
		}
	}
	return &KeySnapshotFiles{root: root}, nil
}

func validKeyDigest(id string) bool {
	if len(id) != 64 || id != strings.ToLower(id) {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func (s *KeySnapshotFiles) Read(id string) ([]byte, error) {
	if !validKeyDigest(id) {
		return nil, errors.New("invalid key snapshot reference")
	}
	root, err := os.OpenRoot(filepath.Join(s.root, "key-snapshots"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	name := id + ".enc"
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 24<<20 {
		return nil, errors.New("invalid key snapshot file")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("key snapshot changed during read")
	}
	data, err := io.ReadAll(io.LimitReader(file, (24<<20)+1))
	if len(data) > 24<<20 {
		return nil, errors.New("key snapshot exceeds limit")
	}
	return data, err
}

func (s *KeySnapshotFiles) Save(id string, encrypted []byte) error {
	if !validKeyDigest(id) || len(encrypted) > 24<<20 {
		return errors.New("invalid key snapshot")
	}
	file, err := os.CreateTemp(filepath.Join(s.root, "key-snapshots"), ".snapshot-")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	if _, err = file.Write(encrypted); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), filepath.Join(s.root, "key-snapshots", id+".enc"))
}

// Materialize writes an authenticated capture for engine consumption. Callers
// must compare the resulting CaptureKeys digest before allowing engine effects.
func (s *KeySnapshotFiles) Materialize(ctx context.Context, id string, raw []byte) (string, error) {
	if !validKeyDigest(id) || len(raw) > 24<<20 {
		return "", errors.New("invalid key snapshot")
	}
	directory := filepath.Join(s.root, "key-material", id)
	if info, err := os.Lstat(directory); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			return "", errors.New("invalid materialized key directory")
		}
		return directory, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	var snapshot keyCapture
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return "", err
	}
	if len(snapshot.Files) == 0 || len(snapshot.Files) > 1024 {
		return "", errors.New("invalid key capture")
	}
	temporary, err := os.MkdirTemp(filepath.Join(s.root, "key-material"), ".material-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(temporary) }()
	total := 0
	for name, data := range snapshot.Files {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		total += len(data)
		if name == "." || !fs.ValidPath(name) || strings.Contains(name, `\`) || len(data) > keyFileLimit || total > keyTreeLimit {
			return "", errors.New("invalid key capture path or size")
		}
		path := filepath.Join(temporary, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return "", err
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return "", err
		}
		if _, err = file.Write(data); err != nil {
			_ = file.Close()
			return "", err
		}
		if err = file.Sync(); err != nil {
			_ = file.Close()
			return "", err
		}
		if err = file.Close(); err != nil {
			return "", err
		}
	}
	if err = os.Rename(temporary, directory); err != nil {
		return "", err
	}
	return directory, nil
}
