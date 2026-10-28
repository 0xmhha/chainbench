<script>
  import { formatTime } from './labels.mjs'
  import StatusBadge from './StatusBadge.svelte'
  import { onMount } from 'svelte'
  import { sharedJobStream } from './job-stream.svelte.js'
  import { jobSummary } from './job-view.mjs'
  import { follow } from './nav.mjs'
  import { nodeTally, attentionJobs, shortId } from './dashboard-view.mjs'
  import { statusIcon } from './status-icon.mjs'
  import StatusMark from './StatusMark.svelte'
  import { liveStale, shownState, feedPhase } from './node-view.mjs'
  import { stateLabel } from './labels.mjs'
  const feed = sharedJobStream(), live = feed.state
  const jobs = $derived(live.snapshot?.jobs ?? [])
  const running = $derived(jobs.filter(j => jobSummary(j).active))
  const recent = $derived(jobs.filter(j => !jobSummary(j).active).slice(0, 10))
  const networks = $derived(live.snapshot?.networks ?? [])
  const ATTENTION_LIMIT = 5
  const attention = $derived(attentionJobs(jobs, ATTENTION_LIMIT))
  const tally = $derived(nodeTally(networks.flatMap(n => n.nodes)))
  const drops = $derived((live.drops ?? 0) + (live.busDrops ?? 0))
  const fmt = formatTime
  // A node cell shows its last state as stale once live state may be out of date.
  let now = $state(Date.now())
  // Until the first snapshot arrives nothing is known, which is not the same as empty.
  const phase = $derived(feedPhase(live))
  const count = n => phase === 'ready' ? n : '–'
  const stale = $derived(liveStale(live, now))
  onMount(() => { const stop = feed.start(); const clock = setInterval(() => { now = Date.now() }, 1000); return () => { stop(); clearInterval(clock) } })
</script>

