import test from 'node:test'
import assert from 'node:assert/strict'
import { chartModel, timeTicks, gapBands, latestValues, nearestPoints, pointerTime } from '../../web/src/metric-chart.mjs'

const metrics = {
 series: [
  { nodeId: 'node1', name: 'block_height', unit: 'blocks', source: 'rpc', samples: [{ time: '2026-10-09T00:00:00Z', value: 10 }, { time: '2026-10-09T00:00:10Z', value: 12 }, { time: '2026-10-09T00:00:20Z', value: 14 }] },
  { nodeId: 'node2', name: 'block_height', unit: 'blocks', source: 'rpc', samples: [{ time: '2026-10-09T00:00:00Z', value: 10 }, { time: '2026-10-09T00:00:10Z', value: 11 }, { time: '2026-10-09T00:00:20Z', value: null }] }
 ],
 coverage: { gaps: [{ from: '2026-10-09T00:00:15Z', to: '2026-10-09T00:00:30Z', scope: 'node', nodeId: 'node2', source: 'rpc', code: 'rpc_unavailable' }, { from: '2026-10-08T00:00:00Z', to: '2026-10-08T00:01:00Z', scope: 'network', source: 'collector', code: 'not_collected' }] }
}
const model = chartModel(metrics, 'block_height', { width: 200, height: 100 })

test('the time axis gets evenly spaced ticks at both ends', () => {
 const ticks = timeTicks(model, 3)
 assert.deepEqual(ticks.map(t => t.x), [0, 100, 200])
 assert.deepEqual(ticks.map(t => t.time), ['2026-10-09T00:00:00.000Z', '2026-10-09T00:00:10.000Z', '2026-10-09T00:00:20.000Z'])
 assert.deepEqual(timeTicks({ ...model, tMin: 5, tMax: 5 }, 3).length, 1)
})

test('collection gaps become bands clipped to the chart, gaps outside it are dropped', () => {
 assert.deepEqual(gapBands(model, metrics.coverage).map(b => [b.x, b.width, b.gap.nodeId, b.gap.code]), [[150, 50, 'node2', 'rpc_unavailable']])
})

test('the latest value of each node and the spread between them show a lagging node', () => {
 const latest = latestValues(model)
 assert.deepEqual(latest.values.map(v => [v.node, v.value]), [['node1', 14], ['node2', 11]])
 assert.equal(latest.spread, 3)
 assert.equal(latestValues({ nodes: [] }).spread, null)
})

test('a cursor time picks each node\'s nearest drawn point', () => {
 assert.deepEqual(nearestPoints(model, '2026-10-09T00:00:09Z').map(p => [p.node, p.value]), [['node1', 12], ['node2', 11]])
})

test('a node with no sample near the cursor shows no value rather than an old one', () => {
 // node2 has nothing after 00:00:10; at 00:00:20 with a 5 s tolerance it is missing.
 assert.deepEqual(nearestPoints(model, '2026-10-09T00:00:20Z', 5000).map(p => [p.node, p.value]), [['node1', 14], ['node2', null]])
})

test('a pointer position maps back to a time on the axis', () => {
 assert.equal(pointerTime(model, 100), '2026-10-09T00:00:10.000Z')
 assert.equal(pointerTime(model, -50), '2026-10-09T00:00:00.000Z')
})
