import test from 'node:test'
import assert from 'node:assert/strict'
import { gapText, stateLabel, formatTime } from '../../web/src/labels.mjs'

test('a coverage gap reads as Korean with its node, and keeps the engine code', () => {
 assert.deepEqual(gapText('node2 rpc rpc_unavailable'), { text: 'node2 · RPC 응답 없음', code: 'rpc_unavailable' })
 assert.deepEqual(gapText('collector not_collected'), { text: '수집 시작 전 구간', code: 'not_collected' })
 assert.deepEqual(gapText('node1 logs remote_log_collection_unavailable'), { text: 'node1 · 원격 로그 수집 작업이 없던 구간', code: 'remote_log_collection_unavailable' })
 assert.deepEqual(gapText('archive corrupt_record 3'), { text: '손상된 보관 기록 3건', code: 'corrupt_record' })
 // An unknown code is shown as it came, never dropped.
 assert.deepEqual(gapText('node1 rpc brand_new_reason'), { text: 'node1 · brand_new_reason', code: 'brand_new_reason' })
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
