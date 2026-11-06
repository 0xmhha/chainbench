import test from 'node:test'
import assert from 'node:assert/strict'
import { warnWhenUnsaved } from '../../web/src/unsaved.mjs'

function fakeWindow() {
 const handlers = new Set()
 return {
  addEventListener(_, h) { handlers.add(h) },
  removeEventListener(_, h) { handlers.delete(h) },
  fire() { let prevented = false; for (const h of handlers) h({ preventDefault() { prevented = true } }); return prevented },
  count: () => handlers.size,
 }
}

test('leaving is held back only while something is unsaved', () => {
 const win = fakeWindow()
 let dirty = false
 const stop = warnWhenUnsaved(() => dirty, win)
 assert.equal(win.fire(), false)
 dirty = true
 assert.equal(win.fire(), true)
 stop()
 assert.equal(win.count(), 0)
})
