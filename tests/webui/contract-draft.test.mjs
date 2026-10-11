import test from 'node:test'
import assert from 'node:assert/strict'
import { draftMeta, contractChanged, changedDefinitions } from '../../web/src/contract-draft.mjs'

test('a draft carries the version and digest of the contract it was opened with', () => {
  assert.deepEqual(draftMeta({ 'x-contractVersion': '2', 'x-contractDigest': 'abc' }), { contractVersion: '2', contractDigest: 'abc' })
  assert.deepEqual(draftMeta(null), { contractVersion: undefined, contractDigest: undefined })
})

test('a refusal for a changed contract is recognised and the changed parts are named', () => {
  assert.ok(contractChanged('409: deployment object conflict: the engine contract changed since this draft was opened; validate it again before saving'))
  assert.ok(!contractChanged('409: revision changed; reload before editing'))
  const before = { $defs: { envSpec: { a: 1 }, step: { b: 1 } }, title: 'x', 'x-contractDigest': 'one' }
  const after = { $defs: { envSpec: { a: 2 }, step: { b: 1 }, reader: {} }, title: 'x', 'x-contractDigest': 'two' }
  assert.deepEqual(changedDefinitions(before, after), ['$defs.envSpec', '$defs.reader'])
})
