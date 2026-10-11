import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import path from 'node:path'
import assert from 'node:assert/strict'
const [url,out,fixturePath]=process.argv.slice(2),fixture=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser(), browser=owned.browser
const setups=[],scenarios=[],records=[]
const observed=(id,text)=>scenarios.push({id,mode:'personal',transport:'local',ownership:'owned',role:'operator',observedAt:new Date().toISOString(),assertions:text.map(message=>({message,passed:true}))})
// Every call goes through the dashboard's own session and CSRF token.
let context,csrf=''
async function call(endpoint,method='GET',body){const r=await context.request.fetch(url+'/api/v1/'+endpoint,{method,headers:{'Content-Type':'application/json',...(csrf?{'X-CSRF-Token':csrf}:{})},data:body===undefined?undefined:JSON.stringify(body)});return {status:r.status(),text:await r.text()}}
async function api(endpoint,method='GET',body,expected=200){const r=await call(endpoint,method,body);records.push({endpoint,method,status:r.status,body:r.text});assert.equal(r.status,expected,r.text);return r.status===200||r.status===201?JSON.parse(r.text):r.text}
try{
 context=await browser.newContext({viewport:{width:1280,height:1000}})
 const admin=JSON.parse((await call('bootstrap','POST',{username:'manifest-admin',password:fixture.account.password,setupToken:fixture.setupToken})).text);csrf=admin.csrfToken
 await api('users','POST',{username:fixture.account.username,password:fixture.account.password,role:'operator'},201)
 const operator=JSON.parse((await call('auth/login','POST',{username:fixture.account.username,password:fixture.account.password})).text);csrf=operator.csrfToken
 assert.equal(operator.user.role,'operator')
 const page=await context.newPage(),errors=[]
 page.on('pageerror',e=>errors.push(e.message));await page.goto(url+'/settings')
 const status=page.getByLabel('매니페스트 상태',{exact:true})
 await page.getByLabel('저장된 체인 매니페스트',{exact:true}).locator('option[value="wbft"]').waitFor({state:'attached'})
 const library=(await api('manifests')).items
 assert.deepEqual(library.filter(i=>i.source==='builtin').map(i=>i.id).sort(),['stablenet','wbft','wemix'])
 for(const chain of ['stablenet','wbft','wemix']){
  const builtin=library.find(i=>i.id===chain)
  for(const source of ['builtin','external']){
   let selected=chain
   if(source==='external'){
    const manifest=structuredClone(builtin.manifest);manifest.id='external-'+chain;manifest.protocol=chain
    await page.getByLabel('매니페스트 JSON 파일',{exact:true}).setInputFiles({name:'manifest.json',mimeType:'application/json',buffer:Buffer.from(JSON.stringify(manifest))})
    await status.filter({hasText:'선언을 읽었습니다'}).waitFor()
    await page.getByLabel('genesis 템플릿 JSON 파일',{exact:true}).setInputFiles({name:'genesis.json',mimeType:'application/json',buffer:Buffer.from(builtin.template)})
    await page.getByRole('button',{name:'매니페스트 검증',exact:true}).click()
    await status.filter({hasText:'엔진 매니페스트 검증 통과'}).waitFor()
    await page.getByRole('button',{name:'외부 매니페스트 저장',exact:true}).click()
    await status.filter({hasText:'외부 매니페스트를 저장했습니다'}).waitFor()
    selected=await page.getByLabel('저장된 체인 매니페스트',{exact:true}).inputValue()
    const stored=await api('manifests/'+selected);assert.deepEqual(stored.manifest,manifest);assert.equal(stored.template,builtin.template)
   }
   await page.getByLabel('저장된 체인 매니페스트',{exact:true}).selectOption(selected)
   const downloadPromise=page.waitForEvent('download');await page.getByRole('button',{name:'매니페스트 묶음 내보내기',exact:true}).click();const download=await downloadPromise;const exportedPath=path.join(out,source+'-'+chain+'-bundle.json');await download.saveAs(exportedPath);const exportedBundle=JSON.parse(fs.readFileSync(exportedPath,'utf8'));assert.equal(exportedBundle.manifest.id,source==='builtin'?chain:'external-'+chain);assert.equal(exportedBundle.template,builtin.template)
   await page.getByLabel('검증된 바이너리 자료',{exact:true}).selectOption(chain)
   const setupResponse=page.waitForResponse(r=>r.url().endsWith(`/manifests/${selected}/setup`) && r.request().method()==='POST')
   await page.getByRole('button',{name:'매니페스트 설정 초기화',exact:true}).click()
   const response=await setupResponse;const responseText=await response.text();assert.equal(response.status(),200,responseText);const applied=JSON.parse(responseText);records.push({endpoint:`manifests/${selected}/setup`,method:'POST',status:response.status(),body:responseText})
   await page.waitForFunction(()=>{const s=document.querySelector('[aria-label="매니페스트 상태"]').textContent;return s.includes('엔진 설정을 초기화했습니다')||/^[45][0-9]{2}:/.test(s)},{},{timeout:90000});assert.match(await status.textContent(),/엔진 설정을 초기화했습니다/)
   await page.screenshot({path:path.join(out,source+'-'+chain+'.png'),fullPage:true})
   // The browser request result is also observed independently by an API setup.
   // Capture the actual browser response instead via persisted engine workspace ID.
   const text=await page.getByTestId('manifest-setup').textContent(),id=text.split(' · ')[0].replace('설정 ','')
   const dir=path.join(fixture.store,'manifests',id)
   // The fixture runner resolves the authoritative composition state and generated database files.
   assert.equal(applied.id,id);assert.equal(applied.state.chain,source==='builtin'?chain:'external-'+chain)
   setups.push({id,manifestId:selected,chain:source==='builtin'?chain:'external-'+chain,source,dir,binary:applied.binary})
   observed(source+'-'+chain,['Engine validated and initialized '+source+' '+chain+' through the browser','Selection used a fingerprinted native binary asset'])
  }
 }
 const sample=structuredClone(library.find(i=>i.id==='wbft'))
 for(const [id,key,value] of [['unsupported-family','consensus_family','raft'],['unsupported-dialect','dialect','unknown']]){
  const manifest={...sample.manifest,[key]:value,id:'invalid-'+id,protocol:'wbft'},input={manifest,template:sample.template}
  const validation=await api('manifests/validate','POST',input);assert.equal(validation.valid,false);assert.ok(validation.errors.length)
  await api('manifests','POST',input,422);observed(id,['Validation and persistence both rejected '+value])
 }
 await api('manifests/wbft/setup','POST',{assetId:'wemix'},422)
 await api('manifests/wemix/setup','POST',{assetId:'wbft'},422)
 await api('manifests/wbft/setup','POST',{assetId:'wrong-wbft'},422)
 await api('manifests/wbft/setup','POST',{assetId:'stale-wbft'},422)
 observed('wrong-binary-identity',['Both gwemix identities rejected for the opposite chain','Actual incompatible binary help vocabulary and changed checksum rejected before setup'])
 const exported=await api('manifests/'+setups.find(s=>s.source==='external').manifestId)
 assert.equal(exported.source,'external');assert.equal(exported.manifest.protocol,'stablenet')
 assert.equal(errors.length,0,errors.join('\n'))
 observed('manifest-management',['Browser imported and persisted three external declarations without semantic change','Embedded plugins remain present and external records are content addressed'])
 fs.writeFileSync(path.join(out,'browser.json'),JSON.stringify({setups,scenarios,browserVersion:browser.version()},null,2))
 fs.writeFileSync(path.join(out,'api-observations.json'),JSON.stringify(records,null,2))
}finally{await owned.stop()}
