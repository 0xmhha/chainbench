import test from 'node:test'
import assert from 'node:assert/strict'
import { schemaProblems, fieldPath, schemaDefault } from '../../web/src/schema-problems.mjs'

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

test('a server pointer into the content names the same field the form does', () => {
 const s = { type: 'object', properties: { pool: schema.properties.pool, aliases: { type: 'object', additionalProperties: { type: 'string' } } } }
 assert.equal(fieldPath(s, '/content/pool/hosts/1/addr'), 'content.pool.hosts[1].addr')
 assert.equal(fieldPath(s, '/content/pool/slots'), 'content.pool.slots')
 assert.equal(fieldPath(s, '/content/aliases/linux~1amd'), 'content.aliases.linux/amd')
 assert.equal(fieldPath(s, '/content/aliases/0'), 'content.aliases.0')
 assert.equal(fieldPath(s, '/name'), 'name')
 assert.equal(fieldPath(s, '/content'), '')
 assert.equal(fieldPath(s, undefined), '')
})

test('a new field starts at the engine default the contract offers, and only then at an empty value', () => {
 const ports = { type: 'object', required: ['base', 'step'], properties: { base: { type: 'integer', default: 31000 }, step: { type: 'integer', default: 10 } } }
 assert.deepEqual(schemaDefault(ports), { base: 31000, step: 10 })
 assert.equal(schemaDefault({ type: 'integer' }), 0)
 assert.equal(schemaDefault({ type: 'string', enum: ['fresh', 'attach'] }), 'fresh')
 assert.equal(schemaDefault({ type: 'string', enum: ['fresh', 'attach'], default: 'attach' }), 'attach')
 assert.deepEqual(schemaDefault({ type: 'array', items: ports }), [])
})
