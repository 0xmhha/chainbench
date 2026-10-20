// Inject a failure only after a real CLI-backed network has initialized its DBs.
import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import path from 'node:path'
import assert from 'node:assert/strict'

const [fixturePath, output] = process.argv.slice(2)
const f = JSON.parse(fs.readFileSync(fixturePath, 'utf8'))
const owned = await launchOwnedBrowser()
const context = await owned.browser.newContext()
const sentinel = 'DELIBERATE_STORAGE_FAILURE_AFTER_NATIVE_SETUP'
let session
async function api(route, method = 'GET', data, expected = 200, key) {
  const response = await context.request.fetch(f.url + '/api/v1/' + route, {
    method, headers: {'Content-Type': 'application/json',
      ...(session ? {'X-CSRF-Token': session.csrfToken} : {}),
      ...(key ? {'Idempotency-Key': key} : {})},
    data: data === undefined ? undefined : JSON.stringify(data)
  })
  const text = await response.text()
  assert.equal(response.status(), expected, route + ': ' + text)
  assert.ok(!text.includes(f.password) && !text.includes(f.setupToken))
  return text ? JSON.parse(text) : undefined
}
function initializedDatabases(directory) {
  let count = 0
  for (const entry of fs.readdirSync(directory, {withFileTypes: true})) {
    if (!entry.isDirectory()) continue
    const name = path.join(directory, entry.name)
    if (entry.name === 'chaindata' && fs.existsSync(path.join(name, 'CURRENT'))) count++
    else count += initializedDatabases(name)
  }
  return count
}
async function waitJob(id) {
  for (let i = 0; i < 600; i++) {
    const job = await api('jobs/' + id)
    if (['succeeded', 'failed', 'cancelled', 'interrupted'].includes(job.state)) {
      assert.equal(job.state, 'succeeded', JSON.stringify(job))
      return job
    }
    await new Promise(resolve => setTimeout(resolve, 200))
  }
  throw new Error('native storage fixture job timed out')
}
try {
  session = await api('bootstrap', 'POST', {username: 'storage-admin', password: f.password, setupToken: f.setupToken}, 201)
  const servers = await api('documents', 'POST', {kind: 'server-set', name: 'storage fixture', contractVersion: '2', assetRefs: [],
    content: {version: 2, pool: {hosts: [{name: 'local', addr: '127.0.0.1'}], slots: 4,
      ports: {p2p: {base: 34000, step: 10}, rpc: {base: 9600, step: 10}}}}}, 201)
  const config = await api('documents', 'POST', {kind: 'workspace-config', name: 'storage paths', contractVersion: '2', assetRefs: [],
    content: {version: 1, dataRoot: f.runtime + '/data-wbft',
      paths: Object.fromEntries(['binaries', 'configs', 'genesis', 'keystore', 'keyrings', 'nodes', 'runtime', 'logs'].map(key => [key, key])),
      control: {artifactRoot: f.runtime + '/artifacts'}, inputs: {mode: 'generated'}, execution: {chain: 'fresh'}}}, 201)
  const workspace = await api('workspaces', 'POST', {name: 'storage failure fixture',
    documents: [servers, config].map(({id, revision}) => ({id, revision}))}, 201)
  const plan = await api('plans', 'POST', {workspaceId: workspace.id, operation: 'chain.setup',
    documentRefs: workspace.documents, assetRefs: ['wbft'], retention: 'retain',
    arguments: {manifestId: 'wbft', assetId: 'wbft', serverRef: 'local', validators: 4}}, 201)
  const accepted = await api('jobs', 'POST', {planId: plan.id}, 202, plan.id)
  const terminal = await waitJob(accepted.id)
  // The public node-control contract accepts exactly one owned node per plan.
  const startJobs = []
  for (const nodeId of ['node1', 'node2', 'node3', 'node4']) {
    const startPlan = await api('plans', 'POST', {workspaceId: workspace.id, operation: 'node.start',
      documentRefs: workspace.documents, assetRefs: ['wbft'], nodeIds: [nodeId], retention: 'retain',
      arguments: {manifestId: 'wbft', assetId: 'wbft', serverRef: 'local', validators: 4}}, 201)
    const started = await api('jobs', 'POST', {planId: startPlan.id}, 202, startPlan.id)
    await waitJob(started.id)
    startJobs.push(started.id)
  }
  const networks = await api('networks')
  assert.equal(networks.items.length, 1)
  assert.equal(networks.items[0].nodes.filter(node => node.pid > 0).length, 4)
  const databases = initializedDatabases(f.runtime)
  assert.equal(databases, 4)
  fs.writeFileSync(output + '/partial.json', JSON.stringify({runtime: f.runtime, outcome: 'incomplete',
    expectedFailure: sentinel, initializedNodeDatabases: databases, successfulSetupJob: terminal.id,
    successfulStartJobs: startJobs, runningNodesBeforeFailure: 4}, null, 2))
  throw new Error(sentinel)
} finally {
  await owned.stop()
}
