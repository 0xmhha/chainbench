import test from 'node:test'
import assert from 'node:assert/strict'
import { apiClient, onUnauthorized } from '../../web/src/api-client.mjs'
import { FormatError, decodeEvidencePage, decodeJob, decodeMetrics, decodeNetwork, decodeRun, decodeRunEvidence } from '../../web/src/api-decode.mjs'

const reply = (status, text) => new Response(status === 204 ? null : text, { status })

function recorder(status = 200, text = '{}') {
 const calls = []
 return { calls, fetcher: async (url, init) => { calls.push({ url, init }); return reply(status, text) } }
}

test('the client sends the session token, revision and idempotency key it is given', async () => {
 const { calls, fetcher } = recorder(200, '{"ok":true}')
 const { request } = apiClient({ csrfToken: 't0ken' }, fetcher)
 assert.deepEqual(await request('jobs', { method: 'POST', body: { planId: 'p' }, revision: 3, idempotencyKey: 'k1' }), { ok: true })
 const { url, init } = calls[0]
 assert.equal(url, '/api/v1/jobs')
 assert.equal(init.method, 'POST')
 assert.equal(init.body, '{"planId":"p"}')
 assert.deepEqual(init.headers, { 'Content-Type': 'application/json', 'X-CSRF-Token': 't0ken', 'If-Match': '"3"', 'Idempotency-Key': 'k1' })
})

test('a form body goes as is, without a JSON content type', async () => {
 const { calls, fetcher } = recorder()
 const form = new FormData()
 form.set('kind', 'binary')
 await apiClient({ csrfToken: 't' }, fetcher).request('assets', { method: 'POST', body: form })
 assert.equal(calls[0].init.body, form)
 assert.equal(calls[0].init.headers['Content-Type'], undefined)
})

test('an error response keeps the status and reason the screens show', async () => {
 const { fetcher } = recorder(422, '{"code":"invalid","message":"pool.slots must be at least 1"}')
 await assert.rejects(apiClient(null, fetcher).request('documents', { method: 'POST', body: {} }), err => err.status === 422 && err.message === '422: pool.slots must be at least 1')
})

test('an empty or 204 response is null, and a parser can read the text itself', async () => {
 assert.equal(await apiClient(null, recorder(204, '').fetcher).request('x', { method: 'DELETE' }), null)
 const value = await apiClient(null, recorder(200, '{"n":12345678901234567890}').fetcher).request('x', { parse: text => ({ raw: text }) })
 assert.deepEqual(value, { raw: '{"n":12345678901234567890}' })
})

test('a response of the wrong shape becomes a format error naming the field', async () => {
 const { fetcher } = recorder(200, JSON.stringify({ ...job(), terminal: 'yes' }))
 await assert.rejects(apiClient(null, fetcher).request('jobs/x', { decode: decodeJob }), err => err instanceof FormatError && err.kind === 'format' && err.path === 'terminal')
})

function job(over = {}) {
 return {
  id: 'j', actorId: 'a', workspaceId: 'w', planId: 'p', operation: 'test.run', state: 'running', terminal: false, needsCheck: false,
  phases: [{ name: 'test.binary', state: 'running', startedAt: '2026-10-27T00:00:00Z' }], createdAt: '2026-10-27T00:00:00Z',
  retention: 'retain', nodeDisposition: 'unknown', runIds: [], partialEffects: [], unresolvedResources: [], credentialRefs: [], ...over,
 }
}

test('decoders accept the model and pass unknown fields through', () => {
 const value = job({ addedLater: 1 })
 assert.equal(decodeJob(value), value)
 const metrics = {
  networkId: 'n', coverage: { from: 'a', to: 'b', sampleIntervalSeconds: 5, complete: false, downsampled: false, gaps: [{ from: 'a', to: 'b', reason: 'node4 collector node_stopped', scope: 'node', nodeId: 'node4', source: 'collector', code: 'node_stopped' }] },
  series: [{ nodeId: 'node1', name: 'chain_head_block', metric: 'block_height', canonical: false, source: 'metrics', unit: 'blocks', samples: [{ time: 'a', value: 7 }, { time: 'b', value: null }] }],
 }
 assert.equal(decodeMetrics(metrics), metrics)
})

