package process

// FailureMode classifies a bring-up or launch failure so a diagnosis names its
// cause rather than reporting everything as "flaky". It is the classification
// nodemonitor reads to decide whether a not-ready node is restartable or fatal.
type FailureMode int

const (
	// UnknownFailure is the zero value: no cause has been established. It is
	// first so an unset value cannot masquerade as a real classification.
	UnknownFailure FailureMode = iota
	// EtcdJoinFailed means a node could not join the etcd cluster.
	EtcdJoinFailed
	// EtcdStale means a stale datadir blocked cluster formation on restart.
	EtcdStale
	// ForkNotCrossed means the target fork block was never reached.
	ForkNotCrossed
	// QuorumLost means the validator set fell below quorum.
	QuorumLost
	// RPCUnready means a node's RPC never became ready.
	RPCUnready
)

// String returns the failure-mode label.
func (m FailureMode) String() string {
	switch m {
	case UnknownFailure:
		return "Unknown"
	case EtcdJoinFailed:
		return "EtcdJoinFailed"
	case EtcdStale:
		return "EtcdStale"
	case ForkNotCrossed:
		return "ForkNotCrossed"
	case QuorumLost:
		return "QuorumLost"
	case RPCUnready:
		return "RPCUnready"
	default:
		return "Unknown"
	}
}
