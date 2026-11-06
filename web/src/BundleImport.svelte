<script>
  import { apiClient } from './api-client.mjs'
  import { keepFocus } from './focus.mjs'
  import StatusMark from './StatusMark.svelte'
  let { webSession = undefined, onsaved = () => {} } = $props()
  let preview = $state(null), saved = $state([]), filename = $state(''), status = $state(''), failed = $state(false), busy = $state(false)
  let workspaceName = $state('가져온 workspace')
  const client = $derived(apiClient(webSession))
  function api(path, method, data) { return client.request(path, { method, body: data }) }
  async function work(fn) { const restoreFocus = keepFocus(); busy = true; status = ''; failed = false; try { await fn() } catch (error) { status = error.message; failed = true } finally { busy = false; await restoreFocus() } }
  const formatOf = name => /\.(ya?ml)$/i.test(name) ? 'yaml' : 'json'
  async function choose(event) {
    const files = [...event.currentTarget.files]
    if (!files.length) return
    preview = null; saved = []; status = ''
    await work(async () => {
      filename = files.map(f => f.name).join(', ')
      const body = files.length === 1
        ? { filename, format: formatOf(files[0].name), source: await files[0].text() }
        : { filename, files: await Promise.all(files.map(async f => ({ filename: f.name, format: formatOf(f.name), source: await f.text() }))) }
      preview = await api('documents/import', 'POST', body)
    })
  }
  // Each remote server needs a login; the list reads in server order.
  const servers = $derived((preview?.privateBindingsRequired ?? []).filter(b => b.startsWith('ssh:')).map(b => b.slice(4)).sort((a, b) => a.localeCompare(b, undefined, { numeric: true })))
  async function commit() {
    await work(async () => {
      saved = (await api('documents/import/commit', 'POST', { previewId: preview.previewId })).items
      status = `${filename}에서 공유 문서 revision ${saved.length}개를 저장했습니다`
      onsaved(saved)
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
      status = `가져온 revision으로 공유 workspace ${w.name}을(를) 만들었습니다. 이어서 서버에 내 자격증명을 연결하세요`
      onsaved(saved, w)
    })
  }
</script>

<section class="bundle" aria-label="구성 가져오기">
  <h3>구성 가져오기</h3>
  <p>가지고 있는 설정 파일을 그대로 고르세요. <code>server-set</code>, <code>localmap</code>, <code>workspace-config</code> 를 함께 고르면 하나의 서버 풀로 합칩니다. 문서 묶음 파일도 됩니다. 미리 보기를 확인하기 전에는 아무것도 저장하지 않고, 파일의 SSH 비밀번호는 저장하지 않습니다.</p>
  <label>구성 파일 <input name="bundle-import-input-1" aria-label="가져올 구성 파일" type="file" multiple accept=".json,.yaml,.yml" disabled={busy} onchange={choose} /></label>
  {#if preview}
    <div data-testid="import-preview" data-valid={preview.validation.valid}>
      {#if preview.validation.valid}
        <p>저장할 수 있는 문서 {preview.redactedDocuments.length}개:</p>
        <ul>{#each preview.redactedDocuments as d}<li data-import-kind={d.kind}>{d.kind} · {d.name}</li>{/each}</ul>
      {:else}
        <p class="error" role="alert"><StatusMark state="failed" /> 가져온 내용을 저장할 수 없습니다:</p>
        <ul>{#each preview.validation.errors as issue}<li class="error" data-import-error={issue.path}><code>{issue.path}</code> {issue.message}</li>{/each}</ul>
      {/if}
      {#each preview.changes as change}<p class="notice">{change}</p>{/each}
      {#if servers.length}<p class="notice">저장한 뒤 서버 {servers.length}대에 내 SSH 자격증명을 연결해야 합니다: {servers.length > 4 ? `${servers[0]} ~ ${servers.at(-1)}` : servers.join(', ')}</p>{/if}
      <button class="primary" disabled={busy || !preview.validation.valid || saved.length > 0} onclick={commit}>가져온 문서 저장</button>
    </div>
  {/if}
  {#if set && config}
    <label>Workspace 이름 <input name="workspace-name" aria-label="가져온 workspace 이름" bind:value={workspaceName} /></label>
    <button class="primary" disabled={busy} onclick={createWorkspace}>가져온 문서로 workspace 만들기</button>
  {/if}
  {#if status}<p role={failed ? 'alert' : 'status'} class:error={failed} data-testid="import-status">{status}</p>{/if}
</section>

<style>
  .bundle { margin: var(--space-4) 0; padding-top: var(--space-3); border-top: 1px solid var(--border-subtle); min-width: 0 }
  li { overflow-wrap: anywhere }
</style>
