<script>
  import { formatTime } from './labels.mjs'
  import StatusBadge from './StatusBadge.svelte'
  import HistoryDetail from './HistoryDetail.svelte'
  import { apiClient } from './api-client.mjs'
  import { decodeRun, decodeRunPage } from './api-decode.mjs'
  import { allItems } from './pages.mjs'
  import { untrack } from 'svelte'
  import { parseHistoryJSON } from './history-json.js'
  import { readFocus, focusSearch } from './job-view.mjs'
  // active is true while History is the shown page; the panel stays mounted
  // (hidden) on the others, so it reads and writes the URL only while shown.
  let { webSession, active = false } = $props()
  let items = $state([]), next = $state(null), detail = $state(null), comparison = $state(null)
  let selected = $state([]), busy = $state(false), error = $state('')
  let detailBox = $state()
  // The open run and what is chosen in it live in the URL, so a link from the
  // dashboard or a copied address opens the same record at the same moment.
  const focus = $state(readFocus(''))
  $effect(() => {
    if (!active || !webSession) return
    untrack(() => {
      const wanted = readFocus(location.search)
      if (wanted.run && wanted.run !== detail?.id) { Object.assign(focus, wanted); open(wanted.run, true) }
    })
  })
  $effect(() => {
    if (!active) return
    const search = focusSearch(focus, location.search)
    if (search !== location.search) history.replaceState(history.state, '', location.pathname + search)
  })
  let search = $state(new URLSearchParams(window.location.search).get('search')??''), chain = $state(''), workspaceId = $state(''), actorId = $state(''), state = $state(''), caseId = $state('')
  let from = $state(''), to = $state(''), workspaces = $state([]), knownChains = $state([]), knownActors = $state([]), knownCases = $state([])
  const administrator = $derived(webSession?.user.role === 'administrator')
  const format = time => time ? formatTime(time) : '자료 없음'
  const client = $derived(apiClient(webSession))
  // History keeps large integers exact: the text is read by parseHistoryJSON.
  const runRead = decode => ({ parse: parseHistoryJSON, decode })
  function remember(rows) {
    knownChains = [...new Set([...knownChains, ...rows.map(r => r.chain).filter(Boolean)])].sort()
    knownActors = [...new Set([...knownActors, ...rows.map(r => r.actorId).filter(Boolean)])].sort()
    knownCases = [...new Set([...knownCases, ...rows.flatMap(r => (r.summary.tests ?? []).map(t => t.id))])].sort()
  }
  async function load(cursor = '') {
    busy = true; error = ''
    try {
      const query = new URLSearchParams({ limit: '50' })
      for (const [key, value] of Object.entries({ search, chain, workspaceId, actorId, state, caseId, cursor })) if (value) query.set(key, value)
      if (from) query.set('from', new Date(from).toISOString())
      if (to) query.set('to', new Date(to).toISOString())
      const page = await client.request('history?' + query, runRead(decodeRunPage))
      items = cursor ? [...items, ...page.items] : page.items; next = page.nextCursor
      remember(page.items)
    } catch (e) { error = e.message } finally { busy = false }
  }
  $effect(() => {
    if (!webSession) return
    untrack(() => {
      load()
      allItems(p => client.request(p), 'workspaces').then(items => { workspaces = items }).catch(e => { error = e.message })
    })
  })
  function select(id, checked) {
    selected = checked ? [...selected, id] : selected.filter(value => value !== id)
    comparison = null
  }
  // kept is true when the URL named the run, so its test, node and moment hold.
  async function open(id, kept = false) {
    busy = true; error = ''
    if (!kept) Object.assign(focus, { run: id, test: '', node: '', at: '' })
    try { detail = await client.request('history/' + encodeURIComponent(id), runRead(decodeRun)) } catch (e) { error = `실행 기록 ${id}을(를) 열지 못했습니다. ${e.message}`; focus.run = '' } finally { busy = false }
    // The detail opens below the list; bring it into view so the failure is what the reader sees.
    requestAnimationFrame(() => detailBox?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
  }
  const seconds = value => value == null ? '자료 없음' : `${Math.round(value * 10) / 10}초`
  const metricValue = value => value ? `${value.samples}개 · 최소 ${value.min ?? '—'} · 최대 ${value.max ?? '—'} · 마지막 ${value.last ?? '—'}` : '표본 없음'
  async function compare() {
    busy = true; error = ''
    try { comparison = await client.request('history/compare', { method: 'POST', body: { runIds: selected } }) } catch (e) { error = e.message } finally { busy = false }
  }
  async function download(id) {
    busy = true; error = ''
    try {
      const response = await client.send('history/' + encodeURIComponent(id) + '/export')
      const url = URL.createObjectURL(await response.blob()), link = document.createElement('a')
      link.href = url; link.download = id + '.json'; link.click(); setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch (e) { error = e.message } finally { busy = false }
  }
  // A refusal is thrown back to the detail, which says it beside the button.
  async function remove(run) {
    busy = true
    try {
      await client.request('history/' + encodeURIComponent(run.id), { method: 'DELETE' })
      selected = selected.filter(id => id !== run.id)
      detail = null; comparison = null
      Object.assign(focus, { run: '', test: '', node: '', at: '' })
      await load()
    } finally { busy = false }
  }
</script>

<section class="panel history-panel" aria-label="실행 히스토리">
  <header><div><h2>실행 기록</h2><p>서버에 보관된 결과 사본입니다. 과거 기록에서 확인할 수 없는 값은 자료 없음으로 표시합니다.</p></div><button disabled={busy} onclick={() => load()}>히스토리 새로고침</button></header>
  <form onsubmit={e => { e.preventDefault(); selected = []; comparison = null; load() }}>
    <label class="search">기록 검색<input name="search" aria-label="히스토리 검색" bind:value={search} placeholder="실행 ID, 명령, 테스트 이름" /></label>
    <label>체인<select name="chain" aria-label="히스토리 체인" bind:value={chain}><option value="">모든 체인</option>{#each knownChains as value}<option>{value}</option>{/each}</select></label>
    <label>Workspace<select name="workspace-id" aria-label="히스토리 Workspace" bind:value={workspaceId}><option value="">모든 Workspace</option>{#each workspaces as w}<option value={w.id}>{w.name}</option>{/each}</select></label>
    <label>실행자<select name="actor-id" aria-label="히스토리 실행자" bind:value={actorId}><option value="">모든 실행자</option>{#each knownActors as value}<option>{value}</option>{/each}</select></label>
    <label>상태<select name="state" aria-label="히스토리 상태" bind:value={state}><option value="">모든 상태</option>{#each ['accepted', 'running', 'cancelling', 'succeeded', 'failed', 'cancelled', 'interrupted', 'unknown'] as value}<option>{value}</option>{/each}</select></label>
    <label>테스트 케이스<select name="case-id" aria-label="히스토리 케이스" bind:value={caseId}><option value="">모든 케이스</option>{#each knownCases as value}<option>{value}</option>{/each}</select></label>
    <label>시작 이후<input name="from" aria-label="히스토리 시작 시각" type="datetime-local" bind:value={from} /></label><label>시작 이전<input name="to" aria-label="히스토리 종료 시각" type="datetime-local" bind:value={to} /></label>
    <button type="submit" disabled={busy}>필터 적용</button>
  </form>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if busy}<p role="status">기록을 확인하는 중…</p>{/if}
  <div class="toolbar"><span>{items.length}개 표시 · 비교는 2–4개 선택</span><button disabled={busy || selected.length < 2} onclick={compare}>선택한 실행 비교</button></div>
  {#if items.length === 0}<p class="empty">조건에 맞는 실행 기록이 없습니다.</p>{:else}
    <div class="table-wrap"><table><thead><tr><th>비교</th><th>실행</th><th>체인 / Workspace</th><th>시작 시각</th><th>상태</th><th>결과</th></tr></thead><tbody>{#each items as run (run.id)}<tr>
      <td><input name="compare" value={run.id} type="checkbox" aria-label={`비교 선택 ${run.id}`} checked={selected.includes(run.id)} disabled={!selected.includes(run.id) && selected.length === 4} onchange={e => select(run.id, e.currentTarget.checked)} /></td>
      <td><button class="record" onclick={() => open(run.id)}>{run.summary.operation ?? run.summary.command ?? run.id}<small>{run.id}</small></button></td>
      <td>{run.chain || '자료 없음'}<small>{run.workspaceId || '자료 없음'}</small></td><td>{format(run.startedAt)}</td><td><StatusBadge state={run.state} /></td>
      <td>{#if run.summary.counts}<span>통과 {run.summary.counts.pass} · 실패 {run.summary.counts.fail} · 막힘 {run.summary.counts.blocked} · 건너뜀 {run.summary.counts.skip}</span>{:else}<span>작업 단계 기록</span>{/if}</td>
    </tr>{/each}</tbody></table></div>
  {/if}
  {#if next}<button disabled={busy} onclick={() => load(next)}>다음 기록 더 보기</button>{/if}
  {#if comparison}<section class="detail" aria-label="실행 비교 결과"><h3>{comparison.comparable ? '테스트 판정 비교' : '동등한 환경으로 비교할 수 없습니다'}</h3>
    <p>아래 판정은 기록에 있는 값입니다. 자료가 없는 항목을 0으로 계산하지 않습니다.</p><ul>{#each comparison.limitations as limitation}<li>{limitation}</li>{/each}</ul>
    <div class="table-wrap"><table data-testid="comparison-cases"><thead><tr><th>케이스 · 단언</th>{#each comparison.runIds as id}<th>{id}</th>{/each}</tr></thead><tbody>
      <tr><td>소요 시간</td>{#each comparison.runIds as id}<td>{seconds(comparison.durations?.[id])}</td>{/each}</tr>
      {#each comparison.results as row}<tr><td>{row.caseId}</td>{#each comparison.runIds as id}<td>{row.statuses[id] || '자료 없음'}</td>{/each}</tr>
        {#each Object.entries(row.assertions ?? {}) as [assertion, results]}<tr class="assertion"><td>└ {assertion}</td>{#each comparison.runIds as id}<td>{id in results ? (results[id] ? '통과' : '실패') : '자료 없음'}</td>{/each}</tr>{/each}
      {/each}</tbody></table></div>
    {#if comparison.metrics?.length}<div class="table-wrap"><table data-testid="comparison-metrics"><thead><tr><th>노드 지표 (출처 · 단위)</th>{#each comparison.runIds as id}<th>{id}</th>{/each}</tr></thead><tbody>{#each comparison.metrics as metric}<tr><td>{metric.name}<small>{metric.source}{metric.unit ? ' · ' + metric.unit : ''}</small></td>{#each comparison.runIds as id}<td>{metricValue(metric.values[id])}</td>{/each}</tr>{/each}</tbody></table></div>{/if}
  </section>{/if}
  {#if detail}<section class="detail" aria-label="실행 상세 결과" bind:this={detailBox}>
    {#key detail.id}<HistoryDetail run={detail} {client} {administrator} {busy} ondownload={download} onremove={remove} bind:test={focus.test} bind:node={focus.node} bind:at={focus.at} />{/key}
  </section>{/if}
</section>

<style>
  header, .toolbar { display: flex; gap: var(--space-3); align-items: center; justify-content: space-between; flex-wrap: wrap; } h3 { margin: 0 0 var(--space-2); } p, li, .toolbar { color: var(--text-muted); } header p { margin: 0; max-width: 700px; }
  form { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: var(--space-3); margin: var(--space-4) 0; align-items: end; } .search { grid-column: span 2; } input, select { width: 100%; }
  table { font-size: var(--text-sm); } small { display: block; margin-top: var(--space-1); } td input { width: 16px; } .record { text-align: left; min-width: 12rem; max-width: 320px; white-space: normal; overflow-wrap: anywhere; } .record small { font-family: var(--font-mono); }
  .detail { margin-top: var(--space-5); min-width: 0; scroll-margin-top: var(--space-4); } .assertion td { color: var(--text-muted); }
  @media (max-width: 760px) { form { grid-template-columns: minmax(0, 1fr); } .search { grid-column: auto; } }
</style>
