import test from 'node:test'
import assert from 'node:assert/strict'
import { schemaProblems } from '../../web/src/schema-problems.mjs'

const schema = { type: 'object', required: ['pool'], properties: { pool: { type: 'object', required: ['hosts', 'slots'], properties: {
 hosts: { type: 'array', items: { type: 'object', required: ['addr'], properties: { name: { type: 'string' }, addr: { type: 'string', minLength: 1 } } } },
 slots: { type: 'integer', minimum: 1, maximum: 64 },
 mode: { type: 'string', enum: ['a', 'b'] },
 port: { type: 'string', pattern: '^[0-9]+$' },
} } } }

test('each schema rule a value breaks is named at the field path the form uses', () => {
 const problems = schemaProblems(schema, { pool: { hosts: [{ name: 'x', addr: '' }, { name: 'y' }], slots: 0, mode: 'c', port: 'p' } })
 assert.deepEqual(problems.map(p => p.path), ['content.pool.hosts[0].addr', 'content.pool.hosts[1].addr', 'content.pool.slots', 'content.pool.mode', 'content.pool.port'])
 assert.ok(problems.every(p => p.message))
 assert.deepEqual(schemaProblems(schema, { pool: { hosts: [{ addr: '10.0.0.1' }], slots: 2 } }), [])
 assert.deepEqual(schemaProblems(schema, {}).map(p => p.path), ['content.pool'])
})

test('a required string left empty is as missing as an absent one', () => {
 const s = { type: 'object', required: ['addr'], properties: { addr: { type: 'string' } } }
 assert.deepEqual(schemaProblems(s, { addr: '' }).map(p => p.path), ['content.addr'])
 assert.deepEqual(schemaProblems(s, { addr: 'x' }), [])
})
