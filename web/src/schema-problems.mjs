// schemaProblems checks a value against the rules its engine schema states
// (required fields, enum, minimum and maximum, minLength and pattern) and
// names each broken rule at the field path the deployment form uses, so the
// field can be marked before the server is asked. The server stays the judge.
export function schemaProblems(schema, value, path = 'content') {
 const out = []
 if (!schema) return out
 if (schema.type === 'object') {
  const missing = (schema.required ?? []).filter(key => value?.[key] === undefined || value[key] === '')
  for (const key of missing) out.push({ path: `${path}.${key}`, message: `${key} 값이 필요합니다.` })
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
