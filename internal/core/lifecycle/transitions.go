package lifecycle

// names is what each declared state is called, for messages and for the checks
// that walk the table. A state missing from here is one the table can still
// reach, so [TestEveryStateIsNamed] holds the two together.
var names = map[Status]string{
	ChainOpenWorkspace:                    "ChainOpenWorkspace",
	ChainOpenWorkspaceFailNoChain:         "ChainOpenWorkspaceFailNoChain",
	ChainBuildNodeTable:                   "ChainBuildNodeTable",
	ChainBuildNodeTableFailTwoLayouts:     "ChainBuildNodeTableFailTwoLayouts",
	ChainBuildNodeTableFailSetContended:   "ChainBuildNodeTableFailSetContended",
	ChainEnsureKeys:                       "ChainEnsureKeys",
	ChainEnsureKeysFromPreset:             "ChainEnsureKeysFromPreset",
	ChainEnsureKeysGenerated:              "ChainEnsureKeysGenerated",
	ChainEnsureKeysFromBlueprint:          "ChainEnsureKeysFromBlueprint",
	ChainEnsureKeysFailUnknownSource:      "ChainEnsureKeysFailUnknownSource",
	ChainEnsureKeysFailCountMismatch:      "ChainEnsureKeysFailCountMismatch",
	ChainEnsureKeysFailKeyNotLocal:        "ChainEnsureKeysFailKeyNotLocal",
	ChainEnsureKeysFailKeyUnreadable:      "ChainEnsureKeysFailKeyUnreadable",
	ChainBuildGenesis:                     "ChainBuildGenesis",
	ChainBuildGenesisFromTemplate:         "ChainBuildGenesisFromTemplate",
	ChainBuildGenesisFromExisting:         "ChainBuildGenesisFromExisting",
	ChainBuildGenesisForkApplied:          "ChainBuildGenesisForkApplied",
	ChainBuildGenesisVariantsWritten:      "ChainBuildGenesisVariantsWritten",
	ChainBuildGenesisFailExistingInvalid:  "ChainBuildGenesisFailExistingInvalid",
	ChainBuildGenesisFailExistingForeign:  "ChainBuildGenesisFailExistingForeign",
	ChainBuildGenesisFailForkUnresolved:   "ChainBuildGenesisFailForkUnresolved",
	ChainBuildGenesisFailDeclUnused:       "ChainBuildGenesisFailDeclUnused",
	ChainBuildGenesisFailTargetUnable:     "ChainBuildGenesisFailTargetUnable",
	ChainBuildNodeConfig:                  "ChainBuildNodeConfig",
	ChainBuildNodeConfigFailBadOverride:   "ChainBuildNodeConfigFailBadOverride",
	ChainBuildNodeConfigFailReadback:      "ChainBuildNodeConfigFailReadback",
	ChainBuildNodeConfigFailPinUnreadable: "ChainBuildNodeConfigFailPinUnreadable",
	ChainBuildNodeCommand:                 "ChainBuildNodeCommand",
	ChainBuildNodeCommandFailBadOption:    "ChainBuildNodeCommandFailBadOption",
	ChainDeployNodes:                      "ChainDeployNodes",
	ChainDeployNodesVerifiedLocal:         "ChainDeployNodesVerifiedLocal",
	ChainDeployNodesShippedRemote:         "ChainDeployNodesShippedRemote",
	ChainDeployNodesFailInputMissing:      "ChainDeployNodesFailInputMissing",
	ChainDeployNodesFailInputForeign:      "ChainDeployNodesFailInputForeign",
	ChainInitNodes:                        "ChainInitNodes",
	ChainInitNodesFailTargetUnable:        "ChainInitNodesFailTargetUnable",
	ChainInitNodesFailGenesisUnreadable:   "ChainInitNodesFailGenesisUnreadable",
	ChainInitNodesFailDatadir:             "ChainInitNodesFailDatadir",
	ChainLaunchNodes:                      "ChainLaunchNodes",
	ChainLaunchNodesPhaseLaunching:        "ChainLaunchNodesPhaseLaunching",
	ChainLaunchNodesPhaseActions:          "ChainLaunchNodesPhaseActions",
	ChainLaunchNodesPhaseDone:             "ChainLaunchNodesPhaseDone",
	ChainLaunchNodesFailNoBinary:          "ChainLaunchNodesFailNoBinary",
	ChainLaunchNodesFailPortBusy:          "ChainLaunchNodesFailPortBusy",
	ChainLaunchNodesFailOccupied:          "ChainLaunchNodesFailOccupied",
	ChainLaunchNodesFailNoKeystore:        "ChainLaunchNodesFailNoKeystore",
	ChainLaunchNodesFailPhaseEmpty:        "ChainLaunchNodesFailPhaseEmpty",
	ChainVerify:                           "ChainVerify",
	ChainVerifyProducing:                  "ChainVerifyProducing",
	ChainVerifyHaltedAsDeclared:           "ChainVerifyHaltedAsDeclared",
	ChainVerifyFailPlanMismatch:           "ChainVerifyFailPlanMismatch",
	ChainVerifyFailNotProducing:           "ChainVerifyFailNotProducing",
	ChainReady:                            "ChainReady",
	AdoptChain:                            "AdoptChain",
	AdoptChainByRPC:                       "AdoptChainByRPC",
	AdoptChainByWorkspace:                 "AdoptChainByWorkspace",
	AdoptChainByDeclaration:               "AdoptChainByDeclaration",
	AdoptChainFailUnreachable:             "AdoptChainFailUnreachable",
	AdoptChainFailWrongChain:              "AdoptChainFailWrongChain",
	CompareChain:                          "CompareChain",
	CompareChainSame:                      "CompareChainSame",
	CompareChainNodesDiffer:               "CompareChainNodesDiffer",
	CompareChainNetworkDiffers:            "CompareChainNetworkDiffers",
	CompareChainNothingComposed:           "CompareChainNothingComposed",
	CompareChainFailUnreadable:            "CompareChainFailUnreadable",
	ReconcileChain:                        "ReconcileChain",
	ReconcileChainAllKept:                 "ReconcileChainAllKept",
	ReconcileChainSomeRedone:              "ReconcileChainSomeRedone",
	ReconcileChainFailSharedInputChanged:  "ReconcileChainFailSharedInputChanged",
	ReconcileChainFailUnreadable:          "ReconcileChainFailUnreadable",
	FailRecordFormat:                      "FailRecordFormat",
	FailRecordSave:                        "FailRecordSave",
	FailWorkspaceConfig:                   "FailWorkspaceConfig",
	FailLoop:                              "FailLoop",
	FailNoHandler:                         "FailNoHandler",
	FailStageUnclassified:                 "FailStageUnclassified",
}

