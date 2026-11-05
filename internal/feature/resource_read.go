package feature

import "github.com/0xmhha/chainbench/internal/app"

func init() {
	Register(Registration{Name: "resource.inspect", Stage: StageCompose, ReadOnly: true, Summary: "Observe target identity, files, bounded logs and recorded process without changes"}, app.ResourceRead)
	// Runs a program on the target, so it is not ReadOnly and no read-only tool
	// may reach it.
	Register(Registration{Name: "resource.probe", Stage: StageCompose, Summary: "Run one inspection of a binary in a remote target's data root"}, app.ResourceProbe)
	Register(Registration{Name: "resource.access", Stage: StageCompose, ReadOnly: true, Summary: "Check server authentication and target permission without changing files"}, app.ResourceCheckAccess)
}
