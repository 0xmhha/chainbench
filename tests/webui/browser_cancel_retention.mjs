import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser()
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
const terminal=['succeeded','failed','cancelled','interrupted']
async function request(ctx,token,path,method='GET',data,key,expected=200){
 const response=await ctx.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(token?{'X-CSRF-Token':token}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const raw=await response.text();assert.equal(response.status(),expected,`${method} ${path}: ${raw}`)
 assert.ok(!raw.includes(f.password)&&!raw.includes(f.setupToken),'private bootstrap data leaked')
 return raw?JSON.parse(raw):undefined
}
async function user(username){const ctx=await owned.browser.newContext({viewport:{width:1440,height:1100}});const s=await request(ctx,undefined,'auth/login','POST',{username,password:f.password});return {ctx,s,api:(path,method,data,key,expected)=>request(ctx,s.csrfToken,path,method,data,key,expected)}}
async function settle(api,id){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(terminal.includes(job.state))return job;await sleep(200)}throw new Error('job did not settle')}
// The test phase also composes the network, so cancellation waits until the
// recorded nodes are actually running and the case is waiting on them.
async function running(api,id){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(job.state==='running'&&job.phases.some(p=>p.name==='test.run'&&p.state==='running')&&startedNodes())return job;if(terminal.includes(job.state))throw new Error('job ended before it could be cancelled: '+JSON.stringify(job));await sleep(100)}throw new Error('job never reached the engine test phase')}
let networkRecord=null
const startedNodes=()=>{try{const r=JSON.parse(fs.readFileSync(networkRecord,'utf8'));return r.nodes.length===4&&r.nodes.every(n=>n.pid>0&&live(n.pid))}catch{return false}}
const live=pid=>{try{return execFileSync('ps',['-p',String(pid),'-o','command='],{encoding:'utf8'}).includes(f.runtime)}catch{return false}}
const observed={}
try{
 const adminCtx=await owned.browser.newContext()
 const adminSession=await request(adminCtx,undefined,'bootstrap','POST',{username:'cancel-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 const admin=(path,method,data,key,expected)=>request(adminCtx,adminSession.csrfToken,path,method,data,key,expected)
 const accounts={}
 for(const name of ['cancel-operator','other-operator'])accounts[name]=await admin('users','POST',{username:name,password:f.password,role:'operator'},undefined,201)
 const op=await user('cancel-operator'),other=await user('other-operator')
 const paths=root=>({version:1,dataRoot:root,paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}})
 const set=await op.api('documents','POST',{kind:'server-set',name:'cancel pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39100,step:10},rpc:{base:11700,step:10}}}}},undefined,201)
 const config=await op.api('documents','POST',{kind:'workspace-config',name:'cancel paths',contractVersion:'2',assetRefs:[],content:paths(f.runtime+'/d')},undefined,201)
 const w=await op.api('workspaces','POST',{name:'cancel chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const preset={chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}}
 // The long case waits for an unreachable height so cancellation always lands mid-test.
 const long=await op.api('documents','POST',{kind:'case',name:'long cancel case',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'cancel-long',chainPreset:preset,steps:[{do:'waitBlock',on:'node1',target:1000000000,timeout:'5m'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}},undefined,201)
 const quick=await op.api('documents','POST',{kind:'case',name:'quick cleanup case',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'cancel-quick',chainPreset:preset,steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}},undefined,201)
 const c=long
 const control={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local'}}
 const start=async(actor,retention,doc=long)=>{const plan=await actor.api('plans','POST',{...control,retention,operation:'test.run',arguments:{...control.arguments,caseRefs:[{id:doc.id,revision:doc.revision}]}},undefined,201);return (await actor.api('jobs','POST',{planId:plan.id},plan.id,202)).id}
 const stopAll=async actor=>{for(const n of record().nodes.filter(n=>n.pid>0)){const plan=await actor.api('plans','POST',{...control,operation:'node.stop',nodeIds:[n.label]},undefined,201);const j=await settle(actor.api,(await actor.api('jobs','POST',{planId:plan.id},plan.id,202)).id);assert.equal(j.state,'succeeded',JSON.stringify(j))}}
 const record=()=>JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
 networkRecord=f.store+'/networks/'+w.id+'/chain-record.json'

 // 0. A normal end with the default retention keeps the nodes running.
 let id=await start(op,'retain',quick)
 let job=await settle(op.api,id)
 assert.equal(job.state,'succeeded');assert.equal(job.nodeDisposition,'retained',JSON.stringify(job))
 assert.ok(record().nodes.every(n=>n.pid>0&&live(n.pid)),'normal end without cleanup stopped nodes')
 observed.normalEndRetain={id,disposition:job.nodeDisposition}
 await stopAll(op)

 // 1. User cancel with the default retention keeps the nodes and the record.
 id=await start(op,'retain');await running(op.api,id)
 await other.api('jobs/'+id+'/cancel','POST',{},undefined,403)
 observed.otherOperatorCancelRefused=403
 await op.api('jobs/'+id+'/cancel','POST',{},undefined,202)
 job=await settle(op.api,id)
 assert.equal(job.state,'cancelled');assert.equal(job.cancelReason,'user');assert.equal(job.nodeDisposition,'retained',JSON.stringify(job))
 const retained=record();assert.ok(retained.nodes.length===4&&retained.nodes.every(n=>n.pid>0&&live(n.pid)),'cancel without cleanup lost nodes')
 observed.userCancelRetain={id,disposition:job.nodeDisposition,liveNodes:retained.nodes.length}

 // 2. An administrator cancels another user's job with selected cleanup.
 // The retained nodes are stopped explicitly before the next composition.
 await stopAll(op)
 id=await start(op,'cleanup');const pids=await (async()=>{await running(op.api,id);return record().nodes.map(n=>n.pid)})()
 await admin('jobs/'+id+'/cancel','POST',{},undefined,202)
 job=await settle(op.api,id)
 assert.equal(job.state,'cancelled');assert.equal(job.nodeDisposition,'cleaned',JSON.stringify(job))
 assert.ok(pids.every(p=>!live(p)),'cleanup left owned processes running')
 assert.equal(record().statePath,'CHAIN/CHAIN_REMOVED')
 observed.adminCancelCleanup={id,disposition:job.nodeDisposition}

 // 3. Cleanup failure is reported, keeps resources unresolved and is not hidden.
 id=await start(op,'cleanup');await running(op.api,id)
 const nodesDir=fs.readdirSync(f.runtime+'/d/nodes').map(d=>f.runtime+'/d/nodes/'+d)[0]
 fs.chmodSync(nodesDir,0o500)
 let failedCleanup
 try{await op.api('jobs/'+id+'/cancel','POST',{},undefined,202);failedCleanup=await settle(op.api,id)}finally{fs.chmodSync(nodesDir,0o700)}
 assert.equal(failedCleanup.nodeDisposition,'cleanup_failed',JSON.stringify(failedCleanup))
 assert.ok(failedCleanup.unresolvedResources.length>0&&failedCleanup.error?.message,'cleanup failure hidden')
 observed.cleanupFailure={id,unresolved:failedCleanup.unresolvedResources.length}
 // Explicit cleanup afterwards removes what the failed cleanup left.
 id=await start(op,'cleanup',quick);job=await settle(op.api,id)
 assert.equal(job.nodeDisposition,'cleaned',JSON.stringify(job))
 observed.cleanupRetry={id,disposition:job.nodeDisposition}

 // 4. Account deactivation cancels the user's job, keeps the nodes and blocks the account.
 id=await start(other,'cleanup');await running(other.api,id)
 await admin('users/'+accounts['other-operator'].id,'PATCH',{active:false},undefined,200)
 job=await settle(admin,id)
 assert.equal(job.state,'cancelled');assert.equal(job.cancelReason,'account_revoked');assert.equal(job.nodeDisposition,'retained','revocation must not clean up')
 assert.ok(record().nodes.every(n=>n.pid>0&&live(n.pid)),'revocation removed or stopped nodes')
 await request(other.ctx,other.s.csrfToken,'jobs','GET',undefined,undefined,401)
 observed.accountRevocation={id,cancelReason:job.cancelReason,disposition:job.nodeDisposition}

 // 5. An attached network cannot be composed, controlled or cleaned by the Web UI.
 const attachConfig=await op.api('documents','POST',{kind:'workspace-config',name:'attach paths',contractVersion:'2',assetRefs:[],content:{...paths(f.runtime+'/d'),execution:{chain:'attach'}}},undefined,201)
 const attached=await op.api('workspaces','POST',{name:'attached chain',documents:[{id:set.id,revision:set.revision},{id:attachConfig.id,revision:attachConfig.revision}]},undefined,201)
 const refusals={}
 for(const operation of ['chain.setup','node.stop','test.run']){
  const input={workspaceId:attached.id,documentRefs:attached.documents,assetRefs:['wbft'],retention:'retain',operation,arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local',...(operation==='test.run'?{caseRefs:[{id:c.id,revision:c.revision}]}:{})},...(operation==='node.stop'?{nodeIds:['node1']}:{})}
  const response=await op.ctx.request.fetch(f.url+'/api/v1/plans',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':op.s.csrfToken},data:JSON.stringify(input)})
  refusals[operation]=response.status();assert.ok(response.status()>=400&&response.status()<500,operation+' accepted on an attached network: '+response.status())
 }
 const page=await op.ctx.newPage();await page.goto(f.url+'/chains')
 await page.getByLabel('작업 Workspace',{exact:true}).selectOption(attached.id)
 await page.getByLabel('작업 매니페스트',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 바이너리',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
 await page.getByTestId('attached-network').waitFor({timeout:15000})
 assert.ok(await page.getByRole('button',{name:'실행 계획 확인',exact:true}).isDisabled(),'attach plan can still be requested from the UI')
 fs.writeFileSync(out+'/attach-notice.txt',await page.getByTestId('attached-network').textContent())
 await page.screenshot({path:out+'/attach-refused.png',fullPage:true})
 observed.attachRefusals=refusals
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observed,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
