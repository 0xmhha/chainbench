import test from 'node:test'
import assert from 'node:assert/strict'
import {variants, initial, selectedVariant, references, finishedGenesisRef, caseAssetRefs} from '../../web/src/dsl-form.js'
test('nested references and variants expose each selected reader',()=>{
 const root={$defs:{read:{oneOf:['a','b'].map(source=>({type:'object',properties:{do:{const:'read'},source:{const:source}},required:['do','source']}))}}}
 const choices=variants({$ref:'#/$defs/read'},root)
 assert.equal(choices.length,2)
 assert.equal(selectedVariant(choices,{do:'read',source:'b'}),1)
 assert.deepEqual(initial(choices[1],root),{do:'read',source:'b'})
})
test('new optional controls do not inject defaults into imported declarations',()=>{
 const schema={type:'object',required:['id'],properties:{id:{type:'string'},timeout:{type:'string',default:'30s'},kind:{const:'case'}}}
 assert.deepEqual(initial(schema,{}),{id:'',kind:'case'})
})
test('references offer only prior variable bindings and declared node indices',()=>{
 const doc={chainPreset:{topology:{bp:2,en:1}},hooks:{pre:[{do:'newAccount',save:'addr',saveKey:'key'}]},steps:[{do:'read',save:'head'},{expect:'blockNumber',is:'$head'}]}
 const suggestions=references(doc,['bp','en','pn'])
 assert.deepEqual(suggestions.before.get('/content/steps/0'),['$addr','$key'])
 assert.deepEqual(suggestions.before.get('/content/steps/1'),['$addr','$key','$head'])
 assert.ok(suggestions.nodes.includes('node3'))
 assert.ok(!suggestions.nodes.includes('node4'))
})
test('legacy readers select their own nested argument schema',()=>{
 const choices=['balanceAt','blockNumber'].map(source=>({type:'object',required:['read'],additionalProperties:false,properties:{read:{type:'object',properties:{source:{const:source}}}}}))
 assert.equal(selectedVariant(choices,{read:{source:'blockNumber'}}),1)
 const suggestions=references({schemaVersion:'1',topology:{bp:2},steps:[{read:{source:'blockNumber',save:'height'}}],assertions:[{assert:'blockNumber',expected:'$height'}]},['bp','en','pn'])
 assert.deepEqual(suggestions.before.get('/content/assertions/0'),['$height'])
})
test('finished genesis dependencies retain only exact registered references in both grammars',()=>{
 const id='0123456789abcdef'.repeat(2)
 for(const doc of [{schemaVersion:'1',chain:{genesisExisting:'asset:'+id}},{schemaVersion:'2',chainPreset:{genesis:{mode:'existing',ref:'asset:'+id}}}]){
  const before=JSON.stringify(doc)
  assert.equal(finishedGenesisRef(doc),'asset:'+id)
  assert.deepEqual(caseAssetRefs(doc),[id])
  assert.equal(JSON.stringify(doc),before,'dependency collection changed imported content')
 }
 for(const ref of ['/private/genesis.json','asset:'+id.toUpperCase(),'asset:'+id+'suffix','asset:bad'])assert.deepEqual(caseAssetRefs({schemaVersion:'2',chainPreset:{genesis:{ref}}}),[])
 assert.deepEqual(caseAssetRefs({schemaVersion:'2',chainPreset:'named-preset'}),[])
 assert.deepEqual(caseAssetRefs(null),[])
})

test('preset references are read from id and extends forms only', async () => {
  const {presetIds, sameDeclaration} = await import('../../web/src/dsl-form.js')
  assert.deepEqual(presetIds({schemaVersion: '2', chainPreset: 'stablenet-bp4'}), ['stablenet-bp4'])
  assert.deepEqual(presetIds({schemaVersion: '2', chainPreset: {extends: 'stablenet-bp4-en1', topology: {bp: 3}}}), ['stablenet-bp4-en1'])
  assert.deepEqual(presetIds({schemaVersion: '2', chainPreset: {chain: 'stablenet'}}), [])
  assert.deepEqual(presetIds({schemaVersion: '1', chain: {name: 'stablenet'}}), [])
  assert.ok(sameDeclaration({a: 1, b: {c: [1, {d: 2, e: 3}]}}, {b: {c: [1, {e: 3, d: 2}]}, a: 1}))
  assert.ok(!sameDeclaration({a: 1}, {a: 2}))
})

