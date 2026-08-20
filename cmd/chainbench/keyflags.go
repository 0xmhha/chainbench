package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/core/target"
	"github.com/0xmhha/chainbench/internal/keymat"
	"github.com/0xmhha/chainbench/internal/serverset"
)

// sourceFlags select where an imported key comes from — a private key, a BIP-39
// mnemonic (with a configurable HD path), or a key file named with the single
// path syntax. Exactly one origin must be set.
//
// --from covers every file case, here or on another host, because where a file
// sits is a property of its path and not a different kind of import. It
// replaces --import, --remote-import, and the --server/--remote-path pair,
// which were three spellings of one idea and grew apart.
type sourceFlags struct {
	privateKey string
	mnemonic   string
	passphrase string
	from       string

	// Superseded by --from. Kept so existing scripts keep working.
	importFile   string
	remoteImport string
	remotePath   string
	server       int

	serverConfig string
	remoteUser   string
	remotePort   int
	coinType     uint32
	hdAccount    uint32
	hdIndex      uint32
}

func (f *sourceFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.privateKey, "private-key", "", "import from a 0x-hex private key")
	cmd.Flags().StringVar(&f.mnemonic, "mnemonic", "", "import from a BIP-39 mnemonic")
	cmd.Flags().StringVar(&f.passphrase, "passphrase", "", "optional BIP-39 passphrase (with --mnemonic)")
	cmd.Flags().StringVar(&f.from, "from", "",
		"import a key file by path: /local/path | srv://<server>/path | [user@]host:path | ssh://user@host:port/path")

	cmd.Flags().StringVar(&f.importFile, "import", "", "deprecated: use --from")
	_ = cmd.Flags().MarkDeprecated("import", "use --from <path>")
	cmd.Flags().StringVar(&f.remoteImport, "remote-import", "", "deprecated: use --from")
	_ = cmd.Flags().MarkDeprecated("remote-import", "use --from [user@]host:path, or --from srv://<server>/path to keep the address out of the command line")
	cmd.Flags().IntVar(&f.server, "server", 0, "deprecated: use --from srv://<server>/path")
	_ = cmd.Flags().MarkDeprecated("server", "use --from srv://<server>/path")
	cmd.Flags().StringVar(&f.remotePath, "remote-path", "", "deprecated: use --from srv://<server>/path")
	_ = cmd.Flags().MarkDeprecated("remote-path", "use --from srv://<server>/path")

	cmd.Flags().StringVar(&f.serverConfig, "server-config", serverset.DefaultConfigFile, "server inventory file for srv:// targets")
	cmd.Flags().StringVar(&f.remoteUser, "remote-user", "", "override the SSH user for a host named directly in --from")
	cmd.Flags().IntVar(&f.remotePort, "remote-port", 0, "override the SSH port for a host named directly in --from (default 22)")
	cmd.Flags().Uint32Var(&f.coinType, "hd-coin-type", keymat.DefaultCoinType, "BIP-44 coin type for --mnemonic (60=Ethereum; set your chain's for exact addresses)")
	cmd.Flags().Uint32Var(&f.hdAccount, "hd-account", 0, "BIP-44 account index for --mnemonic")
	cmd.Flags().Uint32Var(&f.hdIndex, "hd-index", 0, "BIP-44 address index for --mnemonic")
}

// source builds the keymat.Source, requiring exactly one origin. pw guards a
// keystore file import (local or remote). Production passes os.Getenv.
func (f *sourceFlags) source(pw keymat.PasswordSource) (keymat.Source, error) {
	return f.sourceWithEnv(pw, os.Getenv)
}

// sourceWithEnv is source with an injected environment for the remote SSH creds.
func (f *sourceFlags) sourceWithEnv(pw keymat.PasswordSource, env func(string) string) (keymat.Source, error) {
	path, err := f.fromPath()
	if err != nil {
		return nil, err
	}
	switch {
	case f.privateKey != "":
		return keymat.PrivateKeySource{Hex: f.privateKey}, nil
	case f.mnemonic != "":
		return keymat.MnemonicSource{
			Mnemonic: f.mnemonic, Passphrase: f.passphrase,
			Path: keymat.HDPath{CoinType: f.coinType, Account: f.hdAccount, Index: f.hdIndex},
		}, nil
	default:
		files, keyPath, err := f.openFrom(path, env)
		if err != nil {
			return nil, err
		}
		return keymat.FileSource{Files: files, Path: keyPath, Password: pw}, nil
	}
}

