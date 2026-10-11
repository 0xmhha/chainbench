// The statement lists of a case in grammar order: hooks around the steps for
// the current grammar, and the four legacy lists for schemaVersion 1.
const lists = {
 '2': [['/content/hooks/pre', d => d?.hooks?.pre, '사전 hook'], ['/content/steps', d => d?.steps, '단계'], ['/content/hooks/post', d => d?.hooks?.post, '사후 hook'], ['/content/hooks/onFail', d => d?.hooks?.onFail, '실패 시 hook']],
 '1': [['/content/preActions', d => d?.preActions, '사전 동작'], ['/content/steps', d => d?.steps, '단계'], ['/content/assertions', d => d?.assertions, '단언'], ['/content/postActions', d => d?.postActions, '사후 동작']],
}

function summary(statement) {
 if (statement?.do) return `do ${statement.do}${statement.on ? ' · ' + statement.on : ''}`
 if (statement?.expect) return `expect ${statement.expect}${statement.on ? ' · ' + statement.on : ''}`
 if (statement?.assert) return `assert ${statement.assert}`
 const [name] = Object.keys(statement ?? {})
 return name ?? '빈 statement'
}

// statementLists gives each list's path, label and statements with their path
// and a one-line summary, for the statement tree.
export function statementLists(doc) {
 return (lists[doc?.schemaVersion === '1' ? '1' : '2']).map(([path, read, label]) => ({
  path, label, items: (read(doc) ?? []).map((s, i) => ({ path: `${path}/${i}`, summary: summary(s) })),
 }))
}

// statementOf is the statement an error or field path is inside, or '' when
// it is about the case settings (or a statement that no longer exists).
export function statementOf(doc, path) {
 for (const list of statementLists(doc)) for (const item of list.items) if (path === item.path || path.startsWith(item.path + '/')) return item.path
 return ''
}

// declarationDiff lists the JSON pointers where two declarations differ:
// a key only in the second is added, only in the first removed, and a value
// that differs below is named at the deepest place it differs.
export function declarationDiff(a, b, path = '') {
 const object = v => v !== null && typeof v === 'object'
 if (!object(a) || !object(b) || Array.isArray(a) !== Array.isArray(b)) return JSON.stringify(a) === JSON.stringify(b) ? [] : [{ path: path || '/', kind: 'changed' }]
 const out = []
 const keys = Array.isArray(a) ? [...Array(Math.max(a.length, b.length)).keys()].map(String) : [...new Set([...Object.keys(a), ...Object.keys(b)])].sort()
 for (const k of keys) {
  const at = `${path}/${k}`
  if (!(k in a)) out.push({ path: at, kind: 'added' })
  else if (!(k in b)) out.push({ path: at, kind: 'removed' })
  else out.push(...declarationDiff(a[k], b[k], at))
 }
 return out
}

// duplicateCase copies a case under an id no saved case uses yet.
export function duplicateCase(doc, takenIds = []) {
 const copy = structuredClone(doc)
 const base = `${doc.id}-copy`
 let id = base, n = 1
 while (takenIds.includes(id)) id = `${base}-${++n}`
 copy.id = id
 return copy
}

const resolveRef = (schema, root) => schema?.$ref ? resolveRef(root.$defs[schema.$ref.replace('#/$defs/', '')], root) : schema ?? {}

// schemaAt is the schema of the value at pointer (relative to the case)
// inside schema, following properties, array items and $defs references.
export function schemaAt(schema, root, pointer) {
 let at = resolveRef(schema, root)
 for (const key of pointer.split('/').slice(1)) {
  if (/^\d+$/.test(key) && at.items) at = resolveRef(at.items, root)
  else if (at.properties?.[key]) at = resolveRef(at.properties[key], root)
  else return {}
 }
 return at
}

// The top-level keys holding statement lists, edited through the tree.
const listKeys = doc => doc?.schemaVersion === '1' ? ['preActions', 'steps', 'assertions', 'postActions'] : ['hooks', 'steps']

// settingsOf is the case without its statement lists, with a schema that
// leaves them out too, so the settings form does not offer them again.
export function settingsOf(doc, schema) {
 const keys = listKeys(doc)
 const value = Object.fromEntries(Object.entries(doc ?? {}).filter(([k]) => !keys.includes(k)))
 const properties = Object.fromEntries(Object.entries(schema?.properties ?? {}).filter(([k]) => !keys.includes(k)))
 return { value, schema: { ...schema, properties, required: (schema?.required ?? []).filter(k => !keys.includes(k)) } }
}

// withSettings puts edited settings back around the case's statement lists.
export function withSettings(doc, settings) {
 const keep = Object.fromEntries(listKeys(doc).filter(k => k in doc).map(k => [k, doc[k]]))
 return { ...settings, ...keep }
}

const parts = pointer => pointer.replace(/^\/content/, '').split('/').slice(1)

// valueAt reads the value at a /content pointer.
export function valueAt(doc, pointer) {
 return parts(pointer).reduce((v, k) => v?.[k], doc)
}

// withValue returns a copy of doc with the value at pointer replaced; an
// undefined value removes it (from an array, its position closes).
export function withValue(doc, pointer, value) {
 const set = (node, [key, ...rest]) => {
  if (key === undefined) return value
  const copy = Array.isArray(node) ? [...node] : { ...node }
  const child = rest.length ? set(node?.[key], rest) : value
  if (child === undefined && !rest.length) { if (Array.isArray(copy)) copy.splice(Number(key), 1); else delete copy[key] }
  else copy[key] = child
  return copy
 }
 return set(doc, parts(pointer))
}
