import test from 'node:test'
import assert from 'node:assert/strict'
import {chartModel, logWindow, gapSummary} from '../../web/src/metric-chart.mjs'

const metrics = {
 networkId: 'n',
 series: [
  {nodeId: 'node2', name: 'block_height', unit: 'blocks', source: 'rpc', samples: [{time: '2026-10-09T00:00:00Z', value: 4}, {time: '2026-10-09T00:00:10Z', value: 6}]},
  {nodeId: 'node1', name: 'block_height', unit: 'blocks', source: 'rpc', samples: [{time: '2026-10-09T00:00:05Z', value: 5}, {time: '2026-10-09T00:00:15Z', value: null}]},
  {nodeId: 'node1', name: 'p2p_peers', unit: 'peers', source: 'metrics', samples: [{time: '2026-10-09T00:00:05Z', value: 3}]},
 ],
 coverage: {complete: false, gaps: [{from: '2026-10-09T00:00:10Z', to: '2026-10-09T00:00:15Z', reason: 'node2 rpc rpc_unavailable'}], downsampled: false},
}

test('a chart keeps each node separate, sorted, and scaled to the shared time and value range', () => {
 const model = chartModel(metrics, 'block_height', {width: 100, height: 50})
 assert.deepEqual(model.nodes.map(n => n.node), ['node1', 'node2'])
 assert.equal(model.unit, 'blocks'); assert.equal(model.source, 'rpc')
 assert.equal(model.yMin, 4); assert.equal(model.yMax, 6)
 const [node1, node2] = model.nodes
 assert.equal(node2.points[0].x, 0); assert.equal(node2.points[0].y, 50)
 assert.ok(Math.abs(node2.points[1].x - 200 / 3) < 1e-9); assert.equal(node2.points[1].y, 0)
 assert.equal(node1.points.length, 1, 'a missing value is not drawn as zero')
 assert.equal(node1.points[0].time, '2026-10-09T00:00:05Z')
 assert.match(node2.path, /^M0,50 L66\.\d+,0$/)
})

test('an absent metric yields no invented series', () => {
 const model = chartModel(metrics, 'txpool_pending', {width: 100, height: 50})
 assert.equal(model.nodes.length, 0)
})

test('a single value draws a flat line inside the chart', () => {
 const model = chartModel(metrics, 'p2p_peers', {width: 100, height: 50})
 assert.equal(model.nodes[0].points[0].y, 25)
})

test('a selected chart time links logs from 30 seconds before to 30 seconds after', () => {
 assert.deepEqual(logWindow('2026-10-09T00:01:00Z', 30), {from: '2026-10-09T00:00:30.000Z', to: '2026-10-09T00:01:30.000Z'})
})

test('coverage gaps are listed with their reasons rather than hidden', () => {
 assert.deepEqual(gapSummary(metrics.coverage), ['node2 rpc rpc_unavailable (2026-10-09T00:00:10Z – 2026-10-09T00:00:15Z)'])
 assert.deepEqual(gapSummary({complete: true, gaps: []}), [])
})
