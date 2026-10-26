// @ts-check
// Decoders check a Web response against api-model.js before a screen reads
// it. A response that lacks a required field or carries the wrong JSON type
// becomes a FormatError naming where it went wrong, which the screen shows as
// a data format error instead of failing somewhere inside the drawing code.
// Fields the model does not list pass through untouched, so a server that
// adds a field does not break an older screen.

/** @typedef {import('./api-model.js').Network} Network */
/** @typedef {import('./api-model.js').Metrics} Metrics */
/** @typedef {import('./api-model.js').Logs} Logs */
/** @typedef {import('./api-model.js').Job} Job */
/** @typedef {import('./api-model.js').Run} Run */
/** @typedef {import('./api-model.js').Snapshot} Snapshot */

// Korean names of the JSON kinds a checker expects or finds.
const KINDS = { string: '문자열', number: '숫자', boolean: '참·거짓', object: '객체', 배열: '배열', 객체: '객체' }

/** @param {string} word whether the last syllable has a final consonant */
const closed = word => {
 const code = word.charCodeAt(word.length - 1) - 0xac00
 return code >= 0 && code <= 11171 && code % 28 !== 0
}

export class FormatError extends Error {
 /** @param {string} path @param {string} expected @param {unknown} value */
 constructor(path, expected, value) {
  const want = KINDS[/** @type {keyof typeof KINDS} */ (expected)] ?? expected
  const got = value === undefined ? '값이 없습니다' : `${describe(value)}입니다`
  super(`데이터 형식 오류: ${path || '응답'} 값은 ${want}${closed(want) ? '이어야' : '여야'} 하는데 ${got}`)
  this.name = 'FormatError'
  this.kind = 'format'
  this.path = path
 }
}

/** @param {unknown} value */
const describe = value => value === null ? 'null' : Array.isArray(value) ? '배열' : KINDS[/** @type {keyof typeof KINDS} */ (typeof value)] ?? typeof value

/**
 * A checker validates one value at a path and throws FormatError otherwise.
 * @typedef {(value: unknown, path: string) => void} Checker
 */

/** @param {string} type @returns {Checker} */
const primitive = type => (value, path) => {
 if (typeof value !== type || (type === 'number' && !Number.isFinite(value))) throw new FormatError(path, type, value)
}
const string = primitive('string')
const number = primitive('number')
const boolean = primitive('boolean')
/** @type {Checker} */
const any = () => {}

/** @param {Checker} item @returns {Checker} */
const arrayOf = item => (value, path) => {
 if (!Array.isArray(value)) throw new FormatError(path, '배열', value)
 value.forEach((v, i) => item(v, `${path}[${i}]`))
}

/** @param {Checker} item @returns {Checker} */
const mapOf = item => (value, path) => {
 if (!value || typeof value !== 'object' || Array.isArray(value)) throw new FormatError(path, '객체', value)
 for (const [k, v] of Object.entries(value)) item(v, `${path}.${k}`)
}

/** @param {Checker} inner @returns {Checker} */
const nullable = inner => (value, path) => { if (value !== null) inner(value, path) }

/**
 * shape checks the listed fields of an object; a key ending in "?" may be
 * absent.
 * @param {Record<string, Checker>} fields
 * @returns {Checker}
 */
const shape = fields => (value, path) => {
 if (!value || typeof value !== 'object' || Array.isArray(value)) throw new FormatError(path, '객체', value)
 const record = /** @type {Record<string, unknown>} */ (value)
 for (const [key, check] of Object.entries(fields)) {
  const optional = key.endsWith('?')
  const name = optional ? key.slice(0, -1) : key
  if (optional && record[name] === undefined) continue
  check(record[name], path ? `${path}.${name}` : name)
 }
}

