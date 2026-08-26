package deploy

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// ServerIdentity is one server's node identity: which server it is, and the
// public material its nodekey implies.
//
// The identity itself is keyring.Identity — the same type every other part of
// chainbench uses — so nothing here has to be converted before it can be
// registered, written into a genesis, or compared with a declaration. This
// package used to carry its own NodeKeyInfo, which held a subset of the same
// fields under different names.
type ServerIdentity struct {
	// Server is the cluster server index this identity belongs to.
	Server int
	keyring.Identity
}

// ReadServerKeys reads one server's identity: it fetches the node's key over
// SSH and derives the address and BLS material from it locally, and (when
// localKeystoreDir is non-empty) pulls the coinbase and operator keystores to
// localKeystoreDir/keystore_<i> and operator_<i>. env sources the SSH auth
// fallbacks (nil = no env).
//
// It used to run `bootnode -writeaddress` on the server and scrape the
// addresses out of its output. Deriving here instead means the servers need no
// bootnode binary installed, and the identity comes from the same code that
// produced it — a text format between two machines was the fragile part.
func ReadServerKeys(ctx context.Context, c *Cluster, cr *Credentials, hostKey remote.HostKeyCallback, s Server, localKeystoreDir string, env func(string) string) (ServerIdentity, error) {
	rc, err := cr.For(c, s, env)
	if err != nil {
		return ServerIdentity{}, err
	}
	return readServerKeysFrom(ctx, serverFiles(rc, hostKey), filestore.Local{}, c.Paths(), s.Index, localKeystoreDir)
}

// readServerKeysFrom is ReadServerKeys with the store already open. Opening it
// needs credentials and a host; everything after that is just files, so the
// split is what makes the read and the derivation testable without a host.
func readServerKeysFrom(ctx context.Context, files, dest filestore.Store, p RemotePaths, server int, localKeystoreDir string) (ServerIdentity, error) {
	raw, err := files.Read(ctx, p.Nodekey)
	if err != nil {
		return ServerIdentity{}, fmt.Errorf("deploy: server %d read nodekey: %w", server, err)
	}
	key, err := keyring.ParsePrivateKey(string(raw))
	if err != nil {
		return ServerIdentity{}, fmt.Errorf("deploy: server %d nodekey at %s: %w", server, p.Nodekey, err)
	}
	id, err := keyring.Derive(key, keyring.WithBLS)
	if err != nil {
		return ServerIdentity{}, fmt.Errorf("deploy: server %d derive identity: %w", server, err)
	}
	out := ServerIdentity{Server: server, Identity: id}

	if localKeystoreDir == "" {
		return out, nil
	}
	if err := pullKeystores(ctx, files, dest, p, server, localKeystoreDir); err != nil {
		return out, err
	}
	return out, nil
}

// pullKeystores copies the server's coinbase and operator keystores into dest.
// They are encrypted key material, so they are written owner-only.
//
// Both sides go through a file store: from is the host they come off, dest is
// where they land. Writing the destination directly would have fixed it to this
// machine, which is the assumption the file interface exists to remove — the same
// one that left a remote network's genesis on the operator's disk.
func pullKeystores(ctx context.Context, from, dest filestore.Store, p RemotePaths, server int, destDir string) error {
	for _, ks := range []struct{ remote, local string }{
		{p.CoinbaseKeystore, fmt.Sprintf("keystore_%d", server)},
		{p.OperatorKeystore, fmt.Sprintf("operator_%d", server)},
	} {
		data, err := from.Read(ctx, ks.remote)
		if err != nil {
			return fmt.Errorf("deploy: server %d read %s: %w", server, ks.remote, err)
		}
		if err := dest.Write(ctx, filepath.Join(destDir, ks.local), data, keystoreFilePerm); err != nil {
			return err
		}
	}
	return nil
}

// Permissions for the keystores pulled down from a server. They are encrypted,
// but the password travels with the cluster, so they are treated as secrets.
const (
	keystoreDirPerm  fs.FileMode = 0o700
	keystoreFilePerm fs.FileMode = 0o600
)

// serverFiles opens the file store for one server. Reads and writes on that
// host go through it, so the deploy no longer carries its own SSH file I/O
// beside the shared one.
func serverFiles(rc remote.Credentials, hostKey remote.HostKeyCallback) filestore.Store {
	return driver.NewRemoteFileStore(driver.SSHRunner(rc, hostKey))
}

// shellQuote single-quotes a path for safe remote shell use.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// FormatAccountsFragment renders a `validators:` YAML fragment for the accounts
// file from read key info, so the operator does not transcribe by hand.
func FormatAccountsFragment(ids []ServerIdentity) string {
	var b strings.Builder
	b.WriteString("# generated by `chainbench remote keys read` — fill stake amounts.\n")
	b.WriteString("validators:\n")
	for _, id := range ids {
		fmt.Fprintf(&b, "  - server: %d\n", id.Server)
		fmt.Fprintf(&b, "    addr: %q\n", id.Address)
		fmt.Fprintf(&b, "    operator: %q\n", id.Address) // operator addr filled after operator keystore decode
		if id.BLS != nil {
			fmt.Fprintf(&b, "    bls: %q\n", id.BLS.PublicKey)
			fmt.Fprintf(&b, "    bls_pop: %q\n", id.BLS.PoP)
		}
	}
	return b.String()
}
