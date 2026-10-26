// Korean words for what the engine reports as codes: coverage gap reasons and
// job, test and node states, plus the one time format every screen uses. The
// engine code stays available beside the word so it can still be searched.

const GAP_REASONS = {
 not_collected: '수집 시작 전 구간',
 collector_stopped: '대시보드가 꺼져 있던 구간',
 collector_stale: '수집이 멈춘 구간',
 collector_error: '수집 오류',
 rpc_unavailable: 'RPC 응답 없음',
 rpc_disabled: 'RPC가 꺼져 있음',
 metrics_unavailable: 'metrics 응답 없음',
 metrics_disabled: 'metrics가 꺼져 있음',
 log_unavailable: '로그 파일을 읽지 못함',
 log_rotated: '로그 파일이 교체되어 처음부터 다시 읽음',
 log_truncated: '로그 파일이 줄어들어 처음부터 다시 읽음',
 log_line_oversized: '1 MiB를 넘는 줄을 건너뜀',
 remote_log_collection_unavailable: '원격 로그 수집 작업이 없던 구간',
 node_stopped: '노드가 멈춰 있던 구간',
}

// gapText reads a coverage gap (api-model.js Gap) into Korean, keeping the
// engine code beside it. A gap about one node names that node; a code that
// carries a value (the metric not exposed, the deleted run) names it too.
export function gapText(gap) {
 const code = gap?.code ?? ''
 const nodeId = gap?.scope === 'node' ? gap?.nodeId ?? '' : ''
 let words = GAP_REASONS[code] ?? code
 if (code === 'not_exposed') words = `${gap.metric ?? '?'} 지표를 노드가 내놓지 않음`
 if (code === 'deleted_by_admin') words = `관리자가 지운 실행 ${gap.runId ?? '?'}의 구간`
 if (code === 'corrupt_record') {
  // The count lives only in the older joined text, "archive corrupt_record <n>".
  const count = String(gap.reason ?? '').match(/corrupt_record (\d+)$/)?.[1]
  words = count ? `손상된 보관 기록 ${count}건` : '손상된 보관 기록'
 }
 return { text: nodeId ? `${nodeId} · ${words}` : words, code, nodeId }
}

const STATES = {
 succeeded: '성공', success: '성공', pass: '통과', passed: '통과',
 failed: '실패', fail: '실패', error: '오류',
 blocked: '막힘', skip: '건너뜀', skipped: '건너뜀',
 accepted: '접수됨', running: '실행 중', pending: '대기', cancelling: '취소 중', cancelled: '취소됨',
 interrupted: '중단됨', stale: '오래됨', unknown: '알 수 없음',
 stopped: '정지', started: '시작됨', retained: '유지', absent: '없음',
}

// stateLabel is the Korean word for a state, or '' when there is none, in
// which case the engine's own word is shown alone.
export function stateLabel(state) {
 return STATES[state] ?? ''
}

const pad = n => String(n).padStart(2, '0')

export function formatTime(value) {
 if (!value) return '—'
 const d = new Date(value)
 if (Number.isNaN(d.getTime())) return '—'
 return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
