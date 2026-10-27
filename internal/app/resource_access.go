package app

import (
	"context"

	"github.com/0xmhha/chainbench/internal/resource"
)

// ResourceAccessIn selects a server and absolute root for an SSH access check.
type ResourceAccessIn struct {
	ServerSet, Server, DataRoot string
	// Docker dials the server through the localmap beside the server set, as
	// every other command does for a pool of local docker containers.
	Docker bool
}

// ResourceAccessOut reports authentication and permissions, without secrets.
type ResourceAccessOut = resource.AccessCheck

// ResourceCheckAccess shares the credential check with command and agent tools.
func ResourceCheckAccess(ctx context.Context, d Deps, in ResourceAccessIn) (ResourceAccessOut, error) {
	set, err := resource.LoadSet(in.ServerSet)
	if err != nil {
		return ResourceAccessOut{}, err
	}
	host, err := set.ByName(in.Server)
	if err != nil {
		return ResourceAccessOut{}, err
	}
	lookup := d.ServerLookup
	if lookup == nil {
		lookup = resource.SetLookup(in.ServerSet)
	}
	creds, err := lookup(host.Name)
	if err != nil {
		return ResourceAccessOut{}, err
	}
	if creds, err = (resource.Opener{ServerSet: in.ServerSet, Docker: in.Docker}).DialCredentials(creds); err != nil {
		return ResourceAccessOut{}, err
	}
	return resource.CheckAccess(ctx, host, creds, in.DataRoot)
}
