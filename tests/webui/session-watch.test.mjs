import test from 'node:test'
import assert from 'node:assert/strict'
import { expiryDelay } from '../../web/src/session-watch.mjs'

test('the page checks its session when the server says it expires, not every few seconds', () => {
 const now = Date.parse('2026-10-28T00:00:00Z')
 assert.equal(expiryDelay({ expiresAt: '2026-10-28T00:10:00Z' }, now), 600_000)
 assert.equal(expiryDelay({ expiresAt: '2026-10-27T23:00:00Z' }, now), 0)
 assert.equal(expiryDelay({ expiresAt: '2026-12-31T00:00:00Z' }, now), 2 ** 31 - 1, 'a timer cannot wait longer than this')
 assert.equal(expiryDelay({}, now), null)
 assert.equal(expiryDelay({ expiresAt: 'soon' }, now), null)
})
