// Field-level errors, keyboard saving and status icons on a real dashboard:
// a deployment field the contract rejects is marked invalid and reached from
// the error summary, the corrected document is saved with the keyboard alone,
// a chain preset error from the engine opens and focuses its field, and states
// are shown with an icon beside their words. Development proof only.
import {launchOwnedBrowser} from './owned-browser.mjs'
import fs from 'node:fs'
import assert from 'node:assert/strict'
const [fixturePath,out]=process.argv.slice(2),f=JSON.parse(fs.readFileSync(fixturePath,'utf8'))
const owned=await launchOwnedBrowser()
async function request(ctx,token,path,method='GET',data,expected=200){
 const response=await ctx.request.fetch(f.url+'/api/v1/'+path,{method,headers:{'Content-Type':'application/json',...(token?{'X-CSRF-Token':token}:{})},data:data===undefined?undefined:JSON.stringify(data)})
 const raw=await response.text();assert.equal(response.status(),expected,`${method} ${path}: ${raw}`)
 return raw?JSON.parse(raw):undefined
}
async function tabTo(page,name){for(let i=0;i<400;i++){await page.keyboard.press('Tab');if(await page.evaluate(n=>document.activeElement?.textContent?.trim()===n,name))return}throw new Error('keyboard never reached '+name)}
const focused=page=>page.evaluate(()=>({label:document.activeElement?.getAttribute('aria-label'),invalid:document.activeElement?.getAttribute('aria-invalid')}))
const observed={}
let page
try{
 const adminCtx=await owned.browser.newContext()
 const adminSession=await request(adminCtx,undefined,'bootstrap','POST',{username:'forms-admin',password:f.password,setupToken:f.setupToken},201)
 await request(adminCtx,adminSession.csrfToken,'users','POST',{username:'forms-operator',password:f.password,role:'operator'},201)
 const ctx=await owned.browser.newContext({viewport:{width:1440,height:1100}})
 await request(ctx,undefined,'auth/login','POST',{username:'forms-operator',password:f.password})
 page=await ctx.newPage()
 await page.goto(f.url+'/chains')

 // 1. The contract rejects an empty host address and a port above 65535.
 const addr=page.getByLabel('content.pool.hosts[0].addr',{exact:true})
 await addr.waitFor({timeout:20000})
 await addr.fill('')
 const p2p=page.getByLabel('content.pool.ports.p2p.base',{exact:true})
 await p2p.fill('70000')
 await page.getByRole('button',{name:'배포 문서 검증',exact:true}).click()
 const problems=page.getByTestId('deployment-problems')
 await problems.waitFor({timeout:10000})
 assert.equal(await addr.getAttribute('aria-invalid'),'true')
 assert.equal(await p2p.getAttribute('aria-invalid'),'true')
 assert.ok(await addr.getAttribute('aria-describedby'),'the invalid field names no description')
 await problems.getByRole('button',{name:'content.pool.ports.p2p.base'}).click()
 assert.deepEqual(await focused(page),{label:'content.pool.ports.p2p.base',invalid:'true'})
 observed.deploymentFieldErrors=await problems.locator('li').allTextContents()
 await page.screenshot({path:out+'/deployment-errors.png',fullPage:false})

 // 2. Keyboard only: correct both fields and save the shared document.
 await page.keyboard.press('ControlOrMeta+a');await page.keyboard.type('31000')
 await addr.focus();await page.keyboard.type('127.0.0.1')
 await tabTo(page,'공유 문서 저장');await page.keyboard.press('Enter')
 await page.getByTestId('deployment-status').filter({hasText:'저장됨'}).waitFor({timeout:15000})
 assert.equal(await addr.getAttribute('aria-invalid'),null)
 const sets=(await request(ctx,undefined,'documents?kind=server-set')).items
 assert.equal(sets.length,1);assert.equal(sets[0].content.pool.hosts[0].addr,'127.0.0.1')
 observed.keyboardSave=sets[0].id

 // 3. An engine error on a chain preset opens the full form at its field.
 const preset=page.locator('#chain-preset')
 await preset.locator('option').nth(1).waitFor({state:'attached',timeout:15000})
 await preset.selectOption({index:1})
 const bp=page.locator('#count-bp');await bp.waitFor({timeout:15000})
 await bp.fill('0')
 await page.getByRole('button',{name:'체인 구성 검증',exact:true}).click()
 const issue=page.locator('.editor [role=status] button.link').first()
 await issue.waitFor({timeout:15000})
 observed.presetError=await page.locator('.editor [role=status]').textContent()
 assert.match(await issue.textContent(),/^\/content\/topology/)
 await issue.click()
 const presetFocus=await focused(page)
 assert.match(presetFocus.label??'',/^\/content\/topology/,JSON.stringify(presetFocus))
 assert.equal(presetFocus.invalid,'true')
 observed.presetFocus=presetFocus

 // 4. A state is shown with an icon next to its words: the surface verdict.
 const verdict=page.getByTestId('surface-verification').locator('.status')
 await verdict.waitFor({timeout:30000})
 observed.statusBadge=(await verdict.textContent()).trim()
 assert.match(observed.statusBadge,/^[✓✕●◐!○?] \S+/)
 fs.writeFileSync(out+'/browser.json',JSON.stringify({browserVersion:owned.browser.version(),...observed,seedAcceptanceAwarded:false},null,2))
}catch(error){
 if(page){await page.screenshot({path:out+'/failure.png',fullPage:true}).catch(()=>{});fs.writeFileSync(out+'/failure.txt',(await page.locator('[role=alert],[role=status]').allTextContents().catch(()=>[])).join('\n'))}
 throw error
}finally{await owned.stop()}
