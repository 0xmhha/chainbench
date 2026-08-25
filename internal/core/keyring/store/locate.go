package store

// Where a ring lives is storage knowledge, resolved here for every surface:
// an explicit directory wins, the environment may name one, and the default
// is a well-known local ring. A ring is a directory, so the committed
// keys/preset is not a special thing — it is one ring that happens to be in
// the repository.

// DefaultRingDir is the ring a caller gets when it names none, and RingEnv
// overrides it.
const (
	DefaultRingDir = "keys/default"
	RingEnv        = "CHAINBENCH_KEYRING"
)

// Locate returns the ring directory and where that choice came from
// ("explicit", RingEnv, or "default").
//
// The source travels with the answer because a caller that quietly fell back
// to a default is how an operator ends up inspecting one ring and launching
// from another.
func Locate(dir string, env func(string) string) (string, string) {
	if dir != "" {
		return dir, "explicit"
	}
	if env != nil {
		if v := env(RingEnv); v != "" {
			return v, RingEnv
		}
	}
	return DefaultRingDir, "default"
}
