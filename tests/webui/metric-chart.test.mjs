import test from 'node:test'
import assert from 'node:assert/strict'
import {chartModel, logWindow, gapSummary} from '../../web/src/metric-chart.mjs'
import {formatTime} from '../../web/src/labels.mjs'

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
 // Korean first, the engine code kept (tests and searches still find it), local times.
 assert.deepEqual(gapSummary(metrics.coverage), [`node2 · RPC 응답 없음 · rpc_unavailable (${formatTime('2026-10-09T00:00:10Z')} – ${formatTime('2026-10-09T00:00:15Z')})`])
 assert.deepEqual(gapSummary({complete: true, gaps: []}), [])
})

test('a series breaks at a missing value and at a gap longer than the sampling interval', async () => {
 const {chartModel} = await import('../../web/src/metric-chart.mjs')
 const m = {series: [{nodeId: 'node1', name: 'h', unit: '', source: 'rpc', samples: [
  {time: '2026-10-09T00:00:00Z', value: 1}, {time: '2026-10-09T00:00:05Z', value: 2},
  {time: '2026-10-09T00:00:10Z', value: null}, {time: '2026-10-09T00:00:15Z', value: 3},
  {time: '2026-10-09T00:00:20Z', value: 4}, {time: '2026-10-09T00:01:20Z', value: 5}]}],
  coverage: {sampleIntervalSeconds: 5}}
 const [series] = chartModel(m, 'h', {width: 100, height: 50}).nodes
 assert.equal(series.segments.length, 3, 'one line per stretch without gaps')
 assert.deepEqual(series.segments.map(s => s.length), [2, 2, 1])
 assert.equal((series.path.match(/M/g) ?? []).length, 3)
})

test('the metrics to chart are the ones the server collected, with their unit and source', async () => {
 const {metricNames} = await import('../../web/src/metric-chart.mjs')
 assert.deepEqual(metricNames(metrics), [{name: 'block_height', unit: 'blocks', source: 'rpc'}, {name: 'p2p_peers', unit: 'peers', source: 'metrics'}])
})

test('a preset range ends now and a custom range is used as given', async () => {
 const {timeRange} = await import('../../web/src/metric-chart.mjs')
 const now = Date.parse('2026-10-09T01:00:00Z')
 assert.deepEqual(timeRange({preset: 15}, now), {from: '2026-10-09T00:45:00.000Z', to: '2026-10-09T01:00:00.000Z'})
 assert.deepEqual(timeRange({preset: 0, from: '2026-10-09T00:00:00Z', to: '2026-10-09T00:30:00Z'}, now), {from: '2026-10-09T00:00:00.000Z', to: '2026-10-09T00:30:00.000Z'})
 assert.equal(timeRange({preset: 0, from: '2026-10-09T00:30:00Z', to: '2026-10-09T00:00:00Z'}, now), null, 'a reversed range is refused')
})
