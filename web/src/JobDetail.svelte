<script>
  import StatusBadge from './StatusBadge.svelte'
  import { onMount } from 'svelte'
  import { webApi } from './web-api.mjs'
  import { jobStream } from './job-stream.svelte.js'
  import { jobSummary } from './job-view.mjs'
  import { follow } from './nav.mjs'
  import JobCard from './JobCard.svelte'
  import LogViewer from './LogViewer.svelte'
  import RunTimeline from './RunTimeline.svelte'
  import { phaseSegments } from './run-timeline.mjs'
  import { parseHistoryJSON } from './history-json.js'
  let { jobId, webSession } = $props()
  const api = $derived(webApi(webSession))
  const feed = jobStream(), live = feed.state
  let fetched = $state(null), error = $state('')
  // The live feed carries every committed change; the first read covers the
  // moment before it connects and names a job that does not exist.
  const job = $derived(live.snapshot?.jobs.find(j => j.id === jobId) ?? fetched)
  const summary = $derived(job ? jobSummary(job) : null)
  async function load() { error = ''; try { fetched = await api('jobs/' + jobId) } catch (e) { error = e.message } }
  // The failed phase is read against the logs of the network the job worked
  // on, and every test run the job produced is shown as its own timeline.
  const PAD_MS = 5000
  let network = $state(null), runs = $state([])
  const phases = $derived(phaseSegments(job?.phases))
  // Untimed phases keep a small fixed slot; timed ones share the rest.
  const UNTIMED_SLOT = 4
  const phaseWidth = p => {
    const untimed = phases.filter(x => !x.timed).length
    return p.timed ? (p.share * (100 - untimed * UNTIMED_SLOT)) / 100 : UNTIMED_SLOT
  }
  const phaseSpan = $derived.by(() => {
    const p = summary?.failedPhase
    const s = Date.parse(p?.startedAt ?? ''), e = Date.parse(p?.finishedAt ?? '')
    return Number.isFinite(s) && Number.isFinite(e) ? { from: new Date(s - PAD_MS).toISOString(), to: new Date(e + PAD_MS).toISOString() } : null
  })
  const networkNodes = $derived((network?.nodes ?? []).map(n => n.id))
  $effect(() => {
    const workspace = job?.workspaceId
    if (!workspace || !webSession) return
    api('networks').then(page => { network = page.items.find(n => n.workspaceId === workspace) ?? null }).catch(() => { network = null })
  })
  $effect(() => {
    const ids = job?.runIds ?? []
    if (!ids.length || !webSession) { runs = []; return }
    // Raw text keeps large integers exact (parseHistoryJSON), as History does.
    const read = id => fetch('/api/v1/history/' + encodeURIComponent(id), { cache: 'no-store' }).then(r => r.ok ? r.text() : Promise.reject(new Error(String(r.status)))).then(parseHistoryJSON)
    Promise.all(ids.map(id => read(id).catch(() => null)))
      .then(found => { runs = found.filter(Boolean) })
  })
  $effect(() => { void jobId; if (webSession) load() })
  onMount(() => feed.start())
</script>

