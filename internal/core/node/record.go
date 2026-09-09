package node

import "encoding/json"

// Record is what is known about one composed node — the fact record the
// composition writes and every later step reads. One node has exactly one
// Record; everything else that speaks about a node is either a view derived
// from it (process.NodeSpec is the launch input, Node is the runtime hand-off)
// or a different concept wearing a precise name (an observation sample, a
// topology declaration, a process ledger entry).
//
// It exists because the same facts used to live in ten types across seven
// packages, none holding all of them, and the copies drifted — the etcd port
// vanished between the plan and the running network twice. The JSON tags are
// the workspace.json contract: changing one is a migration, not a rename.
type Record struct {
	Index int `json:"index"`
	// Label is the node's identity: the name its datadir, config and log file
	// carry, and the name an operator uses to address it. It is stored rather
	// than derived so that a name, once given, survives.
	//
	// A workspace written before this field falls back to the conventional
	// label for its index, so nothing has to be migrated.
	Label string `json:"label,omitempty"`
	Role  string `json:"role"`
	// SyncMode is the geth sync mode this node's config renders. Validators
	// are always "full" — they must hold full state to seal — while an
	// endpoint may be switched to "snap" or "archive" so a large-gap re-sync
	// exercises that path. Empty means the config's own default.
	SyncMode string `json:"syncMode,omitempty"`
	// Server names the server-set entry whose machine this node runs on, when
	// the pool spread the network across a set ("--all-servers"). Empty means
	// the node lives on the workspace's single target. Every per-node file
	// write, init, and launch resolves this name through the resource module,
	// so each node's material lands on ITS machine, not the first one's.
	Server string `json:"server,omitempty"`
	// Host is the address this node is reachable at. It comes from the
	// allocator, so a remote placement records the server's address rather
	// than this machine's.
	Host       string `json:"host,omitempty"`
	DataDir    string `json:"dataDir"`
	ConfigPath string `json:"configPath"`
	LogPath    string `json:"logPath"`
	// Endpoints is embedded rather than copied field by field: its keys inline
	// into this object, so the persisted shape does not change, and a port can
	// no longer be dropped in a conversion. That is how the etcd port went
	// missing between the plan and the running network, and the first attempt
	// at restoring it dropped the port again in one of the three copies.
	Endpoints
	// Binary names which binary this node runs, by the key declared in the
	// composition's binaries map. Empty means the composition's single binary,
	// so a workspace with no per-node binaries behaves exactly as before.
	Binary string `json:"binary,omitempty"`
	// Config is a pre-written config file this node uses verbatim (topology
	// node table's config field). Empty renders the config as usual. When set,
	// the config step writes this file to ConfigPath rather than rendering one,
	// so the file is the whole config — it carries its own ports, datadir and
	// static-nodes, none of which the composition fills in.
	Config string `json:"config,omitempty"`
	// Args is the assembled launch argv (once launchopts ran).
	Args []string `json:"args,omitempty"`
	// PID is the live process id (once start ran; 0 = stopped). Stopping a
	// node clears it and restarting writes a new one — the pid changes, the
	// Record stays, and only removing the network releases its resources.
	PID int `json:"pid,omitempty"`
}

// NodeLabel is the node's identity, falling back to the conventional label
// for records written before the Label field existed.
func (r Record) NodeLabel() Label {
	if r.Label != "" {
		return Label(r.Label)
	}
	return LabelFor(r.Index)
}

// UnmarshalJSON folds Role onto the canonical vocabulary as the record is read,
// the same way [Node.UnmarshalJSON] does for the hand-off object.
//
// Both boundaries are needed because a workspace keeps two tables: Record is
// what the composition persists in workspace.json, Node is what the phases pass
// around. A fold on only one of them would leave a workspace written before NM6
// reporting "validator" in `chain status` and "bp" everywhere else, which is
// how one vocabulary becomes two.
func (r *Record) UnmarshalJSON(b []byte) error {
	type raw Record // shed the method, or this recurses
	var v raw
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*r = Record(v)
	if canonical, err := NormalizeRole(r.Role); err == nil {
		r.Role = string(canonical)
	}
	return nil
}
