import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(),browser=owned.browser,context=await browser.newContext({viewport:{width:1440,height:1100}})
const verified=[]
let session
async function api(path,method='GET',data,key,expected=200){
 const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const raw=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${raw}`)
 assert.ok(!raw.includes(f.password)&&!raw.includes(f.setupToken),'bootstrap secret leaked')
 return raw?JSON.parse(raw):undefined
}
async function waitJob(id){
 for(let i=0;i<900;i++){const j=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(j.state)){assert.equal(j.state,'succeeded',JSON.stringify(j));return j}await new Promise(r=>setTimeout(r,200))}
 throw new Error('node-table test timed out')
}
try{
 session=await api('bootstrap','POST',{username:'pinned-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 for(const [i,chain] of ['stablenet','wbft','wemix'].entries()){
  const p2p=44300+i*100,rpc=17000+i*100
  const set=await api('documents','POST',{kind:'server-set',name:chain+' table pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:5,ports:{p2p:{base:p2p,step:10},rpc:{base:rpc,step:10}}}}},undefined,201)
  const config=await api('documents','POST',{kind:'workspace-config',name:chain+' table paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d'+i,paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'+i},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
  const w=await api('workspaces','POST',{name:chain+' table tests',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
  const rows=[{index:3,role:'bp'},{index:1,role:'en',sync:'archive'},{index:2,role:'bp'},{index:5,role:'bp'},{index:4,role:'bp'}]
  const content={schemaVersion:'2',kind:'case',id:chain+'-table-test',chainPreset:{chain,binaries:{default:chain==='stablenet'?'gstable':'gwemix'},topology:{nodes:rows},launch:chain==='wemix'?undefined:{all:{ipcdisable:true}}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}
  const c=await api('documents','POST',{kind:'case',name:content.id,contractVersion:'2',assetRefs:[],content},undefined,201)
  const base={workspaceId:w.id,operation:'test.run',documentRefs:w.documents,assetRefs:[chain],retention:'retain',arguments:{manifestId:chain,assetId:chain,serverRef:'local',caseRefs:[{id:c.id,revision:c.revision}]}}
  const plan=await api('plans','POST',base,undefined,201)
  for(let n=1;n<=5;n++)assert.ok(plan.changes.some(v=>v.includes(`Test placement node${n}=${n===1?'en':'bp'}`)&&v.includes(`P2P ${p2p+(n-1)*10}`)&&v.includes(`RPC ${rpc+(n-1)*10}`)),'review lost actual per-node placement')
  if(i===0){
   for(const field of ['config','key']){
    const bad=structuredClone(content);bad.id='bad-'+field;bad.chainPreset.topology.nodes[1][field]='/outside'
    const doc=await api('documents','POST',{kind:'case',name:bad.id,contractVersion:'2',assetRefs:[],content:bad},undefined,201)
    await api('plans','POST',{...base,arguments:{...base.arguments,caseRefs:[{id:doc.id,revision:doc.revision}]}},undefined,422)
    assert.ok(!fs.existsSync(f.store+'/networks/'+w.id+'/chain-record.json'),'invalid table caused native effects')
   }
  }
  const page=await context.newPage();await page.goto(f.url+'/tests')
  await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
  await page.getByLabel('작업 매니페스트',{exact:true}).selectOption(chain)
  await page.getByLabel('작업 바이너리',{exact:true}).selectOption(chain)
  await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
  await page.getByLabel('실행 케이스 '+c.id,{exact:true}).check()
  await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click();await page.getByLabel('실행 계획',{exact:true}).waitFor()
  assert.ok((await page.getByLabel('실행 계획',{exact:true}).innerText()).includes('Test placement node1=en'),'UI hides endpoint-first placement')
  await page.getByLabel('실행 계획',{exact:true}).screenshot({path:out+'/'+chain+'-table-plan.png'})
  const started=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
  await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click()
  const response=await started;assert.equal(response.status(),202,await response.text());const job=await waitJob((await response.json()).id)
  assert.equal(job.runIds.length,1,'native session missing')
  const record=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
  assert.equal(record.nodes.length,5)
  const observed=await api('networks/'+w.id+'/observations')
  for(const n of record.nodes){
   assert.equal(n.role,n.index===1?'en':'bp');assert.equal(n.p2p,p2p+(n.index-1)*10);assert.equal(n.http,rpc+(n.index-1)*10)
   const probe=observed.nodes.find(v=>v.id==='node'+n.index);assert.equal(probe.state,'running');assert.equal(probe.observedPid,n.pid)
   const dirs=fs.readdirSync(n.dataDir,{withFileTypes:true}).filter(e=>e.isDirectory()).map(e=>n.dataDir+'/'+e.name+'/chaindata').filter(d=>fs.existsSync(d+'/CURRENT'))
   assert.equal(dirs.length,1);assert.ok(fs.readdirSync(dirs[0]).some(v=>v.startsWith('MANIFEST-')))
  }
  assert.equal(record.nodes.find(n=>n.index===1).syncMode,'archive')
  const expectedGenesis=JSON.parse(fs.readFileSync(record.genesisPath,'utf8'))
  const rpcResponse=await context.request.post('http://127.0.0.1:'+rpc+'/',{data:{jsonrpc:'2.0',id:1,method:'eth_chainId',params:[]}})
  assert.equal(rpcResponse.status(),200);assert.equal(Number.parseInt((await rpcResponse.json()).result,16),Number(expectedGenesis.config.chainId))
  const genesisRPC=await context.request.post('http://127.0.0.1:'+rpc+'/',{data:{jsonrpc:'2.0',id:2,method:'eth_getBlockByNumber',params:['0x0',false]}})
  assert.equal(genesisRPC.status(),200);const genesisHash=(await genesisRPC.json()).result.hash.toLowerCase()
  async function update(path,revision,value){
   const response=await context.request.patch(f.url+'/api/v1/'+path,{headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrfToken,'If-Match':`"${revision}"`},data:JSON.stringify(value)})
   assert.equal(response.status(),200,await response.text());return response.json()
  }
  const nextSet=structuredClone(set);nextSet.content.pool.ports.rpc.base+=1000;nextSet.name+=' revised'
  const set2=await update('documents/'+set.id,1,{kind:set.kind,name:nextSet.name,contractVersion:'2',assetRefs:[],content:nextSet.content})
  const nextConfig=structuredClone(config);nextConfig.content.dataRoot=f.runtime+'/unselected'+i
  const config2=await update('documents/'+config.id,1,{kind:config.kind,name:config.name+' revised',contractVersion:'2',assetRefs:[],content:nextConfig.content})
  assert.equal(set2.revision,2);assert.equal(config2.revision,2)
  const controlInput={workspaceId:w.id,operation:'node.reset',documentRefs:w.documents,assetRefs:[chain],nodeIds:['node1'],retention:'retain',arguments:{manifestId:chain,assetId:chain,serverRef:'local',validators:4}}
  const pinnedPlan=await api('plans','POST',controlInput,undefined,201)
  const savedPlan=JSON.parse(fs.readFileSync(f.store+'/jobs.json','utf8')).plans[pinnedPlan.id].prepared.payload
  assert.equal(savedPlan.set.revision,1);assert.equal(savedPlan.config.revision,1)
  assert.equal(savedPlan.target.dataPath,config.content.dataRoot,'review silently used newer target paths')
  if(i===0){
   const rebound=await update('workspaces/'+w.id,1,{name:w.name+' rebound',documents:[{id:set.id,revision:2},{id:config.id,revision:2}]})
   await api('jobs','POST',{planId:pinnedPlan.id},pinnedPlan.id,409)
   const untouched=await api('networks/'+w.id+'/observations');assert.ok(untouched.nodes.every(n=>n.state==='running'),'rejected stale review changed node processes')
   await update('workspaces/'+w.id,rebound.revision,{name:w.name,documents:w.documents})
  }
  assert.ok(!fs.existsSync(nextConfig.content.dataRoot),'unselected newer target was materialized')

  const endpoint=record.nodes.find(n=>n.index===1),producer=record.nodes.find(n=>n.index===2)
  const marker=endpoint.dataDir+'/reset-old-data',sibling=producer.dataDir+'/reset-preserved-data'
  fs.writeFileSync(marker,'old endpoint data');fs.writeFileSync(sibling,'preserved producer data')
  const denied=await api('plans','POST',{...controlInput,nodeIds:['node2']},undefined,422)
  assert.ok(fs.existsSync(marker)&&fs.existsSync(sibling),'rejected producer plan changed data')
  await page.goto(f.url+'/chains')
  await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
  await page.getByLabel('작업 매니페스트',{exact:true}).selectOption(chain)
  await page.getByLabel('작업 바이너리',{exact:true}).selectOption(chain)
  await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
  await page.getByLabel('작업 종류',{exact:true}).selectOption('node.reset')
  assert.equal(await page.getByLabel('작업 노드',{exact:true}).locator('option[value="node2"]').count(),0,'UI offers producer reset')
  await page.getByLabel('작업 노드',{exact:true}).selectOption('node1')
  await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click();await page.getByLabel('실행 계획',{exact:true}).waitFor()
  assert.ok((await page.getByLabel('실행 계획',{exact:true}).innerText()).includes('genesis'),'reset plan omitted data replacement')
  const resetResponse=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
  await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click()
  const resetStart=await resetResponse;assert.equal(resetStart.status(),202,await resetStart.text())
  const reset=await waitJob((await resetStart.json()).id)
  const resetRecord=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
  assert.equal(resetRecord.nodes.find(n=>n.index===1).pid||0,0,'reset endpoint did not remain stopped')
  assert.ok(!fs.existsSync(marker),'reset retained endpoint data');assert.equal(fs.readFileSync(sibling,'utf8'),'preserved producer data')
  for(const n of resetRecord.nodes.filter(n=>n.index!==1))assert.equal(n.pid,record.nodes.find(p=>p.index===n.index).pid,'reset changed sibling PID')
  const resetGenesis=JSON.parse(execFileSync(record.binary,['--datadir',endpoint.dataDir,'dumpgenesis'],{encoding:'utf8',timeout:20000}))
  assert.equal(Number(resetGenesis.config.chainId),Number(expectedGenesis.config.chainId))
  for(const key of ['LastBlock','LastHeader']){
   const head=execFileSync(record.binary,['--datadir',endpoint.dataDir,'db','get',key],{encoding:'utf8',timeout:20000})
   assert.equal(head.match(/: (0x[0-9a-f]{64})/i)?.[1].toLowerCase(),genesisHash,'reset database head is not genesis')
  }
  const aliveAfter=await api('networks/'+w.id+'/observations');for(const n of aliveAfter.nodes.filter(n=>n.id!=='node1'))assert.equal(n.state,'running')
  const unknownRecordPath=f.store+'/networks/'+w.id+'/chain-record.json',knownStoppedBytes=fs.readFileSync(unknownRecordPath)
  try{
   const unknownRecord=JSON.parse(knownStoppedBytes);unknownRecord.nodes.find(n=>n.index===1).args=[]
   fs.writeFileSync(unknownRecordPath,JSON.stringify(unknownRecord))
   const unknownObservation=(await api('networks/'+w.id+'/observations')).nodes.find(n=>n.id==='node1')
   assert.equal(unknownObservation.state,'unknown');assert.equal(unknownObservation.supportedControls.length,0)
   await api('plans','POST',controlInput,undefined,409)
  }finally{fs.writeFileSync(unknownRecordPath,knownStoppedBytes)}
  const restartPlan=await api('plans','POST',{...controlInput,operation:'node.start'},undefined,201)
  const restart=await api('jobs','POST',{planId:restartPlan.id},restartPlan.id,202);await waitJob(restart.id)
  const relaunched=await api('networks/'+w.id+'/observations');assert.equal(relaunched.nodes.find(n=>n.id==='node1').state,'running')
  await page.getByLabel('서버 실행 작업',{exact:true}).screenshot({path:out+'/'+chain+'-reset.png'})
  // Native genesis reads require the DB lock. Observe live processes first,
  // then stop each owned node explicitly before inspecting its persisted DB.
  for(const n of record.nodes){
   const control=await api('plans','POST',{workspaceId:w.id,operation:'node.stop',documentRefs:w.documents,assetRefs:[chain],nodeIds:['node'+n.index],retention:'retain',arguments:{manifestId:chain,assetId:chain,serverRef:'local',validators:4}},undefined,201)
   const stopped=await api('jobs','POST',{planId:control.id},control.id,202);await waitJob(stopped.id)
   const genesis=JSON.parse(execFileSync(record.binary,['--datadir',n.dataDir,'dumpgenesis'],{encoding:'utf8',timeout:20000}));assert.equal(Number(genesis.config.chainId),Number(expectedGenesis.config.chainId))
  }
  const h=(await api('history?caseId='+content.id)).items.filter(v=>v.summary.kind==='engine-session');assert.equal(h.length,1);assert.equal(h[0].state,'succeeded')
  const detail=await api('history/'+h[0].id);assert.equal(detail.summary.counts.pass,1);assert.equal(detail.summary.counts.skip||0,0)
  verified.push({chain,jobId:job.id,runIds:job.runIds,nodes:record.nodes.map(n=>({index:n.index,role:n.role,p2p:n.p2p,http:n.http,syncMode:n.syncMode})),historyId:h[0].id})
  await page.close()
 }
 fs.writeFileSync(out+'/browser.json',JSON.stringify({verified,pinnedOlderDocumentsExecuted:true,workspaceRebindInvalidatesReview:true,unselectedNewerTargetsUntouched:true,nativeNonProducerReset:true,nativeHeadAtGenesis:true,producerResetRefused:true,siblingNodesPreserved:true,resetLeftStoppedAndRelaunched:true,unknownPIDResetRefused:true,realPassingSessionsWithoutSkips:true,seedAcceptanceAwarded:false},null,2))
 console.log('PINNED CONTROLS PASS: older workspace inputs execute on three native chains; rebound reviews are refused.')
}finally{await owned.stop()}
