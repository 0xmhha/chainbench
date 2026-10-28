import test from 'node:test'
import assert from 'node:assert/strict'
import { liveStale, shownState, feedPhase } from '../../web/src/node-view.mjs'

const now = Date.parse('2026-10-28T00:00:10Z')

test('live state is stale once the stream drops or falls silent for five seconds', () => {
 assert.equal(liveStale({ connection: '연결됨', lastObserved: '2026-10-28T00:00:08Z' }, now), false)
 assert.equal(liveStale({ connection: '연결됨', lastObserved: '2026-10-28T00:00:04Z' }, now), true)
 assert.equal(liveStale({ connection: '연결 끊김', lastObserved: '2026-10-28T00:00:09Z' }, now), true)
 assert.equal(liveStale({ connection: '복원 중', lastObserved: null }, now), true)
})

test('a node read while live is stale is shown as stale, keeping what it last was', () => {
 assert.deepEqual(shownState('running', false), { state: 'running', last: null })
 assert.deepEqual(shownState('running', true), { state: 'stale', last: 'running' })
 assert.deepEqual(shownState('stopped', true), { state: 'stale', last: 'stopped' })
})

test('a screen tells loading, unreachable and empty apart before it shows an empty list', () => {
 assert.equal(feedPhase({ snapshot: null, error: '' }), 'loading')
 assert.equal(feedPhase({ snapshot: null, error: '서버에 연결할 수 없습니다. 2초 뒤 다시 시도합니다.' }), 'unreachable')
 assert.equal(feedPhase({ snapshot: { jobs: [] }, error: '서버에 연결할 수 없습니다.' }), 'ready', 'a snapshot already shown stays, marked stale elsewhere')
})