test('decoders name the first field that breaks the model', () => {
 const check = (decode, value, path) => assert.throws(() => decode(value), err => err instanceof FormatError && err.path === path, path)
 check(decodeJob, job({ phases: [{ name: 'x', state: 'running' }] }), 'phases[0].startedAt')
 check(decodeJob, job({ runIds: 'web:x' }), 'runIds')
 check(decodeMetrics, { networkId: 'n', series: [], coverage: { from: 'a', to: 'b', sampleIntervalSeconds: 5, complete: true, downsampled: false, gaps: [{ from: 'a', to: 'b', reason: 'collector not_collected' }] } }, 'coverage.gaps[0].scope')
 check(decodeMetrics, { networkId: 'n', series: [{ nodeId: 'node1', name: 'x', metric: 'x', canonical: true, source: 'rpc', samples: [{ time: 'a', value: Infinity }] }], coverage: { from: 'a', to: 'b', sampleIntervalSeconds: 5, complete: true, downsampled: false, gaps: [] } }, 'series[0].samples[0].value')
 check(decodeRun, { id: 'r', workspaceId: '', chain: '', state: 'failed', sessionRefs: [], fingerprints: {}, summary: {}, artifactRefs: [], tests: [{ seq: 1, id: 't', state: 'failed', assertions: [{ assert: 'x', expected: 0, passed: false }], steps: [], postActions: [], artifacts: [] }] }, 'tests[0].assertions[0].expected')
 assert.equal(new FormatError('terminal', 'boolean', 'yes').message, '데이터 형식 오류: terminal 값은 참·거짓이어야 하는데 문자열입니다')
 assert.equal(new FormatError('tests[0].assertions[0].expected', 'string', undefined).message, '데이터 형식 오류: tests[0].assertions[0].expected 값은 문자열이어야 하는데 값이 없습니다')
 assert.equal(new FormatError('', '객체', []).message, '데이터 형식 오류: 응답 값은 객체여야 하는데 배열입니다')
})

test('optional node placement and collector fields are checked when they come', () => {
 const node = { id: 'node1', ref: 'n.node1', networkId: 'n', role: 'bp', state: 'running', pid: 1, observedAt: 'a', supportedControls: [], controlLimits: {} }
 const network = over => ({ id: 'n', workspaceId: 'n', ownership: 'owned', version: 1, nodes: [{ ...node, ...over }] })
 assert.ok(decodeNetwork(network({})))
 assert.ok(decodeNetwork({ ...network({ server: 'east', syncMode: 'full', ports: { p2p: 30303, http: 8545 }, executable: 'ready' }), collector: { intervalSeconds: 5, lastCollected: 'b' } }))
 assert.throws(() => decodeNetwork(network({ ports: { http: '8545' } })), err => err.path === 'nodes[0].ports.http')
 assert.throws(() => decodeNetwork({ ...network({}), collector: { lastCollected: 'b' } }), err => err.path === 'collector.intervalSeconds')
})

test('evidence listings and pages are checked like every other read', () => {
 assert.ok(decodeRunEvidence({ runId: 'r', tests: [{ seq: 1, id: 'fails', files: [{ name: 'node1.log', kind: 'log', bytes: 12 }] }] }))
 assert.ok(decodeEvidencePage({ name: 'node1.log', kind: 'log', offset: 0, totalLines: 1, lines: ['INFO'], nextOffset: null }))
 assert.throws(() => decodeEvidencePage({ name: 'node1.log', kind: 'log', offset: 0, totalLines: 1, lines: ['INFO'] }), err => err.path === 'nextOffset')
})

test('a refused session is announced once per response, so the page can sign in again', async () => {
 const heard = []
 const stop = onUnauthorized(() => heard.push('401'))
 await assert.rejects(apiClient(null, recorder(401, '{"code":"unauthorized","message":"Unauthorized"}').fetcher).request('jobs'))
 await assert.rejects(apiClient(null, recorder(403, '{"code":"forbidden","message":"Forbidden"}').fetcher).request('jobs'))
 assert.deepEqual(heard, ['401'])
 stop()
 await assert.rejects(apiClient(null, recorder(401, '{}').fetcher).request('jobs'))
 assert.deepEqual(heard, ['401'], 'a listener that stopped hears nothing')
})

test('a request that never reaches the server fails in the screen’s words, not the browser’s', async () => {
 const offline = async () => { throw new TypeError('Failed to fetch') }
 await assert.rejects(apiClient(null, offline).request('networks'), err => err.unreachable === true && err.message === '서버에 연결할 수 없습니다.')
 const aborted = async () => { throw new DOMException('aborted', 'AbortError') }
 await assert.rejects(apiClient(null, aborted).request('networks'), err => err.name === 'AbortError')
})
