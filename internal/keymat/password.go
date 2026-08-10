package keymat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// passwordFilePerm keeps a stored password owner-only.
const passwordFilePerm os.FileMode = 0o600

// PasswordSource supplies the password guarding a keystore. Implementations
// differ in where the password comes from and whether the user is asked.
type PasswordSource interface {
	Password() (string, error)
}

// StaticPassword is a password provided inline (flag/env).
type StaticPassword string

// Password returns the static value.
func (p StaticPassword) Password() (string, error) { return string(p), nil }

// FilePassword reads the password from a file (unattended use).
type FilePassword struct {
	Path string
}

// Password reads and trims the file's contents.
func (p FilePassword) Password() (string, error) {
	b, err := os.ReadFile(p.Path)
	if err != nil {
		return "", fmt.Errorf("keymat: read password file: %w", err)
	}
	return strings.TrimRight(string(b), "\r\n"), nil
}

// OnceThenFile asks for the password once, writes it to Path (0600), and reuses
// it thereafter without asking — the "confirm once, then don't prompt" mode.
// Prompt is injected so the behavior is testable without a terminal.
type OnceThenFile struct {
	Path   string
	Prompt func() (string, error)
}

// Password returns the stored password if the file exists; otherwise it prompts
// once, persists the answer, and returns it.
func (p OnceThenFile) Password() (string, error) {
	if b, err := os.ReadFile(p.Path); err == nil {
		return strings.TrimRight(string(b), "\r\n"), nil
	}
	if p.Prompt == nil {
		return "", fmt.Errorf("keymat: no stored password at %s and no prompt provided", p.Path)
	}
	pw, err := p.Prompt()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(p.Path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p.Path, []byte(pw), passwordFilePerm); err != nil {
		return "", fmt.Errorf("keymat: store password: %w", err)
	}
	return pw, nil
}
