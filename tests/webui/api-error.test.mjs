import test from 'node:test'
import assert from 'node:assert/strict'
import {errorMessage} from '../../web/src/api-error.mjs'

test('a validation reason from the server is shown with its status', () => {
 assert.equal(errorMessage(422, '{"code":"Unprocessable Entity","message":"pool.slots must be at least 1","requestId":"web-rejected"}'), '422: pool.slots must be at least 1')
})
test('listed validation errors are joined with their paths', () => {
 assert.equal(errorMessage(422, '{"code":"Unprocessable Entity","message":"Unprocessable Entity","errors":[{"path":"steps[0].on","message":"unknown node"},{"message":"bad"}]}'), '422: steps[0].on: unknown node; bad')
})
test('a generic status text is not repeated as if it were a reason', () => {
 assert.equal(errorMessage(403, '{"code":"Forbidden","message":"Forbidden","requestId":"web-rejected"}'), '403: Forbidden')
 assert.equal(errorMessage(500, ''), '500: request failed')
})
test('plain text bodies are kept and trimmed', () => {
 assert.equal(errorMessage(409, 'revision changed\n'), '409: revision changed')
})
test('a structured message is not printed as an object', () => {
 assert.equal(errorMessage(422, '{"code":"Unprocessable Entity","message":{"nested":true}}'), '422: Unprocessable Entity')
})
test('field errors name the part of the request that failed', () => {
  assert.equal(errorMessage(422, '{"code":"invalid","message":"serverset: no pool hosts","requestId":"a1","fieldErrors":[{"path":"/content","code":"invalid","message":"serverset: no pool hosts"}]}'), '422: /content: serverset: no pool hosts')
})
test('a refused response keeps its status and structured details for the screen', async () => {
 const { responseError } = await import('../../web/src/api-error.mjs')
 const error = await responseError(new Response('{"message":"bind your SSH credential","details":{"missingAccess":["ssh:server2"]}}', { status: 403 }))
 assert.equal(error.status, 403)
 assert.deepEqual(error.details, { missingAccess: ['ssh:server2'] })
 const plain = await responseError(new Response('down', { status: 502 }))
 assert.equal(plain.status, 502)
 assert.deepEqual(plain.details, {})
})
