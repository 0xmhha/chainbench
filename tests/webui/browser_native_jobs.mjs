import {chromium} from 'playwright'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2), f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const browser=await chromium.launch({headless:true,channel:process.env.WEBUI_BROWSER_CHANNEL??'chrome'})
let context=await browser.newContext({viewport:{width:1440,height:1100}})
const records=[]
async function api(path,method='GET',data,session,key,expected=200){
  const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
  const text=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${text}`)
  assert.ok(!text.includes(f.password)&&!text.includes(f.setupToken),'private bootstrap data leaked')
  return text?JSON.parse(text):undefined
}
async function waitJob(id){for(let i=0;i<600;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,'succeeded',JSON.stringify(job));records.push(job);return job}await new Promise(r=>setTimeout(r,200))}throw new Error('job timed out')}
try{
  let session=await api('bootstrap','POST',{username:'native-admin',password:f.password,setupToken:f.setupToken},undefined,undefined,201)
  const workspaces=[]
  for(const [i,chain] of ['wbft','stablenet','wemix'].entries()){
    const set=await api('documents','POST',{kind:'server-set',name:chain+' fixture pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:34000+i*100,step:10},rpc:{base:9600+i*100,step:10}}}}},session,undefined,201)
    const config=await api('documents','POST',{kind:'workspace-config',name:chain+' fixture paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/data-'+chain,paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/artifacts-'+chain},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},session,undefined,201)
    workspaces.push(await api('workspaces','POST',{name:chain+' fixture',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},session,undefined,201))
  }
  const page=await context.newPage();await page.goto(f.url+'/chains')
  await page.getByLabel('작업 Workspace',{exact:true}).selectOption(workspaces[0].id)
  await page.getByLabel('작업 매니페스트',{exact:true}).selectOption('wbft')
  await page.getByLabel('작업 바이너리',{exact:true}).selectOption('wbft')
  await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
  await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
  await page.getByLabel('실행 계획',{exact:true}).waitFor()
  await page.getByLabel('서버 실행 작업',{exact:true}).screenshot({path:out+'/plan-desktop.png'})
  const accepted=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
  await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click()
  const response=await accepted;assert.equal(response.status(),202);const first=await response.json()
  await context.close() // Accepted work must survive the browser disconnect.
  context=await browser.newContext({viewport:{width:390,height:844}})
  session=await api('auth/login','POST',{username:'native-admin',password:f.password})
  await waitJob(first.id)
  for(let i=1;i<workspaces.length;i++){
    const chain=['wbft','stablenet','wemix'][i],w=workspaces[i]
    const plan=await api('plans','POST',{workspaceId:w.id,operation:'chain.setup',documentRefs:w.documents,assetRefs:[chain],retention:'retain',arguments:{manifestId:chain,assetId:chain,serverRef:'local',validators:4}},session,undefined,201)
    const job=await api('jobs','POST',{planId:plan.id},session,plan.id,202)
    const duplicate=await api('jobs','POST',{planId:plan.id},session,plan.id,202);assert.equal(duplicate.id,job.id)
    await waitJob(job.id)
  }
  for(const operation of ['node.start','node.stop']){
    const w=workspaces[0],plan=await api('plans','POST',{workspaceId:w.id,operation,documentRefs:w.documents,assetRefs:['wbft'],nodeIds:['node1'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local',validators:4}},session,undefined,201)
    const job=await api('jobs','POST',{planId:plan.id},session,plan.id,202);await waitJob(job.id)
  }
  const networks=await api('networks');assert.equal(networks.items.length,3);assert.ok(networks.items.every(n=>n.nodes.length===4&&n.ownership==='owned'))
  assert.equal(networks.items.find(n=>n.workspaceId===workspaces[0].id).nodes[0].pid,0)
  const mobile=await context.newPage();await mobile.goto(f.url+'/chains');await mobile.getByLabel('작업 Workspace',{exact:true}).waitFor()
  await mobile.getByRole('button',{name:'새로고침',exact:true}).last().click()
  await mobile.screenshot({path:out+'/jobs-mobile.png',fullPage:true})
  assert.ok(await mobile.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth),'mobile horizontal overflow')
  await mobile.goto(f.url+'/history')
  const historyRows=(await api('history')).items;assert.equal(historyRows.length,5)
  await mobile.getByRole('button',{name:/^node.stop/}).waitFor()
  await mobile.setViewportSize({width:1440,height:1050})
  await mobile.screenshot({path:out+'/history-desktop.png',fullPage:true})
  for(const row of historyRows.slice(0,2))await mobile.getByRole('checkbox',{name:'비교 선택 '+row.id,exact:true}).check()
  await mobile.getByRole('button',{name:'선택한 실행 비교',exact:true}).click()
  await mobile.getByLabel('실행 비교 결과',{exact:true}).waitFor()
  assert.ok((await mobile.getByLabel('실행 비교 결과',{exact:true}).innerText()).includes('No common test cases'))
  await mobile.setViewportSize({width:390,height:844})
  await mobile.getByLabel('히스토리 검색',{exact:true}).fill('node.stop')
  const filtered=mobile.waitForResponse(r=>r.url().includes('/api/v1/history?')&&r.url().includes('search=node.stop'))
  await mobile.getByRole('button',{name:'필터 적용',exact:true}).click();await filtered
  await mobile.getByRole('button',{name:/^node.stop/}).click()
  const download=mobile.waitForEvent('download')
  await mobile.getByRole('button',{name:'결과 내보내기',exact:true}).click()
  const file=await download;await file.saveAs(out+'/history-export.json')
  const exported=JSON.parse(fs.readFileSync(out+'/history-export.json','utf8'))
  assert.equal(exported.run.state,'succeeded');assert.equal(exported.run.summary.operation,'node.stop')
  assert.ok(!JSON.stringify(exported).includes(f.password)&&!JSON.stringify(exported).includes(f.setupToken),'history export leaks bootstrap data')
  await mobile.getByRole('button',{name:'보관 결과 삭제',exact:true}).click()
  const deletion=mobile.waitForResponse(r=>r.url()===f.url+'/api/v1/history/'+exported.run.id&&r.request().method()==='DELETE')
  await mobile.getByRole('button',{name:'결과 사본 삭제 확정',exact:true}).click();assert.equal((await deletion).status(),204)
  assert.equal((await api('history')).items.length,4)
  assert.equal((await api('jobs')).items.length,5,'history deletion changed execution/idempotency records')
  assert.equal((await api('networks')).items.length,3,'history deletion changed owned networks')
  const comparison=await api('history/compare','POST',{runIds:historyRows.filter(r=>r.id!==exported.run.id).slice(0,2).map(r=>r.id)},session)
  assert.equal(comparison.comparable,false);assert.ok(comparison.limitations.some(l=>l.includes('No common test cases')))
  await mobile.screenshot({path:out+'/history-mobile.png',fullPage:true})
  assert.ok(await mobile.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth),'history mobile horizontal overflow')
  fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:browser.version(),jobs:records,networkCount:networks.items.length,disconnectObserved:true,idempotencyObserved:true,history:{captured:5,exported:exported.run.id,deleted:exported.run.id,remaining:4,jobRecordsPreserved:5,networksPreserved:3,missingTestsComparisonRejected:true}},null,2))
}finally{await browser.close()}
