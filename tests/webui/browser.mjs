import { chromium } from 'playwright'
import { readFile, writeFile } from 'node:fs/promises'
import assert from 'node:assert/strict'
const [url, output,privateFile] = process.argv.slice(2)
const privateFixture=JSON.parse(await readFile(privateFile,'utf8'))
let cookie='',csrf=''
const records=[],scenarios=[],coverage={presets:[],fields:[],options:[]}
async function api(endpoint, body) {
 const response=await fetch(url+'/api/v1/'+endpoint,{method:body?'POST':'GET',headers:{'Content-Type':'application/json',Cookie:cookie,'X-CSRF-Token':csrf},body:body?JSON.stringify(body):undefined})
 const data=await response.json(); records.push({endpoint,status:response.status,request:body,response:data})
 assert.equal(response.status,200,JSON.stringify(data));return data
}
async function validate(content,valid=true){const result=await api('documents/validate',{kind:'chain-preset',name:content.id,contractVersion:'2',content});assert.equal(result.valid,valid,JSON.stringify(result));return result}
function observed(id,messages){scenarios.push({id,observedAt:new Date().toISOString(),assertions:messages.map(message=>({message,passed:true}))})}
const browser = await chromium.launch({ headless: true, channel: process.env.WEBUI_BROWSER_CHANNEL ?? 'chrome' })
try {
 const page = await browser.newPage({ viewport: { width: 1280, height: 900 } }), failures=[]
 page.on('pageerror', error => failures.push(error.message))
 await page.goto(url)
 await page.getByLabel('Account username',{exact:true}).fill(privateFixture.username)
 await page.getByLabel('Account password',{exact:true}).fill(privateFixture.password)
 await page.getByLabel('Setup token',{exact:true}).fill(privateFixture.setupToken)
 await page.getByRole('button',{name:'Create administrator',exact:true}).click()
 await page.locator('nav a[href="/chains"]').click()
 await page.getByTestId('account-session').waitFor()
 cookie=(await page.context().cookies()).map(c=>c.name+'='+c.value).join('; ')
 csrf=(await page.evaluate(async()=>{const r=await fetch('/api/v1/auth/me');return r.json()})).csrfToken
 const presets=await api('chain-presets');assert.ok(presets.length)
 await page.locator('#chain-preset option[value="stablenet-bp4"]').waitFor({state:'attached'})
 const offered=await page.locator('#chain-preset option').evaluateAll(options=>options.map(o=>o.value).filter(Boolean))
 assert.deepEqual(offered,presets.map(p=>p.id))
 for(const preset of presets){assert.ok(['stablenet','wbft','wemix'].includes(preset.chain));await validate(preset.content);await page.selectOption('#chain-preset',preset.id);await page.getByText('All configuration fields',{exact:true}).waitFor();coverage.presets.push({id:preset.id,chain:preset.chain,validated:true,rendered:true})}
 for (const chain of ['stablenet', 'wbft', 'wemix']) {
  const schema=await api('contracts/chain-preset?chain='+chain),props=schema.$defs.envSpec.properties
  await writeFile(`${output}/${chain}-contract.json`,JSON.stringify(schema,null,2))
  await page.selectOption('#chain-preset', `${chain}-bp4`)
  await page.getByText('All configuration fields',{exact:true}).waitFor()
  await page.locator('#count-en').fill('1');await page.locator('#count-pn').fill('1')
  if (!(await page.locator('#sync-mode').isVisible())) await page.getByText('Optional settings', { exact: true }).click()
  await page.selectOption('#sync-mode', 'snap')
  await page.getByRole('button', { name: 'Validate configuration',exact:true }).click()
  await page.getByRole('status').filter({ hasText: 'Engine validation passed' }).waitFor()
  await page.getByText('All configuration fields',{exact:true}).click()
  const fields=await page.getByLabel('Add /content field',{exact:true}).locator('option').evaluateAll(options=>options.map(o=>o.value).filter(Boolean))
  const preset=presets.find(p=>p.id===`${chain}-bp4`).content
  assert.deepEqual([...new Set([...Object.keys(preset),...fields])].sort(),Object.keys(props).sort())
  coverage.fields.push({chain,expected:Object.keys(props).sort(),rendered:[...new Set([...Object.keys(preset),...fields])].sort()})
  await page.getByLabel('Add /content field',{exact:true}).selectOption('launch')
  await page.getByLabel('Add /content/launch field',{exact:true}).selectOption('all')
  const allOptions=await page.getByLabel('Add /content/launch/all field',{exact:true}).locator('option').evaluateAll(options=>options.map(o=>o.value).filter(Boolean))
  const all=props.launch.properties.all.properties,single=props.launch.additionalProperties.properties
  assert.deepEqual(allOptions.sort(),Object.keys(all).sort())
  assert.ok(!allOptions.includes('http.port'));assert.ok(!allOptions.includes('docroot'))
  await page.getByLabel('/content/launch entry name',{exact:true}).fill('node1')
  await page.locator('[data-field-path="/content/launch"]').getByRole('button',{name:'Add entry',exact:true}).click()
  const optionSelect=page.getByLabel('Add /content/launch/node1 field',{exact:true})
  assert.deepEqual((await optionSelect.locator('option').evaluateAll(options=>options.map(o=>o.value).filter(Boolean))).sort(),Object.keys(single).sort())
  for(const [key,definition] of Object.entries(single)){
   await optionSelect.selectOption(key)
   const input=page.getByLabel('/content/launch/node1/'+key,{exact:true})
   const value=definition.type==='boolean'?true:key==='syncmode'||key==='gcmode'?'full':key==='datadir'?'preset-validation':key==='miner.etherbase'?'0x0000000000000000000000000000000000000001':key==='unlock'?'0x0000000000000000000000000000000000000001':'1'
   if(definition.type==='boolean')await input.check();else await input.fill(value)
   if(key==='metrics.port'){
    await optionSelect.selectOption('metrics')
    await page.getByLabel('/content/launch/node1/metrics',{exact:true}).check()
   }
   const content=JSON.parse(await page.locator('.editor pre').textContent())
   assert.equal(content.launch.node1[key],value)
   await validate(content)
   if(key==='metrics.port')await page.getByLabel('Remove /content/launch/node1/metrics',{exact:true}).click()
   coverage.options.push({chain,key,flag:definition['x-flag'],boolean:definition.type==='boolean',rendered:true,engineValid:true})
   await page.getByLabel('Remove /content/launch/node1/'+key,{exact:true}).click()
  }
  await page.screenshot({path:`${output}/${chain}-form.png`,fullPage:true})
  await page.selectOption('#chain-preset',`${chain}-bp3-en1-table`)
  await page.getByText('All configuration fields',{exact:true}).waitFor()
  if(!(await page.locator('details.structured').evaluate(e=>e.open)))await page.getByText('All configuration fields',{exact:true}).click()
  await page.getByLabel('Add /content/topology/nodes/0 field',{exact:true}).selectOption('sync')
  await page.getByLabel('/content/topology/nodes/0/sync',{exact:true}).selectOption('snap')
  await page.getByRole('button',{name:'Validate configuration',exact:true}).click()
  await page.getByRole('status').filter({hasText:'Engine validation passed'}).waitFor()
  const table=JSON.parse(await page.locator('.editor pre').textContent())
  assert.equal(table.topology.nodes[0].sync,'snap')
  await page.screenshot({path:`${output}/${chain}-table.png`,fullPage:true})
  observed(chain+':preset-fields',['Preset selected in real browser; required and optional counts/sync override accepted by engine','Every dialect-mapped option rendered, edited, and its generated declaration validated through live API; node table edited and validated in browser'])
 }
 await page.selectOption('#chain-preset','stablenet-bp4');await page.getByText('All configuration fields',{exact:true}).waitFor()
 assert.match(await page.locator('#count-bp').locator('..').innerText(),/Inherited from preset.*4 applied/s)
 assert.match(await page.locator('#count-en').locator('..').innerText(),/Engine default.*0 applied/s)
 await page.locator('#count-bp').fill('5');assert.match(await page.locator('#count-bp').locator('..').innerText(),/Your override.*5 applied/s)
 await page.locator('#count-bp').fill('');assert.match(await page.locator('#count-bp').locator('..').innerText(),/Engine default.*4 applied/s)
 await page.getByRole('button',{name:'Validate configuration',exact:true}).click();await page.getByRole('status').filter({hasText:'Engine validation passed'}).waitFor()
 await page.screenshot({path:`${output}/defaults.png`,fullPage:true})
 observed('inheritance-defaults-effective',['Browser distinguishes inherited, overridden and absent engine-default values','Absent bp validated with engine default four'])
 const base={schemaVersion:'2',kind:'chain-preset',id:'negative',chain:'wbft',topology:{bp:4}}
 for(const content of [ {...base,chain:'unknown'}, {...base,extension:true}, {...base,topology:{bp:0}}, {...base,topology:{bp:1.5}}, {...base,topology:{bp:4,syncMode:'invalid'}}, {...base,launch:{all:{docroot:'x'}}}, {...base,launch:{all:{'chain.consensusmethod':'wbft'}}}, {...base,launch:{all:{http:'true'}}}, {...base,launch:{all:{maxpeers:true}}}, {...base,launch:{all:{chainId:'1'}}}, {...base,launch:{node1:{'metrics.port':'6060'}}}, {...base,launch:{all:{'http.port':'8501'}}}, {...base,command:'dumpgenesis'} ])await validate(content,false)
 await page.locator('#count-bp').fill('0');await page.getByRole('button',{name:'Validate configuration',exact:true}).click();await page.getByRole('status').filter({hasText:'Configuration needs attention'}).waitFor()
 await page.screenshot({path:`${output}/unsupported.png`,fullPage:true})
 observed('unsupported-ui-api',['Unsupported chains, raw commands, unmapped flags, wrong dialect, boolean/value confusion, invalid counts and disabled metrics rejected by live API','No unsupported choices in browser; invalid count displays engine rejection'])
 observed('preset-option-field-coverage',['Every valid catalog preset selected in browser and validated','Parser field denominator equals rendered fields; all dialect launch options edited without omissions'])
 assert.deepEqual(failures,[])
 await writeFile(`${output}/browser.json`,JSON.stringify({scenarios,coverage,records,failures,browserVersion:browser.version()},null,2))
} finally { await browser.close() }
