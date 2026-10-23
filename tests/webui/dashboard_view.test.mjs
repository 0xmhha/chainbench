import test from 'node:test'
import assert from 'node:assert/strict'
import { nodeTally, attentionJobs, shortId } from '../../web/src/dashboard-view.mjs'

test('nodes are tallied as running, stopped and everything else', () => {
 assert.deepEqual(nodeTally([{ state: 'running' }, { state: 'running' }, { state: 'stopped' }, { state: 'unknown' }, {}]), { running: 2, stopped: 1, other: 2, total: 5 })
 assert.deepEqual(nodeTally(undefined), { running: 0, stopped: 0, other: 0, total: 0 })
})

test('jobs that need attention are the failed and interrupted ones, newest first, with their failed phase', () => {
 const jobs = [
  { id: 'a', state: 'succeeded', createdAt: '2026-10-23T07:00:00Z' },
  { id: 'b', state: 'failed', createdAt: '2026-10-23T07:01:00Z', phases: [{ name: 'place', state: 'succeeded' }, { name: 'start', state: 'failed', message: 'node3 exited' }] },
  { id: 'c', state: 'interrupted', createdAt: '2026-10-23T07:02:00Z', phases: [] },
  { id: 'd', state: 'running', createdAt: '2026-10-23T07:03:00Z' },
  { id: 'e', state: 'failed', createdAt: '2026-10-23T06:00:00Z', error: { message: 'plan expired' } }
 ]
 const found = attentionJobs(jobs, 2)
 assert.deepEqual(found.map(j => j.id), ['c', 'b'])
 assert.deepEqual(found[1].failedPhase, { name: 'start', state: 'failed', message: 'node3 exited' })
 assert.equal(attentionJobs(jobs, 10).find(j => j.id === 'e').reason, 'plan expired')
})

test('long ids are shortened for display only', () => {
 assert.equal(shortId('0123456789abcdef0123456789abcdef'), '01234567…')
 assert.equal(shortId('net-1'), 'net-1')
})
