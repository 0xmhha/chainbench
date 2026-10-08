<script>
  import { onMount } from 'svelte'
  import DSLEditor from './DSLEditor.svelte'
  import LoginPanel from './LoginPanel.svelte'
  import ManifestManager from './ManifestManager.svelte'
  import AssetManager from './AssetManager.svelte'
  import ChainPresetEditor from './ChainPresetEditor.svelte'
  import DeploymentEditor from './DeploymentEditor.svelte'
  import JobPanel from './JobPanel.svelte'
  import HistoryPanel from './HistoryPanel.svelte'

  const pages = [
    ['/', '대시보드', '전체 실행과 최근 결과를 확인합니다.'],
    ['/chains', '체인 · Workspace', '엔진 계약을 바탕으로 체인과 서버 배치를 구성합니다.'],
    ['/tests', '테스트', '현행 DSL을 구조 편집하고 실행 의미를 검증합니다.'],
    ['/monitoring', '모니터링', '서버가 전달한 이벤트와 연결 상태를 확인합니다.'],
    ['/history', '히스토리', '보관된 결과를 검색·비교하고 내보냅니다.'],
    ['/settings', '설정', '계정 권한과 체인 매니페스트를 관리합니다.']
  ]
  let route = $state(window.location.pathname)
  let webSession = $state(undefined), ready = $state(false)
  let assetRevision = $state(0)
  let events = $state([]), runs = $state([]), sessions = $state([])
  let connected = $state(false), error = $state('')
  let drops = $state(null), observedAt = $state(null), now = $state(Date.now())
  const stale = $derived(observedAt !== null && now - Date.parse(observedAt) > 5000)
  const current = $derived(pages.find(p => p[0] === route) ?? pages[0])

  function navigate(event, path) {
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
    event.preventDefault()
    history.pushState({}, '', path); route = path; error = ''
  }
  onMount(() => {
    const change = () => { route = window.location.pathname }
    window.addEventListener('popstate', change)
    const clock = setInterval(() => { now = Date.now() }, 1000)
    return () => { window.removeEventListener('popstate', change); clearInterval(clock) }
  })
  async function read(path) {
    const response = await fetch(path)
    if (!response.ok) throw new Error(`요청 실패 (${response.status})`)
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
  const fmtTime = t => t ? new Date(t).toLocaleString() : '—'
  const runOk = s => s === 'success' || s === 'pass'
</script>

<svelte:head><title>{current[1]} · chainbench</title><meta name="theme-color" content="#0b1018" /></svelte:head>
<a class="skip" href="#content">본문으로 이동</a>
<div class="shell">
  <aside aria-label="주 메뉴">
    <a class="brand" href="/" onclick={e => navigate(e, '/')}><span class="brand-symbol">cb</span><span>chainbench<small>CHAIN OPERATIONS</small></span></a>
    <nav>{#each pages as [path, label], i}<a href={path} aria-current={route === path ? 'page' : undefined} onclick={e => navigate(e, path)}><span class="nav-number" aria-hidden="true">0{i + 1}</span>{label}</a>{/each}</nav>
    <div class="nav-footer"><span class="signal" class:on={connected}></span>{connected ? '이벤트 서버 연결됨' : '이벤트 서버 미연결'}<small>연결 표시는 노드 상태와 별개입니다.</small></div>
  </aside>
  <div class="workspace">
    <header class="topbar"><span class="breadcrumb">운영 콘솔 <span>/</span> {current[1]}</span><span class="account">{webSession ? `${webSession.mode} · ${webSession.user.username} · ${webSession.user.role}` : '로그인 필요'}</span></header>
    <main id="content">
      <div class="page-heading"><div><p class="eyebrow">CHAINBENCH WORKSPACE</p><h1>{current[1]}</h1><p>{current[2]}</p></div><span class="connection" class:on={connected}>{connected ? '연결됨' : '미연결'}</span></div>
      <LoginPanel bind:session={webSession} bind:ready />
      {#if error}<p class="error" role="alert">{error}</p>{/if}
      {#if ready && webSession !== null}
        {#key webSession?.user.id}
          <div hidden={route !== '/chains'}><ChainPresetEditor {webSession} /><DeploymentEditor {webSession} /></div>
          <div hidden={route !== '/tests'}><DSLEditor {webSession} /></div>
          {#if route==='/chains'||route==='/tests'}{#key route}<JobPanel {webSession} testOnly={route==='/tests'} />{/key}{/if}
          <div hidden={route !== '/settings'}><AssetManager {webSession} onuploaded={()=>assetRevision++} /><ManifestManager {webSession} {assetRevision} /></div>
          <div hidden={route !== '/history'}><HistoryPanel {webSession} /></div>
        {/key}
        {#if route === '/'}
          <div class="overview">
            <article><span>관측된 실행</span><strong>{runs.length}</strong><small>현재 서버의 실행 인덱스</small></article>
            <article><span>보관된 세션</span><strong>{sessions.length}</strong><small>설정된 artifact root</small></article>
            <article><span>최근 이벤트</span><strong>{events.length}</strong><small>이 브라우저에서 최근 200건까지</small></article>
          </div>
          <section class="panel"><div class="panel-heading"><h2>최근 실행</h2><button onclick={loadRuns}>새로고침</button></div>
            {#if runs.length === 0}<div class="empty"><h3>관측된 실행이 없습니다</h3><p>체인 구성과 테스트 정의를 준비하세요.</p><a href="/chains" onclick={e => navigate(e, '/chains')}>체인 구성 열기 →</a></div>{:else}
              <div class="table-wrap"><table><thead><tr><th>실행 ID</th><th>단계</th><th>체인</th><th>네트워크</th><th>판정</th></tr></thead><tbody>{#each runs as r (r.id)}<tr><td class="mono">{r.id}</td><td>{r.phase}</td><td>{r.chain}</td><td>{r.network}</td><td class:ok={runOk(r.status)}>{r.status}</td></tr>{/each}</tbody></table></div>
            {/if}
          </section>
        {:else if route === '/monitoring'}
          <section class="panel"><h2>이벤트 타임라인</h2><p class="muted">현재 연결에서 수신한 최근 200건입니다. 연결이 끊긴 동안의 이벤트와 전체 기록을 보장하지 않습니다.</p>
            <p data-testid="observation-health">서버 버스 전체 전달 드롭: {drops ?? '확인 중'} · 마지막 관측: {fmtTime(observedAt)}{stale ? ' · stale' : ''}</p>
            {#if drops > 0}<p class="notice">전체 구독자에서 전달 누락이 발생했습니다. 이 값은 현재 브라우저만의 누락 수가 아닙니다.</p>{/if}
            {#if !connected}<p class="notice">이벤트 연결이 끊겼습니다. 마지막 관측 내용을 표시합니다.</p>{/if}
            {#if events.length === 0}<p class="empty">아직 수신한 이벤트가 없습니다.</p>{:else}<ul class="events">{#each events as e, i (i)}<li><time>{fmtTime(e.time)}</time><span class="kind">{e.phase} · {e.kind}</span><span class="mono">{e.network ?? ''} {e.node ? `node${e.node}` : ''}</span><span class="msg">{e.message}</span></li>{/each}</ul>{/if}
          </section>
        {/if}
      {/if}
    </main>
  </div>
</div>

<style>
  :global(*) { box-sizing: border-box; }
  :global(body) { margin: 0; font: 14px/1.55 -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0b1018; color: #f1f5f9; }
  :global(button), :global(input), :global(select), :global(textarea) { min-height: 40px; font: inherit; }
  :global(button) { cursor: pointer; } :global(button:disabled) { cursor: default; }
  :global(:focus-visible) { outline: 2px solid #93c5fd; outline-offset: 3px; }
  :global([hidden]) { display: none !important; }
  :global(fieldset), :global(section) { min-width: 0; }
  :global(p), :global(code) { overflow-wrap: anywhere; }
  .workspace { min-width: 0; }
  aside { min-width: 0; }
  a { color: #93c5fd; text-decoration: none; } a:hover { text-decoration: underline; }
  .shell { display: grid; grid-template-columns: 224px minmax(0, 1fr); min-height: 100vh; }
  aside { background: #0e1520; border-right: 1px solid #253247; display: flex; flex-direction: column; padding: 24px 16px; position: sticky; top: 0; height: 100vh; }
  .brand { display: flex; gap: 10px; align-items: center; color: #f1f5f9; font-size: 19px; font-weight: 650; padding: 0 8px; }
  .brand-symbol { display: grid; place-items: center; border: 1px solid #47668b; border-radius: 10px; width: 36px; height: 36px; color: #93c5fd; font-size: 16px; }
  .brand small { display: block; color: #a8b5c8; font-size: 9px; font-weight: 400; letter-spacing: .13em; }
  nav { display: grid; gap: 6px; margin-top: 40px; } nav a { padding: 12px; color: #a8b5c8; border-radius: 8px; } nav a[aria-current] { background: #192536; color: #dbeafe; } .nav-number { margin-right: 12px; font-size: 11px; opacity: .65; }
  .nav-footer { margin-top: auto; padding: 16px 8px; color: #a8b5c8; font-size: 12px; } .nav-footer small { display: block; margin-top: 8px; }
  .signal { display: inline-block; width: 6px; height: 6px; border-radius: 50%; background: #fca5a5; margin-right: 8px; } .signal.on { background: #6ee7b7; }
  .topbar { height: 64px; border-bottom: 1px solid #253247; padding: 0 24px; display: flex; align-items: center; justify-content: space-between; gap: 16px; color: #a8b5c8; } .breadcrumb span { padding: 0 12px; color: #475569; } .account { font-size: 12px; }
  main { max-width: 1440px; margin: auto; padding: 32px 24px; }
  .page-heading, .panel-heading { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
  .page-heading { margin-bottom: 24px; } .page-heading p { margin: 8px 0 0; color: #a8b5c8; } .page-heading .eyebrow { color: #93c5fd; font-size: 10px; letter-spacing: .14em; } h1 { margin: 8px 0 0; font-size: 26px; letter-spacing: -.03em; } h2 { font-size: 18px; margin: 0 0 12px; } h3 { font-size: 16px; }
  .connection { border: 1px solid #5a343a; border-radius: 20px; padding: 4px 12px; font-size: 12px; color: #fca5a5; } .connection.on { color: #6ee7b7; border-color: #295344; }
  .overview { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin: 24px 0; } .overview article, .panel { border: 1px solid #334155; border-radius: 10px; background: #121b28; padding: 24px; }
  .overview article { display: grid; gap: 8px; color: #a8b5c8; } .overview strong { color: #f1f5f9; font-size: 32px; font-weight: 550; } .overview small { font-size: 12px; }
  .panel { margin-top: 24px; } .panel-heading h2 { margin: 0; } button { color: #dbeafe; background: #192536; border: 1px solid #475569; border-radius: 6px; padding: 8px 12px; }
  .empty { padding: 32px 8px; color: #a8b5c8; text-align: center; } .table-wrap { overflow: auto; margin-top: 16px; } table { width: 100%; border-collapse: collapse; } th, td { text-align: left; padding: 12px; border-bottom: 1px solid #253247; } th { color: #a8b5c8; font-weight: 500; font-size: 12px; } .mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; } .ok { color: #6ee7b7; }
  .events { list-style: none; padding: 0; margin: 0; } .events li { display: grid; grid-template-columns: 160px 100px 100px minmax(0, 1fr); gap: 12px; padding: 12px 0; border-bottom: 1px solid #253247; align-items: start; } time, .kind { color: #a8b5c8; font-size: 12px; } .msg { overflow-wrap: anywhere; } .muted { color: #a8b5c8; } .notice, .error { padding: 12px 16px; background: #30251a; border-left: 3px solid #fcd34d; color: #fcd34d; } .error { border-color: #fca5a5; color: #fca5a5; background: #301d25; }
  .skip { position: absolute; left: 12px; top: -80px; z-index: 10; padding: 12px; background: #192536; } .skip:focus { top: 12px; }
  @media(max-width: 800px) { .shell { grid-template-columns: minmax(0, 1fr); } aside { height: auto; position: static; padding: 12px 16px; border-right: 0; border-bottom: 1px solid #253247; } nav { margin-top: 16px; display: flex; overflow-x: auto; } nav a { white-space: nowrap; padding: 8px 12px; } .nav-number, .nav-footer { display: none; } .topbar { padding: 0 16px; } main { padding: 24px 16px; } .overview { grid-template-columns: 1fr; gap: 8px; } .events li { grid-template-columns: 1fr; gap: 4px; } .account { max-width: 50%; overflow-wrap: anywhere; } }
</style>
