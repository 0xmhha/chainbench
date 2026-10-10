export function resolve(schema, root) {
  if (schema?.$ref) return resolve(root.$defs[schema.$ref.replace('#/$defs/', '')], root)
  return schema ?? {}
}
export function variants(schema, root) {
  schema = resolve(schema, root)
  const choices = schema.oneOf ?? schema.anyOf
  if (!choices || schema.properties) return [schema]
  return choices.flatMap(choice => variants(choice, root))
}
export function variantLabel(schema, index) {
  if(schema.title) return schema.title
  const p = schema.properties ?? {}
  if (p.do?.const) return `do ${p.do.const}${p.source?.const ? ' · ' + p.source.const : ''}`
  if (p.expect?.const) return `expect ${p.expect.const}`
  return schema.type ?? `Choice ${index + 1}`
}
function compatible(schema, value) {
  if(schema.const!==undefined)return schema.const===value
  if(schema.type==='object'){
    if(!value||typeof value!=='object'||Array.isArray(value))return false
    if(schema.additionalProperties===false && (schema.required?.some(k=>!Object.hasOwn(value,k)) || Object.keys(value).some(k=>!Object.hasOwn(schema.properties??{},k))))return false
    return Object.entries(schema.properties??{}).every(([k,p])=>!Object.hasOwn(value,k)||((p.const===undefined&&!p.properties)||compatible(p,value[k])))
  }
  if(schema.type==='array')return Array.isArray(value)
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
  if (schema.type === 'integer' || schema.type === 'number') return 0
  return ''
}
export function references(doc) {
  const nodes = ['bp','en','pn']
  const legacy=doc?.schemaVersion==='1'
  const preset = legacy?{topology:doc.topology,accounts:{}}:typeof doc?.chainPreset === 'object' ? doc.chainPreset : {}
  const topology = preset?.topology ?? (preset?.attach?{en:preset.attach.rpc?.length??0}:{bp:4})
  if (topology.nodes) topology.nodes.forEach((n,i) => nodes.push(`node${n.index ?? i+1}`))
  else { let count=0; for (const role of ['bp','en','pn']) for(let i=1;i<=(topology[role]??0);i++){nodes.push(`${role}${i}`);nodes.push(`node${++count}`)} }
  const bindings = []
  const before = new Map()
  const lists=legacy?[['/content/preActions',doc?.preActions],['/content/steps',doc?.steps],['/content/assertions',doc?.assertions],['/content/postActions',doc?.postActions]]:[['/content/hooks/pre',doc?.hooks?.pre],['/content/steps',doc?.steps],['/content/hooks/post',doc?.hooks?.post],['/content/hooks/onFail',doc?.hooks?.onFail]]
  for (const [path,list] of lists) {
    for (const [i,statement] of (list ?? []).entries()) {
      before.set(`${path}/${i}`, [...bindings])
      const args=legacy&&path!=='/content/assertions'?(Object.values(statement)[0]??{}):statement
      for (const key of ['save','saveKey']) if (args[key]) bindings.push('$'+args[key])
    }
  }
  return {nodes, before}
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
