package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AssetFiles publishes a receipt and its immutable bytes together. Browser file
// names never choose a storage path; unfinished uploads remain invisible.
type AssetFiles struct{ root string }
type AssetStage struct {
	Directory, Path, Checksum string
	Bytes                     int64
}

func OpenAssetFiles(root string) (*AssetFiles, error) {
	dir := filepath.Join(root, "assets")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("asset directory must be private and unlinked")
	}
	return &AssetFiles{root: dir}, nil
}

func (s *AssetFiles) Stage(r io.Reader, limit int64) (AssetStage, error) {
	var out AssetStage
	dir, err := os.MkdirTemp(s.root, ".upload-")
	if err != nil {
		return out, err
	}
	out.Directory, out.Path = dir, filepath.Join(dir, "payload")
	f, err := os.OpenFile(out.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		s.Discard(out)
		return AssetStage{}, err
	}
	h := sha256.New()
	out.Bytes, err = io.Copy(io.MultiWriter(f, h), io.LimitReader(r, limit+1))
	if err == nil && (out.Bytes == 0 || out.Bytes > limit) {
		err = errors.New("asset is empty or exceeds its size limit")
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		s.Discard(out)
		return AssetStage{}, err
	}
	out.Checksum = hex.EncodeToString(h.Sum(nil))
	return out, nil
}

func (s *AssetFiles) Discard(stage AssetStage) {
	if filepath.Dir(stage.Directory) == s.root && strings.HasPrefix(filepath.Base(stage.Directory), ".upload-") {
		_ = os.RemoveAll(stage.Directory)
	}
}

func NewAssetID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func validAssetID(id string) bool {
	if len(id) != 32 || id != strings.ToLower(id) {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func (s *AssetFiles) Publish(stage AssetStage, id string, receipt []byte, executable bool) error {
	if !validAssetID(id) || filepath.Dir(stage.Directory) != s.root || !strings.HasPrefix(filepath.Base(stage.Directory), ".upload-") {
		return errors.New("invalid asset publication")
	}
	if executable {
		if err := os.Chmod(stage.Path, 0700); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(stage.Directory, "receipt.json"), receipt, 0600); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(stage.Directory, "receipt.json"), os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = f.Sync()
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(stage.Directory, filepath.Join(s.root, id))
}

// Read refuses links, oversized receipts and replacement bytes. Metadata is
// useful only while its content-addressed payload still matches the receipt.
func (s *AssetFiles) Read(id string) ([]byte, string, string, int64, error) {
	if !validAssetID(id) {
		return nil, "", "", 0, os.ErrNotExist
	}
	dir := filepath.Join(s.root, id)
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, "", "", 0, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, "", "", 0, errors.New("invalid asset directory")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, "", "", 0, err
	}
	defer func() { _ = root.Close() }()
	read := func(name string, limit int64) ([]byte, error) {
		info, err := root.Lstat(name)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > limit {
			return nil, errors.New("invalid asset file")
		}
		f, err := root.Open(name)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		opened, err := f.Stat()
		if err != nil || !os.SameFile(info, opened) {
			return nil, errors.New("asset changed during read")
		}
		return io.ReadAll(io.LimitReader(f, limit+1))
	}
	receipt, err := read("receipt.json", 64<<10)
	if err != nil {
		return nil, "", "", 0, err
	}
	info, err = root.Lstat("payload")
	if err != nil {
		return nil, "", "", 0, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 256<<20 {
		return nil, "", "", 0, errors.New("invalid asset payload")
	}
	f, err := root.Open("payload")
	if err != nil {
		return nil, "", "", 0, err
	}
	defer func() { _ = f.Close() }()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, "", "", 0, errors.New("asset changed during read")
	}
	h := sha256.New()
	size, err := io.Copy(h, io.LimitReader(f, (256<<20)+1))
	if err != nil || size > 256<<20 {
		return nil, "", "", 0, errors.New("cannot verify asset payload")
	}
	return receipt, filepath.Join(dir, "payload"), hex.EncodeToString(h.Sum(nil)), size, nil
}

func (s *AssetFiles) IDs() ([]string, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, e := range entries {
		if validAssetID(e.Name()) {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}
