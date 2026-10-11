<script>
 import Field from './DSLField.svelte'
 import {variants, selectedVariant, variantLabel, initial, valueProblem} from './dsl-form.js'
 // errors maps a field path to the engine's message about it; that field is
 // marked invalid and described by the message.
 let {schema,root,value,path='/content',label='문서',onchange,suggestions={nodes:[],before:new Map()},errors={}}=$props()
 // A fixed (const) property has no control of its own, so an error about it
 // marks the field that chooses this value's form instead.
 const fieldError=$derived(errors[path]??Object.entries(variants(schema,root)[selectedVariant(variants(schema,root),value)]?.properties??{}).filter(([,p])=>p.const!==undefined).map(([k])=>errors[`${path}/${k}`]).find(Boolean)??'')
 const errorId=$derived(`${path}-error`)
 const invalid=$derived(fieldError?{'aria-invalid':'true','aria-describedby':errorId}:{})
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
  <label>{label} 형식 <select aria-label={`${path} 형식`} {...invalid} value={index} onchange={e=>onchange(initial(choices[Number(e.target.value)],root))}>
   {#each choices as choice,i}<option value={i}>{variantLabel(choice,i)}</option>{/each}
  </select></label>
 {/if}
 {#if current.const!==undefined}
  <span>{label}: <strong>{String(value??current.const)}</strong></span>
  {#if value!==current.const}<p role="alert">지원하지 않는 값을 그대로 두었습니다: {String(value)}. 지원하는 형식을 고르세요.</p>{/if}
 {:else if current.enum}
  <label>{label} <select aria-label={path} {...invalid} value={value} onchange={e=>onchange(e.target.value)}>{#if !current.enum.includes(value)}<option value={value}>지원하지 않음 · {String(value)}</option>{/if}{#each current.enum as option}<option value={option}>{option}</option>{/each}</select></label>
  {#if !current.enum.includes(value)}<p role="alert">지원하지 않는 값을 그대로 두었습니다: {String(value)}</p>{/if}
 {:else if type==='object'}
  <fieldset><legend>{label}</legend>
   {#each Object.entries(value??{}) as [name,child] (name)}
    <div class="row">
     <Field schema={properties[name]??(typeof current.additionalProperties==='object'?current.additionalProperties:{})} {root} value={child} path={`${path}/${name}`} label={name} {suggestions} {errors} onchange={v=>update(name,v)} />
     {#if !current.required?.includes(name) && properties[name]?.const===undefined}<button aria-label={`${path}/${name} 삭제`} onclick={()=>remove(name)}>삭제</button>{/if}
     {#if !Object.hasOwn(properties,name) && current.additionalProperties===false}<p role="alert">지원하지 않는 필드입니다: {name}. 직접 삭제할 때까지 그대로 둡니다.</p>{/if}
    </div>
   {/each}
   {#if optional.length}
    <label>{label} 필드 추가 <select aria-label={`${path} 필드 추가`} value="" onchange={e=>{if(e.target.value)update(e.target.value,initial(properties[e.target.value],root))}}><option value="">필드 선택</option>{#each optional as name}<option value={name}>{name}{current.required?.includes(name)?' (필수)':''}</option>{/each}</select></label>
   {/if}
   {#if current.additionalProperties!==false}
    <label>항목 이름 <input aria-label={`${path} 항목 이름`} bind:value={key} /></label>
    <button disabled={!key||Object.hasOwn(value??{},key)} onclick={()=>{update(key,initial(typeof current.additionalProperties==='object'?current.additionalProperties:{},root));key=''}}>항목 추가</button>
   {/if}
  </fieldset>
 {:else if type==='array'}
  <fieldset><legend>{label} · 순서 목록</legend>
   {#each value??[] as child,i}
    <div class="row">
     <Field schema={current.items??{}} {root} value={child} path={`${path}/${i}`} label={`${label} ${i+1}`} {suggestions} {errors} onchange={v=>item(i,v)} />
     <button aria-label={`${path}/${i} 위로`} disabled={i===0} onclick={()=>move(i,-1)}>↑</button>
     <button aria-label={`${path}/${i} 아래로`} disabled={i===(value.length-1)} onclick={()=>move(i,1)}>↓</button>
     <button aria-label={`${path}/${i} 삭제`} onclick={()=>onchange(value.filter((_,j)=>i!==j))}>삭제</button>
    </div>
   {/each}
   <button aria-label={`${path} 항목 추가`} onclick={()=>onchange([...(value??[]),initial(current.items??{},root)])}>{label} 항목 추가</button>
  </fieldset>
 {:else if type==='boolean'}
  <label><input aria-label={path} {...invalid} type="checkbox" checked={value} onchange={e=>onchange(e.target.checked)} /> {label}</label>
 {:else if type==='number'||type==='integer'}
  <label>{label} <input aria-label={path} {...invalid} type="number" value={value} min={current.minimum} max={current.maximum} step={type==='integer'?1:'any'} oninput={e=>{if(e.target.value!=='')onchange(Number(e.target.value))}} /></label>
 {:else if type==='null'}
  <span>{label}: null</span>
 {:else}
  {@const problem=valueProblem(current,value)}
  <label>{label} <input aria-label={path} value={value??''} list={`${path}-suggestions`} pattern={current.pattern} aria-invalid={problem||fieldError?'true':undefined} aria-describedby={problem?`${path}-problem`:fieldError?errorId:undefined} oninput={e=>onchange(e.target.value)} /></label>
  {#if problem}<p id={`${path}-problem`} role="alert">{problem}</p>{/if}
  <datalist id={`${path}-suggestions`}>{#each label==='on'?suggestions.nodes:path.includes('/onEach/')?['all',...suggestions.nodes]:refs as option}<option value={option}></option>{/each}</datalist>
 {/if}
 {#if !current.type&&!current.properties&&!current.enum&&current.const===undefined&&choices.length===1}
  <label>{label} 값 종류 <select aria-label={`${path} 값 종류`} value={type} onchange={e=>onchange(({string:'',number:0,boolean:false,object:{},array:[],null:null})[e.target.value])}>{#each ['string','number','boolean','object','array','null'] as kind}<option value={kind}>{kind}</option>{/each}</select></label>
 {/if}
 {#if fieldError}<p id={errorId} class="field-error" data-error-path={path}>{fieldError}</p>{/if}
 {#if current.description}<small>{current.description}</small>{/if}
</div>
<style>
.field{min-width:0;flex:1}fieldset{border:1px solid var(--border);margin:8px 0;padding:12px;border-radius:6px}legend{color:var(--text-muted)}.row{display:flex;align-items:start;gap:8px;padding:4px 0;flex-wrap:wrap}label{display:block;margin:4px 0}input,select,button{font:inherit;padding:6px;background:var(--surface-raised);color:var(--text);border:1px solid var(--border-strong);border-radius:4px;min-height:36px}select,input{max-width:100%}button{cursor:pointer}button:disabled{opacity:.4}small{display:block;color:var(--text-muted)}p{color:var(--danger)}[aria-invalid=true]{border-color:var(--danger);outline:1px solid var(--danger)}
</style>
