// A draft remembers the engine contract it was edited against, so the server
// can refuse it after the contract changes instead of judging it by rules the
// editor never showed.
export function draftMeta(contract) {
 return { contractVersion: contract?.['x-contractVersion'], contractDigest: contract?.['x-contractDigest'] }
}

// contractChanged recognises the server's refusal of a draft whose contract
// has changed, by the reason it carries rather than by its wording.
export function contractChanged(error) {
 return error?.status === 409 && error?.details?.reason === 'contract_changed'
}

// changedDefinitions names the parts of a contract that differ: each $defs
// entry, and each other top-level key except the digest itself.
export function changedDefinitions(before, after) {
 const names = new Set()
 const same = (a, b) => JSON.stringify(a) === JSON.stringify(b)
 for (const key of new Set([...Object.keys(before?.$defs ?? {}), ...Object.keys(after?.$defs ?? {})])) {
  if (!same(before?.$defs?.[key], after?.$defs?.[key])) names.add('$defs.' + key)
 }
 for (const key of new Set([...Object.keys(before ?? {}), ...Object.keys(after ?? {})])) {
  if (key !== '$defs' && key !== 'x-contractDigest' && !same(before?.[key], after?.[key])) names.add(key)
 }
 return [...names].sort()
}
