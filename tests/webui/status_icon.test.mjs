import test from 'node:test'
import assert from 'node:assert/strict'
import { statusIcon } from '../../web/src/status-icon.mjs'

test('each state gets a drawn mark and a tone, so color is never the only signal', () => {
 assert.deepEqual(statusIcon('succeeded'), { icon: 'check', tone: 'ok' })
 assert.deepEqual(statusIcon('pass'), { icon: 'check', tone: 'ok' })
 assert.deepEqual(statusIcon('verified'), { icon: 'check', tone: 'ok' })
 assert.deepEqual(statusIcon('failed'), { icon: 'cross', tone: 'bad' })
 assert.deepEqual(statusIcon('cleanup_failed'), { icon: 'cross', tone: 'bad' })
 assert.deepEqual(statusIcon('running'), { icon: 'dot', tone: 'active' })
 assert.deepEqual(statusIcon('cancelling'), { icon: 'half', tone: 'warn' })
 assert.deepEqual(statusIcon('interrupted'), { icon: 'bang', tone: 'warn' })
 assert.deepEqual(statusIcon('stale'), { icon: 'bang', tone: 'warn' })
 assert.deepEqual(statusIcon('cancelled'), { icon: 'ring', tone: 'neutral' })
 assert.deepEqual(statusIcon('anything-else'), { icon: 'unknown', tone: 'neutral' })
})
