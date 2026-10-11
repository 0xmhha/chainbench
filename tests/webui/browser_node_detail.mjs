// Node detail, plan review and binary surface on a real dashboard with native
// wbft nodes: a producer's reset is disabled with its reason, a control chosen
// on the node page becomes a reviewed plan that shows access, bindings,
// pinned documents and surfaces, a server without the user's SSH credential is
// named before anything can run, and the chain screen shows what the chosen
// binary supports. Development proof only; no Seed criterion is awarded.
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
async function settle(api,id){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(terminal.includes(job.state))return job;await sleep(200)}throw new Error('job did not settle')}
const observed={}
let page
try{
 const adminCtx=await owned.browser.newContext()
 const adminSession=await request(adminCtx,undefined,'bootstrap','POST',{username:'node-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 await request(adminCtx,adminSession.csrfToken,'users','POST',{username:'node-operator',password:f.password,role:'operator'},undefined,201)
 const ctx=await owned.browser.newContext({viewport:{width:1440,height:1100}})
 const s=await request(ctx,undefined,'auth/login','POST',{username:'node-operator',password:f.password})
 const api=(path,method,data,key,expected)=>request(ctx,s.csrfToken,path,method,data,key,expected)
 const paths={version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}
 const set=await api('documents','POST',{kind:'server-set',name:'node pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:5,ports:{p2p:{base:39500,step:10},rpc:{base:12100,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'node paths',contractVersion:'2',assetRefs:[],content:paths},undefined,201)
 const w=await api('workspaces','POST',{name:'node chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const preset={chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4,en:1},launch:{all:{ipcdisable:true}}}
 const long=await api('documents','POST',{kind:'case',name:'node long case',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'node-long',chainPreset:preset,steps:[{do:'waitBlock',on:'node1',target:1000000000,timeout:'5m'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}},undefined,201)
 // Compose a retained network: run the long case, cancel it once nodes run.
 const plan=await api('plans','POST',{workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],retention:'retain',operation:'test.run',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local',caseRefs:[{id:long.id,revision:long.revision}]}},undefined,201)
 const started=(await api('jobs','POST',{planId:plan.id},plan.id,202)).id
 const record=()=>JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
 for(let i=0;i<900;i++){const job=await api('jobs/'+started);let up=false;try{up=record().nodes.length===5&&record().nodes.every(n=>n.pid>0&&live(n.pid))}catch{}if(job.state==='running'&&job.phases.some(p=>p.name==='test.run'&&p.state==='running')&&up)break;if(terminal.includes(job.state))throw new Error('composition ended early: '+JSON.stringify(job));await sleep(200)}
 await api('jobs/'+started+'/cancel','POST',{},undefined,202)
 assert.equal((await settle(api,started)).nodeDisposition,'retained')
 const nodes=(await api('networks')).items.find(n=>n.id===w.id).nodes
 const producer=nodes.find(n=>n.role==='bp'),endpoint=nodes.find(n=>n.role==='en')
 assert.ok(producer&&endpoint,JSON.stringify(nodes))

 // 1. The producer's page shows its state and refuses reset with the reason.
 page=await ctx.newPage()
 await page.goto(`${f.url}/nodes/${w.id}/${producer.id}`)
 await page.getByTestId('node-role').waitFor({timeout:15000})
 await page.getByRole('button',{name:'노드 상태 확인',exact:true}).click()
 await page.waitForFunction(()=>/running/.test(document.querySelector('[data-testid=node-state]')?.textContent??''),null,{timeout:60000})
 const resetReason=await page.getByTestId('control-limit-node.reset').textContent()
 assert.match(resetReason,/블록 생산자/)
 assert.ok(await page.getByRole('button',{name:/비생산자 초기화/}).isDisabled(),'producer reset is enabled')
 assert.ok(!(await page.getByRole('button',{name:'노드 정지',exact:true}).isDisabled()),'stop is disabled on a running producer')
 await page.screenshot({path:out+'/node-producer.png',fullPage:true})
 observed.producerResetRefused=resetReason

 // 2. Stopping the endpoint from its page leads to a reviewed plan.
 await page.goto(`${f.url}/nodes/${w.id}/${endpoint.id}`)
 await page.getByRole('button',{name:'노드 상태 확인',exact:true}).click()
 await page.waitForFunction(()=>/running/.test(document.querySelector('[data-testid=node-state]')?.textContent??''),null,{timeout:60000})
 await page.getByRole('button',{name:'노드 정지',exact:true}).click()
 await page.waitForURL(f.url+'/chains')
 await page.getByLabel('작업 노드',{exact:true}).locator(`option[value="${endpoint.id}"]`).waitFor({state:'attached',timeout:30000})
 assert.equal(await page.getByLabel('작업 종류',{exact:true}).inputValue(),'node.stop')
 assert.equal(await page.getByLabel('작업 노드',{exact:true}).inputValue(),endpoint.id)
 if(!await page.getByLabel('작업 매니페스트',{exact:true}).inputValue())await page.getByLabel('작업 매니페스트',{exact:true}).selectOption('wbft')
 if(!await page.getByLabel('작업 바이너리',{exact:true}).inputValue())await page.getByLabel('작업 바이너리',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
 await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
 await page.getByTestId('plan-review').waitFor({timeout:60000})
 const review={access:await page.getByTestId('plan-access').textContent(),bindings:await page.getByTestId('plan-bindings').textContent(),documents:await page.getByTestId('plan-documents').textContent(),surfaces:await page.getByTestId('plan-surfaces').textContent()}
 assert.match(review.access,/local filesystem and process access/)
 assert.match(review.bindings,/로컬 대상/)
 assert.match(review.documents,/node pool \(server-set\) r1/);assert.match(review.documents,/node paths \(workspace-config\) r1/)
 assert.match(review.surfaces,/wbft · 자료 wbft · 계약/)
 await page.screenshot({path:out+'/plan-review.png',fullPage:true})
 observed.planReview=review
 const run=page.getByRole('button',{name:'검토한 계획 실행',exact:true})
 await page.waitForFunction(()=>![...document.querySelectorAll('button')].find(b=>b.textContent==='검토한 계획 실행')?.disabled,null,{timeout:60000})
 await run.click()
 await page.waitForURL(/\/jobs\/[0-9a-f]{32}$/,{timeout:30000})
 const stopJob=await settle(api,page.url().split('/').pop())
 assert.equal(stopJob.state,'succeeded',JSON.stringify(stopJob))
 const after=(await api('networks/'+w.id+'/observations')).nodes.find(n=>n.id===endpoint.id)
 assert.equal(after.state,'stopped',JSON.stringify(after))
 observed.stopFromNodePage=stopJob.id

 // 3. A server reached over SSH without my credential is named; nothing can run.
 const remote=await api('documents','POST',{kind:'server-set',name:'remote pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'remote',addr:'192.0.2.10'}],slots:4,ports:{p2p:{base:39700,step:10},rpc:{base:12300,step:10}}}}},undefined,201)
 const rw=await api('workspaces','POST',{name:'remote chain',documents:[{id:remote.id,revision:remote.revision},{id:config.id,revision:config.revision}]},undefined,201)
 await page.goto(f.url+'/chains')
 await page.getByLabel('작업 Workspace',{exact:true}).locator(`option[value="${rw.id}"]`).waitFor({state:'attached',timeout:30000})
 await page.getByLabel('작업 Workspace',{exact:true}).selectOption(rw.id)
 await page.getByLabel('작업 매니페스트',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 바이너리',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 종류',{exact:true}).selectOption('chain.setup')
 await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('remote')
 await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
 await page.getByTestId('missing-access').waitFor({timeout:30000})
 assert.match(await page.getByTestId('missing-access').textContent(),/ssh:remote/)
 assert.equal(await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).count(),0,'a run button appeared without access')
 await page.screenshot({path:out+'/missing-access.png',fullPage:true})
 observed.missingAccess='ssh:remote'

 // 4. The chain screen judges the chosen binary against the documentation.
 await page.goto(f.url+'/chains')
 await page.locator('#chain-preset').selectOption({index:1})
 const chainOptions=await page.locator('#chain-preset option').allTextContents()
 const wbftIndex=chainOptions.findIndex(t=>/wbft/i.test(t))
 if(wbftIndex>0)await page.locator('#chain-preset').selectOption({index:wbftIndex})
 await page.getByLabel('지원 범위를 확인할 바이너리',{exact:true}).locator('option[value="wbft"]').waitFor({state:'attached',timeout:30000})
 await page.getByLabel('지원 범위를 확인할 바이너리',{exact:true}).selectOption('wbft')
 await page.waitForFunction(()=>/verified/.test(document.querySelector('[data-testid=surface-verification]')?.textContent??''),null,{timeout:60000})
 assert.match(await page.getByTestId('surface-binary').textContent(),/커밋/)
 assert.ok(await page.getByTestId('surface-option').count()>0)
 await page.screenshot({path:out+'/surface.png',fullPage:false})
 observed.surface=await page.getByTestId('surface-verification').textContent()
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observed,seedAcceptanceAwarded:false},null,2))
}catch(error){
 if(page){await page.screenshot({path:out+'/failure.png',fullPage:true}).catch(()=>{});fs.writeFileSync(out+'/failure.txt',(await page.locator('[role=alert],[role=status]').allTextContents().catch(()=>[])).join('\n'))}
 throw error
}finally{await owned.stop()}
