package app

import (
	"context"
	"crypto/rand"
	"errors"
	"os"

	"github.com/0xmhha/chainbench/internal/core/session"
)

type webKeySnapshot struct {
	SHA256 string `json:"sha256"`
}

func (e *WebChainEngine) pinKeys(ctx context.Context) (webKeySnapshot, error) {
	e.keysMu.Lock()
	defer e.keysMu.Unlock()
	raw, err := session.CaptureKeys(ctx, e.keys)
	if err != nil {
		return webKeySnapshot{}, err
	}
	snapshot := webKeySnapshot{SHA256: manifestHash(raw)}
	files, err := session.OpenKeySnapshotFiles(e.root)
	if err != nil {
		return webKeySnapshot{}, err
	}
	if encrypted, err := files.Read(snapshot.SHA256); err == nil {
		if _, err = e.decryptKeys(snapshot, encrypted); err != nil {
			return webKeySnapshot{}, err
		}
		return snapshot, nil
	} else if !os.IsNotExist(err) {
		return webKeySnapshot{}, err
	}
	nonce := make([]byte, e.documents.aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return webKeySnapshot{}, err
	}
	encrypted := e.documents.aead.Seal(nonce, nonce, raw, []byte("web-key-snapshot:"+snapshot.SHA256))
	if err = ctx.Err(); err != nil {
		return webKeySnapshot{}, err
	}
	if err = files.Save(snapshot.SHA256, encrypted); err != nil {
		return webKeySnapshot{}, err
	}
	return snapshot, nil
}

func (e *WebChainEngine) decryptKeys(snapshot webKeySnapshot, encrypted []byte) ([]byte, error) {
	if len(snapshot.SHA256) != 64 || len(encrypted) < e.documents.aead.NonceSize() {
		return nil, errors.New("missing or invalid accepted key snapshot")
	}
	n := e.documents.aead.NonceSize()
	raw, err := e.documents.aead.Open(nil, encrypted[:n], encrypted[n:], []byte("web-key-snapshot:"+snapshot.SHA256))
	if err != nil || manifestHash(raw) != snapshot.SHA256 {
		return nil, errors.New("accepted key snapshot integrity check failed")
	}
	return raw, nil
}

func (e *WebChainEngine) materializeKeys(ctx context.Context, snapshot webKeySnapshot) (string, error) {
	e.keysMu.Lock()
	defer e.keysMu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	files, err := session.OpenKeySnapshotFiles(e.root)
	if err != nil {
		return "", err
	}
	encrypted, err := files.Read(snapshot.SHA256)
	if err != nil {
		return "", err
	}
	raw, err := e.decryptKeys(snapshot, encrypted)
	if err != nil {
		return "", err
	}
	directory, err := files.Materialize(ctx, snapshot.SHA256, raw)
	if err != nil {
		return "", err
	}
	material, err := session.CaptureKeys(ctx, directory)
	if err != nil {
		return "", err
	}
	if manifestHash(material) != snapshot.SHA256 {
		return "", errors.New("materialized keys differ from accepted snapshot")
	}
	return directory, nil
}