const gap = shape({ from: string, to: string, scope: string, 'nodeId?': string, source: string, code: string, 'metric?': string, 'runId?': string })
const coverage = shape({ from: string, to: string, sampleIntervalSeconds: number, complete: boolean, downsampled: boolean, 'bucketSeconds?': number, 'timestampSource?': string, gaps: arrayOf(gap) })
const node = shape({ id: string, ref: string, networkId: string, role: string, state: string, 'observationReason?': string, pid: number, 'observedPid?': number, observedAt: string, supportedControls: arrayOf(string), controlLimits: mapOf(string) })
const network = shape({ id: string, workspaceId: string, ownership: string, version: number, nodes: arrayOf(node) })
const series = shape({ nodeId: string, name: string, metric: string, canonical: boolean, source: string, 'unit?': string, samples: arrayOf(shape({ time: string, value: nullable(number) })) })
const metrics = shape({ networkId: string, series: arrayOf(series), coverage })
const logs = shape({ nodeId: string, entries: arrayOf(shape({ time: string, text: string, 'level?': string })), total: number, coverage, nextCursor: nullable(string) })
const phase = shape({ name: string, state: string, startedAt: string, 'finishedAt?': string, 'message?': string })
const job = shape({
 id: string, actorId: string, workspaceId: string, planId: string, operation: string, state: string, terminal: boolean, needsCheck: boolean,
 phases: arrayOf(phase), createdAt: string, retention: string, nodeDisposition: string, 'cancelReason?': string, 'cancelMessage?': string,
 runIds: arrayOf(string), partialEffects: arrayOf(string), unresolvedResources: arrayOf(string),
 'error?': shape({ code: string, message: string, 'requestId?': string }), 'reconciliation?': any,
})
const assertion = shape({ assert: string, 'nodeId?': string, 'compare?': string, expected: string, 'actual?': string, passed: boolean, 'detail?': string, 'provenance?': mapOf(any) })
const step = shape({ index: number, type: string, 'nodeId?': string, 'signer?': string, 'hash?': string, 'nonce?': string, 'gas?': string, 'error?': string })
const runTest = shape({
 seq: number, id: string, state: string, 'reason?': string, 'startedAt?': string, 'finishedAt?': string, 'env?': string,
 assertions: arrayOf(assertion), steps: arrayOf(step), postActions: arrayOf(shape({ name: string, ok: boolean, 'detail?': string })),
 artifacts: arrayOf(shape({ kind: string, ref: string, 'node?': string })),
})
const run = shape({
 id: string, 'jobId?': string, workspaceId: string, chain: string, 'actorId?': string, state: string, 'startedAt?': string, 'finishedAt?': string,
 sessionRefs: arrayOf(string), fingerprints: mapOf(string), summary: mapOf(any), artifactRefs: arrayOf(string), 'coverage?': coverage, 'tests?': arrayOf(runTest),
})
const snapshot = shape({ version: number, cursor: string, jobs: arrayOf(job), networks: arrayOf(network), busDeliveryDropsTotal: number, jobDeliveryDropsTotal: number, observedAt: string })

/**
 * @template T
 * @param {Checker} check
 * @returns {(value: unknown) => T}
 */
const decoder = check => value => {
 check(value, '')
 return /** @type {T} */ (value)
}

/** @type {(value: unknown) => Network} */
export const decodeNetwork = decoder(network)
/** @type {(value: unknown) => {items: Network[], nextCursor: string|null}} */
export const decodeNetworkPage = decoder(shape({ items: arrayOf(network), nextCursor: nullable(string) }))
/** @type {(value: unknown) => Metrics} */
export const decodeMetrics = decoder(metrics)
/** @type {(value: unknown) => Logs} */
export const decodeLogs = decoder(logs)
/** @type {(value: unknown) => Job} */
export const decodeJob = decoder(job)
/** @type {(value: unknown) => {items: Job[], nextCursor: string|null}} */
export const decodeJobPage = decoder(shape({ items: arrayOf(job), nextCursor: nullable(string) }))
/** @type {(value: unknown) => Run} */
export const decodeRun = decoder(run)
/** @type {(value: unknown) => {items: Run[], nextCursor: string|null}} */
export const decodeRunPage = decoder(shape({ items: arrayOf(run), nextCursor: nullable(string) }))
/** @type {(value: unknown) => Snapshot} */
export const decodeSnapshot = decoder(snapshot)
