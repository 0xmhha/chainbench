// WEB-05 browser proof: the structured DSL editor edits every builtin argument,
// imports and round-trips the current corpus, and saved cases that cover every
// argument path run through a Web test job on a real network.
import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import path from 'node:path'
import assert from 'node:assert/strict'
import {variants, initial} from '../../web/src/dsl-form.js'

const [fixtureFile, out] = process.argv.slice(2)
const f = JSON.parse(fs.readFileSync(fixtureFile, 'utf8'))
const owned = await launchOwnedBrowser()
const context = await owned.browser.newContext({viewport: {width: 1400, height: 1100}, acceptDownloads: true})
const sleep = ms => new Promise(r => setTimeout(r, ms))
const records = [], scenarios = [], coverage = []
let session

async function api(endpoint, method = 'GET', data, expected = 200) {
  const r = await context.request.fetch(f.url + '/api/v1/' + endpoint, {method, headers: {'Content-Type': 'application/json', ...(session ? {'X-CSRF-Token': session.csrfToken} : {})}, data: data === undefined ? undefined : JSON.stringify(data)})
  const text = await r.text()
  records.push({endpoint, method, status: r.status()})
  assert.equal(r.status(), expected, `${method} ${endpoint}: ${text}`)
  assert.ok(!text.includes(f.password) && !text.includes(f.setupToken), 'bootstrap secret leaked from ' + endpoint)
  return text ? JSON.parse(text) : undefined
}
const observed = (id, messages) => scenarios.push({id, mode: 'team', transport: 'local', ownership: 'owned', role: 'admin', observedAt: new Date().toISOString(), assertions: messages.map(message => ({message, passed: true}))})
const readJSON = p => JSON.parse(fs.readFileSync(path.join(f.source, p), 'utf8'))

