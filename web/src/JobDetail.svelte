<script>
  import StatusBadge from './StatusBadge.svelte'
  import { onMount } from 'svelte'
  import { webApi } from './web-api.mjs'
  import { jobStream } from './job-stream.svelte.js'
  import { jobSummary } from './job-view.mjs'
  import { follow } from './nav.mjs'
  import JobCard from './JobCard.svelte'
  let { jobId, webSession } = $props()
  const api = $derived(webApi(webSession))
  const feed = jobStream(), live = feed.state
  let fetched = $state(null), error = $state('')
  // The live feed carries every committed change; the first read covers the
  // moment before it connects and names a job that does not exist.
  const job = $derived(live.snapshot?.jobs.find(j => j.id === jobId) ?? fetched)
  const summary = $derived(job ? jobSummary(job) : null)
  async function load() { error = ''; try { fetched = await api('jobs/' + jobId) } catch (e) { error = e.message } }
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
    <JobCard {job} {webSession} onchanged={load} />
  {:else if !error}<p class="muted">작업을 불러오는 중…</p>{/if}
</section>

<style>
  .panel { margin-top: 24px; padding: 24px; border: 1px solid var(--border); border-radius: 10px; background: var(--surface); }
  .muted { color: var(--text-muted); } a { color: var(--accent); }
  .notice { padding: 12px 16px; border-left: 3px solid var(--warn); background: var(--warn-bg); color: var(--warn); }
  .error { color: var(--danger); }
  .summary { display: grid; grid-template-columns: 9rem minmax(0, 1fr); gap: 8px 16px; } dt { color: var(--text-muted); } dd { margin: 0; overflow-wrap: anywhere; }
  code { font-family: ui-monospace, monospace; }
  @media (max-width: 700px) { .summary { grid-template-columns: 1fr; } }
</style>
