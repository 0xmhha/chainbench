import test from 'node:test'
import assert from 'node:assert/strict'
import {selectedVariant} from '../../web/src/dsl-form.js'

test('preset count and table variants are distinguished by their allowed fields',()=>{
 const choices=[
  {type:'object',additionalProperties:false,properties:{bp:{type:'integer'},en:{type:'integer'},pn:{type:'integer'},syncMode:{type:'string'}}},
  {type:'object',additionalProperties:false,required:['nodes'],properties:{nodes:{type:'array'}}}
 ]
 assert.equal(selectedVariant(choices,{nodes:[{role:'bp'}]}),1)
 assert.equal(selectedVariant(choices,{bp:4}),0)
 assert.equal(selectedVariant(choices,{}),0)
 assert.equal(selectedVariant(choices,{bp:4,extension:true}),0)
})
