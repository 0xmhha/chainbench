package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/chainbench/internal/keymat"
)

// sourceFlags select where an imported key comes from — a private key, a BIP-39
// mnemonic (with a configurable HD path), or a key file. Exactly one must be set.
type sourceFlags struct {
	privateKey string
	mnemonic   string
	passphrase string
	importFile string
	coinType   uint32
	hdAccount  uint32
	hdIndex    uint32
}

func (f *sourceFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.privateKey, "private-key", "", "import from a 0x-hex private key")
	cmd.Flags().StringVar(&f.mnemonic, "mnemonic", "", "import from a BIP-39 mnemonic")
	cmd.Flags().StringVar(&f.passphrase, "passphrase", "", "optional BIP-39 passphrase (with --mnemonic)")
	cmd.Flags().StringVar(&f.importFile, "import", "", "import from a key file (raw hex or keystore JSON)")
	cmd.Flags().Uint32Var(&f.coinType, "hd-coin-type", keymat.DefaultCoinType, "BIP-44 coin type for --mnemonic (60=Ethereum; set your chain's for exact addresses)")
	cmd.Flags().Uint32Var(&f.hdAccount, "hd-account", 0, "BIP-44 account index for --mnemonic")
	cmd.Flags().Uint32Var(&f.hdIndex, "hd-index", 0, "BIP-44 address index for --mnemonic")
}

// source builds the keymat.Source, requiring exactly one origin. pw guards a
// keystore file import.
func (f *sourceFlags) source(pw keymat.PasswordSource) (keymat.Source, error) {
	n := 0
	for _, s := range []string{f.privateKey, f.mnemonic, f.importFile} {
		if s != "" {
			n++
		}
	}
	if n != 1 {
		return nil, fmt.Errorf("provide exactly one of --private-key, --mnemonic, --import")
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
		return keymat.FileSource{Path: f.importFile, Password: pw}, nil
	}
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

// printKey renders an account. showPrivate includes the private key (for a
// freshly generated key); an import shows only the address.
func printKey(out io.Writer, a *account.Account, showPrivate bool, storedPath string, jsonOut bool) error {
	address := a.Address().Hex()
	private := "0x" + hex.EncodeToString(a.PrivateKeyBytes())
	if jsonOut {
		m := map[string]string{"address": address}
		if showPrivate {
			m["privateKey"] = private
		}
		if storedPath != "" {
			m["stored"] = storedPath
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	}
	if showPrivate {
		fmt.Fprintf(out, "privateKey: %s\n", private)
	}
	fmt.Fprintf(out, "address:    %s\n", address)
	if storedPath != "" {
		fmt.Fprintf(out, "stored:     %s\n", storedPath)
	}
	return nil
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
