import test from 'node:test'
import assert from 'node:assert/strict'
import { testRows, testLogSpan, phaseSegments } from '../../web/src/run-timeline.mjs'

// A run detail as the server sends it: each test with its result files read.
const test_ = (seq, id, state, over = {}) => ({ seq, id, state, assertions: [], steps: [], postActions: [], artifacts: [], ...over })
const run = {
 tests: [
  test_(1, 'deploy', 'succeeded', { startedAt: '2026-10-23T07:00:00Z', finishedAt: '2026-10-23T07:00:10Z' }),
  test_(2, 'transfer', 'failed', {
   startedAt: '2026-10-23T07:00:10Z', finishedAt: '2026-10-23T07:00:40Z', reason: 'assertion a1 failed',
   assertions: [
    { assert: 'balance', nodeId: 'node2', compare: 'Equal', expected: '10', actual: '9', passed: false, detail: 'node2: actual=9 expected=10' },
    { assert: 'height', nodeId: 'node1', expected: '0', actual: '5', passed: true },
   ],
   steps: [{ index: 0, type: 'tx', nodeId: 'node3', error: 'nonce too low' }, { index: 1, type: 'tx', nodeId: 'node1' }],
  }),
  test_(3, 'governance', 'blocked', { reason: 'pre-action failed' }),
  test_(4, 'cleanup', 'skipped'),
 ],
}

test('failures come first, each with the assertions and steps that broke', () => {
 const rows = testRows(run)
 assert.deepEqual(rows.map(r => r.id), ['transfer', 'governance', 'deploy', 'cleanup'])
 const failed = rows[0]
 assert.equal(failed.durationMs, 30000)
 assert.equal(failed.reason, 'assertion a1 failed')
 assert.deepEqual(failed.failedAsserts.map(a => [a.assert, a.on, a.compare, a.expected, a.actual]), [['balance', 'node2', 'Equal', '10', '9']])
 assert.deepEqual(failed.stepErrors.map(s => [s.index, s.on, s.error]), [[0, 'node3', 'nonce too low']])
 assert.deepEqual(failed.nodes, ['node2', 'node3'])
 assert.equal(rows[1].reason, 'pre-action failed')
 assert.equal(rows[1].durationMs, null)
})

test('a run with no tests or results gives no rows instead of throwing', () => {
 assert.deepEqual(testRows({}), [])
 assert.deepEqual(testRows(null), [])
 assert.deepEqual(testRows({ tests: [test_(1, 'x', 'succeeded')] }).map(r => [r.status, r.failedAsserts]), [['succeeded', []]])
})

test('a failed test opens the logs of its own span, or the whole run when it has no times', () => {
 const rows = testRows(run)
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
