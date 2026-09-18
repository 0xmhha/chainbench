// Package keyring owns key material: generating a node's root secret, deriving
// every public identity from it, and (as the package grows) storing and naming
// those identities.
//
// Key material is the same on all three chains chainbench drives. The same
// private key yields the same address and the same devp2p public key under
// go-stablenet, go-wbft, and go-wemix; only two things differ, and neither is
// the key's structure:
//
//   - Whether derive.BLS material is used at all. The wbft family consumes it; the
//     poa family (wemix) has no derive.BLS references. Hence [derive.Derivation] — derive.BLS is
//     opt-in, and when it is not asked for it is absent, not zeroed.
//   - Whether the consensus account is the nodekey's own address. The wbft
//     family derives it; wemix conventionally uses a separate keystore.
//
// # A keyring knows nothing about a network
//
// An identity is who a node is. Where it listens, which peers it dials, and
// whether it is a validator are network decisions and live elsewhere — in the
// blueprint and in the node map. That is why this package derives a devp2p
// public key but never an enode URL: an enode is public key plus host plus
// port, and the last two are not this package's to know.
//
// Keeping the split means a keyring can be generated, inspected, and moved
// between hosts without a network existing yet, which is what lets a network be
// declared by hand instead of being implied by a preset.
//
// # derive.Derivation runs in process
//
// Address, devp2p public key, derive.BLS public key and derive.BLS proof of possession are
// all computed here in Go. No chain binary is executed, so key generation works
// with no build of go-wbft present and with CGO disabled. [derive.Derive] is checked
// byte for byte against the shipped keys/preset fixture.
package keyring
