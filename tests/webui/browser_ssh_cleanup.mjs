import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const key=fs.readFileSync(f.runtime+'/ssh/client','utf8')
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext()
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
let session
async function api(path,method='GET',data,expected=200,idempotency){
 const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(idempotency?{'Idempotency-Key':idempotency}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const text=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${text}`)
 assert.ok(!text.includes(key.trim())&&!text.includes(f.password),'credential material leaked')
 return text?JSON.parse(text):undefined
}
async function settle(id){for(let i=0;i<1500;i++){const job=await api('jobs/'+id);if(['succeeded','failed','cancelled','interrupted'].includes(job.state))return job;await sleep(200)}throw new Error('SSH job timed out')}
const live=pid=>{try{return execFileSync('ps',['-p',String(pid),'-o','command='],{encoding:'utf8'}).includes(f.runtime)}catch{return false}}
try{
 session=await api('bootstrap','POST',{username:'ssh-cleanup-admin',password:f.password,setupToken:f.setupToken},201)
 const set=await api('documents','POST',{kind:'server-set',name:'SSH cleanup fixture',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'owned-ssh',addr:'localhost.'}],slots:4,ports:{p2p:{base:35500,step:10},rpc:{base:10800,step:10}}},ssh:{port:f.ssh.port,known_hosts_file:f.knownHosts}}},201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'SSH cleanup paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/artifacts'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},201)
 const w=await api('workspaces','POST',{name:'SSH cleanup fixture',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},201)
 const credential=await api('credentials','POST',{label:'Owned SSH cleanup fixture',kind:'private-key',sshUser:f.ssh.user,privateKey:key},201)
 await api(`workspaces/${w.id}/credential-bindings`,'PUT',{serverRef:'owned-ssh',credentialId:credential.id})
 const c=await api('documents','POST',{kind:'case',name:'SSH cleanup case',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'ssh-cleanup',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:'$head'}]}},201)
 // Normal end of an SSH test run with selected cleanup.
 const plan=await api('plans','POST',{workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],credentialBindings:{'owned-ssh':credential.id},retention:'cleanup',operation:'test.run',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'owned-ssh',caseRefs:[{id:c.id,revision:c.revision}]}},201)
 const job=await settle((await api('jobs','POST',{planId:plan.id},202,plan.id)).id)
 assert.equal(job.state,'succeeded',JSON.stringify(job));assert.equal(job.runIds.length,1,'remote engine session missing')
 assert.equal(job.nodeDisposition,'cleaned',JSON.stringify(job))
 const record=JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
 assert.ok(record.statePath==='CHAIN/CHAIN_REMOVED'&&(record.nodes??[]).length===0,'SSH cleanup did not remove the composition')
 const remaining=fs.existsSync(f.runtime+'/d/nodes')?fs.readdirSync(f.runtime+'/d/nodes').flatMap(d=>fs.readdirSync(f.runtime+'/d/nodes/'+d)):[]
 assert.equal(remaining.length,0,'remote node data kept after cleanup')
 const started=job.partialEffects.find(e=>/start: 4 node/.test(e));assert.ok(started,'remote nodes never started')
 const sessions=(fs.readFileSync(f.runtime+'/ssh/sshd.log','utf8').match(/Accepted publickey/g)||[]).length
 assert.ok(sessions>0,'no real SSH session used')
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),job:{id:job.id,state:job.state,nodeDisposition:job.nodeDisposition,runIds:job.runIds},remoteNodesStartedAndRemoved:true,sshSessions:sessions,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
