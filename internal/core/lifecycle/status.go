package lifecycle

import "fmt"

// Status is where a run stands, as one value.
//
// The value carries three facts, which is what lets the machine work without
// knowing any name:
//
//	0xA000   area   which lifecycle — a chain being composed, a test being run
//	0x0B00   block  which stage of it
//	0x00CD   slot   where inside that stage
//
// A slot below 0x80 is progress and a slot at or above it is a failure, so
// [Status.IsFailure] is a comparison rather than a lookup. Blocks are 0x100
// apart and areas 0x1000, which leaves 127 progress slots and 128 failure slots
// per stage; the largest stage today uses four and five.
//
// It was tempting to give each stage one value and let the next stage be the
// next number. That shape has nowhere to put a failure: BASE+1 is already the
// following stage, so a stage that fails has no state to move to and the code
// goes back to returning errors that nobody can branch on.
type Status uint32

// blockSize is how many slots one stage owns, and failureSlot is the first of
// them that means the stage failed.
const (
	blockSize   = 0x100
	failureSlot = 0x80
)

// IsFailure reports whether this is a failure state.
func (s Status) IsFailure() bool { return uint32(s)%blockSize >= failureSlot }

// Block is the stage this state belongs to — the state at slot zero.
//
// Handlers are registered per block, so this is what the machine looks up. A
// stage's progress and failure states all reach the same handler, which is what
// lets that handler hold the whole stage's logic in one switch.
func (s Status) Block() Status { return s - Status(uint32(s)%blockSize) }

// String is the declared name, or the hex value when nothing declares it.
//
// An undeclared value printing as hex rather than as "Status(4352)" is
// deliberate: the blocks are hex-shaped, so the number a reader sees is the
// number they can look up in the table.
func (s Status) String() string {
	if n, ok := names[s]; ok {
		return n
	}
	return fmt.Sprintf("Status(%#x)", uint32(s))
}

// ---- areas -----------------------------------------------------------------

const (
	// areaChain is a chain being made ready to take tests.
	areaChain Status = 0x1000
	// areaAdopt is a chain that is already up being taken over.
	areaAdopt Status = 0x2000
	// areaChainOp is what happens to a chain after it is ready.
	areaChainOp Status = 0x3000
	// areaTest is testengine's own lifecycle. It is a separate area so that one
	// machine can hold both without the two sets of values meeting.
	areaTest Status = 0x8000
	// areaCommon is the failures that belong to no stage, because the
	// workspace itself is what broke.
	areaCommon Status = 0x0F00
)

// ---- chain: composing ------------------------------------------------------
//
// The order is the one the composition already runs in. This does not reorder
// it; it writes down what upStepNames has said all along.

const (
	// ChainOpenWorkspace opens the workspace and records the request.
	ChainOpenWorkspace Status = areaChain + 0x100*iota
	// ChainBuildNodeTable builds the node table: roles, paths, ports.
	ChainBuildNodeTable
	// ChainEnsureKeys sees whether the key set exists and makes one if not.
	ChainEnsureKeys
	// ChainBuildGenesis builds the genesis from the key set and writes it.
	ChainBuildGenesis
	// ChainBuildNodeConfig renders each node's config and writes it.
	ChainBuildNodeConfig
	// ChainBuildNodeCommand assembles each node's launch command.
	ChainBuildNodeCommand
	// ChainDeployNodes puts the launch inputs on the target and checks them.
	ChainDeployNodes
	// ChainInitNodes initializes each node's datadir.
	ChainInitNodes
	// ChainLaunchNodes launches the stopped nodes and records their pids.
	ChainLaunchNodes
	// ChainVerify asks whether what came up is what was planned.
	//
	// Its handler is registered by whoever owns the plan, which is not this
	// package and not chainsetup: the comparison needs the composed plan and
	// the node monitor, both of which sit above the composition.
	ChainVerify
)

// ChainReady is the state a composition is trying to reach. It sits at the top
// of the area rather than after ChainVerify so that a stage added later does
// not have to move it.
const ChainReady Status = areaChain + 0xF00

