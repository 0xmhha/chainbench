import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2), f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(), browser=owned.browser
const context=await browser.newContext(), jobs=[]
let session
async function api(path,method='GET',data,key,expected=200){
  const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
  const text=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${text}`)
  assert.ok(!text.includes(f.password)&&!text.includes(f.setupToken),'bootstrap secret leaked')
  return text?JSON.parse(text):undefined
}
async function waitJob(id){
  for(let i=0;i<600;i++){
    const job=await api('jobs/'+id)
    if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,'succeeded',JSON.stringify(job));jobs.push(job);return job}
    await new Promise(r=>setTimeout(r,200))
  }
  throw new Error('owned resource job timed out')
}
function planInput(w,operation='chain.setup',retention='retain'){
  return {workspaceId:w.id,operation,retention,documentRefs:w.documents,assetRefs:['wbft'],...(operation==='node.stop'?{nodeIds:['node1']}:{}),arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local',validators:4}}
}
async function start(w,operation,retention){
 const plan=await api('plans','POST',planInput(w,operation,retention),undefined,201)
 return waitJob((await api('jobs','POST',{planId:plan.id},plan.id,202)).id)
}
try{
 session=await api('bootstrap','POST',{username:'resource-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 const set=await api('documents','POST',{kind:'server-set',name:'physical fixture ports',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39000,step:10},rpc:{base:11600,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'owned fixture path',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
 const refs=[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]
 const owner=await api('workspaces','POST',{name:'owned workspace',documents:refs},undefined,201)
 const alias=await api('workspaces','POST',{name:'different physical alias',documents:refs},undefined,201)
 const retained=await start(owner,'chain.setup','retain');assert.equal(retained.nodeDisposition,'retained')
 const before=await api('networks');assert.equal(before.items.length,1);assert.equal(before.items[0].workspaceId,owner.id)
 const nodePaths=before.items[0].nodes.map(n=>n.dataPath);assert.equal(nodePaths.length,4)
 for(const dataPath of nodePaths){assert.ok(dataPath.startsWith(f.runtime+'/d/'));assert.ok(fs.existsSync(dataPath),'native setup did not create '+dataPath)}
 const conflicting=await api('plans','POST',planInput(alias),undefined,201)
 await api('jobs','POST',{planId:conflicting.id},conflicting.id,409)
 assert.equal((await api('jobs')).items.length,1,'alias accepted after retained completion')
 assert.equal((await api('networks')).items[0].workspaceId,owner.id,'conflicting alias changed ownership')
 await start(owner,'node.stop','retain')
 const cleaned=await start(owner,'chain.setup','cleanup');assert.equal(cleaned.nodeDisposition,'cleaned')
 assert.equal((await api('networks')).items.find(n=>n.workspaceId===owner.id).nodes.length,0,'cleanup left owned nodes')
 const controlDir=f.store+'/networks/'+owner.id
 const removed=JSON.parse(fs.readFileSync(controlDir+'/chain-record.json','utf8'))
 assert.equal(removed.statePath,'CHAIN/CHAIN_REMOVED');assert.ok(removed.steps.rm.done);assert.equal((removed.nodes||[]).length,0)
 for(const dataPath of nodePaths)assert.ok(!fs.existsSync(dataPath),'cleaned node data survives on disk: '+dataPath)
 // The pre-existing alias plan stays valid because cleanup is not a document edit.
 const accepted=await api('jobs','POST',{planId:conflicting.id},conflicting.id,202)
 await waitJob(accepted.id)
 const after=await api('networks');assert.equal(after.items.filter(n=>n.nodes.length).length,1);assert.equal(after.items.find(n=>n.nodes.length).workspaceId,alias.id)
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:browser.version(),jobs,retainedAliasRejected:true,ownerControlAllowed:true,actualCleanupReleasedAlias:true,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
