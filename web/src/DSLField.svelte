<script>
 import Field from './DSLField.svelte'
 import {variants, selectedVariant, variantLabel, initial} from './dsl-form.js'
 let {schema,root,value,path='/content',label='Document',onchange,suggestions={nodes:[],before:new Map()}}=$props()
 let key=$state('')
 const choices=$derived(variants(schema,root))
 const index=$derived(selectedVariant(choices,value))
 const current=$derived(choices[index]??{})
 const type=$derived(current.type??(current.properties?'object':Array.isArray(value)?'array':value===null?'null':typeof value==='object'?'object':typeof value))
 const properties=$derived(current.properties??{})
 const optional=$derived(Object.keys(properties).filter(k=>!(value && Object.hasOwn(value,k))))
 const refs=$derived([...suggestions.before].find(([prefix])=>path.startsWith(prefix+'/'))?.[1]??[])
 function update(k,v){onchange({...value,[k]:v})}
 function remove(k){const next={...value};delete next[k];onchange(next)}
 function item(i,v){const next=[...value];next[i]=v;onchange(next)}
 function move(i,delta){const next=[...value];[next[i],next[i+delta]]=[next[i+delta],next[i]];onchange(next)}
</script>
<div class="field" data-field-path={path}>
 {#if choices.length>1}
  <label>{label} variant <select aria-label={`${path} variant`} value={index} onchange={e=>onchange(initial(choices[Number(e.target.value)],root))}>
   {#each choices as choice,i}<option value={i}>{variantLabel(choice,i)}</option>{/each}
  </select></label>
 {/if}
 {#if current.const!==undefined}
  <span>{label}: <strong>{String(value??current.const)}</strong></span>
  {#if value!==current.const}<p role="alert">Unsupported value retained: {String(value)}. Choose a supported variant.</p>{/if}
 {:else if current.enum}
  <label>{label} <select aria-label={path} value={value} onchange={e=>onchange(e.target.value)}>{#if !current.enum.includes(value)}<option value={value}>Unsupported · {String(value)}</option>{/if}{#each current.enum as option}<option value={option}>{option}</option>{/each}</select></label>
  {#if !current.enum.includes(value)}<p role="alert">Unsupported value retained: {String(value)}</p>{/if}
 {:else if type==='object'}
  <fieldset><legend>{label}</legend>
   {#each Object.entries(value??{}) as [name,child] (name)}
    <div class="row">
     <Field schema={properties[name]??(typeof current.additionalProperties==='object'?current.additionalProperties:{})} {root} value={child} path={`${path}/${name}`} label={name} {suggestions} onchange={v=>update(name,v)} />
     {#if !current.required?.includes(name) && properties[name]?.const===undefined}<button aria-label={`Remove ${path}/${name}`} onclick={()=>remove(name)}>Remove</button>{/if}
     {#if !Object.hasOwn(properties,name) && current.additionalProperties===false}<p role="alert">Unsupported field: {name}. Retained until you explicitly remove it.</p>{/if}
    </div>
   {/each}
   {#if optional.length}
    <label>Add {label} field <select aria-label={`Add ${path} field`} value="" onchange={e=>{if(e.target.value)update(e.target.value,initial(properties[e.target.value],root))}}><option value="">Choose a field</option>{#each optional as name}<option value={name}>{name}{current.required?.includes(name)?' (required)':''}</option>{/each}</select></label>
   {/if}
   {#if current.additionalProperties!==false}
    <label>Entry name <input aria-label={`${path} entry name`} bind:value={key} /></label>
    <button disabled={!key||Object.hasOwn(value??{},key)} onclick={()=>{update(key,initial(typeof current.additionalProperties==='object'?current.additionalProperties:{},root));key=''}}>Add entry</button>
   {/if}
  </fieldset>
 {:else if type==='array'}
  <fieldset><legend>{label} · ordered list</legend>
   {#each value??[] as child,i}
    <div class="row">
     <Field schema={current.items??{}} {root} value={child} path={`${path}/${i}`} label={`${label} ${i+1}`} {suggestions} onchange={v=>item(i,v)} />
     <button aria-label={`Move ${path}/${i} up`} disabled={i===0} onclick={()=>move(i,-1)}>↑</button>
     <button aria-label={`Move ${path}/${i} down`} disabled={i===(value.length-1)} onclick={()=>move(i,1)}>↓</button>
     <button aria-label={`Remove ${path}/${i}`} onclick={()=>onchange(value.filter((_,j)=>i!==j))}>Remove</button>
    </div>
   {/each}
   <button aria-label={`Add ${path} item`} onclick={()=>onchange([...(value??[]),initial(current.items??{},root)])}>Add {label} item</button>
  </fieldset>
 {:else if type==='boolean'}
  <label><input aria-label={path} type="checkbox" checked={value} onchange={e=>onchange(e.target.checked)} /> {label}</label>
 {:else if type==='number'||type==='integer'}
  <label>{label} <input aria-label={path} type="number" value={value} min={current.minimum} max={current.maximum} step={type==='integer'?1:'any'} oninput={e=>{if(e.target.value!=='')onchange(Number(e.target.value))}} /></label>
 {:else if type==='null'}
  <span>{label}: null</span>
 {:else}
  <label>{label} <input aria-label={path} value={value??''} list={`${path}-suggestions`} oninput={e=>onchange(e.target.value)} /></label>
  <datalist id={`${path}-suggestions`}>{#each label==='on'?suggestions.nodes:path.includes('/onEach/')?['all',...suggestions.nodes]:refs as option}<option value={option}></option>{/each}</datalist>
 {/if}
 {#if !current.type&&!current.properties&&!current.enum&&current.const===undefined&&choices.length===1}
  <label>{label} value type <select aria-label={`${path} value type`} value={type} onchange={e=>onchange(({string:'',number:0,boolean:false,object:{},array:[],null:null})[e.target.value])}>{#each ['string','number','boolean','object','array','null'] as kind}<option value={kind}>{kind}</option>{/each}</select></label>
 {/if}
 {#if current.description}<small>{current.description}</small>{/if}
</div>
<style>
.field{min-width:0;flex:1}fieldset{border:1px solid #374961;margin:.6rem 0;padding:.7rem;border-radius:6px}legend{color:#9eb6d7}.row{display:flex;align-items:start;gap:.4rem;padding:.3rem 0;flex-wrap:wrap}label{display:block;margin:.3rem 0}input,select,button{font:inherit;padding:.35rem;background:#202b3b;color:#e3ebf5;border:1px solid #47576d;border-radius:4px}select,input{max-width:100%}button{cursor:pointer}button:disabled{opacity:.4}small{display:block;color:#99acc4}p{color:#ffaaa0}
</style>
