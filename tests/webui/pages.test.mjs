import test from 'node:test'
import assert from 'node:assert/strict'
import { allItems } from '../../web/src/pages.mjs'

test('every page of a list is read by following its cursor', async () => {
  const seen = []
  const get = async path => {
    seen.push(path)
    if (!path.includes('cursor=')) return { items: [1, 2], nextCursor: 'b' }
    if (path.includes('cursor=b')) return { items: [3], nextCursor: null }
    throw new Error('unexpected ' + path)
  }
  assert.deepEqual(await allItems(get, 'documents?kind=case'), [1, 2, 3])
  assert.deepEqual(seen, ['documents?kind=case&limit=100', 'documents?kind=case&limit=100&cursor=b'])
  assert.deepEqual(await allItems(async () => ({ items: [], nextCursor: null }), 'users'), [])
})
