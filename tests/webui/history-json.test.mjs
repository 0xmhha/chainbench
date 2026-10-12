import test from 'node:test'
import assert from 'node:assert/strict'
import { parseHistoryJSON } from '../../web/src/history-json.js'

test('large integer evidence stays exact for browser display', () => {
  const value = parseHistoryJSON('{"nonce":18446744073709551615,"height":9007199254740993,"count":4,"metric":1.25}')
  assert.equal(value.nonce, '18446744073709551615')
  assert.equal(value.height, '9007199254740993')
  assert.equal(value.count, 4)
  assert.equal(value.metric, 1.25)
})

test('unrepresentable numeric literals remain visible rather than becoming null', () => {
  const value = parseHistoryJSON('{"value":1e400,"result":"pass","missing":null}')
  assert.equal(value.value, '1e400')
  assert.equal(value.result, 'pass')
  assert.equal(value.missing, null)
})
