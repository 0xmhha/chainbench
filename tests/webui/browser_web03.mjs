import { chromium } from 'playwright'
import fs from 'node:fs'
import path from 'node:path'
import assert from 'node:assert/strict'

const [url, out, fixturePath] = process.argv.slice(2)
const fixture = JSON.parse(fs.readFileSync(fixturePath, 'utf8'))
const observations = [], records = []
const actors = fixture.accounts
const browser = await chromium.launch({ headless: true, channel: process.env.WEBUI_BROWSER_CHANNEL ?? "chrome" })
function auth(actor) { return `Basic ${Buffer.from(`${actor.username}:${actor.password}`).toString('base64')}` }
async function api(actor, endpoint, method='GET', body, revision, expected=200) {
  const headers = { Authorization: auth(actor), 'Content-Type': 'application/json' }
  if (revision) headers['If-Match'] = `"${revision}"`
  const response = await fetch(url+'/api/v1/'+endpoint, {method, headers, body: body===undefined ? undefined : JSON.stringify(body)})
  const text = await response.text()
  assert.equal(response.status, expected, `${method} ${endpoint}: ${text}`)
  let value; try { value = JSON.parse(text) } catch { value = text }
  // Requests carrying secrets are never copied to evidence.
  records.push({actor:actor.id,method,endpoint,status:response.status,response:value,observedAt:new Date().toISOString()})
  return value
}
async function status(page, text) { try { await page.waitForFunction(t => document.querySelector('[data-testid=deployment-status]')?.textContent.includes(t), text) } catch { throw new Error(`Expected status ${text}, observed: ${await page.getByTestId('deployment-status').textContent()}`) } }
async function login(actor) {
  const context = await browser.newContext({viewport:{width:1440,height:1000},acceptDownloads:true})
  const page = await context.newPage()
  const errors=[];page.on('pageerror',e=>errors.push(e.message))
  await page.goto(url+'/chains')
  await page.getByLabel('Deployment account',{exact:true}).fill(actor.username)
  await page.getByLabel('Deployment password',{exact:true}).fill(actor.password)
  await page.getByRole('button',{name:'Sign in to deployment',exact:true}).click()
  await page.waitForFunction(() => document.querySelector('[data-testid=deployment-actor]'))
  await page.getByLabel('content.pool.hosts[0].addr',{exact:true}).waitFor()
  return {context,page,errors}
}
function observed(id, assertions, artifact) { observations.push({id,observedAt:new Date().toISOString(),mode:'team',transport:'SSH',ownership:'owned',actors:actors.map(a=>({id:a.id,role:a.role})),assertions:assertions.map(description=>({description,passed:true})),artifacts:[artifact]}) }