// allowed is every move this machine permits, and it is the whole rule.
//
// A move that is not here is refused, which is what replaces the checks each
// stage used to make by hand — seven of them asked "did the previous stage
// run?" in their own words, and the answer is now structural: a stage cannot be
// entered from anywhere that has not been through what precedes it.
//
// Three things are deliberately absent.
//
// No failure has an outgoing move. The two failures that looked retryable are
// not: the contended server set has already polled for ten seconds before it
// gets here, and a busy port needs somebody to stop something rather than more
// waiting. The shape admits a failure-to-entry edge; nothing registers one.
//
// The common failures are not listed as targets. Any state can reach them,
// because what they report is the workspace itself failing, and enumerating
// that against every state would be seventy lines saying one thing. [Machine]
// permits them from anywhere.
//
// The operational area has no entries yet. Its failures have not been measured
// the way the composition's were, and writing moves between states whose
// failures are unknown is how the first table came to have nowhere to put them.
var allowed = map[Status][]Status{
	// ---- composing, in the order the composition already runs ----
	ChainOpenWorkspace: {ChainBuildNodeTable, ChainOpenWorkspaceFailNoChain},

	ChainBuildNodeTable: {ChainEnsureKeys,
		ChainBuildNodeTableFailTwoLayouts, ChainBuildNodeTableFailSetContended},

	ChainEnsureKeys: {ChainEnsureKeysFromPreset, ChainEnsureKeysGenerated,
		ChainEnsureKeysFromBlueprint, ChainEnsureKeysFailUnknownSource,
		ChainEnsureKeysFailCountMismatch, ChainEnsureKeysFailKeyNotLocal,
		ChainEnsureKeysFailKeyUnreadable},
	// A composition that reconciles against a running network goes to the
	// reconciliation here: after the key set is in place, and before the genesis
	// stage, which is the first that writes to the target. Judging later meant a
	// refusal that had already overwritten the running network's genesis.
	ChainEnsureKeysFromPreset:    {ChainBuildGenesis, ReconcileChain},
	ChainEnsureKeysGenerated:     {ChainBuildGenesis, ReconcileChain},
	ChainEnsureKeysFromBlueprint: {ChainBuildGenesis, ReconcileChain},

	ReconcileChain: {ReconcileChainAllKept, ReconcileChainSomeRedone,
		ReconcileChainFailSharedInputChanged, ReconcileChainFailUnreadable},
	ReconcileChainAllKept:    {ChainBuildGenesis},
	ReconcileChainSomeRedone: {ChainBuildGenesis},

	ChainBuildGenesis: {ChainBuildGenesisFromTemplate, ChainBuildGenesisFromExisting,
		ChainBuildGenesisFailExistingInvalid, ChainBuildGenesisFailExistingForeign,
		ChainBuildGenesisFailTargetUnable},
	ChainBuildGenesisFromTemplate: {ChainBuildGenesisForkApplied,
		ChainBuildGenesisVariantsWritten, ChainBuildNodeConfig,
		ChainBuildGenesisFailForkUnresolved, ChainBuildGenesisFailDeclUnused},
	ChainBuildGenesisFromExisting: {ChainBuildGenesisForkApplied,
		ChainBuildGenesisVariantsWritten, ChainBuildNodeConfig,
		ChainBuildGenesisFailForkUnresolved, ChainBuildGenesisFailDeclUnused},
	ChainBuildGenesisForkApplied:     {ChainBuildGenesisVariantsWritten, ChainBuildNodeConfig},
	ChainBuildGenesisVariantsWritten: {ChainBuildNodeConfig},

	ChainBuildNodeConfig: {ChainBuildNodeCommand,
		ChainBuildNodeConfigFailBadOverride, ChainBuildNodeConfigFailReadback,
		ChainBuildNodeConfigFailPinUnreadable},

	ChainBuildNodeCommand: {ChainDeployNodes, ChainBuildNodeCommandFailBadOption},

	// The two detail states say what the target was, not which way the walk
	// went, so a deploy that cannot say which it did still moves on. The stage
	// cannot say yet: whether anything was shipped is counted inside
	// Workspace.Provision and does not come back out, and a handler that
	// guessed from the request would call a local server set remote.
	ChainDeployNodes: {ChainInitNodes,
		ChainDeployNodesVerifiedLocal, ChainDeployNodesShippedRemote,
		ChainDeployNodesFailInputMissing, ChainDeployNodesFailInputForeign},
	ChainDeployNodesVerifiedLocal: {ChainInitNodes},
	ChainDeployNodesShippedRemote: {ChainInitNodes},

	// The busy-port failure belongs to the launch and is listed here too,
	// because the init stage asks the same question before it writes anything.
	// It is one condition with one state rather than two: what failed is the
	// ports the launch needs, whoever noticed. Naming a second state for it
	// would put one concept under two names, and a reader comparing two runs
	// would have to know which stage happened to ask.
	ChainInitNodes: {ChainLaunchNodes, ChainInitNodesFailTargetUnable,
		ChainInitNodesFailGenesisUnreadable, ChainInitNodesFailDatadir,
		ChainLaunchNodesFailPortBusy},

	ChainLaunchNodes: {ChainLaunchNodesPhaseLaunching,
		ChainLaunchNodesFailNoBinary, ChainLaunchNodesFailPortBusy,
		ChainLaunchNodesFailOccupied},
	ChainLaunchNodesPhaseLaunching: {ChainLaunchNodesPhaseActions,
		ChainLaunchNodesPhaseDone, ChainLaunchNodesFailNoKeystore,
		ChainLaunchNodesFailPortBusy, ChainLaunchNodesFailOccupied},
	ChainLaunchNodesPhaseActions: {ChainLaunchNodesPhaseDone, ChainLaunchNodesFailPhaseEmpty},
	// A family may declare several phases, so done goes back to launching for
	// the next one. This is the only cycle in the composition, and it is bounded
	// by the phase count rather than by the entry limit.
	ChainLaunchNodesPhaseDone: {ChainLaunchNodesPhaseLaunching, ChainVerify},

	ChainVerify: {ChainVerifyProducing, ChainVerifyHaltedAsDeclared,
		ChainVerifyFailPlanMismatch, ChainVerifyFailNotProducing},
	ChainVerifyProducing:        {ChainReady},
	ChainVerifyHaltedAsDeclared: {ChainReady},

	// ---- adopting a chain that is already up ----
	AdoptChain: {AdoptChainByRPC, AdoptChainByWorkspace, AdoptChainByDeclaration,
		AdoptChainFailUnreachable, AdoptChainFailWrongChain},
	AdoptChainByRPC:         {CompareChain},
	AdoptChainByWorkspace:   {CompareChain},
	AdoptChainByDeclaration: {CompareChain},

	CompareChain: {CompareChainSame, CompareChainNodesDiffer,
		CompareChainNetworkDiffers, CompareChainNothingComposed,
		CompareChainFailUnreadable},
	// Every route out of the comparison ends at the same question: is the
	// network the one that was planned. The first draft of this table sent a
	// reused network straight to ChainReady, which read as "a network nobody
	// composed needs no checking" — and it is not what the code does either.
	// The readiness gate runs whatever the verdict was.
	CompareChainSame: {ChainVerify},
	// Restarting the nodes that differ is what preflight does: it does not
	// compose again. The restart IS the launch work for those nodes, so this
	// goes to the check rather than through ChainLaunchNodes, which would run
	// a start over a network where nothing is stopped and re-mark the step.
	CompareChainNodesDiffer: {ChainVerify},
	// Composing from the beginning. Bounded by the entry limit, which is what
	// keeps a network that keeps failing to match from looping forever.
	CompareChainNetworkDiffers:  {ChainOpenWorkspace},
	CompareChainNothingComposed: {ChainOpenWorkspace},
}

