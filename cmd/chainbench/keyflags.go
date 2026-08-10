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