// ---- chain: stage detail ---------------------------------------------------

const (
	// ChainEnsureKeysFromPreset took the identities from a committed key set.
	ChainEnsureKeysFromPreset = ChainEnsureKeys + 1 + iota
	// ChainEnsureKeysGenerated made fresh ones.
	ChainEnsureKeysGenerated
	// ChainEnsureKeysFromBlueprint took what the network declaration names.
	ChainEnsureKeysFromBlueprint
)

const (
	// ChainBuildGenesisFromTemplate built it from the chain's own template.
	ChainBuildGenesisFromTemplate = ChainBuildGenesis + 1 + iota
	// ChainBuildGenesisFromExisting used a finished genesis as it stands.
	ChainBuildGenesisFromExisting
	// ChainBuildGenesisForkApplied scheduled a fork on it.
	ChainBuildGenesisForkApplied
	// ChainBuildGenesisVariantsWritten wrote the per-binary documents.
	ChainBuildGenesisVariantsWritten
)

const (
	// ChainDeployNodesVerifiedLocal checked the inputs where they already are.
	ChainDeployNodesVerifiedLocal = ChainDeployNodes + 1 + iota
	// ChainDeployNodesShippedRemote sent the identity files to the target.
	ChainDeployNodesShippedRemote
)

const (
	// ChainLaunchNodesPhaseLaunching is launching one phase's nodes. A family
	// may declare several: one launches everything at once, another boots a
	// single producer and joins the rest one at a time.
	ChainLaunchNodesPhaseLaunching = ChainLaunchNodes + 1 + iota
	// ChainLaunchNodesPhaseActions runs the work a phase names after its nodes
	// are up and before the next phase starts.
	ChainLaunchNodesPhaseActions
	// ChainLaunchNodesPhaseDone finished one phase. Another may follow.
	ChainLaunchNodesPhaseDone
)

const (
	// ChainVerifyProducing saw blocks being made.
	ChainVerifyProducing = ChainVerify + 1 + iota
	// ChainVerifyHaltedAsDeclared saw the chain stop where it said it would.
	ChainVerifyHaltedAsDeclared
)

// ---- chain: failures -------------------------------------------------------

const (
	// ChainOpenWorkspaceFailNoChain: neither a chain nor a manifest was named.
	ChainOpenWorkspaceFailNoChain = ChainOpenWorkspace + failureSlot + iota
)

const (
	// ChainBuildNodeTableFailTwoLayouts: a blueprint and a topology both
	// describe the layout, so neither can be taken as the answer.
	ChainBuildNodeTableFailTwoLayouts = ChainBuildNodeTable + failureSlot + iota
	// ChainBuildNodeTableFailSetContended: another run holds the server set.
	// The allocator already waited before reaching this.
	ChainBuildNodeTableFailSetContended
)

const (
	// ChainEnsureKeysFailUnknownSource: the declaration names a source that is
	// none of the three.
	ChainEnsureKeysFailUnknownSource = ChainEnsureKeys + failureSlot + iota
	// ChainEnsureKeysFailCountMismatch: the declaration and the node table
	// disagree about how many identities there are.
	ChainEnsureKeysFailCountMismatch
	// ChainEnsureKeysFailKeyNotLocal: a key is named inline or on a server,
	// and a private key is read from a local file or not at all.
	ChainEnsureKeysFailKeyNotLocal
	// ChainEnsureKeysFailKeyUnreadable: the file is named and cannot be read,
	// or holds no key.
	ChainEnsureKeysFailKeyUnreadable
)

const (
	// ChainBuildGenesisFailExistingInvalid: the supplied genesis is not JSON.
	ChainBuildGenesisFailExistingInvalid = ChainBuildGenesis + failureSlot + iota
	// ChainBuildGenesisFailExistingForeign: it is valid and belongs to another
	// key set, which would leave every node unable to seal.
	ChainBuildGenesisFailExistingForeign
	// ChainBuildGenesisFailForkUnresolved: the fork names no name, no chain,
	// no section or no successor.
	ChainBuildGenesisFailForkUnresolved
	// ChainBuildGenesisFailDeclUnused: a genesis is declared for a binary no
	// node runs, so the declaration would be silently ignored.
	ChainBuildGenesisFailDeclUnused
	// ChainBuildGenesisFailTargetUnable: there is no producer to generate on,
	// or the target cannot run a command.
	ChainBuildGenesisFailTargetUnable
)

