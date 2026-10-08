import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(),browser=owned.browser
const context=await browser.newContext({viewport:{width:1440,height:1100}}),verified=[]
let session,releaseCatalog
async function api(path,method='GET',data,key,expected=200){
 const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const text=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${text}`)
 assert.ok(!text.includes(f.password)&&!text.includes(f.setupToken),'bootstrap secret leaked')
 return text?JSON.parse(text):undefined
}
async function upload(source,kind='template'){
 const r=await context.request.post(f.url+'/api/v1/assets',{headers:{'X-CSRF-Token':session.csrfToken},multipart:{kind,file:{name:source.chain+'-finished.json',mimeType:'application/json',buffer:fs.readFileSync(source.path)}}})
 assert.equal(r.status(),201,await r.text());return r.json()
}
async function waitJob(id){
 for(let i=0;i<900;i++){
  const j=await api('jobs/'+id)
  if(['succeeded','failed','cancelled','interrupted'].includes(j.state)){assert.equal(j.state,'succeeded',JSON.stringify(j));return j}
  await new Promise(r=>setTimeout(r,200))
 }
 throw new Error('native finished-genesis test timed out')
}
try{
 session=await api('bootstrap','POST',{username:'test-genesis-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 for(const [i,source] of f.genesis.entries()){
  const chain=source.chain,asset=await upload(source),bytes=fs.readFileSync(source.path)
  assert.equal(asset.checksum,source.checksum)
  const p2p=44700+i*100,rpc=17400+i*100
  const set=await api('documents','POST',{kind:'server-set',name:chain+' test genesis pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:p2p,step:10},rpc:{base:rpc,step:10}}}}},undefined,201)
  const config=await api('documents','POST',{kind:'workspace-config',name:chain+' test genesis paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d'+i,paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'+i},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
  const w=await api('workspaces','POST',{name:chain+' test genesis',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
  const content={schemaVersion:'2',kind:'case',id:chain+'-finished-genesis-test',chainPreset:{chain,binaries:{default:chain==='stablenet'?'gstable':'gwemix'},topology:{bp:4},genesis:{overlay:{config:{chainId:9999}}},launch:chain==='wemix'?undefined:{all:{ipcdisable:true}}},steps:[{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}
  const original=await api('documents','POST',{kind:'case',name:content.id,contractVersion:'2',assetRefs:[],content},undefined,201)
  const page=await context.newPage();await page.goto(f.url+'/tests')
  const pageErrors=[];page.on('pageerror',error=>{pageErrors.push(error.message);fs.writeFileSync(out+'/'+chain+'-page-errors.json',JSON.stringify(pageErrors,null,2))})
  await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
  await page.getByLabel('작업 매니페스트',{exact:true}).selectOption(chain)
  await page.getByLabel('작업 바이너리',{exact:true}).selectOption(chain)
  await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
  await page.getByLabel('실행 케이스 '+original.id,{exact:true}).check()
  await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
  await page.getByLabel('실행 계획',{exact:true}).waitFor()
  const section=page.getByLabel('Structured DSL editor',{exact:true})
  await page.getByLabel('공유 테스트',{exact:true}).selectOption(original.id)
  await page.getByLabel('DSL status',{exact:true}).filter({hasText:'공유 테스트 revision 1'}).waitFor()
  const picker=page.getByLabel('테스트 완성 genesis 자료',{exact:true});await picker.waitFor()
  assert.ok((await picker.locator('option[value="'+asset.id+'"]').innerText()).includes(asset.name),'file chooser hides registered filename')
  await picker.selectOption(asset.id);await picker.selectOption('')
  const declaration=()=>section.locator('details').filter({has:page.getByText('Import and current declaration',{exact:true})}).locator('pre').last().textContent()
  assert.equal(JSON.parse(await declaration()).chainPreset.genesis.overlay.config.chainId,9999,'clearing file selection lost generation settings')
  await picker.selectOption(asset.id)
  assert.equal(JSON.parse(await declaration()).chainPreset.genesis.ref,'asset:'+asset.id,'picker did not change current declaration: '+pageErrors.join('; '))
  const validationResponse=page.waitForResponse(r=>r.url()===f.url+'/api/v1/test-cases/import'&&r.request().method()==='POST')
  await page.getByRole('button',{name:'Validate test scenario',exact:true}).click()
  const vr=await validationResponse
  fs.writeFileSync(out+'/'+chain+'-validated.json',await vr.text())
  assert.equal((await vr.json()).content.chainPreset.genesis.ref,'asset:'+asset.id,'validation changed registered reference')
  await page.getByLabel('DSL status',{exact:true}).filter({hasText:'Engine validation passed'}).waitFor()
  const savedResponse=page.waitForResponse(r=>r.url()===f.url+'/api/v1/documents/'+original.id&&r.request().method()==='PATCH')
  let capturedResolve,deliveredResolve
  const captured=new Promise(resolve=>capturedResolve=resolve),delivered=new Promise(resolve=>deliveredResolve=resolve)
  if(i===0){
   let held=false
   const gate=new Promise(resolve=>releaseCatalog=resolve)
   await page.route(f.url+'/api/v1/documents',async route=>{
    if(held)return route.continue()
    held=true
    const response=await route.fetch()
    assert.equal((await response.json()).items.find(d=>d.id===original.id).revision,2,'held response did not snapshot prior revision')
    capturedResolve();await gate;await route.fulfill({response});deliveredResolve()
   })
  }
  await page.getByRole('button',{name:'공유 테스트 저장',exact:true}).click()
  const sr=await savedResponse;assert.equal(sr.status(),200,await sr.text());let saved=await sr.json()
  assert.deepEqual(saved.assetRefs,[asset.id]);assert.equal(saved.content.chainPreset.genesis.ref,'asset:'+asset.id)
  if(i===0){
   await captured
   const secondResponse=page.waitForResponse(r=>r.url()===f.url+'/api/v1/documents/'+original.id&&r.request().method()==='PATCH')
   await page.getByRole('button',{name:'공유 테스트 저장',exact:true}).click()
   const second=await secondResponse;assert.equal(second.status(),200,await second.text());saved=await second.json();assert.equal(saved.revision,3)
   await page.waitForFunction(id=>document.querySelector(`input[aria-label="실행 케이스 ${id}"]`)?.closest('label')?.textContent.includes('r3'),original.id)
   const staleResponse=page.waitForResponse(async response=>response.url()===f.url+'/api/v1/documents'&&(await response.json()).items?.find(d=>d.id===original.id)?.revision===2)
   releaseCatalog();await delivered;await (await staleResponse).finished()
   await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))))
   assert.ok((await page.getByLabel('실행 케이스 '+original.id,{exact:true}).locator('..').textContent()).includes('r3'),'late response replaced latest catalog')
   await page.unroute(f.url+'/api/v1/documents');releaseCatalog=null
  }
  const base={workspaceId:w.id,operation:'test.run',documentRefs:w.documents,assetRefs:[chain,asset.id],retention:'retain',arguments:{manifestId:chain,assetId:chain,serverRef:'local',caseRefs:[{id:saved.id,revision:saved.revision}]}}
  const plan=await api('plans','POST',base,undefined,201)
  assert.ok(plan.changes.some(v=>v.includes(asset.id)&&v.includes(asset.checksum)),'review lost test genesis identity')
  const pinned=JSON.parse(fs.readFileSync(f.store+'/jobs.json','utf8')).plans[plan.id].prepared.payload.testRun.genesis[0]
  assert.ok(fs.readFileSync(pinned.path).equals(bytes))
  assert.ok(!JSON.stringify(plan).includes(pinned.path),'review disclosed private genesis path')
  if(i===0){
   for(const [kind,id] of [['unknown','f'.repeat(32)],['configuration',(await upload(source,'configuration')).id]]){
    const bad=structuredClone(saved.content);bad.id='bad-'+kind;bad.chainPreset.genesis.ref='asset:'+id
    const doc=await api('documents','POST',{kind:'case',name:bad.id,contractVersion:'2',assetRefs:[id],content:bad},undefined,201)
    await api('plans','POST',{...base,assetRefs:[chain,id],arguments:{...base.arguments,caseRefs:[{id:doc.id,revision:doc.revision}]}},undefined,kind==='unknown'?404:422)
   }
   for(const [path,status] of [[pinned.path,409],[f.store+'/assets/'+asset.id+'/payload',422]]){
    fs.writeFileSync(path,'{"config":{"chainId":1}}')
    await api('jobs','POST',{planId:plan.id},plan.id,status)
    assert.ok(!fs.existsSync(f.store+'/networks/'+w.id+'/chain-record.json'),'changed file produced native node effects')
    fs.writeFileSync(path,bytes)
   }
  }
  // Saving must refresh execution choices without discarding the open editor.
  const executionCase=page.getByLabel('실행 케이스 '+saved.id,{exact:true})
  await page.waitForFunction(({id,revision})=>{
   const input=document.querySelector(`input[aria-label="실행 케이스 ${id}"]`)
   return input?.closest('label')?.textContent.includes(`r${revision}`)
  },{id:saved.id,revision:saved.revision},{timeout:10000})
  assert.equal(await picker.inputValue(),asset.id,'catalog update discarded open editor selection')
  assert.ok(await executionCase.isChecked(),'catalog update discarded selected case')
  assert.equal(await page.getByLabel('작업 Workspace',{exact:true}).inputValue(),w.id,'catalog update discarded workspace')
  assert.equal(await page.getByLabel('작업 바이너리',{exact:true}).inputValue(),chain,'catalog update discarded binary')
  assert.equal(await page.getByLabel('실행 계획',{exact:true}).count(),0,'save left an obsolete plan executable')
  await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
  await page.getByLabel('작업 매니페스트',{exact:true}).selectOption(chain)
  await page.getByLabel('작업 바이너리',{exact:true}).selectOption(chain)
  await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
  await executionCase.check()
  await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click();await page.getByLabel('실행 계획',{exact:true}).waitFor()
  assert.ok((await page.getByLabel('실행 계획',{exact:true}).innerText()).includes(asset.id),'UI omitted selected test file')
  await page.getByLabel('실행 계획',{exact:true}).screenshot({path:out+'/'+chain+'-test-genesis-plan.png'})
  const accepted=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
  await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click()
  const ar=await accepted;assert.equal(ar.status(),202,await ar.text());const job=await waitJob((await ar.json()).id)
  assert.equal(job.runIds.length,1)
  const record=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
  assert.equal(record.nodes.length,4);assert.ok(fs.readFileSync(record.genesisPath).equals(bytes),'engine changed finished genesis bytes')
  const observed=await api('networks/'+w.id+'/observations')
  const response=await context.request.post('http://127.0.0.1:'+rpc+'/',{data:{jsonrpc:'2.0',id:1,method:'eth_chainId',params:[]}})
  assert.equal(response.status(),200);assert.equal(Number.parseInt((await response.json()).result,16),source.chainId)
  for(const n of record.nodes){
   const probe=observed.nodes.find(v=>v.id==='node'+n.index);assert.equal(probe.state,'running');assert.equal(probe.observedPid,n.pid)
   const dirs=fs.readdirSync(n.dataDir,{withFileTypes:true}).filter(e=>e.isDirectory()).map(e=>n.dataDir+'/'+e.name+'/chaindata').filter(d=>fs.existsSync(d+'/CURRENT'))
   assert.equal(dirs.length,1);assert.ok(fs.readdirSync(dirs[0]).some(v=>v.startsWith('MANIFEST-')))
   const control=await api('plans','POST',{workspaceId:w.id,operation:'node.stop',documentRefs:w.documents,assetRefs:[chain],nodeIds:['node'+n.index],retention:'retain',arguments:{manifestId:chain,assetId:chain,serverRef:'local',validators:4}},undefined,201)
   const stop=await api('jobs','POST',{planId:control.id},control.id,202);await waitJob(stop.id)
   const genesis=JSON.parse(execFileSync(record.binary,['--datadir',n.dataDir,'dumpgenesis'],{encoding:'utf8',timeout:20000}));assert.equal(Number(genesis.config.chainId),source.chainId)
  }
  const histories=(await api('history?caseId='+saved.content.id)).items.filter(v=>v.summary.kind==='engine-session')
  assert.equal(histories.length,1);assert.equal(histories[0].state,'succeeded')
  const detail=await api('history/'+histories[0].id);assert.equal(detail.summary.counts.pass,1);assert.equal(detail.summary.counts.skip||0,0)
  verified.push({chain,assetId:asset.id,checksum:asset.checksum,chainId:source.chainId,jobId:job.id,runIds:job.runIds,historyId:histories[0].id,nodes:record.nodes.length})
  assert.deepEqual(pageErrors,[],'browser reported an unhandled editor error')
  await page.close()
 }
 fs.writeFileSync(out+'/browser.json',JSON.stringify({verified,browserCaseGenesisSelection:true,savedCatalogWithoutReload:true,lateCatalogResponseRefused:true,obsoleteReviewCleared:true,editorAndExecutionChoicesPreserved:true,changedSourcesAndSnapshotsRefused:true,unknownAndWrongKindRefused:true,nativeDatabasesRPCAndPIDsVerified:true,realPassingSessionsWithoutSkips:true,seedAcceptanceAwarded:false},null,2))
 console.log('NATIVE TEST GENESIS PASS: browser-selected bytes initialize twelve databases and three passing engine sessions without skips.')
}finally{releaseCatalog?.();await owned.stop()}
