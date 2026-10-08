import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import crypto from 'node:crypto'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext({viewport:{width:1440,height:1100}})
let session
const sha=path=>crypto.createHash('sha256').update(fs.readFileSync(path)).digest('hex')
function treeSHA(dir){
 const files=[]
 function visit(base){for(const entry of fs.readdirSync(base,{withFileTypes:true}).sort((a,b)=>a.name.localeCompare(b.name))){const path=base+'/'+entry.name;if(entry.isDirectory())visit(path);else{assert.ok(entry.isFile());files.push([path.slice(dir.length+1),sha(path)])}}}
 visit(dir);return crypto.createHash('sha256').update(JSON.stringify(files)).digest('hex')
}
async function api(path,method='GET',data,key,expected=200){
 const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const raw=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${raw}`)
 assert.ok(!raw.includes(f.password)&&!raw.includes(f.setupToken),'private bootstrap data leaked')
 return raw?JSON.parse(raw):undefined
}
async function waitJob(id,state='succeeded'){
 for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,state,JSON.stringify(job));return job}await new Promise(r=>setTimeout(r,200))}
 throw new Error('owned binary replacement timed out')
}
async function genesisHash(){
 for(let i=0;i<100;i++){
  try{const response=await context.request.post('http://127.0.0.1:17600/',{data:{jsonrpc:'2.0',id:1,method:'eth_getBlockByNumber',params:['0x0',false]}});const value=await response.json();if(response.status()===200&&value.result?.hash)return value.result.hash.toLowerCase()}catch{}
  await new Promise(r=>setTimeout(r,200))
 }
 throw new Error('selected native node RPC unavailable after replacement')
}
try{
 session=await api('bootstrap','POST',{username:'binary-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 const upload=await context.request.post(f.url+'/api/v1/assets',{headers:{'X-CSRF-Token':session.csrfToken},multipart:{kind:'binary',file:{name:'gwbft',mimeType:'application/octet-stream',buffer:fs.readFileSync(f.replacementPath)}}})
 assert.equal(upload.status(),201,await upload.text());const asset=await upload.json()
 assert.equal(asset.checksum,f.replacementSHA256);assert.equal(asset.compatibility.chain,'wbft');assert.equal(asset.compatibility.commit,f.replacementCommit)
 const set=await api('documents','POST',{kind:'server-set',name:'binary replacement pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:5,ports:{p2p:{base:44700,step:10},rpc:{base:17600,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'binary replacement paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
 const w=await api('workspaces','POST',{name:'owned native replacement',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const c=await api('documents','POST',{kind:'case',name:'wbft-binary-replacement',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'wbft-binary-replacement',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{nodes:[{index:1,role:'en'},...[2,3,4,5].map(index=>({index,role:'bp'}))]},launch:{all:{ipcdisable:true}}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}},undefined,201)
 const base={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local'}}
 const setup=await api('plans','POST',{...base,operation:'test.run',arguments:{...base.arguments,caseRefs:[{id:c.id,revision:c.revision}]}},undefined,201)
 const initialJob=await api('jobs','POST',{planId:setup.id},setup.id,202);await waitJob(initialJob.id)
 const recordPath=f.store+'/networks/'+w.id+'/chain-record.json',read=()=>JSON.parse(fs.readFileSync(recordPath,'utf8'))
 const initial=read(),selected=initial.nodes.find(n=>n.index===1),recordBytes=fs.readFileSync(recordPath)
 const keyDigest=treeSHA(initial.keysDir)
 const siblingMarkers=initial.nodes.filter(n=>n.index!==1).map(n=>n.dataDir+'/binary-sibling-data')
 for(const path of siblingMarkers)fs.writeFileSync(path,'retain sibling data')
 const marker=selected.dataDir+'/binary-preserved-data';fs.writeFileSync(marker,'retain selected data')
 const preservedFiles=new Map([initial.genesisPath,...initial.nodes.map(n=>n.configPath)].map(path=>[path,sha(path)]))
 const beforeObserved=await api('networks/'+w.id+'/observations');assert.ok(beforeObserved.nodes.every(n=>n.state==='running'))
 const initialGenesisHash=await genesisHash()
 const replacement={...base,operation:'node.swap',nodeIds:['node1'],assetRefs:['wbft',asset.id],arguments:{...base.arguments,replacementAssetId:asset.id}}
 const review=await api('plans','POST',replacement,undefined,201)
 assert.ok(review.changes.some(v=>v.includes(asset.id)&&v.includes(asset.checksum)),'review omits candidate identity')
 const invalid=[{...replacement,assetRefs:['wbft'],arguments:{...base.arguments,replacementAssetId:'wbft'}},{...replacement,arguments:{...base.arguments,replacementAssetId:'wrong-wbft'}},{...replacement,operation:'node.stop'}]
 for(const request of invalid)await api('plans','POST',request,undefined,422)
 // Changing registered source after review cannot stop the current process.
 const source=f.store+'/assets/'+asset.id+'/payload',sourceBytes=fs.readFileSync(source)
 try{
  fs.writeFileSync(source,Buffer.concat([sourceBytes,Buffer.from('changed')]))
  await api('plans','POST',replacement,undefined,422)
  await api('jobs','POST',{planId:review.id},review.id,422)
  assert.deepEqual(fs.readFileSync(recordPath),recordBytes)
 }finally{fs.writeFileSync(source,sourceBytes)}
 const target=f.runtime+'/d/binaries/'+asset.checksum+'/gwbft'
 fs.mkdirSync(target.slice(0,target.lastIndexOf('/')),{recursive:true})
 // A linked or occupied destination fails staging before any process stop.
 for(const kind of ['symlink','wrong-bytes']){
  if(kind==='symlink')fs.symlinkSync(f.replacementPath,target);else fs.writeFileSync(target,'unreviewed occupied bytes',{mode:0o755})
  try{
   const plan=await api('plans','POST',replacement,undefined,201),accepted=await api('jobs','POST',{planId:plan.id},plan.id,202)
   const failed=await waitJob(accepted.id,'failed')
   assert.ok(failed.unresolvedResources.includes(target),'failed staging omitted unresolved executable')
   assert.ok(failed.partialEffects.some(v=>v.includes('has not been stopped')),'failure claims complete replacement')
   assert.deepEqual(fs.readFileSync(recordPath),recordBytes,'staging failure changed record')
   const live=await api('networks/'+w.id+'/observations');assert.ok(live.nodes.every(n=>n.state==='running'))
   assert.ok(!fs.existsSync(f.store+'/networks/'+w.id+'/web-node-binaries.json'),'failed bytes authorized binding')
  }finally{fs.unlinkSync(target)}
 }
 const page=await context.newPage();await page.goto(f.url+'/chains')
 for(const [label,value] of [['작업 Workspace',w.id],['작업 매니페스트','wbft'],['작업 바이너리','wbft'],['작업 서버 이름','local'],['작업 종류','node.swap']])await page.getByLabel(label,{exact:true}).selectOption(value)
 await page.getByRole('button',{name:'노드 상태 확인',exact:true}).click();await page.getByText('node1 · 가동 중',{exact:true}).waitFor()
 await page.getByLabel('작업 노드',{exact:true}).selectOption('node1')
 const choices=page.getByLabel('노드 교체 바이너리',{exact:true})
 assert.equal(await choices.locator('option[value="wbft"]').count(),0,'UI offers unchanged executable')
 await choices.selectOption(asset.id)
 const responsePromise=page.waitForResponse(r=>r.url()===f.url+'/api/v1/plans'&&r.request().method()==='POST')
 await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
 const reviewed=await responsePromise;assert.equal(reviewed.status(),201,await reviewed.text())
 assert.equal(reviewed.request().postDataJSON().arguments.replacementAssetId,asset.id)
 assert.equal(Object.hasOwn(reviewed.request().postDataJSON().arguments,'configOverrides'),false,'binary-only UI changed config')
 await page.getByLabel('실행 계획',{exact:true}).waitFor();assert.ok((await page.getByLabel('실행 계획',{exact:true}).innerText()).includes(asset.checksum))
 await page.getByLabel('서버 실행 작업',{exact:true}).screenshot({path:out+'/binary-plan.png'})
 const acceptedPromise=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
 await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click()
 const accepted=await acceptedPromise;assert.equal(accepted.status(),202,await accepted.text());const replaced=await waitJob((await accepted.json()).id)
 const changed=read(),newNode=changed.nodes.find(n=>n.index===1)
 assert.notEqual(newNode.pid,selected.pid);assert.equal(changed.binaries[newNode.binary],target);assert.equal(sha(target),asset.checksum)
 assert.deepEqual(newNode.args,selected.args,'binary-only replacement changed launch arguments')
 for(const [path,digest] of preservedFiles)assert.equal(sha(path),digest,'replacement changed config or genesis')
 for(const n of changed.nodes.filter(n=>n.index!==1))assert.equal(n.pid,initial.nodes.find(v=>v.index===n.index).pid,'replacement changed sibling PID')
 assert.equal(fs.readFileSync(marker,'utf8'),'retain selected data')
 assert.equal(await genesisHash(),initialGenesisHash,'replacement changed native database genesis')
 assert.equal(treeSHA(changed.keysDir),keyDigest);for(const path of siblingMarkers)assert.equal(fs.readFileSync(path,'utf8'),'retain sibling data')
 const observed=await api('networks/'+w.id+'/observations'),ownedNode=observed.nodes.find(n=>n.id==='node1')
 assert.equal(ownedNode.state,'running');assert.equal(ownedNode.observedPid,newNode.pid);assert.equal(ownedNode.binaryAssetId,asset.id);assert.equal(ownedNode.binarySHA256,asset.checksum)
 assert.ok(['node.start','node.stop','node.restart','node.swap','node.reset'].every(c=>ownedNode.supportedControls.includes(c)),'reviewed executable lost controls')
 const bindingPath=f.store+'/networks/'+w.id+'/web-node-binaries.json',bindings=fs.readFileSync(bindingPath)
 assert.equal(fs.statSync(bindingPath).mode&0o777,0o600)
 const restart={...base,operation:'node.restart',nodeIds:['node1']},restartReview=await api('plans','POST',restart,undefined,201)
 const changedBytes=fs.readFileSync(recordPath)
 for(const field of ['binaryChains','genesisPaths','genesisConfigPaths']){
  const invalid=structuredClone(changed);invalid[field]={...invalid[field],[newNode.binary]:field==='binaryChains'?'wemix':'/outside/unreviewed'}
  try{fs.writeFileSync(recordPath,JSON.stringify(invalid));await api('plans','POST',restart,undefined,409)}finally{fs.writeFileSync(recordPath,changedBytes)}
 }
 try{
  fs.writeFileSync(bindingPath,'[]')
  await api('plans','POST',restart,undefined,409);await api('jobs','POST',{planId:restartReview.id},restartReview.id,409)
  const refused=await api('networks/'+w.id+'/observations'),node=refused.nodes.find(n=>n.id==='node1')
  assert.equal(node.state,'ownership_mismatch');assert.deepEqual(node.supportedControls,[]);assert.equal(read().nodes.find(n=>n.index===1).pid,newNode.pid)
 }finally{fs.writeFileSync(bindingPath,bindings)}
 // Durable binding survives dashboard restart without changing node processes.
 fs.writeFileSync(f.runtime+'/restart.request','restart this owned server')
 for(let i=0;i<400&&!fs.existsSync(f.runtime+'/restart.response');i++)await new Promise(r=>setTimeout(r,100))
 assert.ok(fs.existsSync(f.runtime+'/restart.response'),'fixture dashboard did not restart')
 const restartResult=JSON.parse(fs.readFileSync(f.runtime+'/restart.response','utf8'));assert.ok(!restartResult.error,restartResult.error)
 session=await api('auth/login','POST',{username:'binary-admin',password:f.password})
 const afterServer=await api('networks/'+w.id+'/observations');assert.equal(afterServer.nodes.find(n=>n.id==='node1').observedPid,newNode.pid)
 const bounce=await api('plans','POST',restart,undefined,201),bounceJob=await api('jobs','POST',{planId:bounce.id},bounce.id,202);await waitJob(bounceJob.id)
 assert.notEqual(read().nodes.find(n=>n.index===1).pid,newNode.pid);assert.equal(read().binaries[read().nodes.find(n=>n.index===1).binary],target)
 const back={...replacement,assetRefs:['wbft'],arguments:{...base.arguments,replacementAssetId:'wbft'}}
 const rollback=await api('plans','POST',back,undefined,201),rollbackJob=await api('jobs','POST',{planId:rollback.id},rollback.id,202);await waitJob(rollbackJob.id)
 const restored=read(),restoredNode=restored.nodes.find(n=>n.index===1),restoredPath=restored.binaries[restoredNode.binary]
 assert.equal(await genesisHash(),initialGenesisHash,'rollback changed native database genesis')
 assert.equal(sha(restoredPath),(await api('manifest-assets')).items.find(a=>a.id==='wbft').sha256)
 assert.equal(JSON.parse(fs.readFileSync(bindingPath,'utf8')).length,2,'rollback removed prior reviewed binding')
 const history=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/process.json','utf8'))
 assert.ok(history.history.some(n=>n.Label==='node1'&&n.PID===selected.pid));assert.ok(history.history.some(n=>n.Label==='node1'&&n.PID===newNode.pid))
 for(const [path,digest] of preservedFiles)assert.equal(sha(path),digest)
 assert.equal(fs.readFileSync(marker,'utf8'),'retain selected data')
 assert.equal(treeSHA(restored.keysDir),keyDigest);for(const path of siblingMarkers)assert.equal(fs.readFileSync(path,'utf8'),'retain sibling data')
 for(const n of restored.nodes.filter(n=>n.index!==1))assert.equal(n.pid,initial.nodes.find(v=>v.index===n.index).pid)
 const combinedInput={...replacement,arguments:{...replacement.arguments,configOverrides:{metricsHost:'127.0.0.1'}}}
 const combined=await api('plans','POST',combinedInput,undefined,201),combinedJob=await api('jobs','POST',{planId:combined.id},combined.id,202);await waitJob(combinedJob.id)
 const combinedState=read(),combinedNode=combinedState.nodes.find(n=>n.index===1)
 assert.equal(combinedState.binaries[combinedNode.binary],target);assert.equal(combinedNode.args[combinedNode.args.indexOf('--metrics.addr')+1],'127.0.0.1')
 assert.ok(fs.readFileSync(combinedNode.configPath,'utf8').includes('HTTP = "127.0.0.1"'))
 assert.equal(sha(initial.genesisPath),preservedFiles.get(initial.genesisPath));assert.equal(await genesisHash(),initialGenesisHash)
 for(const n of combinedState.nodes.filter(n=>n.index!==1)){assert.equal(n.pid,initial.nodes.find(v=>v.index===n.index).pid);assert.equal(sha(n.configPath),preservedFiles.get(n.configPath))}
 assert.equal(treeSHA(combinedState.keysDir),keyDigest);assert.equal(fs.readFileSync(marker,'utf8'),'retain selected data')
 for(const n of restored.nodes){const plan=await api('plans','POST',{...base,operation:'node.stop',nodeIds:['node'+n.index]},undefined,201),job=await api('jobs','POST',{planId:plan.id},plan.id,202);await waitJob(job.id)}
 const final=await api('networks/'+w.id+'/observations');assert.ok(final.nodes.every(n=>n.state==='stopped'))
 fs.writeFileSync(out+'/browser.json',JSON.stringify({chain:'wbft',workspaceId:w.id,jobId:replaced.id,candidateSHA256:asset.checksum,registeredNativeReplacement:true,binaryOnlyConfigurationPreserved:true,combinedConfigAndBinaryReplacement:true,nativeDatabaseGenesisPreserved:true,stagingFailuresPreserveProcesses:true,sourceTamperingRefused:true,bindingTamperingRefused:true,protocolAndGenesisMappingRefused:true,durableBindingAfterServerRestart:true,laterRestartAndRollback:true,siblingProcessesAndDataPreserved:true,retainedExecutableAndProcessHistory:true,seedAcceptanceAwarded:false},null,2))
 console.log('REGISTERED BINARY REPLACEMENT PASS: reviewed native bytes, one owned node, durable controls and rollback.')
}finally{await owned.stop()}