const (
	// ChainBuildNodeConfigFailBadOverride: an override is not key=value, or
	// names a scope that is not one.
	ChainBuildNodeConfigFailBadOverride = ChainBuildNodeConfig + failureSlot + iota
	// ChainBuildNodeConfigFailReadback: what the target holds is not what was
	// written.
	ChainBuildNodeConfigFailReadback
	// ChainBuildNodeConfigFailPinUnreadable: a pinned config or the genesis it
	// carries cannot be read.
	ChainBuildNodeConfigFailPinUnreadable
)

const (
	// ChainBuildNodeCommandFailBadOption: a launch option is malformed or
	// names a scope that is not one.
	ChainBuildNodeCommandFailBadOption = ChainBuildNodeCommand + failureSlot + iota
	// ChainBuildNodeCommandFailSplitNetwork: the assembled commands do not all
	// name the same devp2p network, so the nodes would not peer. It is its own
	// state rather than a bad option because the composition is what produced
	// the split — an option can cause it, and so can an assembler that took the
	// network from the wrong place.
	ChainBuildNodeCommandFailSplitNetwork
)

const (
	// ChainDeployNodesFailInputMissing: something the launch needs is not on
	// the target.
	ChainDeployNodesFailInputMissing = ChainDeployNodes + failureSlot + iota
	// ChainDeployNodesFailInputForeign: it is there and is not the file this
	// workspace built.
	ChainDeployNodesFailInputForeign
)

const (
	// ChainInitNodesFailTargetUnable: the target's driver cannot initialize a
	// datadir.
	ChainInitNodesFailTargetUnable = ChainInitNodes + failureSlot + iota
	// ChainInitNodesFailGenesisUnreadable: the built genesis cannot be read
	// back.
	ChainInitNodesFailGenesisUnreadable
	// ChainInitNodesFailDatadir: a datadir cannot be cleared or written.
	ChainInitNodesFailDatadir
)

const (
	// ChainLaunchNodesFailNoBinary: none is set, or it is not on the target.
	ChainLaunchNodesFailNoBinary = ChainLaunchNodes + failureSlot + iota
	// ChainLaunchNodesFailPortBusy: a planned port is taken. Somebody has to
	// act; waiting does not clear it.
	ChainLaunchNodesFailPortBusy
	// ChainLaunchNodesFailOccupied: a node is already running outside this
	// workspace.
	ChainLaunchNodesFailOccupied
	// ChainLaunchNodesFailNoKeystore: a node that has to seal has no keystore.
	ChainLaunchNodesFailNoKeystore
	// ChainLaunchNodesFailPhaseEmpty: a phase names work and launched no node
	// to run it on.
	ChainLaunchNodesFailPhaseEmpty
)

const (
	// ChainVerifyFailPlanMismatch: what came up is not what was planned.
	ChainVerifyFailPlanMismatch = ChainVerify + failureSlot + iota
	// ChainVerifyFailNotProducing: no block arrived inside the budget.
	ChainVerifyFailNotProducing
)

// ChainStopped and ChainRemoved are what an operational verb leaves behind when
// it does not leave the network running.
//
// They are chain states rather than operational ones: the area above names what
// is being DONE to a chain, and these name what the chain then IS. A reader
// asking what a workspace holds needs the difference — one can be started
// again, the other has no data plane left to start.
const (
	ChainRemoved Status = areaChain + 0xD00
	ChainStopped Status = areaChain + 0xE00
)

