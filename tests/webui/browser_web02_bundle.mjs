import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(),context=await owned.browser.newContext({viewport:{width:1440,height:1300},acceptDownloads:true})
const sleep=ms=>new Promise(r=>setTimeout(r,ms))
const sshSecret='bundle-private-ssh-'+f.password.slice(0,10)
let session
async function api(path,method='GET',data,expected=200,headers={}){
 const r=await context.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(session?{'X-CSRF-Token':session.csrfToken}:{}),...headers},data:data===undefined?undefined:JSON.stringify(data)})
 const text=await r.text();assert.equal(r.status(),expected,`${method} ${path}: ${text}`)
 for(const secret of [f.password,f.setupToken,sshSecret])assert.ok(!text.includes(secret),'private data leaked from '+path)
 return text?JSON.parse(text):undefined
}
const sorted=v=>Array.isArray(v)?v.map(sorted):v&&typeof v==='object'?Object.fromEntries(Object.keys(v).sort().map(k=>[k,sorted(v[k])])):v
const same=(a,b)=>JSON.stringify(sorted(a))===JSON.stringify(sorted(b))
const observed={}
try{
 session=await api('bootstrap','POST',{username:'bundle-admin',password:f.password,setupToken:f.setupToken},201)
 const set={version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:39100,step:10},rpc:{base:11700,step:10}}},ssh:{port:22,password:sshSecret}}
 const config={version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}
 const bundle=docs=>Buffer.from(JSON.stringify({documents:docs}))
 const page=await context.newPage();await page.goto(f.url+'/chains')
 const input=page.getByLabel('Configuration import file',{exact:true});await input.waitFor({timeout:20000})

 // An unregistered extension is refused with the document position and field.
 await input.setInputFiles({name:'broken.bundle.json',mimeType:'application/json',buffer:bundle([{kind:'server-set',name:'bundle pool',content:set},{kind:'workspace-config',name:'bundle paths',content:{...config,extension:{plugin:'unregistered'}}}])})
 const error=page.locator('[data-import-error="/documents/1/content"]');await error.waitFor({timeout:15000})
 const errorText=await error.innerText();assert.ok(/extension/.test(errorText),'unknown extension error does not name the field: '+errorText)
 assert.ok(await page.getByRole('button',{name:'Save imported documents',exact:true}).isDisabled(),'an invalid bundle can be saved')
 observed.invalidBundle=errorText

 // A valid bundle is previewed, private SSH values are removed, and it is saved atomically.
 await input.setInputFiles({name:'team.bundle.json',mimeType:'application/json',buffer:bundle([{kind:'server-set',name:'bundle pool',content:set},{kind:'workspace-config',name:'bundle paths',content:config}])})
 const preview=page.locator('[data-testid="import-preview"][data-valid="true"]');await preview.waitFor({timeout:15000})
 assert.equal(await preview.locator('[data-import-kind]').count(),2)
 await page.getByText('Bind your own SSH credential for: /documents/0/ssh.password').waitFor()
 await page.getByRole('button',{name:'Save imported documents',exact:true}).click()
 await page.getByTestId('import-status').filter({hasText:'Saved 2 shared document revision(s)'}).waitFor({timeout:15000})
 await page.getByLabel('Imported workspace name',{exact:true}).fill('bundle workspace')
 await page.getByRole('button',{name:'Create workspace from import',exact:true}).click()
 await page.getByTestId('import-status').filter({hasText:'Created shared workspace bundle workspace'}).waitFor({timeout:15000})
 let workspace=(await api('workspaces')).items.find(w=>w.name==='bundle workspace')
 const savedSet=(await api('documents')).items.find(d=>d.kind==='server-set'&&d.name==='bundle pool')
 assert.ok(!JSON.stringify(savedSet.content).includes('password'),'shared server-set kept the SSH password field')
 observed.imported={workspace:workspace.id,documents:workspace.documents.length}

 // Structured edit through the form: a new server-set revision with another P2P base.
 await page.reload();await page.getByLabel('Saved deployment document',{exact:true}).selectOption(savedSet.id)
 const p2p=page.getByLabel('content.pool.ports.p2p.base',{exact:true});await p2p.waitFor({timeout:15000})
 await p2p.fill('39200')
 await page.getByRole('button',{name:'Save shared document',exact:true}).click()
 await page.getByText('Saved shared document revision 2').waitFor({timeout:15000})
 const edited=await api('documents/'+savedSet.id)
 assert.equal(edited.revision,2);assert.equal(edited.content.pool.ports.p2p.base,39200)
 // A stale revision is refused with a reason rather than overwriting the edit.
 const stale=await context.request.fetch(f.url+'/api/v1/documents/'+savedSet.id,{method:'PATCH',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrfToken,'If-Match':'"1"'},data:JSON.stringify({kind:'server-set',name:'bundle pool',contractVersion:'2',assetRefs:[],content:edited.content})})
 assert.equal(stale.status(),409);const staleBody=await stale.json();assert.ok(staleBody.message&&staleBody.message!=='Conflict','revision conflict reason hidden')
 observed.staleRevision=staleBody.message
 // Re-pin the workspace to the edited revision through the form.
 await page.getByLabel('Saved deployment workspace',{exact:true}).selectOption(workspace.id)
 await page.getByLabel('Workspace server-set revision',{exact:true}).selectOption(savedSet.id+':2')
 const configRef=workspace.documents.find(r=>r.id!==savedSet.id)
 await page.getByLabel('Workspace config revision',{exact:true}).selectOption(configRef.id+':'+configRef.revision)
 await page.getByRole('button',{name:'Save shared workspace',exact:true}).click()
 for(let i=0;i<50;i++){workspace=await api('workspaces/'+workspace.id);if(workspace.documents.some(r=>r.id===savedSet.id&&r.revision===2))break;await sleep(200)}
 assert.ok(workspace.documents.some(r=>r.id===savedSet.id&&r.revision===2),'workspace not re-pinned to the edited revision')

 // Export the workspace bundle from the UI; it imports back with the same meaning.
 const download=page.waitForEvent('download')
 await page.getByRole('button',{name:'Export workspace bundle',exact:true}).click()
 const file=await download;await file.saveAs(out+'/workspace.bundle.json')
 const exported=JSON.parse(fs.readFileSync(out+'/workspace.bundle.json','utf8'))
 assert.ok(!JSON.stringify(exported).includes(sshSecret)&&!JSON.stringify(exported).includes('password'),'exported bundle carries private SSH data')
 const exportedSet=exported.documents.find(d=>d.kind==='server-set')
 assert.ok(same(exportedSet.content,edited.content),'export changed the edited declaration')
 const reimport=await api('documents/import','POST',{filename:'workspace.bundle.json',format:'json',source:fs.readFileSync(out+'/workspace.bundle.json','utf8')})
 assert.ok(reimport.validation.valid&&reimport.redactedDocuments.length===2,'exported bundle does not import back')
 for(const d of exported.documents){const back=reimport.redactedDocuments.find(r=>r.kind===d.kind);assert.ok(same(back.content,d.content),'re-import changed '+d.kind)}
 observed.roundTrip={exportedDocuments:exported.documents.length,semanticallyEqual:true}

 // A real deployment from the imported and edited workspace uses the edited ports.
 await page.getByLabel('작업 Workspace',{exact:true}).selectOption(workspace.id)
 await page.getByLabel('작업 매니페스트',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 바이너리',{exact:true}).selectOption('wbft')
 await page.getByLabel('작업 종류',{exact:true}).selectOption('chain.deploy')
 await page.getByLabel('작업 서버 이름',{exact:true}).selectOption('local')
 await page.getByRole('button',{name:'실행 계획 확인',exact:true}).click()
 await page.getByLabel('실행 계획',{exact:true}).waitFor({timeout:20000})
 const started=page.waitForResponse(r=>r.url()===f.url+'/api/v1/jobs'&&r.request().method()==='POST')
 await page.getByRole('button',{name:'검토한 계획 실행',exact:true}).click()
 const job=await (await started).json()
 let done
 for(let i=0;i<1500;i++){done=await api('jobs/'+job.id);if(['succeeded','failed','cancelled','interrupted'].includes(done.state))break;await sleep(200)}
 assert.equal(done.state,'succeeded',JSON.stringify(done))
 const record=JSON.parse(fs.readFileSync(f.store+'/networks/'+workspace.id+'/chain-record.json','utf8'))
 assert.ok(record.nodes.length===4&&record.nodes.every(n=>n.pid>0),'imported workspace did not launch native nodes')
 assert.deepEqual(record.nodes.map(n=>n.p2p).sort((a,b)=>a-b),[39200,39210,39220,39230],'edited P2P ports were not applied')
 let chainId
 for(let i=0;i<100&&!chainId;i++){try{const r=await context.request.post('http://127.0.0.1:11700/',{data:{jsonrpc:'2.0',id:1,method:'eth_chainId',params:[]}});chainId=(await r.json()).result}catch{};if(!chainId)await sleep(100)}
 assert.ok(chainId,'deployed node did not serve RPC')
 await page.screenshot({path:out+'/web02-bundle.png',fullPage:true})
 observed.setup={job:done.id,p2p:record.nodes.map(n=>n.p2p),chainId}
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observed,seedAcceptanceAwarded:false},null,2))
}finally{await owned.stop()}
