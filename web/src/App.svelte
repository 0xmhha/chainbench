<script>
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
  const fmtTime = t => t ? new Date(t).toLocaleString() : '—'
  const runOk = s => s === 'success' || s === 'pass'
</script>

<svelte:head><title>{current[1]} · chainbench</title><meta name="theme-color" content="#0b1018" /></svelte:head>
<a class="skip" href="#content">본문으로 이동</a>
<div class="shell">
  <aside aria-label="주 메뉴">
    <a class="brand" href="/" onclick={e => follow(e, '/')}><span class="brand-symbol">cb</span><span>chainbench<small>CHAIN OPERATIONS</small></span></a>
    <nav>{#each pages as [path, label], i}<a href={path} aria-current={route === path ? 'page' : undefined} onclick={e => follow(e, path)}><span class="nav-number" aria-hidden="true">0{i + 1}</span>{label}</a>{/each}</nav>
    <div class="nav-footer"><span class="signal" class:on={connected}></span>{connected ? '이벤트 서버 연결됨' : '이벤트 서버 미연결'}<small>연결 표시는 노드 상태와 별개입니다.</small></div>
  </aside>
  <div class="workspace">
    <header class="topbar"><span class="breadcrumb">운영 콘솔 <span>/</span> {current[1]}</span>
      <span class="context" data-testid="topbar-context">
        <span>Workspace {workspaceName || '선택 안 함'}</span>
        {#if selection.chain}<span>체인 {selection.chain}</span>{:else}<a href="/chains" onclick={e => follow(e, '/chains')}>체인을 선택하세요</a>{/if}
        <span class="signal-label"><span class="signal" class:on={connected} aria-hidden="true"></span>{connected ? '서버 연결됨' : '서버 미연결'}</span>
      </span>
      <span class="account">{webSession ? `${webSession.mode} · ${webSession.user.username} · ${webSession.user.role}` : '로그인 필요'}</span></header>
    <main id="content">
      <div class="page-heading"><div><p class="eyebrow">CHAINBENCH WORKSPACE</p><h1>{current[1]}</h1><p>{current[2]}</p></div><span class="connection" class:on={connected}>{connected ? '연결됨' : '미연결'}</span></div>
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
  :global(:root) { --bg: #0b1018; --sidebar: #0e1520; --surface: #121b28; --surface-raised: #192536; --border: #334155; --border-subtle: #253247; --border-strong: #475569; --text: #f1f5f9; --text-muted: #a8b5c8; --text-faint: #475569; --accent: #93c5fd; --accent-strong: #dbeafe; --ok: #6ee7b7; --ok-border: #295344; --warn: #fcd34d; --warn-bg: #30251a; --danger: #fca5a5; --danger-bg: #301d25; --danger-border: #5a343a; --accent-border: #47668b; }
  :global(*) { box-sizing: border-box; }
  :global(body) { margin: 0; font: 14px/1.55 -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: var(--bg); color: var(--text); }
  :global(button), :global(input), :global(select), :global(textarea) { min-height: 40px; font: inherit; }
  :global(button) { cursor: pointer; } :global(button:disabled) { cursor: default; }
  :global(:focus-visible) { outline: 2px solid var(--accent); outline-offset: 3px; }
  :global([hidden]) { display: none !important; }
  :global(fieldset), :global(section) { min-width: 0; }
  :global(p), :global(code) { overflow-wrap: anywhere; }
  .workspace { min-width: 0; }
  aside { min-width: 0; }
  a { color: var(--accent); text-decoration: none; } a:hover { text-decoration: underline; }
  .shell { display: grid; grid-template-columns: 224px minmax(0, 1fr); min-height: 100vh; }
  aside { background: var(--sidebar); border-right: 1px solid var(--border-subtle); display: flex; flex-direction: column; padding: 24px 16px; position: sticky; top: 0; height: 100vh; }
  .brand { display: flex; gap: 10px; align-items: center; color: var(--text); font-size: 19px; font-weight: 650; padding: 0 8px; }
  .brand-symbol { display: grid; place-items: center; border: 1px solid var(--accent-border); border-radius: 10px; width: 36px; height: 36px; color: var(--accent); font-size: 16px; }
  .brand small { display: block; color: var(--text-muted); font-size: 9px; font-weight: 400; letter-spacing: .13em; }
  nav { display: grid; gap: 6px; margin-top: 40px; } nav a { padding: 12px; color: var(--text-muted); border-radius: 8px; } nav a[aria-current] { background: var(--surface-raised); color: var(--accent-strong); } .nav-number { margin-right: 12px; font-size: 11px; opacity: .65; }
  .nav-footer { margin-top: auto; padding: 16px 8px; color: var(--text-muted); font-size: 12px; } .nav-footer small { display: block; margin-top: 8px; }
  .signal { display: inline-block; width: 6px; height: 6px; border-radius: 50%; background: var(--danger); margin-right: 8px; } .signal.on { background: var(--ok); }
  .topbar { height: 64px; border-bottom: 1px solid var(--border-subtle); padding: 0 24px; display: flex; align-items: center; justify-content: space-between; gap: 16px; color: var(--text-muted); } .breadcrumb span { padding: 0 12px; color: var(--border-strong); }
  .context { display: flex; flex-wrap: wrap; gap: 16px; align-items: center; font-size: 12px; } .context a { color: var(--warn); } .account { font-size: 12px; }
  main { max-width: 1440px; margin: auto; padding: 32px 24px; }
  .page-heading { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
  .page-heading { margin-bottom: 24px; } .page-heading p { margin: 8px 0 0; color: var(--text-muted); } .page-heading .eyebrow { color: var(--accent); font-size: 10px; letter-spacing: .14em; } h1 { margin: 8px 0 0; font-size: 26px; letter-spacing: -.03em; } h2 { font-size: 18px; margin: 0 0 12px; }
  .connection { border: 1px solid var(--danger-border); border-radius: 20px; padding: 4px 12px; font-size: 12px; color: var(--danger); } .connection.on { color: var(--ok); border-color: var(--ok-border); }
  .panel { border: 1px solid var(--border); border-radius: 10px; background: var(--surface); padding: 24px; margin-top: 24px; }
  .empty { padding: 32px 8px; color: var(--text-muted); text-align: center; } .mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
  .events { list-style: none; padding: 0; margin: 0; } .events li { display: grid; grid-template-columns: 160px 100px 100px minmax(0, 1fr); gap: 12px; padding: 12px 0; border-bottom: 1px solid var(--border-subtle); align-items: start; } time, .kind { color: var(--text-muted); font-size: 12px; } .msg { overflow-wrap: anywhere; } .muted { color: var(--text-muted); } .notice, .error { padding: 12px 16px; background: var(--warn-bg); border-left: 3px solid var(--warn); color: var(--warn); } .error { border-color: var(--danger); color: var(--danger); background: var(--danger-bg); }
  .skip { position: absolute; left: 12px; top: -80px; z-index: 10; padding: 12px; background: var(--surface-raised); } .skip:focus { top: 12px; }
  @media(max-width: 800px) { .shell { grid-template-columns: minmax(0, 1fr); } aside { height: auto; position: static; padding: 12px 16px; border-right: 0; border-bottom: 1px solid var(--border-subtle); } nav { margin-top: 16px; display: flex; overflow-x: auto; } nav a { white-space: nowrap; padding: 8px 12px; } .nav-number, .nav-footer { display: none; } .topbar { padding: 0 16px; } main { padding: 24px 16px; } .events li { grid-template-columns: 1fr; gap: 4px; } .account { max-width: 50%; overflow-wrap: anywhere; } }
</style>
