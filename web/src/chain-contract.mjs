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

// draftChain names the chain of the declaration open in an editor: a legacy
// case's chain name, an inline preset's chain, or the chain of the preset it
// names or extends among the preset files loaded with it. A saved case with
// pinned preset revisions is read by caseChain in dsl-form.js instead.
export function draftChain(doc, presets = {}) {
 if (doc?.schemaVersion === '1') return doc.chain?.name ?? ''
 const preset = doc?.chainPreset
 const named = typeof preset === 'string' ? preset : preset?.extends
 if (typeof preset === 'object' && preset?.chain) return preset.chain
 return named ? presets[named]?.chain ?? '' : ''
}
