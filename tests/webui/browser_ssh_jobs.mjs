import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2), f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const key=fs.readFileSync(f.runtime+'/ssh/client','utf8')
const owned=await launchOwnedBrowser(), browser=owned.browser
const context=await browser.newContext({viewport:{width:1440,height:1100}})
let session
async function api(path,method='GET',data,expected=200,idempotency,revision){
  const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...(idempotency?{'Idempotency-Key':idempotency}:{}),...(revision?{'If-Match':String(revision)}:{})},data:data===undefined?undefined:JSON.stringify(data)})
  const text=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${text}`)
  assert.ok(!text.includes(key.trim())&&!text.includes(f.password),'credential material leaked')
  return text?JSON.parse(text):undefined
}
const phases=[]
async function waitJob(id){
  for(let i=0;i<1200;i++){
    const job=await api('jobs/'+id)
    if(['succeeded','failed','cancelled','interrupted'].includes(job.state)){
      assert.equal(job.state,'succeeded',JSON.stringify(job));phases.push(job);return job
    }
    await new Promise(r=>setTimeout(r,200))
  }
  throw new Error('SSH job timed out')
}
try{
  session=await api('bootstrap','POST',{username:'ssh-admin',password:f.password,setupToken:f.setupToken},201)
  const set=await api('documents','POST',{kind:'server-set',name:'SSH fixture',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'owned-ssh',addr:'localhost.'}],slots:4,ports:{p2p:{base:35500,step:10},rpc:{base:10800,step:10}}},ssh:{port:f.ssh.port,known_hosts_file:f.knownHosts}}},201)
  const config=await api('documents','POST',{kind:'workspace-config',name:'SSH paths',contractVersion:'2',assetRefs:[],content:{version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/artifacts'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}},201)
  const w=await api('workspaces','POST',{name:'SSH fixture',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},201)
  const credential=await api('credentials','POST',{label:'Owned SSH fixture',kind:'private-key',sshUser:f.ssh.user,privateKey:key},201)
  await api(`workspaces/${w.id}/credential-bindings`,'PUT',{serverRef:'owned-ssh',credentialId:credential.id})
  // A direct local spelling must resolve to the same physical host and root
  // even though its declaration, workspace and transport differ.
  const localSet=await api('documents','POST',{kind:'server-set',name:'Local alias',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:35500,step:10},rpc:{base:10800,step:10}}}}},201)
  const local=await api('workspaces','POST',{name:'Local alias',documents:[{id:localSet.id,revision:localSet.revision},{id:config.id,revision:config.revision}]},201)
  const aliasPlan=await api('plans','POST',{workspaceId:local.id,operation:'chain.setup',documentRefs:local.documents,assetRefs:['wbft'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local',validators:4}},201)
  const page=await context.newPage();await page.goto(f.url+'/chains')
  await page.getByLabel('작업 Workspace',{exact:true}).selectOption(w.id)
  await page.getByLabel('작업 매니페스트',{exact:true}).selectOption('wbft')
  await page.getByLabel('작업 바이너리',{exact:true}).selectOption('wbft')
  await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('owned-ssh')
  const planned=page.waitForResponse(r=>r.url()===f.url+'/api/v1/plans'&&r.request().method()==='POST')
  await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
  const plannedResponse=await planned;assert.equal(plannedResponse.status(),201,await plannedResponse.text())
  await page.getByLabel('실행 계획',{exact:true}).waitFor()
  await page.getByLabel('서버 실행 작업',{exact:true}).screenshot({path:out+'/ssh-plan.png'})
  const accept=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
  await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click()
  const response=await accept;assert.equal(response.status(),202)
  const job=await response.json()
  await api(`workspaces/${w.id}`,'PATCH',{name:'Edited after acceptance',documents:w.documents},200,undefined,w.revision)
  const replacement=await api('credentials','POST',{label:'Unauthorised future binding',kind:'private-key',sshUser:f.ssh.user,privateKey:fs.readFileSync(f.runtime+'/ssh/unauthorized','utf8')},201)
  await api(`workspaces/${w.id}/credential-bindings`,'PUT',{serverRef:'owned-ssh',credentialId:replacement.id})
  const jobStore=JSON.parse(fs.readFileSync(f.store+'/jobs.json','utf8'))
  const original=jobStore.plans[job.planId].prepared.claims[0],alias=jobStore.plans[aliasPlan.id].prepared.claims[0]
  assert.equal(original.hostIdentity,alias.hostIdentity);assert.equal(original.dataPath,alias.dataPath)
  await api('jobs','POST',{planId:aliasPlan.id},409,aliasPlan.id)
  await waitJob(job.id)
  await api(`workspaces/${w.id}/credential-bindings`,'PUT',{serverRef:'owned-ssh',credentialId:credential.id})
  const network=(await api('networks')).items.find(n=>n.workspaceId===w.id)
  assert.equal(network.nodes.length,4)
  assert.ok(network.nodes.every(n=>n.hostIdentity.startsWith('machine-sha256:')&&n.supportedControls.includes('node.start')))
  const input={workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],credentialBindings:{'owned-ssh':credential.id},nodeIds:['node1'],retention:'retain',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'owned-ssh',validators:4}}
  for(const operation of ['node.start','node.stop']){
    const plan=await api('plans','POST',{...input,operation},201)
    const accepted=await api('jobs','POST',{planId:plan.id},202,plan.id)
    await waitJob(accepted.id)
    if(operation==='node.start'){
      let rpcReady=false
      for(let i=0;i<100;i++){
        try{const r=await fetch('http://127.0.0.1:10800/',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({jsonrpc:'2.0',id:1,method:'eth_chainId',params:[]}),signal:AbortSignal.timeout(1000)});const result=await r.json();if(result.result==='0x205c'){rpcReady=true;break}}catch{}
        await new Promise(r=>setTimeout(r,100))
      }
      assert.ok(rpcReady,'SSH-launched native node never served its actual chain RPC')
      const observed=await api('networks/'+w.id+'/observations')
      const live=observed.nodes.find(n=>n.id==='node1');assert.equal(live.state,'running');assert.ok(live.observedPid>0)
    }
  }
  const revokePlan=await api('plans','POST',{...input,operation:'node.start'},201)
  await api('credentials/'+credential.id,'DELETE',undefined,204)
  await api('jobs','POST',{planId:revokePlan.id},403,revokePlan.id)
  assert.equal((await api('networks')).items.find(n=>n.workspaceId===w.id).nodes[0].pid,0)
  await page.reload();await page.getByLabel('서버 실행 작업',{exact:true}).waitFor()
  await page.screenshot({path:out+'/ssh-jobs.png',fullPage:true})
  fs.writeFileSync(out+'/browser.json',JSON.stringify({browser:browser.version(),jobs:phases,physicalLocalSSHAlias:'same identity/root; concurrent alias start refused',acceptedInputEdits:'workspace and binding edits did not redirect accepted job',revokedPlan:'refused',nativeNodes:network.nodes.length},null,2))
}finally{await owned.stop()}
