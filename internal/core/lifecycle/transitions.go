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
	FailRecordFormat:                      "FailRecordFormat",
	FailRecordSave:                        "FailRecordSave",
	FailWorkspaceConfig:                   "FailWorkspaceConfig",
	FailLoop:                              "FailLoop",
	FailNoHandler:                         "FailNoHandler",
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
	ChainEnsureKeysFromPreset:    {ChainBuildGenesis},
	ChainEnsureKeysGenerated:     {ChainBuildGenesis},
	ChainEnsureKeysFromBlueprint: {ChainBuildGenesis},

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

	ChainDeployNodes: {ChainDeployNodesVerifiedLocal, ChainDeployNodesShippedRemote,
		ChainDeployNodesFailInputMissing, ChainDeployNodesFailInputForeign},
	ChainDeployNodesVerifiedLocal: {ChainInitNodes},
	ChainDeployNodesShippedRemote: {ChainInitNodes},

	ChainInitNodes: {ChainLaunchNodes, ChainInitNodesFailTargetUnable,
		ChainInitNodesFailGenesisUnreadable, ChainInitNodesFailDatadir},

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
	CompareChainSame: {ChainReady},
	// Restarting the nodes that differ, which is what preflight does today: it
	// does not compose again.
	CompareChainNodesDiffer: {ChainLaunchNodes},
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
