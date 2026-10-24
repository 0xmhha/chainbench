<script>
  import StatusBadge from './StatusBadge.svelte'
  import { onMount } from 'svelte'
  import { webApi } from './web-api.mjs'
  import { jobStream } from './job-stream.svelte.js'
  import { jobSummary, failureHeadline, jobDuration, phaseRows, readJobFocus, jobFocusSearch, nodeInMessage } from './job-view.mjs'
  import { formatTime } from './labels.mjs'
  import { follow } from './nav.mjs'
  import JobCard from './JobCard.svelte'
  import LogViewer from './LogViewer.svelte'
  import RunTimeline from './RunTimeline.svelte'
  import FailureMetrics from './FailureMetrics.svelte'
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
  let network = $state(null), runs = $state([]), runsLoaded = $state(false)
  const phases = $derived(phaseSegments(job?.phases))
  const rows = $derived(phaseRows(job?.phases))
  // Untimed phases keep a small fixed slot; timed ones share the rest.
  const UNTIMED_SLOT = 4
  const phaseWidth = p => {
    const untimed = phases.filter(x => !x.timed).length
    return p.timed ? (p.share * (100 - untimed * UNTIMED_SLOT)) / 100 : UNTIMED_SLOT
  }
  const failedRow = $derived(rows.find(r => r.open) ?? null)
  const phaseSpan = $derived.by(() => {
    const s = Date.parse(failedRow?.startedAt ?? ''), e = Date.parse(failedRow?.finishedAt ?? '')
    return Number.isFinite(s) && Number.isFinite(e) ? { from: new Date(s - PAD_MS).toISOString(), to: new Date(e + PAD_MS).toISOString() } : null
  })
  const networkNodes = $derived((network?.nodes ?? []).map(n => n.id))
  const failed = run => (run?.summary?.counts?.fail ?? 0) + (run?.summary?.counts?.blocked ?? 0) > 0
  const failingRuns = $derived(runs.filter(failed))
  const passingRuns = $derived(runs.filter(r => !failed(r)))
  const head = $derived(job ? failureHeadline(job, summary, failingRuns) : null)
  const duration = $derived(job ? jobDuration(job) : null)
  const stale = $derived(summary?.active && live.connection !== '연결됨')

  // The chosen test and log node live in the URL, so a copied link opens the
  // same evidence. Without a choice the likeliest cause is shown.
  const focus = $state(readJobFocus(location.search))
  $effect(() => {
    const search = jobFocusSearch(focus)
    if (search !== location.search) history.replaceState(history.state, '', location.pathname + search)
  })
  const ALL = '*'
  // A failed phase whose message names one node opens on that node.
  const messageNode = $derived(failingRuns.length ? '' : nodeInMessage(summary?.failedPhase?.message, networkNodes))
  const trailNode = $derived(head?.node || messageNode)
  const phaseNode = $derived(focus.node && (networkNodes.includes(focus.node) || focus.node === ALL) ? focus.node : messageNode || (networkNodes.length > 1 ? ALL : networkNodes[0] ?? ''))

  // Each step of the failure trail scrolls to its evidence; the node also
  // switches the logs to it.
  function reveal(id) { document.getElementById(id)?.scrollIntoView({ block: 'start', behavior: 'smooth' }) }
  function showTest() { focus.test = head.test; reveal('run-' + head.runId) }
  function showNode() {
    focus.node = trailNode
    if (head.test) { focus.test = head.test; reveal('run-' + head.runId) } else reveal('phase-cause')
  }

  $effect(() => {
    const workspace = job?.workspaceId
    if (!workspace || !webSession) return
    api('networks').then(page => { network = page.items.find(n => n.workspaceId === workspace) ?? null }).catch(() => { network = null })
  })
  $effect(() => {
    const ids = job?.runIds ?? []
    if (!ids.length || !webSession) { runs = []; runsLoaded = true; return }
    // Raw text keeps large integers exact (parseHistoryJSON), as History does.
    const read = id => fetch('/api/v1/history/' + encodeURIComponent(id), { cache: 'no-store' }).then(r => r.ok ? r.text() : Promise.reject(new Error(String(r.status)))).then(parseHistoryJSON)
    Promise.all(ids.map(id => read(id).catch(() => null)))
      .then(found => { runs = found.filter(Boolean); runsLoaded = true })
  })
  $effect(() => { void jobId; if (webSession) load() })
  onMount(() => feed.start())
  const seconds = ms => ms === null ? '—' : ms < 60000 ? `${(ms / 1000).toFixed(1)}초` : `${Math.floor(ms / 60000)}분 ${Math.round((ms % 60000) / 1000)}초`