// entryLimit is how many times one BLOCK may be entered in a single run.
//
// One, because the only move that goes backwards between blocks is a comparison
// deciding to compose again, and today that decision is taken exactly once. A
// second arrival means the composition it asked for did not satisfy the
// comparison that asked for it, which is a loop rather than progress.
//
// It bounds blocks and not states on purpose. The launch stage walks its own
// states once per phase, and a family declares how many phases it has, so that
// count bounds the cycle; a limit that counted states would refuse the second
// producer.
const entryLimit = 1

// Detailed reports whether a stage MUST walk states of its own before it moves
// on — that is, whether the table gives it no way out of its own block.
//
// The question is not "does this stage have detail states". Two kinds of stage
// have them and they are not the same. The key stage's three sources are the
// only ways out, so a key step that named none went somewhere the table does
// not allow; the deploy stage's two say what the target was and it may move on
// without naming either, which is what it does while whether anything was
// shipped is still counted inside Workspace.Provision and never comes back out.
//
// Read off the table rather than declared a second time beside it, and kept
// apart from "the stage classifies its failures" — that is a different thing a
// stage can do on its own, and confusing the two made every stage that did one
// demand the other.
func Detailed(entry Status) bool {
	inside, outside := false, false
	for _, to := range allowed[entry] {
		if to.IsFailure() {
			continue
		}
		if to.Block() == entry.Block() {
			inside = true
		} else {
			outside = true
		}
	}
	return inside && !outside
}
