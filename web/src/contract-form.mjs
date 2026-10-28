// @ts-check
// Helpers for the laid-out deployment form. Field paths use the form's
// notation (content.pool.hosts[1].addr); the contract supplies the words.

/** @typedef {{type?: string, title?: string, properties?: Record<string, Field>, items?: Field, additionalProperties?: Field}} Field */

/**
 * @param {string} path
 * @returns {(string|number)[]} keys after "content"; array positions as numbers
 */
function keysOf(path) {
 /** @type {(string|number)[]} */
 const out = []
 for (const part of path.replace(/^content\.?/, '').split('.')) {
  if (!part) continue
  const m = part.match(/^([^[]*)((?:\[\d+\])*)$/)
  if (!m) { out.push(part); continue }
  if (m[1]) out.push(m[1])
  for (const i of m[2].matchAll(/\[(\d+)\]/g)) out.push(Number(i[1]))
 }
 return out
}

/**
 * fieldTitle names a field by the titles on its way down, so the error
 * summary reads "풀 · 서버 2 · 주소" rather than a path. A path outside the
 * contract stays as written; a top-level request field keeps the name the
 * editor gives it.
 * @param {Field|undefined} schema
 * @param {string} path
 */
export function fieldTitle(schema, path) {
 if (path === 'name') return '이름'
 if (!path.startsWith('content')) return path
 /** @type {string[]} */
 const words = []
 let spec = schema
 for (const key of keysOf(path)) {
  if (typeof key === 'number') {
   if (spec?.type !== 'array') return path
   words[words.length - 1] = `${words[words.length - 1]} ${key + 1}`
   spec = spec.items
   continue
  }
  const next = spec?.properties?.[key]
  if (next) { words.push(next.title || key); spec = next; continue }
  if (spec?.additionalProperties) { words.push(key); spec = spec.additionalProperties; continue }
  return path
 }
 return words.join(' · ')
}

/**
 * remainder is the schema without the fields a laid-out form places itself,
 * given as dotted paths below the content; what is left (and whatever the
 * contract gains later) goes to the general fields under "추가 설정".
 * @param {Field} schema
 * @param {string[]} placed
 * @returns {Field}
 */
export function remainder(schema, placed) {
 /** @type {Field} */
 const out = { ...schema, properties: { ...schema.properties } }
 for (const path of placed) {
  const [head, ...rest] = path.split('.')
  if (!out.properties?.[head]) continue
  if (rest.length === 0) { delete out.properties[head]; continue }
  const child = remainder(out.properties[head], [rest.join('.')])
  if (Object.keys(child.properties ?? {}).length === 0 && !child.additionalProperties) delete out.properties[head]
  else out.properties[head] = child
 }
 return out
}

/**
 * changedPaths lists the leaf fields whose values differ between two
 * contents, in form notation, so a past revision can mark what it changed.
 * @param {unknown} a
 * @param {unknown} b
 * @param {string} [path]
 * @returns {Set<string>}
 */
export function changedPaths(a, b, path = 'content') {
 const out = new Set()
 const object = (/** @type {unknown} */ v) => v !== null && typeof v === 'object'
 if (Array.isArray(a) || Array.isArray(b)) {
  const left = Array.isArray(a) ? a : [], right = Array.isArray(b) ? b : []
  for (let i = 0; i < Math.max(left.length, right.length); i++) for (const p of changedPaths(left[i], right[i], `${path}[${i}]`)) out.add(p)
  return out
 }
 if (object(a) || object(b)) {
  const left = object(a) ? /** @type {Record<string, unknown>} */ (a) : {}, right = object(b) ? /** @type {Record<string, unknown>} */ (b) : {}
  for (const key of new Set([...Object.keys(left), ...Object.keys(right)])) for (const p of changedPaths(left[key], right[key], `${path}.${key}`)) out.add(p)
  return out
 }
 if (a !== b) out.add(path)
 return out
}
