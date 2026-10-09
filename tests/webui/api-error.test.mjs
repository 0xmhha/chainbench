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
