// Statement tree, error links, duplication, v1 import differences and
// keyboard-only use on a real dashboard: an engine error selects its
// statement and focuses the invalid field, statements are reached and moved
// with the keyboard, a case is saved and duplicated with the keyboard alone,
// and a running native job is cancelled with the keyboard alone.
// Development proof only; no Seed criterion is awarded.
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
async function settle(api,id){for(let i=0;i<900;i++){const job=await api('jobs/'+id);if(terminal.includes(job.state))return job;await sleep(200)}throw new Error('job did not settle')}
// Tab until the named control has focus, then return; the mouse is never used.
async function tabTo(page,name){for(let i=0;i<300;i++){await page.keyboard.press('Tab');if(await page.evaluate(n=>document.activeElement?.textContent?.trim()===n,name))return}throw new Error('keyboard never reached '+name)}
const file=(name,value)=>({name,mimeType:'application/json',buffer:Buffer.from(JSON.stringify(value))})
const observed={}
let page
try{
 const adminCtx=await owned.browser.newContext()
 const adminSession=await request(adminCtx,undefined,'bootstrap','POST',{username:'dsl-admin',password:f.password,setupToken:f.setupToken},undefined,201)
 await request(adminCtx,adminSession.csrfToken,'users','POST',{username:'dsl-operator',password:f.password,role:'operator'},undefined,201)
 const ctx=await owned.browser.newContext({viewport:{width:1440,height:1100}})
 const s=await request(ctx,undefined,'auth/login','POST',{username:'dsl-operator',password:f.password})
 const api=(path,method,data,key,expected)=>request(ctx,s.csrfToken,path,method,data,key,expected)
 page=await ctx.newPage()
 await page.goto(f.url+'/tests')
 const upload=page.getByLabel('테스트 JSON 가져오기',{exact:true})
 await upload.waitFor({timeout:15000})

 // 1. An engine error names its statement; choosing it selects that statement
 // and focuses the field marked invalid.
 const bad={schemaVersion:'2',kind:'case',id:'dsl-proof',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4}},steps:[{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0},{do:'nope'}]}
 await upload.setInputFiles(file('bad.json',bad))
 const summary=page.getByTestId('dsl-error-summary')
 await summary.waitFor({timeout:15000})
 await summary.getByRole('button',{name:'/content/steps/1/do'}).click()
 assert.equal(await page.locator('[data-tree-path="/content/steps/1"]').getAttribute('aria-current'),'true','the error did not select its statement')
 assert.equal(await page.evaluate(()=>document.activeElement?.getAttribute('aria-invalid')),'true','focus is not on the invalid field')
 observed.errorLink={path:'/content/steps/1/do',focusedInvalid:true}

 // 2. The keyboard walks the tree and moves a statement with Alt.
 await page.locator('[data-tree-path="/content/steps/0"]').focus()
 await page.keyboard.press('ArrowDown')
 assert.equal(await page.evaluate(()=>document.activeElement?.dataset.treePath),'/content/steps/1')
 await page.keyboard.press('Alt+ArrowUp')
 await page.waitForFunction(()=>document.querySelector('[data-tree-path="/content/steps/0"]')?.textContent.includes('nope'),null,{timeout:5000})
 observed.keyboardMove=true
 // Removing the bad statement leaves a valid case.
 await page.locator('[data-tree-path="/content/steps/0"]').click()
 await page.getByRole('group',{name:'고른 statement'}).getByRole('button',{name:'삭제',exact:true}).click()

 // 3. Keyboard only: validate, save, duplicate, validate and save the copy.
 await page.locator('[data-tree-path=""]').focus()
 await tabTo(page,'테스트 시나리오 검증');await page.keyboard.press('Enter')
 await page.getByRole('status',{name:'DSL 상태'}).filter({hasText:'엔진 검증 통과'}).waitFor({timeout:15000})
 await tabTo(page,'공유 테스트 저장');await page.keyboard.press('Enter')
 await page.getByRole('status',{name:'DSL 상태'}).filter({hasText:'저장됨'}).waitFor({timeout:15000})
 await tabTo(page,'케이스 복제');await page.keyboard.press('Enter')
 await page.getByRole('status',{name:'DSL 상태'}).filter({hasText:'복제했습니다'}).waitFor({timeout:5000})
 await page.locator('[data-tree-path=""]').focus()
 await tabTo(page,'테스트 시나리오 검증');await page.keyboard.press('Enter')
 await page.getByRole('status',{name:'DSL 상태'}).filter({hasText:'엔진 검증 통과'}).waitFor({timeout:15000})
 await tabTo(page,'공유 테스트 저장');await page.keyboard.press('Enter')
 await page.getByRole('status',{name:'DSL 상태'}).filter({hasText:'저장됨'}).waitFor({timeout:15000})
 const cases=(await api('documents?kind=case&limit=100')).items
 const ids=cases.map(c=>c.content.id).sort()
 assert.deepEqual(ids,['dsl-proof','dsl-proof-copy'],JSON.stringify(ids))
 assert.notEqual(cases[0].id,cases[1].id)
 observed.keyboardSaveAndDuplicate=ids

 // 4. A v1 import lists what the migration changed.
 await upload.setInputFiles(file('legacy.json',{schemaVersion:'1',id:'legacy-browser',chain:{name:'stablenet',binary:'gstable'},topology:{bp:4},steps:[{read:{source:'blockNumber',on:'bp1',save:'head'}}],assertions:[{assert:'blockNumber',compare:'GreaterOrEqual',expected:'$head'}]}))
 const diff=page.getByTestId('import-diff')
 await diff.waitFor({timeout:15000})
 observed.importDiff=(await diff.locator('li').allTextContents()).slice(0,8)
 assert.ok(observed.importDiff.length>0)
 await page.screenshot({path:out+'/dsl-editor.png',fullPage:true})

 // The structured external profile has its own form, role selectors, and
 // conditional auth fields. No RPC request is sent during editor validation.
 const endpointCase={schemaVersion:'2',kind:'case',id:'endpoint-browser',chainPreset:{schemaVersion:'2',kind:'chain-preset',id:'external',chain:'wbft',attach:{endpoints:[{rpc:'http://example.invalid/bp',role:'bp'},{rpc:'http://example.invalid/en',role:'en',auth:{type:'jwt',env:'TEST_TOKEN'}}]}},steps:[{expect:'blockNumber',on:'en1',is:0}]}
 await upload.setInputFiles(file('endpoints.json',endpointCase))
 await page.getByRole('status',{name:'DSL 상태'}).filter({hasText:'엔진 검증 통과'}).waitFor({timeout:15000})
 await page.locator('[data-tree-path=""]').click()
 const attachPath='/content/chainPreset/attach'
 const selector=page.getByLabel(attachPath+' 형식',{exact:true})
 assert.equal(await selector.inputValue(),'1','profile was rendered as RPC list')
 const add=page.getByLabel(attachPath+' 필드 추가',{exact:true})
 assert.ok(!(await add.locator('option').allTextContents()).some(text=>text.trim()==='rpc'),'conflicting rpc field is addable')
 const authAdd=page.getByLabel(attachPath+'/endpoints/1/auth 필드 추가',{exact:true})
 if(await authAdd.count())assert.ok(!(await authAdd.locator('option').allTextContents()).some(text=>text.trim()==='header'),'jwt offers ignored header')
 await page.locator('[data-tree-path="/content/steps/0"]').click()
 const on=page.getByLabel('/content/steps/0/on',{exact:true})
 const names=await page.locator('[id="/content/steps/0/on-suggestions"] option').evaluateAll(options=>options.map(option=>option.value))
 assert.ok(names.includes('en1')&&names.includes('bp1')&&names.includes('node2'),JSON.stringify(names))
 await on.fill('en1')
 await page.getByRole('button',{name:'테스트 시나리오 검증',exact:true}).click()
 await page.getByRole('status',{name:'DSL 상태'}).filter({hasText:'엔진 검증 통과'}).waitFor({timeout:15000})
 const exportedPromise=page.waitForEvent('download')
 await page.getByRole('button',{name:'테스트 시나리오 내보내기',exact:true}).click()
 const exported=await exportedPromise
 const filePath=out+'/endpoint-case-export.json'
 await exported.saveAs(filePath)
 assert.deepEqual(JSON.parse(fs.readFileSync(filePath,'utf8')),endpointCase,'editor changed the imported endpoint declaration')
 observed.endpointProfile={roleSelectors:names,importExportPreserved:true,exclusiveForms:true,jwtOmitsHeader:true}

 // Add a typed access tuple through the shared engine schema, retaining a
 // saved address reference rather than replacing it with a literal address.
 const listCase={...endpointCase,id:'list-browser',steps:[
  {do:'newAccount',save:'created',saveKey:'key'},
  {do:'sendTx',on:'en1',key:'$key',to:'0x000000000000000000000000000000000000dead',value:'1',gas:'70000',maxFeePerGas:'3',maxPriorityFeePerGas:'1',accessList:[]},
  {expect:'blockNumber',on:'en1',is:0},
 ]}
 await upload.setInputFiles(file('list.json',listCase))
 await page.locator('[data-tree-path="/content/steps/1"]').click()
 const listPath='/content/steps/1/accessList'
 await page.getByLabel(listPath+' 항목 추가',{exact:true}).click()
 await page.getByLabel(listPath+'/0/address',{exact:true}).fill('$created')
 await page.getByLabel(listPath+'/0/storageKeys 항목 추가',{exact:true}).click()
 const slot='0x'+'0'.repeat(63)+'1'
 await page.getByLabel(listPath+'/0/storageKeys/0',{exact:true}).fill(slot)
 await page.getByRole('button',{name:'테스트 시나리오 검증',exact:true}).click()
 await page.getByRole('status',{name:'DSL 상태'}).filter({hasText:'엔진 검증 통과'}).waitFor({timeout:15000})
 await page.screenshot({path:out+'/access-list-editor.png',fullPage:true})
 await page.getByRole('button',{name:'공유 테스트 저장',exact:true}).click()
 await page.getByRole('status',{name:'DSL 상태'}).filter({hasText:'저장됨'}).waitFor({timeout:15000})
 const listDownload=page.waitForEvent('download')
 await page.getByRole('button',{name:'테스트 시나리오 내보내기',exact:true}).click()
 await (await listDownload).saveAs(out+'/access-list-export.json')
 listCase.steps[1].accessList=[{address:'$created',storageKeys:[slot]}]
 assert.deepEqual(JSON.parse(fs.readFileSync(out+'/access-list-export.json','utf8')),listCase,'access-list authoring/import/export changed a reference or key')
 observed.accessList={structuredFields:true,savedAddressReference:true,storageKey:slot,importExportPreserved:true}

 // 5. Keyboard only: cancel a running native job from its page.
 const paths={version:1,dataRoot:f.runtime+'/d',paths:Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k=>[k,k])),control:{artifactRoot:f.runtime+'/output'},inputs:{mode:'generated'},execution:{chain:'fresh'},limits:{minFreeDisk:'0'}}
 const set=await api('documents','POST',{kind:'server-set',name:'dsl pool',contractVersion:'2',assetRefs:[],content:{version:2,pool:{hosts:[{name:'local',addr:'127.0.0.1'}],slots:4,ports:{p2p:{base:40100,step:10},rpc:{base:12700,step:10}}}}},undefined,201)
 const config=await api('documents','POST',{kind:'workspace-config',name:'dsl paths',contractVersion:'2',assetRefs:[],content:paths},undefined,201)
 const w=await api('workspaces','POST',{name:'dsl chain',documents:[{id:set.id,revision:set.revision},{id:config.id,revision:config.revision}]},undefined,201)
 const long=await api('documents','POST',{kind:'case',name:'dsl long case',contractVersion:'2',assetRefs:[],content:{schemaVersion:'2',kind:'case',id:'dsl-long',chainPreset:{chain:'wbft',binaries:{default:'gwemix'},topology:{bp:4},launch:{all:{ipcdisable:true}}},steps:[{do:'waitBlock',on:'node1',target:1000000000,timeout:'5m'},{expect:'blockNumber',on:'node1',compare:'GreaterOrEqual',is:0}]}},undefined,201)
 const plan=await api('plans','POST',{workspaceId:w.id,documentRefs:w.documents,assetRefs:['wbft'],retention:'cleanup',operation:'test.run',arguments:{manifestId:'wbft',assetId:'wbft',serverRef:'local',caseRefs:[{id:long.id,revision:long.revision}]}},undefined,201)
 const jobId=(await api('jobs','POST',{planId:plan.id},plan.id,202)).id
 const record=()=>JSON.parse(fs.readFileSync(f.store+'/networks/'+w.id+'/chain-record.json','utf8'))
 for(let i=0;i<900;i++){const job=await api('jobs/'+jobId);let up=false;try{up=record().nodes.length===4&&record().nodes.every(n=>n.pid>0&&live(n.pid))}catch{}if(job.state==='running'&&job.phases.some(p=>p.name==='test.run'&&p.state==='running')&&up)break;if(terminal.includes(job.state))throw new Error('job ended early: '+JSON.stringify(job));await sleep(200)}
 await page.goto(f.url+'/jobs/'+jobId)
 await page.getByTestId('job-state').waitFor({timeout:15000})
 page.once('dialog',d=>d.accept())
 await tabTo(page,'작업 취소');await page.keyboard.press('Enter')
 const cancelled=await settle(api,jobId)
 assert.equal(cancelled.state,'cancelled');assert.equal(cancelled.nodeDisposition,'cleaned',JSON.stringify(cancelled))
 observed.keyboardCancel={jobId,disposition:cancelled.nodeDisposition}
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observed,seedAcceptanceAwarded:false},null,2))
}catch(error){
 if(page){await page.screenshot({path:out+'/failure.png',fullPage:true}).catch(()=>{});fs.writeFileSync(out+'/failure.txt',(await page.locator('[role=alert],[role=status]').allTextContents().catch(()=>[])).join('\n'))}
 throw error
}finally{await owned.stop()}
