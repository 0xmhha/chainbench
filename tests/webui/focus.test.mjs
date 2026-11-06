import test from 'node:test'
import assert from 'node:assert/strict'
import { keepFocus } from '../../web/src/focus.mjs'

function page() {
 const body = { isConnected: true }
 const control = { isConnected: true, focused: 0, focus() { this.focused++ } }
 return { body, control, activeElement: control }
}

test('a control that lost focus to the body gets it back', async () => {
 const doc = page()
 const restore = keepFocus(doc)
 doc.activeElement = doc.body
 await restore()
 assert.equal(doc.control.focused, 1)
})

test('focus moved on purpose stays where it went', async () => {
 const doc = page()
 const restore = keepFocus(doc)
 doc.activeElement = { isConnected: true }
 await restore()
 assert.equal(doc.control.focused, 0)
})

test('a control removed from the page is left alone', async () => {
 const doc = page()
 const restore = keepFocus(doc)
 doc.control.isConnected = false
 doc.activeElement = doc.body
 await restore()
 assert.equal(doc.control.focused, 0)
})