test('a case that extends a pinned preset runs on the preset chain', async () => {
  const {caseChain} = await import('../../web/src/dsl-form.js')
  const preset = {id: 'p1', kind: 'chain-preset', content: {id: 'base', chain: 'stablenet'}}
  assert.equal(caseChain({content: {schemaVersion: '2', chainPreset: {extends: 'base'}}, presetRefs: [{id: 'p1', revision: 1}]}, [preset]), 'stablenet')
  assert.equal(caseChain({content: {schemaVersion: '2', chainPreset: 'base'}, presetRefs: [{id: 'p1', revision: 1}]}, [preset]), 'stablenet')
  assert.equal(caseChain({content: {schemaVersion: '2', chainPreset: {chain: 'wbft'}}}, [preset]), 'wbft')
  assert.equal(caseChain({content: {schemaVersion: '1', chain: {name: 'wemix'}}}), 'wemix')
  assert.equal(caseChain({content: {schemaVersion: '2', chainPreset: 'base'}}, [preset]), undefined)
})

test('named binaries come from the case over its pinned preset', async () => {
  const {declaredBinaries} = await import('../../web/src/dsl-form.js')
  const preset = {id: 'p1', content: {id: 'wemix-to-wbft', chain: 'wemix', binaries: {default: 'gwemix', next: {binary: 'gwbft', chain: 'wbft'}}}}
  assert.deepEqual(declaredBinaries({content: {schemaVersion: '2', chainPreset: {extends: 'wemix-to-wbft'}}, presetRefs: [{id: 'p1', revision: 1}]}, [preset]), [{name: 'next', chain: 'wbft'}])
  assert.deepEqual(declaredBinaries({content: {schemaVersion: '2', chainPreset: {chain: 'stablenet', binaries: {default: 'gstable', upgrade: 'gstable-next'}}}}), [{name: 'upgrade', chain: 'stablenet'}])
  assert.deepEqual(declaredBinaries({content: {schemaVersion: '2', chainPreset: {chain: 'stablenet', binaries: {default: 'gstable'}}}}), [])
})

test('a case attaches when its own or its pinned preset declares attach', async () => {
  const {caseAttaches} = await import('../../web/src/dsl-form.js')
  const attached = {id: 'p2', kind: 'chain-preset', content: {chain: 'stablenet', attach: {rpc: ['http://x']}}}
  const composed = {id: 'p1', kind: 'chain-preset', content: {chain: 'stablenet', topology: {bp: 4}}}
  assert.equal(caseAttaches({content: {chainPreset: {chain: 'stablenet', attach: {rpc: ['http://x']}}}}), true)
  assert.equal(caseAttaches({content: {chainPreset: 'stablenet-attached'}, presetRefs: [{id: 'p2', revision: 1}]}, [attached]), true)
  assert.equal(caseAttaches({content: {chainPreset: {extends: 'base'}}, presetRefs: [{id: 'p1', revision: 1}]}, [composed]), false)
  assert.equal(caseAttaches({content: {chainPreset: {chain: 'stablenet', topology: {bp: 4}}}}), false)
})

test('key file accounts come from the case and the preset it pins', async () => {
  const {keyFileAccounts} = await import('../../web/src/dsl-form.js')
  const funded = {id: 'p3', kind: 'chain-preset', content: {chain: 'stablenet', attach: {rpc: ['x']}, accounts: {payer: {keyFile: '${KEY}'}, minted: {}}}}
  assert.deepEqual(keyFileAccounts({content: {chainPreset: 'stablenet-testnet-funded'}, presetRefs: [{id: 'p3', revision: 1}]}, [funded]), ['payer'])
  assert.deepEqual(keyFileAccounts({content: {chainPreset: {chain: 'stablenet', attach: {rpc: ['x']}, accounts: {b: {keyFile: 'k'}, a: {keyFile: 'k'}}}}}), ['a', 'b'])
  assert.deepEqual(keyFileAccounts({content: {chainPreset: {chain: 'stablenet'}}}), [])
})

