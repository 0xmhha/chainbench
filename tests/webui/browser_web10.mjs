import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [url,out,fixturePath,phase]=process.argv.slice(2)
const f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(), browser=owned.browser
const responses=[],scenarios=[]
const secrets=[f.setupToken,...f.accounts.map(a=>a.password),f.sshKey,f.marker,f.nodeKey,...f.sshKey.split('\n').slice(1,-1)].filter(Boolean)
function scan(text){for(const s of secrets)assert.ok(!text.includes(s),'secret leaked in response')}
async function request(context,endpoint,method='GET',body,csrf,expected=200,extra={}){
 const r=await context.request.fetch(url+endpoint,{method,headers:{'Content-Type':'application/json',...(csrf?{'X-CSRF-Token':csrf}:{}),...extra},data:body===undefined?undefined:JSON.stringify(body)})
 const text=await r.text();scan(text);assert.equal(r.status(),expected,`${method} ${endpoint}: ${text}`)
 let data;try{data=JSON.parse(text)}catch{data=text}
 // CSRF is not copied into public acceptance records.
 const publicData=data?.csrfToken?{user:data.user,mode:data.mode,expiresAt:data.expiresAt}:data
 responses.push({endpoint,method,status:r.status(),response:publicData,observedAt:new Date().toISOString()})
 return data
}
function observed(id,descriptions,artifact='browser.json'){scenarios.push({id,observedAt:new Date().toISOString(),mode:phase,transport:'SSH',ownership:'owned',assertions:descriptions.map(description=>({description,passed:true})),artifacts:[artifact]})}
async function login(account){const context=await browser.newContext({viewport:{width:1440,height:1000}});const page=await context.newPage();await page.goto(url);await page.getByLabel('Account username',{exact:true}).fill(account.username);await page.getByLabel('Account password',{exact:true}).fill(account.password);await page.getByRole('button',{name:'Sign in',exact:true}).click();await page.getByTestId('account-session').waitFor();const session=await request(context,'/api/v1/auth/me');return {context,page,session}}
try{
 if(phase==='personal'){
  const context=await browser.newContext(), page=await context.newPage();await page.goto(url)
  await request(context,'/api/runs','GET',undefined,undefined,401)
  await request(context,'/api/v1/bootstrap','POST',{username:'admin',password:f.accounts[0].password,setupToken:'incorrect'},undefined,403)
  await page.getByLabel('Account username',{exact:true}).fill(f.accounts[0].username)
  await page.getByLabel('Account password',{exact:true}).fill(f.accounts[0].password)
  await page.getByLabel('Setup token',{exact:true}).fill(f.setupToken)
  await page.getByRole('button',{name:'Create administrator',exact:true}).click()
  await page.getByTestId('account-session').waitFor()
  let session=await request(context,'/api/v1/auth/me');assert.equal(session.mode,'personal')
  await request(context,'/api/v1/bootstrap','POST',{username:'other',password:f.accounts[0].password,setupToken:f.setupToken},undefined,409)
  for(const account of f.accounts.slice(1))await request(context,'/api/v1/users','POST',{username:account.username,password:account.password,role:account.role},session.csrfToken,201)
  const cookie=(await context.cookies()).find(c=>c.name==='chainbench_session');assert.ok(cookie.httpOnly);assert.equal(cookie.sameSite,'Strict')
  await page.screenshot({path:out+'/personal-login.png'})
  await page.getByRole('button',{name:'Sign out',exact:true}).click();await page.getByRole('button',{name:'Sign in',exact:true}).waitFor();await request(context,'/api/v1/auth/me','GET',undefined,undefined,401)
  observed('bootstrap-restrictions',['wrong token forbidden','first administrator created in browser','repeat bootstrap conflicts','private HttpOnly SameSite cookie'], 'personal-login.png')
  fs.writeFileSync(out+'/personal.json',JSON.stringify({scenarios,responses,browserVersion:browser.version()},null,2))
 }else{
  const users=[];for(const a of f.accounts)users.push(await login(a))
  const matrix={}
  const setContent={version:2,pool:{hosts:[{name:'ssh',addr:'localhost.'}],slots:1,ports:{p2p:{base:31000,step:10},rpc:{base:8600,step:10}}},ssh:{port:f.ssh.port,known_hosts_file:f.ssh.knownHosts}}
  function doc(name,kind='server-set',content=setContent){return {kind,name,contractVersion:'2',content,assetRefs:[]}}
  for(let i=0;i<users.length;i++){
   const {context,page,session}=users[i],role=f.accounts[i].role
   await page.locator('nav a[href="/chains"]').click();await page.getByTestId('deployment-actor').waitFor()
   await page.locator('#chain-preset').waitFor()
   const preset=await page.locator('#chain-preset option').nth(1).getAttribute('value')
   await page.locator('#chain-preset').selectOption(preset)
   assert.equal(await page.getByRole('button',{name:'Validate configuration',exact:true}).count()>0,role!=='viewer')
   if(role==='viewer') assert.equal(await page.locator('#count-bp').isDisabled(),true)
   else {await page.getByRole('button',{name:'Validate configuration',exact:true}).click();await page.getByText('Engine validation passed',{exact:true}).waitFor()}
   const writable=await page.getByRole('button',{name:'Save shared document',exact:true}).count()>0
   assert.equal(writable,role!=='viewer');assert.equal(await page.getByText('User management',{exact:true}).count()>0,role==='administrator')
   const writeStatus=role==='viewer'?403:201, usersStatus=role==='administrator'?200:403
   await request(context,'/api/v1/documents','POST',doc(role),session.csrfToken,writeStatus)
   await request(context,'/api/v1/users','GET',undefined,undefined,usersStatus)
   await request(context,'/api/v1/users','POST',{username:'forbidden',password:f.marker,role:'administrator'},session.csrfToken,role==='administrator'?201:403)
   matrix[role]={uiWritable:writable,writeStatus,usersStatus}
   await page.screenshot({path:out+`/${role}.png`})
  }
  const [admin,operator,viewer]=users
  for(const id of f.legacyDocumentIDs){
   await request(viewer.context,`/api/v1/documents/${id}`,'GET',undefined,undefined,404)
   await request(admin.context,`/api/v1/documents/${id}/export`,'GET',undefined,undefined,404)
  }
  for(const id of f.legacyImportIDs) await request(admin.context,'/api/v1/documents/import/commit','POST',{previewId:id},admin.session.csrfToken,422)
  const docs=(await request(admin.context,'/api/v1/documents')).items
  const set=docs.find(d=>d.name==='operator')
  const config=await request(operator.context,'/api/v1/documents','POST',doc('paths','workspace-config',{version:1,dataRoot:f.ssh.runtime+'/allowed',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:'chainbench-out'},inputs:{mode:'generated'},execution:{chain:'fresh'}}),operator.session.csrfToken,201)
  const workspace=await request(operator.context,'/api/v1/workspaces','POST',{name:'security target',documents:[{id:set.id,revision:1},{id:config.id,revision:1}]},operator.session.csrfToken,201)
  const credential=await request(operator.context,'/api/v1/credentials','POST',{label:'SSH access',kind:'private-key',sshUser:f.ssh.user,privateKey:f.sshKey},operator.session.csrfToken,201)
  const markerCredential=await request(operator.context,'/api/v1/credentials','POST',{label:'password sentinel',kind:'password',sshUser:f.ssh.user,password:f.marker},operator.session.csrfToken,201)
  await request(operator.context,`/api/v1/workspaces/${workspace.id}/credential-bindings`,'PUT',{serverRef:'ssh',credentialId:credential.id},operator.session.csrfToken)
  const access=await request(operator.context,`/api/v1/credentials/${credential.id}/check`,'POST',{workspaceId:workspace.id,serverRef:'ssh'},operator.session.csrfToken)
  assert.equal(access.authenticated,true);assert.deepEqual(access.allowedOperations,['deploy'])
  for(const user of [admin]){const list=(await request(user.context,'/api/v1/credentials')).items;assert.ok(!list.some(c=>c.id===credential.id));await request(user.context,`/api/v1/credentials/${credential.id}`,'GET',undefined,undefined,404)}
  await request(viewer.context,'/api/v1/credentials','GET',undefined,undefined,403)
  await request(viewer.context,`/api/v1/credentials/${credential.id}`,'GET',undefined,undefined,403)
  // Even administrators cannot inspect or use another user's personal key.
  await request(admin.context,`/api/v1/credentials/${credential.id}/check`,'POST',{workspaceId:workspace.id,serverRef:'ssh'},admin.session.csrfToken,404)
  await request(admin.context,`/api/v1/workspaces/${workspace.id}/credential-bindings`,'PUT',{serverRef:'ssh',credentialId:credential.id},admin.session.csrfToken,404)
  await request(operator.context,'/api/v1/documents','POST',doc('no csrf'),undefined,403)
  await request(operator.context,'/api/v1/documents','POST',doc('foreign origin'),operator.session.csrfToken,403,{Origin:'https://foreign.invalid'})
  await request(operator.context,'/api/v1/documents','POST',doc('reject secrets','server-set',{...setContent,ssh:{...setContent.ssh,password:f.marker}}),operator.session.csrfToken,422)
  const topology={nodes:[{index:1,role:'bp',key:f.nodeKey}]}
  for(const declaration of [
   {schemaVersion:'2',kind:'chain-preset',id:'private-preset',chain:'stablenet',topology},
   {schemaVersion:'2',kind:'case',id:'private-case-v2',chainPreset:{chain:'stablenet',topology},steps:[{expect:'blockNumber',is:0}]},
   {schemaVersion:'1',id:'private-case-v1',chain:{name:'stablenet',binary:'gstable'},topology,assertions:[{assert:'blockNumber',expected:0}]}
  ]){
   const kind=declaration.kind||'case',input=doc(declaration.id,kind,declaration)
   await request(operator.context,'/api/v1/documents','POST',input,operator.session.csrfToken,422)
   const validation=await request(operator.context,'/api/v1/documents/validate','POST',input,operator.session.csrfToken)
   assert.equal(validation.valid,false,'node key material passed shared validation')
   const preview=await request(operator.context,'/api/v1/documents/import','POST',{filename:declaration.id+'.json',format:'json',kind,source:JSON.stringify(declaration)},operator.session.csrfToken)
   assert.equal(preview.validation.valid,false);assert.deepEqual(preview.redactedDocuments,[]);assert.equal(preview.sourcePreserved,true)
   await request(operator.context,'/api/v1/documents/import/commit','POST',{previewId:preview.previewId},operator.session.csrfToken,422)
   if(kind==='case') await request(operator.context,'/api/v1/test-cases/import','POST',{content:declaration},operator.session.csrfToken,422)
  }
  await request(viewer.context,'/api/v1/documents')
  // Deliberately place the known password sentinel into otherwise allowed declaration text.
  const sentinelDoc=await request(operator.context,'/api/v1/documents','POST',doc(f.marker,'server-set',{...setContent,pool:{...setContent.pool,hosts:[{name:f.marker,addr:'localhost.'}]}}),operator.session.csrfToken,201)
  const exported=await request(viewer.context,`/api/v1/documents/${sentinelDoc.id}/export`)
  assert.equal(exported.pool.hosts[0].name,'[REDACTED]')
  await request(operator.context,`/api/v1/credentials/${markerCredential.id}`)
  await request(operator.context,'/api/events','POST',{kind:'info',message:f.marker+' '+f.nodeKey,fields:{password:f.marker,privateKey:f.sshKey,nodeKey:f.nodeKey}},operator.session.csrfToken,202)
  await viewer.page.locator('nav a[href="/monitoring"]').click();try{await viewer.page.waitForFunction(()=>[...document.querySelectorAll('.msg')].some(e=>e.textContent.includes('[REDACTED]')),{},{timeout:10000})}catch{throw new Error('SSE observation: '+await viewer.page.locator('main').innerText())}
  scan(await viewer.page.content())
  await request(viewer.context,'/api/events','POST',{kind:'info',message:'denied'},viewer.session.csrfToken,403)
  await request(viewer.context,'/api/v1/documents/validate','POST',doc('denied'),viewer.session.csrfToken,403)
  // Role changes revoke all existing sessions and are applied at the next request.
  await request(admin.context,`/api/v1/users/${viewer.session.user.id}`,'PATCH',{active:false},admin.session.csrfToken)
  await request(viewer.context,'/api/v1/auth/me','GET',undefined,undefined,401)
  await request(viewer.context,'/api/v1/auth/login','POST',{username:f.accounts[2].username,password:f.accounts[2].password},undefined,401)
  await request(admin.context,`/api/v1/users/${viewer.session.user.id}`,'PATCH',{active:true},admin.session.csrfToken)
  await request(operator.context,'/api/v1/auth/logout','POST',undefined,operator.session.csrfToken,204)
  await request(operator.context,'/api/v1/auth/me','GET',undefined,undefined,401)
  const relogin=await request(operator.context,'/api/v1/auth/login','POST',{username:f.accounts[1].username,password:f.accounts[1].password})
  assert.equal(relogin.mode,'team')
  await request(admin.context,`/api/v1/users/${operator.session.user.id}`,'PATCH',{role:'viewer'},admin.session.csrfToken)
  await request(operator.context,'/api/v1/auth/me','GET',undefined,undefined,401)
  const downgraded=await request(operator.context,'/api/v1/auth/login','POST',{username:f.accounts[1].username,password:f.accounts[1].password})
  assert.equal(downgraded.user.role,'viewer')
  await request(operator.context,'/api/v1/documents','POST',doc('revoked operator'),downgraded.csrfToken,403)
  await request(operator.context,'/api/v1/credentials','GET',undefined,undefined,403)
  await request(admin.context,`/api/v1/users/${operator.session.user.id}`,'PATCH',{role:'operator'},admin.session.csrfToken)

  observed('personal-team-login',['personal bootstrap login observed','team accounts login through real browser','logout permits new login'])
  observed('role-ui-api-matrix',['administrator contains operator controls','viewer forms absent and writes forbidden','user management administrator only'])
  observed('csrf-ownership',['missing CSRF and foreign Origin forbidden','active account required on every request'])
  observed('foreign-credential-refusal',['foreign keys absent from lists','foreign GET/check/binding rejected including administrator','actual owner SSH access allowed'])
  observed('secret-free-output',['export and SSE sentinel removed','credential metadata has no material','validation error has no secrets','preset and v1/v2 case node key material refused before shared save or import publication','legacy preset and v1/v2 case histories absent from public reads and exports after restart; old import approvals refused'])
  fs.writeFileSync(out+'/browser.json',JSON.stringify({scenarios,responses,matrix,foreignStatuses:[404,404,404],csrfStatuses:[403,403],sessionStatuses:[401,401],secretScan:{leaks:0,responsesScanned:responses.length},access,credentialId:credential.id,browserVersion:browser.version()},null,2))
 }
}finally{await owned.stop()}
