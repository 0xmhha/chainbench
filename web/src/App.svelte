<script>
  import { formatTime } from './labels.mjs'
  import { responseError } from './api-error.mjs'
  import { onMount } from 'svelte'
  import DSLEditor from './DSLEditor.svelte'
  import LoginPanel from './LoginPanel.svelte'
  import SettingsPanel from './SettingsPanel.svelte'
  import ChainPresetEditor from './ChainPresetEditor.svelte'
  import DeploymentEditor from './DeploymentEditor.svelte'
  import JobPanel from './JobPanel.svelte'
  import HistoryPanel from './HistoryPanel.svelte'
  import MonitorPanel from './MonitorPanel.svelte'
  import Dashboard from './Dashboard.svelte'
  import JobDetail from './JobDetail.svelte'
  import NodeDetail from './NodeDetail.svelte'
  import { parseRoute } from './routes.mjs'
  import { follow } from './nav.mjs'
  import { selection, monitorRange } from './selection.svelte.js'
  import { timeRange } from './metric-chart.mjs'
  import { webApi } from './web-api.mjs'

  const pages = [
    ['/', '대시보드', '전체 실행과 최근 결과를 확인합니다.'],
    ['/chains', '체인 · Workspace', '엔진 계약을 바탕으로 체인과 서버 배치를 구성합니다.'],
    ['/tests', '테스트', '현행 DSL을 구조 편집하고 실행 의미를 검증합니다.'],
    ['/monitoring', '모니터링', '서버가 전달한 이벤트와 연결 상태를 확인합니다.'],
    ['/history', '히스토리', '보관된 결과를 검색·비교하고 내보냅니다.'],
    ['/settings', '설정', '개인 자격증명, 사용자와 감사 기록, 등록 자료, 체인 매니페스트를 관리합니다.']
  ]
  let path = $state(window.location.pathname)
  const parsed = $derived(parseRoute(path))
  const route = $derived(parsed.page)
  let workspaceNames = $state({})
  const workspaceName = $derived(selection.workspaceId ? workspaceNames[selection.workspaceId] ?? selection.workspaceId : '')
  let webSession = $state(undefined), ready = $state(false)
  let assetRevision = $state(0)
  let catalogRevision = $state(0)
  let events = $state([]), runs = $state([]), sessions = $state([])
  let connected = $state(false), error = $state('')
  let drops = $state(null), observedAt = $state(null), now = $state(Date.now())
  const stale = $derived(observedAt !== null && now - Date.parse(observedAt) > 5000)
  const details = { '/jobs': ['/jobs', '작업 상세', '서버가 실행하는 작업 하나의 단계와 결과를 확인합니다.'], '/nodes': ['/nodes', '노드 상세', '노드 하나의 상태, 구성, 로그와 쓸 수 있는 제어를 확인합니다.'] }
  const current = $derived(details[route] ?? pages.find(p => p[0] === route) ?? pages[0])
  // The top bar names the selected workspace; names are read once per session.
  $effect(() => { if (ready && webSession) webApi(webSession)('workspaces?limit=100').then(page => { workspaceNames = Object.fromEntries(page.items.map(w => [w.id, w.name])) }).catch(() => {}) })
  onMount(() => {
    const change = () => { path = window.location.pathname; error = '' }
    window.addEventListener('popstate', change)
    const clock = setInterval(() => { now = Date.now() }, 1000)
    return () => { window.removeEventListener('popstate', change); clearInterval(clock) }
  })
  async function read(path) {
    const response = await fetch(path)
    if (!response.ok) throw await responseError(response)
    return response.json()
  }
  async function loadRuns() {
    try { runs = (await read('/api/runs')) ?? [] } catch (e) { error = e.message }
  }
  async function loadSessions() {
    try { sessions = ((await read('/api/sessions')) ?? []).map(id => ({ id })) } catch (e) { error = e.message }
  }
  $effect(() => {
    if (!ready || webSession === null) return
    loadRuns(); loadSessions()
    const es = new EventSource('/events')
    es.onopen = () => { connected = true }
    es.onerror = () => { connected = false }
    es.addEventListener('observation', m => {
      try { const metadata = JSON.parse(m.data); drops = metadata.busDeliveryDropsTotal; observedAt = metadata.observedAt } catch { error = '관측 상태를 해석할 수 없습니다.' }
    })
    es.onmessage = m => {
      try {
        const e = JSON.parse(m.data)
        events = [e, ...events].slice(0, 200)
        if (e.kind === 'result') { loadRuns(); loadSessions() }
      } catch { error = '해석할 수 없는 이벤트를 받았습니다.' }
    }
    return () => { es.close(); connected = false }
  })
  // The monitoring range also narrows the event timeline.
  const range = $derived(timeRange(monitorRange, now))
  const rangedEvents = $derived(range ? events.filter(e => { const t = Date.parse(e.time); return !Number.isFinite(t) || (t >= Date.parse(range.from) && t <= Date.parse(range.to)) }) : events)
  const fmtTime = formatTime
  const runOk = s => s === 'success' || s === 'pass'
