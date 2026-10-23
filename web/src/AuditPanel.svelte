<script>
  import { formatTime } from './labels.mjs'
  import { webApi } from './web-api.mjs'
  let { webSession } = $props()
  const api = $derived(webApi(webSession))
  let items = $state([]), next = $state(null), from = $state(''), to = $state(''), status = $state(''), busy = $state(false)
  const query = cursor => { const q = new URLSearchParams({ limit: '50' }); if (from) q.set('from', new Date(from).toISOString()); if (to) q.set('to', new Date(to).toISOString()); if (cursor) q.set('cursor', cursor); return `audit?${q}` }
  async function load(more) {
    busy = true; status = ''
    try { const page = await api(query(more ? next : null)); items = more ? [...items, ...page.items] : page.items; next = page.nextCursor ?? null; if (!items.length) status = '이 기간에 기록이 없습니다' }
    catch (e) { status = e.message } finally { busy = false }
  }
  $effect(() => { if (webSession) load(false) })
  // A result is an HTTP status or the outcome a store recorded.
  const refused = result => /^[45]\d\d$/.test(result) || result === 'failed'
  const fmt = formatTime
</script>

<section class="panel" aria-label="감사 기록">
  <h2>감사 기록</h2>
  <p class="muted">관리자만 봅니다. 요청마다 누가, 무엇을, 어떤 결과로 했는지 남으며 비밀 값은 기록하지 않습니다.</p>
  <form class="filters" onsubmit={e => { e.preventDefault(); load(false) }}>
    <label>시작 <input name="from" aria-label="감사 기록 시작 시각" type="datetime-local" bind:value={from} /></label>
    <label>끝 <input name="to" aria-label="감사 기록 끝 시각" type="datetime-local" bind:value={to} /></label>
    <button disabled={busy}>기간 조회</button>
  </form>
  {#if items.length}
    <div class="table-wrap"><table><thead><tr><th>시각</th><th>사용자</th><th>작업</th><th>대상</th><th>결과</th><th>요청 ID</th></tr></thead>
      <tbody>{#each items as r (r.id)}<tr data-testid="audit-row"><td>{fmt(r.time)}</td><td class="mono">{r.actorId}</td><td>{r.operation}</td><td class="mono">{r.target}</td><td class:refused={refused(r.result)}><span aria-hidden="true">{refused(r.result) ? '⊘' : '✓'}</span> {r.result}</td><td class="mono">{r.requestId ?? ''}</td></tr>{/each}</tbody>
    </table></div>
  {/if}
  {#if next}<button disabled={busy} onclick={() => load(true)}>더 보기</button>{/if}
  <p role="status" data-testid="audit-status">{status}</p>
</section>

<style>
  .filters { display: flex; flex-wrap: wrap; gap: var(--space-3); align-items: end; }
  .mono { overflow-wrap: anywhere; } .refused { color: var(--danger); }
</style>
