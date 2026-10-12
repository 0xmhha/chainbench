import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext({viewport:{width:1440,height:1100},acceptDownloads:true})
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
const credentialSecret='web09-personal-ssh-password-'+f.password.slice(0,8)
async function request(ctx,token,path,method='GET',data,key,expected=200){
 const response=await ctx.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(token?{'X-CSRF-Token':token}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const raw=await response.text();assert.equal(response.status(),expected,`${method} ${path}: ${raw}`)
 for(const secret of [f.password,f.setupToken,credentialSecret])assert.ok(!raw.includes(secret),'private data leaked in '+path)
 return raw?JSON.parse(raw):undefined
}
let session
const api=(path,method,data,key,expected)=>request(context,session?.csrfToken,path,method,data,key,expected)
async function finish(id){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,'succeeded',JSON.stringify(job));return job}await sleep(200)}throw new Error('history job timed out')}
const record={}
try{
 session=await api('bootstrap','POST',{username:'history-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 for(const [username,role] of [['history-operator','operator'],['history-viewer','viewer']])await api('users','POST',{username,password:f.password,role},undefined,201)
 await api('credentials','POST',{label:'personal secret for export scan',kind:'password',sshUser:'history',password:credentialSecret},undefined,201)
 const set=await api('documents','POST',{kind:'server-set',name:'history pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39100,step:10},rpc:{base:11700,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'history paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
 const w=await api('workspaces','POST',{name:'history native chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const caseDoc=id=>api('documents','POST',{kind:'case',name:id,contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id,chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}},undefined,201)
 const shared=await caseDoc('history-shared-case'),other=await caseDoc('history-other-case')
 const base={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local'}}
 async function testRun(doc,retention){const plan=await api('plans','POST',{...base,retention,operation:'test.run',arguments:{...base.arguments,caseRefs:[{id:doc.id,revision:doc.revision}]}},undefined,201);return (await api('jobs','POST',{planId:plan.id},plan.id,202)).id}
 // Two real runs of the same case revision and one of another case; earlier
 // runs clean their nodes so each run composes the network afresh.
 const firstJob=await finish(await testRun(shared,'cleanup'))
 // Selected cleanup after a test that left nodes running stops and removes them.
 assert.equal(firstJob.nodeDisposition,'cleaned',JSON.stringify(firstJob))
 const removed=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
 assert.ok(removed.statePath==='CHAIN/CHAIN_REMOVED'&&(removed.nodes??[]).length===0,'cleanup did not remove the composition as the CLI does')
 assert.ok(!fs.existsSync(f.runtime+'/d/nodes')||fs.readdirSync(f.runtime+'/d/nodes').every(c=>fs.readdirSync(f.runtime+'/d/nodes/'+c).length===0),'cleanup kept node data')
 const secondJob=await finish(await testRun(shared,'cleanup'))
 const thirdJob=await finish(await testRun(other,'cleanup'))
 const sessions=(await api('history?caseId=history-shared-case&limit=50')).items.filter(r=>r.summary.kind==='engine-session')
 assert.equal(sessions.length,2,'two engine sessions of the shared case expected')
 const otherSession=(await api('history?caseId=history-other-case&limit=50')).items.find(r=>r.summary.kind==='engine-session')
 assert.ok(otherSession,'engine session of the other case missing')

 // Filters through the real UI and the API.
 const page=await context.newPage();await page.goto(f.url+'/history')
 await page.getByLabel('히스토리 케이스',{exact:true}).selectOption('history-shared-case')
 await page.getByLabel('히스토리 상태',{exact:true}).selectOption('succeeded')
 const filtered=page.waitForResponse(r=>r.url().includes('/api/v1/history?')&&r.url().includes('caseId=history-shared-case')&&r.url().includes('state=succeeded'))
 await page.getByRole('button',{name:'필터 적용',exact:true}).click()
 const filteredBody=await (await filtered).json()
 assert.ok(filteredBody.items.length>=2&&filteredBody.items.every(r=>r.state==='succeeded'),'case/state filter')
 for(const [query,check] of [['chain=wbft',r=>r.chain==='wbft'],['actorId='+session.user.id,r=>r.actorId===session.user.id],['workspaceId='+w.id,r=>r.workspaceId===w.id],['state=failed',r=>r.state==='failed']]){
  const items=(await api('history?'+query+'&limit=50')).items;assert.ok(items.every(check),'filter '+query)
 }
 const future=(await api('history?from='+encodeURIComponent(new Date(Date.now()+3600e3).toISOString()))).items
 assert.equal(future.length,0,'time filter returned future-less runs')
 record.filters=['case','state','chain','actor','workspace','time']

 // Compatible comparison of the same case, and an explained refusal otherwise.
 for(const run of sessions)await page.getByLabel('비교 선택 '+run.id,{exact:true}).check()
 const compared=page.waitForResponse(r=>r.url().endsWith('/api/v1/history/compare'))
 await page.getByRole('button',{name:'선택한 실행 비교',exact:true}).click()
 const comparison=await (await compared).json()
 assert.equal(comparison.comparable,true,JSON.stringify(comparison))
 assert.ok(comparison.results.some(r=>Object.values(r.statuses??{}).every(s=>s==='pass')),'compared verdicts missing')
 await page.getByLabel('실행 비교 결과',{exact:true}).waitFor()
 const refused=await api('history/compare','POST',{runIds:[sessions[0].id,otherSession.id]})
 assert.equal(refused.comparable,false);assert.ok(refused.limitations.length>0,'incompatibility reason missing')
 record.comparison={comparable:comparison.comparable,refusal:refused.limitations}

 // Detail and secret-free export from the UI.
 await page.getByRole('button',{name:new RegExp(sessions[0].id.replace(/[.*+?^${}()|[\]\\]/g,'\\$&'))}).first().click()
 await page.getByLabel('실행 상세 결과',{exact:true}).waitFor()
 const download=page.waitForEvent('download')
 await page.getByRole('button',{name:'결과 내보내기',exact:true}).click()
 const file=await download;await file.saveAs(out+'/history-export.json')
 const exported=fs.readFileSync(out+'/history-export.json','utf8')
 for(const secret of [f.password,f.setupToken,credentialSecret])assert.ok(!exported.includes(secret),'export leaks private data')
 assert.ok(JSON.parse(exported).files['session.json'].includes('history-shared-case'),'export lacks the engine session')
 await page.screenshot({path:out+'/history-web09-desktop.png',fullPage:true})

 // A running execution cannot be deleted; non-administrators cannot delete at all.
 const live=await testRun(shared,'retain')
 let liveCapture
 for(let i=0;i<50;i++){liveCapture=(await api('history?search='+live+'&limit=50')).items.find(r=>r.jobId===live&&!['succeeded','failed','cancelled','interrupted'].includes(r.state));if(liveCapture)break;await sleep(200)}
 assert.ok(liveCapture,'running execution not listed')
 await api('history/'+liveCapture.id,'DELETE',undefined,undefined,409)
 for(const user of ['history-operator','history-viewer']){const ctx=await owned.browser.newContext();const s=await request(ctx,undefined,'auth/login','POST',{username:user,password:f.password});await request(ctx,s.csrfToken,'history/job-'+firstJob.id,'DELETE',undefined,undefined,403);await ctx.close()}
 await finish(live)

 // Administrator deletion keeps shared declarations and the other runs.
 await api('history/job-'+firstJob.id,'DELETE',undefined,undefined,204)
 await api('history/job-'+firstJob.id,'GET',undefined,undefined,404)
 assert.equal((await api('documents/'+shared.id)).id,shared.id,'shared case removed with a run')
 assert.equal((await api('history/job-'+secondJob.id)).id,'job-'+secondJob.id,'another run removed')
 const before=(await api('history?limit=50')).items.map(r=>r.id).sort()

 // Restart: runs, tombstone and archived windows stay without expiry.
 fs.writeFileSync(f.runtime+'/restart.request','explicit fixture dashboard restart')
 for(let i=0;i<300&&!fs.existsSync(f.runtime+'/restart.response');i++)await sleep(100)
 const restarted=JSON.parse(fs.readFileSync(f.runtime+'/restart.response','utf8'));assert.ok(!restarted.error&&restarted.oldPid!==restarted.newPid)
 session=await request(context,undefined,'auth/login','POST',{username:'history-admin',password:f.password})
 const after=(await api('history?limit=50')).items.map(r=>r.id).sort()
 assert.deepEqual(after,before,'history changed across restart')
 assert.ok(!after.includes('job-'+firstJob.id),'deleted run reappeared')
 const kept=await api('history/job-'+thirdJob.id)
 assert.ok(kept.summary.observations,'archived window lost across restart')
 await page.setViewportSize({width:390,height:844});await page.reload();await page.getByLabel('히스토리 검색',{exact:true}).waitFor()
 await page.screenshot({path:out+'/history-web09-mobile.png',fullPage:true})
 assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'history mobile overflow')
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...record,runningDeletionRefused:true,nonAdminDeletionRefused:true,sharedDeclarationKept:true,restartRetention:true,exportSecretFree:true,runs:{first:firstJob.id,second:secondJob.id,third:thirdJob.id,live},seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