</script>

<svelte:head><title>{current[1]} · chainbench</title><meta name="theme-color" content="#e6ece4" media="(prefers-color-scheme: light)" /><meta name="theme-color" content="#131a15" media="(prefers-color-scheme: dark)" /></svelte:head>
<a class="skip-link" href="#content">본문으로 이동</a>
<div class="shell">
  <aside aria-label="주 메뉴">
    <a class="brand" href="/" onclick={e => follow(e, '/')}><span class="brand-symbol" aria-hidden="true">cb</span>chainbench</a>
    <nav>{#each pages as [path, label]}<a href={path} aria-current={route === path ? 'page' : undefined} onclick={e => follow(e, path)}>{label}</a>{/each}</nav>
  </aside>
  <div class="workspace">
    <header class="topbar">
      <span class="context" data-testid="topbar-context">
        <span><span class="label">Workspace</span> {workspaceName || '선택 안 함'}</span>
        {#if selection.chain}<span><span class="label">체인</span> {selection.chain}</span>{:else if route === '/chains' || route === '/tests'}<a href="/chains" onclick={e => follow(e, '/chains')}>체인을 선택하세요</a>{:else}<a class="quiet" href="/chains" onclick={e => follow(e, '/chains')}>체인을 선택하세요</a>{/if}
      </span>
      <span class="status">
        <span class="signal-label" title="이벤트 서버 연결 상태입니다. 노드 상태와는 별개입니다."><span class="signal" class:on={connected} aria-hidden="true"></span>{connected ? '서버 연결됨' : '서버 미연결'}</span>
        <span class="account">{webSession?.accountsEnabled === false ? '로컬 사용' : webSession ? `${webSession.mode} · ${webSession.user.username} · ${webSession.user.role}` : '로그인 필요'}</span>
      </span>
    </header>
    <main id="content">
      <header class="page-heading"><h1>{current[1]}</h1><p>{current[2]}</p></header>
      <LoginPanel bind:session={webSession} bind:ready />
      {#if error}<p class="error" role="alert">{error}</p>{/if}
      {#if ready && webSession !== null}
        {#key webSession?.user.id}
          <div hidden={route !== '/chains'}><ChainPresetEditor {webSession} {assetRevision} onsaved={()=>catalogRevision++} /><DeploymentEditor {webSession} onsaved={()=>catalogRevision++} /></div>
          <div hidden={route !== '/tests'}><DSLEditor {webSession} {assetRevision} onsaved={()=>catalogRevision++} /></div>
          {#if route==='/chains'||route==='/tests'}{#key route}<JobPanel {webSession} catalogRevision={catalogRevision+assetRevision} testOnly={route==='/tests'} />{/key}{/if}
          <div hidden={route !== '/settings'}><SettingsPanel {webSession} {assetRevision} onuploaded={()=>assetRevision++} /></div>
          <div hidden={route !== '/history'}><HistoryPanel {webSession} /></div>
        {/key}
        {#if route === '/'}
          <Dashboard {runs} busDrops={drops} {connected} onrefresh={loadRuns} />
        {:else if route === '/jobs'}
          {#key parsed.jobId}<JobDetail jobId={parsed.jobId} {webSession} />{/key}
        {:else if route === '/nodes'}
          {#key parsed.networkId + parsed.nodeId}<NodeDetail networkId={parsed.networkId} nodeId={parsed.nodeId} {webSession} />{/key}
        {:else if route === '/monitoring'}
          {#key webSession?.user.id}<MonitorPanel {webSession} />{/key}
          <section class="panel"><h2>이벤트 타임라인</h2><p class="muted">현재 연결에서 수신한 최근 200건입니다. 연결이 끊긴 동안의 이벤트와 전체 기록을 보장하지 않습니다.</p>
            <p data-testid="observation-health">서버 버스 전체 전달 드롭: {drops ?? '확인 중'} · 마지막 관측: {fmtTime(observedAt)}{stale ? ' · stale' : ''}</p>
            {#if drops > 0}<p class="notice">전체 구독자에서 전달 누락이 발생했습니다. 이 값은 현재 브라우저만의 누락 수가 아닙니다.</p>{/if}
            {#if !connected}<p class="notice">이벤트 연결이 끊겼습니다. 마지막 관측 내용을 표시합니다.</p>{/if}
            <p class="muted" data-testid="ranged-events">선택한 시간 범위의 이벤트 {rangedEvents.length}건 (이 연결에서 받은 최근 {events.length}건 중)</p>
            {#if rangedEvents.length === 0}<p class="empty">이 범위에 수신한 이벤트가 없습니다.</p>{:else}<ul class="events">{#each rangedEvents as e, i (i)}<li><time>{fmtTime(e.time)}</time><span class="kind">{e.phase} · {e.kind}</span><span class="mono">{e.network ?? ''} {e.node ? `node${e.node}` : ''}</span><span class="msg">{e.message}</span></li>{/each}</ul>{/if}
          </section>
        {/if}
      {/if}
    </main>
  </div>
</div>

<style>
  .shell { display: grid; grid-template-columns: 200px minmax(0, 1fr); min-height: 100vh; }
  aside { min-width: 0; position: sticky; top: 0; height: 100vh; display: flex; flex-direction: column; padding: var(--space-4); }
  .brand { display: flex; align-items: center; gap: var(--space-2); padding: var(--space-1) 0 var(--space-3); border-bottom: var(--rule-strong); color: var(--text); font: 700 var(--text-md) var(--font-cond); letter-spacing: .08em; }
  .brand:hover { text-decoration: none; }
  .brand-symbol { display: grid; place-items: center; width: 26px; height: 26px; border: 1.5px solid var(--danger); color: var(--danger); font: 700 var(--text-sm) var(--font-mono); letter-spacing: 0; }
  nav { display: grid; margin-top: var(--space-3); counter-reset: page; }
  nav a { display: flex; justify-content: space-between; padding: var(--space-2) 0; border-bottom: var(--rule); color: var(--text-muted); counter-increment: page; }
  nav a::after { content: counter(page, decimal-leading-zero); font: var(--text-xs) var(--font-mono); color: var(--text-faint); }
  nav a:hover { color: var(--text); text-decoration: none; }
  nav a[aria-current] { color: var(--text); font-weight: 700; border-bottom: 1px solid var(--text); }
  .workspace { min-width: 0; border-left: var(--rule-strong); }
  .topbar { position: sticky; top: 0; z-index: 5; display: flex; align-items: center; justify-content: space-between; gap: var(--space-4); min-height: 44px; padding: var(--space-2) var(--space-5); background: var(--bg); border-bottom: var(--rule-strong); font-size: var(--text-sm); color: var(--text-muted); }
  .context, .status { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-4); min-width: 0; }
  .context .label { color: var(--text-faint); }
  .context a { color: var(--warn); } .context a.quiet { color: var(--text-muted); }
  .signal { display: inline-block; width: 7px; height: 7px; margin-right: var(--space-2); background: var(--danger); }
  .signal.on { background: var(--ok); }
  main { max-width: 1440px; margin: 0 auto; padding: var(--space-4) var(--space-5) var(--space-5); }
  .page-heading { display: flex; flex-wrap: wrap; align-items: baseline; gap: var(--space-2) var(--space-4); }
  .page-heading h1 { font: 700 var(--text-xl) var(--font-cond); letter-spacing: .01em; }
  .page-heading p { margin: 0; color: var(--text-muted); font-size: var(--text-sm); }
  .events { list-style: none; padding: 0; margin: 0; }
  .events li { display: grid; grid-template-columns: 170px 110px 110px minmax(0, 1fr); gap: var(--space-3); padding: var(--space-2) 0; border-bottom: 1px solid var(--border-subtle); align-items: start; }
  time, .kind { color: var(--text-muted); font-size: var(--text-xs); font-variant-numeric: tabular-nums; }
  @media (max-width: 800px) {
    .shell { grid-template-columns: minmax(0, 1fr); }
    .workspace { border-left: 0; }
    aside { position: static; height: auto; padding: var(--space-2) var(--space-4); border-right: 0; border-bottom: 1px solid var(--border-subtle); }
    nav { display: flex; gap: var(--space-4); margin-top: var(--space-2); overflow-x: auto; scrollbar-width: none; scroll-snap-type: x proximity; mask-image: linear-gradient(to right, black 85%, transparent); }
    nav a { white-space: nowrap; scroll-snap-align: start; }
    .topbar { position: static; padding: var(--space-2) var(--space-4); flex-direction: column; align-items: flex-start; gap: var(--space-1); }
    main { padding: var(--space-3) var(--space-4); }
    .events li { grid-template-columns: 1fr; gap: 2px; }
  }
</style>
