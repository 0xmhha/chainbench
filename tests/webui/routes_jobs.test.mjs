import test from 'node:test'
import assert from 'node:assert/strict'
import { parseRoute } from '../../web/src/routes.mjs'
import { jobSummary } from '../../web/src/job-view.mjs'

test('routes name the fixed screens and one job detail', () => {
 assert.deepEqual(parseRoute('/'), { page: '/' })
 assert.deepEqual(parseRoute('/monitoring'), { page: '/monitoring' })
 assert.deepEqual(parseRoute('/jobs/0123456789abcdef0123456789abcdef'), { page: '/jobs', jobId: '0123456789abcdef0123456789abcdef' })
 assert.deepEqual(parseRoute('/jobs/nope'), { page: '/' })
 assert.deepEqual(parseRoute('/elsewhere'), { page: '/' })
})

test('a job summary names the failed phase, its reason and what is left to check', () => {
 const failed = jobSummary({ state: 'failed', nodeDisposition: 'unknown', phases: [{ name: 'validate', state: 'succeeded' }, { name: 'launch', state: 'failed', message: 'node2 did not start' }], error: { code: 'execution_failed', message: 'launch failed' }, partialEffects: ['node1 started'], unresolvedResources: ['node2'] })
 assert.equal(failed.active, false)
 assert.deepEqual(failed.failedPhase, { name: 'launch', state: 'failed', message: 'node2 did not start' })
 assert.equal(failed.failure, 'launch failed')
 assert.deepEqual(failed.effects, ['node1 started'])
 assert.deepEqual(failed.unresolved, ['node2'])
 assert.equal(failed.needsCheck, true)
 const running = jobSummary({ state: 'running', nodeDisposition: 'unknown', phases: [{ name: 'launch', state: 'running' }] })
 assert.equal(running.active, true)
 assert.equal(running.failedPhase, null)
 assert.equal(running.needsCheck, false)
 assert.deepEqual(running.effects, [])
})
