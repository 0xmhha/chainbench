package node

// LaunchReq is one node to launch: its role, and the launch choices that are
// not derivable from the network's shape.
//
// It carries no name. The name a node answers to is its Label — it used to be
// invented here in four different spellings by different callers and read by
// none, which is why a node's label is now assigned once and persisted.
//
// It lived in its own package (core/place) that once owned allocation as well:
// an Allocator with three modes deciding which host and ports each node took.
// Assign replaced that — two of the modes turned out to be one grid of
// addresses and port slots read two ways, and the third asked the OS for free
// ports and had no callers. What was left was this request, which is a launch
// concern rather than a placement one, so it belongs beside the node model.
type LaunchReq struct {
	Role   Role
	Sync   string
	Binary string
}
