<script>
  import { untrack } from 'svelte'
  import { parseHistoryJSON } from './history-json.js'
  let { webSession } = $props()
  let items = $state([]), next = $state(null), detail = $state(null), comparison = $state(null)
  let selected = $state([]), pendingDelete = $state(null), busy = $state(false), error = $state('')
  let search = $state(''), chain = $state(''), workspaceId = $state(''), actorId = $state(''), state = $state(''), caseId = $state('')
  let from = $state(''), to = $state(''), workspaces = $state([]), knownChains = $state([]), knownActors = $state([]), knownCases = $state([])
  const administrator = $derived(webSession?.user.role === 'administrator')
  const terminal = run => ['succeeded', 'failed', 'cancelled', 'interrupted'].includes(run.state)
  const format = time => time ? new Date(time).toLocaleString() : '자료 없음'
  async function api(path, method = 'GET', body) {
    const response = await fetch('/api/v1/' + path, { method, headers: { 'Content-Type': 'application/json', ...(method !== 'GET' ? { 'X-CSRF-Token': webSession?.csrfToken ?? '' } : {}) }, body: body === undefined ? undefined : JSON.stringify(body) })
    if (!response.ok) throw new Error(`요청 실패 (${response.status})`)
    return response
  }
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
      const page = parseHistoryJSON(await (await api('history?' + query)).text())
      items = cursor ? [...items, ...page.items] : page.items; next = page.nextCursor
      remember(page.items)
    } catch (e) { error = e.message } finally { busy = false }
  }
  $effect(() => {
    if (!webSession) return
    untrack(() => {
      load()
      api('workspaces').then(r => r.json()).then(page => { workspaces = page.items }).catch(e => { error = e.message })
    })
  })
  function select(id, checked) {
    selected = checked ? [...selected, id] : selected.filter(value => value !== id)
    comparison = null
  }
  async function open(id) {
    busy = true; error = ''; pendingDelete = null
    try { detail = parseHistoryJSON(await (await api('history/' + encodeURIComponent(id))).text()) } catch (e) { error = e.message } finally { busy = false }
  }
  async function compare() {
    busy = true; error = ''
    try { comparison = await (await api('history/compare', 'POST', { runIds: selected })).json() } catch (e) { error = e.message } finally { busy = false }
  }
  async function download(id) {
    busy = true; error = ''
    try {
      const response = await api('history/' + encodeURIComponent(id) + '/export')
      const url = URL.createObjectURL(await response.blob()), link = document.createElement('a')
      link.href = url; link.download = id + '.json'; link.click(); setTimeout(() => URL.revokeObjectURL(url), 1000)
    } catch (e) { error = e.message } finally { busy = false }
  }
  async function remove() {
    busy = true; error = ''
    try {
      await api('history/' + encodeURIComponent(pendingDelete.id), 'DELETE')
      selected = selected.filter(id => id !== pendingDelete.id)
      pendingDelete = null; detail = null; comparison = null
      await load()
    } catch (e) { error = e.message + '. 삭제를 완료하지 못했습니다. 보관 결과를 다시 확인하세요.' } finally { busy = false }
  }
</script>

