import test from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync, readdirSync} from 'node:fs'

// A reference conversion of the recorded stablenet sample into the model in
// docs/dev/web-domain-model.md. It shows the model can carry real data; once
// the server answers in the model's shape (plan step 3-4) this file goes away.
const dir = new URL('./fixtures/stablenet-sample/', import.meta.url)
const read = name => JSON.parse(readFileSync(new URL(name + '.json', dir), 'utf8')).body

const EXECUTION = ['pending', 'running', 'cancelling', 'succeeded', 'failed', 'blocked', 'skipped', 'cancelled', 'interrupted', 'unknown']
const TERMINAL = ['succeeded', 'failed', 'blocked', 'skipped', 'cancelled', 'interrupted']
const PROCESS = ['running', 'stopped', 'conflict', 'unknown']
const ZERO_TIME = '0001-01-01T00:00:00Z'

const jobState = s => s === 'accepted' ? 'pending' : s
const testState = s => ({pass: 'succeeded', fail: 'failed', blocked: 'blocked', skip: 'skipped'})[s] ?? 'unknown'
const processState = (s, reason) => ({
 running: ['running'], unrecorded_running: ['running', 'unrecorded_pid_found'], stopped: ['stopped', 'no_matching_process'],
 missing: ['stopped', 'recorded_pid_absent'], ownership_mismatch: ['conflict', reason], unknown: ['unknown', reason ?? 'not_observed'],
})[s] ?? ['unknown', 'not_observed']
const decimal = v => v === undefined || v === null || v === '' ? undefined : String(v)
const time = t => t && t !== ZERO_TIME ? t : undefined
const compact = o => Object.fromEntries(Object.entries(o).filter(([, v]) => v !== undefined))

const CONCEPT = {block_height: ['block_height', true], chain_head_block: ['block_height', false], peer_count: ['peer_count', true], p2p_peers: ['peer_count', false], txpool_pending: ['txpool_pending', true], txpool_queued: ['txpool_queued', true]}

function toNode(n) {
 const [state, reason] = processState(n.state, n.observationReason)
 const allowed = new Set(n.supportedControls ?? [])
 const names = new Set([...allowed, ...Object.keys(n.controlLimits ?? {})])
 return {
  id: n.id, ref: `${n.networkId}.${n.id}`, networkId: n.networkId, role: n.role,
  observation: compact({state, reason, pid: n.observedPid || undefined, observedAt: n.observedAt}),
  controls: [...names].map(name => compact({name, allowed: allowed.has(name), reason: allowed.has(name) ? undefined : 'limited'})),
 }
}

// The current gap string is "<node> <source> <reason...>"; the network-wide form drops the node.
function toGap(g) {
 const words = g.reason.split(' ')
 const scoped = /^node/.test(words[0])
 const [source, ...rest] = scoped ? words.slice(1) : words
 const gap = {from: g.from, to: g.to, scope: scoped ? 'node' : 'network', nodeId: scoped ? words[0] : undefined, source}
 if (rest.at(-1) === 'not_exposed') return compact({...gap, code: 'not_exposed', metric: rest[0]})
 if (rest[0] === 'deleted_by_admin') return compact({...gap, code: 'deleted_by_admin', runId: rest[1]})
 return compact({...gap, code: rest.join('_')})
}

// The server skips a stopped node without a gap; the model records one. Here it
// is inferred from sample spacing, which the server will not need to do.
function stoppedGaps(series, interval) {
 const gaps = []
 for (const s of series.filter(s => s.canonical && s.metric === 'block_height')) {
  for (let i = 1; i < s.samples.length; i++) {
   const [a, b] = [s.samples[i - 1].time, s.samples[i].time]
   if (Date.parse(b) - Date.parse(a) > interval * 4000) gaps.push({from: a, to: b, scope: 'node', nodeId: s.nodeId, source: 'collector', code: 'node_stopped'})
  }
 }
 return gaps
}

function toMetrics(body) {
 const series = body.series.map(s => ({metric: CONCEPT[s.name][0], source: s.source, canonical: CONCEPT[s.name][1], nodeId: s.nodeId, unit: s.unit, samples: s.samples}))
 return {series, gaps: [...body.coverage.gaps.map(toGap), ...stoppedGaps(series, body.coverage.sampleIntervalSeconds)]}
}

