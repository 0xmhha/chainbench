// schemaProblems checks a value against the rules its engine schema states
// (required fields, enum, minimum and maximum, minLength and pattern) and
// names each broken rule at the field path the deployment form uses, so the
// field can be marked before the server is asked. The server stays the judge.
export function schemaProblems(schema, value, path = 'content') {
 const out = []
 if (!schema) return out
 if (schema.type === 'object') {
  const missing = (schema.required ?? []).filter(key => value?.[key] === undefined || value[key] === '')
  for (const key of missing) out.push({ path: `${path}.${key}`, message: `${schema.properties?.[key]?.title ?? key} 값이 필요합니다.` })
  for (const [key, spec] of Object.entries(schema.properties ?? {})) if (value?.[key] !== undefined && !missing.includes(key)) out.push(...schemaProblems(spec, value[key], `${path}.${key}`))
  return out
 }
 if (schema.type === 'array') {
  (value ?? []).forEach((item, i) => out.push(...schemaProblems(schema.items, item, `${path}[${i}]`)))
  return out
 }
 if (schema.enum && !schema.enum.includes(value)) return [{ path, message: `${schema.enum.join(', ')} 중 하나여야 합니다.` }]
 if (typeof value === 'number') {
  if (schema.minimum !== undefined && value < schema.minimum) return [{ path, message: `${schema.minimum} 이상이어야 합니다.` }]
  if (schema.maximum !== undefined && value > schema.maximum) return [{ path, message: `${schema.maximum} 이하여야 합니다.` }]
 }
 if (typeof value === 'string') {
  if (schema.minLength && value.length < schema.minLength) return [{ path, message: '비어 있을 수 없습니다.' }]
  if (schema.pattern && value !== '' && !new RegExp(schema.pattern).test(value)) return [{ path, message: '허용된 형식이 아닙니다.' }]
 }
 return out
}

// fieldPath turns a server pointer (/content/pool/hosts/1/addr) into the field
// path the form uses (content.pool.hosts[1].addr), following the schema so an
// array position and an object key that happens to be a number stay apart.
// A request field outside the content keeps its name (/name is name), and a
// refusal of the content as a whole names no field.
export function fieldPath(schema, pointer) {
 if (typeof pointer !== 'string' || !pointer.startsWith('/')) return ''
 const keys = pointer.slice(1).split('/').map(k => k.replaceAll('~1', '/').replaceAll('~0', '~'))
 if (keys[0] !== 'content') return keys.join('.')
 let path = 'content', spec = schema
 for (const key of keys.slice(1)) {
  if (spec?.type === 'array') { path += `[${key}]`; spec = spec.items; continue }
  path += `.${key}`
  spec = spec?.properties?.[key] ?? spec?.additionalProperties
 }
 return path === 'content' ? '' : path
}

// schemaDefault is the value a newly added field starts with: the engine
// default the contract offers, else the first choice or an empty value of
// its type. An object starts with its required fields filled the same way.
export function schemaDefault(schema) {
 if (schema.default !== undefined) return schema.default
 if (schema.enum) return schema.enum[0]
 if (schema.type === 'object') return Object.fromEntries((schema.required ?? []).map(k => [k, schemaDefault(schema.properties[k])]))
 if (schema.type === 'array') return []
 if (schema.type === 'integer') return 0
 if (schema.type === 'boolean') return false
 return ''
}