<section class="history-panel" aria-label="실행 히스토리">
  <header><div><h2>실행 기록</h2><p>서버에 보관된 결과 사본입니다. 과거 기록에서 확인할 수 없는 값은 자료 없음으로 표시합니다.</p></div><button disabled={busy} onclick={() => load()}>히스토리 새로고침</button></header>
  <form onsubmit={e => { e.preventDefault(); selected = []; comparison = null; load() }}>
    <label class="search">기록 검색<input aria-label="히스토리 검색" bind:value={search} placeholder="실행 ID, 명령, 테스트 이름" /></label>
    <label>체인<select aria-label="히스토리 체인" bind:value={chain}><option value="">모든 체인</option>{#each knownChains as value}<option>{value}</option>{/each}</select></label>
    <label>Workspace<select aria-label="히스토리 Workspace" bind:value={workspaceId}><option value="">모든 Workspace</option>{#each workspaces as w}<option value={w.id}>{w.name}</option>{/each}</select></label>
    <label>실행자<select aria-label="히스토리 실행자" bind:value={actorId}><option value="">모든 실행자</option>{#each knownActors as value}<option>{value}</option>{/each}</select></label>
    <label>상태<select aria-label="히스토리 상태" bind:value={state}><option value="">모든 상태</option>{#each ['accepted', 'running', 'cancelling', 'succeeded', 'failed', 'cancelled', 'interrupted', 'unknown'] as value}<option>{value}</option>{/each}</select></label>
    <label>테스트 케이스<select aria-label="히스토리 케이스" bind:value={caseId}><option value="">모든 케이스</option>{#each knownCases as value}<option>{value}</option>{/each}</select></label>
    <label>시작 이후<input aria-label="히스토리 시작 시각" type="datetime-local" bind:value={from} /></label><label>시작 이전<input aria-label="히스토리 종료 시각" type="datetime-local" bind:value={to} /></label>
    <button type="submit" disabled={busy}>필터 적용</button>
  </form>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if busy}<p role="status">기록을 확인하는 중…</p>{/if}
  <div class="toolbar"><span>{items.length}개 표시 · 비교는 2–4개 선택</span><button disabled={busy || selected.length < 2} onclick={compare}>선택한 실행 비교</button></div>
  {#if items.length === 0}<p class="empty">조건에 맞는 실행 기록이 없습니다.</p>{:else}
    <div class="table-wrap"><table><thead><tr><th>비교</th><th>실행</th><th>체인 / Workspace</th><th>시작 시각</th><th>상태</th><th>결과</th></tr></thead><tbody>{#each items as run (run.id)}<tr>
      <td><input type="checkbox" aria-label={`비교 선택 ${run.id}`} checked={selected.includes(run.id)} disabled={!selected.includes(run.id) && selected.length === 4} onchange={e => select(run.id, e.currentTarget.checked)} /></td>
      <td><button class="record" onclick={() => open(run.id)}>{run.summary.operation ?? run.summary.command ?? run.id}<small>{run.id}</small></button></td>
      <td>{run.chain || '자료 없음'}<small>{run.workspaceId || '자료 없음'}</small></td><td>{format(run.startedAt)}</td><td><span class="state" class:success={run.state === 'succeeded'}>{run.state}</span></td>
      <td>{#if run.summary.counts}<span>통과 {run.summary.counts.pass} · 실패 {run.summary.counts.fail} · 차단 {run.summary.counts.blocked} · 제외 {run.summary.counts.skip}</span>{:else}<span>작업 단계 기록</span>{/if}</td>
    </tr>{/each}</tbody></table></div>
  {/if}
  {#if next}<button disabled={busy} onclick={() => load(next)}>다음 기록 더 보기</button>{/if}
  {#if comparison}<section class="detail" aria-label="실행 비교 결과"><h3>{comparison.comparable ? '테스트 판정 비교' : '동등한 환경으로 비교할 수 없습니다'}</h3>
    <p>아래 판정은 기록에 있는 값입니다. 자료가 없는 항목을 0으로 계산하지 않습니다.</p><ul>{#each comparison.limitations as limitation}<li>{limitation}</li>{/each}</ul>
    <div class="table-wrap"><table><thead><tr><th>케이스</th>{#each comparison.runIds as id}<th>{id}</th>{/each}</tr></thead><tbody>{#each comparison.results as row}<tr><td>{row.caseId}</td>{#each comparison.runIds as id}<td>{row.statuses[id] || '자료 없음'}</td>{/each}</tr>{/each}</tbody></table></div>
  </section>{/if}
  {#if detail}<section class="detail" aria-label="실행 상세 결과"><header><h3>상세 결과</h3><div class="actions"><button disabled={busy} onclick={() => download(detail.id)}>결과 내보내기</button>{#if administrator}<button class="danger" disabled={busy || !terminal(detail)} onclick={() => { pendingDelete = detail }}>보관 결과 삭제</button>{/if}</div></header>
    <p class="mono">{detail.id}</p><p>원본 세션: {detail.sessionRefs.join(', ') || '연결된 엔진 세션 없음'}</p>
    <p>큰 정수는 정확한 숫자 문자열로 표시합니다. 다운로드 파일은 원래 JSON 수치 형식을 유지합니다.</p>
    {#if detail.summary.missingDimensions?.length}<p class="notice">자료 없음: {detail.summary.missingDimensions.join(', ')}</p>{/if}
    {#if detail.summary.captureGaps?.length}<ul>{#each detail.summary.captureGaps as gap}<li>{gap}</li>{/each}</ul>{/if}
    {#if pendingDelete}<div class="confirm" role="group" aria-label="보관 결과 삭제 확인"><p>이 결과 사본을 삭제합니다. 원본 CLI 세션, 공유 자료와 노드 데이터는 유지됩니다. 삭제 기록은 감사 이력에 남습니다.</p><button class="danger" disabled={busy} onclick={remove}>결과 사본 삭제 확정</button><button disabled={busy} onclick={() => { pendingDelete = null }}>돌아가기</button></div>{/if}
    <details open><summary>테스트 판정 · 어세션 · 단계 · 사용 자료</summary><pre>{JSON.stringify(detail.summary, null, 2)}</pre></details>
    <details><summary>환경 fingerprint</summary><pre>{JSON.stringify(detail.fingerprints, null, 2)}</pre></details>
  </section>{/if}
</section>

<style>
  .history-panel { background: #121b28; border: 1px solid #334155; border-radius: 10px; padding: 24px; margin-top: 24px; min-width: 0; }
  header, .toolbar, .actions { display: flex; gap: 12px; align-items: center; justify-content: space-between; flex-wrap: wrap; } h2, h3 { margin: 0 0 8px; } p, li, .toolbar { color: #a8b5c8; } header p { margin: 0; max-width: 700px; }
  form { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin: 24px 0; align-items: end; } label { display: grid; gap: 6px; color: #a8b5c8; } .search { grid-column: span 2; } input, select { width: 100%; min-width: 0; color: #f1f5f9; background: #0b1018; border: 1px solid #475569; border-radius: 6px; padding: 8px; }
  button { color: #dbeafe; background: #192536; border: 1px solid #475569; border-radius: 6px; padding: 8px 12px; } button:disabled { opacity: .55; } .danger { color: #fecaca; border-color: #9f4b52; } .error, .notice { color: #fca5a5; } .confirm { padding: 16px; border: 1px solid #9f4b52; border-radius: 8px; } .confirm button { margin-right: 8px; }
  .table-wrap { overflow: auto; margin: 16px 0; } table { border-collapse: collapse; width: 100%; font-size: 12px; } th, td { padding: 12px; text-align: left; border-bottom: 1px solid #253247; } th { color: #a8b5c8; font-weight: 500; } small { display: block; color: #a8b5c8; margin-top: 4px; } td input { width: 20px; min-height: 24px; } .record { text-align: left; max-width: 320px; overflow-wrap: anywhere; } .record small { font-family: ui-monospace, monospace; font-size: 10px; } .state { border: 1px solid #475569; border-radius: 20px; padding: 4px 8px; } .success { color: #6ee7b7; border-color: #295344; }
  .detail { margin-top: 24px; border-top: 1px solid #334155; padding-top: 24px; min-width: 0; } .mono, pre { font-family: ui-monospace, monospace; } pre { font-size: 12px; overflow: auto; padding: 16px; background: #0b1018; border-radius: 8px; max-height: 560px; } summary { cursor: pointer; padding: 12px 0; color: #dbeafe; } .empty { padding: 32px; text-align: center; }
  @media (max-width: 760px) { .history-panel { padding: 16px; } form { grid-template-columns: minmax(0, 1fr); } .search { grid-column: auto; } }
</style>
