export function resolve(schema, root) {
  if (schema?.$ref) return resolve(root.$defs[schema.$ref.replace('#/$defs/', '')], root)
  return schema ?? {}
}
export function variants(schema, root) {
  schema = resolve(schema, root)
  const choices = schema.oneOf ?? schema.anyOf
  if (!choices) return [schema]
  if (schema.properties) {
    if (!choices.every(choice=>choice.required)) return [schema]
    return choices.map(choice=>({...schema,...choice,properties:{...schema.properties,...choice.properties},required:[...new Set([...(schema.required??[]),...(choice.required??[])])],oneOf:undefined,anyOf:undefined}))
  }
  return choices.flatMap(choice => variants(choice, root))
}
export function variantLabel(schema, index) {
  if(schema.title) return schema.title
  if(schema.not?.required && schema.required?.length) return schema.required.join(' / ')
  const p = schema.properties ?? {}
  if (p.do?.const) return `do ${p.do.const}${p.source?.const ? ' · ' + p.source.const : ''}`
  if (p.expect?.const) return `expect ${p.expect.const}`
  return schema.type ?? `Choice ${index + 1}`
}
function compatible(schema, value) {
  if(schema.const!==undefined)return schema.const===value
  if(schema.enum && !schema.enum.includes(value))return false
  if(schema.required?.some(k=>!value || !Object.hasOwn(value,k)))return false
  if(schema.not?.required?.every(k=>value && Object.hasOwn(value,k)))return false
  if(schema.type==='object'){
    if(!value||typeof value!=='object'||Array.isArray(value))return false
    if(schema.additionalProperties===false && Object.keys(value).some(k=>!Object.hasOwn(schema.properties??{},k)))return false
    return Object.entries(schema.properties??{}).every(([k,p])=>!Object.hasOwn(value,k)||compatible(p,value[k]))
  }
  if(schema.type==='array')return Array.isArray(value)
  if(schema.type==='null')return value===null
  return schema.type===undefined||schema.type===typeof value||(schema.type==='integer'&&Number.isInteger(value))
}
export function selectedVariant(choices, value) {
  const index=choices.findIndex(s=>compatible(s,value))
  return index<0?0:index
}
export function initial(schema, root) {
  schema = variants(schema, root)[0]
  if (schema.const !== undefined) return schema.const
  if (schema.default !== undefined) return structuredClone(schema.default)
  if (schema.enum) return schema.enum[0]
  if (schema.type === 'object' || schema.properties) {
    return Object.fromEntries(Object.entries(schema.properties ?? {}).filter(([k,p]) => schema.required?.includes(k) || p.const !== undefined).map(([k,p]) => [k,initial(p,root)]))
  }
  if (schema.type === 'array') return []
  if (schema.type === 'boolean') return false
  if (schema.type === 'null') return null
  if (schema.type === 'integer' || schema.type === 'number') return 0
  return ''
}
// references lists the node names and prior variable bindings a statement may
// use. roles are the node roles the chain contract lists, in its order, and
// defaults their counts when the case declares no topology. An attached
// network keeps positions; structured profiles also expose declared roles.
export function references(doc, roles = [], defaults = {}, presets = {}) {
  const legacy=doc?.schemaVersion==='1'
  const ref=typeof doc?.chainPreset==='string'?doc.chainPreset:doc?.chainPreset?.extends
  const inherited=presets[ref]??{}
  const own=typeof doc?.chainPreset==='object'?doc.chainPreset:{}
  const preset=legacy?{topology:doc.topology,accounts:{}}:{...inherited,...own,topology:own.topology?{...inherited.topology,...own.topology}:inherited.topology}
  const profiles=preset?.attach?.endpoints
  const nodes=profiles?[]:[...roles]
  const attached = preset?.attach ? {nodes:(preset.attach.rpc??[]).map(()=>({}))} : null
  const topology = preset?.topology ?? attached ?? defaults
  if(profiles){
    const counts={}
    profiles.forEach((n,i)=>{
      nodes.push(`node${i+1}`)
      const role=n.role??'en'
      if(roles.includes(role)){
        counts[role]=(counts[role]??0)+1
        if(counts[role]===1)nodes.push(role)
        nodes.push(`${role}${counts[role]}`)
      }
    })
  }
  else if (topology.nodes) topology.nodes.forEach((n,i) => nodes.push(`node${n.index ?? i+1}`))
  else { let count=0; for (const role of roles) for(let i=1;i<=(topology[role]??0);i++){nodes.push(`${role}${i}`);nodes.push(`node${++count}`)} }
  const values={...inherited.values}
  if(own.values===null){for(const name of Object.keys(values))delete values[name]}
  else for(const [name,value] of Object.entries(own.values??{})){
    if(ref&&value===null)delete values[name]
    else values[name]=value
  }
  const bindings = [...new Set([...Object.keys(values),...Object.keys(doc?.prepared?.imports??{})])].sort().map(name=>'$'+name)
  const privateBindings = new Set()
  const before = new Map()
  const lists=legacy?[['/content/preActions',doc?.preActions],['/content/steps',doc?.steps],['/content/assertions',doc?.assertions],['/content/postActions',doc?.postActions]]:[['/content/hooks/pre',doc?.hooks?.pre],['/content/steps',doc?.steps],['/content/hooks/post',doc?.hooks?.post],['/content/hooks/onFail',doc?.hooks?.onFail]]
  for (const [path,list] of lists) {
    if(path==='/content/hooks/onFail')before.set('/content/prepared/exports',bindings.filter(name=>!privateBindings.has(name)))
    for (const [i,statement] of (list ?? []).entries()) {
      before.set(`${path}/${i}`, [...bindings])
      const args=legacy&&path!=='/content/assertions'?(Object.values(statement)[0]??{}):statement
      for (const key of ['save','saveKey']) if (args[key]&&!bindings.includes('$'+args[key])) bindings.push('$'+args[key])
      if(args.saveKey)privateBindings.add('$'+args.saveKey)
    }
  }
  if(!before.has('/content/prepared/exports'))before.set('/content/prepared/exports',bindings.filter(name=>!privateBindings.has(name)))
  const accounts=Object.keys({...inherited.accounts,...own.accounts})
    .filter(name=>name!=='faucet'&&!/^node[1-9][0-9]*$/.test(name)&&!/^0[xX]/.test(name)).sort()
  return {nodes, before, accounts}
}

