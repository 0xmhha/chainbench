import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser()
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
async function request(ctx,token,path,method='GET',data,key,expected=200){
 const response=await ctx.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(token?{'X-CSRF-Token':token}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const raw=await response.text();assert.equal(response.status(),expected,`${method} ${path}: ${raw}`)
 assert.ok(!raw.includes(f.password)&&!raw.includes(f.setupToken),'private bootstrap data leaked')
 return raw?JSON.parse(raw):undefined
}
async function signIn(page,username){
 await page.getByLabel('Account username',{exact:true}).fill(username)
 await page.getByLabel('Account password',{exact:true}).fill(f.password)
 await page.getByRole('button',{name:'Sign in',exact:true}).click()
}
const observations={}
try{
 const first=await owned.browser.newContext({viewport:{width:1440,height:1100}})
 const session=await request(first,undefined,'bootstrap','POST',{username:'continuity-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 const api=(path,method,data,key,expected)=>request(first,session.csrfToken,path,method,data,key,expected)
 const set=await api('documents','POST',{kind:'server-set',name:'continuity pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39100,step:10},rpc:{base:11700,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'continuity paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
 const w=await api('workspaces','POST',{name:'continuity chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const c=await api('documents','POST',{kind:'case',name:'continuity test',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'continuity-test',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}},undefined,201)
 const base={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local'}}
 const plan=await api('plans','POST',{...base,operation:'test.run',arguments:{...base.arguments,caseRefs:[{id:c.id,revision:c.revision}]}},undefined,201)
 const page=await first.newPage();await page.goto(f.url+'/monitoring')
 await page.getByTestId('job-observation-health').filter({hasText:'연결됨'}).waitFor({timeout:15000})
 const job=await api('jobs','POST',{planId:plan.id},plan.id,202)

 // Sign out through the UI while the job is still in progress, then close the browser context.
 const running=await api('jobs/'+job.id);assert.ok(['accepted','running'].includes(running.state),'job ended before sign-out: '+running.state)
 const signedOutAt=new Date().toISOString()
 await page.getByRole('button',{name:'Sign out',exact:true}).click()
 await page.getByRole('button',{name:'Sign in',exact:true}).waitFor()
 await request(first,undefined,'auth/me','GET',undefined,undefined,401)
 await first.close()
 observations.signedOutAt=signedOutAt;observations.stateAtSignOut=running.state

 // A new browser context signs in later and finds the job finished on the server.
 await sleep(3000)
 const second=await owned.browser.newContext({viewport:{width:1440,height:1100}})
 const page2=await second.newPage();await page2.goto(f.url+'/monitoring');await signIn(page2,'continuity-admin')
 await page2.getByTestId('job-observation-health').filter({hasText:'연결됨'}).waitFor({timeout:20000})
 const session2=await request(second,undefined,'auth/me')
 let done
 for(let i=0;i<900;i++){done=await request(second,session2.csrfToken,'jobs/'+job.id);if(['succeeded','failed','cancelled','interrupted'].includes(done.state))break;await sleep(200)}
 assert.equal(done.state,'succeeded',JSON.stringify(done))
 const finished=done.phases.map(p=>p.finishedAt).filter(Boolean).sort().pop()
 assert.ok(finished>signedOutAt,'job did not progress after sign-out')
 assert.equal(done.runIds.length,1,'actual engine session missing')
 await page2.locator('[data-job-id="'+job.id+'"]').filter({hasText:'succeeded'}).waitFor({timeout:20000})
 observations.finishedAfterSignOut=finished;observations.job={id:done.id,state:done.state,runIds:done.runIds}

 // With the stream and snapshot unreachable the page keeps the last state,
 // reports the disconnect and marks the observation stale after five seconds.
 await page2.route('**/api/v1/events**',route=>route.abort())
 await page2.route('**/api/v1/snapshot**',route=>route.abort())
 await page2.getByRole('button',{name:'실행 상태 새로고침',exact:true}).click()
 const health=page2.getByTestId('job-observation-health')
 await health.filter({hasText:'연결 끊김'}).waitFor({timeout:20000})
 await page2.getByText('마지막 수신 상태를 표시합니다. 연결 복원 전에는 최신 상태로 판단하지 마세요.',{exact:true}).waitFor()
 await health.filter({hasText:'stale'}).waitFor({timeout:20000})
 await page2.screenshot({path:out+'/stream-disconnected.png',fullPage:true})
 const drops=await page2.getByText(/현재 서버의 전체 작업 구독자 전달 드롭: \d+ · 이벤트 버스 전달 드롭: \d+/).textContent()
 await page2.unroute('**/api/v1/events**');await page2.unroute('**/api/v1/snapshot**')
 await health.filter({hasText:'연결됨'}).waitFor({timeout:20000})
 const snapshot=await request(second,session2.csrfToken,'snapshot')
 assert.ok(snapshot.jobs.some(j=>j.id===job.id&&j.state==='succeeded'),'restored snapshot lacks the finished job')
 observations.streamDisconnectShown=true;observations.staleShown=true;observations.dropCounters=drops.trim();observations.reconnected=true
 await page2.setViewportSize({width:390,height:844});await page2.screenshot({path:out+'/continuity-mobile.png',fullPage:true})
 assert.ok(await page2.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'monitoring mobile overflow')
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observations,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
