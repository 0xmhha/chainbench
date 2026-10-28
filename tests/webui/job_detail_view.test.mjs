import test from 'node:test'
import assert from 'node:assert/strict'
import { jobSummary, failureHeadline, jobDuration, phaseRows, readJobFocus, jobFocusSearch } from '../../web/src/job-view.mjs'
import { failureMetrics } from '../../web/src/metric-chart.mjs'

const ZERO = '0001-01-01T00:00:00Z'
const failingRun = {
 id: 'run-1',
 summary: { counts: { fail: 1, pass: 1 } },
 tests: [
  { seq: 1, id: 'deploy', state: 'succeeded', startedAt: '2026-10-23T07:00:00Z', finishedAt: '2026-10-23T07:00:10Z', assertions: [], steps: [], postActions: [], artifacts: [] },
  { seq: 2, id: 'transfer', state: 'failed', startedAt: '2026-10-23T07:00:10Z', finishedAt: '2026-10-23T07:00:40Z', reason: 'assertion a1 failed',
   assertions: [{ assert: 'balance == 10', nodeId: 'node2', expected: '10', actual: '9', passed: false }], steps: [], postActions: [], artifacts: [] },
 ],
}
const ENDED = ['succeeded', 'failed', 'cancelled', 'interrupted']
const job = (over = {}) => { const state = over.state ?? 'failed'; return { id: 'j1', operation: 'test.run', state, terminal: ENDED.includes(state), needsCheck: false, createdAt: '2026-10-23T06:59:50Z', phases: [], runIds: [], ...over } }
const headline = (j, runs = []) => failureHeadline(j, jobSummary(j), runs)

test('a failed test names the test, the node and the broken assertion first', () => {
 const h = headline(job({ phases: [{ name: 'run', state: 'failed', message: 'tests failed' }] }), [failingRun])
 assert.equal(h.tone, 'fail')
 assert.equal(h.title, 'transfer 테스트가 node2에서 실패했습니다')
 assert.equal(h.reason, 'balance == 10 · 기대 10 · 실제 9')
 assert.deepEqual([h.phase, h.test, h.node, h.runId], ['run', 'transfer', 'node2', 'run-1'])
})

test('a test without a failing assertion falls back to its step error, then its reason', () => {
 const step = structuredClone(failingRun)
 Object.assign(step.tests[1], { assertions: [], steps: [{ index: 3, type: 'tx', nodeId: 'node3', error: 'nonce too low' }] })
 assert.deepEqual([headline(job(), [step]).title, headline(job(), [step]).reason], ['transfer 테스트가 node3에서 실패했습니다', '#3 tx · nonce too low'])
 const bare = structuredClone(failingRun)
 Object.assign(bare.tests[1], { assertions: [], steps: [], reason: 'timeout' })
 assert.deepEqual([headline(job(), [bare]).title, headline(job(), [bare]).reason, headline(job(), [bare]).node], ['transfer 테스트가 실패했습니다', 'timeout', ''])
})

test('a failed phase without a failing test names the phase and its message', () => {
 const h = headline(job({ phases: [{ name: 'deploy', state: 'failed', message: 'port 8545 in use' }] }))
 assert.deepEqual([h.tone, h.title, h.reason, h.phase, h.test], ['fail', 'deploy 단계에서 실패했습니다', 'port 8545 in use', 'deploy', ''])
})

test('success, running, cancelled and interrupted jobs each read differently', () => {
 assert.deepEqual(Object.values((({ tone, title }) => ({ tone, title }))(headline(job({ state: 'succeeded' })))), ['ok', '작업이 성공했습니다'])
 const running = headline(job({ state: 'running', phases: [{ name: 'deploy', state: 'succeeded' }, { name: 'start', state: 'running' }] }))
 assert.deepEqual([running.tone, running.title], ['active', 'start 단계를 실행하고 있습니다'])
 const cancelled = headline(job({ state: 'cancelled', cancelReason: 'user', cancelMessage: 'stop' }))
 assert.deepEqual([cancelled.tone, cancelled.title, cancelled.reason], ['warn', '작업이 취소되었습니다', 'user · stop'])
 const interrupted = headline(job({ state: 'interrupted' }))
 assert.deepEqual([interrupted.tone, interrupted.title], ['warn', '작업이 중단되었습니다'])
 assert.match(interrupted.reason, /실제 상태/)
 const failed = headline(job({ error: { message: 'plan expired' } }))
 assert.deepEqual([failed.title, failed.reason], ['작업이 실패했습니다', 'plan expired'])
})

test('duration runs from creation to the last phase end and ignores unset times', () => {
 assert.equal(jobDuration(job({ phases: [{ finishedAt: '2026-10-23T07:00:00Z' }, { finishedAt: ZERO }] })), 10000)
 assert.equal(jobDuration(job({ phases: [{ finishedAt: ZERO }] })), null)
 assert.equal(jobDuration(job({ createdAt: ZERO, phases: [{ finishedAt: '2026-10-23T07:00:00Z' }] })), null)
})

test('phase rows treat the zero time as missing and open only the failed phase', () => {
 const rows = phaseRows([
  { name: 'deploy', state: 'succeeded', startedAt: '2026-10-23T07:00:00Z', finishedAt: '2026-10-23T07:00:02Z' },
  { name: 'run', state: 'failed', startedAt: '2026-10-23T07:00:02Z', finishedAt: '2026-10-23T07:00:05Z', message: 'x' },
  { name: 'cleanup', state: 'pending', startedAt: ZERO, finishedAt: ZERO }
 ])
 assert.deepEqual(rows.map(r => [r.name, r.durationMs, r.open]), [['deploy', 2000, false], ['run', 3000, true], ['cleanup', null, false]])
 assert.deepEqual([rows[2].startedAt, rows[2].finishedAt], [null, null])
})

