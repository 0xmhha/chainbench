package keyringcmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/0xmhha/chainbench/internal/app"
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

	cmd.Flags().StringVar(&f.serverSet, "server-set", app.DefaultServerSetFile, "server-set file for srv:// targets")
	cmd.Flags().StringVar(&f.remoteUser, "remote-user", "", "override the SSH user for a host named directly in --from")
	cmd.Flags().IntVar(&f.remotePort, "remote-port", 0, "override the SSH port for a host named directly in --from (default 22)")
	// Zero, not 60: a flag's default is not something the operator named, and
	// app refuses HD options that qualify a mnemonic nobody asked for. The
	// keyring applies 60 when the coin type is left unset.
	cmd.Flags().Uint32Var(&f.coinType, "hd-coin-type", 0,
		fmt.Sprintf("BIP-44 coin type for --mnemonic (default %d = Ethereum; set your chain's for exact addresses)", app.DefaultHDCoinType))
	cmd.Flags().Uint32Var(&f.hdAccount, "hd-account", 0, "BIP-44 account index for --mnemonic")
	cmd.Flags().Uint32Var(&f.hdIndex, "hd-index", 0, "BIP-44 address index for --mnemonic")
}

// Ref describes the key origin the operator named, for app to read.
//
// The surface's job ends at describing it. Turning "exactly one of
// --private-key, --mnemonic, --from" into a key is one reading, and it lives in
// the keyring module: this file used to carry a second copy of that reading, so
// the two were free to disagree about what a bare mnemonic, or a password with
// no path, meant.
func (f *SourceFlags) Ref(pw func() (string, error)) (app.KeyRef, error) {
	path, err := f.fromPath()
	if err != nil {
		return app.KeyRef{}, err
	}
	return app.KeyRef{
		PrivateKey: f.privateKey,
		Mnemonic:   f.mnemonic,
		Passphrase: f.passphrase,
		HDCoinType: f.coinType,
		HDAccount:  f.hdAccount,
		HDIndex:    f.hdIndex,
		From:       path,
		Password:   pw,
		ServerSet:  f.serverSetPath(),
	}, nil
}

// Resolve reads the key the flags name.
func (f *SourceFlags) Resolve(ctx context.Context, d app.Deps, pw func() (string, error)) (app.PrivateKey, error) {
	ref, err := f.Ref(pw)
	if err != nil {
		return app.PrivateKey{}, err
	}
	return app.ResolveKey(ctx, d, ref)
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
		name, err := app.ServerNameByIndex(f.serverSetPath(), f.server)
		if err != nil {
			return "", err
		}
		path = "srv://" + name + f.remotePath
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

// serverSetPath is the server-set file to consult, defaulting when unset.
func (f *SourceFlags) serverSetPath() string {
	if f.serverSet != "" {
		return f.serverSet
	}
	return app.DefaultServerSetFile
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

// Source returns how to obtain the password, or nil when none was named.
//
// It is a function rather than a value so the password is fetched only if it is
// actually needed: --password-once prompts the operator, and asking for a
// password that the command then never uses is a poor way to treat them.
func (f *PasswordFlags) Source() func() (string, error) {
	switch {
	case f.password != "":
		pw := f.password
		return func() (string, error) { return pw, nil }
	case f.passwordFile != "":
		path := f.passwordFile
		return func() (string, error) {
			b, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("read password file %s: %w", path, err)
			}
			return strings.TrimSpace(string(b)), nil
		}
	case f.passwordOnce != "":
		return oncePassword(f.passwordOnce)
	default:
		return nil
	}
}

// oncePassword prompts the first time, saves the answer, and reuses it after,
// so a long sequence of key operations asks once rather than each time.
func oncePassword(path string) func() (string, error) {
	return func() (string, error) {
		if b, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(b)), nil
		}
		pw, err := promptPassword()
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(pw), 0o600); err != nil {
			return "", fmt.Errorf("save password at %s: %w", path, err)
		}
		return pw, nil
	}
}

// saveKey persists the key per the store/password flags, returning the file path, or
// "" when storage is disabled. A keystore store requires a password.
func saveKey(ctx context.Context, d app.Deps, sf *storeFlags, pf *PasswordFlags, key app.PrivateKey) (string, error) {
	if !sf.enabled() {
		return "", nil
	}
	return app.SaveKey(ctx, d, key, app.SaveKeyIn{
		Dir: sf.out, Name: sf.name, Format: sf.store, Password: pf.Source(),
	})
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
