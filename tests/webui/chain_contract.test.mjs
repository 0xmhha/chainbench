import test from 'node:test'
import assert from 'node:assert/strict'
import { topologyChoices, draftChain } from '../../web/src/chain-contract.mjs'

const contract = { $defs: { envSpec: { properties: { topology: { oneOf: [
 { title: 'Node counts', properties: { bp: { type: 'integer', default: 4, minimum: 1, title: '블록 생산자' }, en: { type: 'integer', default: 0, minimum: 0, title: '엔드포인트' }, syncMode: { type: 'string', enum: ['full', 'snap'], default: 'full', title: '동기화 방식' } } },
 { title: 'Node table', properties: { nodes: { items: { properties: { role: { enum: ['bp', 'en'] } } } } } }
] } } } } }

test('topology choices come from the chain contract', () => {
 const c = topologyChoices(contract)
 assert.deepEqual(c.roles, ['bp', 'en'])
 assert.deepEqual(c.defaults, { bp: 4, en: 0 })
 assert.deepEqual(c.counts.map(f => [f.id, f.title, f.minimum]), [['bp', '블록 생산자', 1], ['en', '엔드포인트', 0]])
 assert.deepEqual(c.sync, { title: '동기화 방식', options: ['full', 'snap'], default: 'full' })
 assert.deepEqual(topologyChoices(null), { roles: [], defaults: {}, counts: [], sync: null })
})

test('a draft names its chain in either grammar, directly or through its preset', () => {
 assert.equal(draftChain({ schemaVersion: '1', chain: { name: 'wbft' } }), 'wbft')
 assert.equal(draftChain({ schemaVersion: '2', chainPreset: { chain: 'stablenet' } }), 'stablenet')
 assert.equal(draftChain({ schemaVersion: '2', chainPreset: 'p' }, { p: { chain: 'wemix' } }), 'wemix')
 assert.equal(draftChain({ schemaVersion: '2', chainPreset: 'missing' }, {}), '')
 assert.equal(draftChain({ schemaVersion: '2', chainPreset: { extends: 'p' } }, { p: { chain: 'wemix' } }), 'wemix')
})
