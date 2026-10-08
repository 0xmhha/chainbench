import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
const [fixturePath,out]=process.argv.slice(2), f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(), browser=owned.browser
const context=await browser.newContext({viewport:{width:1440,height:1050}}), jobs=[], applied=[]
let session
async function api(path,method='GET',data,key,expected=200,revision){
 const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(key?{'Idempotency-Key':key}:{}),...(revision?{'If-Match':String(revision)}:{})},data:data===undefined?undefined:JSON.stringify(data)})
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
 throw new Error('preset setup timed out')
}
function document(chain,chainId,table){return {kind:'chain-preset',name:chain+' saved configuration',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'chain-preset',id:chain+'-reviewed',chain,topology:table?{nodes:[{index:1,role:'en',sync:'archive'},...Array.from({length:4},(_,i)=>({index:i+2,role:'bp'}))]}:{bp:4,en:1,syncMode:'archive'},genesis:{overlay:{config:{chainId}}},launch:{all:{maxpeers:'40',cache:'64'},node1:{maxpeers:'50'}},config:{node1:{httpHost:'0.0.0.0'}}}}}
try{
 session=await api('bootstrap','POST',{username:'preset-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 for(const [i,chain] of ['wbft','stablenet','wemix'].entries()){
  const set=await api('documents','POST',{kind:'server-set',name:chain+' preset pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:5,ports:{p2p:{base:39700+i*100,step:10},rpc:{base:11900+i*100,step:10}}}}},undefined,201)
  const config=await api('documents','POST',{kind:'workspace-config',name:chain+' preset paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d'+i,paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'+i},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
  const w=await api('workspaces','POST',{name:chain+' preset workspace',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
  const first=await api('documents','POST',document(chain,900+i,i===1),undefined,201)
  const base={workspaceId:w.id,operation:'chain.setup',documentRefs:w.documents,assetRefs:[chain],retention:'retain',arguments:{manifestId:chain,assetId:chain,serverRef:'local',chainPresetRef:{id:first.id,revision:first.revision}}}
  const oldPlan=await api('plans','POST',base,undefined,201)
  const revised=await api('documents/'+first.id,'PATCH',document(chain,910+i,i===1),undefined,200,first.revision)
  await api('jobs','POST',{planId:oldPlan.id},oldPlan.id,409)
  base.arguments.chainPresetRef={id:revised.id,revision:revised.revision}
  let accepted
  if(i===0){
   const page=await context.newPage();await page.goto(f.url+'/chains')
   await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
   await page.getByLabel('작업 매니페스트',{exact:true}).selectOption(chain)
   await page.getByLabel('작업 바이너리',{exact:true}).selectOption(chain)
   await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
   await page.getByLabel('작업 체인 구성',{exact:true}).selectOption(first.id)
   await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
   await page.getByLabel('실행 계획',{exact:true}).waitFor()
   const review=await page.getByLabel('실행 계획',{exact:true}).innerText();assert.ok(review.includes(first.id)&&review.includes('r2')&&review.includes('maxpeers=40'),'review omitted actual preset snapshot')
   await page.getByLabel('실행 계획',{exact:true}).screenshot({path:out+'/preset-plan.png'})
   const response=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
   await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click()
   const started=await response;assert.equal(started.status(),202);accepted=await started.json()
  }else{
   const plan=await api('plans','POST',base,undefined,201)
   assert.ok(plan.changes.some(v=>v.includes(first.id)&&v.includes('r2')))
   accepted=await api('jobs','POST',{planId:plan.id},plan.id,202)
  }
  await api('documents/'+revised.id,'PATCH',document(chain,999+i,i===1),undefined,200,revised.revision)
  await waitJob(accepted.id)
  const record=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json'))
  assert.equal(record.nodes.length,5,'preset endpoint layout was dropped')
  assert.equal(record.nodes.filter(n=>n.role==='bp').length,4)
  assert.equal(record.nodes.filter(n=>n.role==='en').length,1)
  if(i===1)assert.equal(record.nodes.find(n=>n.index===1).role,'en','node table order was lost')
  const assets=JSON.parse(fs.readFileSync(f.runtime+'/assets.json'))
  const binary=assets.find(a=>a.id===chain).path
  for(const n of record.nodes){
   const dumped=execFileSync(binary,['--datadir',n.dataDir,'dumpgenesis'],{encoding:'utf8'})
   assert.equal(JSON.parse(dumped).config.chainId,910+i,'accepted preset genesis was replaced or ignored')
   const configPath=n.args[n.args.indexOf('--config')+1]
   const configText=fs.readFileSync(configPath,'utf8')
   if(n.index===1)assert.ok(configText.includes('HTTPHost = \"0.0.0.0\"'),'scoped configuration override was ignored')
   if(n.role==='en'){
    const gc=n.args.indexOf('--gcmode');assert.ok(gc>=0);assert.equal(n.args[gc+1],'archive','endpoint retention mode was ignored')
    assert.ok(configText.includes('SyncMode = "full"')&&configText.includes('NoPruning = true'),'archive must use a valid native sync mode without pruning')
   }
   const flag='--maxpeers'
   const pos=n.args.indexOf(flag);assert.ok(pos>=0);assert.equal(n.args[pos+1],n.index===1?'50':'40','scoped preset option was dropped')
   assert.ok(fs.existsSync(n.dataDir+'/nodekey')||fs.readdirSync(n.dataDir).length>0,'native database absent')
  }
  const endpoint=record.nodes.find(n=>n.role==='en')
  for(const operation of ['node.start','node.stop']){
   const plan=await api('plans','POST',{...base,operation,nodeIds:['node'+endpoint.index],arguments:{manifestId:chain,assetId:chain,serverRef:'local',validators:4}},undefined,201)
   await waitJob((await api('jobs','POST',{planId:plan.id},plan.id,202)).id)
   if(operation==='node.start'){
    let actualId
    const host='127.0.0.1', port=11900+i*100+(endpoint.index-1)*10
    for(let attempt=0;attempt<200;attempt++){
     try{const response=await fetch(`http://${host}:${port}/`,{signal:AbortSignal.timeout(1000),method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({jsonrpc:'2.0',id:1,method:'eth_chainId',params:[]})});actualId=(await response.json()).result;if(actualId)break}catch{}
     await new Promise(r=>setTimeout(r,200))
    }
    assert.equal(Number(actualId),910+i,'native archive endpoint did not start with the reviewed genesis')
    const observed=await api('networks/'+w.id+'/observations')
    assert.equal(observed.nodes.find(n=>n.id==='node'+endpoint.index).state,'running')
   }
  }
  applied.push({chain,workspaceId:w.id,documentId:first.id,revision:2,chainId:910+i,nodes:record.nodes.map(n=>({index:n.index,role:n.role,args:n.args,dataDir:n.dataDir}))})
 }
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:browser.version(),jobs,applied,stalePresetRejected:true,acceptedPresetEditIgnored:true,nativeDatabasesVerified:true,archiveEndpointsServedRPC:true,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
