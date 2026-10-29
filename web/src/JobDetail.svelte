<script>
  import StatusBadge from './StatusBadge.svelte'
  import { onMount } from 'svelte'
  import { apiClient } from './api-client.mjs'
  import { decodeJob, decodeNetworkPage, decodeRun } from './api-decode.mjs'
  import { sharedJobStream } from './job-stream.svelte.js'
  import { jobSummary, failureHeadline, jobDuration, jobElapsed, phaseRows, readFocus, focusSearch, nodeInMessage } from './job-view.mjs'
  import { formatTime } from './labels.mjs'
  import { follow } from './nav.mjs'
  import JobCard from './JobCard.svelte'
  import CasualtyEntry from './CasualtyEntry.svelte'
  import CauseBench from './CauseBench.svelte'
  import RunBench from './RunBench.svelte'
  import PhaseWatches from './PhaseWatches.svelte'
  import { testRows } from './run-timeline.mjs'
  import { parseHistoryJSON } from './history-json.js'
  let { jobId, webSession } = $props()
  const client = $derived(apiClient(webSession))
  const feed = sharedJobStream(), live = feed.state
  let fetched = $state(null), error = $state('')
  // The live feed carries every committed change; the first read covers the
  // moment before it connects and names a job that does not exist.
  const job = $derived(live.snapshot?.jobs.find(j => j.id === jobId) ?? fetched)
  const summary = $derived(job ? jobSummary(job) : null)
  async function load() { error = ''; try { fetched = await client.request('jobs/' + jobId, { decode: decodeJob }) } catch (e) { error = e.message } }
  // The failed phase is read against the logs of the network the job worked
  // on, and every test run the job produced is shown as its own timeline.
  const PAD_MS = 5000
  let network = $state(null), runs = $state([]), runsLoaded = $state(false), runsError = $state('')
  const rows = $derived(phaseRows(job?.phases))
  const failedRow = $derived(rows.find(r => r.open) ?? null)
  const phaseSpan = $derived.by(() => {
    const s = Date.parse(failedRow?.startedAt ?? ''), e = Date.parse(failedRow?.finishedAt ?? '')
    return Number.isFinite(s) && Number.isFinite(e) ? { from: new Date(s - PAD_MS).toISOString(), to: new Date(e + PAD_MS).toISOString() } : null
  })
  // Without phase times the bench reads the two hours before the page opened;
  // derived once, so a re-render does not move the window.
  const phaseWindow = $derived(phaseSpan ?? { from: new Date(Date.now() - 7200000).toISOString(), to: new Date().toISOString() })
  const networkNodes = $derived((network?.nodes ?? []).map(n => n.id))
  const failed = run => (run?.summary?.counts?.fail ?? 0) + (run?.summary?.counts?.blocked ?? 0) > 0
  const failingRuns = $derived(runs.filter(failed))
  const passingRuns = $derived(runs.filter(r => !failed(r)))
  // Until the job's runs are read, a failed job's first line waits for them.
  const runsPending = $derived(Boolean(job?.runIds?.length) && !runsLoaded)
  const head = $derived(job ? failureHeadline(job, summary, failingRuns, runsPending, passingRuns) : null)
  const duration = $derived(job ? jobDuration(job) : null)
  // A running job's elapsed time ticks once a second, and only while it runs.
  let now = $state(Date.now())
  $effect(() => { if (!summary?.active) return; const clock = setInterval(() => { now = Date.now() }, 1000); return () => clearInterval(clock) })
  const elapsed = $derived(summary?.active ? jobElapsed(job, now) : null)
  const stale = $derived(summary?.active && live.connection !== '연결됨')

  // The chosen test, log node and cursor moment live in the URL, so a copied
  // link opens the same evidence. Without a choice the likeliest cause is shown.
  const focus = $state(readFocus(location.search))
  $effect(() => {
    const search = focusSearch(focus, location.search)
    if (search !== location.search) history.replaceState(history.state, '', location.pathname + search)
  })
  const ALL = '*'
  // A failed phase whose message names one node opens on that node.
  const messageNode = $derived(failingRuns.length ? '' : nodeInMessage(summary?.failedPhase?.message, networkNodes))
  const trailNode = $derived(head?.node || messageNode)
  // Whether the bench found the asserted node at fault; null until it reads.
  let finding = $state(null)
  // The first failing run is read on the cause bench (RunBench): its chosen
  // test, the first failure unless the URL names another, sets the window. A
  // job that succeeded opens its first run instead, so what passed is in view.
  const firstRun = $derived(failingRuns[0] ?? (job?.state === 'succeeded' ? passingRuns[0] : null) ?? null)
  const testList = $derived(testRows(firstRun))
  const headRow = $derived(testList.find(r => r.id === head?.test) ?? null)
  const evidence = $derived(headRow?.failedAsserts[0] ?? null)
  // The time gutter holds the failure moment, or for a finished job its end.
  const endedAt = $derived(summary?.active ? null : rows.map(r => r.finishedAt).filter(Boolean).sort().at(-1) ?? null)
  const failureAt = $derived(headRow?.finishedAt ?? (summary?.failedPhase ? failedRow?.finishedAt : null) ?? endedAt)
  const phaseNode = $derived(focus.node && (networkNodes.includes(focus.node) || focus.node === ALL) ? focus.node : messageNode || (networkNodes.length > 1 ? ALL : networkNodes[0] ?? ''))

  // Each step of the failure trail scrolls to its evidence; the node also
  // switches the logs to it.
  let opened = $state({})
  const verdicts = run => { const c = run.summary?.counts; return c ? `통과 ${c.pass ?? 0} · 실패 ${c.fail ?? 0} · 막힘 ${c.blocked ?? 0} · 건너뜀 ${c.skip ?? 0}` : '' }
  function reveal(id) { document.getElementById(id)?.scrollIntoView({ block: 'start', behavior: 'smooth' }) }
  function showTest() { focus.test = head.test; focus.node = ''; focus.at = ''; reveal('cause') }
  function showNode() {
    if (head.test) focus.test = head.test
    focus.node = trailNode
    reveal('cause')
  }

  $effect(() => {
    const workspace = job?.workspaceId
    if (!workspace || !webSession) return
    client.request('networks', { decode: decodeNetworkPage }).then(page => { network = page.items.find(n => n.workspaceId === workspace) ?? null }).catch(() => { network = null })
  })
  // Runs are read again only when the job's list of run ids changes, not on
  // every live update of the job.
  const runKey = $derived((job?.runIds ?? []).join('\n'))
  $effect(() => {
    const ids = runKey ? runKey.split('\n') : []
    if (!ids.length || !webSession) { runs = []; runsLoaded = !!job; runsError = ''; return }
    runsLoaded = false
    const key = runKey
    // Raw text keeps large integers exact (parseHistoryJSON), as History does.
    // A run that cannot be read is named, not dropped without a word.
    const read = id => client.request('history/' + encodeURIComponent(id), { cache: 'no-store', parse: parseHistoryJSON, decode: decodeRun })
    Promise.all(ids.map(id => read(id).then(run => ({ run }), e => ({ failure: `${id}: ${e.message}` }))))
      .then(found => { if (key !== runKey) return; runs = found.map(f => f.run).filter(Boolean); runsError = found.map(f => f.failure).filter(Boolean).join(' · '); runsLoaded = true })
  })
  $effect(() => { void jobId; if (webSession) load() })
  onMount(() => feed.start())