// fromPath folds the superseded flags into the one --from spelling and enforces
// that exactly one origin was named. Doing the fold here means the rest of the
// command works in a single vocabulary regardless of which flag was typed.
func (f *sourceFlags) fromPath() (string, error) {
	path := f.from
	switch {
	case f.importFile != "":
		path = f.importFile
	case f.remoteImport != "":
		path = f.remoteImport
	case f.server != 0:
		if f.remotePath == "" {
			return "", fmt.Errorf("--server needs --remote-path; prefer --from srv://<server>/path")
		}
		// --server took an index; --from names the entry. The inventory answers
		// both, so translate here rather than teaching the path syntax about
		// indexes — a number is not a name.
		cfg, err := serverset.Load(f.serverConfigPath())
		if err != nil {
			return "", err
		}
		srv, err := cfg.Server(f.server)
		if err != nil {
			return "", err
		}
		path = "srv://" + srv.Name + f.remotePath
	}

	origins := 0
	for _, set := range []bool{f.privateKey != "", f.mnemonic != "", path != ""} {
		if set {
			origins++
		}
	}
	if origins != 1 {
		return "", fmt.Errorf("provide exactly one of --private-key, --mnemonic, --from")
	}
	return path, nil
}

// openFrom resolves a --from path to the store that holds it and the path on
// that store. Local, inventory-named, and directly-addressed hosts all end up
// here, so none of them can grow its own read.
func (f *sourceFlags) openFrom(path string, env func(string) string) (provision.FileStore, string, error) {
	spec, err := target.ParseTarget(path)
	if err != nil {
		return nil, "", err
	}
	if f.remoteUser != "" && spec.Kind == target.TargetRemote {
		spec.User = f.remoteUser
	}
	if f.remotePort != 0 && spec.Kind == target.TargetRemote {
		spec.Port = f.remotePort
	}
	t, err := spec.ResolveWith(env, serverset.InventoryLookup(f.serverConfigPath()))
	if err != nil {
		return nil, "", err
	}
	return t.Files, t.DataRoot, nil
}

// serverConfigPath is the inventory file to consult, defaulting when unset.
func (f *sourceFlags) serverConfigPath() string {
	if f.serverConfig != "" {
		return f.serverConfig
	}
	return serverset.DefaultConfigFile
}

// remoteFiles opens the file store for a host. The host-key policy is resolved
// here rather than inside the key source, so a key source is only ever told
// *where* a file is, never how to reach it.
func remoteFiles(creds remote.Credentials, env func(string) string) (provision.FileStore, error) {
	hostKey, err := remote.ResolveHostKeyCallback(env)
	if err != nil {
		return nil, err
	}
	return driver.NewRemoteFileStore(driver.SSHRunner(creds, hostKey)), nil
}

// serverCreds resolves the SSH credentials for --server N from the inventory,
// applying the --remote-user / --remote-port overrides on top of the file.
func (f *sourceFlags) serverCreds(env func(string) string) (remote.Credentials, error) {
	cfg, err := serverset.Load(f.serverConfig)
	if err != nil {
		return remote.Credentials{}, err
	}
	srv, err := cfg.Server(f.server)
	if err != nil {
		return remote.Credentials{}, err
	}
	if f.remoteUser != "" {
		srv.SSH.User = f.remoteUser
	}
	if f.remotePort != 0 {
		srv.SSH.Port = f.remotePort
	}
	return srv.Credentials(env)
}

// parseRemoteImport splits a "[user@]host:path" spec. The user may also come
// from CHAINBENCH_REMOTE_USER, so it may be empty here.
func parseRemoteImport(s string) (user, host, path string, err error) {
	rest := s
	if at := strings.Index(s, "@"); at >= 0 {
		user, rest = s[:at], s[at+1:]
	}
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return "", "", "", fmt.Errorf("--remote-import must be [user@]host:path")
	}
	host, path = rest[:colon], rest[colon+1:]
	if host == "" || path == "" {
		return "", "", "", fmt.Errorf("--remote-import must be [user@]host:path")
	}
	return user, host, path, nil
}

// storeFlags select whether and how a key is persisted. Storage is off unless
// --out is given (the command then only prints).
type storeFlags struct {
	out   string
	name  string
	store string
}

func (f *storeFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.out, "out", "", "directory to store the key in (omit to only print)")
	cmd.Flags().StringVar(&f.name, "name", "key", "stored key name (file base)")
	cmd.Flags().StringVar(&f.store, "store", "keystore", "storage format: keystore|file")
}

func (f *storeFlags) enabled() bool { return f.out != "" }

func (f *storeFlags) build() (keymat.Store, error) {
	switch f.store {
	case "keystore":
		return keymat.KeystoreStore{}, nil
	case "file":
		return keymat.RawFileStore{}, nil
	default:
		return nil, fmt.Errorf("--store must be keystore or file")
	}
}