// Producer choices are declarations from the saved library, never guessed
// exports or an ambiguous case ID. Unknown imported choices remain visible.
export function preparedChoices(doc, saved = []) {
  const counts=new Map()
  for(const item of saved){const id=item.content?.id;if(id)counts.set(id,(counts.get(id)??0)+1)}
  const sources=saved.map(item=>item.content).filter(c=>c?.id&&c.id!==doc?.id&&counts.get(c.id)===1&&Object.keys(c.prepared?.exports??{}).length)
  const preparedCases=sources.map(c=>c.id).sort()
  const preparedNames=Object.fromEntries(Object.entries(doc?.prepared?.imports??{}).map(([alias,ref])=>[alias,Object.keys(sources.find(c=>c.id===ref.case)?.prepared?.exports??{}).sort()]))
  return {preparedCases,preparedNames}
}

export function finishedGenesisRef(doc) {
  if (doc?.schemaVersion === '1') return doc.chain?.genesisExisting ?? ''
  return typeof doc?.chainPreset === 'object' ? doc.chainPreset?.genesis?.ref ?? '' : ''
}
export function caseAssetRefs(doc) {
  const ref = finishedGenesisRef(doc)
  return /^asset:[0-9a-f]{32}$/.test(ref) ? [ref.slice(6)] : []
}
// The chain preset ids a v2 case names by id or extends; the declarations
// behind them are pinned as shared document revisions when the case is saved.
export function presetIds(doc) {
  if (!doc || doc.schemaVersion !== '2') return []
  const preset = doc.chainPreset
  if (typeof preset === 'string') return [preset]
  if (preset && typeof preset.extends === 'string') return [preset.extends]
  return []
}
// Declarations compare by content, not by key order.
export function sameDeclaration(a, b) {
  const norm = v => Array.isArray(v) ? v.map(norm) : v && typeof v === 'object' ? Object.fromEntries(Object.keys(v).sort().map(k => [k, norm(v[k])])) : v
  return JSON.stringify(norm(a)) === JSON.stringify(norm(b))
}
// The chain a saved case runs on: written in the case, or in the shared
// preset revision it pins when it names its preset by id or extends it.
export function caseChain(doc, documents = []) {
  const content = doc?.content ?? {}
  const preset = content.chainPreset
  const own = (typeof preset === 'object' ? preset?.chain : undefined) ?? content.chain?.name
  if (own) return own
  for (const ref of doc?.presetRefs ?? []) {
    const shared = documents.find(d => d.id === ref.id)
    if (shared?.content?.chain) return shared.content.chain
  }
  return undefined
}
// Whether a saved case attaches to a running network rather than composing
// one: from the case, or from a shared preset revision it pins.
export function caseAttaches(doc, documents = []) {
  const preset = doc?.content?.chainPreset
  if (typeof preset === 'object' && preset?.attach) return true
  return (doc?.presetRefs ?? []).some(ref => Boolean(documents.find(d => d.id === ref.id)?.content?.attach))
}
// The account labels a saved case reads from a key file, from the case and
// the shared preset revisions it pins. Each needs a private account key.
export function keyFileAccounts(doc, documents = []) {
  const own = typeof doc?.content?.chainPreset === 'object' ? doc.content.chainPreset?.accounts ?? {} : {}
  let inherited = {}
  for (const ref of doc?.presetRefs ?? []) inherited = {...inherited, ...(documents.find(d => d.id === ref.id)?.content?.accounts ?? {})}
  return Object.entries({...inherited, ...own}).filter(([, a]) => a?.keyFile).map(([label]) => label).sort()
}
// The binary names besides default a saved case declares, with the chain
// each runs: from the case, over the shared preset revision it pins.
export function declaredBinaries(doc, documents = []) {
  const content = doc?.content ?? {}
  const own = typeof content.chainPreset === 'object' ? content.chainPreset?.binaries ?? {} : {}
  let inherited = {}
  for (const ref of doc?.presetRefs ?? []) {
    const shared = documents.find(d => d.id === ref.id)
    inherited = {...inherited, ...(shared?.content?.binaries ?? {})}
  }
  const chain = caseChain(doc, documents)
  return Object.entries({...inherited, ...own}).filter(([name]) => name !== 'default')
    .map(([name, value]) => ({name, chain: (typeof value === 'object' && value?.chain) || chain}))
    .sort((a, b) => a.name.localeCompare(b.name))
}

