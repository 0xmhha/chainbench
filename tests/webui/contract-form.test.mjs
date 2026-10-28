import test from 'node:test'
import assert from 'node:assert/strict'
import { fieldTitle, remainder, changedPaths } from '../../web/src/contract-form.mjs'

const schema = { type: 'object', title: '서버 풀', properties: {
 version: { type: 'integer', title: '형식 버전' },
 pool: { type: 'object', title: '풀', properties: {
  hosts: { type: 'array', title: '서버', items: { type: 'object', title: '서버', properties: { name: { type: 'string', title: '이름' }, addr: { type: 'string', title: '주소' } } } },
  slots: { type: 'integer', title: '서버당 노드 수' },
  ports: { type: 'object', title: '포트 대역', properties: { p2p: { type: 'object', title: 'P2P', properties: { base: { type: 'integer', title: 'P2P 시작 포트' } } } } },
 } },
 ssh: { type: 'object', title: 'SSH 연결', properties: { port: { type: 'integer', title: 'SSH 포트' } } },
 aliases: { type: 'object', title: '별칭', additionalProperties: { type: 'string', title: '별칭 경로' } },
} }

test('a field is named by its titles, with an array position counted from one', () => {
 assert.equal(fieldTitle(schema, 'content.pool.hosts[1].addr'), '풀 · 서버 2 · 주소')
 assert.equal(fieldTitle(schema, 'content.pool.ports.p2p.base'), '풀 · 포트 대역 · P2P · P2P 시작 포트')
 assert.equal(fieldTitle(schema, 'content.aliases.gwemix'), '별칭 · gwemix')
 assert.equal(fieldTitle(schema, 'name'), '이름')
 assert.equal(fieldTitle(schema, 'content.unknown.key'), 'content.unknown.key', 'a path the contract does not know stays as it is')
})

test('what a laid-out form does not place is left for the general fields', () => {
 const rest = remainder(schema, ['version', 'pool.hosts', 'pool.slots', 'pool.ports'])
 assert.deepEqual(Object.keys(rest.properties), ['ssh', 'aliases'])
 assert.equal(schema.properties.pool.properties.hosts.title, '서버', 'the contract itself is not changed')
 assert.deepEqual(Object.keys(remainder(schema, ['pool.slots']).properties.pool.properties), ['hosts', 'ports'])
})

test('a past revision differs from the current one at exactly the changed fields', () => {
 const before = { version: 2, pool: { hosts: [{ name: 'a', addr: '10.0.0.1' }], slots: 1 } }
 const after = { version: 2, pool: { hosts: [{ name: 'a', addr: '10.0.0.2' }, { name: 'b', addr: '10.0.0.3' }], slots: 1 }, ssh: { port: 22 } }
 assert.deepEqual([...changedPaths(before, after)].sort(), ['content.pool.hosts[0].addr', 'content.pool.hosts[1].addr', 'content.pool.hosts[1].name', 'content.ssh.port'])
 assert.equal(changedPaths(before, before).size, 0)
})
