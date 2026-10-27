// Positions refer to committed jobs. They never assert that a cached PID is live.
function position(value) {
 if (!Number.isSafeInteger(value?.version) || value.version < 0 || typeof value.cursor !== 'string') throw new Error('관측 버전을 해석할 수 없습니다.')
 const match = /^([a-f0-9]{32}):(0|[1-9][0-9]*)$/.exec(value.cursor)
 if (!match || Number(match[2]) !== value.version) throw new Error('관측 커서와 버전이 일치하지 않습니다.')
 return match[1]
}
export function snapshotState(current, snapshot) {
 const instance = position(snapshot)
 if (!Array.isArray(snapshot.jobs) || !Array.isArray(snapshot.networks) || !Number.isFinite(Date.parse(snapshot.observedAt))) throw new Error('실행 상태 스냅샷이 올바르지 않습니다.')
 if (current && position(current) === instance && current.version > snapshot.version) return current
 return snapshot
}
export function applyJobChange(state, event) {
 const resync = reason => ({state,resync:true,reason})
 if (!state) return resync('initial_snapshot_required')
 let instance
 try { instance = position(event) } catch { return resync('invalid_update') }
 if (instance !== position(state)) return resync('server_restarted')
 if (event.type === 'resync_required') return resync(event.reason || 'resync_required')
 if (event.type === 'observation') return event.version > state.version ? resync('version_gap') : {state,resync:false}
 if (!['job_changed','version_changed'].includes(event.type)) return resync('invalid_update')
 if (event.version <= state.version) return {state,resync:false}
 if (event.version !== state.version+1) return resync('version_gap')
 let jobs = state.jobs
 if (event.type === 'job_changed') {
  if (!event.job || typeof event.job.id !== 'string' || typeof event.job.state !== 'string') return resync('invalid_update')
  jobs = [event.job,...jobs.filter(job => job.id !== event.job.id)]
 }
 return {state:{...state,version:event.version,cursor:event.cursor,jobs},resync:false}
}
// applyNodeReading puts a collector reading (a node event) on the node it
// names. It is not a committed job change, so the version stays; a node the
// snapshot does not hold is left for the next snapshot rather than invented.
export function applyNodeReading(state, event) {
 const reading = event?.node
 const network = state?.networks?.find(n => n.id === event?.networkId)
 if (!reading || !network?.nodes?.some(n => n.id === reading.id)) return state
 const nodes = network.nodes.map(n => n.id === reading.id ? {...n, state: reading.state, observationReason: reading.observationReason, observedPid: reading.observedPid, observedAt: reading.observedAt} : n)
 return {...state, networks: state.networks.map(n => n === network ? {...n, nodes} : n)}
}
// appendEngineEvent keeps what CLI runs published, newest first, at most
// limit of them; the list is what this connection received, not a history.
export function appendEngineEvent(list, event, limit = 200) {
 return [event, ...(list ?? [])].slice(0, limit)
}