</script>

<section class="panel" aria-label="작업 상세">
  <p><a href="/chains" onclick={e => follow(e, '/chains')}>← 체인 화면의 작업 목록</a></p>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if job}
    <header class="headline {head.tone}" data-testid="job-headline">
      <div class="what">
        <h2>{head.title}</h2>
        {#if head.reason}<p class="reason">{head.reason}</p>{/if}
        {#if head.tone === 'fail' && (head.phase || head.test)}
          <ol class="trail" aria-label="실패 경로">
            {#if head.phase}<li><a href="#phases" onclick={e => { e.preventDefault(); reveal('phases') }}><small>단계</small> {head.phase}</a></li>{/if}
            {#if head.test}<li><button type="button" class="link" onclick={showTest}><small>테스트</small> {head.test}</button></li>{/if}
            {#if trailNode}<li><button type="button" class="link" onclick={showNode}><small>노드</small> {trailNode} 로그</button></li>{/if}
          </ol>
        {/if}
        {#if stale}<p class="notice">실시간 연결이 {live.connection} 상태입니다. 보이는 상태가 최신이 아닐 수 있습니다.</p>{/if}
      </div>
      <dl class="facts">
        <dt>상태</dt><dd data-testid="job-state"><StatusBadge state={job.state} /></dd>
        <dt>걸린 시간</dt><dd class="num">{summary.active ? '실행 중' : seconds(duration)}</dd>
        <dt>작업</dt><dd><code>{job.operation}</code></dd>
        <dt>네트워크</dt><dd>{#if network}<code title={network.id}>{network.id.slice(0, 8)}</code> · 노드 {networkNodes.length}개{:else}<span class="muted">기록 없음</span>{/if}</dd>
        {#if head.runId}<dt>실패한 실행</dt><dd><code title={head.runId}>{head.runId}</code></dd>{/if}
        <dt>시작</dt><dd class="num">{formatTime(job.createdAt)}</dd>
      </dl>
    </header>

    <!-- A failing test run is the cause a reader came for, so it comes first. -->
    {#each failingRuns as run, i (run.id)}
      <section class="run" id="run-{run.id}" aria-label="실패한 테스트 실행">
        <h3>테스트 실행 <code>{run.id}</code></h3>
        {#if i === 0}<RunTimeline {run} nodes={networkNodes} bind:test={focus.test} bind:focusNode={focus.node} />
        {:else}<RunTimeline {run} nodes={networkNodes} />{/if}
      </section>
    {/each}
    {#if !runsLoaded && job.runIds?.length}<p class="muted">테스트 실행 기록을 불러오는 중…</p>{/if}

    {#if summary.failedPhase && !failingRuns.length}
      <section class="cause" id="phase-cause" aria-label="실패한 단계 시점 로그">
        <h3>원인 · {summary.failedPhase.name} 단계 구간의 로그와 지표</h3>
        {#if !network}<p class="muted">이 작업의 네트워크 기록이 없어 노드 로그를 열 수 없습니다.</p>
        {:else if !networkNodes.length}<p class="muted">네트워크에 기록된 노드가 없어 로그를 열 수 없습니다.</p>
        {:else}
          {#if !phaseSpan}<p class="notice">실패한 단계에 시각이 없어 최근 2시간의 로그를 엽니다.</p>{/if}
          <LogViewer networkId={network.id} nodes={networkNodes} bind:node={() => phaseNode, v => { focus.node = v }} from={phaseSpan?.from ?? null} to={phaseSpan?.to ?? null} initialLevels={['ERROR', 'WARN']} focus={failedRow?.finishedAt ?? null} title={phaseSpan ? `실패한 단계 ${summary.failedPhase.name} 구간(앞뒤 5초) · ${phaseNode === ALL ? '모든 노드' : phaseNode} · ERROR·WARN 먼저` : '최근 로그 · ERROR·WARN 먼저'} testid="phase-logs" />
          {#if phaseSpan}
            {#if phaseNode === ALL}<p class="muted">노드 하나를 고르면 그 노드의 지표를 같은 구간으로 봅니다.</p>
            {:else}<FailureMetrics networkId={network.id} node={phaseNode} from={phaseSpan.from} to={phaseSpan.to} at={failedRow.finishedAt} atLabel="단계가 끝난 시각" />{/if}
          {/if}
        {/if}
      </section>
    {/if}

    {#if rows.length}
      <section class="phases-block" id="phases" aria-label="단계">
        <h3>단계 <small>{rows.length}개</small></h3>
        <div class="phases" role="img" aria-label={phases.map(p => `${p.name} ${p.state}`).join(', ')}>
          {#each phases as p}<span class="phase {p.state}" style:width="{phaseWidth(p)}%" title="{p.name} · {p.state}"></span>{/each}
        </div>
        <ol class="phase-list">
          {#each rows as p, i (i)}
            <li><details open={p.open} class={p.state}>
              <summary><StatusBadge state={p.state} /><span class="pname">{p.name}</span><span class="num">{seconds(p.durationMs)}</span></summary>
              <dl class="phase-facts">
                <dt>시각</dt><dd class="num">{p.startedAt ? `${formatTime(p.startedAt)} – ${formatTime(p.finishedAt)}` : '기록 없음'}</dd>
                {#if p.message}<dt>메시지</dt><dd class:error-text={p.state === 'failed'}>{p.message}</dd>{/if}
              </dl>
            </details></li>
          {/each}
        </ol>
      </section>
    {/if}

    {#each passingRuns as run (run.id)}
      <section class="run" aria-label="테스트 실행 결과">
        <h3>테스트 실행 <code>{run.id}</code></h3>
        <RunTimeline {run} nodes={networkNodes} />
      </section>
    {/each}
    <details class="meta" open={!summary.failedPhase && !summary.failure && !failingRuns.length}>
      <summary>작업 정보</summary>
      <dl class="summary">
        <dt>작업</dt><dd>{job.operation} · <code>{job.id}</code></dd>
        <dt>실패한 단계</dt><dd data-testid="job-failed-phase">{summary.failedPhase ? `${summary.failedPhase.name}${summary.failedPhase.message ? ' · ' + summary.failedPhase.message : ''}` : '없음'}</dd>
        {#if summary.failure}<dt>{job.state === 'cancelled' ? '중단 메시지' : '실패 이유'}</dt><dd class:error-text={job.state !== 'cancelled'}>{summary.failure}</dd>{/if}
        <dt>이미 반영된 변경</dt><dd data-testid="job-effects">{summary.effects.length ? `${summary.effects.length}건 기록됨 · 아래 '수행된 변경'에서 봅니다` : '기록 없음'}</dd>
        <dt>확인할 자원</dt><dd data-testid="job-unresolved">{summary.unresolved.length ? `${summary.unresolved.length}개` : '없음'}</dd>
        <dt>노드 처리</dt><dd data-testid="job-disposition">{job.nodeDisposition}{summary.needsCheck ? ' · 실제 상태 확인이 필요합니다' : ''}</dd>
        {#if job.cancelReason}<dt>취소</dt><dd>{job.cancelReason}{job.cancelMessage ? ' · ' + job.cancelMessage : ''}</dd>{/if}
      </dl>
    </details>
    <JobCard {job} {webSession} compact onchanged={load} />
  {:else if !error}<p class="muted">작업을 불러오는 중…</p>{/if}
  <p class="muted note" data-testid="job-continues">이 작업은 서버에서 실행됩니다. 브라우저를 닫거나 다른 화면으로 가도 계속되며, 다시 열면 기록된 상태를 보여 줍니다.</p>
  <p class="muted note" data-testid="job-stream">실시간 연결: {live.connection}{live.gap ? ' · ' + live.gap : ''}</p>
</section>

<style>
  .headline { display: grid; grid-template-columns: minmax(0, 1fr) minmax(220px, 300px); gap: var(--space-4) var(--space-5); margin: var(--space-3) 0 0; padding: var(--space-4) var(--space-5); border: 1px solid var(--border-subtle); border-radius: var(--radius); background: var(--surface-raised); }
  .headline.fail { border-color: var(--danger-border); background: color-mix(in srgb, var(--danger-bg) 70%, var(--surface)); }
  .headline.warn { border-color: var(--warn-border); }
  .headline.ok { border-color: var(--ok-border); }
  .headline h2 { font-size: var(--text-xl); text-wrap: balance; }
  .headline.fail h2 { color: var(--danger); }
  .reason { margin: var(--space-2) 0 0; font: var(--text-sm) var(--font-mono); color: var(--text); max-width: 75ch; }
  .trail { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-1) var(--space-2); margin: var(--space-3) 0 0; padding: 0; list-style: none; font-size: var(--text-sm); }
  .trail li + li::before { content: '›'; margin-right: var(--space-2); color: var(--text-faint); }
  .trail small { margin-right: 2px; color: var(--text-muted); }
  .trail button.link { min-height: 24px; text-decoration: none; }
  .trail button.link:hover { text-decoration: underline; }
  .what .notice { margin-bottom: 0; }
  .facts { display: grid; grid-template-columns: auto minmax(0, 1fr); gap: var(--space-1) var(--space-3); margin: 0; align-content: start; font-size: var(--text-sm); }
  .facts dt { color: var(--text-muted); } .facts dd { margin: 0; min-width: 0; overflow-wrap: anywhere; }
  .num { font-variant-numeric: tabular-nums; }
  .summary { display: grid; grid-template-columns: 9rem minmax(0, 1fr); gap: var(--space-2) var(--space-4); } dt { color: var(--text-muted); } dd { margin: 0; }
  .phases { display: flex; gap: 2px; height: 8px; margin-top: var(--space-2); border-radius: 999px; overflow: hidden; background: var(--border-subtle); }
  .phase { flex: none; min-width: 4px; background: var(--border-strong); } .phase.succeeded { background: var(--ok); } .phase.failed { background: var(--danger); } .phase.running { background: var(--accent); }
  .phases-block h3 small { margin-left: var(--space-1); font-weight: 400; }
  .phase-list { margin: var(--space-3) 0 0; padding: 0; list-style: none; border-top: 1px solid var(--border-subtle); }
  .phase-list li { border-bottom: 1px solid var(--border-subtle); }
  .phase-list summary { display: grid; grid-template-columns: 7.5rem minmax(0, 1fr) auto; align-items: center; gap: var(--space-3); min-height: 40px; padding: var(--space-1) var(--space-2); cursor: pointer; list-style-position: outside; }
  .phase-list summary:hover { background: color-mix(in srgb, var(--surface-raised) 60%, transparent); }
  .phase-list details.failed > summary .pname { color: var(--danger); font-weight: 600; }
  .phase-list details.running > summary .pname { color: var(--accent); }
  .pname { font-family: var(--font-mono); font-size: var(--text-sm); overflow-wrap: anywhere; }
  .phase-list summary .num { color: var(--text-muted); font-size: var(--text-sm); }
  .phase-facts { display: grid; grid-template-columns: 5rem minmax(0, 1fr); gap: var(--space-1) var(--space-3); margin: 0; padding: 0 var(--space-2) var(--space-3) calc(7.5rem + var(--space-5)); font-size: var(--text-sm); }
  .error-text { color: var(--danger); }
  .cause, .run, .phases-block { margin-top: var(--space-5); padding-top: var(--space-4); border-top: 1px solid var(--border-subtle); }
  .meta { margin-top: var(--space-5); } .meta summary { cursor: pointer; color: var(--text-muted); }
  .note { margin: var(--space-3) 0 0; font-size: var(--text-xs); }
  @media (max-width: 800px) {
    .headline { grid-template-columns: minmax(0, 1fr); padding: var(--space-3) var(--space-4); }
    .phase-facts { padding-left: var(--space-2); }
    .phase-list summary { grid-template-columns: auto minmax(0, 1fr) auto; }
  }
  @media (max-width: 700px) { .summary { grid-template-columns: 1fr; } }
</style>
