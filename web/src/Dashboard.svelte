<script>
  import { formatTime } from './labels.mjs'
  import StatusBadge from './StatusBadge.svelte'
  import { onMount } from 'svelte'
  import { sharedJobStream } from './job-stream.svelte.js'
  import { jobSummary, jobDuration } from './job-view.mjs'
  import PhaseSteps from './PhaseSteps.svelte'
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
  const took = ms => ms === null ? '—' : ms < 60000 ? `${Math.round(ms / 1000)}초` : `${Math.floor(ms / 60000)}분 ${Math.round((ms % 60000) / 1000)}초`
  onMount(() => { const stop = feed.start(); const clock = setInterval(() => { now = Date.now() }, 1000); return () => { stop(); clearInterval(clock) } })
</script>

{#snippet notReady()}<p class="muted">{phase === 'unreachable' ? '서버에 연결되면 표시합니다.' : '불러오는 중…'}</p>{/snippet}

<!-- The dashboard is the log's cover page: the totals ruled across the top,
     then the jobs someone must look at in red ink, what is running, the
     nodes as a ruled grid and the latest results. -->
{#if phase === 'unreachable'}<p class="error" role="alert">{live.error}</p>{:else if phase === 'loading'}<p class="muted" role="status">작업 상태를 불러오는 중…</p>{/if}
<dl class="totals" aria-label="전체 상태">
  <div><dt>실행 중 작업</dt><dd data-testid="running-count">{count(running.length)}</dd></div>
  <div><dt>네트워크</dt><dd>{count(networks.length)}</dd></div>
  <div><dt>노드</dt><dd>{count(tally.total)}{#if tally.total}<small>실행 {tally.running} · 정지 {tally.stopped}{tally.other ? ` · 기타 ${tally.other}` : ''}</small>{/if}</dd></div>
  <div class:warn={drops > 0} title="작업 {live.drops ?? '확인 중'} · 이벤트 버스 {live.busDrops ?? '확인 중'} · 연결 {live.connection}"><dt>관측 누락</dt><dd data-testid="observation-drops">{drops}</dd></div>
</dl>

{#if attention.length}
  <section class="panel attention" aria-label="주의가 필요한 작업"><h2>주의가 필요한 작업 <small>최근 {attention.length}건</small></h2>
    <div class="table-wrap"><table class="stack"><thead><tr><th>생성</th><th>작업</th><th>상태</th><th>원인</th><th><span class="sr">열기</span></th></tr></thead><tbody>
      {#each attention as j (j.id)}<tr data-job-id={j.id}>
        <td class="num" data-label="생성">{fmt(j.createdAt)}</td>
        <td data-label="작업"><a href={'/jobs/' + j.id} onclick={e => follow(e, '/jobs/' + j.id)}>{j.operation}</a><small class="mono" title={j.id}>{shortId(j.id)}</small></td>
        <td data-label="상태"><StatusBadge state={j.state} /></td>
        <td class="why" data-label="원인">{j.failedPhase ? `${j.failedPhase.name} 단계${j.failedPhase.message ? ' · ' + j.failedPhase.message : ''}` : j.reason || '실행이 끝나지 않았습니다'}</td>
        <td><a class="open" href={'/jobs/' + j.id} onclick={e => follow(e, '/jobs/' + j.id)}>원인 보기 →</a></td>
      </tr>{/each}
    </tbody></table></div>
  </section>
{/if}

<section class="panel" aria-label="실행 중 작업"><h2>실행 중 작업</h2>
  {#if phase !== 'ready'}{@render notReady()}{:else if running.length === 0}<p class="muted">진행 중인 작업이 없습니다.</p>{:else}
    <div class="table-wrap"><table class="stack"><thead><tr><th>작업</th><th>상태</th><th>진행 단계</th><th>시작</th></tr></thead><tbody>
      {#each running as j (j.id)}<tr data-job-id={j.id}>
        <td data-label="작업"><a href={'/jobs/' + j.id} onclick={e => follow(e, '/jobs/' + j.id)}>{j.operation}</a><small class="mono" title={j.id}>{shortId(j.id)}</small></td>
        <td data-label="상태"><StatusBadge state={j.state} /></td>
        <td data-label="진행 단계"><PhaseSteps phases={j.phases} /></td>
        <td class="num" data-label="시작">{fmt(j.createdAt)}</td>
      </tr>{/each}
    </tbody></table></div>
  {/if}
</section>

<section class="panel" aria-label="노드 상태"><h2>노드 상태</h2>
  <p class="muted lead">서버 수집기가 5초마다 본 마지막 모습입니다. 노드를 제어하기 전에는 노드 상세에서 상태를 다시 확인합니다.</p>
  {#each networks as n (n.id)}
    {@const t = nodeTally(n.nodes)}
    <div class="network">
      <p class="folio"><span>네트워크 <code title={n.id}>{shortId(n.id)}</code> · {n.ownership}</span><span>실행 {t.running} · 정지 {t.stopped}{t.other ? ` · 기타 ${t.other}` : ''}</span></p>
      <ul class="grid">{#each n.nodes as node (node.id)}{@const shown = shownState(node.state, stale)}{@const s = statusIcon(shown.state)}<li><a class="cell {s.tone}" href={`/nodes/${n.id}/${node.id}`} onclick={e => follow(e, `/nodes/${n.id}/${node.id}`)}><span class="node">{node.id}</span><span class="role">{node.role}</span><span class="state"><StatusMark state={shown.state} /> {shown.state}{#if shown.last}<small title="{stateLabel(shown.last)}"> · 마지막 {shown.last}</small>{/if}</span></a></li>{/each}</ul>
    </div>
  {:else}{#if phase === 'ready'}<p class="muted">기록된 네트워크가 없습니다.</p>{:else}{@render notReady()}{/if}{/each}
</section>

<section class="panel" aria-label="최근 결과"><div class="panel-heading"><h2>최근 결과</h2><button onclick={() => feed.refresh()}>새로고침</button></div>
  {#if recent.length}<div class="table-wrap"><table class="stack"><thead><tr><th>작업</th><th>결과</th><th>걸린 시간</th><th>노드 처리</th><th>생성</th></tr></thead><tbody>
    {#each recent as j (j.id)}<tr data-job-id={j.id}>
      <td data-label="작업"><a href={'/jobs/' + j.id} onclick={e => follow(e, '/jobs/' + j.id)}>{j.operation}</a><small class="mono" title={j.id}>{shortId(j.id)}</small></td>
      <td data-label="결과"><StatusBadge state={j.state} /></td>
      <td class="num" data-label="걸린 시간">{took(jobDuration(j))}</td>
      <td data-label="노드 처리">{j.nodeDisposition}</td>
      <td class="num" data-label="생성">{fmt(j.createdAt)}</td>
    </tr>{/each}
  </tbody></table></div>{/if}
  {#if phase !== 'ready'}{@render notReady()}{:else if recent.length === 0}<div class="empty"><h3>관측된 실행이 없습니다</h3><p>체인 구성과 테스트 정의를 준비하세요.</p><a href="/chains" onclick={e => follow(e, '/chains')}>체인 구성 열기 →</a></div>{/if}
</section>

<style>
  /* The totals: figures ruled apart, each named above in the label face. */
  .totals { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); margin: var(--space-4) 0 0; border-top: var(--rule-strong); border-bottom: var(--rule-strong); }
  .totals > div { padding: var(--space-2) var(--space-4) var(--space-3); min-width: 0; }
  .totals > div + div { border-left: var(--rule); }
  .totals dt { font: 600 var(--text-xs) var(--font-cond); letter-spacing: .04em; color: var(--text-muted); }
  .totals dd { margin: var(--space-1) 0 0; font: 700 var(--text-xl) var(--font-mono); font-variant-numeric: tabular-nums; color: var(--text); }
  .totals dd small { display: block; font: var(--text-xs) var(--font-sans); color: var(--text-muted); }
  .totals .warn dt, .totals .warn dd { color: var(--warn); }
  .attention { border-top-color: var(--danger); }
  .attention h2, .attention .why { color: var(--danger); }
  h2 small { margin-left: var(--space-2); font: 400 var(--text-xs) var(--font-sans); color: var(--text-muted); }
  .why { overflow-wrap: anywhere; }
  .open { white-space: nowrap; }
  td small { display: block; }
  .num { font: var(--text-sm) var(--font-mono); font-variant-numeric: tabular-nums; white-space: nowrap; }
  .lead { margin: 0 0 var(--space-2); font-size: var(--text-sm); }
  .network { margin-top: var(--space-4); }
  .network .folio { margin: 0; }
  /* Nodes on a ruled grid: the state is a drawn mark and a word, in the ink of
     its meaning; no cell is filled with colour. */
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); margin: 0; padding: 0; list-style: none; border-left: var(--rule); }
  .grid li { border-right: var(--rule); border-bottom: var(--rule); min-width: 0; }
  .cell { display: grid; gap: 2px; min-height: 44px; padding: var(--space-2) var(--space-3); color: var(--text); }
  .cell:hover { text-decoration: none; background: var(--surface-raised); }
  .node { font: 600 var(--text-sm) var(--font-mono); } .role { color: var(--text-faint); font-size: var(--text-xs); }
  .state { font-size: var(--text-xs); } .ok .state { color: var(--ok); } .bad .state, .bad .node { color: var(--danger); } .warn .state { color: var(--warn); } .neutral .state, .active .state { color: var(--text-muted); }
  .sr { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); white-space: nowrap; }
  @media (max-width: 700px) {
    .totals { grid-template-columns: repeat(2, minmax(0, 1fr)); }
    .totals > div:nth-child(3) { border-left: 0; }
    .totals > div:nth-child(n+3) { border-top: var(--rule); }
  }
</style>
