package testengine

import "github.com/0xmhha/chainbench/internal/chainsetup/verb"

func forkGateFromRecord(f verb.ChainForkOut) forkGate {
	if f.Fork == nil {
		return forkGate{at: f.HaltsAt}
	}
	g := forkGate{at: f.Fork.At, restart: f.Fork.Restart, handedOver: f.HandedOver, preFork: make(map[int]bool, len(f.PreFork))}
	for _, i := range f.PreFork {
		g.preFork[i] = true
	}
	return g
}
