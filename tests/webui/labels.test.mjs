import test from 'node:test'
import assert from 'node:assert/strict'
import { gapText, stateLabel, formatTime } from '../../web/src/labels.mjs'

const gap = (over) => ({ from: 'a', to: 'b', scope: 'network', source: 'collector', ...over })

test('a coverage gap reads as Korean with its node, and keeps the engine code', () => {
 assert.deepEqual(gapText(gap({ scope: 'node', nodeId: 'node2', source: 'rpc', code: 'rpc_unavailable' })), { text: 'node2 · RPC 응답 없음', code: 'rpc_unavailable', nodeId: 'node2' })
 assert.deepEqual(gapText(gap({ code: 'not_collected' })), { text: '수집 시작 전 구간', code: 'not_collected', nodeId: '' })
 assert.deepEqual(gapText(gap({ scope: 'node', nodeId: 'node1', source: 'logs', code: 'remote_log_collection_unavailable' })), { text: 'node1 · 원격 로그 수집 작업이 없던 구간', code: 'remote_log_collection_unavailable', nodeId: 'node1' })
 assert.deepEqual(gapText(gap({ scope: 'node', nodeId: 'node4', code: 'node_stopped' })), { text: 'node4 · 노드가 멈춰 있던 구간', code: 'node_stopped', nodeId: 'node4' })
 assert.deepEqual(gapText(gap({ source: 'archive', code: 'corrupt_record', reason: 'archive corrupt_record 3' })), { text: '손상된 보관 기록 3건', code: 'corrupt_record', nodeId: '' })
 // An unknown code is shown as it came, never dropped.
 assert.deepEqual(gapText(gap({ scope: 'node', nodeId: 'node1', source: 'rpc', code: 'brand_new_reason' })), { text: 'node1 · brand_new_reason', code: 'brand_new_reason', nodeId: 'node1' })
})

test('the value beside a code is named instead of read as the code', () => {
 // The joined string used to lose the metric and read the run id as the code.
 assert.deepEqual(gapText(gap({ scope: 'node', nodeId: 'node1', source: 'metrics', code: 'not_exposed', metric: 'txpool_pending' })), { text: 'node1 · txpool_pending 지표를 노드가 내놓지 않음', code: 'not_exposed', nodeId: 'node1' })
 assert.deepEqual(gapText(gap({ source: 'archive', code: 'deleted_by_admin', runId: 'job-0123' })), { text: '관리자가 지운 실행 job-0123의 구간', code: 'deleted_by_admin', nodeId: '' })
 // A node label that does not start with "node" is still a node.
 assert.equal(gapText(gap({ scope: 'node', nodeId: 'faucet', source: 'rpc', code: 'rpc_unavailable' })).nodeId, 'faucet')
})

test('states get one Korean word everywhere; unknown states stay as the engine wrote them', () => {
 assert.equal(stateLabel('failed'), '실패')
 assert.equal(stateLabel('fail'), '실패')
 assert.equal(stateLabel('blocked'), '막힘')
 assert.equal(stateLabel('skip'), '건너뜀')
 assert.equal(stateLabel('interrupted'), '중단됨')
 assert.equal(stateLabel('something_new'), '')
})

test('times use one local format with seconds', () => {
 const d = new Date(2026, 9, 23, 7, 5, 9)
 assert.equal(formatTime(d.toISOString()), '2026-10-23 07:05:09')
 assert.equal(formatTime(null), '—')
 assert.equal(formatTime('not a time'), '—')
})
