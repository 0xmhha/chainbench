<script>
  import { formatTime } from './labels.mjs'
  import StatusBadge from './StatusBadge.svelte'
  import { webApi } from './web-api.mjs'
  import { jobSummary } from './job-view.mjs'
  import { follow } from './nav.mjs'
  // link shows the way to the job's own page; the detail page leaves it off.
  // compact drops what the detail page already shows above it: the heading,
  // the phases and the error.
  let { job, webSession, onchanged = () => {}, link = false, compact = false } = $props()
  const api = $derived(webApi(webSession))
  const summary = $derived(jobSummary(job))
  const owner = $derived(webSession && (job.actorId === webSession.user.id || webSession.user.role === 'administrator'))
  const outcomes = { absent: '남은 노드 없음, 자원 해제', present: '노드가 남아 있어 보존 네트워크로 유지', unverified: '일부를 확인하지 못해 계속 잠금' }
  let busy = $state(false), error = $state('')
  async function work(fn) { busy = true; error = ''; try { await fn(); onchanged() } catch (e) { error = e.message } finally { busy = false } }
</script>

<article data-job-id={job.id}>
  {#if !compact}<div class="heading"><strong>{job.operation}</strong><StatusBadge state={job.state} /></div>
  <p>{#if link}<a href={'/jobs/' + job.id} onclick={e => follow(e, '/jobs/' + job.id)}><code>{job.id}</code></a>{:else}<code>{job.id}</code>{/if} · {formatTime(job.createdAt)} · 노드 처리: {job.nodeDisposition}</p>
  <ol>{#each job.phases as phase}<li>{phase.name} · <StatusBadge state={phase.state} />{#if phase.message}<pre>{phase.message}</pre>{/if}</li>{/each}</ol>{/if}
  {#if job.runIds?.length}<p>실제 엔진 세션: {job.runIds.join(', ')}</p><a href={'/history?search=' + encodeURIComponent(job.id)}>세션 결과 보기</a>{/if}
  {#if job.error && !compact}<p class:error={job.state !== 'cancelled'}>{job.error.message}</p>{/if}
  {#if summary.effects.length}<details><summary>수행된 변경</summary><ul>{#each summary.effects as effect}<li>{effect}</li>{/each}</ul></details>{/if}
  {#if summary.unresolved.length}<p class="error">확인이 필요한 자원: {summary.unresolved.join(', ')}</p>{/if}
  {#if job.reconciliation}<div data-testid={'reconciliation-' + job.id}><p>실제 상태 확인 {formatTime(job.reconciliation.checkedAt)} · {outcomes[job.reconciliation.outcome] ?? job.reconciliation.outcome}{job.reconciliation.message ? ' · ' + job.reconciliation.message : ''}</p>{#if job.reconciliation.nodes?.length}<ul>{#each job.reconciliation.nodes as n}<li>{n.nodeId}: 프로세스 {n.process} · 데이터 {n.data}</li>{/each}</ul>{/if}</div>{/if}
  {#if summary.needsCheck && owner}<button disabled={busy} onclick={() => work(() => api('jobs/' + job.id + '/reconcile', 'POST', {}))}>실제 상태 확인</button>{/if}
  {#if summary.active && owner && webSession.user.role !== 'viewer'}<button disabled={busy || job.state === 'cancelling'} onclick={() => work(() => api(`jobs/${job.id}/cancel`, 'POST', {}))}>작업 취소</button>{/if}
  {#if error}<p role="alert" class="error">{error}</p>{/if}
</article>

<style>
  article { margin-top: var(--space-3); padding: var(--space-3) 0; border-top: 1px solid var(--border-subtle); }
  .heading { display: flex; justify-content: space-between; gap: var(--space-4); }
  button { margin-top: var(--space-3); }
  li { margin-top: var(--space-2); } summary { cursor: pointer; }
</style>