try {
  session = await api('bootstrap', 'POST', {username: 'dsl-admin', password: f.password, setupToken: f.setupToken}, 201)
  const vocabulary = await api('vocabulary'), contract = await api('contracts/dsl')
  fs.writeFileSync(path.join(out, 'contract.json'), JSON.stringify({vocabulary, contract}, null, 2))
  const page = await context.newPage(), errors = []
  page.on('pageerror', e => errors.push(e.message))
  await page.goto(f.url + '/tests')
  const status = page.getByLabel('DSL status', {exact: true})
  await status.filter({hasText: 'DSL contract loaded'}).waitFor({timeout: 30000})
  const idle = () => status.filter({hasText: 'Working…'}).waitFor({state: 'hidden'})
  async function upload(doc, name = 'case.json') {
    const pending = page.waitForResponse(r => r.url().endsWith('/test-cases/import'))
    await page.getByLabel('Import test JSON', {exact: true}).setInputFiles({name, mimeType: 'application/json', buffer: Buffer.from(JSON.stringify(doc))})
    const response = await pending
    await idle()
    return response
  }
  async function exportDoc(name) {
    const pending = page.waitForEvent('download')
    await page.getByRole('button', {name: 'Export test scenario', exact: true}).click()
    const download = await pending
    await download.saveAs(path.join(out, name))
    return JSON.parse(fs.readFileSync(path.join(out, name), 'utf8'))
  }

  // Node selectors and variable references, edited in the browser.
  const base = {schemaVersion: '2', kind: 'case', id: 'browser', chainPreset: {chain: 'stablenet', binaries: {default: 'gstable'}, topology: {bp: 4}}, steps: [{do: 'read', source: 'blockNumber', on: 'node1', save: 'head'}, {expect: 'blockNumber', onEach: ['node1', 'node2'], compare: 'GreaterOrEqual', is: '$head'}, {expect: 'chainId', is: '8283'}]}
  assert.equal((await upload(base)).status(), 200)
  const before = (await api('test-cases/import', 'POST', {content: base})).semanticFingerprint
  await page.getByLabel('/content/id', {exact: true}).fill('browser-edited')
  await page.getByLabel('/content/steps/0/on', {exact: true}).fill('bp1')
  assert.equal(await page.getByRole('button', {name: 'Export test scenario', exact: true}).isDisabled(), true)
  await page.getByRole('button', {name: 'Validate test scenario', exact: true}).click()
  await status.filter({hasText: 'Engine validation passed'}).waitFor()
  const exported = await exportDoc('edited.json')
  assert.equal(exported.id, 'browser-edited'); assert.equal(exported.steps[0].on, 'bp1')
  assert.deepEqual(exported.steps.slice(1), base.steps.slice(1))
  const first = await api('test-cases/import', 'POST', {content: exported}), second = await api('test-cases/import', 'POST', {content: first.content})
  assert.equal(first.semanticFingerprint, second.semanticFingerprint); assert.notEqual(first.semanticFingerprint, before)
  await page.screenshot({path: path.join(out, 'editor.png'), fullPage: true})
  observed('references', ['Node selector and preceding variable reference edited in the browser', 'Export re-import preserves the executable fingerprint'])

  // v1 import: migrated when the meaning survives, retained and editable when not.
  const legacy = {schemaVersion: '1', id: 'legacy-browser', chain: {name: 'stablenet', binary: 'gstable'}, topology: {bp: 4}, steps: [{read: {source: 'blockNumber', on: 'bp1', save: 'head'}}], assertions: [{assert: 'blockNumber', compare: 'GreaterOrEqual', expected: '$head'}]}
  assert.equal((await upload(legacy)).status(), 200)
  await status.filter({hasText: 'v1 migrated; executable meaning preserved'}).waitFor()
  const migrated = await exportDoc('migrated.json'), migration = await api('test-cases/import', 'POST', {content: legacy})
  assert.equal((await api('test-cases/import', 'POST', {content: migrated})).semanticFingerprint, migration.semanticFingerprint)
  const retained = {...legacy, id: 'legacy-config-browser', chain: {...legacy.chain, config: 'site.toml'}}
  assert.equal((await upload(retained)).status(), 200)
  await status.filter({hasText: 'Retained v1:'}).waitFor()
  await page.getByLabel('/content/chain/config', {exact: true}).fill('edited-site.toml')
  await page.getByRole('button', {name: 'Validate test scenario', exact: true}).click()
  await status.filter({hasText: 'Engine validation passed'}).waitFor()
  const retainedExport = await exportDoc('retained-v1.json')
  assert.equal(retainedExport.schemaVersion, '1'); assert.equal(retainedExport.chain.config, 'edited-site.toml')
  const retainedPrepared = await api('test-cases/import', 'POST', {content: retainedExport})
  assert.equal(retainedPrepared.migrated, false); assert.ok(retainedPrepared.migrationMessage)
  observed('v1-migration', ['Browser migrated v1 through the engine migration', 'Migrated v2 fingerprint equals the v1 projection', 'Unrepresentable v1 declaration edited and round-tripped without losing fields'])

  // Unknown fields, builtins, readers, references and ignored arguments are refused by name.
  const refusals = []
  for (const [invalid, word] of [
    [{...base, extension: true}, 'extension'],
    [{...base, steps: [{expect: 'unknown', is: 1}]}, 'unknown'],
    [{...base, steps: [{expect: 'blockNumber', is: '$unbound'}]}, 'unbound'],
    [{...base, steps: [{do: 'read', source: 'unknown'}, {expect: 'blockNumber', is: 1}]}, 'unknown'],
    [{...base, steps: [{expect: 'chainId', is: '8283', timeout: '5s'}]}, 'timeout'],
    [{...base, steps: [{do: 'restartNode', on: 'bp1', expectFail: true}, {expect: 'chainId', is: '8283'}]}, 'expectFail'],
    [{...legacy, extension: true}, 'extension'],
  ]) {
    const result = await api('test-cases/import', 'POST', {content: invalid}, 422)
    const text = JSON.stringify(result)
    assert.ok(text.includes(word), `refusal does not name ${word}: ${text}`)
    refusals.push(word)
  }
  assert.equal((await upload({...base, extension: true})).status(), 422)
  assert.equal(await page.getByRole('button', {name: 'Export test scenario', exact: true}).isDisabled(), true)
  await page.getByText('extension', {exact: false}).first().waitFor()
  observed('invalid-unknown', ['Unknown fields, builtins, readers and unbound references refused by name', 'Arguments a builtin never reads refused by name', 'An invalid browser import blocks export'])

  // Every case in the corpus imports, and its prepared form re-imports unchanged.
  const presets = {}
  for (const file of f.presetFiles) { const p = readJSON(file); presets[p.id] = p }
  let imported = 0
  for (const file of f.corpusFiles) {
    const content = readJSON(file)
    const prepared = await api('test-cases/import', 'POST', {content, presets})
    const again = await api('test-cases/import', 'POST', {content: prepared.content, presets})
    assert.equal(again.semanticFingerprint, prepared.semanticFingerprint, file + ' changed meaning on re-import')
    imported++
  }
  observed('grammar', [`${imported} corpus cases imported and re-imported with unchanged executable meaning`])

  // Every registration's arguments are edited through the form, each read
  // source separately so its own arguments appear.
  const edited = {}
  for (const entry of vocabulary.entries) {
    const key = entry.kind + ':' + entry.name, seen = new Set()
    for (const schema of variants({$ref: entry.schemaRef}, contract)) {
      const statement = {}
      for (const [name, field] of Object.entries(schema.properties ?? {})) statement[name] = initial(field, contract)
      if (statement.on !== undefined) statement.on = 'bp1'
      if (statement.onEach !== undefined) statement.onEach = ['bp1']
      if (!statement.on && !statement.onEach && ['stopNode', 'startNode', 'restartNode', 'resetNode', 'swapNode', 'readNodeLog'].includes(entry.name)) statement.on = 'bp1'
      if (statement.save !== undefined) statement.save = 'sample'
      if (entry.kind === 'reader') { statement.do = 'read'; statement.source = entry.name }
      const doc = {...base, id: `coverage-${entry.kind}-${entry.name}`, steps: [statement, {expect: 'blockNumber', is: 1}]}
      await upload(doc)
      const fields = Object.keys(schema.properties ?? {}).filter(k => !['do', 'source', 'isPerChain', 'expectPerChain'].includes(k) && !(k === 'expect' && schema.properties[k].const !== undefined))
      for (const field of fields) {
        const locator = page.locator(`[data-field-path="/content/steps/0/${field}"]`).first()
        assert.equal(await locator.count(), 1, `${key} missing field ${field}`)
        const control = locator.locator('input,select,button').first()
        if (await control.count()) {
          const [tag, type] = await control.evaluate(e => [e.tagName, e.type])
          if (tag === 'INPUT' && type === 'checkbox') { await control.check(); await control.uncheck() }
          else if (tag === 'INPUT') { const val = await control.inputValue(); await control.fill(type === 'number' ? '1' : (val || 'sample')) }
          else if (tag === 'SELECT') await control.selectOption(await control.inputValue())
          else if (tag === 'BUTTON') await control.click()
        }
        seen.add(field)
      }
    }
    edited[key] = [...seen].sort()
    coverage.push({kind: entry.kind, name: entry.name, editedArgumentPaths: edited[key]})
  }
  assert.deepEqual(errors, [])
  fs.writeFileSync(path.join(out, 'edited.json'), JSON.stringify(edited, null, 2))
  observed('arguments-edited', [`${vocabulary.entries.length} registrations rendered and edited field by field`])

  // The coverage cases: imported with their preset, round-tripped, saved with
  // the preset revision pinned.
  await page.getByLabel('Referenced preset files', {exact: true}).setInputFiles(f.presetFiles.filter(p => p.endsWith('stablenet-bp4-en1.json')).map(p => ({name: path.basename(p), mimeType: 'application/json', buffer: fs.readFileSync(path.join(f.source, p))})))
  await status.filter({hasText: 'Preset references loaded'}).waitFor()
  const saved = []
  for (const file of f.coverageFiles) {
    const content = readJSON(file)
    assert.equal((await upload(content, path.basename(file))).status(), 200, file)
    await status.filter({hasText: 'Imported; engine validation passed'}).waitFor()
    const exportedCase = await exportDoc('roundtrip-' + path.basename(file))
    const a = await api('test-cases/import', 'POST', {content, presets}), b = await api('test-cases/import', 'POST', {content: exportedCase, presets})
    assert.equal(a.semanticFingerprint, b.semanticFingerprint, file + ' changed meaning through the editor')
    await page.getByRole('button', {name: '공유 테스트 저장', exact: true}).click()
    await status.filter({hasText: '저장됨'}).waitFor({timeout: 20000})
    await page.getByTestId('dsl-preset-refs').filter({hasText: 'revision'}).waitFor()
    saved.push(content.id)
  }
  // Selected in file order, so the case that stops and swaps nodes runs last.
  const listed = (await api('documents?kind=case')).items
  const cases = saved.map(id => listed.find(d => d.content.id === id))
  assert.ok(cases.every(Boolean) && cases.length === saved.length, 'a saved coverage case is missing')
  assert.ok(cases.every(d => d.presetRefs?.length === 1), 'a saved case lost its pinned preset')
  observed('round-trip', [`${saved.length} coverage cases imported, exported and saved with pinned preset revisions and unchanged meaning`])

  // A Web test job runs every saved coverage case on a real network.
  const set = await api('documents', 'POST', {kind: 'server-set', name: 'web05 pool', contractVersion: '2', assetRefs: [], content: {version: 2, pool: {hosts: [{name: 'local', addr: '127.0.0.1'}], slots: 5, ports: {p2p: {base: f.p2pBase, step: 10}, rpc: {base: f.rpcBase, step: 10}}}}}, 201)
  const config = await api('documents', 'POST', {kind: 'workspace-config', name: 'web05 paths', contractVersion: '2', assetRefs: [], content: {version: 1, dataRoot: f.runtime + '/d', paths: Object.fromEntries(['binaries', 'configs', 'genesis', 'keystore', 'keyrings', 'nodes', 'runtime', 'logs'].map(k => [k, k])), control: {artifactRoot: f.runtime + '/output'}, inputs: {mode: 'generated'}, execution: {chain: 'fresh'}, limits: {minFreeDisk: '0'}}}, 201)
  const workspace = await api('workspaces', 'POST', {name: 'web05 coverage', documents: [{id: set.id, revision: set.revision}, {id: config.id, revision: config.revision}]}, 201)
  await page.reload()
  await page.getByLabel('작업 Workspace', {exact: true}).selectOption(workspace.id)
  await page.getByLabel('작업 매니페스트', {exact: true}).selectOption('stablenet')
  await page.getByLabel('작업 바이너리', {exact: true}).selectOption('stablenet')
  await page.getByLabel('작업 서버 이름', {exact: true}).selectOption('local')
  // Cleaned up when it ends: the attach target below runs the same gstable
  // build, which the engine refuses to run twice on one machine.
  await page.getByLabel('작업 종료 후 처리', {exact: true}).selectOption('cleanup')
  for (const [i, doc] of cases.entries()) {
    await page.getByLabel('실행 케이스 ' + doc.id, {exact: true}).check()
    await page.getByTestId('case-order-' + doc.id).filter({hasText: String(i + 1)}).waitFor()
  }
  await page.getByRole('button', {name: '실행 계획 확인', exact: true}).click()
  await page.getByLabel('실행 계획', {exact: true}).waitFor({timeout: 60000})
  await page.getByLabel('서버 실행 작업', {exact: true}).screenshot({path: path.join(out, 'test-plan.png')})
  const started = page.waitForResponse(r => r.url() === f.url + '/api/v1/jobs' && r.request().method() === 'POST')
  await page.getByRole('button', {name: '검토한 계획 실행', exact: true}).click()
  const job = await (await started).json()
  let done
  for (let i = 0; i < 18000; i++) { done = await api('jobs/' + job.id); if (['succeeded', 'failed', 'cancelled', 'interrupted'].includes(done.state)) break; await sleep(200) }
  fs.writeFileSync(path.join(out, 'job.json'), JSON.stringify(done, null, 2))
  assert.equal(done.state, 'succeeded', JSON.stringify(done))
  assert.equal(done.nodeDisposition, 'cleaned', 'the coverage network was not cleaned up: ' + done.nodeDisposition)
  await page.screenshot({path: path.join(out, 'test-results.png'), fullPage: true})
  // The fork case runs a second build under the name "next": the job binds it
  // to the registered wbft asset, and the network crosses from gwemix to it.
  await page.goto(f.url + '/tests')
  await status.filter({hasText: 'DSL contract loaded'}).waitFor({timeout: 30000})
  await page.getByLabel('Referenced preset files', {exact: true}).setInputFiles([{name: path.basename(f.forkPreset), mimeType: 'application/json', buffer: fs.readFileSync(path.join(f.source, f.forkPreset))}])
  await status.filter({hasText: 'Preset references loaded'}).waitFor()
  const forkContent = readJSON(f.forkCase)
  assert.equal((await upload(forkContent, path.basename(f.forkCase))).status(), 200)
  await status.filter({hasText: 'Imported; engine validation passed'}).waitFor()
  await page.getByRole('button', {name: '공유 테스트 저장', exact: true}).click()
  await status.filter({hasText: '저장됨'}).waitFor({timeout: 20000})
  const forkDoc = (await api('documents?kind=case')).items.find(d => d.content.id === forkContent.id)
  assert.equal(forkDoc.presetRefs?.length, 1, 'fork case lost its pinned preset')
  const forkSet = await api('documents', 'POST', {kind: 'server-set', name: 'web05 fork pool', contractVersion: '2', assetRefs: [], content: {version: 2, pool: {hosts: [{name: 'local', addr: '127.0.0.1'}], slots: 5, ports: {p2p: {base: f.forkP2PBase, step: 10}, rpc: {base: f.forkRPCBase, step: 10}}}}}, 201)
  const forkConfig = await api('documents', 'POST', {kind: 'workspace-config', name: 'web05 fork paths', contractVersion: '2', assetRefs: [], content: {version: 1, dataRoot: f.runtime + '/fork', paths: Object.fromEntries(['binaries', 'configs', 'genesis', 'keystore', 'keyrings', 'nodes', 'runtime', 'logs'].map(k => [k, k])), control: {artifactRoot: f.runtime + '/fork-output'}, inputs: {mode: 'generated'}, execution: {chain: 'fresh'}, limits: {minFreeDisk: '0'}}}, 201)
  const forkWorkspace = await api('workspaces', 'POST', {name: 'web05 fork', documents: [{id: forkSet.id, revision: forkSet.revision}, {id: forkConfig.id, revision: forkConfig.revision}]}, 201)
  await page.reload()
  await page.getByLabel('작업 Workspace', {exact: true}).selectOption(forkWorkspace.id)
  await page.getByLabel('작업 매니페스트', {exact: true}).selectOption('wemix')
  await page.getByLabel('작업 바이너리', {exact: true}).selectOption('wemix')
  await page.getByLabel('작업 서버 이름', {exact: true}).selectOption('local')
  await page.getByLabel('실행 케이스 ' + forkDoc.id, {exact: true}).check()
  await page.getByLabel('테스트 바이너리 next', {exact: true}).selectOption('wbft')
  await page.getByRole('button', {name: '실행 계획 확인', exact: true}).click()
  const forkPlan = page.getByLabel('실행 계획', {exact: true})
  await forkPlan.waitFor({timeout: 60000})
  await forkPlan.getByText('Binary "next" runs wbft', {exact: false}).waitFor()
  await page.getByLabel('서버 실행 작업', {exact: true}).screenshot({path: path.join(out, 'fork-plan.png')})
  const forkStarted = page.waitForResponse(r => r.url() === f.url + '/api/v1/jobs' && r.request().method() === 'POST')
  await page.getByRole('button', {name: '검토한 계획 실행', exact: true}).click()
  const forkJob = await (await forkStarted).json()
  let forkDone
  for (let i = 0; i < 9000; i++) { forkDone = await api('jobs/' + forkJob.id); if (['succeeded', 'failed', 'cancelled', 'interrupted'].includes(forkDone.state)) break; await sleep(200) }
  fs.writeFileSync(path.join(out, 'fork-job.json'), JSON.stringify(forkDone, null, 2))
  assert.equal(forkDone.state, 'succeeded', JSON.stringify(forkDone))
  assert.ok(forkDone.partialEffects.some(v => v.includes('"next" provisioned')), 'the successor build was not provisioned')
  observed('live-execution', [`Web test job ${done.id} ran ${cases.length} saved cases on native gstable and succeeded`, `Web test job ${forkDone.id} crossed the croissant fork from gwemix to the registered wbft build and succeeded`])

  // Attach cases run only against a network this service composed and kept:
  // a test job brings one up in its own workspace, then an attach job runs the
  // corpus attach cases against its recorded endpoints and key set.
  const shared = (await api('documents?kind=chain-preset')).items.find(d => d.content.id === 'stablenet-bp4-en1')
  assert.ok(shared, 'the shared stablenet-bp4-en1 preset is missing')
  // The target declares accounts: a composed Web run mints and funds them in
  // memory over the read-only accepted key set.
  const declared = readJSON(f.declaredAccountsCase)
  const target = await api('documents', 'POST', {kind: 'case', name: declared.id, contractVersion: '2', assetRefs: [], presetRefs: [{id: shared.id, revision: shared.revision}], content: declared}, 201)
  await page.goto(f.url + '/tests')
  await status.filter({hasText: 'DSL contract loaded'}).waitFor({timeout: 30000})
  await page.getByLabel('Referenced preset files', {exact: true}).setInputFiles(f.presetFiles.filter(p => /stablenet-(attached|testnet)/.test(p)).map(p => ({name: path.basename(p), mimeType: 'application/json', buffer: fs.readFileSync(path.join(f.source, p))})))
  await status.filter({hasText: 'Preset references loaded'}).waitFor()
  const attachIds = []
  for (const file of f.attachFiles) {
    const content = readJSON(file)
    assert.equal((await upload(content, path.basename(file))).status(), 200, file)
    await status.filter({hasText: 'Imported; engine validation passed'}).waitFor()
    await page.getByRole('button', {name: '공유 테스트 저장', exact: true}).click()
    await status.filter({hasText: '저장됨'}).waitFor({timeout: 20000})
    attachIds.push(content.id)
  }
  const attachDocs = (await api('documents?kind=case')).items.filter(d => attachIds.includes(d.content.id))
  assert.equal(attachDocs.length, attachIds.length, 'an attach case was not saved')
  const attachSet = await api('documents', 'POST', {kind: 'server-set', name: 'web05 attach pool', contractVersion: '2', assetRefs: [], content: {version: 2, pool: {hosts: [{name: 'local', addr: '127.0.0.1'}], slots: 5, ports: {p2p: {base: f.attachP2PBase, step: 10}, rpc: {base: f.attachRPCBase, step: 10}}}}}, 201)
  const attachConfig = await api('documents', 'POST', {kind: 'workspace-config', name: 'web05 attach paths', contractVersion: '2', assetRefs: [], content: {version: 1, dataRoot: f.runtime + '/attach', paths: Object.fromEntries(['binaries', 'configs', 'genesis', 'keystore', 'keyrings', 'nodes', 'runtime', 'logs'].map(k => [k, k])), control: {artifactRoot: f.runtime + '/attach-output'}, inputs: {mode: 'generated'}, execution: {chain: 'fresh'}, limits: {minFreeDisk: '0'}}}, 201)
  const attachWorkspace = await api('workspaces', 'POST', {name: 'web05 attach', documents: [{id: attachSet.id, revision: attachSet.revision}, {id: attachConfig.id, revision: attachConfig.revision}]}, 201)
  async function runJob(label, choose, budget) {
    await page.reload()
    await page.getByLabel('작업 Workspace', {exact: true}).selectOption(attachWorkspace.id)
    await page.getByLabel('작업 매니페스트', {exact: true}).selectOption('stablenet')
    await page.getByLabel('작업 바이너리', {exact: true}).selectOption('stablenet')
    await page.getByLabel('작업 서버 이름', {exact: true}).selectOption('local')
    await choose()
    await page.getByRole('button', {name: '실행 계획 확인', exact: true}).click()
    await page.getByLabel('실행 계획', {exact: true}).waitFor({timeout: 60000})
    await page.getByLabel('서버 실행 작업', {exact: true}).screenshot({path: path.join(out, label + '-plan.png')})
    const begun = page.waitForResponse(r => r.url() === f.url + '/api/v1/jobs' && r.request().method() === 'POST')
    await page.getByRole('button', {name: '검토한 계획 실행', exact: true}).click()
    const accepted = await (await begun).json()
    let result
    for (let i = 0; i < budget; i++) { result = await api('jobs/' + accepted.id); if (['succeeded', 'failed', 'cancelled', 'interrupted'].includes(result.state)) break; await sleep(200) }
    fs.writeFileSync(path.join(out, label + '-job.json'), JSON.stringify(result, null, 2))
    assert.equal(result.state, 'succeeded', JSON.stringify(result))
    return result
  }
  const targetDone = await runJob('attach-target', async () => {
    await page.getByLabel('작업 종류', {exact: true}).selectOption('test.run')
    await page.getByLabel('실행 케이스 ' + target.id, {exact: true}).check()
  }, 9000)
  // A case naming a key file is refused until the caller binds one of their
  // own account keys; the key never travels in the case or the plan.
  for (const doc of attachDocs.filter(d => f.attachCredentialCases.includes(d.content.id))) {
    const refused = await api('plans', 'POST', {workspaceId: attachWorkspace.id, operation: 'test.attach', documentRefs: attachWorkspace.documents, assetRefs: ['stablenet'], credentialBindings: {}, nodeIds: [], retention: 'retain',
      arguments: {manifestId: 'stablenet', assetId: 'stablenet', serverRef: 'local', caseRefs: [{id: doc.id, revision: doc.revision}]}}, 422)
    assert.ok(JSON.stringify(refused).includes('private account credential'), 'key file refusal does not explain the credential: ' + JSON.stringify(refused))
  }
  await page.goto(f.url + '/chains')
  await page.getByLabel('Credential label', {exact: true}).fill('payer key')
  await page.getByLabel('Credential kind', {exact: true}).selectOption('account-key')
  await page.getByLabel('Account private key', {exact: true}).fill(fs.readFileSync(f.payerKeyFile, 'utf8').trim())
  await page.getByRole('button', {name: 'Save personal credential', exact: true}).click()
  await page.getByText('Saved encrypted personal credential', {exact: false}).waitFor()
  const payer = (await api('credentials')).items.find(c => c.kind === 'account-key' && c.label === 'payer key')
  assert.ok(payer, 'the account key was not saved')
  const attachDone = await runJob('attach', async () => {
    await page.getByLabel('작업 종류', {exact: true}).selectOption('test.attach')
    await page.getByTestId('attach-explanation').waitFor()
    assert.equal(await page.getByLabel('실행 케이스 ' + target.id, {exact: true}).count(), 0, 'a composing case is offered to an attach job')
    for (const doc of attachDocs) await page.getByLabel('실행 케이스 ' + doc.id, {exact: true}).check()
    await page.getByLabel('테스트 계정 키 payer', {exact: true}).selectOption(payer.id)
  }, 9000)
  const plan = JSON.stringify(await api('jobs/' + attachDone.id))
  assert.ok(!plan.includes(fs.readFileSync(f.payerKeyFile, 'utf8').trim().replace(/^0x/, '')), 'the account key appears in the job record')
  observed('attach-execution', [`Web test job ${targetDone.id} minted and funded declared accounts on a composed stablenet network and kept it`, `Web attach job ${attachDone.id} ran ${attachDocs.length} corpus attach cases against its recorded endpoints and key set and succeeded`, 'A key file account is refused until the caller binds a private account key, which then signs without appearing in the job'])
  fs.writeFileSync(path.join(out, 'browser.json'), JSON.stringify({browserVersion: owned.browser.version(), scenarios, coverage, savedCases: cases.map(d => ({id: d.id, caseId: d.content.id, revision: d.revision, presetRefs: d.presetRefs})), job: {id: done.id, state: done.state, runIds: [...done.runIds, ...forkDone.runIds]}, forkJob: {id: forkDone.id, state: forkDone.state, runIds: forkDone.runIds}, attachJob: {id: attachDone.id, state: attachDone.state, runIds: attachDone.runIds, targetRunIds: targetDone.runIds}, corpusImported: imported, refusals, seedAcceptanceAwarded: false}, null, 2))
  fs.writeFileSync(path.join(out, 'api-observations.json'), JSON.stringify(records, null, 2))
} finally { await owned.stop() }