</script>

<section class="page" aria-label="작업 상세">
  <p class="back"><a href="/chains" onclick={e => follow(e, '/chains')}>체인 화면의 작업 목록으로</a></p>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if job}
    <div class="folio"><span>작업 일지 · <strong>{job.operation}</strong> · <code>{job.id.slice(0, 8)}</code></span><span>{formatTime(job.createdAt).slice(0, 10)}{#if network} · 네트워크 <code>{network.id.slice(0, 8)}</code> · 노드 {networkNodes.length}{/if}</span></div>
    <CasualtyEntry {job} {head} {summary} {network} nodeCount={networkNodes.length} at={failureAt} {evidence} {duration} {elapsed} {stale} connection={live.connection} {trailNode} {finding} onphase={() => reveal('phases')} ontest={showTest} onnode={showNode} />

    <!-- A failing test run is the cause a reader came for, so it comes first. -->
    {#if firstRun}<RunBench run={firstRun} nodes={networkNodes} networkId={network?.id} bind:test={focus.test} bind:node={focus.node} bind:at={focus.at} bind:finding />{/if}
    {#if !runsLoaded && job.runIds?.length}<p class="muted">테스트 실행 기록을 불러오는 중…</p>{/if}
    {#if runsError}<p class="error" role="alert">테스트 실행 기록을 읽지 못했습니다. {runsError}</p>{/if}

    {#if summary.failedPhase && !failingRuns.length && !runsPending}
      <section class="cause" id="cause" aria-label="실패한 단계 시점 로그">
        {#if !network}<p class="muted">이 작업의 네트워크 기록이 없어 노드 로그를 열 수 없습니다.</p>
        {:else if !networkNodes.length}<p class="muted">네트워크에 기록된 노드가 없어 로그를 열 수 없습니다.</p>
        {:else}
          {#if !phaseSpan}<p class="notice">실패한 단계에 시각이 없어 최근 2시간의 로그를 엽니다.</p>{/if}
          <CauseBench networkId={network.id} nodes={networkNodes} bind:node={() => phaseNode, v => { focus.node = v }} bind:at={focus.at} phases={rows} from={phaseWindow.from} to={phaseWindow.to} start={failedRow?.startedAt} end={failedRow?.finishedAt} logTitle={`${phaseNode === ALL ? '모든 노드' : phaseNode} 로그 · ${summary.failedPhase.name} 단계 구간 · 오류·경고 먼저`} logTestid="phase-logs" />
        {/if}
      </section>
    {/if}

    {#if rows.length}<PhaseWatches {rows} />{/if}

    <!-- The job's other runs wait folded; each reads its logs and metrics only once opened. -->
    {#each [...failingRuns, ...passingRuns].filter(r => r !== firstRun) as run (run.id)}
      <details class="run" ontoggle={e => { if (e.currentTarget.open) opened = { ...opened, [run.id]: true } }}>
        <summary>테스트 실행 <code>{run.id}</code> <span class="muted">{verdicts(run)}</span></summary>
        {#if opened[run.id]}<RunBench {run} nodes={networkNodes} networkId={network?.id} anchor={'run-' + run.id} logTestid="run-logs" />{/if}
      </details>
    {/each}
    <details class="meta" open={!summary.failedPhase && !summary.failure && !failingRuns.length && !passingRuns.length}>
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
  .page { min-width: 0; }
  .back { margin: 0 0 var(--space-2); font-size: var(--text-sm); }
  @media (pointer: coarse) { .back a { display: inline-flex; align-items: center; min-height: 44px; } }
  .run { margin-top: var(--space-4); padding-top: var(--space-2); border-top: var(--rule-strong); }
  .run > summary { cursor: pointer; padding: var(--space-2) 0; font: 600 var(--text-sm) var(--font-cond); }
  .run > summary .muted { margin-left: var(--space-2); font: var(--text-xs) var(--font-mono); }
  .summary { display: grid; grid-template-columns: 9rem minmax(0, 1fr); gap: var(--space-2) var(--space-4); } dt { color: var(--text-muted); } dd { margin: 0; }
  .error-text { color: var(--danger); }
  .meta { margin-top: var(--space-5); padding-top: var(--space-3); border-top: var(--rule); } .meta summary { cursor: pointer; color: var(--text-muted); font: 600 var(--text-sm) var(--font-cond); }
  .note { margin: var(--space-3) 0 0; font-size: var(--text-xs); }
  @media (max-width: 700px) { .summary { grid-template-columns: 1fr; } }
</style>
