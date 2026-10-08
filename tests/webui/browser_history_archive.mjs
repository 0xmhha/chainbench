import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
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
async function waitJob(id){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,'succeeded',JSON.stringify(job));return job}await sleep(200)}throw new Error('archive job timed out')}
const heights=m=>m.series.filter(s=>s.name==='block_height'&&s.source==='rpc').flatMap(s=>s.samples)
async function login(username){const ctx=await owned.browser.newContext();const s=await request(ctx,undefined,'auth/login','POST',{username,password:f.password});return {ctx,s}}
try{
 session=await api('bootstrap','POST',{username:'archive-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 for(const [username,role] of [['archive-operator','operator'],['archive-viewer','viewer']])await api('users','POST',{username,password:f.password,role},undefined,201)
 const set=await api('documents','POST',{kind:'server-set',name:'archive pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39100,step:10},rpc:{base:11700,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'archive paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
 const w=await api('workspaces','POST',{name:'archive native chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const c=await api('documents','POST',{kind:'case',name:'archive real test',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'archive-real-test',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}},undefined,201)
 const base={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local'}}
 const plan=await api('plans','POST',{...base,operation:'test.run',arguments:{...base.arguments,caseRefs:[{id:c.id,revision:c.revision}]}},undefined,201)
 const testJob=await waitJob((await api('jobs','POST',{planId:plan.id},plan.id,202)).id)
 // Let collection run beyond the test job, then stop one node in a second real job.
 await sleep(12000)
 const stopPlan=await api('plans','POST',{...base,operation:'node.stop',nodeIds:['node4'],arguments:{...base.arguments,validators:4}},undefined,201)
 const stopJob=await waitJob((await api('jobs','POST',{planId:stopPlan.id},stopPlan.id,202)).id)
 // Compare deletion against the record after both real jobs changed it.
 const record=fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json')
 const runId='job-'+testJob.id
 const run=await api('history/'+runId)
 assert.equal(run.summary.observations?.networkId,w.id,'run lacks its archived window')
 assert.ok(!run.summary.missingDimensions.includes('metrics')&&!run.summary.missingDimensions.includes('logs'),'archived dimensions reported missing')
 const runMetrics=await api('networks/'+w.id+'/metrics?runId='+runId)
 const runSamples=heights(runMetrics)
 assert.ok(runSamples.length>0,'no samples archived inside the real test run')
 assert.ok(runSamples.every(s=>Date.parse(s.time)>=Date.parse(run.summary.observations.from)&&Date.parse(s.time)<=Date.parse(run.summary.observations.to)),'run query left its window')
 const after=Date.parse(run.summary.observations.to)
 const all=await api('networks/'+w.id+'/metrics')
 const later=heights(all).filter(s=>Date.parse(s.time)>after)
 assert.ok(later.length>0,'no samples after the run; protection check would be vacuous')

 // Detail shows the window and its archived series in the real UI.
 const page=await context.newPage();await page.goto(f.url+'/history?search='+encodeURIComponent(testJob.id))
 await page.getByRole('button',{name:new RegExp('test.run')}).first().click()
 await page.getByTestId('run-observations').waitFor({timeout:15000})
 await page.getByRole('button',{name:'이 실행의 지표 확인',exact:true}).click()
 const shown=page.getByTestId('run-observation-series');await shown.waitFor({timeout:15000})
 assert.ok(Number(await shown.getAttribute('data-series'))>0)
 await page.screenshot({path:out+'/history-archive-desktop.png',fullPage:true})

 // Only an administrator can delete; operators and viewers are refused.
 for(const user of ['archive-operator','archive-viewer']){const {ctx,s}=await login(user);await request(ctx,s.csrfToken,'history/'+runId,'DELETE',undefined,undefined,403);await ctx.close()}

 // The archive survives a dashboard restart before deletion.
 fs.writeFileSync(f.runtime+'/restart.request','explicit fixture dashboard restart')
 for(let i=0;i<300&&!fs.existsSync(f.runtime+'/restart.response');i++)await sleep(100)
 const restarted=JSON.parse(fs.readFileSync(f.runtime+'/restart.response','utf8'));assert.ok(!restarted.error&&restarted.oldPid!==restarted.newPid)
 session=await request(context,undefined,'auth/login','POST',{username:'archive-admin',password:f.password})
 assert.deepEqual(heights(await api('networks/'+w.id+'/metrics?runId='+runId)).map(s=>s.time),runSamples.map(s=>s.time),'archived run samples changed across restart')

 // A failed archive rewrite keeps the run for retry and is reported, not hidden.
 const archive=f.store+'/observations/'+w.id
 fs.chmodSync(archive,0o500)
 let failed
 try{failed=await context.request.fetch(f.url+'/api/v1/history/'+runId,{method:'DELETE',headers:{'X-CSRF-Token':session.csrfToken}})}finally{fs.chmodSync(archive,0o700)}
 assert.equal(failed.status(),500,await failed.text())
 assert.equal((await api('history/'+runId)).id,runId,'run removed although its observations were not')

 // Retrying through the UI removes only the run-owned samples and logs.
 await page.reload();await page.getByRole('button',{name:new RegExp('test.run')}).first().click()
 await page.getByRole('button',{name:'보관 결과 삭제',exact:true}).click()
 const removed=page.waitForResponse(r=>r.url().endsWith('/api/v1/history/'+runId)&&r.request().method()==='DELETE')
 await page.getByRole('button',{name:'결과 사본 삭제 확정',exact:true}).click()
 assert.equal((await removed).status(),204)
 await api('history/'+runId,'GET',undefined,undefined,404)
 const remaining=await api('networks/'+w.id+'/metrics?from='+encodeURIComponent(new Date(Date.parse(run.summary.observations.from)-60000).toISOString()))
 const remainingTimes=new Set(heights(remaining).map(s=>s.time))
 assert.ok(runSamples.every(s=>!remainingTimes.has(s.time)),'run-owned samples survived deletion')
 assert.ok(later.every(s=>remainingTimes.has(s.time)),'samples outside the run were deleted')
 assert.ok(remaining.coverage.gaps.some(g=>g.reason==='archive deleted_by_admin '+runId),'deletion not distinguishable from missing collection')
 assert.equal((await api('history/job-'+stopJob.id)).id,'job-'+stopJob.id,'another run was removed')
 // Live node data and records are never part of history deletion.
 assert.deepEqual(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json'),record,'deletion changed the chain record')
 const rpc=await context.request.post('http://127.0.0.1:11700/',{data:{jsonrpc:'2.0',id:1,method:'eth_blockNumber',params:[]}})
 assert.ok(parseInt((await rpc.json()).result,16)>0,'live node stopped serving after deletion')
 for(const node of JSON.parse(record).nodes)assert.ok(fs.existsSync(node.dataDir),'node data removed')
 const nodeCommand=execFileSync('ps',['-p',String(JSON.parse(record).nodes[0].pid),'-o','command='],{encoding:'utf8'})
 assert.ok(nodeCommand.includes(f.runtime),'live node process not intact')
 await page.setViewportSize({width:390,height:844});await page.screenshot({path:out+'/history-archive-mobile.png',fullPage:true})
 assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'history mobile overflow')
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),runId,ownedSamples:runSamples.length,laterSamplesKept:later.length,partialFailureKeptRun:true,adminOnly:true,archiveAcrossRestart:true,liveNodeIntact:true,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