test('the URL keeps the chosen test and node, and only those', () => {
 assert.deepEqual(readJobFocus('?test=transfer&node=node2&x=1'), { test: 'transfer', node: 'node2' })
 assert.deepEqual(readJobFocus(''), { test: '', node: '' })
 assert.equal(jobFocusSearch({ test: 'a b', node: 'node2' }), '?test=a+b&node=node2')
 assert.equal(jobFocusSearch({ test: '', node: '' }), '')
})

test('failure metrics use the canonical series of each concept and fall back to the second source', () => {
 const series = (name, metric, canonical, nodeId) => ({ name, metric, canonical, nodeId, unit: 'u', source: canonical ? 'rpc' : 'metrics', samples: [{ time: '2026-10-23T07:00:00Z', value: 1 }] })
 const metrics = { series: [series('chain_head_block', 'block_height', false, 'node2'), series('block_height', 'block_height', true, 'node2'), series('p2p_peers', 'peer_count', false, 'node2'), series('txpool_pending', 'txpool_pending', true, 'node1')] }
 assert.deepEqual(failureMetrics(metrics, 'node2').map(m => [m.label, m.name, m.series.length]), [['블록 높이', 'block_height', 1], ['피어 수', 'p2p_peers', 1], ['대기 트랜잭션', 'txpool_pending', 0]])
 // The concept, not a list of spellings, decides: a renamed gauge still lands.
 const renamed = { series: [series('eth_head', 'block_height', true, 'node2')] }
 assert.equal(failureMetrics(renamed, 'node2')[0].name, 'eth_head')
})

test('a phase message that names exactly one network node points at it', async () => {
 const { nodeInMessage } = await import('../../web/src/job-view.mjs')
 const nodes = ['node1', 'node2', 'node3', 'node10']
 assert.equal(nodeInMessage('node3 did not answer RPC on 127.0.0.1:8547', nodes), 'node3')
 assert.equal(nodeInMessage('node1 and node2 disagree', nodes), '')
 assert.equal(nodeInMessage('node10 stalled', nodes), 'node10')
 assert.equal(nodeInMessage('port in use', nodes), '')
 assert.equal(nodeInMessage('', nodes), '')
})

test('an assertion between two integers shows their exact difference', async () => {
 const { assertDiff } = await import('../../web/src/run-timeline.mjs')
 assert.equal(assertDiff('1000000000000000000000', '999999999999999979000'), '-21000')
 assert.equal(assertDiff('10', '12'), '+2')
 assert.equal(assertDiff(10, 9), '-1')
 assert.equal(assertDiff('>10', '5'), '')
 assert.equal(assertDiff('5', '5'), '')
 assert.equal(assertDiff(null, '5'), '')
})

test('a recorded run opens with its own headline: the failing test first, then how it ended', async () => {
 const { runHeadline } = await import('../../web/src/job-view.mjs')
 const failed = runHeadline({ ...failingRun, state: 'failed' })
 // The run is the page itself, so its headline does not name it again.
 assert.deepEqual([failed.tone, failed.title, failed.reason, failed.test, failed.node, failed.runId], ['fail', 'transfer 테스트가 node2에서 실패했습니다', 'balance == 10 · 기대 10 · 실제 9', 'transfer', 'node2', ''])
 assert.deepEqual(Object.values((({ tone, title }) => ({ tone, title }))(runHeadline({ id: 'r', state: 'succeeded', tests: [] }))), ['ok', '실행이 성공했습니다'])
 assert.deepEqual([runHeadline({ id: 'r', state: 'running' }).tone, runHeadline({ id: 'r', state: 'running' }).title], ['active', '실행 중입니다'])
 assert.equal(runHeadline({ id: 'r', state: 'cancelled' }).tone, 'warn')
 assert.equal(runHeadline({ id: 'r', state: 'interrupted' }).title, '실행이 중단되었습니다')
 assert.equal(runHeadline({ id: 'r', state: 'failed', tests: [] }).title, '실행이 실패했습니다')
})

test('a run is read on its first failing test, or on its first test when none failed', async () => {
 const { benchTest } = await import('../../web/src/job-view.mjs')
 const { testRows } = await import('../../web/src/run-timeline.mjs')
 const rows = testRows(failingRun)
 assert.equal(benchTest(rows, '').id, 'transfer')
 assert.equal(benchTest(rows, 'deploy').id, 'deploy')
 assert.equal(benchTest(rows, 'missing').id, 'transfer')
 const passing = testRows({ tests: [failingRun.tests[0]] })
 assert.equal(benchTest(passing, '').id, 'deploy')
 assert.equal(benchTest([], ''), null)
})

test('a recorded job run without test verdicts names its failed phase and the reason', async () => {
 const { runHeadline } = await import('../../web/src/job-view.mjs')
 const run = { id: 'job-1', state: 'failed', tests: [], summary: { phases: [{ name: 'test.inputs', state: 'succeeded' }, { name: 'test.run', state: 'failed', message: 'one or more selected test cases failed or were blocked' }], error: { message: 'engine refused' } } }
 const h = runHeadline(run)
 assert.deepEqual([h.tone, h.title, h.reason, h.phase], ['fail', 'test.run 단계에서 실패했습니다', 'one or more selected test cases failed or were blocked', 'test.run'])
 assert.equal(runHeadline({ id: 'r', state: 'failed', summary: { error: { message: 'plan expired' } } }).reason, 'plan expired')
})