function toPhase(p, jobState) {
 const state = p.state === 'running' && jobState === 'interrupted' ? 'interrupted' : p.state
 return compact({name: p.name, state, startedAt: p.startedAt, finishedAt: time(p.finishedAt), message: p.message})
}

function toJob(j) {
 const state = jobState(j.state)
 return compact({
  id: j.id, operation: j.operation, state, terminal: TERMINAL.includes(state), workspaceId: j.workspaceId, planId: j.planId, actorId: j.actorId,
  createdAt: j.createdAt, phases: j.phases.map(p => toPhase(p, state)), retention: j.retention, nodeDisposition: j.nodeDisposition,
  error: j.error, runIds: j.runIds, partialEffects: j.partialEffects, unresolvedResources: j.unresolvedResources, credentialRefs: j.credentialRefs ?? [],
 })
}

function toTest(t, results) {
 const dir = `tests/${String(t.seq).padStart(3, '0')}_${t.id}/`
 const file = name => results[dir + name]
 const status = file('status.json') ?? {}
 return compact({
  seq: t.seq, id: t.id, state: testState(t.status), reason: status.reason, startedAt: t.startedAt, finishedAt: t.finishedAt, env: t.env,
  assertions: (file('assert.json') ?? []).map(a => compact({assert: a.Assert, nodeId: a.On || a.Provenance?.on, compare: a.Provenance?.compare, expected: decimal(a.Expected) ?? '', actual: decimal(a.Actual), passed: a.Pass, detail: a.Source || undefined, provenance: a.Provenance})),
  steps: (file('steps.json') ?? []).map(s => compact({index: s.Index, type: s.Type, nodeId: s.On || undefined, signer: s.Signer || undefined, hash: s.Hash || undefined, nonce: s.Nonce ? decimal(s.Nonce) : undefined, gas: decimal(s.Gas), error: s.Error || undefined, deployment: s.Deployment})),
  postActions: (file('postaction.json') ?? []).map(p => compact({name: p.Name, ok: p.OK, detail: p.Detail || undefined})),
  artifacts: file('artifacts.json')?.refs ?? [],
 })
}

function toRun(r) {
 const s = r.summary
 const state = s.kind === 'job' ? jobState(r.state) : r.state
 const missing = (s.missingDimensions ?? []).filter(d => !(d === 'finishedAt' && r.finishedAt))
 const counts = s.counts && {succeeded: s.counts.pass, failed: s.counts.fail, blocked: s.counts.blocked, skipped: s.counts.skip}
 return compact({
  id: r.id, kind: s.kind, jobId: r.jobId, workspaceId: r.workspaceId || undefined, chain: r.chain || undefined, actorId: r.actorId,
  state, startedAt: r.startedAt, finishedAt: r.finishedAt, missing, counts, tests: (s.tests ?? []).map(t => toTest(t, s.results ?? {})),
  observations: s.observations, captureGaps: s.captureGaps ?? [], fingerprints: r.fingerprints ?? {},
 })
}

const runs = readdirSync(dir).filter(n => n.startsWith('history-') && n !== 'history.json').map(n => toRun(read(n.slice(0, -5))))
const jobs = [...read('jobs').items, ...['job-test-run-failed', 'job-test-run-passed', 'job-node-stop', 'job-node-start'].map(read)].map(toJob)

test('every recorded node converts with one process state from the model vocabulary', () => {
 for (const name of ['observations-all-running', 'observations-node4-stopped', 'observations-node4-restarted']) {
  for (const node of read(name).nodes.map(toNode)) {
   assert.ok(PROCESS.includes(node.observation.state), `${name} ${node.id} ${node.observation.state}`)
   assert.equal(node.ref, `${node.networkId}.${node.id}`)
   if (node.observation.state !== 'running') assert.ok(node.observation.reason, `${name} ${node.id} has no reason`)
  }
 }
 assert.deepEqual(toNode(read('observations-node4-stopped').nodes[3]).observation.state, 'stopped')
})

test('the logs API accepts the node ref the model hands out', () => {
 const node = toNode(read('observations-all-running').nodes[0])
 assert.equal(read('logs-node1').nodeId, node.ref)
})