// valueProblem checks a string value against the rule an engine schema
// carries: a pattern and, for integers kept as strings, x-minimum and
// x-maximum. The server applies the same rule; this only tells the editor
// before it asks. An empty value is left to the required check.
export function valueProblem(schema, value) {
  if (typeof value !== 'string' || value === '' || !schema?.pattern) return ''
  if (!new RegExp(schema.pattern).test(value)) {
    return schema['x-minimum'] !== undefined ? '정수를 입력하세요.' : '허용된 형식이 아닙니다.'
  }
  const min = schema['x-minimum'], max = schema['x-maximum']
  if (min === undefined && max === undefined) return ''
  const n = Number(value)
  if (max !== undefined && (n < min || n > max)) return `${min}부터 ${max}까지의 정수를 입력하세요.`
  if (min !== undefined && n < min) return `${min} 이상의 정수를 입력하세요.`
  return ''
}

// Disallowed imported fields stay visible; this only removes forbidden additions.
export function forbiddenFields(schema, value) {
 const forbidden=new Set()
 function exclude(required){
  for(const field of required??[]){
   if(required.filter(other=>other!==field).every(other=>value&&Object.hasOwn(value,other)))forbidden.add(field)
  }
 }
 exclude(schema?.not?.required)
 for(const rule of schema?.allOf??[]){
  exclude(rule.not?.required)
  const condition=rule.if?{...rule.if,type:rule.if.type??(rule.if.properties?'object':undefined)}:null
  const active=condition&&compatible(condition,value)?rule.then:rule.else
  if(active?.not?.required?.length===1)for(const key of active.not.required)forbidden.add(key)
 }
 return forbidden
}

// An open map can reserve particular names without inventing a fixed field set.
export function propertySchema(schema, name) {
 const rules=[]
 if(schema.properties?.[name])rules.push(schema.properties[name])
 for(const [pattern,rule] of Object.entries(schema.patternProperties??{})){
  if(new RegExp(pattern).test(name))rules.push(rule)
 }
 if(rules.length)return rules.length===1?rules[0]:{...rules[0],allOf:[...(rules[0].allOf??[]),...rules.slice(1)]}
 return typeof schema.additionalProperties==='object'?schema.additionalProperties:{}
}
