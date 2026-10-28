import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import path from 'node:path'
import assert from 'node:assert/strict'
const field=(page,path)=>page.locator(`[data-field-path="${path}"]`).locator('input,select')

const [url, out, fixturePath] = process.argv.slice(2)
const fixture = JSON.parse(fs.readFileSync(fixturePath, 'utf8'))
const observations = [], records = []
const actors = fixture.accounts
const owned = await launchOwnedBrowser(), browser = owned.browser
// Each operator works through their own browser context: its session cookie
// and CSRF token carry every API call, as they do for the screens.
async function api(actor, endpoint, method='GET', body, revision, expected=200) {
  const headers = { 'Content-Type': 'application/json', 'X-CSRF-Token': actor.csrf }
  if (revision) headers['If-Match'] = `"${revision}"`
  const response = await actor.context.request.fetch(url+'/api/v1/'+endpoint, {method, headers, data: body===undefined ? undefined : JSON.stringify(body)})
  const text = await response.text()
  assert.equal(response.status(), expected, `${method} ${endpoint}: ${text}`)
  let value; try { value = JSON.parse(text) } catch { value = text }
  // Requests carrying secrets are never copied to evidence.
  records.push({actor:actor.id,method,endpoint,status:response.status(),response:value,observedAt:new Date().toISOString()})
  return value
}
async function status(page, text) { try { await page.waitForFunction(t => document.querySelector('[data-testid=deployment-status]')?.textContent.includes(t), text) } catch { throw new Error(`Expected status ${text}, observed: ${await page.getByTestId('deployment-status').textContent()}`) } }
async function login(actor) {
  const context = await browser.newContext({viewport:{width:1440,height:1000},acceptDownloads:true})
  const reply = await context.request.post(url+'/api/v1/auth/login',{data:{username:actor.username,password:actor.password}})
  assert.equal(reply.status(),200,await reply.text())
  const session = await reply.json()
  Object.assign(actor,{context,csrf:session.csrfToken,id:session.user.id})
  const page = await context.newPage()
  const errors=[];page.on('pageerror',e=>errors.push(e.message))
  await page.goto(url+'/chains')
  await page.getByTestId('deployment-actor').filter({hasText:actor.username}).waitFor()
  await field(page,'content.pool.hosts[0].addr').waitFor()
  return {context,page,errors}
}
function observed(id, assertions, artifact) { observations.push({id,observedAt:new Date().toISOString(),mode:'team',transport:'SSH',ownership:'owned',actors:actors.map(a=>({id:a.id,username:a.username,role:a.role})),assertions:assertions.map(description=>({description,passed:true})),artifacts:[artifact]}) }