// ---- chain: what is done to it once it is ready ----------------------------
//
// The operational area is not the composition with different names, and the
// measurement is why (state-machine-04-operational-failures.md). There, nine
// stages each had their own failures: 118 belonged to exactly one, 16 were
// shared. Here 25 of 65 are shared, and four of the six verbs have one or two
// of their own — stop, rm, restart and hardfork have nine between them.
//
// What they share is not a verb but three moves: decide which binary to launch,
// render a node's config again, take a node down and bring it back. So the
// blocks below are named for what happens, and the failures they share are
// declared once and reachable from each of them. Two sets are not declared here
// at all: rendering a config again fails the way the config stage already
// fails, and re-initializing one node's datadir fails the way the init stage
// does. Giving those second names would put one condition under two.

const (
	// ChainOpStopNodes halts what is running, the whole network or one node.
	ChainOpStopNodes Status = areaChainOp + 0x100*iota
	// ChainOpStartNodes brings a node back on the argv it was launched with.
	ChainOpStartNodes
	// ChainOpReplaceNode gives one node a different binary, config or genesis
	// and brings it back on that.
	ChainOpReplaceNode
	// ChainOpCrossFork restarts the network across a fork boundary. It is the
	// only operational verb that waits on the chain rather than on a process,
	// and the only one with states of its own.
	ChainOpCrossFork
	// ChainOpRemoveNodes removes the composed data plane.
	ChainOpRemoveNodes
	// ChainOpHardfork runs the upgrade a hardfork plan describes.
	ChainOpHardfork
)

const (
	// ChainOpCrossForkBeforeFork saw the chain short of the fork block, which
	// is where the hand-over has to begin.
	ChainOpCrossForkBeforeFork = ChainOpCrossFork + 1 + iota
	// ChainOpCrossForkHandingOver has the pre-fork nodes down and the post-fork
	// build coming up in their place.
	ChainOpCrossForkHandingOver
	// ChainOpCrossForkCrossed saw the chain past the fork block on the new
	// build.
	ChainOpCrossForkCrossed
)

// opShared is the block the failures more than one operational verb can raise
// live in.
//
// A block of its own, at the top of the area, because the first draft put them
// at areaChainOp + failureSlot — which is the STOP block's failure range, since
// stop is the area's first block. Two names, one value. The area has sixteen
// blocks and the composition already keeps its terminal state at the top of
// its own, so this is the same shape rather than a new one.
const opShared Status = areaChainOp + 0xF00

// The failures several operational verbs share. They are declared once because
// they are one condition: a caller reading two runs should not have to know
// which verb happened to ask.
const (
	// ChainOpFailPrecondition: the workspace is not in a state this verb can
	// act on — no node table, a node still running, a node never started.
	//
	// The composition has no equivalent because the transition table removes
	// its preconditions: a stage cannot be entered without the one before it.
	// Operational verbs have no such order — stop may be followed by rm or by
	// restart and neither is wrong — so this one stays a state.
	ChainOpFailPrecondition = opShared + failureSlot + iota
	// ChainOpFailNoSuchNode: the index named is not in the node table.
	ChainOpFailNoSuchNode
)

const (
	// ChainOpStopNodesFailSomeStillUp: not every node came down, and the
	// failure names which. Every node is attempted before this is raised.
	ChainOpStopNodesFailSomeStillUp = ChainOpStopNodes + failureSlot
	// ChainOpReplaceNodeFailNothingAsked: a swap that names no binary, no
	// config change and no genesis overlay has nothing to do.
	ChainOpReplaceNodeFailNothingAsked = ChainOpReplaceNode + failureSlot
)

const (
	// ChainOpCrossForkFailNoFork: this network is composed to cross no fork, or
	// no node runs the build that would take over.
	ChainOpCrossForkFailNoFork = ChainOpCrossFork + failureSlot + iota
	// ChainOpCrossForkFailHeadUnreadable: the chain's head could not be read,
	// or did not arrive inside the budget.
	ChainOpCrossForkFailHeadUnreadable
	// ChainOpCrossForkFailAlreadyPast: the chain is past the fork block, so the
	// hand-over cannot be the thing that crosses it.
	ChainOpCrossForkFailAlreadyPast
	// ChainOpCrossForkFailNobodyCameBack: the restart left no node running.
	ChainOpCrossForkFailNobodyCameBack
)

