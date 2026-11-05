package app

import (
	"context"

	"github.com/0xmhha/chainbench/internal/resource"
)

// ResourceReadIn selects bounded observations on a target, with file-backed logins.
type ResourceReadIn struct {
	resource.ReadRequest
	ServerSet string `json:"serverSet,omitempty"`
	Docker    bool   `json:"docker,omitempty"`
}

// ResourceReadOut contains observations without changing the target or a ledger.
type ResourceReadOut = resource.ReadResult

// ResourceRead is the target observation shared by command and agent adapters.
func ResourceRead(ctx context.Context, d Deps, in ResourceReadIn) (ResourceReadOut, error) {
	return (resource.Opener{ServerSet: in.ServerSet, Lookup: d.ServerLookup, Docker: in.Docker, Env: d.Env, Report: d.Logf}).Read(ctx, in.ReadRequest)
}

// ResourceProbeIn asks a target to run one inspection of a binary in its data
// root, with file-backed logins. It is no read: nothing read-only may reach it.
type ResourceProbeIn struct {
	resource.ProbeRequest
	ServerSet string `json:"serverSet,omitempty"`
	Docker    bool   `json:"docker,omitempty"`
}

// ResourceProbe runs that inspection for the command adapter.
func ResourceProbe(ctx context.Context, d Deps, in ResourceProbeIn) (resource.BinaryProbeResult, error) {
	return (resource.Opener{ServerSet: in.ServerSet, Lookup: d.ServerLookup, Docker: in.Docker, Env: d.Env, Report: d.Logf}).Probe(ctx, in.ProbeRequest)
}