try {
  // The first administrator is created with the private setup token; it then
  // creates the two operator accounts the proof signs in with.
  const adminContext = await browser.newContext()
  const admin = await (await adminContext.request.post(url+'/api/v1/bootstrap',{data:{username:'deployment-admin',password:actors[0].password,setupToken:fixture.setupToken}})).json()
  for (const actor of actors) { const r = await adminContext.request.post(url+'/api/v1/users',{headers:{'X-CSRF-Token':admin.csrfToken},data:{username:actor.username,password:actor.password,role:actor.role}}); assert.equal(r.status(),201,await r.text()) }
  await adminContext.close()
  const first = await login(actors[0]), second = await login(actors[1])
  const a=first.page,b=second.page
  await field(a,'content.pool.hosts[0].name').fill('ssh')
  await field(a,'content.pool.hosts[0].addr').fill('localhost.')
  await a.locator('details.more > summary').click()
  await a.locator('[data-add-path="content.ssh"]').click()
  await a.locator('[data-add-path="content.ssh.port"]').click()
  await field(a,'content.ssh.port').fill(String(fixture.ssh.port))
  await a.locator('[data-add-path="content.ssh.known_hosts_file"]').click()
  await field(a,'content.ssh.known_hosts_file').fill(fixture.ssh.knownHosts)
  await a.getByRole('button',{name:'배포 문서 검증',exact:true}).click()
  await status(a,'엔진 검증 통과')
  await a.getByRole('button',{name:'공유 문서 저장',exact:true}).click()
  await status(a,'공유 문서 revision 1 저장됨')
  let docs=(await api(actors[0],'documents')).items
  let set=docs.find(d=>d.kind==='server-set')
  assert.ok(set)
  await a.getByRole('button',{name:'새 대상 경로',exact:true}).click()
  await field(a,'content.dataRoot').fill(fixture.ssh.runtime+'/allowed')
  await a.getByRole('button',{name:'공유 문서 저장',exact:true}).click()
  await status(a,'공유 문서 revision 1 저장됨')
  docs=(await api(actors[0],'documents')).items
  let config=docs.find(d=>d.kind==='workspace-config')
  await a.getByLabel('workspace 서버 풀 revision',{exact:true}).selectOption(`${set.id}:1`)
  await a.getByLabel('workspace 대상 경로 revision',{exact:true}).selectOption(`${config.id}:1`)
  await a.getByRole('button',{name:'공유 workspace 저장',exact:true}).click()
  await status(a,'공유 workspace revision 1 저장됨')
  let workspace=(await api(actors[0],'workspaces')).items[0]
  // Reload the second operator's view to observe current shared state.
  await b.reload();await b.getByLabel('저장된 배포 문서',{exact:true}).locator(`option[value="${set.id}"]`).waitFor({state:'attached'})
  await b.getByLabel('저장된 배포 문서',{exact:true}).selectOption(set.id)
  await field(b,'content.pool.slots').fill('2')
  await b.getByRole('button',{name:'공유 문서 저장',exact:true}).click()
  await status(b,'공유 문서 revision 2 저장됨')
  await b.getByLabel('저장된 배포 workspace',{exact:true}).selectOption(workspace.id)
  await b.waitForFunction(() => document.querySelector('[aria-label="배포 workspace 이름"]').value==='팀 workspace')
  await b.getByLabel('workspace 서버 풀 revision',{exact:true}).selectOption(`${set.id}:2`)
  await b.getByLabel('배포 workspace 이름',{exact:true}).fill('Two operator deployment')
  await b.getByRole('button',{name:'공유 workspace 저장',exact:true}).click()
  await status(b,'공유 workspace revision 2 저장됨')
  await a.getByRole('button',{name:'공유 workspace 저장',exact:true}).click()
  await status(a,'409:')
  workspace=await api(actors[0],`workspaces/${workspace.id}`)
  set=await api(actors[0],`documents/${set.id}`)
  assert.equal(set.revision,2);assert.equal(workspace.revision,2);assert.ok(workspace.documents.some(r=>r.id===set.id && r.revision===2));assert.equal(workspace.name,'Two operator deployment')
  await api(actors[0],`workspaces/${workspace.id}`,'PATCH',{name:'stale',documents:workspace.documents},1,409)
  await b.screenshot({path:path.join(out,'shared-editor.png'),fullPage:true})
  observed('two-operator-shared-edit',['Both operators created/edited shared engine documents and workspace in the browser','Other operator observes revision 2','Stale shared workspace edit returned 409'], 'shared-editor.png')

  // Each operator supplies their own key; the second key is intentionally not authorized at sshd.
  // Credentials are registered on the settings screen, then bound to the
  // workspace server on the chain screen.
  for (const [i,page] of [[0,a],[1,b]]) {
    await page.goto(url+'/settings')
    await page.getByLabel('SSH 사용자',{exact:true}).fill(fixture.ssh.user)
    await page.getByLabel('SSH 개인키',{exact:true}).fill(fs.readFileSync(fixture.ssh.runtime+(i===0?'/client':'/unauthorized'),'utf8'))
    await page.getByRole('button',{name:'개인 자격증명 저장',exact:true}).click()
    await page.getByTestId('credential-status').filter({hasText:'암호화한 개인 자격증명을 저장했습니다'}).waitFor()
    assert.equal(await page.getByLabel('SSH 개인키',{exact:true}).inputValue(),'')
    const credentialId=(await api(actors[i],'credentials')).items[0].id
    await page.goto(url+'/chains')
    await page.getByLabel('저장된 배포 workspace',{exact:true}).locator(`option[value="${workspace.id}"]`).waitFor({state:'attached'})
    await page.getByLabel('저장된 배포 workspace',{exact:true}).selectOption(workspace.id)
    await page.getByLabel('연결 서버',{exact:true}).selectOption('ssh')
    await page.getByLabel('연결 자격증명',{exact:true}).selectOption(credentialId)
    await page.getByRole('button',{name:'내 자격증명 연결',exact:true}).click()
    await status(page,'내 SSH 연결 저장됨')
    await page.getByRole('button',{name:'SSH 접근 확인',exact:true}).click()
    await status(page,i===0?'SSH 배포 접근 확인됨':'SSH authentication or host access denied')
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
  await a.getByLabel('저장된 배포 문서',{exact:true}).selectOption(set.id)
  const downloadPromise=a.waitForEvent('download')
  await a.getByRole('button',{name:'공유 문서 내보내기',exact:true}).click()
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
  fs.writeFileSync(path.join(out,'browser.json'),JSON.stringify({scenarios:observations,accountIds:actors.map(a=>a.id),workspaceId:workspace.id,workspaceRevision:workspace.revision,credentialId:aliceBinding.ssh,serverRef:'ssh',browserVersion:browser.version()},null,2))
} finally { await owned.stop() }
