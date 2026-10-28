import { formatTime } from './labels.mjs'

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

// stateSource says where a node's shown state came from: a check the reader
// just made (fresh), the collector's latest reading with its time, or nothing
// yet. Only a fresh check opens controls, so the collector case says so.
export function stateSource(node, fresh) {
 const pid = node?.observedPid ? String(node.observedPid) : '없음'
 if (fresh) return { note: `확인 ${formatTime(node?.observedAt)}`, pid }
 // A running reading carries no reason, so only not_observed (or an unknown
 // state with no reason, from a server before readings) means nothing seen.
 if (node?.observationReason === 'not_observed' || (node?.state === 'unknown' && !node?.observationReason)) return { note: '아직 관측하지 않음 · 기록된 PID는 실제 가동 증거가 아닙니다', pid: '확인 안 함' }
 return { note: `수집기 관측 ${formatTime(node.observedAt)} · 제어하려면 노드 상태를 확인하세요`, pid: `${pid} (수집기)` }
}

// Live state is stale once the job stream drops or has heard nothing for this
// long; the server sends an observation every second while it is connected.
const staleAfterMs = 5000

// liveStale reports whether what the page shows may no longer be true.
export function liveStale(live, now = Date.now()) {
 if (live.connection !== '연결됨' || !live.lastObserved) return true
 return now - Date.parse(live.lastObserved) > staleAfterMs
}

// shownState is the state a node row shows: its own while live state is
// fresh, and stale while it is not, keeping what the node last was.
export function shownState(state, stale) {
 return stale ? { state: 'stale', last: state } : { state, last: null }
}

// feedPhase says what a screen built on the job stream can show: nothing yet
// (loading), nothing because the server cannot be reached, or the snapshot.
// An empty list is only "empty" once the phase is ready.
export function feedPhase(live) {
 if (live.snapshot) return 'ready'
 return live.error ? 'unreachable' : 'loading'
}
