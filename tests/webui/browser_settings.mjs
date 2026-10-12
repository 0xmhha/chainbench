// Settings on a real dashboard with a native stablenet network: an administrator
// creates an operator on the settings screen and reads the audit trail there;
// the operator sees neither; the operator registers an account key, an attach
// job uses it, and revoking it on the settings screen cancels that job while
// the network's nodes are kept. Development proof only; no criterion awarded.
import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser()
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
const terminal=['succeeded','failed','cancelled','interrupted']
const accountKey='0x8f2a55949038a9610f50fb23b5883af3b4ecb3c3bb792cbcefbd1542c692be63'
async function request(ctx,token,path,method='GET',data,key,expected=200){
 const response=await ctx.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(token?{'X-CSRF-Token':token}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const raw=await response.text();assert.equal(response.status(),expected,`${method} ${path}: ${raw}`)
 assert.ok(!raw.includes(f.password)&&!raw.includes(accountKey.slice(2)),'private data leaked')
 return raw?JSON.parse(raw):undefined
}
const live=pid=>{try{return execFileSync('ps',['-p',String(pid),'-o','command='],{encoding:'utf8'}).includes(f.runtime)}catch{return false}}
async function settle(api,id){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(terminal.includes(job.state))return job;await sleep(200)}throw new Error('job did not settle')}
async function running(api,id,phase){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(job.state==='running'&&job.phases.some(p=>p.name===phase&&p.state==='running'))return job;if(terminal.includes(job.state))throw new Error('job ended early: '+JSON.stringify(job));await sleep(200)}throw new Error('job never ran '+phase)}
const observed={}
let page
try{
 // 1. The administrator creates the operator and reads the audit trail on Settings.
 const adminCtx=await owned.browser.newContext({viewport:{width:1440,height:1100}})
 await request(adminCtx,undefined,'bootstrap','POST',{username:'settings-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 page=await adminCtx.newPage()
 await page.goto(f.url+'/settings')
 await page.getByLabel('새 사용자 이름',{exact:true}).fill('settings-operator')
 await page.getByLabel('새 사용자 비밀번호',{exact:true}).fill(f.password)
 await page.getByLabel('새 사용자 역할',{exact:true}).selectOption('operator')
 await page.getByRole('button',{name:'사용자 만들기',exact:true}).click()
 await page.getByTestId('user-status').filter({hasText:'settings-operator 계정을 만들었습니다'}).waitFor({timeout:15000})
 await page.getByTestId('user-row').filter({hasText:'settings-operator'}).waitFor()
 // The audit entry is written after the response the screen reacted to, so
 // the period is queried again until it appears.
 let auditText=''
 for(let i=0;i<20&&!/users/.test(auditText);i++){await page.getByRole('button',{name:'기간 조회',exact:true}).click();await page.getByTestId('audit-row').first().waitFor({timeout:15000});auditText=(await page.getByTestId('audit-row').allTextContents()).join('\n');if(!/users/.test(auditText))await sleep(500)}
 assert.match(auditText,/users/,'user creation is not in the audit trail')
 observed.adminAudit=await page.getByTestId('audit-row').count()
 await page.screenshot({path:out+'/settings-admin.png',fullPage:true})

 // 2. The operator has no user management or audit; the API refuses audit too.
 const ctx=await owned.browser.newContext({viewport:{width:1440,height:1100}})
 const s=await request(ctx,undefined,'auth/login','POST',{username:'settings-operator',password:f.password})
 const api=(path,method,data,key,expected)=>request(ctx,s.csrfToken,path,method,data,key,expected)
 page=await ctx.newPage()
 await page.goto(f.url+'/settings')
 await page.getByRole('region',{name:'개인 자격증명'}).waitFor({timeout:15000})
 assert.equal(await page.getByRole('region',{name:'사용자 관리'}).count(),0,'an operator sees user management')
 assert.equal(await page.getByRole('region',{name:'감사 기록'}).count(),0,'an operator sees the audit trail')
 await api('audit',undefined,undefined,undefined,403)
 observed.operatorHasNoAdminScreens=true

 // 3. The operator registers an account key on Settings.
 await page.getByLabel('자격증명 종류',{exact:true}).selectOption('account-key')
 await page.getByLabel('자격증명 이름',{exact:true}).fill('payer key')
 await page.getByLabel('테스트 계정 개인키',{exact:true}).fill(accountKey)
 await page.getByRole('button',{name:'개인 자격증명 저장',exact:true}).click()
 await page.getByTestId('credential-status').filter({hasText:'저장했습니다'}).waitFor({timeout:15000})
 const credential=(await api('credentials')).items.find(c=>c.label==='payer key')
 assert.ok(credential&&credential.kind==='account-key')

 // 4. A retained network, and an attach job that signs with that key.
 const paths={version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}
 const set=await api('documents','POST',{kind:'server-set',name:'settings pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:40300,step:10},rpc:{base:12900,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'settings paths',contractVersion:'2',assetRefs:[],content:paths},undefined,201)
 const w=await api('workspaces','POST',{name:'settings chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const preset={chain:'stablenet',binaries:{default:'gstable'},topology:{bp:4},launch:{all:{ipcdisable:true}}}
 const long=await api('documents','POST',{kind:'case',name:'settings compose',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'settings-compose',chainPreset:preset,steps:[{do:'waitBlock',on:'node1',target:1000000000,timeout:'5m'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}},undefined,201)
 const base={workspaceId:w.id,documentRefs:w.documents,assetRefs:['stablenet'],retention:'retain',arguments:{manifestId:'stablenet',assetId:'stablenet',serverRef:'local'}}
 const startJob=async input=>{const p=await api('plans','POST',input,undefined,201);return (await api('jobs','POST',{planId:p.id},p.id,202)).id}
 const compose=await startJob({...base,operation:'test.run',arguments:{...base.arguments,caseRefs:[{id:long.id,revision:long.revision}]}})
 await running(api,compose,'test.run')
 const recordPath=f.store+'/networks/'+w.id+'/chain-record.json'
 for(let i=0;i<900;i++){let up=false;try{const r=JSON.parse(fs.readFileSync(recordPath,'utf8'));up=r.nodes.length===4&&r.nodes.every(n=>n.pid>0&&live(n.pid))}catch{}if(up)break;await sleep(200)}
 await api('jobs/'+compose+'/cancel','POST',{},undefined,202)
 assert.equal((await settle(api,compose)).nodeDisposition,'retained')
 const record=JSON.parse(fs.readFileSync(recordPath,'utf8'))
 assert.ok(record.nodes.every(n=>n.pid>0&&live(n.pid)),'the composed network is not running')
 const attach=await api('documents','POST',{kind:'case',name:'settings attach',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'settings-attach',requires:['rpc'],chainPreset:{chain:'stablenet',attach:{rpc:['http://203.0.113.9:8545'],keysDir:'presets/keys',provides:['rpc','consensus']},accounts:{payer:{keyFile:'${PAYER_KEY_FILE}'}}},steps:[{do:'waitBlock',on:'node1',target:1000000000,timeout:'5m'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}},undefined,201)
 const jobId=await startJob({...base,operation:'test.attach',accountBindings:{payer:credential.id},arguments:{...base.arguments,caseRefs:[{id:attach.id,revision:attach.revision}]}})
 await running(api,jobId,'test.run').catch(async()=>running(api,jobId,'test.attach'))

 // 5. Settings shows the job holding the key; revoking it cancels the job and keeps the nodes.
 observed.liveBeforeRevoke=record.nodes.map(n=>live(n.pid))
 await page.reload()
 const row=page.getByTestId('credential-row').filter({hasText:'payer key'})
 await row.filter({hasText:'진행 중 작업 1개'}).waitFor({timeout:15000})
 page.once('dialog',d=>{observed.revokeWarning=d.message();d.accept()})
 await row.getByRole('button',{name:'payer key 폐기',exact:true}).click()
 const job=await settle(api,jobId)
 assert.equal(job.state,'cancelled',JSON.stringify(job));assert.equal(job.cancelReason,'credential_revoked');assert.equal(job.nodeDisposition,'retained')
 observed.liveAfterRevoke=record.nodes.map(n=>live(n.pid))
 observed.recordAfter=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8')).nodes.map(n=>n.pid)
 fs.writeFileSync(out+'/diagnostic.json',JSON.stringify(observed,null,2))
 assert.ok(record.nodes.every(n=>n.pid>0&&live(n.pid)),'revocation stopped the network')
 await row.filter({hasText:'폐기됨'}).waitFor({timeout:15000})
 assert.match(observed.revokeWarning,/진행 중 작업 1개/)
 await page.screenshot({path:out+'/settings-revoked.png',fullPage:true})
 observed.revocation={jobId,cancelReason:job.cancelReason,disposition:job.nodeDisposition}
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observed,seedAcceptanceAwarded:false},null,2))
}catch(error){
 if(page){await page.screenshot({path:out+'/failure.png',fullPage:true}).catch(()=>{});fs.writeFileSync(out+'/failure.txt',(await page.locator('[role=alert],[role=status]').allTextContents().catch(()=>[])).join('\n'))}
 throw error
}finally{await owned.stop()}
