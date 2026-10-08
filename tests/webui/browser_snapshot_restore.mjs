import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext({viewport:{width:1440,height:1100}})
let session
async function api(path,method='GET',data,key,expected=200){
 const response=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(key?{'Idempotency-Key':key}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const raw=await response.text();assert.equal(response.status(),expected,`${method} ${path}: ${raw}`)
 assert.ok(!raw.includes(f.password)&&!raw.includes(f.setupToken),'private bootstrap data leaked')
 return raw?JSON.parse(raw):undefined
}
// Preserve exact observation evidence; a failed liveness assertion must name each node's state/reason.
function processRows(nodes){return nodes.map(n=>{try{return {pid:n.pid,ps:execFileSync('ps',['-p',String(n.pid),'-o','pid=,ppid=,stat=,command='],{encoding:'utf8'}).trim()}}catch(error){return {pid:n.pid,ps:null,exit:error.status}}})}
async function observeNodes(id,nodes,label){const observed=await api('networks/'+id+'/observations');fs.writeFileSync(out+'/observations-'+label+'.json',JSON.stringify({at:new Date().toISOString(),observed,processes:processRows(nodes)},null,2));return observed}
function assertRunning(observed,label){assert.ok(observed.nodes.every(n=>n.state==='running'),label+' observation not running: '+JSON.stringify(observed.nodes.map(n=>({id:n.id,state:n.state,reason:n.observationReason,pid:n.pid,observedPid:n.observedPid}))))}
async function waitJob(id){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){assert.equal(job.state,'succeeded',JSON.stringify(job));return job}await new Promise(r=>setTimeout(r,200))}throw new Error('native snapshot job timed out')}
try{
 session=await api('bootstrap','POST',{username:'snapshot-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 const page=await context.newPage();await page.goto(f.url+'/monitoring')
 await page.getByRole('heading',{name:'실행 상태',exact:true}).waitFor({timeout:10000})
 await page.getByTestId('job-observation-health').filter({hasText:'연결됨'}).waitFor()
 const set=await api('documents','POST',{kind:'server-set',name:'snapshot pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39100,step:10},rpc:{base:11700,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'snapshot paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},undefined,201)
 const w=await api('workspaces','POST',{name:'snapshot native chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const c=await api('documents','POST',{kind:'case',name:'snapshot real test',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'snapshot-real-test',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}},undefined,201)
 const base={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local'}}
 const plan=await api('plans','POST',{...base,operation:'test.run',arguments:{...base.arguments,caseRefs:[{id:c.id,revision:c.revision}]}},undefined,201)
 const accepted=await api('jobs','POST',{planId:plan.id},plan.id,202)
 await context.setOffline(true)
 const terminal=await waitJob(accepted.id) // Independent API polling while browser networking is offline.
 assert.equal(terminal.runIds.length,1,'actual engine test session missing')
 await context.setOffline(false)
 const row=page.locator('[data-job-id="'+accepted.id+'"]')
 await row.filter({hasText:'succeeded'}).waitFor({timeout:20000})
 await page.getByTestId('job-observation-health').filter({hasText:'연결됨'}).waitFor()
 const before=await api('snapshot'),recordPath=f.store+'/networks/'+w.id+'/chain-record.json'
 const record=fs.readFileSync(recordPath),nodes=JSON.parse(record).nodes
 assert.ok(nodes.every(n=>n.pid>0),'test did not retain native processes')
 for(const node of nodes){const command=execFileSync('ps',['-p',String(node.pid),'-o','command='],{encoding:'utf8'});assert.ok(command.includes(f.runtime)&&command.includes(node.dataDir),'record is not a live owned native process')}
 const rpc=await context.request.post('http://127.0.0.1:11700/',{data:{jsonrpc:'2.0',id:1,method:'eth_getBlockByNumber',params:['0x0',false]}})
 assert.equal(rpc.status(),200);assert.ok((await rpc.json()).result?.hash,'native database genesis unavailable')
 assert.ok(before.networks.every(n=>n.nodes.every(node=>node.state==='unknown'&&node.supportedControls.length===0)),'cached snapshot grants controls or claims liveness')
 await page.screenshot({path:out+'/snapshot-desktop.png',fullPage:true})
 assertRunning(await observeNodes(w.id,nodes,'before-restart'),'before restart')
 fs.writeFileSync(f.runtime+'/restart.request','explicit fixture dashboard restart')
 for(let i=0;i<300&&!fs.existsSync(f.runtime+'/restart.response');i++)await new Promise(r=>setTimeout(r,100))
 const restarted=JSON.parse(fs.readFileSync(f.runtime+'/restart.response','utf8'));assert.ok(!restarted.error&&restarted.oldPid!==restarted.newPid)
 // Account sessions are intentionally process-local. Reauthenticate through
 // the actual login UI before reading the restarted service's durable jobs.
 const expired=await context.request.get(f.url+'/api/v1/auth/me');assert.equal(expired.status(),401)
 await page.getByLabel('Account username',{exact:true}).fill('snapshot-admin')
 await page.getByLabel('Account password',{exact:true}).fill(f.password)
 await page.getByRole('button',{name:'Sign in',exact:true}).click()
 await page.waitForFunction(old=>{const cursor=document.querySelector('[data-testid="job-observation-health"]')?.getAttribute('data-cursor');return !!cursor&&cursor.split(':')[0]!==old},before.cursor.split(':')[0],{timeout:20000})
 await page.getByTestId('job-observation-health').filter({hasText:'연결됨'}).waitFor()
 session=await api('auth/me')
 const after=await api('snapshot');assert.equal(after.version,before.version);assert.notEqual(after.cursor.split(':')[0],before.cursor.split(':')[0]);assert.deepEqual(after.jobs,before.jobs)
 assert.deepEqual(fs.readFileSync(recordPath),record,'snapshot restart changed live records')
 await row.filter({hasText:'succeeded'}).waitFor()
 await page.setViewportSize({width:390,height:844});await page.screenshot({path:out+'/snapshot-mobile.png',fullPage:true})
 assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'monitoring mobile overflow')
 assertRunning(await observeNodes(w.id,nodes,'after-restart'),'after restart')
 const stopPlan=await api('plans','POST',{...base,operation:'node.stop',nodeIds:['node1'],arguments:{...base.arguments,validators:4}},undefined,201)
 const stop=await api('jobs','POST',{planId:stopPlan.id},stopPlan.id,202);await waitJob(stop.id)
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),job:terminal,disconnectRestored:true,serverRestartRestoredAfterReauthentication:true,noAutomaticExecution:true,cachedPIDsNotLiveness:true,actualNativeProcessesAndGenesis:true,beforeCursor:before.cursor,afterCursor:after.cursor,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
