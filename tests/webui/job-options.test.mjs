import test from 'node:test'
import assert from 'node:assert/strict'
import { scalarOptions, optionArguments } from '../../web/src/job-options.mjs'

const run = { id: 'test.run', arguments: [
 { name: 'serverRef', type: 'string', title: '서버', cli: '' },
 { name: 'noSkips', type: 'boolean', title: '건너뛴 케이스를 실패로', cli: '' },
 { name: 'waitBlocks', type: 'integer', title: '시작 전 대기 높이', minimum: 1, maximum: 1000000, cli: '' },
 { name: 'nodeMonitorTimeoutSeconds', type: 'integer', title: '노드 준비 대기(초)', minimum: 1, maximum: 3600, cli: '' },
] }

test('the run options are the yes-or-no and number arguments the contract lists for the operation', () => {
 assert.deepEqual(scalarOptions(run).map(a => a.name), ['noSkips', 'waitBlocks', 'nodeMonitorTimeoutSeconds'])
 assert.deepEqual(scalarOptions(undefined), [])
})

test('only options the user set reach the plan, as the types the engine reads', () => {
 const options = scalarOptions(run)
 assert.deepEqual(optionArguments(options, {}), {})
 assert.deepEqual(optionArguments(options, { noSkips: false, waitBlocks: '', nodeMonitorTimeoutSeconds: null }), {})
 assert.deepEqual(optionArguments(options, { noSkips: true, waitBlocks: '12', nodeMonitorTimeoutSeconds: 300 }), { noSkips: true, waitBlocks: 12, nodeMonitorTimeoutSeconds: 300 })
})

test('a number outside the contract range is named rather than sent', () => {
 assert.throws(() => optionArguments(scalarOptions(run), { waitBlocks: 0 }), /시작 전 대기 높이.*1.*1000000/)
 assert.throws(() => optionArguments(scalarOptions(run), { nodeMonitorTimeoutSeconds: 1.5 }), /노드 준비 대기/)
})