test('a string launch value is checked against the engine rule the schema carries', async () => {
  const { valueProblem } = await import('../../web/src/dsl-form.js')
  const port = { type: 'string', pattern: '^-?[0-9]+$', 'x-minimum': 0, 'x-maximum': 65535 }
  assert.equal(valueProblem(port, '8545'), '')
  assert.match(valueProblem(port, 'abc'), /정수/)
  assert.match(valueProblem(port, '70000'), /65535/)
  assert.equal(valueProblem(port, ''), '')
  const peers = { type: 'string', pattern: '^-?[0-9]+$', 'x-minimum': 0 }
  assert.match(valueProblem(peers, '-1'), /0 이상/)
  assert.equal(valueProblem({ type: 'string' }, 'anything'), '')
  assert.match(valueProblem({ type: 'string', pattern: '^(any|none)$' }, 'other'), /형식/)
})
test('node role suggestions come from the roles the chain contract lists',()=>{
 const doc={chainPreset:{topology:{bp:1,en:1,xx:1}},steps:[]}
 const nodes=references(doc,['bp','xx']).nodes
 assert.ok(nodes.includes('xx') && nodes.includes('xx1') && nodes.includes('bp1'))
 assert.ok(!nodes.includes('en') && !nodes.includes('en1'))
 assert.deepEqual(references(doc).nodes.filter(n=>!n.startsWith('node')),[],'roles were invented without a contract')
})
test('an attached network offers its endpoints by position only',()=>{
 const nodes=references({chainPreset:{attach:{rpc:['a','b']}},steps:[]},['bp','en','pn']).nodes
 assert.deepEqual(nodes.filter(n=>n.startsWith('node')),['node1','node2'])
 assert.ok(!nodes.some(n=>/^(bp|en|pn)\d/.test(n)))
})

test('structured attach profiles expose only their declared role selectors and positions',()=>{
 const doc={chainPreset:{attach:{endpoints:[{rpc:'http://a',role:'bp'},{rpc:'http://b',role:'en'},{rpc:'http://c'}]}},steps:[]}
 const names=references(doc,['bp','en','pn']).nodes
 for(const name of ['node1','node2','node3','bp','bp1','en','en1','en2'])assert.ok(names.includes(name),name)
 for(const name of ['node4','bp2','pn','pn1'])assert.ok(!names.includes(name),name)
})

test('object constraints produce distinguishable required alternatives without changing values',()=>{
 const schema={type:'object',additionalProperties:false,properties:{rpc:{type:'array'},endpoints:{type:'array'}},oneOf:[{title:'RPC URLs',required:['rpc'],not:{required:['endpoints']}},{title:'Profiles',required:['endpoints'],not:{required:['rpc']}}]}
 const value={endpoints:[{rpc:'http://node'}]};const before=JSON.stringify(value)
 const choices=variants(schema,{})
 assert.equal(choices.length,2)
 assert.equal(selectedVariant(choices,value),1)
 assert.deepEqual(initial(choices[1],{}),{endpoints:[]})
 assert.equal(JSON.stringify(value),before)
})

test('endpoint and auth constraints hide forbidden additions while preserving imported fields',async()=>{
 const fs=await import('node:fs');const {forbiddenFields}=await import('../../web/src/dsl-form.js')
 const contract=JSON.parse(fs.readFileSync(new URL('../../internal/dsl/schema/v2.schema.json',import.meta.url),'utf8'))
 const attach=contract.$defs.envSpec.properties.attach
 const choices=variants(attach,contract)
 assert.equal(choices.length,2)
 assert.deepEqual([...forbiddenFields(choices[0],{rpc:[]})],['endpoints'])
 assert.deepEqual([...forbiddenFields(choices[1],{endpoints:[]})],['rpc'])
 const auth=attach.properties.endpoints.items.properties.auth
 const imported={type:'jwt',env:'TOKEN',header:'X-Ignored'};const before=JSON.stringify(imported)
 assert.deepEqual([...forbiddenFields(auth,imported)],['header'])
 assert.deepEqual([...forbiddenFields(auth,{type:'api-key',env:'TOKEN'})],[])
 assert.equal(JSON.stringify(imported),before,'a forbidden imported field was discarded')
})

