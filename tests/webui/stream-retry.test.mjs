import test from 'node:test'
import assert from 'node:assert/strict'
import {retryDelay, unreachableMessage} from '../../web/src/stream-retry.mjs'

test('reconnecting waits longer after each failure, up to a ceiling', () => {
 assert.deepEqual([0, 1, 2, 3, 4, 5, 9].map(retryDelay), [2000, 4000, 8000, 16000, 30000, 30000, 30000])
})
test('a dropped connection says the server is unreachable and when it is tried again', () => {
 assert.equal(unreachableMessage(new TypeError('Failed to fetch'), 8000), '서버에 연결할 수 없습니다. 8초 뒤 다시 시도합니다.')
})
test('a server answer keeps its own reason', () => {
 assert.equal(unreachableMessage(new Error('503: draining'), 2000), '503: draining')
})
