import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'

const dir = new URL('../../web/src/', import.meta.url).pathname
const sources = fs.readdirSync(dir).filter(f => /\.(svelte|mjs|js)$/.test(f)).map(f => [f, fs.readFileSync(path.join(dir, f), 'utf8')])

test('colors come from the design tokens defined once in theme.css', () => {
 for (const [file, text] of sources) {
  const lines = text.split('\n').filter(l => !l.includes(':global(:root)') && !l.includes('name="theme-color"'))
  const raw = lines.join('\n').match(/#[0-9a-fA-F]{6}\b|#[0-9a-fA-F]{3}\b(?![\w-])/g) ?? []
  assert.deepEqual(raw, [], `${file} uses colors outside the tokens`)
 }
})

test('no screen keeps its own Basic login', () => {
 for (const [file, text] of sources) {
  assert.ok(!/['"`]Basic /.test(text) && !/Authorization\s*:/.test(text) && !text.includes('deployment-account'), `${file} still sends Basic credentials`)
 }
})
