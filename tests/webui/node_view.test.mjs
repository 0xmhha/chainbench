import test from 'node:test'
import assert from 'node:assert/strict'
import { latestSamples, nodeControls } from '../../web/src/node-view.mjs'

test('the latest sample of each series of one node, skipping gaps', () => {
 const metrics = { series: [
  { nodeId: 'node1', name: 'head', unit: 'block', source: 'rpc', samples: [{ time: 't1', value: 5 }, { time: 't2', value: 6 }, { time: 't3', value: null }] },
  { nodeId: 'node2', name: 'head', source: 'rpc', samples: [{ time: 't1', value: 9 }] },
  { nodeId: 'node1', name: 'peers', source: 'metrics', samples: [{ time: 't1', value: null }] },
 ] }
 assert.deepEqual(latestSamples(metrics, 'node1'), [{ name: 'head', unit: 'block', source: 'rpc', time: 't2', value: 6 }, { name: 'peers', unit: '', source: 'metrics', time: null, value: null }])
 assert.deepEqual(latestSamples(null, 'node1'), [])
})

test('controls follow the operations contract and carry the server reason when refused', () => {
 const operations = [{ id: 'chain.deploy', label: '배포' }, { id: 'node.start', label: '노드 시작' }, { id: 'node.stop', label: '노드 정지' }, { id: 'node.reset', label: '초기화' }]
 const node = { supportedControls: ['node.stop'], controlLimits: { 'node.start': '이미 실행 중', 'node.reset': '생산자' } }
 assert.deepEqual(nodeControls(node, operations, ''), [
  { id: 'node.start', label: '노드 시작', enabled: false, reason: '이미 실행 중' },
  { id: 'node.stop', label: '노드 정지', enabled: true, reason: '' },
  { id: 'node.reset', label: '초기화', enabled: false, reason: '생산자' },
 ])
 const busy = nodeControls(node, operations, '다른 작업이 실행 중입니다')
 assert.ok(busy.every(c => !c.enabled && c.reason))
 assert.equal(busy.find(c => c.id === 'node.reset').reason, '생산자', 'a permanent limit is not hidden by a busy network')
})
