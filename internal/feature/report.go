package feature

import "github.com/0xmhha/chainbench/internal/app"

// The report stage: reading what a run left behind. Every one of these only
// looks, which is what the query projection and MCP's read-only tool list are
// derived from.
func init() {
	Register(Registration{
		Name: "report.session", Stage: StageReport, ReadOnly: true,
		Summary: "Show a run's report (from a session directory)",
	}, app.Report)
	Register(Registration{
		Name: "report.log", Stage: StageReport, ReadOnly: true,
		Summary: "Search a workspace's per-node logs",
	}, app.LogSearch)
	Register(Registration{
		Name: "report.timeline", Stage: StageReport, ReadOnly: true,
		Summary: "Merge per-node logs into one chronological timeline",
	}, app.LogTimeline)
	Register(Registration{
		Name: "network.status", Stage: StageReport, ReadOnly: true,
		Summary: "Show a composed network's node set (from its workspace)",
	}, app.NetworkStatus)
}
