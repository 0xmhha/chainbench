package feature

import (
	"context"

	"github.com/0xmhha/chainbench/internal/app"
)

// These typed registrations describe existing app operations. Registering does
// not execute them or expose private material through the read-only projection.
func init() {
	Register(Registration{Name: "blueprint.from-preset", Stage: StageCompose, ReadOnly: true, Summary: "Resolve a chain preset into its blueprint"}, app.BlueprintFromPreset)
	Register(Registration{Name: "blueprint.from-topology", Stage: StageCompose, ReadOnly: true, Summary: "Resolve a topology into its blueprint"}, app.BlueprintFromTopology)
	Register(Registration{Name: "chain.up", Stage: StageCompose, Summary: "Compose and start the declared network"}, app.ChainUp)
	Register(Registration{Name: "chain.stop", Stage: StageCompose, Summary: "Stop the workspace's owned nodes and keep their data"}, app.ChainStop)
	Register(Registration{Name: "chain.rm", Stage: StageCompose, Summary: "Remove the workspace's owned node data"}, app.ChainRm)
	Register(Registration{Name: "chain.resume", Stage: StageCompose, Summary: "Resume an interrupted composition from its recorded state"}, app.ChainResume)
	Register(Registration{Name: "network.stop", Stage: StageCompose, Summary: "Stop a named composed network"}, app.NetworkStop)
	Register(Registration{Name: "network.remove", Stage: StageCompose, Summary: "Remove a named composition and its node data"}, app.NetworkRemove)
	Register(Registration{Name: "network.map", Stage: StageCompose, ReadOnly: true, Summary: "Read the workspace's node placement and endpoints"}, app.NetworkMap)
	Register(Registration{Name: "network.pool", Stage: StageCompose, ReadOnly: true, Summary: "Inspect a server pool's capacity and occupied slots"}, app.NetworkPool)
	Register(Registration{Name: "network.plan", Stage: StageCompose, ReadOnly: true, Summary: "Preview placement without reserving slots or changing nodes"}, app.NetworkPlan)
	Register(Registration{Name: "node.start", Stage: StageCompose, Summary: "Start one owned node with its recorded launch inputs"}, app.NodeStart)
	Register(Registration{Name: "node.stop", Stage: StageCompose, Summary: "Stop one owned node while retaining its data"}, func(ctx context.Context, deps app.Deps, in app.NodeStopIn) (struct{}, error) {
		return struct{}{}, app.NodeStop(ctx, deps, in)
	})
	Register(Registration{Name: "node.rpc", Stage: StageTest, Summary: "Call an arbitrary node RPC method; the method may change state"}, app.NodeCall)
	Register(Registration{Name: "test.attach", Stage: StageTest, Summary: "Run a definition on an existing network with explicit ownership policy"}, app.AttachRun)
	Register(Registration{Name: "network.detect", Stage: StageReport, ReadOnly: true, Summary: "Detect a running network's chain and node roles"}, app.DetectNetwork)
	Register(Registration{Name: "network.verify", Stage: StageReport, ReadOnly: true, Summary: "Sample the network's health and block production"}, app.VerifyNetwork)
	Register(Registration{Name: "hardfork.plan", Stage: StageCompose, ReadOnly: true, Summary: "Preview a hardfork's binary and configuration changes"}, app.HardforkPlan)
	Register(Registration{Name: "hardfork.execute", Stage: StageCompose, Summary: "Apply a hardfork to the owned network"}, app.HardforkExecute)
	Register(Registration{Name: "session.gc", Stage: StageReport, Summary: "Inspect sessions and explicitly remove eligible old records"}, app.GCSessions)
	Register(Registration{Name: "tx.send", Stage: StageTest, Summary: "Sign and send a transaction"}, app.TxSend)
	Register(Registration{Name: "tx.wait", Stage: StageTest, ReadOnly: true, Summary: "Wait for a transaction receipt without sending a transaction"}, app.TxWait)
	Register(Registration{Name: "contract.deploy", Stage: StageTest, Summary: "Deploy a contract through a signed transaction"}, app.ContractDeploy)
	Register(Registration{Name: "contract.call", Stage: StageTest, ReadOnly: true, Summary: "Read a contract through eth_call"}, app.ContractCall)
	Register(Registration{Name: "account.faucet", Stage: StageTest, Summary: "Fund an account through the configured faucet key"}, app.Faucet)
	Register(Registration{Name: "account.faucet-key", Stage: StageTest, Summary: "Fund an account with an explicitly resolved key"}, app.FaucetFromKey)
	Register(Registration{Name: "account.state", Stage: StageReport, ReadOnly: true, Summary: "Read an account's balance and nonce"}, app.AccountState)
	Register(Registration{Name: "keyring.new", Stage: StageCompose, Summary: "Create a key ring with the requested identities"}, app.KeyringNew)
	Register(Registration{Name: "keyring.add", Stage: StageCompose, Summary: "Add identities to an existing key ring"}, app.KeyringAdd)
	Register(Registration{Name: "keyring.list", Stage: StageReport, ReadOnly: true, Summary: "List a key ring's public identity metadata"}, app.KeyringList)
	Register(Registration{Name: "keyring.show", Stage: StageReport, ReadOnly: true, Summary: "Show one identity's public material"}, app.KeyringShow)
	Register(Registration{Name: "keyring.export", Stage: StageCompose, Summary: "Export one identity; the result may contain private key material"}, app.KeyringExport)
	Register(Registration{Name: "keyring.import", Stage: StageCompose, Summary: "Import one identity into a key ring"}, app.KeyringImport)
	Register(Registration{Name: "keyring.import-ring", Stage: StageCompose, Summary: "Import a complete key ring"}, app.KeyringImportRing)
	Register(Registration{Name: "key.resolve", Stage: StageCompose, Summary: "Resolve private key material from a caller-selected source"}, app.ResolveKey)
}
