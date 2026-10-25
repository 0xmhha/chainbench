import test from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync, readdirSync} from 'node:fs'

// Responses recorded from a local 4-node stablenet on 2026-10-25, with node4
// stopped from minute 10 to minute 20. docs/dev/web-data-inventory.md explains them.
const dir = new URL('./fixtures/stablenet-sample/', import.meta.url)
const read = name => JSON.parse(readFileSync(new URL(name + '.json', dir), 'utf8')).body
const node = (body, id) => body.nodes.find(n => n.id === id)

test('the sample carries no machine, account, or runtime path that identifies the recorder', () => {
 for (const name of readdirSync(dir)) {
  const text = readFileSync(new URL(name, dir), 'utf8')
  assert.doesNotMatch(text, /\/(Users|home|private|var)\//, name)
  for (const [, value] of text.matchAll(/machine-sha256:([0-9a-f]{64})/g)) assert.equal(value, '1'.repeat(64), name)
  for (const [, value] of text.matchAll(/"actorId":"([^"]*)"/g)) assert.ok(['2'.repeat(64), 'unauthenticated'].includes(value), name)
  for (const [, value] of text.matchAll(/"csrfToken":"([^"]*)"/g)) assert.equal(value, '3'.repeat(64), name)
 }
})

test('networks reports every node as unknown while observations sees node4 stop and come back', () => {
 assert.deepEqual(read('networks-node4-stopped').items[0].nodes.map(n => n.state), ['unknown', 'unknown', 'unknown', 'unknown'])
 assert.equal(node(read('observations-all-running'), 'node4').state, 'running')
 const stopped = node(read('observations-node4-stopped'), 'node4')
 assert.equal(stopped.state, 'stopped')
 assert.equal(stopped.observationReason, 'no_matching_process')
 const restarted = node(read('observations-node4-restarted'), 'node4')
 assert.equal(restarted.state, 'running')
 assert.notEqual(restarted.observedPid, node(read('observations-all-running'), 'node4').observedPid)
})

test('the metric hole while node4 was stopped has no gap reason', () => {
 const metrics = read('metrics-30min')
 assert.deepEqual(metrics.coverage.gaps.map(g => g.reason), ['collector not_collected'])
 const times = metrics.series.find(s => s.name === 'block_height' && s.nodeId === 'node4').samples.map(s => Date.parse(s.time))
 const longest = Math.max(...times.slice(1).map((t, i) => t - times[i]))
 assert.ok(longest > 600_000, `longest node4 interval ${longest} ms`)
 const node1 = metrics.series.find(s => s.name === 'block_height' && s.nodeId === 'node1').samples
 assert.ok(node1.length > times.length)
})

test('the same metric arrives under two names', () => {
 const names = new Set(read('metrics-30min').series.map(s => s.name))
 for (const pair of [['block_height', 'chain_head_block'], ['peer_count', 'p2p_peers']]) for (const name of pair) assert.ok(names.has(name), name)
})

test('observation events carry only a version, not node state or metrics', () => {
 const events = readFileSync(new URL('events.txt', dir), 'utf8').split('\n')
  .map(line => line.split('\t')[1] ?? '').filter(line => line.startsWith('data: ')).map(line => JSON.parse(line.slice(6)))
 const observations = events.filter(e => e.type !== 'resync_required')
 assert.ok(observations.length > 60)
 for (const event of observations) assert.deepEqual(Object.keys(event).sort(), ['cursor', 'jobDeliveryDropsTotal', 'observedAt', 'version'])
})
