// Chart geometry for archived node metrics. Missing values stay missing: a
// sample with a null value widens the time axis but is never drawn as zero,
// and the line stops there. It also stops where two samples are further apart
// than twice the sampling interval (or the downsampling bucket), so a stretch
// nothing was collected for is never bridged.
export function chartModel(metrics, name, {width, height}) {
 const series = (metrics?.series ?? []).filter(s => s.name === name).sort((a, b) => a.nodeId.localeCompare(b.nodeId))
 const times = series.flatMap(s => s.samples.map(p => Date.parse(p.time)))
 const values = series.flatMap(s => s.samples.filter(p => p.value !== null).map(p => p.value))
 const tMin = Math.min(...times), tMax = Math.max(...times)
 const yMin = Math.min(...values), yMax = Math.max(...values)
 const x = t => tMax === tMin ? width / 2 : (t - tMin) / (tMax - tMin) * width
 const y = v => yMax === yMin ? height / 2 : height - (v - yMin) / (yMax - yMin) * height
 const step = metrics?.coverage?.bucketSeconds || metrics?.coverage?.sampleIntervalSeconds || 0
 const nodes = series.map(s => {
  const segments = []
  let current = null, last = null
  for (const p of s.samples) {
   const t = Date.parse(p.time)
   if (p.value === null || p.value === undefined) { current = null; continue }
   if (current && last !== null && step > 0 && t - last > 2 * step * 1000) current = null
   if (!current) { current = []; segments.push(current) }
   current.push({time: p.time, value: p.value, x: x(t), y: y(p.value)})
   last = t
  }
  const points = segments.flat()
  return {node: s.nodeId, points, segments, path: segments.map(seg => seg.map((p, i) => `${i ? 'L' : 'M'}${p.x},${p.y}`).join(' ')).join(' ')}
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

// metricNames lists every metric the server collected for the range, with the
// unit and source of its first series, so the screen charts all of them.
export function metricNames(metrics) {
 const seen = new Map()
 for (const s of metrics?.series ?? []) if (!seen.has(s.name)) seen.set(s.name, {name: s.name, unit: s.unit ?? '', source: s.source})
 return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name))
}

// timeRange turns a range choice into the query window: preset minutes
// ending now, or a custom from/to. A reversed or unreadable range is null.
export function timeRange(choice, now = Date.now()) {
 if (choice?.preset > 0) return {from: new Date(now - choice.preset * 60000).toISOString(), to: new Date(now).toISOString()}
 const from = Date.parse(choice?.from), to = Date.parse(choice?.to)
 if (!Number.isFinite(from) || !Number.isFinite(to) || from >= to) return null
 return {from: new Date(from).toISOString(), to: new Date(to).toISOString()}
}
