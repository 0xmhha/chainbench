// Chart geometry for archived node metrics. Missing values stay missing: a
// sample with a null value widens the time axis but is never drawn as zero.
export function chartModel(metrics, name, {width, height}) {
 const series = (metrics?.series ?? []).filter(s => s.name === name).sort((a, b) => a.nodeId.localeCompare(b.nodeId))
 const times = series.flatMap(s => s.samples.map(p => Date.parse(p.time)))
 const values = series.flatMap(s => s.samples.filter(p => p.value !== null).map(p => p.value))
 const tMin = Math.min(...times), tMax = Math.max(...times)
 const yMin = Math.min(...values), yMax = Math.max(...values)
 const x = t => tMax === tMin ? width / 2 : (t - tMin) / (tMax - tMin) * width
 const y = v => yMax === yMin ? height / 2 : height - (v - yMin) / (yMax - yMin) * height
 const nodes = series.map(s => {
  const points = s.samples.filter(p => p.value !== null).map(p => ({time: p.time, value: p.value, x: x(Date.parse(p.time)), y: y(p.value)}))
  return {node: s.nodeId, points, path: points.map((p, i) => `${i ? 'L' : 'M'}${p.x},${p.y}`).join(' ')}
 }).filter(n => n.points.length > 0)
 return {name, unit: series[0]?.unit ?? '', source: series[0]?.source ?? '', nodes, yMin, yMax, tMin, tMax}
}

// logWindow is the archived log range linked to a selected chart time.
export function logWindow(time, seconds) {
 const at = Date.parse(time)
 return {from: new Date(at - seconds * 1000).toISOString(), to: new Date(at + seconds * 1000).toISOString()}
}

export function gapSummary(coverage) {
 return (coverage?.gaps ?? []).map(g => `${g.reason} (${g.from} – ${g.to})`)
}
