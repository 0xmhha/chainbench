import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import crypto from 'node:crypto'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const key=fs.readFileSync(f.runtime+'/ssh/client','utf8')
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext({viewport:{width:1440,height:1100}})
let session
const digest=path=>crypto.createHash('sha256').update(fs.readFileSync(path)).digest('hex')
async function api(path,method='GET',data,expected=200,idempotency){
 const response=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(idempotency?{'Idempotency-Key':idempotency}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const text=await response.text();assert.equal(response.status(),expected,`${method} ${path}: ${text}`)
 assert.ok(!text.includes(key.trim())&&!text.includes(f.password),'private SSH data leaked')
 return text?JSON.parse(text):undefined
}
async function waitJob(id,state='succeeded'){
 for(let i=0;i<1200;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,state,JSON.stringify(job));return job}await new Promise(r=>setTimeout(r,200))}
 throw new Error('SSH executable replacement timed out')
}
async function run(input,state='succeeded'){
 const plan=await api('plans','POST',input,201),job=await api('jobs','POST',{planId:plan.id},202,plan.id)
 return waitJob(job.id,state)
}
async function genesisHash(){
 for(let i=0;i<100;i++){
  try{const response=await context.request.post('http://127.0.0.1:10800/',{data:{jsonrpc:'2.0',id:1,method:'eth_getBlockByNumber',params:['0x0',false]}});const result=await response.json();if(result.result?.hash)return result.result.hash.toLowerCase()}catch{}
  await new Promise(r=>setTimeout(r,100))
 }
 throw new Error('SSH native node did not serve genesis RPC')
}
try{
 session=await api('bootstrap','POST',{username:'ssh-binary-admin',password:f.password,setupToken:f.setupToken},201)
 const set=await api('documents','POST',{kind:'server-set',name:'SSH replacement pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'owned-ssh',addr:'localhost.'}],slots:4,ports:{p2p:{base:35500,step:10},rpc:{base:10800,step:10}}},ssh:{port:f.ssh.port,known_hosts_file:f.knownHosts}}},201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'SSH replacement paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/artifacts'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},201)
 const w=await api('workspaces','POST',{name:'SSH registered replacement',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},201)
 const credential=await api('credentials','POST',{label:'Owned SSH replacement',kind:'private-key',sshUser:f.ssh.user,privateKey:key},201)
 await api(`workspaces/${w.id}/credential-bindings`,'PUT',{serverRef:'owned-ssh',credentialId:credential.id})
 const base={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],credentialBindings:{'owned-ssh':credential.id},retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'owned-ssh'}}
 await run({...base,operation:'chain.setup',arguments:{...base.arguments,validators:4}})
 for(const id of ['node1','node2'])await run({...base,operation:'node.start',nodeIds:[id]})
 const recordPath=f.store+'/networks/'+w.id+'/chain-record.json',read=()=>JSON.parse(fs.readFileSync(recordPath,'utf8'))
 const initial=read(),selected=initial.nodes.find(n=>n.index===1),sibling=initial.nodes.find(n=>n.index===2),recordBytes=fs.readFileSync(recordPath)
 const selectedMarker=selected.dataDir+'/ssh-retained-data',siblingMarker=sibling.dataDir+'/ssh-retained-data'
 for(const path of [selectedMarker,siblingMarker])fs.writeFileSync(path,'retain SSH native data')
 const preserved=new Map([initial.genesisPath,...initial.nodes.map(n=>n.configPath)].map(path=>[path,digest(path)]))
 const nativeGenesis=await genesisHash()
 const upload=await context.request.post(f.url+'/api/v1/assets',{headers:{'X-CSRF-Token':session.csrfToken},multipart:{kind:'binary',file:{name:'gwbft',mimeType:'application/octet-stream',buffer:fs.readFileSync(f.replacementPath)}}})
 assert.equal(upload.status(),201,await upload.text());const asset=await upload.json();assert.equal(asset.checksum,f.replacementSHA256);assert.equal(asset.compatibility.chain,'wbft')
 const replacement={...base,operation:'node.swap',nodeIds:['node1'],assetRefs:['wbft',asset.id],arguments:{...base.arguments,replacementAssetId:asset.id}}
 const review=await api('plans','POST',replacement,201)
 assert.ok(review.changes.some(v=>v.includes(asset.checksum)&&v.includes(asset.id)))
 const target=f.runtime+'/d/binaries/'+asset.checksum+'/gwbft'
 fs.mkdirSync(target.slice(0,target.lastIndexOf('/')),{recursive:true})
 // This is a loopback SSH fixture: direct writes inject target faults, while
 // production checks, copy, stop, launch and observation use the SSH adapter.
 for(const kind of ['linked-file','occupied-bytes']){
  if(kind==='linked-file')fs.symlinkSync(f.replacementPath,target);else fs.writeFileSync(target,'wrong deployed bytes',{mode:0o755})
  try{
   const failed=await run(replacement,'failed');assert.ok(failed.unresolvedResources.includes(target))
   assert.ok(failed.partialEffects.some(v=>v.includes('has not been stopped')))
   assert.deepEqual(fs.readFileSync(recordPath),recordBytes)
   const observed=await api('networks/'+w.id+'/observations');assert.equal(observed.nodes.find(n=>n.id==='node1').observedPid,selected.pid);assert.equal(observed.nodes.find(n=>n.id==='node2').observedPid,sibling.pid)
  }finally{fs.unlinkSync(target)}
 }
 const page=await context.newPage();await page.goto(f.url+'/chains')
 for(const [label,value] of [['작업 Workspace',w.id],['작업 매니페스트','wbft'],['작업 바이너리','wbft'],['작업 서버 이름','owned-ssh'],['작업 종류','node.swap']])await page.getByLabel(label,{exact:true}).selectOption(value)
 await page.getByRole('button',{name:'노드 상태 확인',exact:true}).click();await page.getByText('node1 · 가동 중',{exact:true}).waitFor()
 await page.getByLabel('작업 노드',{exact:true}).selectOption('node1');await page.getByLabel('노드 교체 바이너리',{exact:true}).selectOption(asset.id)
 const planResponse=page.waitForResponse(r=>r.url()===f.url+'/api/v1/plans'&&r.request().method()==='POST')
 await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click();const planned=await planResponse;assert.equal(planned.status(),201,await planned.text())
 await page.getByLabel('실행 계획',{exact:true}).waitFor();await page.getByLabel('서버 실행 작업',{exact:true}).screenshot({path:out+'/ssh-binary-plan.png'})
 const jobResponse=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
 await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click();const accepted=await jobResponse;assert.equal(accepted.status(),202,await accepted.text());const replaced=await waitJob((await accepted.json()).id)
 const changed=read(),newNode=changed.nodes.find(n=>n.index===1)
 assert.notEqual(newNode.pid,selected.pid);assert.equal(changed.binaries[newNode.binary],target);assert.equal(digest(target),asset.checksum)
 assert.equal(changed.nodes.find(n=>n.index===2).pid,sibling.pid);assert.deepEqual(newNode.args,selected.args);assert.equal(await genesisHash(),nativeGenesis)
 for(const [path,sha] of preserved)assert.equal(digest(path),sha)
 for(const path of [selectedMarker,siblingMarker])assert.equal(fs.readFileSync(path,'utf8'),'retain SSH native data')
 const observed=await api('networks/'+w.id+'/observations'),live=observed.nodes.find(n=>n.id==='node1')
 assert.equal(live.state,'running');assert.equal(live.binaryAssetId,asset.id);assert.equal(live.binarySHA256,asset.checksum);assert.ok(live.supportedControls.includes('node.restart'))
 const restart={...base,operation:'node.restart',nodeIds:['node1']};await run(restart)
 assert.notEqual(read().nodes.find(n=>n.index===1).pid,newNode.pid);assert.equal(read().nodes.find(n=>n.index===2).pid,sibling.pid)
 const back={...replacement,assetRefs:['wbft'],arguments:{...base.arguments,replacementAssetId:'wbft'}};await run(back)
 const restored=read(),restoredNode=restored.nodes.find(n=>n.index===1)
 assert.equal(digest(restored.binaries[restoredNode.binary]),digest(initial.binary));assert.equal(await genesisHash(),nativeGenesis)
 const history=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/process.json','utf8'))
 assert.ok(history.history.some(n=>n.Label==='node1'&&n.PID===newNode.pid));assert.equal(JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/web-node-binaries.json','utf8')).length,2)
 const revokedPlan=await api('plans','POST',replacement,201)
 await api('credentials/'+credential.id,'DELETE',undefined,204);await api('jobs','POST',{planId:revokedPlan.id},403,revokedPlan.id)
 assert.equal(read().nodes.find(n=>n.index===1).pid,restoredNode.pid,'revoked review stopped a retained node')
 const renewed=await api('credentials','POST',{label:'Renewed owned SSH binding',kind:'private-key',sshUser:f.ssh.user,privateKey:key},201)
 await api(`workspaces/${w.id}/credential-bindings`,'PUT',{serverRef:'owned-ssh',credentialId:renewed.id})
 for(const id of ['node1','node2'])await run({...base,credentialBindings:{'owned-ssh':renewed.id},operation:'node.stop',nodeIds:[id]})
 for(const [path,sha] of preserved)assert.equal(digest(path),sha)
 for(const path of [selectedMarker,siblingMarker])assert.equal(fs.readFileSync(path,'utf8'),'retain SSH native data')
 fs.writeFileSync(out+'/browser.json',JSON.stringify({chain:'wbft',workspaceId:w.id,jobId:replaced.id,candidateSHA256:asset.checksum,privateSSHNativeReplacement:true,remoteRegularFileGuard:true,stagingFailurePreservesProcesses:true,reviewedBindingSupportsLaterRestart:true,registeredRollbackPreservesHistory:true,revokedReviewPreservesRetainedNode:true,configGenesisDataAndSiblingPreserved:true,seedAcceptanceAwarded:false},null,2))
 console.log('SSH REGISTERED REPLACEMENT PASS: real private SSH copy, native launch, refused targets, restart and rollback.')
}finally{await owned.stop()}
