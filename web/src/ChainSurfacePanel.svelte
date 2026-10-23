<script>
  import { formatTime } from './labels.mjs'
  import StatusBadge from './StatusBadge.svelte'
  import { webApi } from './web-api.mjs'
  // chain is the chain being configured; the panel judges its documentation
  // alone, or against a registered binary the user picks.
  let { webSession, chain } = $props()
  const api = $derived(webApi(webSession))
  let assets = $state([]), assetId = $state(''), surface = $state(null), error = $state(''), busy = $state(false)
  let status = $state(''), search = $state('')
  const canProbe = $derived(webSession && webSession.user.role !== 'viewer')
  const statuses = $derived([...new Set((surface?.options ?? []).map(o => o.support.status))].sort())
  const counts = $derived(Object.fromEntries(statuses.map(s => [s, surface.options.filter(o => o.support.status === s).length])))
  const shown = $derived((surface?.options ?? []).filter(o => (!status || o.support.status === status) && (!search || o.spelling.includes(search))))
  const icon = v => v === 'verified' ? '✓' : v === 'incompatible' ? '✕' : v === 'stale' ? '!' : '○'
  const time = formatTime
  async function load() {
    busy = true; error = ''
    try { surface = await api(`chains/${encodeURIComponent(chain)}/surface${assetId ? '?assetId=' + encodeURIComponent(assetId) : ''}`) } catch (e) { surface = null; error = e.message } finally { busy = false }
  }
  $effect(() => { void chain; assetId = ''; if (webSession && chain) { api('manifest-assets').then(v => { assets = v.items.filter(a => a.chain === chain) }).catch(() => { assets = [] }); load() } })
</script>

<section class="surface" aria-label="바이너리 지원 범위">
  <h3>명령과 옵션 지원 범위</h3>
  <p class="muted">체인 문서가 정한 명령과 옵션을 엔진 매핑과 맞춰 봅니다. 등록된 바이너리를 고르면 그 바이너리의 help로 다시 확인합니다.</p>
  {#if canProbe}<label>확인할 바이너리 <select name="asset-id" aria-label="지원 범위를 확인할 바이너리" bind:value={assetId} onchange={load} disabled={busy}><option value="">문서만으로 판단</option>{#each assets as a}<option value={a.id}>{a.id} · {a.sha256.slice(0, 12)}</option>{/each}</select></label>{/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if busy}<p class="muted">확인하는 중…</p>{/if}
  {#if surface}
    <p data-testid="surface-verification" class="verdict" class:bad={surface.verification === 'incompatible'} class:warn={surface.verification === 'stale'}><StatusBadge state={surface.verification} /> · {surface.chainId} · dialect {surface.dialect} · 계약 {surface.contractVersion} · 지문 <code>{surface.fingerprint.slice(0, 12)}</code></p>
    {#if surface.binary}<p data-testid="surface-binary">바이너리 {surface.binary.assetId} · {surface.binary.version} · 커밋 <code>{surface.binary.chainCommit}</code> · {surface.binary.os}/{surface.binary.architecture}</p>{/if}
    {#each surface.limitations as l}<p class="warn">{l}</p>{/each}
    <details><summary>문서 출처 {surface.sources.length}개</summary><ul>{#each surface.sources as src}<li><code>{src.path}</code> · {src.kind}{src.chainCommit ? ` · 커밋 ${src.chainCommit}` : ''}{src.capturedAt ? ` · 수집 ${time(src.capturedAt)}` : ''}</li>{/each}</ul></details>
    <div class="filters">
      <label>지원 상태 <select name="status" aria-label="옵션 지원 상태" bind:value={status}><option value="">전체 {surface.options.length}개</option>{#each statuses as s}<option value={s}>{s} {counts[s]}개</option>{/each}</select></label>
      <label>옵션 찾기 <input name="search" aria-label="옵션 찾기" bind:value={search} placeholder="--http" /></label>
    </div>
    <div class="table-wrap"><table><thead><tr><th>옵션</th><th>범위</th><th>지원</th><th>이유</th></tr></thead><tbody>
      {#each shown.slice(0, 200) as o (o.id)}<tr data-testid="surface-option"><td><code>{o.spelling}</code>{o.hidden ? ' (숨김)' : ''}</td><td>{o.scope}{o.commandPath.length ? ' · ' + o.commandPath.join(' ') : ''}</td><td><StatusBadge state={o.support.status} />{o.support.selectable ? ' · 선택 가능' : ''}</td><td>{o.support.reason}</td></tr>{/each}
    </tbody></table></div>
    {#if shown.length > 200}<p class="muted">{shown.length}개 중 200개를 보입니다. 상태나 이름으로 좁히세요.</p>{/if}
  {/if}
</section>

<style>
  .surface { margin-top: var(--space-5); padding-top: var(--space-4); border-top: 1px solid var(--border-subtle); }
  .error { margin: 0; padding: 0; border: 0; background: none; } .warn { color: var(--warn); }
  .verdict { color: var(--ok); } .verdict.warn { color: var(--warn); } .verdict.bad { color: var(--danger); }
  label { margin: var(--space-2) 0; } .filters { display: flex; flex-wrap: wrap; gap: var(--space-4); }
  .table-wrap { max-height: 32rem; }
</style>
