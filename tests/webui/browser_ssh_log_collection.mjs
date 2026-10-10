import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const key=fs.readFileSync(f.runtime+'/ssh/client','utf8')
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext({viewport:{width:1440,height:1100}})
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
let session
async function api(path,method='GET',data,expected=200,idempotency){
 const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(idempotency?{'Idempotency-Key':idempotency}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const text=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${text}`)
 assert.ok(!text.includes(key.trim())&&!text.includes(f.password),'credential material leaked')
 return text?JSON.parse(text):undefined
}
async function waitJob(id,states=['succeeded']){for(let i=0;i<1500;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.ok(states.includes(job.state),JSON.stringify(job));return job}await sleep(200)}throw new Error('SSH job timed out')}
const sshdLog=()=>fs.readFileSync(f.runtime+'/ssh/sshd.log','utf8')
const accepted=()=>(sshdLog().match(/Accepted publickey/g)||[]).length
const logsOf=(ws,node,query='')=>api('nodes/'+ws+'.'+node+'/logs'+query)
try{
 session=await api('bootstrap','POST',{username:'ssh-log-admin',password:f.password,setupToken:f.setupToken},201)
 const set=await api('documents','POST',{kind:'server-set',name:'SSH log fixture',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'owned-ssh',addr:'localhost.'}],slots:4,ports:{p2p:{base:35500,step:10},rpc:{base:10800,step:10}}},ssh:{port:f.ssh.port,known_hosts_file:f.knownHosts}}},201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'SSH log paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/artifacts'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},201)
 const w=await api('workspaces','POST',{name:'SSH log fixture',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},201)
 const credential=await api('credentials','POST',{label:'Owned SSH log fixture',kind:'private-key',sshUser:f.ssh.user,privateKey:key},201)
 await api(`workspaces/${w.id}/credential-bindings`,'PUT',{serverRef:'owned-ssh',credentialId:credential.id})
 const bindings={'owned-ssh':credential.id}
 const deploy=await api('plans','POST',{workspaceId:w.id,operation:'chain.deploy',documentRefs:w.documents,assetRefs:['wbft'],credentialBindings:bindings,retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'owned-ssh',validators:4}},201)
 await waitJob((await api('jobs','POST',{planId:deploy.id},202,deploy.id)).id)
 const record=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
 assert.equal(record.nodes.length,4);assert.ok(record.nodes.every(n=>n.pid>0),'SSH deployment did not start native nodes')

 // Without an operator's collection job, remote logs are reported as absent.
 let before
 for(let i=0;i<30;i++){before=await logsOf(w.id,'node1');if(before.coverage.gaps.some(g=>g.reason==='node1 logs remote_log_collection_unavailable'))break;await sleep(1000)}
 assert.equal(before.entries.length,0,'remote lines archived without a collection job')
 assert.ok(before.coverage.gaps.some(g=>g.reason==='node1 logs remote_log_collection_unavailable'),'missing remote collection not reported')

 // The operator starts collection through the job form with their own binding.
 const page=await context.newPage();await page.goto(f.url+'/chains')
 await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
 await page.getByLabel('작업 종류',{exact:true}).selectOption('network.monitor')
 await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('owned-ssh')
 const planned=page.waitForResponse(r=>r.url()===f.url+'/api/v1/plans'&&r.request().method()==='POST')
 await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
 const plannedResponse=await planned;assert.equal(plannedResponse.status(),201,await plannedResponse.text())
 await page.getByLabel('실행 계획',{exact:true}).waitFor()
 const start=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
 await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click()
 const startResponse=await start;assert.equal(startResponse.status(),202,await startResponse.text())
 const monitor=await startResponse.json()
 let collected
 for(let i=0;i<60;i++){collected=await logsOf(w.id,'node1');if(collected.entries.length>0)break;await sleep(1000)}
 const nodeLog=fs.readFileSync(record.nodes[0].logPath,'utf8')
 assert.ok(collected.entries.length>0&&collected.entries.every(e=>nodeLog.includes(e.text)),'remote lines are not the native node log')
 assert.ok(['source','mixed'].includes(collected.coverage.timestampSource))
 assert.equal((await api('jobs/'+monitor.id)).state,'running')

 // Collection does not block owned node controls on the same network.
 const stopPlan=await api('plans','POST',{workspaceId:w.id,operation:'node.stop',documentRefs:w.documents,assetRefs:['wbft'],credentialBindings:bindings,nodeIds:['node4'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'owned-ssh'}},201)
 assert.equal((await api('plans/'+stopPlan.id+'/conflicts')).items.length,0,'log collection blocks node controls')
 await waitJob((await api('jobs','POST',{planId:stopPlan.id},202,stopPlan.id)).id)

 await page.goto(f.url+'/monitoring')
 const chart=page.getByTestId('metric-chart-block_height');await chart.waitFor({timeout:30000})
 await chart.locator('circle[data-time]').last().click()
 const panel=page.getByTestId('time-linked-logs');await panel.getByLabel('로그 노드').selectOption('node1')
 await page.locator('[data-testid="time-linked-logs"][data-log-node="node1"]').waitFor({timeout:20000})
 await panel.locator('[data-log-time]').first().waitFor({timeout:20000})
 const shown=await panel.locator('[data-log-time] .log-text').allTextContents()
 assert.ok(shown.length>0&&shown.every(t=>fs.readFileSync(record.nodes[0].logPath,'utf8').includes(t)),'linked remote logs are not the native node log')
 await page.screenshot({path:out+'/ssh-log-collection.png',fullPage:true})

 // Revoking the credential cancels collection and opens no further SSH session.
 await api('credentials/'+credential.id,'DELETE',undefined,204)
 const cancelled=await waitJob(monitor.id,['cancelled'])
 assert.equal(cancelled.cancelReason,'credential_revoked');assert.equal(cancelled.nodeDisposition,'retained')
 const revokedAt=Date.now(),sessions=accepted()
 await sleep(20000)
 assert.equal(accepted(),sessions,'SSH sessions opened after revocation')
 const after=await logsOf(w.id,'node1','?from='+encodeURIComponent(new Date(revokedAt+2000).toISOString())+'&to='+encodeURIComponent(new Date().toISOString()))
 assert.equal(after.entries.length,0,'lines archived after revocation')
 assert.ok(/INFO \[\d\d-\d\d\|/.test(fs.readFileSync(record.nodes[0].logPath,'utf8').slice(-2000)),'native node stopped logging; revocation check would be vacuous')
 assert.ok(fs.statSync(record.nodes[0].logPath).mtimeMs>revokedAt+2000,'node log did not grow after revocation; check would be vacuous')
 assert.ok(after.coverage.gaps.some(g=>g.reason==='node1 logs remote_log_collection_unavailable'),'absence after revocation not reported')
 fs.writeFileSync(out+'/logs-after-revocation.json',JSON.stringify(after,null,2))
 // A newly registered credential is an explicit new choice; it stops the
 // remaining nodes so the runner can inspect their native databases.
 const renewed=await api('credentials','POST',{label:'Renewed SSH log fixture',kind:'private-key',sshUser:f.ssh.user,privateKey:key},201)
 await api(`workspaces/${w.id}/credential-bindings`,'PUT',{serverRef:'owned-ssh',credentialId:renewed.id})
 for(const node of ['node1','node2','node3']){
  const plan=await api('plans','POST',{workspaceId:w.id,operation:'node.stop',documentRefs:w.documents,assetRefs:['wbft'],credentialBindings:{'owned-ssh':renewed.id},nodeIds:[node],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'owned-ssh'}},201)
  await waitJob((await api('jobs','POST',{planId:plan.id},202,plan.id)).id)
 }
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),monitorJob:{id:cancelled.id,state:cancelled.state,cancelReason:cancelled.cancelReason},collectedLines:collected.entries.length,nodeControlNotBlocked:true,noSSHAfterRevocation:true,acceptedSessions:sessions,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
