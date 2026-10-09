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
const live=pid=>{try{return execFileSync('ps',['-p',String(pid),'-o','command='],{encoding:'utf8'}).includes(f.runtime)}catch{return false}}
const readRecord=ws=>{try{return JSON.parse(fs.readFileSync(f.store+'/networks/'+ws+'/chain-record.json','utf8'))}catch{return null}}
const observed={}
try{
 const adminCtx=await owned.browser.newContext({viewport:{width:1440,height:1100}})
 const adminSession=await request(adminCtx,undefined,'bootstrap','POST',{username:'parallel-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 const admin=(path,method,data,key,expected)=>request(adminCtx,adminSession.csrfToken,path,method,data,key,expected)
 const operator=await admin('users','POST',{username:'parallel-operator',password:f.password,role:'operator'},undefined,201)
 const opCtx=await owned.browser.newContext();const opSession=await request(opCtx,undefined,'auth/login','POST',{username:'parallel-operator',password:f.password})
 const op=(path,method,data,key,expected)=>request(opCtx,opSession.csrfToken,path,method,data,key,expected)
 async function settle(api,id){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(terminal.includes(job.state))return job;await sleep(200)}throw new Error('job did not settle')}
 const workspace=async(name,root,p2p,rpc)=>{
  const set=await admin('documents','POST',{kind:'server-set',name:name+' pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:p2p,step:10},rpc:{base:rpc,step:10}}}}},undefined,201)
  const config=await admin('documents','POST',{kind:'workspace-config',name:name+' paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:root,paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output-'+name},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
  return {set,config,w:await admin('workspaces','POST',{name,documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)}
 }
 const a=await workspace('owned-a',f.runtime+'/d-a',39100,11700)
 const b=await workspace('same-binary-b',f.runtime+'/d-b',39600,12200)
 const cWs=await workspace('stablenet-c',f.runtime+'/d-c',40100,12700)
 // The alias names the same physical server, data root and ports under another workspace.
 const alias={w:await admin('workspaces','POST',{name:'alias-of-a',documents:[{id:a.set.id,revision:a.set.revision},{id:a.config.id,revision:a.config.revision}]},undefined,201)}
 const preset={chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}}
 // The test stops and restarts node4 itself, then waits so it is still running when others act.
 const internal=await admin('documents','POST',{kind:'case',name:'internal control',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'internal-control',chainPreset:preset,steps:[{do:'stopNode',on:'node4'},{do:'startNode',on:'node4'},{do:'waitBlock',on:'node1',target:1000000000,timeout:'5m'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}},undefined,201)
 const quick=await admin('documents','POST',{kind:'case',name:'independent quick',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'independent-quick',chainPreset:preset,steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}},undefined,201)
 const stable=await admin('documents','POST',{kind:'case',name:'stablenet independent',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'stablenet-independent',chainPreset:{chain:'stablenet',binaries:{default:'gstable'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}},undefined,201)
 const input=(ws,doc,retention='retain',chain='wbft')=>({workspaceId:ws.w.id,documentRefs:ws.w.documents,assetRefs:[chain],retention,operation:'test.run',arguments:{manifestId:chain,assetId:chain,serverRef:'local',caseRefs:[{id:doc.id,revision:doc.revision}]}})
 const planA=await op('plans','POST',input(a,internal),undefined,201)
 const jobA=(await op('jobs','POST',{planId:planA.id},planA.id,202)).id

 // Wait until the test restarted node4 itself: a new PID with the other three unchanged.
 let first=null,restarted=null
 for(let i=0;i<1200&&!restarted;i++){
  const r=readRecord(a.w.id)
  if(r&&r.nodes.length===4&&r.nodes.every(n=>n.pid>0&&live(n.pid))){
   if(!first)first=r.nodes.map(n=>n.pid)
   else if(r.nodes[3].pid!==first[3]&&r.nodes.slice(0,3).every((n,j)=>n.pid===first[j]))restarted=r.nodes.map(n=>n.pid)
  }
  const job=await op('jobs/'+jobA);if(terminal.includes(job.state))throw new Error('internal control test ended early: '+JSON.stringify(job))
  await sleep(100)
 }
 assert.ok(restarted,'test-internal stop/start of node4 was not observed')
 observed.internalControl={before:first[3],after:restarted[3]}

 // An external control of the same network is refused and names the running test and its executor.
 const external=await admin('plans','POST',{workspaceId:a.w.id,documentRefs:a.w.documents,assetRefs:['wbft'],retention:'retain',operation:'node.stop',nodeIds:['node1'],arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local'}},undefined,201)
 const conflicts=(await admin('plans/'+external.id+'/conflicts')).items
 assert.ok(conflicts.some(c=>c.jobId===jobA&&c.actorId===operator.id),'external control conflict does not name the test job and executor: '+JSON.stringify(conflicts))
 await admin('jobs','POST',{planId:external.id},external.id,409)
 const viaAlias=await admin('plans','POST',input(alias,quick),undefined,201)
 const aliasConflicts=(await admin('plans/'+viaAlias.id+'/conflicts')).items
 assert.ok(aliasConflicts.some(c=>c.jobId===jobA&&c.workspaceId===a.w.id),'workspace alias bypassed the physical conflict')
 await admin('jobs','POST',{planId:viaAlias.id},viaAlias.id,409)
 observed.externalRefused={conflicts:conflicts.length,aliasConflicts:aliasConflicts.length}

 // The UI shows the owner before execution and offers no way to run the conflicting plan.
 const page=await adminCtx.newPage();await page.goto(f.url+'/chains')
 await page.getByLabel('작업 Workspace',{exact:true}).selectOption(alias.w.id)
 await page.getByLabel('작업 매니페스트',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 바이너리',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 종류',{exact:true}).selectOption('chain.setup')
 await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
 await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
 const panel=page.getByLabel('자원 충돌',{exact:true});await panel.waitFor({timeout:15000})
 const panelText=await panel.innerText();assert.ok(panelText.includes(jobA)&&panelText.includes(operator.id),'UI omitted the conflicting owner')
 assert.ok(await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).isDisabled(),'UI offered conflicting execution')
 await page.screenshot({path:out+'/physical-conflict.png',fullPage:true})

 // Another path and other ports do not make the same node binary independent:
 // the engine refuses to run it twice on one machine, so the plan names the owner first.
 const planB=await admin('plans','POST',input(b,quick),undefined,201)
 const sameBinary=(await admin('plans/'+planB.id+'/conflicts')).items
 assert.ok(sameBinary.some(c=>c.jobId===jobA&&c.actorId===operator.id&&c.resources.some(r=>r.executable)),'same binary on the machine was not refused before execution: '+JSON.stringify(sameBinary))
 await admin('jobs','POST',{planId:planB.id},planB.id,409)
 observed.sameBinaryRefused={jobId:jobA,executable:sameBinary[0].resources.find(r=>r.executable).executable}

 // A workspace with its own data root, ports and node binary runs to completion meanwhile.
 const planC=await admin('plans','POST',input(cWs,stable,'retain','stablenet'),undefined,201)
 assert.equal((await admin('plans/'+planC.id+'/conflicts')).items.length,0,'independent resources reported as conflicting')
 const jobC=await settle(admin,(await admin('jobs','POST',{planId:planC.id},planC.id,202)).id)
 assert.equal(jobC.state,'succeeded',JSON.stringify(jobC))
 const stillRunning=await op('jobs/'+jobA);assert.equal(stillRunning.state,'running','the other job was disturbed by the independent run')
 observed.parallel={a:jobA,c:jobC.id,chains:['wbft','stablenet']}

 await op('jobs/'+jobA+'/cancel','POST',{},undefined,202)
 const cancelled=await settle(op,jobA);assert.equal(cancelled.nodeDisposition,'retained')
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observed,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
