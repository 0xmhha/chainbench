<script>
  import { responseError } from './api-error.mjs'
  let { webSession = undefined, onsaved = () => {} } = $props()
  let preview = $state(null), saved = $state([]), filename = $state(''), status = $state(''), busy = $state(false)
  let workspaceName = $state('Imported workspace')
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
      status = `Saved ${saved.length} shared document revision(s) from ${filename}`
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
      status = `Created shared workspace ${w.name} from the imported revisions`
      onsaved()
    })
  }
</script>

<section class="bundle" aria-label="Configuration import">
  <h3>Import configuration</h3>
  <p>Import one declaration or a bundle of server, path, preset and case documents (JSON with a <code>documents</code> list, or a YAML stream). Nothing is saved until you review the preview. Private SSH values are removed and must be bound to your own credential.</p>
  <label>Configuration file <input aria-label="Configuration import file" type="file" accept=".json,.yaml,.yml" disabled={busy} onchange={choose} /></label>
  {#if preview}
    <div data-testid="import-preview" data-valid={preview.validation.valid}>
      {#if preview.validation.valid}
        <p>{preview.redactedDocuments.length} document(s) ready to save:</p>
        <ul>{#each preview.redactedDocuments as d}<li data-import-kind={d.kind}>{d.kind} · {d.name}</li>{/each}</ul>
      {:else}
        <p class="error">The import cannot be saved:</p>
        <ul>{#each preview.validation.errors as issue}<li class="error" data-import-error={issue.path}><code>{issue.path}</code> {issue.message}</li>{/each}</ul>
      {/if}
      {#each preview.changes as change}<p class="notice">{change}</p>{/each}
      {#if preview.privateBindingsRequired.length}<p class="notice">Bind your own SSH credential for: {preview.privateBindingsRequired.join(', ')}</p>{/if}
      <button disabled={busy || !preview.validation.valid || saved.length > 0} onclick={commit}>Save imported documents</button>
    </div>
  {/if}
  {#if set && config}
    <label>Workspace name <input aria-label="Imported workspace name" bind:value={workspaceName} /></label>
    <button disabled={busy} onclick={createWorkspace}>Create workspace from import</button>
  {/if}
  {#if status}<p role="status" data-testid="import-status">{status}</p>{/if}
</section>

<style>
  .bundle { border: 1px solid #344154; border-radius: 8px; padding: 16px; margin: 16px 0; min-width: 0 }
  .error { color: #fca5a5 } .notice { color: #fcd34d } li { overflow-wrap: anywhere }
</style>
