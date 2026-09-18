package process

// Plan is the fully-resolved launch description for one network: what genesis
// to write and which nodes to provision/launch. Building a Plan performs no
// I/O, so it is unit testable and inspectable before anything runs.
//
// It lives here — next to NodeSpec, the per-node half of the same contract —
// so the packages that execute plans (launcher, engine, chainsetup) do not
// depend on the legacy pipeline package that used to declare it
// (core/pipeline/setup keeps a type alias for its remaining legacy
// consumers until they migrate).
type Plan struct {
	Chain        string
	Network      string
	DataRoot     string
	GenesisPath  string
	Genesis      []byte // genesis.json bytes (supplied by the consensus family)
	Capabilities []string
	Nodes        []NodeSpec
}
