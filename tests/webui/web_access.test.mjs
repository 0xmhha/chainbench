import test from 'node:test'
import assert from 'node:assert/strict'
import { openWebAccess } from '../../web/src/web-access.mjs'

function fixture(config, sessionStatus = 200) {
  const requests = []
  const session = { user: { id: 'stable-local-actor', role: 'administrator' }, csrfToken: 'browser-csrf', accountsEnabled: false }
  const fetcher = async (url, options) => {
    requests.push([url, options?.method ?? 'GET'])
    if (url === '/api/v1/auth/config') return Response.json(config)
    return Response.json(sessionStatus === 200 ? session : { message: 'login required' }, { status: sessionStatus })
  }
  return { fetcher, requests, session }
}

test('accounts=false automatically opens local access without login or bootstrap', async () => {
  const { fetcher, requests, session } = fixture({ accountsEnabled: false, bootstrapRequired: false })
  const access = await openWebAccess(fetcher)
  assert.equal(access.accountsEnabled, false)
  assert.equal(access.bootstrapRequired, false)
  assert.deepEqual(access.session, session)
  assert.deepEqual(requests, [['/api/v1/auth/config', 'GET'], ['/api/v1/auth/local', 'POST']])
})

test('accounts=true preserves the login requirement and administrator bootstrap', async () => {
  const { fetcher, requests } = fixture({ accountsEnabled: true, bootstrapRequired: true }, 401)
  const access = await openWebAccess(fetcher)
  assert.equal(access.accountsEnabled, true)
  assert.equal(access.bootstrapRequired, true)
  assert.equal(access.session, null)
  assert.deepEqual(requests, [['/api/v1/auth/config', 'GET'], ['/api/v1/auth/me', 'GET']])
})

test('refreshing access observes a service mode change', async () => {
  const local = fixture({ accountsEnabled: false, bootstrapRequired: false })
  assert.ok((await openWebAccess(local.fetcher)).session)
  const accounts = fixture({ accountsEnabled: true, bootstrapRequired: false }, 401)
  assert.equal((await openWebAccess(accounts.fetcher)).session, null)
  assert.ok((await openWebAccess(local.fetcher)).session)
})

test('legacy services and older account configs retain their existing behavior', async () => {
  const legacy = await openWebAccess(async () => new Response('', { status: 404 }))
  assert.equal(legacy.session, undefined)
  const old = fixture({ bootstrapRequired: false }, 401)
  assert.equal((await openWebAccess(old.fetcher)).accountsEnabled, true)
})
