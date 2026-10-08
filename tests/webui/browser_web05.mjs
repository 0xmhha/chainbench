import {chromium} from 'playwright'
import fs from 'node:fs'
import path from 'node:path'
import assert from 'node:assert/strict'
import {initial,variants} from '../../web/src/dsl-form.js'
const [url,out,fixtureFile]=process.argv.slice(2),fixture=JSON.parse(fs.readFileSync(fixtureFile,'utf8'))
const authorization='Basic '+Buffer.from(fixture.account.username+':'+fixture.account.password).toString('base64')
const records=[],scenarios=[],coverage=[]
async function api(endpoint,body,expected=200){const response=await fetch(url+'/api/v1/'+endpoint,{method:body?'POST':'GET',headers:{Authorization:authorization,'Content-Type':'application/json'},body:body?JSON.stringify(body):undefined});const text=await response.text();records.push({endpoint,status:response.status,response:text});assert.equal(response.status,expected,text);return JSON.parse(text)}
const vocabulary=await api('vocabulary'),contract=await api('contracts/dsl')
fs.writeFileSync(path.join(out,'contract.json'),JSON.stringify({vocabulary,contract},null,2))
const browser=await chromium.launch({headless:true,channel:process.env.WEBUI_BROWSER_CHANNEL??'chrome'})
const observed=(id,messages)=>scenarios.push({id,mode:'personal',transport:'local',ownership:'owned',role:'operator',observedAt:new Date().toISOString(),assertions:messages.map(message=>({message,passed:true}))})
const base={schemaVersion:'2',kind:'case',id:'browser',chainPreset:{chain:'stablenet',binaries:{default:'gstable'},topology:{bp:4}},steps:[{do:'read',source:'blockNumber',on:'node1',save:'head'},{expect:'blockNumber',onEach:['node1','node2'],compare:'GreaterOrEqual',is:'$head'},{expect:'chainId',is:'8283'}]}
try {
 const page=await browser.newPage({viewport:{width:1280,height:1000}}),errors=[]
 page.on('pageerror',e=>{errors.push(e.message);console.error(e.stack)})
 await page.goto(url+'/tests')
 await page.getByLabel('DSL username',{exact:true}).fill(fixture.account.username)
 await page.getByLabel('DSL password',{exact:true}).fill(fixture.account.password)
 await page.getByRole('button',{name:'Open DSL editor',exact:true}).click()
 const status=page.getByLabel('DSL status',{exact:true})
 await status.filter({hasText:'DSL contract loaded'}).waitFor()
 async function upload(doc){const pending=page.waitForResponse(r=>r.url().endsWith('/test-cases/import'));await page.getByLabel('Import test JSON',{exact:true}).setInputFiles({name:'case.json',mimeType:'application/json',buffer:Buffer.from(JSON.stringify(doc))});const response=await pending;await status.filter({hasText:'Working…'}).waitFor({state:'hidden'});return response}
 async function exportDoc(name){const pending=page.waitForEvent('download');await page.getByRole('button',{name:'Export test scenario',exact:true}).click();const download=await pending;await download.saveAs(path.join(out,name));return JSON.parse(fs.readFileSync(path.join(out,name),'utf8'))}
 assert.equal((await upload(base)).status(),200)
 const before=(await api('test-cases/import',{content:base})).semanticFingerprint
 await page.getByLabel('/content/id',{exact:true}).fill('browser-edited')
 await page.getByLabel('/content/steps/0/on',{exact:true}).fill('bp1')
 assert.equal(await page.getByRole('button',{name:'Export test scenario',exact:true}).isDisabled(),true)
 await page.getByRole('button',{name:'Validate test scenario',exact:true}).click()
 await status.filter({hasText:'Engine validation passed'}).waitFor()
 const exported=await exportDoc('edited.json')
 assert.equal(exported.id,'browser-edited');assert.equal(exported.steps[0].on,'bp1')
 assert.deepEqual(exported.steps.slice(1),base.steps.slice(1))
 const first=await api('test-cases/import',{content:exported}),second=await api('test-cases/import',{content:first.content})
 assert.equal(first.semanticFingerprint,second.semanticFingerprint);assert.notEqual(first.semanticFingerprint,before)
 await page.screenshot({path:path.join(out,'editor.png'),fullPage:true})
 observed('references',['Node selector and preceding variable reference edited in browser','Export re-import preserves executable fingerprint'])
 const legacy={schemaVersion:'1',id:'legacy-browser',chain:{name:'stablenet',binary:'gstable'},topology:{bp:4},steps:[{read:{source:'blockNumber',on:'bp1',save:'head'}}],assertions:[{assert:'blockNumber',compare:'GreaterOrEqual',expected:'$head'}]}
 assert.equal((await upload(legacy)).status(),200)
 await status.filter({hasText:'v1 migrated; executable meaning preserved'}).waitFor()
 const migrated=await exportDoc('migrated.json'),migration=await api('test-cases/import',{content:legacy})
 assert.equal((await api('test-cases/import',{content:migrated})).semanticFingerprint,migration.semanticFingerprint)
 fs.writeFileSync(path.join(out,'legacy.json'),JSON.stringify(legacy,null,2))
 const retained={...legacy,id:'legacy-config-browser',chain:{...legacy.chain,config:'site.toml'}}
 assert.equal((await upload(retained)).status(),200)
 await status.filter({hasText:'Retained v1:'}).waitFor()
 await page.getByLabel('/content/chain/config',{exact:true}).fill('edited-site.toml')
 await page.getByRole('button',{name:'Validate test scenario',exact:true}).click()
 await status.filter({hasText:'Engine validation passed'}).waitFor()
 const retainedExport=await exportDoc('retained-v1.json')
 assert.equal(retainedExport.schemaVersion,'1');assert.equal(retainedExport.chain.config,'edited-site.toml')
 const retainedPrepared=await api('test-cases/import',{content:retainedExport})
 assert.equal(retainedPrepared.migrated,false);assert.ok(retainedPrepared.migrationMessage)
 assert.equal(retainedPrepared.semanticFingerprint,(await api('test-cases/import',{content:retainedPrepared.content})).semanticFingerprint)
 await page.screenshot({path:path.join(out,'legacy-editor.png'),fullPage:true})
 observed('v1-migration',['Browser migrated v1 through engine migration','Original retained; v2 executable fingerprint equals v1 projection','Unrepresentable v1 config declaration structurally edited and round-tripped without losing its fields'])
 for(const invalid of [
  {...base,extension:true},
  {...base,steps:[{expect:'unknown',is:1}]},
  {...base,steps:[{expect:'blockNumber',is:'$unbound'}]},
  {...base,steps:[{do:'read',source:'unknown'},{expect:'blockNumber',is:1}]},
  {...legacy,extension:true}
 ]){const result=await api('test-cases/import',{content:invalid},422);assert.ok(result.valid===false||result.code==='Unprocessable Entity');assert.ok(result.errors?.length||result.message)}
 const invalid={...base,extension:true};assert.equal((await upload(invalid)).status(),422)
 assert.equal(await page.getByRole('button',{name:'Export test scenario',exact:true}).isDisabled(),true)
 await page.getByText('Unsupported field: extension.',{exact:false}).waitFor()
 observed('invalid-unknown',['Unknown fields/actions/readers and unbound variables rejected','Invalid browser import kept original field and blocked executable export'])
 // Enumerate the immutable registration denominator. Even statements requiring
 // a specialized network must render all contract fields; execution is measured
 // separately from the live session, never inferred from editor coverage.
 for(const entry of vocabulary.entries){
  const schemas=variants({$ref:entry.schemaRef},contract)
  const schema=schemas[0],statement={}
  for(const [name,field] of Object.entries(schema.properties??{}))statement[name]=initial(field,contract)
  if(entry.kind==='reader'){statement.do='read';statement.source=entry.name}
  if(entry.kind==='action'){statement.do=entry.name;delete statement.expectPerChain;delete statement.isPerChain}
  if(entry.kind==='assertion'){statement.expect=entry.name;delete statement.isPerChain}
  if(statement.on!==undefined)statement.on='bp1'
  if(statement.onEach!==undefined)statement.onEach=['bp1']
  if(statement.save!==undefined)statement.save='sample'
  if(statement.timeout!==undefined)statement.timeout='30s'
  if(statement.pollInterval!==undefined)statement.pollInterval='500ms'
  const doc={...base,id:`coverage-${entry.kind}-${entry.name}`,steps:[statement,{expect:'blockNumber',is:1}]}
  await upload(doc)
  const fields=Object.keys(schema.properties??{}).filter(k=>!['do','expect','source','isPerChain','expectPerChain'].includes(k))
  const edited=[]
  for(const field of fields){
   const locator=page.locator(`[data-field-path="/content/steps/0/${field}"]`).first()
   assert.equal(await locator.count(),1,`${entry.kind} ${entry.name} missing field ${field}`)
   const control=locator.locator('input,select,button').first()
   if(await control.count()){
    const tag=await control.evaluate(e=>[e.tagName,e.type])
    if(tag[0]==='INPUT'){
     if(tag[1]==='checkbox'){await control.check();await control.uncheck()}
     else {const val=await control.inputValue();await control.fill(tag[1]==='number'?'1':(val||'sample'));if((await control.getAttribute('aria-label'))?.endsWith(' entry name'))await locator.getByRole('button',{name:'Add entry',exact:true}).last().click();}
    } else if(tag[0]==='SELECT'){const val=await control.inputValue();await control.selectOption(val)}
    else if(tag[0]==='BUTTON'){await control.click()}
   }
   edited.push(field)
  }
  coverage.push({kind:entry.kind,name:entry.name,schema:true,edited:true,roundTrip:false,executed:false,argumentPaths:fields.sort(),editedArgumentPaths:edited.sort(),executedArgumentPaths:[]})
 }
 assert.deepEqual(errors,[])
 fs.writeFileSync(path.join(out,'coverage.json'),JSON.stringify(coverage,null,2))
 fs.writeFileSync(path.join(out,'browser.json'),JSON.stringify({browserVersion:browser.version(),scenarios,coverage},null,2))
 fs.writeFileSync(path.join(out,'api-observations.json'),JSON.stringify(records,null,2))
} finally {await browser.close()}
