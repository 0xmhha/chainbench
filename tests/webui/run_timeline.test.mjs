import test from 'node:test'
import assert from 'node:assert/strict'
import { testRows, verdictSegments, placeOnAxis, testLogSpan, phaseSegments } from '../../web/src/run-timeline.mjs'

const summary = {
 tests: [
  { seq: 1, id: 'deploy', status: 'pass', startedAt: '2026-10-23T07:00:00Z', finishedAt: '2026-10-23T07:00:10Z' },
  { seq: 2, id: 'transfer', status: 'fail', startedAt: '2026-10-23T07:00:10Z', finishedAt: '2026-10-23T07:00:40Z' },
  { seq: 3, id: 'governance', status: 'blocked' },
  { seq: 4, id: 'cleanup', status: 'skip' }
 ],
 results: {
  'tests/002_transfer/assert.json': [
   { ID: 'a1', Source: 'rpc', On: 'node2', Assert: 'balance == 10', Expected: '10', Actual: '9', Pass: false },
   { ID: 'a2', Source: 'rpc', On: 'node1', Assert: 'height > 0', Expected: '>0', Actual: '5', Pass: true }
  ],
  'tests/002_transfer/steps.json': [{ Index: 0, Type: 'tx', On: 'node3', Error: 'nonce too low' }, { Index: 1, Type: 'tx', On: 'node1', Error: '' }],
  'tests/002_transfer/status.json': { result: 'fail', reason: 'assertion a1 failed' },
  'tests/003_governance/status.json': { result: 'blocked', reason: 'pre-action failed' }
 }
}

test('failures come first, each with the assertions and steps that broke', () => {
 const rows = testRows(summary)
 assert.deepEqual(rows.map(r => r.id), ['transfer', 'governance', 'deploy', 'cleanup'])
 const failed = rows[0]
 assert.equal(failed.durationMs, 30000)
 assert.equal(failed.reason, 'assertion a1 failed')
 assert.deepEqual(failed.failedAsserts.map(a => [a.id, a.on, a.expected, a.actual]), [['a1', 'node2', '10', '9']])
 assert.deepEqual(failed.stepErrors.map(s => [s.index, s.on, s.error]), [[0, 'node3', 'nonce too low']])
 assert.deepEqual(failed.nodes, ['node2', 'node3'])
 assert.equal(rows[1].reason, 'pre-action failed')
 assert.equal(rows[1].durationMs, null)
})

test('a run with no tests or results gives no rows instead of throwing', () => {
 assert.deepEqual(testRows({}), [])
 assert.deepEqual(testRows({ tests: [{ seq: 1, id: 'x', status: 'pass' }] }).map(r => r.failedAsserts), [[]])
})

test('the verdict bar shows only verdicts that occurred, in a fixed order', () => {
 assert.deepEqual(verdictSegments({ pass: 6, fail: 2, blocked: 0, skip: 2 }), [
  { status: 'fail', count: 2, percent: 20 },
  { status: 'pass', count: 6, percent: 60 },
  { status: 'skip', count: 2, percent: 20 }
 ])
 assert.deepEqual(verdictSegments({}), [])
})

test('timed tests sit on the run axis; untimed ones are left off it', () => {
 const placed = placeOnAxis(testRows(summary), '2026-10-23T07:00:00Z', '2026-10-23T07:01:40Z')
 const transfer = placed.find(r => r.id === 'transfer')
 assert.deepEqual([transfer.left, transfer.width], [10, 30])
 assert.equal(placed.find(r => r.id === 'governance').left, null)
 // A zero-length test still gets a visible sliver.
 const instant = placeOnAxis([{ id: 'i', startedAt: '2026-10-23T07:00:00Z', finishedAt: '2026-10-23T07:00:00Z' }], '2026-10-23T07:00:00Z', '2026-10-23T07:01:40Z')
 assert.ok(instant[0].width >= 0.5)
})

test('a failed test opens the logs of its own span, or the whole run when it has no times', () => {
 const rows = testRows(summary)
 assert.deepEqual(testLogSpan(rows[0], 5), { from: '2026-10-23T07:00:05.000Z', to: '2026-10-23T07:00:45.000Z', exact: true })
 assert.deepEqual(testLogSpan(rows[1], 5), { from: null, to: null, exact: false })
})

test('job phases become shares of the time they took, untimed phases flagged', () => {
 const segments = phaseSegments([
  { name: 'place', state: 'succeeded', startedAt: '2026-10-23T07:00:00Z', finishedAt: '2026-10-23T07:00:10Z' },
  { name: 'start', state: 'failed', startedAt: '2026-10-23T07:00:10Z', finishedAt: '2026-10-23T07:00:40Z', message: 'node3 exited' },
  { name: 'verify', state: 'pending' }
 ])
 assert.deepEqual(segments.map(s => [s.name, s.share, s.timed]), [['place', 25, true], ['start', 75, true], ['verify', 0, false]])
 assert.equal(segments[1].message, 'node3 exited')
})
