package feature

import "github.com/0xmhha/chainbench/internal/app"

// File transfer to and from a server's data plane. They belong to the compose
// stage: like deploy, they act on the target's data plane — placing a
// replacement binary, or pulling a node's log or a key back off a server.
// Neither is read-only: upload writes on the target, download writes locally.
func init() {
	Register(Registration{
		Name: "file.upload", Stage: StageCompose,
		Summary: "Upload local files to a server (refuses a name collision unless forced)",
	}, app.Upload)
	Register(Registration{
		Name: "file.download", Stage: StageCompose,
		Summary: "Download a server file to a local path (elevates through sudo where permitted)",
	}, app.Download)
}