try {
  const first = await login(actors[0]), second = await login(actors[1])
  const a=first.page,b=second.page
  await a.getByLabel('content.pool.hosts[0].name',{exact:true}).fill('ssh')
  await a.getByLabel('content.pool.hosts[0].addr',{exact:true}).fill('localhost.')
  await a.getByRole('button',{name:'Add ssh',exact:true}).click()
  await a.getByRole('button',{name:'Add port',exact:true}).click()
  await a.getByLabel('content.ssh.port',{exact:true}).fill(String(fixture.ssh.port))
  await a.getByRole('button',{name:'Add known_hosts_file',exact:true}).click()
  await a.getByLabel('content.ssh.known_hosts_file',{exact:true}).fill(fixture.ssh.knownHosts)
  await a.getByRole('button',{name:'Validate deployment document',exact:true}).click()
  await status(a,'Engine validation passed')
  await a.getByRole('button',{name:'Save shared document',exact:true}).click()
  await status(a,'Saved shared document revision 1')
  let docs=(await api(actors[0],'documents')).items
  let set=docs.find(d=>d.kind==='server-set')
  assert.ok(set)
  await a.getByRole('button',{name:'New workspace-config',exact:true}).click()
  await a.getByLabel('content.dataRoot',{exact:true}).fill(fixture.ssh.runtime+'/allowed')
  await a.getByRole('button',{name:'Save shared document',exact:true}).click()
  await status(a,'Saved shared document revision 1')
  docs=(await api(actors[0],'documents')).items
  let config=docs.find(d=>d.kind==='workspace-config')
  await a.getByLabel('Workspace server-set revision',{exact:true}).selectOption(`${set.id}:1`)
  await a.getByLabel('Workspace config revision',{exact:true}).selectOption(`${config.id}:1`)
  await a.getByRole('button',{name:'Save shared workspace',exact:true}).click()
  await status(a,'Saved shared workspace revision 1')
  let workspace=(await api(actors[0],'workspaces')).items[0]
  // Refresh the second operator's account view to observe current shared state.
  await b.getByRole('button',{name:'Sign out of deployment',exact:true}).click()
  await b.getByLabel('Deployment password',{exact:true}).fill(actors[1].password)
  await b.getByRole('button',{name:'Sign in to deployment',exact:true}).click()
  await b.getByLabel('Saved deployment document',{exact:true}).selectOption(set.id)
  await b.getByLabel('content.pool.slots',{exact:true}).fill('2')
  await b.getByRole('button',{name:'Save shared document',exact:true}).click()
  await status(b,'Saved shared document revision 2')
  await b.getByLabel('Saved deployment workspace',{exact:true}).selectOption(workspace.id)
  await b.waitForFunction(() => document.querySelector('[aria-label="Deployment workspace name"]').value==='Team deployment')
  await b.getByLabel('Workspace server-set revision',{exact:true}).selectOption(`${set.id}:2`)
  await b.getByLabel('Deployment workspace name',{exact:true}).fill('Two operator deployment')
  await b.getByRole('button',{name:'Save shared workspace',exact:true}).click()
  await status(b,'Saved shared workspace revision 2')
  await a.getByRole('button',{name:'Save shared workspace',exact:true}).click()
  await status(a,'409:')
  workspace=await api(actors[0],`workspaces/${workspace.id}`)
  set=await api(actors[0],`documents/${set.id}`)
  assert.equal(set.revision,2);assert.equal(workspace.revision,2);assert.ok(workspace.documents.some(r=>r.id===set.id && r.revision===2));assert.equal(workspace.name,'Two operator deployment')
  await api(actors[0],`workspaces/${workspace.id}`,'PATCH',{name:'stale',documents:workspace.documents},1,409)
  await b.screenshot({path:path.join(out,'shared-editor.png'),fullPage:true})
  observed('two-operator-shared-edit',['Both operators created/edited shared engine documents and workspace in the browser','Other operator observes revision 2','Stale shared workspace edit returned 409'], 'shared-editor.png')

  // Each operator supplies their own key; the second key is intentionally not authorized at sshd.
  for (const [i,page] of [[0,a],[1,b]]) {
    if (i===0) {await page.getByRole('button',{name:'Sign out of deployment',exact:true}).click();await page.getByLabel('Deployment password',{exact:true}).fill(actors[0].password);await page.getByRole('button',{name:'Sign in to deployment',exact:true}).click();await page.getByLabel('Saved deployment workspace',{exact:true}).selectOption(workspace.id)}
    await page.getByLabel('SSH user',{exact:true}).fill(fixture.ssh.user)
    await page.getByLabel('Private SSH key',{exact:true}).fill(fs.readFileSync(fixture.ssh.runtime+(i===0?'/client':'/unauthorized'),'utf8'))
    await page.getByRole('button',{name:'Save personal credential',exact:true}).click()
    await status(page,'Saved encrypted personal credential')
    assert.equal(await page.getByLabel('Private SSH key',{exact:true}).inputValue(),'')
    await page.getByLabel('Binding server',{exact:true}).selectOption('ssh')
    await page.getByRole('button',{name:'Bind my credential',exact:true}).click()
    await status(page,'Saved personal SSH binding')
    await page.getByRole('button',{name:'Check SSH access',exact:true}).click()
    await status(page,i===0?'SSH deployment access verified':'SSH authentication or host access denied')
  }
  const aliceCredentials=(await api(actors[0],'credentials')).items,bobCredentials=(await api(actors[1],'credentials')).items
  assert.equal(aliceCredentials.length,1);assert.equal(bobCredentials.length,1)
  const aliceBinding=await api(actors[0],`workspaces/${workspace.id}/credential-bindings`),bobBinding=await api(actors[1],`workspaces/${workspace.id}/credential-bindings`)
  assert.equal(aliceBinding.ssh,aliceCredentials[0].id);assert.equal(bobBinding.ssh,bobCredentials[0].id);assert.notEqual(aliceBinding.ssh,bobBinding.ssh)
  await api(actors[1],`workspaces/${workspace.id}/credential-bindings`,'PUT',{serverRef:'ssh',credentialId:aliceCredentials[0].id},undefined,404)
  assert.equal((await api(actors[0],`workspaces/${workspace.id}`)).revision,2)
  await a.screenshot({path:path.join(out,'ssh-allowed.png'),fullPage:true})
  await b.screenshot({path:path.join(out,'ssh-denied.png'),fullPage:true})
  observed('personal-ssh-binding',['Each user owns and sees only their credential metadata and binding','Binding leaves shared revision unchanged','Foreign credential binding rejected'], 'ssh-allowed.png')

  const allowed=await api(actors[0],`credentials/${aliceBinding.ssh}/check`,'POST',{workspaceId:workspace.id,serverRef:'ssh'})
  const denied=await api(actors[1],`credentials/${bobBinding.ssh}/check`,'POST',{workspaceId:workspace.id,serverRef:'ssh'})
  assert.equal(allowed.authenticated,true);assert.deepEqual(allowed.allowedOperations,['deploy']);assert.equal(denied.authenticated,false);assert.deepEqual(denied.allowedOperations,[])
  assert.ok(allowed.hostIdentity.includes(fixture.ssh.hostFingerprint))
  fs.chmodSync(fixture.ssh.runtime+'/denied',0)
  let blockedConfig=structuredClone(config);blockedConfig.content.dataRoot=fixture.ssh.runtime+'/denied'
  const input=d=>({kind:d.kind,name:d.name,contractVersion:d.contractVersion,content:d.content,assetRefs:[]})
  config=await api(actors[0],`documents/${config.id}`,'PATCH',input(blockedConfig),1)
  workspace=await api(actors[0],`workspaces/${workspace.id}`,'PATCH',{name:workspace.name,documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},2)
  const pathDenied=await api(actors[0],`credentials/${aliceBinding.ssh}/check`,'POST',{workspaceId:workspace.id,serverRef:'ssh'})
  assert.equal(pathDenied.authenticated,true);assert.deepEqual(pathDenied.allowedOperations,[])
  fs.chmodSync(fixture.ssh.runtime+'/denied',0o700)
  config.content.dataRoot=fixture.ssh.runtime+'/allowed'
  config=await api(actors[1],`documents/${config.id}`,'PATCH',input(config),config.revision)
  workspace=await api(actors[1],`workspaces/${workspace.id}`,'PATCH',{name:workspace.name,documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},workspace.revision)
  observed('ssh-access-permissions',['Actual dedicated sshd accepted one operator key and rejected the other','Authenticated SSH user denied deployment when actual target directory was not writable','Verified target host-key fingerprint'], 'ssh-denied.png')

  const cases=[]
  async function validateCase(label,d,valid) {const result=await api(actors[0],'documents/validate','POST',input(d));assert.equal(result.valid,valid,label);cases.push({label,result})}
  await validateCase('valid remote placement',set,true)
  let local=structuredClone(set);local.content.pool.hosts=[{name:'local',addr:'127.0.0.1'}];delete local.content.ssh
  await validateCase('valid local placement',local,true)
  for (const [label,mutate] of [
    ['overlapping port bands',d=>d.content.pool.ports.p2p.base=8600],
    ['port above 65535',d=>d.content.pool.ports.rpc.base=65535],
    ['negative stride',d=>d.content.pool.ports.rpc.step=-1],
    ['duplicate address',d=>d.content.pool.hosts.push({name:'duplicate',addr:'localhost.'})],
    ['mixed local and remote pool',d=>d.content.pool.hosts.push({name:'local',addr:'127.0.0.1'})],
    ['unknown host field',d=>d.content.pool.hosts[0].unknown='x']
  ]) {let bad=structuredClone(set);mutate(bad);await validateCase(label,bad,false);await api(actors[0],'documents','POST',input(bad),undefined,422)}
  for (const [label,mutate] of [ ['relative target root',d=>d.content.dataRoot='relative'],['path traversal',d=>d.content.paths.nodes='../nodes'],['invalid input mode',d=>d.content.inputs.mode='unsupported'] ]) {let bad=structuredClone(config);mutate(bad);await validateCase(label,bad,false);await api(actors[0],'documents','POST',input(bad),undefined,422)}
  await api(actors[0],'workspaces','POST',{name:'Invalid pinned revision',documents:[{id:set.id,revision:999},{id:config.id,revision:config.revision}]},undefined,409)
  fs.writeFileSync(path.join(out,'validation.json'),JSON.stringify(cases,null,2))
  observed('ports-paths-placement-validation',['Engine validates both local and remote placement','Port overlap/range/stride, duplicate/mixed hosts and invalid paths rejected by validation and save','Invalid document revision cannot form a workspace'], 'validation.json')

  const exported=await api(actors[0],`documents/${set.id}/export`)
  assert.deepEqual(exported,set.content)
  await api(actors[1],`credentials/${aliceBinding.ssh}`, 'GET',undefined,undefined,404)
  const secretSet=structuredClone(set);secretSet.content.ssh.password='leak-probe-secret'
  await api(actors[0],'documents','POST',input(secretSet),undefined,422)
  await a.getByLabel('Saved deployment document',{exact:true}).selectOption(set.id)
  const downloadPromise=a.waitForEvent('download')
  await a.getByRole('button',{name:'Export shared document',exact:true}).click()
  const download=await downloadPromise;await download.saveAs(path.join(out,'shared-export.json'))
  assert.deepEqual(JSON.parse(fs.readFileSync(path.join(out,'shared-export.json'),'utf8')),set.content)
  const sensitive=[actors[0].password,actors[1].password,'leak-probe-secret',fs.readFileSync(fixture.ssh.runtime+'/client','utf8'),fs.readFileSync(fixture.ssh.runtime+'/unauthorized','utf8')]
  const recorded=JSON.stringify(records)
  for (const value of sensitive) assert.equal(recorded.includes(value),false,'secret returned by API')
  const persisted=fs.readFileSync(fixture.store+'/deployment.json','utf8')
  for (const value of sensitive) assert.equal(persisted.includes(value),false,'plaintext secret persisted')
  assert.equal(first.errors.length+second.errors.length,0,JSON.stringify([...first.errors,...second.errors]))
  observed('secret-free-export',['Shared browser export preserves server declaration and contains no secret material','API responses and persisted deployment snapshot contain no plaintext credentials','Shared document save refuses embedded SSH secrets'], 'shared-export.json')
  fs.writeFileSync(path.join(out,'api-observations.json'),JSON.stringify(records,null,2))
  fs.writeFileSync(path.join(out,'browser.json'),JSON.stringify({scenarios:observations,workspaceId:workspace.id,workspaceRevision:workspace.revision,credentialId:aliceBinding.ssh,serverRef:'ssh',browserVersion:browser.version()},null,2))
} finally { await browser.close() }
