package feature

import "github.com/0xmhha/chainbench/internal/app"

func init() {
	Register(Registration{Name: "resource.inspect", Stage: StageCompose, ReadOnly: true, Summary: "Observe target identity, files, bounded logs and recorded process without changes"}, app.ResourceRead)
	Register(Registration{Name: "resource.access", Stage: StageCompose, ReadOnly: true, Summary: "Check server authentication and target permission without changing files"}, app.ResourceCheckAccess)
}
