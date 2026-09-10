// Package filestore materializes a node's on-disk environment — data dir,
// config, genesis, and key files — through a Store, so the same flow serves a
// local and a remote target.
//
// A file already present is reused rather than overwritten, and the decision is
// made on CONTENT: [Hash] is the one checksum form the interface and the
// session records share, so a file there with a different digest is a different
// file and is rewritten, while an identical one is never re-sent. Presence alone
// was not enough — that is how a stale artifact came to be reused as if it were
// the one the run asked for.
//
// The remote Store is SSH-backed and hashes on the host, so an unchanged file is
// not downloaded just to be compared. This package ships [Local] and the
// provisioning flow; core/process supplies the remote implementation.
//
// It has fan-out zero and is depended on by twelve packages: it is a primitive,
// and it stays one.
package filestore
