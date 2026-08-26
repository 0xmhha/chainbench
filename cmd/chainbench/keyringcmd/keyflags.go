package keyringcmd

import (
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/netmap"
)

// SourceFlags select where an imported key comes from — a private key, a BIP-39
// mnemonic (with a configurable HD path), or a key file named with the single
// path syntax. Exactly one origin must be set.
//
// --from covers every file case, here or on another host, because where a file
// sits is a property of its path and not a different kind of import. It
// replaces --import, --remote-import, and the --server/--remote-path pair,
// which were three spellings of one idea and grew apart.
type SourceFlags struct {
	privateKey string
	mnemonic   string
	passphrase string
	from       string

	// Superseded by --from. Kept so existing scripts keep working.
	importFile   string
	remoteImport string
	remotePath   string
	server       int

	serverSet  string
	remoteUser string
	remotePort int
	coinType   uint32
	hdAccount  uint32
	hdIndex    uint32
}

func (f *SourceFlags) Bind(cmd *cobra.Command) {
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

	cmd.Flags().StringVar(&f.serverSet, "server-set", netmap.DefaultSetFile, "server-set file for srv:// targets")
	cmd.Flags().StringVar(&f.remoteUser, "remote-user", "", "override the SSH user for a host named directly in --from")
	cmd.Flags().IntVar(&f.remotePort, "remote-port", 0, "override the SSH port for a host named directly in --from (default 22)")
	cmd.Flags().Uint32Var(&f.coinType, "hd-coin-type", keyring.DefaultCoinType, "BIP-44 coin type for --mnemonic (60=Ethereum; set your chain's for exact addresses)")
	cmd.Flags().Uint32Var(&f.hdAccount, "hd-account", 0, "BIP-44 account index for --mnemonic")
	cmd.Flags().Uint32Var(&f.hdIndex, "hd-index", 0, "BIP-44 address index for --mnemonic")
}

// source builds the keyring.Source, requiring exactly one origin. pw guards a
// keystore file import (local or remote). Production passes os.Getenv.
func (f *SourceFlags) Source(pw keyring.PasswordSource) (keyring.Source, error) {
	return f.sourceWithEnv(pw, os.Getenv)
}

// sourceWithEnv is source with an injected environment for the remote SSH creds.
func (f *SourceFlags) sourceWithEnv(pw keyring.PasswordSource, env func(string) string) (keyring.Source, error) {
	path, err := f.fromPath()
	if err != nil {
		return nil, err
	}
	switch {
	case f.privateKey != "":
		return keyring.PrivateKeySource{Hex: f.privateKey}, nil
	case f.mnemonic != "":
		return keyring.MnemonicSource{
			Mnemonic: f.mnemonic, Passphrase: f.passphrase,
			Path: keyring.HDPath{CoinType: f.coinType, Account: f.hdAccount, Index: f.hdIndex},
		}, nil
	default:
		files, keyPath, err := f.openFrom(path, env)
		if err != nil {
			return nil, err
		}
		return keyring.FileSource{Files: files, Path: keyPath, Password: pw}, nil
	}
}

// fromPath folds the superseded flags into the one --from spelling and enforces
// that exactly one origin was named. Doing the fold here means the rest of the
// command works in a single vocabulary regardless of which flag was typed.
func (f *SourceFlags) fromPath() (string, error) {
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
		// --server took an index; --from names the entry. The server set answers
		// both, so translate here rather than teaching the path syntax about
		// indexes — a number is not a name.
		cfg, err := netmap.LoadSet(f.serverSetPath())
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
// that store. Local, server set-named, and directly-addressed hosts all end up
// here, so none of them can grow its own read.
func (f *SourceFlags) openFrom(path string, env func(string) string) (filestore.Store, string, error) {
	spec, err := machine.Parse(path)
	if err != nil {
		return nil, "", err
	}
	// The overrides only mean anything for a directly-named host — a local
	// path has no user, a server-set entry gets both from the set — so
	// setting them unconditionally changes nothing there and keeps this
	// consumer free of kind branches.
	if f.remoteUser != "" {
		spec.User = f.remoteUser
	}
	if f.remotePort != 0 {
		spec.Port = f.remotePort
	}
	t, err := netmap.Opener{ServerSet: f.serverSetPath(), Env: env}.Open(spec)
	if err != nil {
		return nil, "", err
	}
	return t.Files, t.DataRoot, nil
}

// serverSetPath is the server-set file to consult, defaulting when unset.
func (f *SourceFlags) serverSetPath() string {
	if f.serverSet != "" {
		return f.serverSet
	}
	return netmap.DefaultSetFile
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

func (f *storeFlags) build() (store.Backend, error) {
	switch f.store {
	case "keystore":
		return store.KeystoreBackend{}, nil
	case "file":
		return store.RawFileBackend{}, nil
	default:
		return nil, fmt.Errorf("--store must be keystore or file")
	}
}

// PasswordFlags select how the keystore password is supplied.
type PasswordFlags struct {
	password     string
	passwordFile string
	passwordOnce string
}

func (f *PasswordFlags) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.password, "password", "", "keystore password (inline)")
	cmd.Flags().StringVar(&f.passwordFile, "password-file", "", "read the keystore password from a file")
	cmd.Flags().StringVar(&f.passwordOnce, "password-once", "", "prompt for the password once, store it at this path, and reuse it without asking")
}

func (f *PasswordFlags) Source() keyring.PasswordSource {
	switch {
	case f.password != "":
		return keyring.StaticPassword(f.password)
	case f.passwordFile != "":
		return keyring.FilePassword{Path: f.passwordFile}
	case f.passwordOnce != "":
		return keyring.OnceThenFile{Path: f.passwordOnce, Prompt: promptPassword}
	default:
		return nil
	}
}

// saveKey persists the key per the store/password flags, returning the file path, or
// "" when storage is disabled. A keystore store requires a password.
func saveKey(sf *storeFlags, pf *PasswordFlags, key derive.PrivateKey) (string, error) {
	if !sf.enabled() {
		return "", nil
	}
	backend, err := sf.build()
	if err != nil {
		return "", err
	}
	pw := pf.Source()
	if _, isKeystore := backend.(store.KeystoreBackend); isKeystore && pw == nil {
		return "", fmt.Errorf("keystore storage needs a password (--password / --password-file / --password-once)")
	}
	return backend.Save(sf.out, sf.name, key, pw)
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
