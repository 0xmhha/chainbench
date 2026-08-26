package store

// Where a key set lives is storage knowledge, resolved here for every surface:
// an explicit directory wins, the environment may name one, and the default
// is a well-known local set. A set is a directory, so the committed
// keys/preset is not a special thing — it is one key set that happens to be
// in the repository.

// DefaultKeySetDir is the key set a caller gets when it names none, and
// KeySetEnv overrides it.
const (
	DefaultKeySetDir = "keys/default"
	KeySetEnv        = "CHAINBENCH_KEYRING"
)

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
	return DefaultKeySetDir, "default"
}
