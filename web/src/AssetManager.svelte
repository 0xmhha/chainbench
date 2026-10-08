<script>
  let { webSession, onuploaded = () => {} } = $props()
  let items = $state([]), kind = $state('binary'), selected = $state(null), busy = $state(false), message = $state(''), filter = $state('all')
  const writable = $derived(['administrator', 'admin', 'operator'].includes(webSession?.user.role))
  const visible = $derived(items.filter(item => filter === 'all' || item.kind === filter))
  const names = {binary:'체인 바이너리',configuration:'설정 선언',template:'템플릿',material:'테스트 자료'}
  async function refresh() {
    const response = await fetch('/api/v1/assets')
    if (!response.ok) throw new Error(`${response.status}: ${await response.text()}`)
    items = (await response.json()).items
  }
  $effect(() => { if (webSession) refresh().catch(error => { message = error.message }) })
  async function upload() {
    busy = true; message = ''
    try {
      const body = new FormData(); body.append('kind', kind); body.append('file', selected)
      const response = await fetch('/api/v1/assets', {method:'POST',headers:{'X-CSRF-Token':webSession.csrfToken},body})
      if (!response.ok) throw new Error(`${response.status}: ${await response.text()}`)
      const item = await response.json(); await refresh(); onuploaded()
      message = `등록 완료 · ${item.name} · ${item.id}`
    } catch (error) { message = error.message } finally { busy = false }
  }
</script>

<section class="asset-library" aria-label="등록 자료">
  <div class="heading"><div><p class="eyebrow">INPUT LIBRARY</p><h2>등록 자료</h2></div><button disabled={busy} onclick={() => refresh().catch(error => {message=error.message})}>자료 새로고침</button></div>
  <p>파일을 등록하면 고유 ID와 체크섬이 부여됩니다. 호환성을 검증한 바이너리는 체인 실행 계획에서 선택할 수 있습니다.</p>
  {#if writable}
    <fieldset disabled={busy}><legend>자료 업로드</legend>
      <div class="upload-row"><label>자료 종류<select aria-label="업로드 자료 종류" bind:value={kind}>{#each Object.entries(names) as [value,name]}<option {value}>{name}</option>{/each}</select></label>
        <label>파일<input aria-label="업로드 자료 파일" type="file" onchange={event=>{selected=event.currentTarget.files[0]||null;message=''}} /></label>
        <button disabled={!selected} onclick={upload}>{busy?'검증 중…':'자료 등록'}</button></div>
      <small>바이너리 256 MiB · 그 외 16 MiB. 설정·템플릿은 JSON/YAML, 테스트 자료는 JSON ABI 또는 16진수 bytecode를 지원합니다. 개인 키와 자격증명은 개인 자격증명에 보관하세요.</small>
    </fieldset>
  {/if}
  <label class="filter">종류 필터<select aria-label="등록 자료 종류 필터" bind:value={filter}><option value="all">전체</option>{#each Object.entries(names) as [value,name]}<option {value}>{name}</option>{/each}</select></label>
  {#if visible.length===0}<p class="empty">등록된 자료가 없습니다.</p>{:else}
    <div class="table-wrap"><table><thead><tr><th>파일 · 등록 ID</th><th>종류</th><th>크기 · 체크섬</th><th>검증 결과</th></tr></thead><tbody>{#each visible as item}
      <tr><td>{item.name}<small>{item.id}</small></td><td>{names[item.kind]}</td><td>{(item.bytes/1024/1024).toFixed(2)} MiB<small title={item.checksum}>SHA-256 · {item.checksum.slice(0,16)}</small></td><td>{#if item.kind==='binary'}{item.compatibility.chain}<small>{item.compatibility.os} / {item.compatibility.architecture} · 버전·도움말 검증</small>{:else}{item.compatibility.format}<small>형식 확인 · 실행 계획에서 용도 검토</small>{/if}</td></tr>
    {/each}</tbody></table></div>
  {/if}
  <p role="status" aria-label="자료 등록 상태">{message}</p>
</section>

<style>
.asset-library{margin-top:1.5rem;padding:1.25rem;border:1px solid #344154;border-radius:12px;background:#151b24}.heading{display:flex;justify-content:space-between;align-items:center;gap:1rem}.eyebrow{font-size:.7rem;letter-spacing:.15em;color:#7cc6b6;margin:0}h2{margin:.35rem 0}p,small{color:#a7b5c7}fieldset{border:1px solid #344154;border-radius:8px;margin:1rem 0;padding:1rem}.upload-row{display:flex;flex-wrap:wrap;align-items:end;gap:.8rem;margin-bottom:.75rem}label{display:grid;gap:.4rem;min-width:0}input,select,button{font:inherit;padding:.55rem;background:#202b3b;color:#e3ebf5;border:1px solid #47576d;border-radius:6px;max-width:100%}button{cursor:pointer}button:disabled{opacity:.5;cursor:default}.filter{max-width:14rem;margin:1rem 0}small{display:block;font-size:.75rem;overflow-wrap:anywhere;line-height:1.6}.table-wrap{overflow-x:auto}table{border-collapse:collapse;width:100%;text-align:left}th,td{padding:.8rem;border-bottom:1px solid #344154;vertical-align:top}th{font-size:.75rem;color:#a7b5c7}td:first-child{overflow-wrap:anywhere}.empty{padding:1rem;background:#101722;border-radius:8px}@media(max-width:700px){.heading{align-items:start;flex-direction:column}.upload-row{display:grid}.asset-library{padding:1rem}}
</style>
