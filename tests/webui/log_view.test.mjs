import test from 'node:test'
import assert from 'node:assert/strict'
import { highlightSegments, mergeByTime, logQuery, visibleRange, nearestIndex, levelTone, LOG_LEVELS } from '../../web/src/log-view.mjs'

test('a match is cut out of the line case-insensitively, keeping the original text', () => {
 assert.deepEqual(highlightSegments('Peer dropped reason=TimeOut, timeout again', 'timeout'), [
  { text: 'Peer dropped reason=', match: false },
  { text: 'TimeOut', match: true },
  { text: ', ', match: false },
  { text: 'timeout', match: true },
  { text: ' again', match: false }
 ])
 assert.deepEqual(highlightSegments('no hit', 'xyz'), [{ text: 'no hit', match: false }])
 assert.deepEqual(highlightSegments('anything', ''), [{ text: 'anything', match: false }])
 // Segments always rejoin into the exact line: a test compares it with the node's file.
 const line = 'a<b>&"c" a<b>'
 assert.equal(highlightSegments(line, 'a<b>').map(s => s.text).join(''), line)
})

test('several nodes interleave by time and keep their node, ties in node order', () => {
 const merged = mergeByTime({
  node2: [{ time: '2026-10-09T00:00:01Z', text: 'b1' }, { time: '2026-10-09T00:00:03Z', text: 'b3' }],
  node1: [{ time: '2026-10-09T00:00:01Z', text: 'a1' }, { time: '2026-10-09T00:00:02Z', text: 'a2' }]
 }, ['node1', 'node2'])
 assert.deepEqual(merged.map(e => `${e.node}:${e.text}`), ['node1:a1', 'node2:b1', 'node1:a2', 'node2:b3'])
})

test('the query names only what is set', () => {
 assert.equal(logQuery({ limit: 100 }), 'limit=100')
 assert.equal(logQuery({ from: 'A', to: 'B', search: ' peer ', levels: ['ERROR', 'WARN'], limit: 100 }), 'from=A&to=B&q=peer&level=ERROR%2CWARN&limit=100')
 // A cursor already carries its window; sending the window again is allowed but not needed.
 assert.equal(logQuery({ cursor: '5-1-2-ab', search: 'peer', limit: 100 }), 'q=peer&cursor=5-1-2-ab&limit=100')
 // A run names its own window on the server.
 assert.equal(logQuery({ runId: 'run-1', levels: ['WARN'], limit: 100 }), 'runId=run-1&level=WARN&limit=100')
})

test('only the rows in view (plus a margin) are rendered', () => {
 assert.deepEqual(visibleRange({ scrollTop: 0, height: 220, rowHeight: 22, count: 1000, overscan: 5 }), { start: 0, end: 15 })
 assert.deepEqual(visibleRange({ scrollTop: 2200, height: 220, rowHeight: 22, count: 1000, overscan: 5 }), { start: 95, end: 115 })
 assert.deepEqual(visibleRange({ scrollTop: 99999, height: 220, rowHeight: 22, count: 10, overscan: 5 }), { start: 0, end: 10 })
})

test('the line nearest a time is found, or -1 for nothing', () => {
 const entries = [{ time: '2026-10-09T00:00:01Z' }, { time: '2026-10-09T00:00:05Z' }, { time: '2026-10-09T00:00:09Z' }]
 assert.equal(nearestIndex(entries, '2026-10-09T00:00:06Z'), 1)
 assert.equal(nearestIndex(entries, '2026-10-09T00:00:08Z'), 2)
 assert.equal(nearestIndex([], '2026-10-09T00:00:08Z'), -1)
})

test('levels map to a tone, and every server level has one', () => {
 assert.deepEqual(LOG_LEVELS, ['CRIT', 'ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'])
 for (const level of LOG_LEVELS) assert.ok(levelTone(level))
 assert.equal(levelTone('ERROR'), 'danger')
 assert.equal(levelTone('WARN'), 'warn')
 assert.equal(levelTone(undefined), 'none')
})

test('a log query never asks for more lines than the server accepts', async () => {
 const { LOG_PAGE_MAX } = await import('../../web/src/log-view.mjs')
 // internal/dashboard/monitor.go refuses limit above 100 with 400.
 assert.equal(LOG_PAGE_MAX, 100)
 assert.equal(new URLSearchParams(logQuery({ limit: 200 })).get('limit'), '100')
 assert.equal(new URLSearchParams(logQuery({ limit: 40 })).get('limit'), '40')
})
