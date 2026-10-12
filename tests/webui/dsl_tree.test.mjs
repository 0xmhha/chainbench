import test from 'node:test'
import assert from 'node:assert/strict'
import { statementLists, statementOf, declarationDiff, duplicateCase } from '../../web/src/dsl-tree.mjs'

const v2 = { schemaVersion: '2', kind: 'case', id: 'c', chainPreset: { chain: 'wbft' }, hooks: { pre: [{ do: 'newAccount', save: 'a' }] }, steps: [{ do: 'read', source: 'blockNumber', on: 'node1' }, { expect: 'blockNumber', on: 'node2', is: 1 }] }

test('statements are listed per list in grammar order with their path and summary', () => {
 const lists = statementLists(v2)
 assert.deepEqual(lists.map(l => l.path), ['/content/hooks/pre', '/content/steps', '/content/hooks/post', '/content/hooks/onFail'])
 assert.deepEqual(lists[1].items.map(i => [i.path, i.summary]), [['/content/steps/0', 'do read · node1'], ['/content/steps/1', 'expect blockNumber · node2']])
 const v1 = statementLists({ schemaVersion: '1', steps: [{ read: { source: 'x' } }], assertions: [{ assert: 'blockNumber' }] })
 assert.deepEqual(v1.map(l => l.path), ['/content/preActions', '/content/steps', '/content/assertions', '/content/postActions'])
 assert.equal(v1[1].items[0].summary, 'read')
})

test('an error path selects the statement it is inside, or the case settings', () => {
 assert.equal(statementOf(v2, '/content/steps/1/is'), '/content/steps/1')
 assert.equal(statementOf(v2, '/content/hooks/pre/0'), '/content/hooks/pre/0')
 assert.equal(statementOf(v2, '/content/chainPreset/topology'), '')
 assert.equal(statementOf(v2, '/content/steps/9'), '', 'a statement that is not there is not selected')
})

test('an import diff names what migration added, removed or changed', () => {
 const diff = declarationDiff({ id: 'c', chain: { name: 'wbft' }, steps: [1] }, { id: 'c', chainPreset: { chain: 'wbft' }, steps: [2] })
 assert.deepEqual(diff, [{ path: '/chain', kind: 'removed' }, { path: '/chainPreset', kind: 'added' }, { path: '/steps/0', kind: 'changed' }])
 assert.deepEqual(declarationDiff({ a: [1, 2] }, { a: [1, 2] }), [])
})

test('a duplicate is a new case with its own id', () => {
 const copy = duplicateCase(v2, ['c-copy'])
 assert.equal(copy.id, 'c-copy-2')
 assert.deepEqual(copy.steps, v2.steps)
 assert.notEqual(copy.steps, v2.steps, 'the copy shares no structure with the original')
})

test('schemaAt follows properties, items and references to a statement schema', async () => {
 const { schemaAt } = await import('../../web/src/dsl-tree.mjs')
 const root = { $defs: { stmt: { type: 'object', properties: { do: { const: 'read' } } }, hooks: { type: 'object', properties: { pre: { type: 'array', items: { $ref: '#/$defs/stmt' } } } } } }
 const caseSpec = { type: 'object', properties: { steps: { type: 'array', items: { $ref: '#/$defs/stmt' } }, hooks: { $ref: '#/$defs/hooks' } } }
 assert.deepEqual(schemaAt(caseSpec, root, '/steps/3'), root.$defs.stmt)
 assert.deepEqual(schemaAt(caseSpec, root, '/hooks/pre/0'), root.$defs.stmt)
 assert.deepEqual(schemaAt(caseSpec, root, '/nope'), {})
})

test('case settings leave out the statement lists, and changes merge back', async () => {
 const { settingsOf, withSettings, valueAt, withValue } = await import('../../web/src/dsl-tree.mjs')
 const schema = { type: 'object', required: ['id', 'steps'], properties: { id: {}, steps: {}, hooks: {}, chainPreset: {} } }
 const { value, schema: s } = settingsOf(v2, schema)
 assert.deepEqual(Object.keys(value).sort(), ['chainPreset', 'id', 'kind', 'schemaVersion'])
 assert.deepEqual(Object.keys(s.properties).sort(), ['chainPreset', 'id'])
 assert.deepEqual(s.required, ['id'])
 const next = withSettings(v2, { ...value, id: 'renamed' })
 assert.equal(next.id, 'renamed'); assert.deepEqual(next.steps, v2.steps); assert.deepEqual(next.hooks, v2.hooks)
 assert.deepEqual(valueAt(v2, '/content/steps/1'), v2.steps[1])
 const changed = withValue(v2, '/content/steps/1', { expect: 'blockNumber', on: 'node3', is: 2 })
 assert.equal(changed.steps[1].on, 'node3'); assert.equal(v2.steps[1].on, 'node2', 'the original is not changed')
 assert.equal(withValue(v2, '/content/hooks/pre/0', undefined).hooks.pre.length, 0, 'undefined removes a statement')
})
