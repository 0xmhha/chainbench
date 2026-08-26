package remote

// The environment variables this package reads, named once so no caller
// spells one itself. A misspelled literal in one of a dozen call sites is a
// login that silently falls back to "unset", which is exactly the failure the
// single-source rule elsewhere in this repo exists to prevent.
//
// WHY THESE STILL EXIST. Connection settings are data in the server set: a
// named server's user, password (or password_file), key, and host-key policy
// all live there, and the environment is never consulted for them. These four
// cover the ONE form that has no entry to read — a host named directly on the
// command line (user@host:/path, ssh://…). That form is deliberately
// file-less, so its secret has to arrive out of band, and an environment
// variable keeps it off the command line and out of the process table.
//
// A workflow that wants no environment at all names its servers in a server
// set and uses srv://<name>/path.
const (
	// EnvUser overrides the SSH user for a directly named host.
	EnvUser = "CHAINBENCH_REMOTE_USER"
	// EnvPass supplies its password.
	EnvPass = "CHAINBENCH_REMOTE_PASS"
	// EnvKeyFile supplies a private key file instead of a password.
	EnvKeyFile = "CHAINBENCH_REMOTE_KEY_FILE"
	// EnvKeyPassphrase decrypts that key when it is encrypted.
	EnvKeyPassphrase = "CHAINBENCH_REMOTE_KEY_PASSPHRASE"
)

// EnvNames lists every environment variable this package reads, so a surface
// can document or clear them without keeping its own copy of the list.
func EnvNames() []string {
	return []string{EnvUser, EnvPass, EnvKeyFile, EnvKeyPassphrase}
}