// passwordFlags select how the keystore password is supplied.
type passwordFlags struct {
	password     string
	passwordFile string
	passwordOnce string
}

func (f *passwordFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.password, "password", "", "keystore password (inline)")
	cmd.Flags().StringVar(&f.passwordFile, "password-file", "", "read the keystore password from a file")
	cmd.Flags().StringVar(&f.passwordOnce, "password-once", "", "prompt for the password once, store it at this path, and reuse it without asking")
}

func (f *passwordFlags) source() keymat.PasswordSource {
	switch {
	case f.password != "":
		return keymat.StaticPassword(f.password)
	case f.passwordFile != "":
		return keymat.FilePassword{Path: f.passwordFile}
	case f.passwordOnce != "":
		return keymat.OnceThenFile{Path: f.passwordOnce, Prompt: promptPassword}
	default:
		return nil
	}
}

// saveKey persists a per the store/password flags, returning the file path, or
// "" when storage is disabled. A keystore store requires a password.
func saveKey(sf *storeFlags, pf *passwordFlags, a *account.Account) (string, error) {
	if !sf.enabled() {
		return "", nil
	}
	store, err := sf.build()
	if err != nil {
		return "", err
	}
	pw := pf.source()
	if _, isKeystore := store.(keymat.KeystoreStore); isKeystore && pw == nil {
		return "", fmt.Errorf("keystore storage needs a password (--password / --password-file / --password-once)")
	}
	return store.Save(sf.out, sf.name, a, pw)
}

// keyView selects what identity a command reports for a key. The `keys` layer
// deals in raw keypairs, so it shows the public key; the `account`/`validator`
// layers deal in on-chain identity, so they show the address.
type keyView int

const (
	viewKeys    keyView = iota // privateKey + publicKey
	viewAccount                // privateKey + address
)

// publicKeyHex is the 64-byte (ethereum-style, uncompressed minus the 0x04 tag)
// public key as 0x-hex.
func publicKeyHex(a *account.Account) string {
	uncompressed := a.PublicKey().SerializeUncompressed() // 0x04 || X(32) || Y(32)
	return "0x" + hex.EncodeToString(uncompressed[1:])
}

// printKey renders a key/account. showPrivate includes the private key (for a
// freshly generated key). view selects publicKey (keys) vs address (account).
func printKey(out io.Writer, a *account.Account, showPrivate bool, view keyView, storedPath string, jsonOut bool) error {
	private := "0x" + hex.EncodeToString(a.PrivateKeyBytes())
	m := map[string]string{}
	if showPrivate {
		m["privateKey"] = private
	}
	switch view {
	case viewKeys:
		m["publicKey"] = publicKeyHex(a)
	default:
		m["address"] = a.Address().Hex()
	}
	if storedPath != "" {
		m["stored"] = storedPath
	}
	if jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	}
	if showPrivate {
		fmt.Fprintf(out, "privateKey: %s\n", private)
	}
	if view == viewKeys {
		fmt.Fprintf(out, "publicKey:  %s\n", m["publicKey"])
	} else {
		fmt.Fprintf(out, "address:    %s\n", m["address"])
	}
	if storedPath != "" {
		fmt.Fprintf(out, "stored:     %s\n", storedPath)
	}
	return nil
}

// runGenerate generates a fresh account, optionally stores it, and prints it
// (with the private key). Shared by `keys new` and `account new`.
func runGenerate(cmd *cobra.Command, sf *storeFlags, pf *passwordFlags, view keyView, jsonOut bool) error {
	a, err := keymat.RandomSource{}.Resolve(cmd.Context())
	if err != nil {
		return err
	}
	path, err := saveKey(sf, pf, a)
	if err != nil {
		return err
	}
	return printKey(cmd.OutOrStdout(), a, true, view, path, jsonOut)
}

// runImport resolves an account from the source flags, optionally stores it, and
// prints it (address only — the caller already holds the secret). Shared by
// `keys import` and `account import`.
func runImport(cmd *cobra.Command, src *sourceFlags, sf *storeFlags, pf *passwordFlags, view keyView, jsonOut bool) error {
	source, err := src.source(pf.source())
	if err != nil {
		return err
	}
	a, err := source.Resolve(cmd.Context())
	if err != nil {
		return err
	}
	path, err := saveKey(sf, pf, a)
	if err != nil {
		return err
	}
	return printKey(cmd.OutOrStdout(), a, false, view, path, jsonOut)
}

// promptPassword reads a password from the terminal without echo.
func promptPassword() (string, error) {
	fmt.Fprint(os.Stderr, "password: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
