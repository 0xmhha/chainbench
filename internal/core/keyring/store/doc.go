// Package store persists a key set and reads it back.
//
// core/keyring is the model — what an identity is. This package is where one
// lands on disk: a ring is a directory, and each identity is a file under it.
// Two backends put the same identity in different forms, and they differ only
// here: [RawFileBackend] writes the bare hex a node reads as its nodekey, and
// [KeystoreBackend] writes the encrypted keystore an account unlocks from.
//
// [LoadPreset] reads the committed 5-node fixture and anything `validator set`
// generates in the same shape, which is what lets a network larger than the
// fixture compose without a second code path.
//
// The keys under keys/preset are PUBLIC TEST FIXTURES — the upstream
// go-ethereum test key among them — and exist so a test can fund a transfer
// reproducibly. A secret scanner reports them and those reports are expected.
// Never move one, or an address derived from one, onto a shared network
// (docs/SECURITY_KEY_HANDLING.md).
package store