// ---- adopt: a chain that is already up -------------------------------------

const (
	// AdoptChain takes over a network this run did not compose.
	AdoptChain Status = areaAdopt + 0x100*iota
	// CompareChain asks whether what is there is what is wanted.
	CompareChain
	// ReconcileChain asks the same question from inside a composition that has
	// already started, which is a different question.
	//
	// CompareChain runs before anything is written and may answer "compose it
	// again". This one runs after the key set is in place and before the first
	// stage that writes to the target, so composing again is not one of its
	// answers: a shared input that changed is a refusal, and what it can do is
	// keep the nodes that still match and tear down the ones that do not.
	ReconcileChain
)

const (
	// AdoptChainByRPC knows only the endpoints.
	AdoptChainByRPC = AdoptChain + 1 + iota
	// AdoptChainByWorkspace has the record a composition left.
	AdoptChainByWorkspace
	// AdoptChainByDeclaration was told where to attach by the case's own env.
	AdoptChainByDeclaration
)

const (
	// CompareChainSame: what is composed serves what is wanted.
	CompareChainSame = CompareChain + 1 + iota
	// CompareChainNodesDiffer: the shape is right and some nodes are not.
	CompareChainNodesDiffer
	// CompareChainNetworkDiffers: a network-wide fact differs, so composing
	// starts again.
	CompareChainNetworkDiffers
	// CompareChainNothingComposed: there is nothing on the target yet.
	CompareChainNothingComposed
)

const (
	// ReconcileChainAllKept: every node still matches what this run will write,
	// so the composition goes on and every one of them stays up.
	ReconcileChainAllKept = ReconcileChain + 1 + iota
	// ReconcileChainSomeRedone: the nodes that drifted were torn down. The
	// composition goes on, and the later stages bring those back — init and
	// start skip a node that still carries a pid, which is what leaves the rest
	// running.
	ReconcileChainSomeRedone
)

const (
	// ReconcileChainFailSharedInputChanged: something every node shares is not
	// what the running network was built on — the genesis above all. A running
	// network cannot be reconciled onto a new chain.
	ReconcileChainFailSharedInputChanged = ReconcileChain + failureSlot + iota
	// ReconcileChainFailUnreadable: what this run would write could not be
	// rendered, or the target could not be read, so there was nothing to
	// compare.
	ReconcileChainFailUnreadable
)

const (
	// AdoptChainFailUnreachable: the endpoints do not answer.
	AdoptChainFailUnreachable = AdoptChain + failureSlot + iota
	// AdoptChainFailWrongChain: they answer for a different chain.
	AdoptChainFailWrongChain
)

// CompareChainFailUnreadable: the record cannot be read, so nothing can be
// compared against it.
const CompareChainFailUnreadable = CompareChain + failureSlot

// ---- common failures -------------------------------------------------------

const (
	// FailRecordFormat: the composition record is a format this build does not
	// know.
	FailRecordFormat = areaCommon + failureSlot + iota
	// FailRecordSave: the record cannot be written back.
	FailRecordSave
	// FailWorkspaceConfig: the workspace configuration cannot be read.
	FailWorkspaceConfig
	// FailLoop: a block was entered more times than the table allows.
	FailLoop
	// FailNoHandler: a state was reached that nothing owns. It is a failure
	// rather than a skip because a stage nobody handles is a stage nobody
	// wrote, and passing it silently is how a composition comes to skip work
	// it was asked for.
	FailNoHandler
	// FailStageUnclassified: a stage failed and its reason is in the error
	// rather than in a state.
	//
	// It is a debt, not a design. Every stage has failure states of its own,
	// and a handler reaches for this one only while the work it drives still
	// returns a string nobody can branch on. A handler that uses it says which
	// stage it stands for, and the list of handlers still doing so only
	// shrinks — when a stage's work moves into its handler, its failures move
	// with it and this stops being reachable from there.
	FailStageUnclassified
)
