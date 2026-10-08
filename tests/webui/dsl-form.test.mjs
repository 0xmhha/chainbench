import test from 'node:test'
import assert from 'node:assert/strict'
import {variants, initial, selectedVariant, references} from '../../web/src/dsl-form.js'
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
