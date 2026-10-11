// latestSamples gives the last recorded value of each series one node has.
// A series with no value yet stays listed with null, never as zero.
export function latestSamples(metrics, nodeId) {
 return (metrics?.series ?? []).filter(s => s.nodeId === nodeId).map(s => {
  const last = [...(s.samples ?? [])].reverse().find(p => p.value !== null && p.value !== undefined)
  return { name: s.name, unit: s.unit ?? '', source: s.source, time: last?.time ?? null, value: last?.value ?? null }
 })
}

// nodeControls lists the node operations of the jobs contract that the server
// either offers for this node or refuses with a reason. busy is why the
// network takes no control right now; a node's own limit is shown first.
export function nodeControls(node, operations, busy) {
 const known = new Set([...(node?.supportedControls ?? []), ...Object.keys(node?.controlLimits ?? {})])
 return operations.filter(o => known.has(o.id)).map(o => {
  const limit = node.controlLimits?.[o.id] ?? ''
  const offered = (node.supportedControls ?? []).includes(o.id)
  return { id: o.id, label: o.label, enabled: offered && !busy, reason: limit || (offered && busy ? busy : '') }
 })
}
