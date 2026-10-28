// Pure helpers behind the recorder strip on the job detail: where the two
// cursors start, what every gauge read at each, and which readings stand out
// against the other nodes of the network.
import { failureMetrics } from './metric-chart.mjs'

const at = value => Date.parse(value ?? '')

// cursorDefaults puts A on the first warning before the first error and B on
// that error; without them the test's own start and end stand in. A moment
// from the URL (at) places B instead while it falls inside the window.
export function cursorDefaults({ events, start, end, at: moment = '', from = start, to = end }) {
 const sorted = [...(events ?? [])].sort((x, y) => at(x.time) - at(y.time))
 const firstError = sorted.find(e => e.level === 'ERROR' || e.level === 'CRIT')
 const first = sorted[0]
 // When the first event is the first error, A opens the window instead, so
 // the measure still spans the lead-up to it.
 const a = first && first !== firstError ? first.time : start
 const kept = at(moment) >= at(from) && at(moment) <= at(to)
 return { a, b: kept ? moment : firstError?.time ?? end }
}

const tolerance = metrics => 2 * (metrics?.coverage?.bucketSeconds || metrics?.coverage?.sampleIntervalSeconds || 5) * 1000

// valueAt is node's sample nearest to time. A null sample (a recorded gap) or
// no sample within two collection intervals reads as no value, never as the
// last value before the gap.
function valueAt(metrics, name, node, time) {
 const samples = (metrics?.series ?? []).find(s => s.name === name && s.nodeId === node)?.samples ?? []
 let best = null
 for (const p of samples) if (!best || Math.abs(at(p.time) - at(time)) < Math.abs(at(best.time) - at(time))) best = p
 if (!best || Math.abs(at(best.time) - at(time)) > tolerance(metrics)) return null
 return typeof best.value === 'number' ? best.value : null
}

// Which way a gauge goes wrong: a lagging block height, fewer peers, or more
// queued transactions than every other node at the same moment.
const WRONG = [(v, o) => v < Math.min(...o), (v, o) => v < Math.min(...o), (v, o) => v > Math.max(...o)]

// measureBetween reads each failure gauge of node at cursors a and b, plus the
// highest block any other node reached. A gauge is abnormal when its value at
// b goes wrong against every other node at b (the same rule that rings a
// reading), so red marks a fault rather than any change.
export function measureBetween(metrics, node, a, b) {
 const nodesOf = name => [...new Set((metrics?.series ?? []).filter(s => s.name === name && s.nodeId !== node).map(s => s.nodeId))]
 const gauges = failureMetrics(metrics, node).map((g, i) => {
  const va = valueAt(metrics, g.name, node, a), vb = valueAt(metrics, g.name, node, b)
  const others = nodesOf(g.name).map(n => valueAt(metrics, g.name, n, b)).filter(v => v !== null)
  return { label: g.label, name: g.name, a: va, b: vb, changed: va !== null && vb !== null && va !== vb, abnormal: vb !== null && others.length > 0 && WRONG[i](vb, others) }
 })
 const block = failureMetrics(metrics, node)[0].name
 const top = time => { const v = nodesOf(block).map(n => valueAt(metrics, block, n, time)).filter(x => x !== null); return v.length ? Math.max(...v) : null }
 return { seconds: Math.round((at(b) - at(a)) / 100) / 10, gauges, others: { a: top(a), b: top(b) } }
}

// readingRows is one row per sample time of node, with each failure gauge's
// value (null in a gap) and whether it stands out against the other nodes.
export function readingRows(metrics, node) {
 const gauges = failureMetrics(metrics, node)
 const times = [...new Set(gauges.flatMap(g => g.series.flatMap(s => s.samples.map(p => p.time))))].sort((x, y) => at(x) - at(y))
 return times.map(time => ({
  time,
  cells: gauges.map((g, i) => {
   const value = g.series[0]?.samples.find(p => p.time === time)?.value ?? null
   const others = (metrics?.series ?? []).filter(s => s.name === g.name && s.nodeId !== node).map(s => s.samples.find(p => p.time === time)?.value).filter(v => typeof v === 'number')
   return { value, abnormal: value !== null && others.length > 0 && WRONG[i](value, others) }
  })
 }))
}

// stripX is a time's position as a percentage of the from–to window, or null
// outside it; stripTime is the reverse.
export function stripX(time, from, to) {
 const t = at(time), s = at(from), e = at(to)
 if (!Number.isFinite(t) || t < s || t > e || !(e > s)) return null
 return Math.round(((t - s) / (e - s)) * 10000) / 100
}

export function stripTime(percent, from, to) {
 const s = at(from), e = at(to)
 return new Date(s + ((e - s) * Math.max(0, Math.min(100, percent))) / 100).toISOString()
}

// tracePath draws one gauge across the strip window in 0–100 coordinates (y
// grows downward). The line breaks at a null sample and wherever two samples
// are more than two collection intervals apart, so a gap is never bridged.
export function tracePath(samples, from, to, stepSeconds) {
 const points = (samples ?? []).filter(p => stripX(p.time, from, to) !== null || p.value === null)
 const values = points.map(p => p.value).filter(v => typeof v === 'number')
 if (!values.length) return { segments: [], min: null, max: null }
 const min = Math.min(...values), max = Math.max(...values)
 const y = v => (max === min ? 50 : Math.round((100 - ((v - min) / (max - min)) * 100) * 100) / 100)
 const segments = []
 let current = [], last = null
 for (const p of points) {
  const x = stripX(p.time, from, to)
  if (typeof p.value !== 'number' || x === null || (last !== null && at(p.time) - last > 2 * stepSeconds * 1000)) {
   if (current.length) segments.push(current)
   current = []
   if (typeof p.value !== 'number' || x === null) { last = null; continue }
  }
  current.push(`${current.length ? 'L' : 'M'}${x},${y(p.value)}`)
  last = at(p.time)
 }
 if (current.length) segments.push(current)
 return { segments: segments.map(s => s.join(' ')), min, max }
}
