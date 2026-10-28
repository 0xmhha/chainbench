import test from 'node:test'
import assert from 'node:assert/strict'
import { cursorDefaults, nextEvent, measureBetween, readingRows, stripX, stripTime } from '../../web/src/recorder.mjs'

const t = s => new Date(Date.parse('2026-10-23T07:00:00Z') + s * 1000).toISOString()

test('cursor A sits on the first warning or error and B on the first error, else the test span', () => {
 const events = [{ time: t(15), level: 'WARN' }, { time: t(22), level: 'ERROR' }, { time: t(25), level: 'ERROR' }]
 assert.deepEqual(cursorDefaults({ events, start: t(6), end: t(24) }), { a: t(15), b: t(22) })
 assert.deepEqual(cursorDefaults({ events: [{ time: t(15), level: 'WARN' }], start: t(6), end: t(24) }), { a: t(15), b: t(24) })
 assert.deepEqual(cursorDefaults({ events: [], start: t(6), end: t(24) }), { a: t(6), b: t(24) })
 assert.deepEqual(cursorDefaults({ events: [{ time: t(10), level: 'ERROR' }], start: t(6), end: t(24) }), { a: t(6), b: t(10) })
})

const series = (name, nodeId, values) => ({ name, metric: name, canonical: true, nodeId, unit: 'u', source: 'rpc', samples: values.map((value, i) => ({ time: t(i * 5), value })) })
const metrics = {
 coverage: { sampleIntervalSeconds: 5 },
 series: [
  series('block_height', 'node1', [40, 43, 45, 48, 50, 51]),
  series('block_height', 'node2', [40, 43, 45, 46, null, 46]),
  series('block_height', 'node3', [40, 43, 45, 47, 50, 51]),
  series('peer_count', 'node1', [3, 3, 3, 3, 3, 3]),
  series('peer_count', 'node2', [3, 3, 3, 1, null, 1]),
  series('peer_count', 'node3', [3, 3, 3, 3, 3, 3]),
  series('txpool_pending', 'node1', [2, 2, 2, 2, 2, 2]),
  series('txpool_pending', 'node2', [2, 2, 2, 18, null, 20]),
 ]
}

test('a cursor moment from the URL puts B there while it falls inside the window', () => {
 const events = [{ time: t(15), level: 'WARN' }, { time: t(22), level: 'ERROR' }]
 assert.deepEqual(cursorDefaults({ events, start: t(6), end: t(24), at: t(18), from: t(1), to: t(29) }), { a: t(15), b: t(18) })
 // Outside the window (another test was chosen since) the defaults hold.
 assert.deepEqual(cursorDefaults({ events, start: t(6), end: t(24), at: t(40), from: t(1), to: t(29) }), { a: t(15), b: t(22) })
})

test('the measure reads each gauge at both cursors, and a gap reads as no value', () => {
 const m = measureBetween(metrics, 'node2', t(10), t(25))
 assert.deepEqual(m.gauges.map(g => [g.label, g.a, g.b, g.changed]), [['블록 높이', 45, 46, true], ['피어 수', 3, 1, true], ['대기 트랜잭션', 2, 20, true]])
 assert.deepEqual(m.others, { a: 45, b: 51 })
 assert.equal(m.seconds, 15)
 assert.equal(measureBetween(metrics, 'node2', t(20), t(20)).gauges[0].a, null)
 // Red marks what is wrong, not what moved: node2's block stalls behind the
 // others, its peers drop below theirs and its queue grows past theirs.
 assert.deepEqual(m.gauges.map(g => g.abnormal), [true, true, true])
 const early = measureBetween(metrics, 'node2', t(0), t(10))
 assert.deepEqual(early.gauges.map(g => [g.changed, g.abnormal]), [[true, false], [false, false], [false, false]])
})

test('readings ring a node that lags, loses peers or queues more than every other node', () => {
 const rows = readingRows(metrics, 'node2')
 assert.equal(rows.length, 6)
 assert.deepEqual(rows[2].cells.map(c => [c.value, c.abnormal]), [[45, false], [3, false], [2, false]])
 assert.deepEqual(rows[3].cells.map(c => [c.value, c.abnormal]), [[46, true], [1, true], [18, true]])
 assert.deepEqual(rows[4].cells.map(c => c.value), [null, null, null])
})

test('strip positions map time to a share of the window and back', () => {
 assert.equal(stripX(t(10), t(0), t(40)), 25)
 assert.equal(stripX(t(50), t(0), t(40)), null)
 assert.equal(stripTime(25, t(0), t(40)), t(10))
})


test('a trace sits on the strip window, breaks at a recorded gap and spans its own range', async () => {
 const { tracePath } = await import('../../web/src/recorder.mjs')
 const samples = [{ time: t(0), value: 40 }, { time: t(10), value: 50 }, { time: t(20), value: null }, { time: t(30), value: 60 }]
 const trace = tracePath(samples, t(0), t(40), 10)
 assert.deepEqual([trace.min, trace.max], [40, 60])
 assert.deepEqual(trace.segments, ['M0,100 L25,50', 'M75,0'])
 assert.deepEqual(tracePath([], t(0), t(40), 10).segments, [])
 const flat = tracePath([{ time: t(0), value: 3 }, { time: t(40), value: 3 }], t(0), t(40), 40)
 assert.deepEqual(flat.segments, ['M0,50 L100,50'])
})

test('the next-error key moves to the error after the cursor, and back before it', () => {
 const errors = [{ time: t(10) }, { time: t(20) }, { time: t(30) }]
 assert.equal(nextEvent(errors, t(20), 1), t(30))
 assert.equal(nextEvent(errors, t(20), -1), t(10))
 assert.equal(nextEvent(errors, t(15), 1), t(20))
 // Past the last (or before the first) error the cursor stays.
 assert.equal(nextEvent(errors, t(30), 1), null)
 assert.equal(nextEvent(errors, t(10), -1), null)
 assert.equal(nextEvent([], t(10), 1), null)
})
