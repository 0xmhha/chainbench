import test from 'node:test'
import assert from 'node:assert/strict'
import { statusIcon } from '../../web/src/status-icon.mjs'

test('each state gets an icon and a tone, so color is never the only signal', () => {
 assert.deepEqual(statusIcon('succeeded'), { icon: '✓', tone: 'ok' })
 assert.deepEqual(statusIcon('pass'), { icon: '✓', tone: 'ok' })
 assert.deepEqual(statusIcon('verified'), { icon: '✓', tone: 'ok' })
 assert.deepEqual(statusIcon('failed'), { icon: '✕', tone: 'bad' })
 assert.deepEqual(statusIcon('cleanup_failed'), { icon: '✕', tone: 'bad' })
 assert.deepEqual(statusIcon('running'), { icon: '●', tone: 'active' })
 assert.deepEqual(statusIcon('cancelling'), { icon: '◐', tone: 'warn' })
 assert.deepEqual(statusIcon('interrupted'), { icon: '!', tone: 'warn' })
 assert.deepEqual(statusIcon('stale'), { icon: '!', tone: 'warn' })
 assert.deepEqual(statusIcon('cancelled'), { icon: '○', tone: 'neutral' })
 assert.deepEqual(statusIcon('anything-else'), { icon: '?', tone: 'neutral' })
})