test('profile role selectors also follow the imported shared preset',()=>{
 const external={schemaVersion:'2',kind:'chain-preset',id:'external',chain:'wbft',attach:{endpoints:[{rpc:'http://a',role:'bp'},{rpc:'http://b',role:'en'}]}}
 const names=references({schemaVersion:'2',chainPreset:'external',steps:[]},['bp','en','pn'],{}, {external}).nodes
 for(const name of ['node1','node2','bp1','en1'])assert.ok(names.includes(name),name)
 assert.ok(!names.includes('node3'))
})

test('fee payer choices merge declared accounts and exclude node-signed identities',()=>{
 const own={schemaVersion:'2',chainPreset:{extends:'base',accounts:{sponsor:{},node1:{},faucet:{},'0xdead':{}}}}
 const inherited={base:{accounts:{payer:{keyFile:'${PAYER_KEY}'},sponsor:{keyFile:'${SPONSOR_KEY}'}}}}
 const before=JSON.stringify({own,inherited})
 assert.deepEqual(references(own,['bp','en'],{},inherited).accounts,['payer','sponsor'])
 assert.equal(JSON.stringify({own,inherited}),before)
})
test('mutually exclusive transaction fields cannot be added together',async()=>{
 const {forbiddenFields}=await import('../../web/src/dsl-form.js')
 const schema={allOf:[{not:{required:['feePayer','feePayerKey']}},{not:{required:['feePayer','gasPrice']}}]}
 assert.deepEqual([...forbiddenFields(schema,{feePayer:'sponsor'})],['feePayerKey','gasPrice'])
 assert.deepEqual([...forbiddenFields(schema,{feePayerKey:'$key'})],['feePayer'])
 assert.deepEqual([...forbiddenFields(schema,{})],[])
})

test('preset values are offered before hooks with deep-merge deletion and shadowing',()=>{
 const base={values:{count:1,inherited:true,removed:'old'}}
 const own={schemaVersion:'2',chainPreset:{extends:'base',values:{count:2,removed:null,recipient:'0xabc'}},hooks:{pre:[{do:'read',source:'blockNumber',save:'count'}]},steps:[{expect:'blockNumber',is:'$count'}]}
 const before=JSON.stringify({base,own})
 const choices=references(own,['bp'],{}, {base})
 assert.deepEqual(choices.before.get('/content/hooks/pre/0'),['$count','$inherited','$recipient'])
 assert.deepEqual(choices.before.get('/content/steps/0'),['$count','$inherited','$recipient'])
 assert.equal(JSON.stringify({base,own}),before)
 assert.ok(references({schemaVersion:'2',chainPreset:{values:{empty:null}},steps:[{expect:'blockNumber'}]}).before.get('/content/steps/0').includes('$empty'))
})
test('recursive preset JSON values expose a real null variant',async()=>{
 const fs=await import('node:fs')
 const root=JSON.parse(fs.readFileSync(new URL('../../internal/dsl/schema/v2.schema.json',import.meta.url),'utf8'))
 const choices=variants({$ref:'#/$defs/presetValue'},root)
 assert.equal(choices.length,6)
 assert.equal(selectedVariant(choices,null),3)
 assert.equal(initial(choices[3],root),null)
})

test('map fields use the engine pattern contract for declared chain identity',async()=>{
 const {propertySchema}=await import('../../web/src/dsl-form.js')
 const schema={patternProperties:{'^chainId$':{type:'integer',minimum:0,maximum:9007199254740991}},additionalProperties:{type:'string'}}
 assert.equal(propertySchema(schema,'chainId').type,'integer')
 assert.equal(propertySchema(schema,'other').type,'string')
 assert.equal(initial(propertySchema(schema,'chainId'),{}),0)
})
