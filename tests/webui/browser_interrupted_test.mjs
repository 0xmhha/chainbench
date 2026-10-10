import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import {createHash} from 'node:crypto'
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
const pause=()=>new Promise(r=>setTimeout(r,200))
async function waitJob(id){
 for(let i=0;i<600;i++){
  const job=await api('jobs/'+id)
  if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){jobs.push(job);return job}
  await pause()
 }
 throw new Error('owned job timed out')
}
const hash=b=>createHash('sha256').update(b).digest('hex')
try{
 session=await api('bootstrap','POST',{username:'restart-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 const set=await api('documents','POST',{kind:'server-set',name:'restart native pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39400,step:10},rpc:{base:11700,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'restart owned paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
 const refs=[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]
 const w=await api('workspaces','POST',{name:'restart owned network',documents:refs},undefined,201)
 const alias=await api('workspaces','POST',{name:'restart physical alias',documents:refs},undefined,201)
 const c=await api('documents','POST',{kind:'case',name:'long native test',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'restart-wait',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{do:'waitBlock',on:'node1',target:1000000000,timeout:'5m'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}},undefined,201)
 const input={workspaceId:w.id,operation:'test.run',documentRefs:refs,assetRefs:['wbft'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local',caseRefs:[{id:c.id,revision:c.revision}]}}
 const plan=await api('plans','POST',input,undefined,201)
 const accepted=await api('jobs','POST',{planId:plan.id},plan.id,202)
 const recordPath=f.store+'/networks/'+w.id+'/chain-record.json'
 let record, before
 for(let i=0;i<600;i++){
  before=await api('jobs/'+accepted.id)
  assert.equal(before.state,'running','test finished before restart fixture')
  if(fs.existsSync(recordPath)){
   record=JSON.parse(fs.readFileSync(recordPath))
   if(record.nodes?.length===4&&record.nodes.every(n=>n.pid>0))break
  }
  await pause()
 }
 assert.ok(record?.nodes.every(n=>n.pid>0),'actual nodes were not recorded')
 let height
 for(let i=0;i<300;i++){
  try{
   const rpc=await fetch('http://127.0.0.1:11700/',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({jsonrpc:'2.0',id:1,method:'eth_blockNumber',params:[]})})
   height=(await rpc.json()).result;if(height)break
  }catch{}
  await pause()
 }
 assert.ok(height,'fixture nodes never served RPC')
 const recordBefore=fs.readFileSync(recordPath), dirStats=record.nodes.map(n=>({path:n.dataDir,inode:fs.statSync(n.dataDir).ino,pid:n.pid}))
 fs.writeFileSync(f.runtime+'/restart.request','restart exclusively owned dashboard')
 for(let i=0;i<200&&!fs.existsSync(f.runtime+'/restart.response');i++)await pause()
 const restart=JSON.parse(fs.readFileSync(f.runtime+'/restart.response'));assert.ok(!restart.error,JSON.stringify(restart));assert.notEqual(restart.oldPid,restart.newPid)
 session=undefined
 await api('jobs/'+accepted.id,'GET',undefined,undefined,401)
 session=await api('auth/login','POST',{username:'restart-admin',password:f.password})
 const interrupted=await api('jobs/'+accepted.id)
 assert.equal(interrupted.state,'interrupted');assert.equal(interrupted.nodeDisposition,'unknown');assert.ok(interrupted.unresolvedResources.length>0)
 assert.equal((await api('jobs','POST',{planId:plan.id},plan.id,202)).id,accepted.id,'idempotent replay created an execution')
 assert.equal((await api('jobs')).items.length,1,'restart automatically executed another job')
 assert.equal(hash(fs.readFileSync(recordPath)),hash(recordBefore),'restart changed the owned launch record')
 for(const n of dirStats)assert.equal(fs.statSync(n.path).ino,n.inode,'restart recreated node data')
 const observed=await api('networks/'+w.id+'/observations')
 for(const n of dirStats){const live=observed.nodes.find(v=>v.pid===n.pid);assert.equal(live?.state,'running');assert.equal(live.observedPid,n.pid)}
 const page=await context.newPage();await page.goto(f.url+'/chains')
 await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
 await page.getByRole('button',{name:'노드 상태 확인',exact:true}).click()
 await page.getByLabel('노드 실제 관측',{exact:true}).waitFor();assert.ok((await page.getByLabel('노드 실제 관측',{exact:true}).innerText()).includes('가동 중'))
 assert.ok((await page.getByLabel('서버 실행 작업',{exact:true}).innerText()).includes('interrupted'),'UI lost the interrupted job')
 await page.screenshot({path:out+'/restart-observation.png',fullPage:true})
 const conflictInput={workspaceId:alias.id,operation:'chain.setup',documentRefs:refs,assetRefs:['wbft'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local',validators:4}}
 const conflict=await api('plans','POST',conflictInput,undefined,201)
 assert.ok((await api('plans/'+conflict.id+'/conflicts')).items.some(v=>v.jobId===interrupted.id))
 await api('jobs','POST',{planId:conflict.id},conflict.id,409)
 const control=await api('plans','POST',{...conflictInput,workspaceId:w.id,operation:'node.stop',nodeIds:['node1']},undefined,201)
 assert.notEqual(control.id,plan.id)
 const explicit=await api('jobs','POST',{planId:control.id},control.id,202)
 assert.notEqual(explicit.id,accepted.id)
 assert.equal((await waitJob(explicit.id)).state,'succeeded')
 assert.equal((await api('jobs/'+accepted.id)).state,'interrupted','explicit recovery overwrote interrupted job')
 const after=JSON.parse(fs.readFileSync(recordPath));assert.equal(after.nodes.find(n=>n.index===1).pid??0,0)
 for(const n of dirStats)assert.equal(fs.statSync(n.path).ino,n.inode,'explicit stop reset retained data')
 for(const nodeId of ['node2','node3','node4']){
  const stop=await api('plans','POST',{...conflictInput,workspaceId:w.id,operation:'node.stop',nodeIds:[nodeId]},undefined,201)
  assert.equal((await waitJob((await api('jobs','POST',{planId:stop.id},stop.id,202)).id)).state,'succeeded')
 }
 const short=await api('documents','POST',{kind:'case',name:'explicit restart result',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'explicit-restart',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}},undefined,201)
 const rerunPlan=await api('plans','POST',{...input,arguments:{...input.arguments,caseRefs:[{id:short.id,revision:short.revision}]}},undefined,201)
 assert.notEqual(rerunPlan.id,plan.id)
 const rerun=await api('jobs','POST',{planId:rerunPlan.id},rerunPlan.id,202)
 assert.notEqual(rerun.id,accepted.id)
 const finished=await waitJob(rerun.id);assert.equal(finished.state,'succeeded');assert.equal(finished.runIds.length,1)
 assert.deepEqual(await api('jobs/'+accepted.id),interrupted,'explicit rerun overwrote the interrupted result')
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:browser.version(),jobs,interrupted,observed,restart,recordHash:hash(recordBefore),dirStats,idempotentReplayDidNotResume:true,noAutomaticReset:true,newPlanAndJobForExplicitControl:true,explicitNativeTestRerun:true,aliasStillExcluded:true,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