<section class="panel" aria-label="작업 상세">
  <p><a href="/chains" onclick={e => follow(e, '/chains')}>← 체인 화면의 작업 목록</a></p>
  <h2>작업 상세</h2>
  <p class="notice" data-testid="job-continues">이 작업은 서버에서 실행됩니다. 브라우저를 닫거나 다른 화면으로 가도 계속되며, 다시 열면 기록된 상태를 보여 줍니다.</p>
  <p class="muted" data-testid="job-stream">실시간 연결: {live.connection}{live.gap ? ' · ' + live.gap : ''}</p>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if job}
    <dl class="summary">
      <dt>작업</dt><dd>{job.operation} · <code>{job.id}</code></dd>
      <dt>상태</dt><dd data-testid="job-state"><StatusBadge state={job.state} /></dd>
      <dt>실패한 단계</dt><dd data-testid="job-failed-phase">{summary.failedPhase ? `${summary.failedPhase.name}${summary.failedPhase.message ? ' · ' + summary.failedPhase.message : ''}` : '없음'}</dd>
      {#if summary.failure}<dt>{job.state === 'cancelled' ? '중단 메시지' : '실패 이유'}</dt><dd class:error={job.state !== 'cancelled'}>{summary.failure}</dd>{/if}
      <dt>이미 반영된 변경</dt><dd data-testid="job-effects">{summary.effects.length ? `${summary.effects.length}건 기록됨 · 아래 '수행된 변경'에서 봅니다` : '기록 없음'}</dd>
      <dt>확인할 자원</dt><dd data-testid="job-unresolved">{summary.unresolved.length ? `${summary.unresolved.length}개` : '없음'}</dd>
      <dt>노드 처리</dt><dd data-testid="job-disposition">{job.nodeDisposition}{summary.needsCheck ? ' · 실제 상태 확인이 필요합니다' : ''}</dd>
      {#if job.cancelReason}<dt>취소</dt><dd>{job.cancelReason}{job.cancelMessage ? ' · ' + job.cancelMessage : ''}</dd>{/if}
    </dl>
    {#if phases.length}
      <div class="phases" role="img" aria-label={phases.map(p => `${p.name} ${p.state}`).join(', ')}>
        {#each phases as p}<span class="phase {p.state}" style:width="{phaseWidth(p)}%" title="{p.name} · {p.state}{p.durationMs !== null ? ` · ${(p.durationMs / 1000).toFixed(1)}초` : ''}"></span>{/each}
      </div>
      <ol class="phase-names">{#each phases as p}<li class={p.state}>{p.name}{#if p.durationMs !== null} <small>{(p.durationMs / 1000).toFixed(1)}초</small>{/if}</li>{/each}</ol>
    {/if}
    {#if summary.failedPhase && network && networkNodes.length}
      <section class="cause" aria-label="실패한 단계 시점 로그">
        {#if !phaseSpan}<p class="notice">실패한 단계에 시각이 없어 최근 2시간의 로그를 엽니다.</p>{/if}
        <LogViewer networkId={network.id} nodes={networkNodes} node={networkNodes.length > 1 ? '*' : networkNodes[0]} from={phaseSpan?.from ?? null} to={phaseSpan?.to ?? null} title={phaseSpan ? `실패한 단계 ${summary.failedPhase.name} 구간(앞뒤 5초) · 모든 노드` : '최근 로그'} testid="phase-logs" />
      </section>
    {/if}
    {#each runs as run (run.id)}
      <section class="run" aria-label="테스트 실행 결과">
        <h3>테스트 실행 <code>{run.id}</code></h3>
        <RunTimeline {run} nodes={networkNodes} />
      </section>
    {/each}
    <JobCard {job} {webSession} onchanged={load} />
  {:else if !error}<p class="muted">작업을 불러오는 중…</p>{/if}
</section>

<style>
  .summary { display: grid; grid-template-columns: 9rem minmax(0, 1fr); gap: var(--space-2) var(--space-4); } dt { color: var(--text-muted); } dd { margin: 0; }
  @media (max-width: 700px) { .summary { grid-template-columns: 1fr; } }
  .phases { display: flex; gap: 2px; height: 10px; margin-top: var(--space-4); border-radius: 999px; overflow: hidden; background: var(--border-subtle); }
  .phase { flex: none; min-width: 4px; background: var(--border-strong); } .phase.succeeded { background: var(--ok); } .phase.failed { background: var(--danger); } .phase.running { background: var(--accent); }
  .phase-names { display: flex; flex-wrap: wrap; gap: var(--space-1) var(--space-3); margin: var(--space-2) 0 0; padding: 0; list-style: none; color: var(--text-muted); font-size: var(--text-sm); }
  .phase-names li.failed { color: var(--danger); font-weight: 600; } .phase-names li.running { color: var(--accent); }
  .cause, .run { margin-top: var(--space-4); padding-top: var(--space-3); border-top: 1px solid var(--border-subtle); }
</style>
