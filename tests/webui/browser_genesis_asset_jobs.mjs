import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import crypto from 'node:crypto'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(),browser=owned.browser
const context=await browser.newContext({viewport:{width:1440,height:1050}}),applied=[],jobs=[]
let session
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
 for(let i=0;i<600;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,'succeeded',JSON.stringify(job));jobs.push(job);return job}await new Promise(r=>setTimeout(r,200))}
 throw new Error('finished genesis setup timed out')
}
try{
 session=await api('bootstrap','POST',{username:'genesis-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 for(const [i,source] of f.genesis.entries()){
  const chain=source.chain,asset=await upload(source),bytes=fs.readFileSync(source.path)
  assert.equal(asset.checksum,source.checksum)
  const set=await api('documents','POST',{kind:'server-set',name:chain+' genesis pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:42900+i*100,step:10},rpc:{base:15800+i*100,step:10}}}}},undefined,201)
  const config=await api('documents','POST',{kind:'workspace-config',name:chain+' genesis paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d'+i,paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'+i},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
  const w=await api('workspaces','POST',{name:chain+' genesis workspace',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
  const initial=await api('documents','POST',{kind:'chain-preset',name:chain+' genesis preset',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'chain-preset',id:chain+'-genesis-review',chain,topology:{bp:4},genesis:{overlay:{config:{chainId:9999}}},launch:{all:{maxpeers:'40',cache:'64'}}}},undefined,201)
  const page=await context.newPage();await page.goto(f.url+'/chains')
  await page.locator('#saved-chain-document').selectOption(initial.id)
  const picker=page.getByLabel('완성된 genesis 자료',{exact:true});await picker.waitFor();await picker.selectOption(asset.id)
  // Returning to generation preserves the deliberately replaced declaration.
  await picker.selectOption('')
  const section=page.locator('section[aria-labelledby="chain-heading"]')
  const declaration=()=>section.locator('details').filter({has:page.getByText('Generated declaration',{exact:true})}).locator('pre').textContent()
  assert.equal(JSON.parse(await declaration()).genesis.overlay.config.chainId,9999,'file selection silently lost generation settings')
  await picker.selectOption(asset.id)
  await section.getByRole('button',{name:'Validate configuration',exact:true}).click()
  await section.getByText('Engine validation passed',{exact:true}).waitFor()
  const savedResponse=page.waitForResponse(r=>r.url()===f.url+'/api/v1/documents/'+initial.id&&r.request().method()==='PATCH')
  await page.getByRole('button',{name:'공유 체인 구성 저장',exact:true}).click()
  const sr=await savedResponse;assert.equal(sr.status(),200,await sr.text());const saved=await sr.json()
  assert.deepEqual(saved.assetRefs,[asset.id]);assert.equal(saved.content.genesis.ref,'asset:'+asset.id)
  const base={workspaceId:w.id,operation:'chain.setup',documentRefs:w.documents,assetRefs:[chain,asset.id],retention:'retain',arguments:{manifestId:chain,assetId:chain,serverRef:'local',chainPresetRef:{id:saved.id,revision:saved.revision}}}
  const plan=await api('plans','POST',base,undefined,201)
  assert.ok(plan.changes.some(v=>v.includes(asset.id)&&v.includes(asset.checksum)),'review omitted genesis ID/checksum')
  const pinned=JSON.parse(fs.readFileSync(f.store+'/jobs.json','utf8')).plans[plan.id].prepared.payload.preset.genesis
  assert.equal(pinned.asset.id,asset.id);assert.ok(fs.readFileSync(pinned.path).equals(bytes))
  if(i===0){
   for(const [kind,id] of [['unknown','f'.repeat(32)],['configuration',(await upload(source,'configuration')).id]]){
    const bad=await api('documents','POST',{kind:'chain-preset',name:'bad '+kind,contractVersion:'2',content:{...saved.content,id:'bad-'+kind,genesis:{mode:'existing',ref:'asset:'+id}},assetRefs:[id]},undefined,201)
    await api('plans','POST',{...base,assetRefs:[chain,id],arguments:{...base.arguments,chainPresetRef:{id:bad.id,revision:bad.revision}}},undefined,kind==='unknown'?404:422)
   }
   fs.writeFileSync(pinned.path,'{"config":{"chainId":1}}')
   await api('jobs','POST',{planId:plan.id},plan.id,422)
   assert.ok(!fs.existsSync(f.store+'/networks/'+w.id+'/chain-record.json'),'changed snapshot produced native effects')
   fs.writeFileSync(pinned.path,bytes)
   const raw=f.store+'/assets/'+asset.id+'/payload';fs.writeFileSync(raw,'{"config":{"chainId":2}}')
   await api('jobs','POST',{planId:plan.id},plan.id,422)
   fs.writeFileSync(raw,bytes)
  }
  // Reload refreshes the job panel's immutable saved dependency list.
  await page.reload();await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
  await page.getByLabel('작업 매니페스트',{exact:true}).selectOption(chain)
  await page.getByLabel('작업 바이너리',{exact:true}).selectOption(chain)
  await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
  await page.getByLabel('작업 체인 구성',{exact:true}).selectOption(saved.id)
  await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click();await page.getByLabel('실행 계획',{exact:true}).waitFor()
  assert.ok((await page.getByLabel('실행 계획',{exact:true}).innerText()).includes(asset.id),'UI plan omitted selected file dependency')
  await page.getByLabel('실행 계획',{exact:true}).screenshot({path:out+'/'+chain+'-genesis-plan.png'})
  const acceptedResponse=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
  await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click();const ar=await acceptedResponse;assert.equal(ar.status(),202,await ar.text());await waitJob((await ar.json()).id)
  const record=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
  assert.ok(fs.readFileSync(record.genesisPath).equals(bytes),'finished genesis was not used verbatim')
  for(const node of record.nodes){
   const dirs=fs.readdirSync(node.dataDir,{withFileTypes:true}).filter(e=>e.isDirectory()).map(e=>node.dataDir+'/'+e.name+'/chaindata').filter(d=>fs.existsSync(d+'/CURRENT'))
   assert.equal(dirs.length,1,'native database missing');assert.ok(fs.readdirSync(dirs[0]).some(n=>n.startsWith('MANIFEST-')))
   const genesis=JSON.parse(execFileSync(record.binary,['--datadir',node.dataDir,'dumpgenesis'],{encoding:'utf8',timeout:20000}));assert.equal(genesis.config.chainId,source.chainId,'database ignored uploaded genesis')
  }
  applied.push({chain,assetId:asset.id,checksum:asset.checksum,chainId:source.chainId,nodes:record.nodes.length})
  await page.close()
 }
 fs.writeFileSync(out+'/browser.json',JSON.stringify({applied,jobs:jobs.map(j=>({id:j.id,state:j.state})),unknownAndWrongKindRefused:true,changedSourcesAndSnapshotsRefused:true,nativeDatabaseGenesisVerified:true,seedAcceptanceAwarded:false},null,2))
 console.log('WEB GENESIS ASSETS PASS: browser selection, immutable references, twelve native databases and actual genesis verification.')
}finally{await owned.stop()}
