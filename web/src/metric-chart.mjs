import {gapText, formatTime} from './labels.mjs'

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
 return {name, unit: series[0]?.unit ?? '', source: series[0]?.source ?? '', nodes, yMin, yMax, tMin, tMax, width, height}
}

const toX = (model, t) => (model.tMax === model.tMin ? model.width / 2 : ((t - model.tMin) / (model.tMax - model.tMin)) * model.width)

// timeTicks spaces count labels evenly over the chart's time span, both ends
// included; a single instant gets one tick.
export function timeTicks(model, count) {
 if (!(model.tMax > model.tMin)) return [{x: model.width / 2, time: new Date(model.tMin).toISOString()}]
 return Array.from({length: count}, (_, i) => {
  const t = model.tMin + ((model.tMax - model.tMin) * i) / (count - 1)
  return {x: toX(model, t), time: new Date(t).toISOString()}
 })
}

// gapBands shades each collection gap that overlaps the chart, clipped to it.
export function gapBands(model, coverage) {
 return (coverage?.gaps ?? []).flatMap(g => {
  const from = Math.max(Date.parse(g.from), model.tMin), to = Math.min(Date.parse(g.to), model.tMax)
  if (!(to > from)) return []
  const x = toX(model, from)
  return [{x, width: toX(model, to) - x, reason: g.reason}]
 })
}

// latestValues is each node's last drawn value and the spread between the
// highest and lowest of them, which is how far the slowest node lags.
export function latestValues(model) {
 const values = model.nodes.map(n => ({node: n.node, value: n.points[n.points.length - 1].value}))
 const numbers = values.map(v => v.value).filter(v => typeof v === 'number')
 return {values, spread: numbers.length > 1 ? Math.max(...numbers) - Math.min(...numbers) : null}
}

// nearestPoints is each node's drawn point closest to time. Beyond maxMs (when
// given) the node had no sample near that moment and its value is null, so a
// gap is never read as the last value before it.
export function nearestPoints(model, time, maxMs = Infinity) {
 const at = Date.parse(time)
 return model.nodes.flatMap(n => {
  let best = null
  for (const p of n.points) if (!best || Math.abs(Date.parse(p.time) - at) < Math.abs(Date.parse(best.time) - at)) best = p
  if (!best) return []
  return Math.abs(Date.parse(best.time) - at) > maxMs ? [{node: n.node, time: best.time, value: null, x: null, y: null}] : [{node: n.node, ...best}]
 })
}

// pointerTime maps an x position in chart coordinates to a time on the axis.
export function pointerTime(model, x) {
 const share = Math.max(0, Math.min(1, x / model.width))
 return new Date(model.tMin + (model.tMax - model.tMin) * share).toISOString()
}

// logWindow is the archived log range linked to a selected chart time.
export function logWindow(time, seconds) {
 const at = Date.parse(time)
 return {from: new Date(at - seconds * 1000).toISOString(), to: new Date(at + seconds * 1000).toISOString()}
}

export function gapSummary(coverage) {
 return (coverage?.gaps ?? []).map(g => {
  const {text, code} = gapText(g.reason)
  return `${text}${text.endsWith(code) ? '' : ` · ${code}`} (${formatTime(g.from)} – ${formatTime(g.to)})`
 })
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

// The three gauges that place a failure: chain progress, connectivity and
// queued work. Each lists the rpc spelling before the metrics-endpoint one.
const FAILURE_METRICS = [
 {label: '블록 높이', short: '블록', names: ['block_height', 'chain_head_block']},
 {label: '피어 수', short: '피어', names: ['peer_count', 'p2p_peers']},
 {label: '대기 트랜잭션', short: '대기 tx', names: ['txpool_pending']},
]

// failureMetrics picks, for one node, the first collected spelling of each
// failure gauge; a gauge with no series keeps its first name and no series.
export function failureMetrics(metrics, node) {
 const all = metrics?.series ?? []
 return FAILURE_METRICS.map(({label, short, names}) => {
  const name = names.find(n => all.some(s => s.name === n && s.nodeId === node)) ?? names[0]
  return {label, short, name, series: all.filter(s => s.name === name && s.nodeId === node)}
 })
}

// timeX is a moment's x on the chart, or null when it falls outside it.
export function timeX(model, time) {
 const t = Date.parse(time ?? '')
 if (!Number.isFinite(t) || t < model.tMin || t > model.tMax) return null
 return toX(model, t)
}
