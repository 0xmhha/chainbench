// Package derive turns one secret into every key material a node needs.
//
// A node key, an account address and a validator's BLS material are all derived
// from the same secret, which is why the key module has one command group
// rather than one per purpose. This package is that derivation, and it is pure:
// no files, no network, no external binary.
//
// The BLS path mirrors blst's blst_keygen exactly so that what it produces
// matches what the go-wbft bootnode tool produced. Getting it wrong yields a
// well-formed key that no wbft node will accept — a failure that surfaces as a
// consensus problem rather than a key problem.
//
// The derivation is pinned against known-good vectors: keys/preset ships each
// nodekey next to the address, devp2p public key, BLS public key and proof of
// possession it produced, and derive_test.go re-derives all of them and compares
// byte for byte. That the check catches a real drift is itself verified — the
// version-3 salt (the subtlety §blsKeyGen names) makes the preset comparison
// fail, not merely change.
//
// Deriving in process is what removed the external bootnode binary from the
// preset generator (docs/dev/keyring-design.md).
package derive
