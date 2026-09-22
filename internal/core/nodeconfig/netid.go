// Holding a composed network to one devp2p id.
//
// The id is a silent-failure trap. go-wemix defaults it to 1111 (Wemix mainnet)
// independent of the chain id, while go-wbft derives it from the chain id, so
// two builds meant to form one chain refuse to peer unless both are told the
// same number. [NetworkOf] is what decides that number; this file checks that
// the argv actually assembled from it still says one thing.
//
// It reads the assembled argv rather than the Network it came from, because
// what reaches a process is the argv. An override can name the flag on one
// scope and not another, and a second assembler can be written that does not
// go through NetworkOf — both leave the Network value agreeing with itself
// while the command lines disagree, which is the shape the defect had.

package nodeconfig

import (
	"fmt"
	"sort"
	"strconv"
)

// ValidateUniformNetworkID reports whether every node was assembled with the
// same devp2p id, naming the first pair that disagrees.
//
// argv is each node's assembled command line, keyed by the label a reader knows
// the node by. A node whose argv does not carry the flag at all is reported too:
// the value is emitted for every node a composition builds, so its absence means
// the node was assembled some other way and would take its build's own default.
//
// An empty map is an error rather than a pass. Nothing to check means the caller
// asked the wrong question, and answering "fine" to that is how a check comes to
// guard nothing.
func ValidateUniformNetworkID(argv map[string][]string) error {
	if len(argv) == 0 {
		return fmt.Errorf("nodeconfig: no argv to check for a uniform network id")
	}
	labels := make([]string, 0, len(argv))
	for label := range argv {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	first, firstLabel := int64(0), ""
	for _, label := range labels {
		id, ok := networkIDOf(argv[label])
		if !ok {
			return fmt.Errorf("nodeconfig: %s was assembled without %s, so it would use its build's own default", label, flagNetworkID)
		}
		if firstLabel == "" {
			first, firstLabel = id, label
			continue
		}
		if id != first {
			return fmt.Errorf("nodeconfig: network id mismatch — %s has %d and %s has %d; they will not peer", firstLabel, first, label, id)
		}
	}
	return nil
}

// flagNetworkID is the spelling every dialect uses for the devp2p id. It is
// stated once here so the check reads the same flag the assembler writes.
const flagNetworkID = "--networkid"

// networkIDOf reads the devp2p id out of one assembled command line.
func networkIDOf(args []string) (int64, bool) {
	for i, a := range args {
		if a != flagNetworkID || i+1 >= len(args) {
			continue
		}
		id, err := strconv.ParseInt(args[i+1], 10, 64)
		if err != nil {
			return 0, false
		}
		return id, true
	}
	return 0, false
}
