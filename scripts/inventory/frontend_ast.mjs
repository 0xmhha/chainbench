// Parse source only; no source file is executed.
import fs from 'node:fs'
import {createRequire} from 'node:module'
const require=createRequire(new URL('../../web/package.json',import.meta.url))
const acorn=require('acorn')
const svelte=require('svelte/compiler')
const postcss=require('postcss')
if(process.argv.includes('--versions')){
 process.stdout.write(JSON.stringify({node:process.version,acorn:require('acorn/package.json').version,svelte:require('svelte/package.json').version,postcss:require('postcss/package.json').version}));process.exit(0)
}
const inputs=JSON.parse(fs.readFileSync(0,'utf8'))
function parse(input){
 const nodes=[];let count=0
 try{
  const ast=input.language==='svelte'?svelte.parse(input.source,{modern:true}):input.language==='css'?postcss.parse(input.source):acorn.parse(input.source,{ecmaVersion:'latest',sourceType:'module',locations:true})
  const seen=new WeakSet()
  function name(n){if(!n)return '';if(n.type==='Identifier')return n.name;if(n.type==='MemberExpression')return name(n.object)+'.'+name(n.property);return 'anonymous'}
  function add(kind,n,label){nodes.push({kind,name:label,line:n.loc?.start?.line??n.source?.start?.line??input.source.slice(0,n.start??0).split('\n').length,start:n.start??0,end:n.end??0})}
  function visit(n){
   if(!n||typeof n!=='object'||seen.has(n))return
   seen.add(n)
   if(n.type){count++
    if(n.type==='ImportDeclaration'||n.type==='ExportAllDeclaration'||n.type==='ExportNamedDeclaration'&&n.source)add('import',n,n.source.value)
    if(n.type==='FunctionDeclaration'||n.type==='ClassDeclaration')add(n.type==='ClassDeclaration'?'type':'function',n,n.id?.name??'anonymous')
    if(n.type==='VariableDeclarator')add('binding',n,name(n.id))
    if(n.type==='CallExpression')add('call',n,name(n.callee))
    if(n.type==='ImportExpression'&&typeof n.source?.value==='string')add('import',n,n.source.value)
    if(n.type==='rule')add('selector',n,n.selector)
   }
   for(const [key,value] of Object.entries(n))if(!['parent','loc','source','comments','tokens'].includes(key)){
    if(Array.isArray(value))value.forEach(visit);else visit(value)
   }
  }
  visit(ast)
  return {path:input.path,astNodes:count,nodes}
 }catch(error){return {path:input.path,error:String(error.message),astNodes:0,nodes:[]}}
}
process.stdout.write(JSON.stringify(inputs.map(parse)))
