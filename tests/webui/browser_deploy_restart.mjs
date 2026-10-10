import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import crypto from 'node:crypto'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext({viewport:{width:1440,height:1100}})
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
const hash=b=>crypto.createHash('sha256').update(b).digest('hex')
let session
async function api(path,method='GET',data,key,expected=200){
 const response=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const raw=await response.text();assert.equal(response.status(),expected,`${method} ${path}: ${raw}`)
 assert.ok(!raw.includes(f.password)&&!raw.includes(f.setupToken),'private bootstrap data leaked')
 return raw?JSON.parse(raw):undefined
}
const terminal=['succeeded','failed','cancelled','interrupted']
try{
 session=await api('bootstrap','POST',{username:'deploy-restart-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 const set=await api('documents','POST',{kind:'server-set',name:'deploy restart pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39100,step:10},rpc:{base:11700,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'deploy restart paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
 const w=await api('workspaces','POST',{name:'deploy restart chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const input={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],retention:'retain',operation:'chain.deploy',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local',validators:4}}
 const plan=await api('plans','POST',input,undefined,201)
 const accepted=await api('jobs','POST',{planId:plan.id},plan.id,202)
 // Restart while the deployment is past its file phases but before every node runs.
 let during
 for(let i=0;i<3000;i++){
  during=await api('jobs/'+accepted.id)
  if(terminal.includes(during.state))throw new Error('deployment finished before the restart: '+JSON.stringify(during))
  if(during.phases.some(p=>['deploy','init','start'].includes(p.name)&&p.state==='running'))break
  await sleep(20)
 }
 const runningPhase=during.phases.find(p=>p.state==='running')?.name
 assert.ok(runningPhase,'no running deployment phase observed')
 const recordPath=f.store+'/networks/'+w.id+'/chain-record.json'
 fs.writeFileSync(f.runtime+'/restart.request','restart exclusively owned dashboard during deployment')
 for(let i=0;i<300&&!fs.existsSync(f.runtime+'/restart.response');i++)await sleep(100)
 const restart=JSON.parse(fs.readFileSync(f.runtime+'/restart.response','utf8'));assert.ok(!restart.error&&restart.oldPid!==restart.newPid,JSON.stringify(restart))
 session=undefined
 session=await api('auth/login','POST',{username:'deploy-restart-admin',password:f.password})
 const interrupted=await api('jobs/'+accepted.id)
 assert.equal(interrupted.state,'interrupted',JSON.stringify(interrupted));assert.equal(interrupted.nodeDisposition,'unknown');assert.ok(interrupted.unresolvedResources.length>0,'interrupted deployment hides its resources')
 const recordAfter=fs.existsSync(recordPath)?fs.readFileSync(recordPath):null
 // Nothing resumes on its own: no new job, no phase change, no record change.
 await sleep(15000)
 const jobs=(await api('jobs')).items
 assert.equal(jobs.length,1,'restart automatically executed another job')
 const later=await api('jobs/'+accepted.id)
 assert.deepEqual(later.phases,interrupted.phases,'interrupted deployment continued after restart')
 if(recordAfter)assert.equal(hash(fs.readFileSync(recordPath)),hash(recordAfter),'restart changed the deployment record')
 assert.equal((await api('jobs','POST',{planId:plan.id},plan.id,202)).id,accepted.id,'idempotent replay created an execution')
 // The interrupted deployment keeps its resources claimed against other workspaces.
 const alias=await api('workspaces','POST',{name:'deploy restart alias',documents:w.documents},undefined,201)
 const aliasPlan=await api('plans','POST',{...input,workspaceId:alias.id,documentRefs:alias.documents},undefined,201)
 assert.ok((await api('plans/'+aliasPlan.id+'/conflicts')).items.some(c=>c.jobId===accepted.id),'interrupted deployment released its claims')
 // Actual node state is read again rather than trusted from the record.
 let observed=null
 if(recordAfter&&JSON.parse(recordAfter).nodes?.length){observed=await api('networks/'+w.id+'/observations');assert.ok(observed.nodes.every(n=>n.state!==undefined&&n.observedAt),'node state was not observed')}
 const page=await context.newPage();await page.goto(f.url+'/chains')
 await page.locator('[data-job-id="'+accepted.id+'"]').filter({hasText:'interrupted'}).waitFor({timeout:20000}).catch(async()=>{await page.goto(f.url+'/');await page.getByText('interrupted').first().waitFor({timeout:20000})})
 await page.screenshot({path:out+'/deploy-interrupted.png',fullPage:true})
 // Only an explicit new plan runs the deployment again.
 const again=await api('plans','POST',input,undefined,201)
 assert.notEqual(again.id,plan.id)
 const rerun=await api('jobs','POST',{planId:again.id},again.id,202)
 assert.notEqual(rerun.id,accepted.id)
 let finished
 for(let i=0;i<1500;i++){finished=await api('jobs/'+rerun.id);if(terminal.includes(finished.state))break;await sleep(200)}
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),interruptedPhase:runningPhase,interruptedJob:{id:accepted.id,state:interrupted.state,unresolved:interrupted.unresolvedResources.length},noAutomaticResume:true,claimsKept:true,observedNodes:observed?observed.nodes.map(n=>n.state):[],explicitRerun:{id:rerun.id,state:finished.state,error:finished.error?.message??null},seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
