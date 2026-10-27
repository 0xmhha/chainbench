// @ts-check
// Types of the Web responses the screens read, following
// docs/dev/web-domain-model.md. The decoders in api-decode.mjs check a
// response against these before a screen sees it.

/**
 * @typedef {object} Gap
 * @property {string} from
 * @property {string} to
 * @property {'node'|'network'|string} scope
 * @property {string} [nodeId]
 * @property {string} source
 * @property {string} code
 * @property {string} [metric] the gauge a not_exposed gap is about
 * @property {string} [runId] the run a deleted_by_admin gap records
 */

/**
 * @typedef {object} Coverage
 * @property {string} from
 * @property {string} to
 * @property {number} sampleIntervalSeconds
 * @property {boolean} complete
 * @property {boolean} downsampled
 * @property {number} [bucketSeconds]
 * @property {string} [timestampSource]
 * @property {Gap[]} gaps
 */

/**
 * @typedef {object} NodeView
 * @property {string} id
 * @property {string} ref "<networkId>.<id>", what the logs API takes
 * @property {string} networkId
 * @property {string} role
 * @property {string} [server] the server-set entry the node runs on
 * @property {string} [syncMode]
 * @property {{p2p?: number, http?: number, ws?: number, metrics?: number}} [ports] ports the node listens on
 * @property {string} state
 * @property {string} [observationReason]
 * @property {string} [observationDetail] why the node could not be observed
 * @property {string} [executable] ready, unavailable or unknown
 * @property {number} pid
 * @property {number} [observedPid]
 * @property {string} observedAt
 * @property {string[]} supportedControls
 * @property {Record<string, string>} controlLimits
 */

/**
 * @typedef {object} Network
 * @property {string} id
 * @property {string} workspaceId
 * @property {string} ownership
 * @property {number} version
 * @property {NodeView[]} nodes
 * @property {{firstCollected?: string, lastCollected?: string, intervalSeconds: number, remoteLogs?: string}} [collector]
 */

/**
 * @typedef {object} MetricSample
 * @property {string} time
 * @property {number|null} value
 */

/**
 * @typedef {object} MetricSeries
 * @property {string} nodeId
 * @property {string} name the collected name
 * @property {string} metric the concept, shared by both sources of a value
 * @property {boolean} canonical false for the second source of a concept
 * @property {string} source
 * @property {string} [unit]
 * @property {MetricSample[]} samples
 */

/**
 * @typedef {object} Metrics
 * @property {string} networkId
 * @property {MetricSeries[]} series
 * @property {Coverage} coverage
 */

/**
 * @typedef {object} LogEntry
 * @property {string} time
 * @property {string} text
 * @property {string} [level]
 * @property {string} [timeSource] source (the node's own timestamp) or collection
 */

/**
 * @typedef {object} Logs
 * @property {string} nodeId
 * @property {LogEntry[]} entries
 * @property {number} total
 * @property {Coverage} coverage
 * @property {string|null} nextCursor
 */

/**
 * @typedef {object} Phase
 * @property {string} name
 * @property {string} state
 * @property {string} startedAt
 * @property {string} [finishedAt] absent while the phase runs
 * @property {string} [message]
 */

/**
 * @typedef {object} Job
 * @property {string} id
 * @property {string} actorId
 * @property {string} workspaceId
 * @property {string} planId
 * @property {string} operation
 * @property {string} state
 * @property {boolean} terminal
 * @property {boolean} needsCheck
 * @property {Phase[]} phases
 * @property {string} createdAt
 * @property {string} retention
 * @property {string} nodeDisposition
 * @property {string} [cancelReason]
 * @property {string} [cancelMessage]
 * @property {string[]} runIds
 * @property {string[]} partialEffects
 * @property {string[]} unresolvedResources
 * @property {{code: string, message: string, requestId?: string}} [error]
 * @property {object} [reconciliation]
 */

/**
 * @typedef {object} Assertion
 * @property {string} assert
 * @property {string} [nodeId]
 * @property {string} [compare]
 * @property {string} expected decimal text when the engine recorded a number
 * @property {string} [actual]
 * @property {boolean} passed
 * @property {string} [detail]
 * @property {Record<string, unknown>} [provenance]
 */

/**
 * @typedef {object} Step
 * @property {number} index
 * @property {string} type
 * @property {string} [nodeId]
 * @property {string} [signer]
 * @property {string} [hash]
 * @property {string} [nonce]
 * @property {string} [gas]
 * @property {string} [error]
 */

/**
 * @typedef {object} RunTest
 * @property {number} seq
 * @property {string} id
 * @property {string} state
 * @property {string} [reason]
 * @property {string} [startedAt]
 * @property {string} [finishedAt]
 * @property {string} [env]
 * @property {Assertion[]} assertions
 * @property {Step[]} steps
 * @property {{name: string, ok: boolean, detail?: string}[]} postActions
 * @property {{kind: string, ref: string, node?: string}[]} artifacts
 */

/**
 * @typedef {object} Run
 * @property {string} id
 * @property {string} [jobId]
 * @property {string} workspaceId
 * @property {string} chain
 * @property {string} [actorId]
 * @property {string} state
 * @property {string} [startedAt]
 * @property {string} [finishedAt]
 * @property {string[]} sessionRefs
 * @property {Record<string, string>} fingerprints
 * @property {Record<string, any>} summary
 * @property {string[]} artifactRefs
 * @property {Coverage} [coverage]
 * @property {RunTest[]} [tests] in detail reads of engine sessions
 */

/**
 * @typedef {object} Snapshot
 * @property {number} version
 * @property {string} cursor
 * @property {Job[]} jobs
 * @property {Network[]} networks
 * @property {number} busDeliveryDropsTotal
 * @property {number} jobDeliveryDropsTotal
 * @property {string} observedAt
 */

export {}
