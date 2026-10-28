import test from 'node:test'
import assert from 'node:assert/strict'
import { readdirSync, readFileSync } from 'node:fs'

// Every screen reaches /api/v1 through the shared client, which sends the
// session token, If-Match and Idempotency-Key and reads every error the same
// way. Only the client itself and the access bootstrap that runs before a
// session exists call fetch directly.
const allowed = new Set(['api-client.mjs', 'web-api.mjs', 'web-access.mjs'])

test('no screen calls fetch outside the shared client', () => {
 const dir = new URL('../../web/src/', import.meta.url)
 const offenders = readdirSync(dir).filter(name => /\.(svelte|m?js)$/.test(name) && !allowed.has(name) && /\bfetch\(/.test(readFileSync(new URL(name, dir), 'utf8')))
 assert.deepEqual(offenders, [])
})
