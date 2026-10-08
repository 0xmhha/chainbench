import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext({viewport:{width:1440,height:1100}})
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
async function request(ctx,token,path,method='GET',data,key,expected=200){
 const response=await ctx.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(token?{'X-CSRF-Token':token}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const raw=await response.text();assert.equal(response.status(),expected,`${method} ${path}: ${raw}`)
 assert.ok(!raw.includes(f.password)&&!raw.includes(f.setupToken),'private bootstrap data leaked')
 return raw?JSON.parse(raw):undefined
}
let session
const api=(path,method,data,key,expected)=>request(context,session?.csrfToken,path,method,data,key,expected)
async function waitJob(id){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,'succeeded',JSON.stringify(job));return job}await sleep(200)}throw new Error('monitoring job timed out')}
const series=(m,node,name,source)=>m.series.find(s=>s.nodeId===node&&s.name===name&&s.source===source)
async function rpc(method){const r=await context.request.post('http://127.0.0.1:11700/',{data:{jsonrpc:'2.0',id:1,method,params:[]}});assert.equal(r.status(),200);return parseInt((await r.json()).result,16)}
try{
 session=await api('bootstrap','POST',{username:'monitor-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 await api('users','POST',{username:'monitor-viewer',password:f.password,role:'viewer'},undefined,201)
 const set=await api('documents','POST',{kind:'server-set',name:'monitor pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39100,step:10},rpc:{base:11700,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'monitor paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
 const w=await api('workspaces','POST',{name:'monitor native chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const c=await api('documents','POST',{kind:'case',name:'monitor real test',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'monitor-real-test',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}},undefined,201)
 const base={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local'}}
 const plan=await api('plans','POST',{...base,operation:'test.run',arguments:{...base.arguments,caseRefs:[{id:c.id,revision:c.revision}]}},undefined,201)
 const job=await waitJob((await api('jobs','POST',{planId:plan.id},plan.id,202)).id)
 const record=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
 const labels=record.nodes.map(n=>n.label)
 assert.equal(labels.length,4);assert.ok(record.nodes.every(n=>n.pid>0&&n.metrics>0),'retained nodes must expose recorded metrics ports')

 // Collected values must come from the native nodes: two RPC samples per node,
 // a metrics-endpoint head, and values no greater than a later direct RPC read.
 let metrics
 for(let i=0;i<60;i++){metrics=await api('networks/'+w.id+'/metrics');if(labels.every(l=>(series(metrics,l,'block_height','rpc')?.samples.length||0)>=2&&series(metrics,l,'chain_head_block','metrics')))break;await sleep(1000)}
 fs.writeFileSync(out+'/metrics-before-restart.json',JSON.stringify(metrics,null,2))
 for(const l of labels){const s=series(metrics,l,'block_height','rpc');assert.ok(s&&s.samples.length>=2&&s.unit==='blocks','missing rpc block height for '+l);assert.ok(series(metrics,l,'chain_head_block','metrics'),'missing metrics endpoint head for '+l);assert.ok(series(metrics,l,'peer_count','rpc'),'missing rpc peers for '+l)}
 const latest=Math.max(...series(metrics,'node1','block_height','rpc').samples.map(s=>s.value)),direct=await rpc('eth_blockNumber')
 assert.ok(latest>0&&latest<=direct,`collected head ${latest} must be a real head no later than ${direct}`)
 assert.equal(metrics.coverage.sampleIntervalSeconds,5);assert.ok(Array.isArray(metrics.coverage.gaps))

 const page=await context.newPage();await page.goto(f.url+'/monitoring')
 await page.getByRole('heading',{name:'노드 지표',exact:true}).waitFor({timeout:15000})
 const chart=page.getByTestId('metric-chart-block_height')
 await chart.waitFor({timeout:20000})
 for(const l of labels)await chart.locator(`[data-node="${l}"]`).first().waitFor({timeout:20000})
 await page.screenshot({path:out+'/monitoring-desktop.png',fullPage:true})

 // Selecting a chart time opens that node's archived log lines around it; each
 // shown line must exist verbatim in the native node's own log file.
 // Equal heads overlap, so the topmost point selects the time and the log
 // panel's node choice selects node1 for that same time.
 await chart.locator('circle[data-time]').last().click()
 const panel=page.getByTestId('time-linked-logs');await panel.waitFor({timeout:15000})
 await panel.getByLabel('로그 노드').selectOption('node1')
 const selected=await panel.getAttribute('data-selected-time');assert.ok(selected,'selected time missing')
 await page.locator('[data-testid="time-linked-logs"][data-log-node="node1"]').waitFor({timeout:20000})
 const line=panel.locator('[data-log-time]').first();await line.waitFor({timeout:20000})
 const nodeLog=fs.readFileSync(record.nodes[0].logPath,'utf8'),shown=await panel.locator('[data-log-time] .log-text').allTextContents()
 assert.ok(shown.length>0&&shown.every(t=>nodeLog.includes(t)),'linked logs are not the native node log')
 const logs=await api('nodes/'+w.id+'.node1/logs?from='+encodeURIComponent(new Date(Date.parse(selected)-30000).toISOString())+'&to='+encodeURIComponent(new Date(Date.parse(selected)+30000).toISOString()))
 assert.ok(logs.entries.length>0&&logs.entries.every(e=>Math.abs(Date.parse(e.time)-Date.parse(selected))<=30000),'log window is not time linked')
 assert.ok(['source','mixed'].includes(logs.coverage.timestampSource),'native geth timestamps not recognised')

 // A viewer reads shared observations but cannot inspect owned process state.
 const viewer=await owned.browser.newContext()
 const vs=await request(viewer,undefined,'auth/login','POST',{username:'monitor-viewer',password:f.password})
 await request(viewer,vs.csrfToken,'networks/'+w.id+'/metrics');await request(viewer,vs.csrfToken,'nodes/'+w.id+'.node1/logs')
 await request(viewer,vs.csrfToken,'networks/'+w.id+'/observations','GET',undefined,undefined,403)
 await viewer.close()

 // Collection gaps across a dashboard restart are explicit and archived samples survive.
 fs.writeFileSync(f.runtime+'/restart.request','explicit fixture dashboard restart')
 for(let i=0;i<300&&!fs.existsSync(f.runtime+'/restart.response');i++)await sleep(100)
 const restarted=JSON.parse(fs.readFileSync(f.runtime+'/restart.response','utf8'));assert.ok(!restarted.error&&restarted.oldPid!==restarted.newPid)
 session=await request(context,undefined,'auth/login','POST',{username:'monitor-admin',password:f.password})
 let after
 for(let i=0;i<30;i++){after=await api('networks/'+w.id+'/metrics');if(after.coverage.gaps.some(g=>g.reason.includes('collector_stopped')))break;await sleep(1000)}
 fs.writeFileSync(out+'/metrics-after-restart.json',JSON.stringify(after,null,2))
 assert.ok(after.coverage.gaps.some(g=>g.reason.includes('collector_stopped'))&&!after.coverage.complete,'restart interval not reported as a collection gap')
 const first=series(metrics,'node1','block_height','rpc').samples[0]
 assert.ok(series(after,'node1','block_height','rpc').samples.some(s=>s.time===first.time&&s.value===first.value),'archived samples lost across restart')
 await page.reload() // The page shares the context cookie from the login above.
 await page.getByTestId('metric-coverage').filter({hasText:'collector_stopped'}).waitFor({timeout:20000})
 await page.setViewportSize({width:390,height:844});await page.screenshot({path:out+'/monitoring-mobile.png',fullPage:true})
 assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'monitoring mobile overflow')
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),job:{id:job.id,state:job.state,runIds:job.runIds},nodes:labels,rpcAndMetricsSources:true,timeLinkedNativeLogs:true,viewerReadOnly:true,restartGapReported:true,archivedAcrossRestart:true,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
