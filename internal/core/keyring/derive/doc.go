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
// This package has no tests of its own. The derivation is exercised only
// indirectly, through core/keyring's preset tests, so a change here that still
// produces well-formed output would not be caught until a chain refused to
// seal. The committed keys/preset holds nodekeys next to the BLS public keys
// they produced, which is a known-good vector set a direct test could pin
// against.
//
// Deriving in process is what removed the external bootnode binary from the
// preset generator (docs/dev/keyring-design.md).
package derive