test('each concept has exactly one canonical metric series per node', () => {
 const {series} = toMetrics(read('metrics-30min'))
 const canonical = new Map()
 for (const s of series.filter(s => s.canonical)) {
  const key = `${s.metric} ${s.nodeId}`
  assert.ok(!canonical.has(key), `two canonical series for ${key}`)
  canonical.set(key, s)
 }
 for (const node of ['node1', 'node2', 'node3', 'node4']) for (const metric of ['block_height', 'peer_count']) assert.ok(canonical.has(`${metric} ${node}`), `${metric} ${node}`)
 for (const s of series.filter(s => !s.canonical)) assert.equal(s.source, 'metrics')
})

test('gaps come out structured, and the stopped node gets a gap with a reason', () => {
 const {gaps} = toMetrics(read('metrics-30min'))
 for (const g of gaps) {
  assert.ok(['network', 'node'].includes(g.scope))
  assert.ok(g.source && g.code && !g.code.includes(' '), JSON.stringify(g))
 }
 const stopped = gaps.filter(g => g.code === 'node_stopped')
 assert.equal(stopped.length, 1)
 assert.equal(stopped[0].nodeId, 'node4')
 assert.ok(Date.parse(stopped[0].to) - Date.parse(stopped[0].from) > 600_000)
 assert.deepEqual(toGap({from: 'a', to: 'b', reason: 'node1 metrics txpool_pending not_exposed'}), {from: 'a', to: 'b', scope: 'node', nodeId: 'node1', source: 'metrics', code: 'not_exposed', metric: 'txpool_pending'})
 assert.deepEqual(toGap({from: 'a', to: 'b', reason: 'archive deleted_by_admin r123'}), {from: 'a', to: 'b', scope: 'network', source: 'archive', code: 'deleted_by_admin', runId: 'r123'})
})

test('jobs and phases use the execution vocabulary and never carry a zero time', () => {
 assert.ok(jobs.length >= 4)
 for (const job of jobs) {
  assert.ok(EXECUTION.includes(job.state), job.state)
  assert.equal(job.terminal, TERMINAL.includes(job.state))
  for (const phase of job.phases) {
   assert.ok(EXECUTION.includes(phase.state), phase.state)
   assert.ok(phase.startedAt)
   assert.notEqual(phase.finishedAt, ZERO_TIME)
  }
 }
 const running = toPhase({name: 'deploy', state: 'running', startedAt: '2026-10-25T08:00:00Z', finishedAt: ZERO_TIME}, 'running')
 assert.equal(running.finishedAt, undefined)
 assert.equal(toPhase({name: 'deploy', state: 'running', startedAt: 'x'}, 'interrupted').state, 'interrupted')
})

test('run records carry tests, assertions and steps without engine file names or PascalCase', () => {
 assert.ok(runs.length >= 6)
 const sessions = runs.filter(r => r.kind === 'engine-session')
 assert.equal(sessions.length, 2)
 for (const run of runs) {
  assert.ok(EXECUTION.includes(run.state), `${run.id} ${run.state}`)
  if (run.finishedAt) assert.ok(!run.missing.includes('finishedAt'), run.id)
 }
 for (const run of sessions) {
  assert.equal(run.tests.length, 1)
  const [t] = run.tests
  assert.ok(EXECUTION.includes(t.state))
  assert.ok(t.assertions.length > 0 && t.steps.length > 0)
  for (const a of t.assertions) {
   assert.equal(typeof a.expected, 'string')
   assert.equal(typeof a.actual, 'string')
   assert.match(a.expected, /^\d+$/)
   assert.equal(a.nodeId, 'node1')
  }
  assert.doesNotMatch(JSON.stringify(t), /"(Expected|Actual|Index|On|Pass)":/)
 }
 const failed = sessions.find(r => r.state === 'failed').tests[0]
 assert.equal(failed.state, 'failed')
 assert.equal(failed.assertions[0].passed, false)
 assert.equal(failed.assertions[0].expected, '0')
})

test('fields the model adds are not in today\'s responses yet', () => {
 const node = read('networks-node4-stopped').items[0].nodes[0]
 for (const field of ['ref', 'server', 'ports', 'syncMode']) assert.equal(node[field], undefined, field)
 assert.equal(read('networks-node4-stopped').items[0].chain, undefined)
 const events = readFileSync(new URL('events.txt', dir), 'utf8')
 for (const type of ['node', 'metrics', 'gap', 'logs']) assert.doesNotMatch(events, new RegExp(`event: ${type}\\n`), type)
})