{#snippet notReady()}<p class="muted">{phase === 'unreachable' ? '서버에 연결되면 표시합니다.' : '불러오는 중…'}</p>{/snippet}

{#if phase === 'unreachable'}<p class="error" role="alert">{live.error}</p>{:else if phase === 'loading'}<p class="muted" role="status">작업 상태를 불러오는 중…</p>{/if}
<p class="strip" aria-label="전체 상태">
  <span>실행 중 작업 <b data-testid="running-count">{count(running.length)}</b></span>
  <span>네트워크 <b>{count(networks.length)}</b></span>
  <span>노드 <b>{count(tally.total)}</b>{#if tally.total}<span class="tally"> · 실행 {tally.running} · 정지 {tally.stopped}{tally.other ? ` · 기타 ${tally.other}` : ''}</span>{/if}</span>
  <span class:warn={drops > 0} title="작업 {live.drops ?? '확인 중'} · 이벤트 버스 {live.busDrops ?? '확인 중'} · 연결 {live.connection}">관측 누락 <b data-testid="observation-drops">{drops}</b></span>
</p>

{#if attention.length}
  <section class="panel attention" aria-label="주의가 필요한 작업"><h2>주의가 필요한 작업</h2>
    <ul>{#each attention as j (j.id)}<li data-job-id={j.id}>
      <StatusBadge state={j.state} />
      <a href={'/jobs/' + j.id} onclick={e => follow(e, '/jobs/' + j.id)}>{j.operation}</a>
      <span class="why">{j.failedPhase ? `${j.failedPhase.name} 단계${j.failedPhase.message ? ' · ' + j.failedPhase.message : ''}` : j.reason || '실행이 끝나지 않았습니다'}</span>
      <time>{fmt(j.createdAt)}</time>
      <a class="open" href={'/jobs/' + j.id} onclick={e => follow(e, '/jobs/' + j.id)}>원인 보기 →</a>
    </li>{/each}</ul>
  </section>
{/if}

<section class="panel" aria-label="실행 중 작업"><h2>실행 중 작업</h2>
  {#if phase !== 'ready'}{@render notReady()}{:else if running.length === 0}<p class="muted">진행 중인 작업이 없습니다.</p>{:else}
    <div class="table-wrap"><table><thead><tr><th>작업</th><th>상태</th><th>진행 단계</th><th>시작</th></tr></thead><tbody>
      {#each running as j (j.id)}<tr data-job-id={j.id}><td><a href={'/jobs/' + j.id} onclick={e => follow(e, '/jobs/' + j.id)}>{j.operation}</a><small class="mono" title={j.id}>{shortId(j.id)}</small></td><td><StatusBadge state={j.state} /></td>
        <td><span class="steps" role="img" aria-label="단계 {j.phases.filter(p => p.state === 'succeeded').length}/{j.phases.length}">{#each j.phases as p}<i class={p.state} title="{p.name} · {p.state}"></i>{/each}</span><small>{j.phases.filter(p => p.state === 'succeeded').length}/{j.phases.length} · {j.phases.find(p => p.state === 'running')?.name ?? '대기'}</small></td>
        <td>{fmt(j.createdAt)}</td></tr>{/each}
    </tbody></table></div>
  {/if}
</section>

<section class="panel" aria-label="노드 상태"><h2>노드 상태</h2>
  <p class="muted">서버 수집기가 5초마다 본 마지막 모습입니다. 노드를 제어하기 전에는 노드 상세에서 상태를 다시 확인합니다.</p>
  {#each networks as n (n.id)}
    {@const t = nodeTally(n.nodes)}
    <div class="network">
      <p class="net-head"><b class="mono" title={n.id}>{shortId(n.id)}</b> <span class="muted">{n.ownership} · 실행 {t.running} · 정지 {t.stopped}{t.other ? ` · 기타 ${t.other}` : ''}</span></p>
      <ul class="grid">{#each n.nodes as node (node.id)}{@const shown = shownState(node.state, stale)}{@const s = statusIcon(shown.state)}<li><a class="cell {s.tone}" href={`/nodes/${n.id}/${node.id}`} onclick={e => follow(e, `/nodes/${n.id}/${node.id}`)}><span class="node">{node.id}</span><span class="role">{node.role}</span><span class="state"><StatusMark state={shown.state} /> {shown.state}{#if shown.last}<small title="{stateLabel(shown.last)}"> · 마지막 {shown.last}</small>{/if}</span></a></li>{/each}</ul>
    </div>
  {:else}{#if phase === 'ready'}<p class="muted">기록된 네트워크가 없습니다.</p>{:else}{@render notReady()}{/if}{/each}
</section>

<section class="panel" aria-label="최근 결과"><div class="panel-heading"><h2>최근 결과</h2><button onclick={() => feed.refresh()}>새로고침</button></div>
  {#if recent.length}<div class="table-wrap"><table><thead><tr><th>작업</th><th>결과</th><th>노드 처리</th><th>생성</th></tr></thead><tbody>
    {#each recent as j (j.id)}<tr data-job-id={j.id}><td><a href={'/jobs/' + j.id} onclick={e => follow(e, '/jobs/' + j.id)}>{j.operation}</a></td><td><StatusBadge state={j.state} /></td><td>{j.nodeDisposition}</td><td>{fmt(j.createdAt)}</td></tr>{/each}
  </tbody></table></div>{/if}
  {#if phase !== 'ready'}{@render notReady()}{:else if recent.length === 0}<div class="empty"><h3>관측된 실행이 없습니다</h3><p>체인 구성과 테스트 정의를 준비하세요.</p><a href="/chains" onclick={e => follow(e, '/chains')}>체인 구성 열기 →</a></div>{/if}
</section>

<style>
  .strip { display: flex; flex-wrap: wrap; gap: var(--space-2) var(--space-5); margin: var(--space-4) 0 0; padding: var(--space-3) var(--space-4); border: 1px solid var(--border-subtle); border-radius: var(--radius); background: var(--surface); color: var(--text-muted); font-size: var(--text-sm); }
  .strip b { color: var(--text); font-size: var(--text-lg); font-variant-numeric: tabular-nums; margin-left: var(--space-1); }
  .strip .warn, .strip .warn b { color: var(--warn); }
  .tally { font-size: var(--text-xs); }
  .attention { border-color: var(--danger-border); }
  .attention ul { list-style: none; margin: 0; padding: 0; }
  .attention li { display: grid; grid-template-columns: auto auto minmax(0, 1fr) auto auto; gap: var(--space-3); align-items: baseline; padding: var(--space-2) 0; border-top: 1px solid var(--border-subtle); }
  .attention li:first-child { border-top: 0; }
  .why { color: var(--text-muted); overflow-wrap: anywhere; } .attention time { color: var(--text-faint); font-size: var(--text-xs); white-space: nowrap; }
  .open { white-space: nowrap; }
  small { display: block; }
  .steps { display: flex; gap: 2px; width: 120px; margin-bottom: 2px; }
  .steps i { flex: 1; height: 6px; background: var(--border-strong); }
  .steps i.succeeded { background: var(--ok); } .steps i.running { background: var(--accent); } .steps i.failed { background: var(--danger); }
  .network { margin-top: var(--space-3); }
  .net-head { margin: 0 0 var(--space-2); }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: var(--space-2); margin: 0; padding: 0; list-style: none; }
  .cell { display: grid; gap: 2px; padding: var(--space-2) var(--space-3); border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); background: var(--bg); color: var(--text); }
  .cell:hover { text-decoration: none; border-color: var(--accent-border); }
  .cell.ok { border-color: var(--ok-border); } .cell.bad { border-color: var(--danger-border); background: var(--danger-bg); } .cell.warn { border-color: var(--warn-border); background: var(--warn-bg); }
  .node { font-family: var(--font-mono); font-size: var(--text-sm); } .role { color: var(--text-faint); font-size: var(--text-xs); }
  .state { font-size: var(--text-xs); } .ok .state { color: var(--ok); } .bad .state { color: var(--danger); } .warn .state { color: var(--warn); } .neutral .state, .active .state { color: var(--text-muted); }
  @media (max-width: 700px) { .attention li { grid-template-columns: auto minmax(0, 1fr); } .why, .attention time, .open { grid-column: 2; } }
</style>
