<script>
  import { formatTime } from './labels.mjs'
  import StatusBadge from './StatusBadge.svelte'
  import { webApi } from './web-api.mjs'
  import { jobSummary } from './job-view.mjs'
  import { follow } from './nav.mjs'
  import { runLabel } from './dashboard-view.mjs'
  import ConfirmAction from './ConfirmAction.svelte'
  // link shows the way to the job's own page; the detail page leaves it off.
  // compact drops what the detail page already shows above it: the heading,
  // the phases and the error.
  let { job, webSession, onchanged = () => {}, link = false, compact = false } = $props()
  const api = $derived(webApi(webSession))
  const summary = $derived(jobSummary(job))
  const owner = $derived(webSession && (job.actorId === webSession.user.id || webSession.user.role === 'administrator'))
  const outcomes = { absent: '남은 노드 없음, 자원 해제', present: '노드가 남아 있어 보존 네트워크로 유지', unverified: '일부를 확인하지 못해 계속 잠금' }
  let busy = $state(false), error = $state(''), asking = $state(false)
  async function work(fn) { busy = true; error = ''; try { await fn(); onchanged() } catch (e) { error = e.message } finally { busy = false } }
</script>

<article data-job-id={job.id}>
  {#if !compact}<div class="heading"><strong>{job.operation}</strong><StatusBadge state={job.state} /></div>
  <p>{#if link}<a href={'/jobs/' + job.id} onclick={e => follow(e, '/jobs/' + job.id)}><code>{job.id}</code></a>{:else}<code>{job.id}</code>{/if} · {formatTime(job.createdAt)} · 노드 처리: {job.nodeDisposition}</p>
  <ol>{#each job.phases as phase}<li>{phase.name} · <StatusBadge state={phase.state} />{#if phase.message}<pre>{phase.message}</pre>{/if}</li>{/each}</ol>{/if}
  <!-- runIds name the engine sessions; each opens its archived record in History. -->
  {#if job.runIds?.length}<p class="sessions">엔진 세션{#each job.runIds as id (id)}<a class="session-link" href={'/history?run=' + encodeURIComponent(id)} title={id} onclick={e => follow(e, '/history?run=' + encodeURIComponent(id))}>{runLabel(id)} 결과</a>{/each}</p>{/if}
  {#if job.error && !compact}<p class:error={job.state !== 'cancelled'}>{job.error.message}</p>{/if}
  {#if summary.effects.length}<details><summary>수행된 변경</summary><ul class="lines">{#each summary.effects as effect}<li>{effect}</li>{/each}</ul></details>{/if}
  <!-- Resources to check are a warning, not a fault: an ochre rule, each path on its own line. -->
  {#if summary.unresolved.length}<div class="check" data-testid="job-unresolved-list"><p>확인이 필요한 자원 {summary.unresolved.length}개{summary.needsCheck && owner ? ' · 아래 실제 상태 확인으로 지금 상태를 읽습니다' : ''}</p><ul class="lines">{#each summary.unresolved as path}<li>{path}</li>{/each}</ul></div>{/if}
  {#if job.reconciliation}<div data-testid={'reconciliation-' + job.id}><p>실제 상태 확인 {formatTime(job.reconciliation.checkedAt)} · {outcomes[job.reconciliation.outcome] ?? job.reconciliation.outcome}{job.reconciliation.message ? ' · ' + job.reconciliation.message : ''}</p>{#if job.reconciliation.nodes?.length}<ul>{#each job.reconciliation.nodes as n}<li>{n.nodeId}: 프로세스 {n.process} · 데이터 {n.data}</li>{/each}</ul>{/if}</div>{/if}
  {#if summary.needsCheck && owner}<button disabled={busy} onclick={() => work(() => api('jobs/' + job.id + '/reconcile', 'POST', {}))}>실제 상태 확인</button>{/if}
  {#if summary.active && owner && webSession.user.role !== 'viewer'}<button disabled={busy || job.state === 'cancelling'} onclick={() => { asking = true }}>작업 취소</button>
    {#if asking}<ConfirmAction label="작업 취소 확인" message={`${job.operation} 작업을 취소합니다. 이미 반영된 변경은 되돌리지 않고, 노드는 작업의 보존 설정(${job.retention})에 따라 처리됩니다.`} confirmLabel="작업 취소 확정" {busy} onconfirm={() => { asking = false; work(() => api(`jobs/${job.id}/cancel`, 'POST', {})) }} oncancel={() => { asking = false }} />{/if}{/if}
  {#if error}<p role="alert" class="error">{error}</p>{/if}
</article>

<style>
  article { margin-top: var(--space-3); padding: var(--space-3) 0; border-top: 1px solid var(--border-subtle); }
  .heading { display: flex; justify-content: space-between; gap: var(--space-4); }
  button { margin-top: var(--space-3); }
  /* A pointer target of at least 24px, as WCAG 2.5.8 asks. */
  .sessions { display: flex; flex-wrap: wrap; align-items: center; gap: 0 var(--space-3); }
  .session-link { display: inline-flex; align-items: center; min-height: 24px; padding: 2px 0; font: var(--text-sm) var(--font-mono); }
  @media (pointer: coarse) { .session-link { min-height: 44px; } }
  .lines { padding-left: var(--space-4); font: var(--text-xs) var(--font-mono); max-width: 110ch; overflow-wrap: anywhere; }
  .check { margin-top: var(--space-3); padding-top: var(--space-2); border-top: 2px solid var(--warn); }
  .check p { margin: 0; color: var(--warn); font-weight: 600; }
  li { margin-top: var(--space-2); } summary { cursor: pointer; }
</style>
