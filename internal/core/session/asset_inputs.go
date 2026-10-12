package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
)

func privateAssetInput(dir, name string, limit int64) ([]byte, error) {
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("invalid asset input directory")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	info, err = root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > limit {
		return nil, errors.New("invalid asset input file")
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("asset input changed during read")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("asset input exceeds limit")
	}
	return data, nil
}

// MaterializeFile atomically pins a small registered file for an accepted
// composition. Existing changed snapshots are refused, never silently repaired.
func (s *AssetFiles) MaterializeFile(ctx context.Context, id, checksum, fingerprint string) (string, error) {
	if !validAssetID(id) || !validKeyDigest(checksum) || !validKeyDigest(fingerprint) {
		return "", errors.New("invalid asset input identity")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	data, err := privateAssetInput(filepath.Join(s.root, id), "payload", 16<<20)
	if err != nil {
		return "", errors.New("registered input unavailable")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != checksum {
		return "", errors.New("registered input checksum changed")
	}
	base := filepath.Join(filepath.Dir(s.root), "asset-material")
	parent := filepath.Join(base, fingerprint)
	for _, dir := range []string{base, parent} {
		if err = os.MkdirAll(dir, 0700); err != nil {
			return "", err
		}
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			return "", errors.New("invalid asset snapshot directory")
		}
	}
	directory := filepath.Join(parent, checksum)
	path := filepath.Join(directory, "input.json")
	verify := func() (string, error) {
		got, err := privateAssetInput(directory, "input.json", 16<<20)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(got)
		if hex.EncodeToString(sum[:]) != checksum {
			return "", errors.New("pinned asset input checksum changed")
		}
		return path, nil
	}
	if _, err = os.Lstat(directory); err == nil {
		return verify()
	} else if !os.IsNotExist(err) {
		return "", err
	}
	tmp, err := os.MkdirTemp(parent, ".input-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	f, err := os.OpenFile(filepath.Join(tmp, "input.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	if err = os.Rename(tmp, directory); err != nil {
		// A concurrent identical plan may have published the same immutable bytes.
		if verified, verifyErr := verify(); verifyErr == nil {
			return verified, nil
		}
		return "", err
	}
	return verify()
}
