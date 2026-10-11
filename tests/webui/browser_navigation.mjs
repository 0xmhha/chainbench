// Shared selection, top bar, dashboard and job detail on a real dashboard with
// a native wbft network: a workspace picked on the chain screen is the
// monitoring selection, a started job opens its own page, the dashboard lists
// it while it runs and links to that page, and cancelling there shows the
// outcome. Development proof only; no Seed criterion is awarded.
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
const observed={}
let page
try{
 const adminCtx=await owned.browser.newContext()
 const adminSession=await request(adminCtx,undefined,'bootstrap','POST',{username:'nav-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 await request(adminCtx,adminSession.csrfToken,'users','POST',{username:'nav-operator',password:f.password,role:'operator'},undefined,201)
 const ctx=await owned.browser.newContext({viewport:{width:1440,height:1100}})
 const s=await request(ctx,undefined,'auth/login','POST',{username:'nav-operator',password:f.password})
 const api=(path,method,data,key,expected)=>request(ctx,s.csrfToken,path,method,data,key,expected)
 const paths={version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}
 const set=await api('documents','POST',{kind:'server-set',name:'nav pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39300,step:10},rpc:{base:11900,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'nav paths',contractVersion:'2',assetRefs:[],content:paths},undefined,201)
 const w=await api('workspaces','POST',{name:'nav chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const preset={chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}}
 // The case waits for an unreachable height so the job is still running when checked.
 const long=await api('documents','POST',{kind:'case',name:'nav long case',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'nav-long',chainPreset:preset,steps:[{do:'waitBlock',on:'node1',target:1000000000,timeout:'5m'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}},undefined,201)

 page=await ctx.newPage()
 await page.goto(f.url+'/chains')
 // 1. With no chain chosen the top bar asks for one.
 await page.getByTestId('topbar-context').getByRole('link',{name:'체인을 선택하세요'}).waitFor({timeout:15000})
 observed.chainPrompt=true
 // 2. A workspace picked on the chain screen becomes the monitoring selection.
 await page.getByLabel('저장된 배포 workspace',{exact:true}).selectOption(w.id)
 await page.getByTestId('topbar-context').getByText('Workspace nav chain').waitFor({timeout:15000})
 await page.getByRole('navigation').getByRole('link',{name:/모니터링/}).click()
 await page.waitForURL(f.url+'/monitoring')
 await page.getByLabel('모니터링 Workspace',{exact:true}).locator(`option[value="${w.id}"]`).waitFor({state:'attached',timeout:15000})
 assert.equal(await page.getByLabel('모니터링 Workspace',{exact:true}).inputValue(),w.id,'monitoring did not keep the chain screen workspace')
 observed.workspaceCarriedToMonitoring=w.id
 // 3. The test screen starts from the same workspace; starting a job opens its page.
 await page.getByRole('navigation').getByRole('link',{name:/테스트/}).click()
 await page.waitForURL(f.url+'/tests')
 await page.getByLabel('작업 Workspace',{exact:true}).locator(`option[value="${w.id}"]`).waitFor({state:'attached',timeout:30000})
 assert.equal(await page.getByLabel('작업 Workspace',{exact:true}).inputValue(),w.id,'job form did not start from the selected workspace')
 await page.getByLabel('작업 매니페스트',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 바이너리',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 종류',{exact:true}).selectOption('test.run')
 await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
 await page.getByLabel('작업 종료 후 처리',{exact:true}).selectOption('cleanup')
 await page.getByLabel('실행 케이스 '+long.id,{exact:true}).check()
 await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
 const run=page.getByRole('button',{name:'검토한 계획 실행',exact:true})
 await run.waitFor({timeout:60000});await page.waitForFunction(()=>![...document.querySelectorAll('button')].find(b=>b.textContent==='검토한 계획 실행')?.disabled,null,{timeout:60000})
 await run.click()
 await page.waitForURL(/\/jobs\/[0-9a-f]{32}$/,{timeout:30000})
 const jobId=page.url().split('/').pop()
 await page.getByTestId('job-continues').waitFor()
 observed.startOpensDetail=jobId
 // 4. While the job runs the dashboard lists it, and its row opens the detail page.
 for(let i=0;i<900;i++){const job=await api('jobs/'+jobId);if(job.state==='running'&&job.phases.some(p=>p.name==='test.run'&&p.state==='running'))break;if(terminal.includes(job.state))throw new Error('job ended early: '+JSON.stringify(job));await sleep(200)}
 await page.getByRole('navigation').getByRole('link',{name:/대시보드/}).click()
 await page.waitForURL(f.url+'/')
 const row=page.getByRole('region',{name:'실행 중 작업'}).locator(`tr[data-job-id="${jobId}"]`)
 await row.waitFor({timeout:15000})
 assert.ok(Number(await page.getByTestId('running-count').textContent())>=1)
 await page.screenshot({path:out+'/dashboard-running.png',fullPage:true})
 await row.getByRole('link').click()
 await page.waitForURL(f.url+'/jobs/'+jobId)
 assert.match(await page.getByTestId('job-state').textContent(),/running/)
 observed.dashboardOpensDetail=true
 // 5. Cancelling on the detail page shows the outcome there, live.
 const pids=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8')).nodes.map(n=>n.pid)
 page.once('dialog',d=>d.accept())
 await page.getByRole('button',{name:'작업 취소',exact:true}).click()
 await page.waitForFunction(()=>/cancelled/.test(document.querySelector('[data-testid=job-state]')?.textContent??''),null,{timeout:120000})
 const disposition=await page.getByTestId('job-disposition').textContent()
 assert.match(disposition,/cleaned/,disposition)
 assert.equal(await page.getByTestId('job-failed-phase').textContent(),'없음')
 assert.equal(await page.getByTestId('job-unresolved').textContent(),'없음','a cleaned job still lists resources to check')
 assert.ok(pids.every(p=>!live(p)),'cleanup left owned processes running')
 await page.screenshot({path:out+'/job-detail-cancelled.png',fullPage:true})
 observed.cancelFromDetail={jobId,disposition}
 // 6. A reload of the job page restores it from the server.
 await page.reload();await page.getByTestId('job-state').waitFor()
 assert.match(await page.getByTestId('job-state').textContent(),/cancelled/)
 observed.reloadRestores=true
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observed,seedAcceptanceAwarded:false},null,2))
}catch(error){
 // Keep what the page showed when a step failed, then fail.
 if(page){await page.screenshot({path:out+'/failure.png',fullPage:true}).catch(()=>{});fs.writeFileSync(out+'/failure.txt',(await page.locator('[role=alert],[role=status]').allTextContents().catch(()=>[])).join('\n'))}
 throw error
}finally{await owned.stop()}
