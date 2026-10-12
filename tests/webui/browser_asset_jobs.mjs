import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import crypto from 'node:crypto'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(),browser=owned.browser
const context=await browser.newContext({viewport:{width:1440,height:1050}}),uploaded=[],jobs=[],observed=[]
const sources=JSON.parse(fs.readFileSync(f.runtime+'/assets.json')).filter(a=>['stablenet','wbft','wemix'].includes(a.id))
let session
async function api(path,method='GET',data,key,expected=200){
 const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const text=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${text}`)
 assert.ok(!text.includes(f.password)&&!text.includes(f.setupToken),'bootstrap secret leaked')
 return text?JSON.parse(text):undefined
}
async function waitJob(id){
 for(let i=0;i<600;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,'succeeded',JSON.stringify(job));jobs.push(job);return job}await new Promise(r=>setTimeout(r,200))}
 throw new Error('uploaded asset job timed out')
}
async function run(input){const plan=await api('plans','POST',input,undefined,201);assert.ok(plan.changes.some(v=>v.includes(input.arguments.assetId)&&v.includes('SHA-256')),'review omitted pinned asset identity');const job=await api('jobs','POST',{planId:plan.id},plan.id,202);return waitJob(job.id)}
try{
 session=await api('bootstrap','POST',{username:'asset-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 const page=await context.newPage();await page.goto(f.url+'/settings')
 for(const source of sources){
  await page.getByLabel('업로드 자료 종류',{exact:true}).selectOption('binary')
  await page.getByLabel('업로드 자료 파일',{exact:true}).setInputFiles(source.path)
  const response=page.waitForResponse(r=>r.url()===f.url+'/api/v1/assets'&&r.request().method()==='POST')
  await page.getByRole('button',{name:'자료 등록',exact:true}).click()
  const r=await response;assert.equal(r.status(),201,await r.text());const asset=await r.json()
  assert.equal(asset.compatibility.chain,source.chain);assert.equal(asset.checksum,source.sha256);assert.equal(asset.kind,'binary');assert.notEqual(asset.id,source.id)
  assert.ok(!JSON.stringify(asset).includes(f.runtime),'server path exposed')
  uploaded.push(asset)
 }
 await page.getByLabel('등록 자료',{exact:true}).screenshot({path:out+'/asset-library.png'})
 const raw=JSON.stringify([...uploaded].sort((a,b)=>a.id.localeCompare(b.id)))
 fs.writeFileSync(f.runtime+'/restart.request','owned asset persistence check')
 for(let i=0;i<300&&!fs.existsSync(f.runtime+'/restart.response');i++)await new Promise(r=>setTimeout(r,100))
 const restart=JSON.parse(fs.readFileSync(f.runtime+'/restart.response'));assert.ok(!restart.error&&restart.newPid!==restart.oldPid)
 await api('auth/me','GET',undefined,undefined,401)
 session=await api('auth/login','POST',{username:'asset-admin',password:f.password})
 assert.equal(JSON.stringify((await api('assets')).items),raw)
 for(const [i,asset] of uploaded.entries()){
  const chain=asset.compatibility.chain
  const set=await api('documents','POST',{kind:'server-set',name:chain+' upload pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:41800+i*100,step:10},rpc:{base:13200+i*100,step:10}}}}},undefined,201)
  const config=await api('documents','POST',{kind:'workspace-config',name:chain+' upload paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d'+i,paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'+i},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
  const w=await api('workspaces','POST',{name:chain+' upload workspace',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
  const base={workspaceId:w.id,operation:'chain.setup',documentRefs:w.documents,assetRefs:[asset.id],retention:'retain',arguments:{manifestId:chain,assetId:asset.id,serverRef:'local',validators:4}}
  // A reviewed asset whose bytes change must fail before any native database exists.
  if(i===0){
   const plan=await api('plans','POST',base,undefined,201)
   const payload=f.store+'/assets/'+asset.id+'/payload',bytes=fs.readFileSync(payload),damaged=Buffer.from(bytes);damaged[0]^=1
   fs.writeFileSync(payload,damaged)
   await api('jobs','POST',{planId:plan.id},plan.id,422)
   assert.ok(!fs.existsSync(f.store+'/networks/'+w.id+'/chain-record.json'),'changed binary caused native effects')
   const r=await context.request.get(f.url+'/api/v1/assets/'+asset.id);assert.equal(r.status(),422);assert.ok(!(await r.text()).includes(payload),'integrity error exposed private path')
   fs.writeFileSync(payload,bytes)
  }
  await run(base)
  const path=f.store+'/networks/'+w.id+'/chain-record.json',record=JSON.parse(fs.readFileSync(path))
  assert.equal(record.binary,f.store+'/assets/'+asset.id+'/payload','setup ignored uploaded binary')
  assert.equal(crypto.createHash('sha256').update(fs.readFileSync(record.binary)).digest('hex'),asset.checksum)
  assert.equal(record.nodes.length,4)
  const declaredGenesis=JSON.parse(fs.readFileSync(record.genesisPath,'utf8'))
  for(const node of record.nodes){
   const databases=fs.readdirSync(node.dataDir,{withFileTypes:true}).filter(e=>e.isDirectory()).map(e=>node.dataDir+'/'+e.name+'/chaindata').filter(dir=>fs.existsSync(dir+'/CURRENT'))
   assert.equal(databases.length,1,'native database missing')
   assert.ok(fs.readdirSync(databases[0]).some(name=>name.startsWith('MANIFEST-')),'native database manifest missing')
   const actual=JSON.parse(execFileSync(record.binary,['--datadir',node.dataDir,'dumpgenesis'],{encoding:'utf8',timeout:20000}))
   assert.equal(actual.config.chainId,declaredGenesis.config.chainId,'uploaded binary initialized a different genesis')
  }
  await run({...base,operation:'node.start',nodeIds:['node1']})
  let result
  for(let n=0;n<200;n++){
   try{const r=await fetch('http://127.0.0.1:'+(13200+i*100),{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({jsonrpc:'2.0',id:1,method:'eth_chainId',params:[]}),signal:AbortSignal.timeout(1000)});result=(await r.json()).result;if(result)break}catch{}
   await new Promise(r=>setTimeout(r,100))
  }
  assert.ok(result,'uploaded native process did not serve RPC')
  const observation=await api('networks/'+w.id+'/observations');assert.ok(observation.nodes.some(n=>n.state==='running'),'actual process was not confirmed')
  observed.push({chain,assetId:asset.id,checksum:asset.checksum,rpcChainId:result})
  await run({...base,operation:'node.stop',nodeIds:['node1']})
  const final=await api('networks/'+w.id+'/observations');assert.ok(final.nodes.every(n=>n.state!=='running'),'owned process not stopped')
 }
 await page.goto(f.url+'/settings')
 await page.getByLabel('등록 자료',{exact:true}).screenshot({path:out+'/asset-library-persisted.png'})
 fs.writeFileSync(out+'/browser.json',JSON.stringify({uploaded:uploaded.map(a=>({id:a.id,chain:a.compatibility.chain,checksum:a.checksum})),observed,jobs:jobs.map(j=>({id:j.id,state:j.state,operation:j.operation})),nativeDatabaseGenesisVerified:true,restartPersisted:true,changedBinaryRejectedBeforeEffects:true,seedAcceptanceAwarded:false},null,2))
 console.log('WEB ASSET JOBS PASS: browser upload, immutable checksums, restart, native DB/RPC/observation/stop for all three chains.')
}finally{await owned.stop()}
