import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2), f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(), browser=owned.browser
const context=await browser.newContext({viewport:{width:1440,height:1100}})
const jobs=[]
let session
async function api(path,method='GET',data,key,expected=200,revision){
  const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(revision?{'If-Match':String(revision)}:{}),...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
  const text=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${text}`)
  assert.ok(!text.includes(f.password)&&!text.includes(f.setupToken),'bootstrap secret leaked')
  return text?JSON.parse(text):undefined
}
async function waitJob(id,state){
  for(let i=0;i<900;i++){
    const job=await api('jobs/'+id)
    if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,state,JSON.stringify(job));jobs.push(job);return job}
    await new Promise(r=>setTimeout(r,200))
  }
  throw new Error('test job timed out')
}
function input(chain,id,pass=true){return {kind:'case',name:id,contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id,chainPreset:{chain,binaries:{default:chain==='stablenet'?'gstable':'gwemix'},topology:{bp:4},launch:chain==='wemix'?undefined:{all:{ipcdisable:true}}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:pass?'GreaterOrEqual':'Less',is:pass?'$head':0}]}}}
try{
  session=await api('bootstrap','POST',{username:'test-admin',password:f.password,setupToken:f.setupToken},undefined,201)
  for(const [i,chain] of ['wbft','stablenet','wemix'].entries()){
    const set=await api('documents','POST',{kind:'server-set',name:chain+' test pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:37000+i*100,step:10},rpc:{base:10600+i*100,step:10}}}}},undefined,201)
    const config=await api('documents','POST',{kind:'workspace-config',name:chain+' test paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d'+i,paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/untrusted-output-'+chain},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
    const w=await api('workspaces','POST',{name:chain+' browser tests',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
    const c=await api('documents','POST',input(chain,'saved-'+chain),undefined,201)
    const base={workspaceId:w.id,operation:'test.run',documentRefs:w.documents,assetRefs:[chain],retention:'retain',arguments:{manifestId:chain,assetId:chain,serverRef:'local',caseRefs:[{id:c.id,revision:c.revision}]}}
    const plan=await api('plans','POST',base,undefined,201)
    assert.ok(plan.changes.some(v=>v.includes(c.id)&&v.includes('r1')),'plan omits selected immutable case')
    const changed=await api('documents/'+c.id,'PATCH',input(chain,'edited-'+chain,false),undefined,200,c.revision)
    await api('jobs','POST',{planId:plan.id},plan.id,409)
    base.arguments.caseRefs=[{id:changed.id,revision:changed.revision}]
    const failPlan=await api('plans','POST',base,undefined,201)
    const failJob=await api('jobs','POST',{planId:failPlan.id},failPlan.id,202)
    // An accepted job must keep the reviewed bytes despite a subsequent edit.
    const newest=await api('documents/'+changed.id,'PATCH',input(chain,'newest-'+chain),undefined,200,changed.revision)
    const failed=await waitJob(failJob.id,'failed')
    assert.equal(failed.runIds.length,1,'test verdict lost its actual engine session')
    assert.ok(failed.partialEffects.some(v=>v.includes('Verified test binary provisioned:')&&v.includes('SHA-256')),'provisioned binary effect was lost')
    const history=(await api('history?caseId=edited-'+chain)).items.filter(r=>r.summary.kind==='engine-session')
    assert.equal(history.length,1);assert.equal(history[0].state,'failed');assert.equal(history[0].jobId,failed.id)
    const detail=await api('history/'+history[0].id)
    assert.equal(detail.summary.counts.fail,1)
    assert.ok(detail.artifactRefs.some(p=>p.startsWith('tests/')),'actual assertion evidence not captured')
    const exported=await api('history/'+history[0].id+'/export')
    assert.equal(exported.run.summary.tests[0].id,'edited-'+chain)
    assert.ok(exported.files['session.json'].includes('edited-'+chain),'accepted case was replaced by shared edit')
    if(i===0){
      const page=await context.newPage();await page.goto(f.url+'/tests')
      await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
      await page.getByLabel('작업 매니페스트',{exact:true}).selectOption(chain)
      await page.getByLabel('작업 바이너리',{exact:true}).selectOption(chain)
      await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
      await page.getByLabel('실행 케이스 '+newest.id,{exact:true}).check()
      await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
      await page.getByLabel('실행 계획',{exact:true}).waitFor()
      await page.getByLabel('서버 실행 작업',{exact:true}).screenshot({path:out+'/test-plan.png'})
      const response=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
      await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click()
      const started=await response;assert.equal(started.status(),202)
      await waitJob((await started.json()).id,'succeeded')
      await page.getByText('세션 결과 보기',{exact:true}).first().waitFor()
      await page.screenshot({path:out+'/test-results.png',fullPage:true})
    }
  }
  fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:browser.version(),jobs,realEngineSessions:true,staleCaseRejected:true,acceptedCaseEditIgnored:true,historyAssertionEvidence:true,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
