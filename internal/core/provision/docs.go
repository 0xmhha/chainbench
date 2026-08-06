// Package provision materializes a node's on-disk environment — data dir,
// config, genesis, and key files — through a FileSink so the same flow serves
// local and remote targets (DDD context C2). Files that already exist are
// reused rather than overwritten (upload-if-absent), so a key already present
// on a remote host is not clobbered.
//
// The remote FileSink (SSH/SFTP-backed, with a test -f existence check) lands
// with the remote vertical slice; this package ships the LocalFileSink and the
// provisioning flow.
package provision
