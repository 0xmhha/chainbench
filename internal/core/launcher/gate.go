package launcher

import (
	"strings"
	"time"
)

// Etcd join-slot gaps, derived from the cluster size the way go-wemix's
// etcdAutoJoin does: each node sleeps until its own slot before attempting a
// join, and the slot spacing widens with the cluster. Reproducing the schedule
// here is what lets the launcher wait for a join that is merely scheduled
// rather than declaring it failed (design §3.3 L7, C-etcd).
const (
	smallClusterGap  = 7 * time.Second  // size <= 11
	mediumClusterGap = 11 * time.Second // size <= 23
	largeClusterGap  = 17 * time.Second // size <= 41
	hugeClusterGap   = 23 * time.Second // larger
)

// Cluster-size thresholds for the join-slot gaps above.
const (
	smallCluster  = 11
	mediumCluster = 23
	largeCluster  = 41
)

// JoinGap returns the etcd join-slot spacing for a cluster of the given size.
// The caller never passes a gap in: it is a property of the cluster, and a
// hard-coded value is exactly what made bring-up look flaky.
func JoinGap(clusterSize int) time.Duration {
	switch {
	case clusterSize <= smallCluster:
		return smallClusterGap
	case clusterSize <= mediumCluster:
		return mediumClusterGap
	case clusterSize <= largeCluster:
		return largeClusterGap
	default:
		return hugeClusterGap
	}
}

// JoinWindow is how long every node in a cluster of the given size needs to
// reach its join slot, plus one gap of settle margin. It is the deadline a
// leader gate should use when Options.AlignJoinGap is set: polling for a shorter
// time reports a join failure for a join that simply has not come round yet.
func JoinWindow(clusterSize int) time.Duration {
	if clusterSize < 1 {
		clusterSize = 1
	}
	gap := JoinGap(clusterSize)
	return time.Duration(clusterSize+1) * gap
}

// failureSignatures maps a substring of a real error to the mode it indicates,
// most specific first. Classification exists so a failure names its cause:
// "flaky" is not a diagnosis, and reporting every failure as RPCUnready hides
// the etcd problems that actually cause most bring-up failures.
var failureSignatures = []struct {
	needle string
	mode   FailureMode
}{
	{"cannot fetch cluster info", EtcdStale},
	{"stale", EtcdStale},
	{"etcd join", EtcdJoinFailed},
	{"join failed", EtcdJoinFailed},
	{"etcdinit", EtcdJoinFailed},
	{"no leader", EtcdJoinFailed},
	{"quorum", QuorumLost},
	{"fork", ForkNotCrossed},
	{"connection refused", RPCUnready},
	{"rpc", RPCUnready},
	{"dial", RPCUnready},
}

// Classify maps an error to the failure mode its text indicates, or
// UnknownFailure when nothing matches. An unrecognized error stays unknown
// rather than being filed under a plausible-sounding mode — a wrong diagnosis is
// worse than an honest "unknown", because it sends the reader to the wrong logs.
func Classify(err error) FailureMode {
	if err == nil {
		return UnknownFailure
	}
	msg := strings.ToLower(err.Error())
	for _, sig := range failureSignatures {
		if strings.Contains(msg, sig.needle) {
			return sig.mode
		}
	}
	return UnknownFailure
}
