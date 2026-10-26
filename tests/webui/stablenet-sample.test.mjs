import test from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync, readdirSync} from 'node:fs'
import {decodeJob, decodeJobPage, decodeLogs, decodeMetrics, decodeNetwork, decodeNetworkPage, decodeRun, decodeRunPage, decodeSnapshot} from '../../web/src/api-decode.mjs'
import {parseHistoryJSON} from '../../web/src/history-json.js'

// Responses recorded from a local 4-node stablenet on 2026-10-27, with node4
// stopped from minute 10 to minute 20. docs/dev/web-data-inventory.md explains them.
const dir = new URL('./fixtures/stablenet-sample/', import.meta.url)
const text = name => readFileSync(new URL(name, dir), 'utf8')
const read = name => JSON.parse(text(name + '.json')).body
const node = (body, id) => body.nodes.find(n => n.id === id)

test('the sample carries no machine, account, or runtime path that identifies the recorder', () => {
 for (const name of readdirSync(dir)) {
  const body = text(name)
  assert.doesNotMatch(body, /\/(Users|home|private|var)\//, name)
  for (const [, value] of body.matchAll(/machine-sha256:([0-9a-f]{64})/g)) assert.equal(value, '1'.repeat(64), name)
  for (const [, value] of body.matchAll(/"actorId":"([^"]*)"/g)) assert.ok(['2'.repeat(64), 'unauthenticated'].includes(value), name)
  for (const [, value] of body.matchAll(/"csrfToken":"([^"]*)"/g)) assert.equal(value, '3'.repeat(64), name)
 }
})

test('every recorded response passes the decoder the screens read it with', () => {
 const decoders = [
  [/^networks-/, decodeNetworkPage], [/^observations-/, decodeNetwork], [/^snapshot-/, decodeSnapshot], [/^metrics-/, decodeMetrics],
  [/^logs-/, decodeLogs], [/^jobs\.json$/, decodeJobPage], [/^job-/, decodeJob], [/^history\.json$/, decodeRunPage], [/^history-/, decodeRun],
 ]
 let decoded = 0
 for (const name of readdirSync(dir)) {
  const decode = decoders.find(([pattern]) => pattern.test(name))?.[1]
  if (!decode) continue
  const body = /^history/.test(name) ? parseHistoryJSON(text(name)).body : read(name.slice(0, -5))
  assert.doesNotThrow(() => decode(body), name)
  decoded++
 }
 assert.ok(decoded >= 20, `decoded ${decoded} responses`)
})

test('networks still reports every node as unknown while observations sees node4 stop and come back', () => {
 // Moved to step 3-5: the list needs timed observations from the collector.
 assert.deepEqual(read('networks-node4-stopped').items[0].nodes.map(n => n.state), ['unknown', 'unknown', 'unknown', 'unknown'])
 assert.equal(node(read('observations-all-running'), 'node4').state, 'running')
 const stopped = node(read('observations-node4-stopped'), 'node4')
 assert.deepEqual([stopped.state, stopped.observationReason], ['stopped', 'no_matching_process'])
 const restarted = node(read('observations-node4-restarted'), 'node4')
 assert.equal(restarted.state, 'running')
 assert.notEqual(restarted.observedPid, node(read('observations-all-running'), 'node4').observedPid)
})

test('a node ref is what the logs API answers to', () => {
 const n = node(read('observations-all-running'), 'node1')
 assert.equal(n.ref, `${n.networkId}.node1`)
 assert.equal(read('logs-node1').nodeId, n.ref)
})

test('the ten minutes node4 was stopped is one node_stopped gap, and only its data is missing', () => {
 const metrics = read('metrics-30min')
 const stopped = metrics.coverage.gaps.filter(g => g.code === 'node_stopped')
 assert.equal(stopped.length, 1)
 assert.deepEqual([stopped[0].scope, stopped[0].nodeId, stopped[0].source], ['node', 'node4', 'collector'])
 const span = Date.parse(stopped[0].to) - Date.parse(stopped[0].from)
 assert.ok(span > 9 * 60_000 && span < 12 * 60_000, `node_stopped spans ${span} ms`)
 for (const g of metrics.coverage.gaps) assert.ok(g.scope && g.source && g.code && !g.code.includes(' '), JSON.stringify(g))
})

test('each concept has one canonical series per node, RPC first', () => {
 const series = read('metrics-30min').series
 for (const nodeId of ['node1', 'node2', 'node3', 'node4']) for (const metric of ['block_height', 'peer_count']) {
  const canonical = series.filter(s => s.nodeId === nodeId && s.metric === metric && s.canonical)
  assert.deepEqual(canonical.map(s => s.source), ['rpc'], `${metric} ${nodeId}`)
 }
 assert.ok(series.filter(s => !s.canonical).every(s => s.source === 'metrics'))
})

test('a running phase has no end time, and the job says it has not ended', () => {
 for (const name of ['job-test-run-running', 'job-node-stop-running']) {
  const job = read(name)
  assert.deepEqual([job.state, job.terminal, job.needsCheck], ['running', false, false], name)
  const running = job.phases.filter(p => p.state === 'running')
  assert.equal(running.length, 1, name)
  assert.equal(running[0].finishedAt, undefined, name)
 }
 for (const name of ['job-test-run-failed', 'job-test-run-passed']) assert.equal(read(name).terminal, true, name)
})

test('run details carry their tests with decimal values and an up-to-date missing list', () => {
 const sessions = readdirSync(dir).filter(n => n.startsWith('history-session-')).map(n => read(n.slice(0, -5)))
 assert.equal(sessions.length, 2)
 for (const run of sessions) {
  assert.ok(run.finishedAt && !run.summary.missingDimensions.includes('finishedAt'), run.id)
  const [t] = run.tests
  for (const a of t.assertions) {
   assert.match(a.expected, /^\d+$/)
   assert.match(a.actual, /^\d+$/)
   assert.equal(a.nodeId, 'node1')
  }
 }
 const failed = sessions.find(r => r.state === 'failed').tests[0]
 assert.deepEqual([failed.state, failed.assertions[0].compare, failed.assertions[0].passed], ['failed', 'Less', false])
})

test('observation events still carry only a version, not node state or metrics', () => {
 const events = text('events.txt').split('\n')
  .map(line => line.split('\t')[1] ?? '').filter(line => line.startsWith('data: ')).map(line => JSON.parse(line.slice(6)))
 const observations = events.filter(e => e.type !== 'resync_required')
 assert.ok(observations.length > 60)
 for (const event of observations) assert.deepEqual(Object.keys(event).sort(), ['cursor', 'jobDeliveryDropsTotal', 'observedAt', 'version'])
})
