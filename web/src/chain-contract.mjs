// topologyChoices reads a chain preset contract's node roles, the count fields
// with their titles and defaults, and the sync mode choices, so screens never
// keep their own copy of them. A missing contract offers no choices.
export function topologyChoices(contract) {
 const forms = contract?.$defs?.envSpec?.properties?.topology?.oneOf ?? []
 const countForm = forms.find(f => f?.properties && !f.properties.nodes)?.properties ?? {}
 const tableForm = forms.find(f => f?.properties?.nodes)?.properties.nodes
 const roles = [...(tableForm?.items?.properties?.role?.enum ?? [])]
 const counts = roles.filter(r => countForm[r]).map(r => ({ id: r, title: countForm[r].title ?? r, minimum: countForm[r].minimum ?? 0, default: countForm[r].default ?? 0 }))
 const defaults = Object.fromEntries(counts.map(f => [f.id, f.default]))
 const s = countForm.syncMode
 const sync = s?.enum ? { title: s.title ?? 'syncMode', options: [...s.enum], default: s.default ?? '' } : null
 return { roles, defaults, counts: counts.map(({ default: _, ...f }) => f), sync }
}

// caseChain names the chain a test case runs on: a legacy case's chain name,
// an inline preset's chain, or the chain of the named preset it loaded.
export function caseChain(doc, presets = {}) {
 if (doc?.schemaVersion === '1') return doc.chain?.name ?? ''
 const preset = doc?.chainPreset
 if (typeof preset === 'object' && preset) return preset.chain ?? ''
 return typeof preset === 'string' ? presets[preset]?.chain ?? '' : ''
}
