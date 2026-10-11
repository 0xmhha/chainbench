// Monitoring selection, cancel, gaps and the shared time range on native wbft
// nodes: the monitoring screen narrows to one workspace and network, cancels a
// running job there, draws a node's stop as a break in its line, narrows the
// charts to a node and a metric, and reloads charts and logs for a new range.
// Development proof only; no Seed criterion is awarded.
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
const series=(m,node,name)=>m.series.find(s=>s.nodeId===node&&s.name===name)
const observed={}
let page
try{
 const adminCtx=await owned.browser.newContext()
 const adminSession=await request(adminCtx,undefined,'bootstrap','POST',{username:'range-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 await request(adminCtx,adminSession.csrfToken,'users','POST',{username:'range-operator',password:f.password,role:'operator'},undefined,201)
 const ctx=await owned.browser.newContext({viewport:{width:1440,height:1100}})
 const s=await request(ctx,undefined,'auth/login','POST',{username:'range-operator',password:f.password})
 const api=(path,method,data,key,expected)=>request(ctx,s.csrfToken,path,method,data,key,expected)
 const paths={version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}
 const set=await api('documents','POST',{kind:'server-set',name:'range pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39900,step:10},rpc:{base:12500,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'range paths',contractVersion:'2',assetRefs:[],content:paths},undefined,201)
 const w=await api('workspaces','POST',{name:'range chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const long=await api('documents','POST',{kind:'case',name:'range long case',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'range-long',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{do:'waitBlock',on:'node1',target:1000000000,timeout:'5m'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}},undefined,201)
 const base={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local'}}
 const plan=await api('plans','POST',{...base,operation:'test.run',arguments:{...base.arguments,caseRefs:[{id:long.id,revision:long.revision}]}},undefined,201)
 const jobId=(await api('jobs','POST',{planId:plan.id},plan.id,202)).id
 const record=()=>JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
 for(let i=0;i<900;i++){const job=await api('jobs/'+jobId);let up=false;try{up=record().nodes.length===4&&record().nodes.every(n=>n.pid>0&&live(n.pid))}catch{}if(job.state==='running'&&job.phases.some(p=>p.name==='test.run'&&p.state==='running')&&up)break;if(terminal.includes(job.state))throw new Error('job ended early: '+JSON.stringify(job));await sleep(200)}

 // 1. Monitoring narrows to this workspace and network and cancels the job there.
 page=await ctx.newPage()
 await page.goto(f.url+'/monitoring')
 await page.getByLabel('모니터링 Workspace',{exact:true}).locator(`option[value="${w.id}"]`).waitFor({state:'attached',timeout:30000})
 await page.getByLabel('모니터링 Workspace',{exact:true}).selectOption(w.id)
 await page.getByLabel('모니터링 네트워크',{exact:true}).selectOption(w.id)
 const row=page.locator(`tr[data-job-id="${jobId}"]`)
 await row.waitFor({timeout:20000})
 page.once('dialog',d=>d.accept())
 await row.getByRole('button',{name:'작업 취소',exact:true}).click()
 const cancelled=await settle(api,jobId)
 assert.equal(cancelled.state,'cancelled');assert.equal(cancelled.nodeDisposition,'retained',JSON.stringify(cancelled))
 observed.cancelFromMonitoring={jobId,disposition:cancelled.nodeDisposition}

 // 2. Stopping node2 for a while leaves a stretch with no samples.
 let metrics
 for(let i=0;i<90;i++){metrics=await api('networks/'+w.id+'/metrics');if(['node1','node2','node3','node4'].every(l=>(series(metrics,l,'block_height')?.samples.filter(p=>p.value!==null).length||0)>=2))break;await sleep(1000)}
 const control=async op=>{const p=await api('plans','POST',{...base,operation:op,nodeIds:['node2']},undefined,201);const j=await settle(api,(await api('jobs','POST',{planId:p.id},p.id,202)).id);assert.equal(j.state,'succeeded',JSON.stringify(j))}
 await control('node.stop');await sleep(20000);await control('node.start')
 const restartedAt=Date.now()
 for(let i=0;i<90;i++){metrics=await api('networks/'+w.id+'/metrics');if((series(metrics,'node2','block_height')?.samples.filter(p=>p.value!==null&&Date.parse(p.time)>restartedAt).length||0)>=2)break;await sleep(1000)}
 const values=series(metrics,'node2','block_height').samples
 observed.node2Samples={total:values.length,missing:values.filter(p=>p.value===null).length}

 // 3. With node2 chosen the chart shows only node2, broken where it was stopped.
 await page.getByLabel('모니터링 노드',{exact:true}).selectOption('node2')
 const chart=page.getByTestId('metric-chart-block_height')
 await page.waitForFunction(()=>{const c=document.querySelector('[data-testid="metric-chart-block_height"]');return c&&Number(c.dataset.segments)>=2&&[...c.querySelectorAll('[data-node]')].every(e=>e.dataset.node==='node2')},null,{timeout:30000})
 observed.node2Segments=Number(await chart.getAttribute('data-segments'))
 // node1 was never stopped, so its line must not break: the break above comes from the stop.
 await page.getByLabel('모니터링 노드',{exact:true}).selectOption('node1')
 await page.waitForFunction(()=>{const c=document.querySelector('[data-testid="metric-chart-block_height"]');return c&&[...c.querySelectorAll('[data-node]')].length>0&&[...c.querySelectorAll('[data-node]')].every(e=>e.dataset.node==='node1')},null,{timeout:30000})
 observed.node1Segments=Number(await chart.getAttribute('data-segments'))
 observed.node1Samples=series(await api('networks/'+w.id+'/metrics'),'node1','block_height').samples.length
 assert.equal(observed.node1Segments,1,'an unstopped node drew a broken line')
 await page.getByLabel('모니터링 노드',{exact:true}).selectOption('node2')
 const rangeLogs=page.locator('[data-testid="range-logs"][data-log-node="node2"]')
 await rangeLogs.locator('[data-log-time]').first().waitFor({timeout:30000})
 assert.equal(await rangeLogs.locator('[role=alert]').count(),0,'range logs failed to load')
 observed.rangeLogLines=await rangeLogs.locator('[data-log-time]').count()
 // 4. One metric narrows the charts to one figure; a point names its time and source.
 await page.getByLabel('지표 선택 '+w.id,{exact:true}).selectOption('block_height')
 await page.waitForFunction(()=>document.querySelectorAll('svg[data-testid^="metric-chart-"]').length===1,null,{timeout:15000})
 await chart.locator('circle').last().click()
 const point=await page.getByTestId('selected-point').textContent()
 assert.match(point,/block_height · node2 · .* · 출처 rpc/)
 observed.selectedPoint=point
 await page.screenshot({path:out+'/monitoring-range.png',fullPage:true})
 // 5. A new range reloads charts and range logs for that range; events follow it.
 const metricsRequest=page.waitForRequest(r=>r.url().includes(`/networks/${w.id}/metrics?from=`),{timeout:15000})
 const logsRequest=page.waitForRequest(r=>r.url().includes('/logs?from=')&&r.url().includes('limit=100'),{timeout:15000})
 await page.getByLabel('모니터링 시간 범위',{exact:true}).selectOption('60')
 const [m,l]=await Promise.all([metricsRequest,logsRequest])
 const span=u=>{const q=new URL(u).searchParams;return (Date.parse(q.get('to'))-Date.parse(q.get('from')))/60000}
 assert.equal(Math.round(span(m.url())),60);assert.equal(Math.round(span(l.url())),60)
 assert.match(await page.getByTestId('ranged-events').textContent().catch(()=>''),/선택한 시간 범위의 이벤트/)
 observed.rangeReload={metricsMinutes:Math.round(span(m.url())),logsMinutes:Math.round(span(l.url()))}
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observed,seedAcceptanceAwarded:false},null,2))
}catch(error){
 if(page){await page.screenshot({path:out+'/failure.png',fullPage:true}).catch(()=>{});fs.writeFileSync(out+'/failure.txt',(await page.locator('[role=alert],[role=status]').allTextContents().catch(()=>[])).join('\n'))}
 throw error
}finally{await owned.stop()}
