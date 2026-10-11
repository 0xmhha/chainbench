import { snapshotState, applyJobChange } from './job-observation.mjs'

// jobStream follows the server's committed job state: it restores a snapshot,
// then applies ordered changes from /api/v1/events, and restores again on a
// gap, a restart or a dropped connection. start() returns the stop function.
export function jobStream() {
 const s = $state({ snapshot: null, connection: '복원 중', error: '', gap: '', drops: null, busDrops: null, lastObserved: null })
 let active = false, epoch = 0, stream = null, request = null, retry = null
 function disconnect() { stream?.close(); stream = null; clearTimeout(retry) }
 async function restore(reason = '') {
  if (!active) return
  if (reason) s.gap = reason
  disconnect(); request?.abort(); request = new AbortController()
  const current = ++epoch
  s.connection = '복원 중'
  try {
   const response = await fetch('/api/v1/snapshot', { signal: request.signal, cache: 'no-store' })
   if (!response.ok) throw new Error(`실행 상태 조회 실패 (${response.status})`)
   const value = await response.json()
   if (!active || current !== epoch) return
   s.snapshot = snapshotState(s.snapshot, value); s.drops = value.jobDeliveryDropsTotal; s.busDrops = value.busDeliveryDropsTotal; s.lastObserved = value.observedAt; s.error = ''
   stream = new EventSource('/api/v1/events?cursor=' + encodeURIComponent(s.snapshot.cursor))
   stream.onopen = () => { if (active && current === epoch) s.connection = '연결됨' }
   stream.onerror = () => {
    if (!active || current !== epoch) return
    disconnect(); s.connection = '연결 끊김'; s.gap = '연결이 끊긴 동안의 이벤트 전체 수신을 보장하지 않습니다. 최신 실행 상태를 다시 확인합니다.'
    retry = setTimeout(() => restore(), 2000)
   }
   for (const type of ['job_changed', 'version_changed', 'resync_required', 'observation']) stream.addEventListener(type, message => {
    if (!active || current !== epoch) return
    try {
     const event = JSON.parse(message.data), result = applyJobChange(s.snapshot, { ...event, type })
     if (result.resync) { restore(event.missingTo ? `변경 ${event.missingTo - event.missingFrom + 1}건(버전 ${event.missingFrom}–${event.missingTo})을 다시 받을 수 없습니다. 최신 스냅샷으로 복원합니다.` : '변경 이력 누락 또는 서버 재시작을 확인했습니다. 최신 스냅샷으로 복원합니다.'); return }
     s.snapshot = result.state; s.lastObserved = event.observedAt ?? s.lastObserved
     if (event.jobDeliveryDropsTotal !== undefined) s.drops = event.jobDeliveryDropsTotal
    } catch { restore('수신한 변경 내용을 해석할 수 없어 최신 스냅샷을 다시 확인합니다.') }
   })
  } catch (failure) {
   if (!active || current !== epoch || failure.name === 'AbortError') return
   s.error = failure.message; s.connection = '연결 끊김'; retry = setTimeout(() => restore(), 2000)
  }
 }
 return {
  state: s,
  refresh: () => restore(),
  start() { active = true; restore(); return () => { active = false; epoch++; disconnect(); request?.abort() } },
 }
}
