package store

import (
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/home"
)

// Where a key set lives is storage knowledge, resolved here for every surface:
// an explicit directory wins, the environment may name one, and the default
// is a well-known local set. A set is a directory, so the committed
// keys/preset is not a special thing — it is one key set that happens to be
// in the repository.

// DefaultKeySetName is the key set a caller gets when it names none, under the
// promised location; KeySetEnv overrides where that is.
const (
	DefaultKeySetName = "default"
	KeySetEnv         = "CHAINBENCH_KEYRING"
)

// DefaultKeySetDir is where an unnamed key set lives: ~/.chainbench/keys/default.
//
// It used to be the relative "keys/default", which made the default mean
// "beside wherever you are standing" — a ring created in one directory was
// invisible from another, and the operator had no way to tell from the name
// that this was happening.
func DefaultKeySetDir() (string, error) {
	keys, err := home.KeySets()
	if err != nil {
		return "", err
	}
	return filepath.Join(keys, DefaultKeySetName), nil
}

// Locate returns the ring directory and where that choice came from
// ("explicit", KeySetEnv, or "default").
//
// The source travels with the answer because a caller that quietly fell back
// to a default is how an operator ends up inspecting one ring and launching
// from another.
func Locate(dir string, env func(string) string) (string, string) {
	if dir != "" {
		return dir, "explicit"
	}
	if env != nil {
		if v := env(KeySetEnv); v != "" {
			return v, KeySetEnv
		}
	}
	d, err := DefaultKeySetDir()
	if err != nil {
		// Without a home directory there is no promised place, and the old
		// relative name is a worse answer than saying so — but Locate's callers
		// report the source, so returning it named "default" keeps the failure
		// where the directory is actually used.
		return filepath.Join("keys", DefaultKeySetName), "default"
	}
	return d, "default"
}
