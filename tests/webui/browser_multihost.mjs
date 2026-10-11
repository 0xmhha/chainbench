// Two-server placement and per-node work through the Web UI, on Docker
// servers reached over SSH from a control container: one deploy spreads the
// nodes over both servers and the chain makes blocks on both; a node on the
// second server is observed, stopped and started there; its logs are
// collected from that server; an attach test runs against the spread network.
// Development proof only; no Seed criterion is awarded.
import {launchOwnedBrowser} from './owned-browser.mjs'
import {clickRun} from './webui-steps.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser()
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
const terminal=['succeeded','failed','cancelled','interrupted']
async function request(ctx,token,path,method='GET',data,key,expected=200){
 const response=await ctx.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(token?{'X-CSRF-Token':token}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data),timeout:180000})
 const raw=await response.text();assert.equal(response.status(),expected,`${method} ${path}: ${raw}`)
 assert.ok(!raw.includes(f.sshPassword)&&!raw.includes(f.password)&&!raw.includes(f.setupToken),'private data leaked')
 return raw?JSON.parse(raw):undefined
}
async function settle(api,id,limit=1800){for(let i=0;i<limit;i++){const job=await api('jobs/'+id);if(terminal.includes(job.state))return job;await sleep(500)}throw new Error('job did not settle')}
const observed={}
let page
try{
 // The control dashboard serves a throwaway self-signed certificate.
 const adminCtx=await owned.browser.newContext({ignoreHTTPSErrors:true})
 const admin=await request(adminCtx,undefined,'bootstrap','POST',{username:'mh-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 await request(adminCtx,admin.csrfToken,'users','POST',{username:'mh-operator',password:f.password,role:'operator'},undefined,201)
 const ctx=await owned.browser.newContext({viewport:{width:1440,height:1100},ignoreHTTPSErrors:true})
 const s=await request(ctx,undefined,'auth/login','POST',{username:'mh-operator',password:f.password})
 const api=(path,method,data,key,expected)=>request(ctx,s.csrfToken,path,method,data,key,expected)

 // Shared documents name both servers; the SSH login is my own credential.
 const set=await api('documents','POST',{kind:'server-set',name:'two servers',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'server1',addr:'172.30.0.11'},{name:'server2',addr:'172.30.0.12'}],slots:4,ports:{p2p:{base:30301,step:2},rpc:{base:8601,step:1},ws:{base:8701,step:1},auth:{base:8501,step:1},metrics:{base:6060,step:0}}},ssh:{port:10022,insecure_host_key:true}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'two server paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.dataRoot,paths:{binaries:'bin',configs:'configs',genesis:'genesis',keystore:'keystore',keyrings:'keys',nodes:'node',runtime:'runtime',logs:'logs'},control:{artifactRoot:'/state/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
 const w=await api('workspaces','POST',{name:'two server chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const credential=await api('credentials','POST',{label:'docker login',kind:'password',sshUser:f.sshUser,password:f.sshPassword},undefined,201)
 for(const server of ['server1','server2'])await api(`workspaces/${w.id}/credential-bindings`,'PUT',{serverRef:server,credentialId:credential.id})
 const bindings={server1:credential.id,server2:credential.id}

 // 1. One deploy over the whole server set, reviewed and started in the browser.
 page=await ctx.newPage()
 await page.goto(f.url+'/chains')
 await page.getByLabel('작업 Workspace',{exact:true}).locator(`option[value="${w.id}"]`).waitFor({state:'attached',timeout:30000})
 await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
 await page.getByLabel('작업 매니페스트',{exact:true}).selectOption('stablenet')
 await page.getByLabel('작업 바이너리',{exact:true}).selectOption('stablenet')
 await page.getByLabel('작업 종류',{exact:true}).selectOption('chain.deploy')
 await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('*all*')
 await page.getByTestId('spread-explanation').waitFor()
 await page.getByLabel('작업 생산자 수',{exact:true}).fill('4')
 await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
 await page.getByTestId('plan-review').waitFor({timeout:180000})
 observed.planBindings=await page.getByTestId('plan-bindings').textContent()
 assert.match(observed.planBindings,/server1 → docker login/);assert.match(observed.planBindings,/server2 → docker login/)
 observed.planAccess=await page.getByTestId('plan-access').textContent()
 const planText=await page.getByLabel('실행 계획',{exact:true}).textContent()
 assert.match(planText,/172\.30\.0\.11|server1/);assert.match(planText,/172\.30\.0\.12|server2/)
 await page.screenshot({path:out+'/spread-plan.png',fullPage:true})
 await page.waitForFunction(()=>![...document.querySelectorAll('button')].find(b=>b.textContent==='검토한 계획 실행')?.disabled,null,{timeout:120000})
 const started=await clickRun(page)
 assert.equal(started.status(),202)
 const deploy=await settle(api,(await started.json()).id)
 assert.equal(deploy.state,'succeeded',JSON.stringify(deploy))
 observed.deploy={id:deploy.id,disposition:deploy.nodeDisposition}

 // 2. The nodes run on both machines and every node makes blocks.
 let network=await api(`networks/${w.id}/observations`)
 const machines=[...new Set(network.nodes.map(n=>n.hostIdentity))]
 assert.equal(machines.length,2,'nodes are not on two machines: '+JSON.stringify(network.nodes.map(n=>[n.id,n.hostIdentity])))
 assert.ok(network.nodes.every(n=>n.state==='running'),JSON.stringify(network.nodes.map(n=>[n.id,n.state,n.observationReason])))
 observed.placement=network.nodes.map(n=>({id:n.id,role:n.role,machine:n.hostIdentity,state:n.state}))
 let heights={}
 for(let i=0;i<120;i++){
  const to=new Date(),from=new Date(to.getTime()-10*60*1000)
  const m=await api(`networks/${w.id}/metrics?from=${encodeURIComponent(from.toISOString())}&to=${encodeURIComponent(to.toISOString())}`)
  heights=Object.fromEntries(network.nodes.map(n=>[n.id,Math.max(0,...(m.series.find(x=>x.nodeId===n.id&&x.name==='block_height')?.samples??[]).map(p=>p.value??0))]))
  if(Object.values(heights).every(h=>h>=3))break
  await sleep(2000)
 }
 assert.ok(Object.values(heights).every(h=>h>=3),'not every node made blocks: '+JSON.stringify(heights))
 observed.blockHeights=heights

 // 3. A node on the second server: observed, stopped and started there.
 const firstMachine=network.nodes[0].hostIdentity
 const remote=network.nodes.find(n=>n.hostIdentity!==firstMachine)
 await page.goto(`${f.url}/nodes/${w.id}/${remote.id}`)
 await page.getByRole('button',{name:'노드 상태 확인',exact:true}).click()
 await page.waitForFunction(()=>/running/.test(document.querySelector('[data-testid=node-state]')?.textContent??''),null,{timeout:120000})
 await page.getByRole('button',{name:'노드 정지',exact:true}).click()
 await page.waitForURL(f.url+'/chains')
 await page.getByLabel('작업 노드',{exact:true}).locator(`option[value="${remote.id}"]`).waitFor({state:'attached',timeout:60000})
 if(!await page.getByLabel('작업 매니페스트',{exact:true}).inputValue())await page.getByLabel('작업 매니페스트',{exact:true}).selectOption('stablenet')
 if(!await page.getByLabel('작업 바이너리',{exact:true}).inputValue())await page.getByLabel('작업 바이너리',{exact:true}).selectOption('stablenet')
 await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('*all*')
 await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
 await page.getByTestId('plan-review').waitFor({timeout:180000})
 await page.waitForFunction(()=>![...document.querySelectorAll('button')].find(b=>b.textContent==='검토한 계획 실행')?.disabled,null,{timeout:120000})
 const stop=await settle(api,(await (await clickRun(page)).json()).id)
 assert.equal(stop.state,'succeeded',JSON.stringify(stop))
 network=await api(`networks/${w.id}/observations`)
 assert.equal(network.nodes.find(n=>n.id===remote.id).state,'stopped')
 assert.ok(network.nodes.filter(n=>n.id!==remote.id).every(n=>n.state==='running'),'other nodes stopped with it')
 const control={workspaceId:w.id,documentRefs:w.documents,assetRefs:['stablenet'],credentialBindings:bindings,retention:'retain',arguments:{manifestId:'stablenet',assetId:'stablenet',allServers:true}}
 const startJob=async input=>{const p=await api('plans','POST',input,undefined,201);return (await api('jobs','POST',{planId:p.id},p.id,202)).id}
 const start=await settle(api,await startJob({...control,operation:'node.start',nodeIds:[remote.id]}))
 assert.equal(start.state,'succeeded',JSON.stringify(start))
 network=await api(`networks/${w.id}/observations`)
 assert.equal(network.nodes.find(n=>n.id===remote.id).state,'running')
 observed.remoteControl={node:remote.id,machine:remote.hostIdentity,stop:stop.id,start:start.id}

 // 4. Logs of that node are collected from its own server.
 const since=new Date()
 const monitor=await startJob({...control,assetRefs:[],operation:'network.monitor',arguments:{allServers:true}})
 for(let i=0;i<60;i++){const j=await api('jobs/'+monitor);if(j.state==='running')break;if(terminal.includes(j.state))throw new Error('log collection ended: '+JSON.stringify(j));await sleep(1000)}
 await sleep(20000)
 await api('jobs/'+monitor+'/cancel','POST',{},undefined,202)
 await settle(api,monitor)
 const logs=await api(`nodes/${encodeURIComponent(w.id+'.'+remote.id)}/logs?from=${encodeURIComponent(new Date(since.getTime()-60000).toISOString())}&to=${encodeURIComponent(new Date().toISOString())}&limit=100`)
 assert.ok(logs.entries.length>0,'no log lines were collected from the second server')
 observed.remoteLogs={node:remote.id,lines:logs.entries.length}

 // 5. An attach test runs against the spread network.
 const attachCase=await api('documents','POST',{kind:'case',name:'spread attach',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'spread-attach',requires:['rpc'],chainPreset:{chain:'stablenet',attach:{rpc:['http://203.0.113.9:8545'],keysDir:'presets/keys',provides:['rpc','consensus']}},steps:[{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:1}]}},undefined,201)
 const attach=await settle(api,await startJob({...control,operation:'test.attach',arguments:{manifestId:'stablenet',assetId:'stablenet',allServers:true,caseRefs:[{id:attachCase.id,revision:attachCase.revision}]}}))
 assert.equal(attach.state,'succeeded',JSON.stringify(attach))
 observed.attach={id:attach.id,runIds:attach.runIds}
 await page.goto(`${f.url}/monitoring`)
 await page.screenshot({path:out+'/spread-monitoring.png',fullPage:true})
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observed},null,2))
}catch(error){
 if(page){await page.screenshot({path:out+'/failure.png',fullPage:true}).catch(()=>{});fs.writeFileSync(out+'/failure.txt',(await page.locator('[role=alert],[role=status]').allTextContents().catch(()=>[])).join('\n'))}
 throw error
}finally{await owned.stop()}
