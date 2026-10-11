<script>
  import { responseError } from './api-error.mjs'
  let { webSession = undefined, onsaved = () => {} } = $props()
  let preview = $state(null), saved = $state([]), filename = $state(''), status = $state(''), busy = $state(false)
  let workspaceName = $state('가져온 workspace')
  async function api(path, method, data) {
    const response = await fetch('/api/v1/' + path, { method, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': webSession?.csrfToken ?? '' }, body: JSON.stringify(data) })
    if (!response.ok) throw await responseError(response)
    return response.json()
  }
  async function work(fn) { busy = true; status = ''; try { await fn() } catch (error) { status = error.message } finally { busy = false } }
  async function choose(event) {
    const file = event.currentTarget.files?.[0]
    if (!file) return
    await work(async () => {
      saved = []; filename = file.name
      const format = /\.(ya?ml)$/i.test(file.name) ? 'yaml' : 'json'
      preview = await api('documents/import', 'POST', { filename: file.name, format, source: await file.text() })
    })
  }
  async function commit() {
    await work(async () => {
      saved = (await api('documents/import/commit', 'POST', { previewId: preview.previewId })).items
      status = `${filename}에서 공유 문서 revision ${saved.length}개를 저장했습니다`
      onsaved()
    })
  }
  // A workspace pins exactly one server set and one path configuration.
  const sets = $derived(saved.filter(d => d.kind === 'server-set'))
  const configs = $derived(saved.filter(d => d.kind === 'workspace-config'))
  const set = $derived(sets.length === 1 ? sets[0] : null)
  const config = $derived(configs.length === 1 ? configs[0] : null)
  async function createWorkspace() {
    await work(async () => {
      const w = await api('workspaces', 'POST', { name: workspaceName, documents: [{ id: set.id, revision: set.revision }, { id: config.id, revision: config.revision }] })
      status = `가져온 revision으로 공유 workspace ${w.name}을(를) 만들었습니다`
      onsaved()
    })
  }
</script>

<section class="bundle" aria-label="구성 가져오기">
  <h3>구성 가져오기</h3>
  <p>선언 하나나 서버, 경로, 프리셋, 케이스 문서 묶음(<code>documents</code> 목록이 있는 JSON 또는 YAML stream)을 가져옵니다. 미리 보기를 확인하기 전에는 아무것도 저장하지 않습니다. 개인 SSH 값은 지우며, 내 자격증명으로 다시 연결해야 합니다.</p>
  <label>구성 파일 <input aria-label="가져올 구성 파일" type="file" accept=".json,.yaml,.yml" disabled={busy} onchange={choose} /></label>
  {#if preview}
    <div data-testid="import-preview" data-valid={preview.validation.valid}>
      {#if preview.validation.valid}
        <p>저장할 수 있는 문서 {preview.redactedDocuments.length}개:</p>
        <ul>{#each preview.redactedDocuments as d}<li data-import-kind={d.kind}>{d.kind} · {d.name}</li>{/each}</ul>
      {:else}
        <p class="error"><span aria-hidden="true">✕</span> 가져온 내용을 저장할 수 없습니다:</p>
        <ul>{#each preview.validation.errors as issue}<li class="error" data-import-error={issue.path}><code>{issue.path}</code> {issue.message}</li>{/each}</ul>
      {/if}
      {#each preview.changes as change}<p class="notice">{change}</p>{/each}
      {#if preview.privateBindingsRequired.length}<p class="notice">다음 서버에 내 SSH 자격증명을 연결해야 합니다: {preview.privateBindingsRequired.join(', ')}</p>{/if}
      <button disabled={busy || !preview.validation.valid || saved.length > 0} onclick={commit}>가져온 문서 저장</button>
    </div>
  {/if}
  {#if set && config}
    <label>Workspace 이름 <input aria-label="가져온 workspace 이름" bind:value={workspaceName} /></label>
    <button disabled={busy} onclick={createWorkspace}>가져온 문서로 workspace 만들기</button>
  {/if}
  {#if status}<p role="status" data-testid="import-status">{status}</p>{/if}
</section>

<style>
  .bundle { border: 1px solid var(--border); border-radius: 8px; padding: 16px; margin: 16px 0; min-width: 0 }
  .error { color: var(--danger) } .notice { color: var(--warn) } li { overflow-wrap: anywhere }
</style>
