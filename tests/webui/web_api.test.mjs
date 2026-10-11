import test from 'node:test'
import assert from 'node:assert/strict'
import { webApi } from '../../web/src/web-api.mjs'

test('webApi sends the session CSRF token and the expected revision', async () => {
 const calls = []
 const api = webApi({ csrfToken: 'token' }, async (url, init) => { calls.push([url, init]); return new Response('{"ok":true}', { status: 200 }) })
 assert.deepEqual(await api('documents/a', 'PATCH', { name: 'x' }, 3), { ok: true })
 assert.equal(calls[0][0], '/api/v1/documents/a')
 assert.equal(calls[0][1].headers['X-CSRF-Token'], 'token')
 assert.equal(calls[0][1].headers['If-Match'], '"3"')
 assert.equal(calls[0][1].body, '{"name":"x"}')
})

test('webApi returns null for no content and raises the server reason', async () => {
 const empty = webApi({ csrfToken: 't' }, async () => new Response(null, { status: 204 }))
 assert.equal(await empty('credentials/a', 'DELETE'), null)
 const refused = webApi({}, async () => new Response('{"code":"forbidden","message":"관리자만 볼 수 있습니다"}', { status: 403 }))
 await assert.rejects(refused('audit'), /403: 관리자만 볼 수 있습니다/)
})
