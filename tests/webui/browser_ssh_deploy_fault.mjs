import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import crypto from 'node:crypto'
import {execFileSync} from 'node:child_process'
import {ownedGatedConnection} from './owned-ssh-gate.mjs'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const key=fs.readFileSync(f.runtime+'/ssh/client','utf8')
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext({viewport:{width:1440,height:1100}})
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
const digest=path=>crypto.createHash('sha256').update(fs.readFileSync(path)).digest('hex')
let session
async function api(path,method='GET',data,expected=200,idempotency){
 const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(idempotency?{'Idempotency-Key':idempotency}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const text=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${text}`)
 assert.ok(!text.includes(key.trim())&&!text.includes(f.password),'credential material leaked')
 return text?JSON.parse(text):undefined
}
async function settle(id){for(let i=0;i<1500;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state))return job;await sleep(200)}throw new Error('SSH job timed out')}
const runtimeProcesses=()=>execFileSync('ps',['-axo','command='],{encoding:'utf8'}).split('\n').filter(c=>c.includes(f.runtime+'/d/binaries/'))
try{
 session=await api('bootstrap','POST',{username:'ssh-fault-admin',password:f.password,setupToken:f.setupToken},201)
 const set=await api('documents','POST',{kind:'server-set',name:'SSH deploy fault',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'owned-ssh',addr:'localhost.'}],slots:4,ports:{p2p:{base:35500,step:10},rpc:{base:10800,step:10}}},ssh:{port:f.ssh.port,known_hosts_file:f.knownHosts}}},201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'SSH deploy fault paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/artifacts'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},201)
 const w=await api('workspaces','POST',{name:'SSH deploy fault',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},201)
 const credential=await api('credentials','POST',{label:'Owned SSH deploy fault',kind:'private-key',sshUser:f.ssh.user,privateKey:key},201)
 await api(`workspaces/${w.id}/credential-bindings`,'PUT',{serverRef:'owned-ssh',credentialId:credential.id})
 const input={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],credentialBindings:{'owned-ssh':credential.id},retention:'retain',operation:'chain.deploy',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'owned-ssh',validators:4}}
 const plan=await api('plans','POST',input,201)
 const job=await api('jobs','POST',{planId:plan.id},202,plan.id)
 // The node binary upload completes on the target, then its reply is held.
 for(let i=0;i<600&&!fs.existsSync(f.runtime+'/ssh/gate-copied');i++)await sleep(100)
 assert.ok(fs.existsSync(f.runtime+'/ssh/gate-copied'),'real SSH upload did not reach the owned gate')
 assert.equal((await api('jobs/'+job.id)).state,'running')
 const uploaded=execFileSync('find',[f.runtime+'/d/binaries','-type','f'],{encoding:'utf8'}).trim().split('\n').filter(p=>p&&digest(p)===f.baseSHA256)
 assert.equal(uploaded.length,1,'completed binary upload not found on the target')
 const sessions=(fs.readFileSync(f.runtime+'/ssh/sshd.log','utf8').match(/Accepted publickey/g)||[]).length
 // Cut only this fixture's own SSH connection carrying the paused upload.
 const connection=await ownedGatedConnection(f)
 process.kill(connection.pid,'SIGKILL')
 const failed=await settle(job.id)
 fs.writeFileSync(f.runtime+'/ssh/gate-release','release only the owned paused command')
 assert.equal(failed.state,'failed',JSON.stringify(failed))
 assert.ok(failed.partialEffects.length>0,'completed steps before the failure were not recorded')
 assert.ok(failed.unresolvedResources.length>0,'unfinished remote resources were not reported')
 assert.ok(failed.error?.message,'failure reason missing')
 assert.equal(digest(uploaded[0]),f.baseSHA256,'completed upload was rolled back or altered')
 // Nothing retries on its own, and no node was launched by the failed deployment.
 await sleep(8000)
 assert.equal(runtimeProcesses().length,0,'a node started after the deployment failed')
 assert.equal((await api('jobs')).items.length,1,'failed deployment was retried automatically')
 const page=await context.newPage();await page.goto(f.url+'/chains')
 await page.locator('[data-job-id="'+failed.id+'"]').filter({hasText:'failed'}).waitFor({timeout:20000})
 await page.locator('[data-job-id="'+failed.id+'"]').getByText('수행된 변경').click()
 await page.screenshot({path:out+'/ssh-deploy-fault.png',fullPage:true})
 // An explicit new deployment completes over the same owned SSH target.
 const again=await api('plans','POST',input,201)
 const done=await settle((await api('jobs','POST',{planId:again.id},202,again.id)).id)
 assert.equal(done.state,'succeeded',JSON.stringify(done))
 const record=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
 assert.ok(record.nodes.length===4&&record.nodes.every(n=>n.pid>0),'explicit redeploy did not launch remote nodes')
 let chainId
 for(let i=0;i<100&&!chainId;i++){try{const r=await fetch('http://127.0.0.1:10800/',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({jsonrpc:'2.0',id:1,method:'eth_chainId',params:[]}),signal:AbortSignal.timeout(1000)});chainId=(await r.json()).result}catch{};if(!chainId)await sleep(100)}
 assert.ok(chainId,'redeployed SSH node never served RPC')
 // Leave the network stopped so the runner can inspect the native databases.
 for(const n of record.nodes){const stop=await api('plans','POST',{...input,operation:'node.stop',nodeIds:[n.label],arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'owned-ssh'}},201);const s=await settle((await api('jobs','POST',{planId:stop.id},202,stop.id)).id);assert.equal(s.state,'succeeded',JSON.stringify(s))}
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),fault:{id:failed.id,state:failed.state,partialEffects:failed.partialEffects.length,unresolved:failed.unresolvedResources.length,completedUploadRetained:true,noNodeStarted:true},sessionsBeforeFault:sessions,noAutomaticRetry:true,explicitRedeploy:{id:done.id,state:done.state,chainId},seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
