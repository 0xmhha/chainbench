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
 const suggestions=references(doc)
 assert.deepEqual(suggestions.before.get('/content/steps/0'),['$addr','$key'])
 assert.deepEqual(suggestions.before.get('/content/steps/1'),['$addr','$key','$head'])
 assert.ok(suggestions.nodes.includes('node3'))
 assert.ok(!suggestions.nodes.includes('node4'))
})
test('legacy readers select their own nested argument schema',()=>{
 const choices=['balanceAt','blockNumber'].map(source=>({type:'object',required:['read'],additionalProperties:false,properties:{read:{type:'object',properties:{source:{const:source}}}}}))
 assert.equal(selectedVariant(choices,{read:{source:'blockNumber'}}),1)
 const suggestions=references({schemaVersion:'1',topology:{bp:2},steps:[{read:{source:'blockNumber',save:'height'}}],assertions:[{assert:'blockNumber',expected:'$height'}]})
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
