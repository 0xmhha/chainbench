import test from 'node:test'
import assert from 'node:assert/strict'
import { differingParts } from '../../web/src/hex-diff.mjs'

test('only the characters that differ are marked', () => {
 assert.deepEqual(differingParts('0x2a', '0x2b'), [{ text: '0x2', differs: false }, { text: 'a', differs: true }])
 assert.deepEqual(differingParts('0x2b', '0x2a'), [{ text: '0x2', differs: false }, { text: 'b', differs: true }])
})

test('values of different lengths or kinds stay whole', () => {
 assert.deepEqual(differingParts('12', '123'), [{ text: '12', differs: false }])
 assert.deepEqual(differingParts(12, 13), [{ text: '12', differs: false }])
 assert.deepEqual(differingParts('same', 'same'), [{ text: 'same', differs: false }])
})
