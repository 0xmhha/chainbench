<script>
  let { webSession = undefined } = $props()
  import DeploymentFields from './DeploymentFields.svelte'
  let username = $state(''), password = $state(''), authorization = $state('')
  let actor = $state(null), contract = $state(null), docs = $state([]), workspaces = $state([]), credentials = $state([])
  let kind = $state('server-set'), name = $state('Server pool'), content = $state({}), docId = $state(''), revision = $state(0)
  let workspaceId = $state(''), workspaceName = $state('Team deployment'), workspaceRevision = $state(0), setRef = $state(''), configRef = $state('')
  let credentialLabel = $state('SSH login'), credentialKind = $state('private-key'), sshUser = $state(''), privateKey = $state(''), sshPassword = $state(''), passphrase = $state('')
  let selectedCredential = $state(''), serverRef = $state(''), bindings = $state({}), status = $state(''), access = $state(null), busy = $state(false)
  async function api(path, method = 'GET', data, rev) {
    const headers = { Authorization: authorization, 'Content-Type': 'application/json' }
    if (webSession?.csrfToken) headers['X-CSRF-Token']=webSession.csrfToken
    if (rev) headers['If-Match'] = `"${rev}"`
    const response = await fetch(`/api/v1/${path}`, { method, headers, body: data === undefined ? undefined : JSON.stringify(data) })
    if (!response.ok) throw new Error(`${response.status}: ${await response.text()}`)
    return response.json()
  }
  async function action(fn) { busy = true; status = ''; try { await fn() } catch (error) { status = error.message } finally { busy = false } }
  async function refresh(role = webSession?.user.role ?? actor?.role) {
    docs = (await api('documents')).items.filter(d => d.kind === 'server-set' || d.kind === 'workspace-config')
    workspaces = (await api('workspaces')).items
    credentials = role==='viewer' ? [] : (await api('credentials')).items
    for (const workspace of workspaces) { for (const ref of workspace.documents) { if (!docs.some(d => d.id === ref.id && d.revision === ref.revision)) { docs = [...docs, await api(`documents/${ref.id}?revision=${ref.revision}`)] } } }
  }
  function newDocument(nextKind) {
    kind = nextKind; docId = ''; revision = 0; name = kind === 'server-set' ? 'Server pool' : 'Target paths'
    content = kind === 'server-set' ? { version: 2, pool: { hosts: [{ name: 'local', addr: '127.0.0.1' }], slots: 1, ports: { p2p: { base: 31000, step: 10 }, rpc: { base: 8600, step: 10 } } } } : { version: 1, dataRoot: '/data/chainbench', paths: Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k => [k,k])), control: { artifactRoot: 'chainbench-out' }, inputs: { mode: 'generated' }, execution: { chain: 'fresh' } }
  }
  function loadDocument(id) { const d = docs.find(d => d.id === id); if (!d) return; kind = d.kind; name = d.name; content = structuredClone($state.snapshot(d.content)); docId = d.id; revision = d.revision }
  async function loadWorkspace(id) { workspaceId = id; const w = workspaces.find(w => w.id === id); if (!w) return; workspaceName = w.name; workspaceRevision = w.revision; for (const ref of w.documents) { if (!docs.some(d => d.id === ref.id && d.revision === ref.revision)) { const pinned = await api(`documents/${ref.id}?revision=${ref.revision}`); docs = [...docs, pinned] } } setRef = ''; configRef = ''; for (const r of w.documents) { const d = docs.find(d => d.id === r.id && d.revision === r.revision); if (d?.kind === 'server-set') setRef = `${r.id}:${r.revision}`; if (d?.kind === 'workspace-config') configRef = `${r.id}:${r.revision}` } bindings = await api(`workspaces/${id}/credential-bindings`); access = null }
  function documentInput() { return { kind, name, content, contractVersion: '2', assetRefs: [] } }
  async function saveDocument() { const d = await api(docId ? `documents/${docId}` : 'documents', docId ? 'PATCH' : 'POST', documentInput(), revision); docId = d.id; revision = d.revision; await refresh(); status = `Saved shared document revision ${revision}` }
  async function saveWorkspace() { const documents = [setRef, configRef].map(ref => { const [id, rev] = ref.split(':'); return { id, revision: Number(rev) } }); const w = await api(workspaceId ? `workspaces/${workspaceId}` : 'workspaces', workspaceId ? 'PATCH' : 'POST', { name: workspaceName, documents }, workspaceRevision); await refresh(); await loadWorkspace(w.id); status = `Saved shared workspace revision ${w.revision}` }
  async function exportDocument() { const exported = await api(`documents/${docId}/export`); const url = URL.createObjectURL(new Blob([JSON.stringify(exported, null, 2)], { type: 'application/json' })); const a = document.createElement('a'); a.href = url; a.download = `${kind}.json`; a.click(); URL.revokeObjectURL(url); status = 'Exported shared declaration' }
  function logout() { actor = null; contract = null; docs = []; workspaces = []; authorization = ''; password = ''; privateKey = ''; passphrase = ''; sshPassword = ''; credentials = []; bindings = {}; access = null; status = '' }
  $effect(() => { if(webSession) action(async()=>{ const account={id:webSession.user.id,role:webSession.user.role==='administrator'?'admin':webSession.user.role};contract=await api('contracts/deployment');await refresh();newDocument('server-set');actor=account }) })
  const canEdit = $derived(actor && ['admin','operator'].includes(actor.role))
  const hosts = $derived.by(() => { const id = setRef.split(':')[0]; const rev = Number(setRef.split(':')[1]); return docs.find(d => d.id === id && d.revision === rev)?.content.pool.hosts ?? [] })
</script>

<section class="deployment" aria-label="Deployment configuration">
  <h2>Shared deployment</h2>
  <p>Team server pools and target paths · personal SSH access</p>
  {#if !actor}
    <form onsubmit={e => { e.preventDefault(); action(async () => { authorization = `Basic ${btoa(unescape(encodeURIComponent(`${username}:${password}`)))}`; const account = await api('deployment-account'); password = ''; contract = await api('contracts/deployment'); await refresh(account.role); newDocument('server-set'); actor = account }) }}>
      <label>Account <input aria-label="Deployment account" autocomplete="username" bind:value={username} required /></label>
      <label>Password <input aria-label="Deployment password" type="password" autocomplete="current-password" bind:value={password} required /></label>
      <button disabled={busy}>Sign in to deployment</button>
    </form>
  {:else}
    <p data-testid="deployment-actor">{actor.id} · {actor.role} {#if !webSession}<button onclick={logout}>Sign out of deployment</button>{/if}</p>
    <fieldset class="editor-controls" disabled={busy}><div class="columns">
      <div>
        <h3>Shared documents</h3>
        <label>Saved document <select aria-label="Saved deployment document" value={docId} onchange={e => loadDocument(e.target.value)}><option value="">New document</option>{#each docs as d}<option value={d.id}>{d.name} · {d.kind} · revision {d.revision}</option>{/each}</select></label>
        {#if canEdit}
          <button onclick={() => newDocument('server-set')}>New server-set</button><button onclick={() => newDocument('workspace-config')}>New workspace-config</button>
          <label>Name <input aria-label="Deployment document name" bind:value={name} /></label>
          {#if contract}<DeploymentFields schema={contract[kind]} value={content} change={v => content = v} />{/if}
          <button disabled={busy} onclick={() => action(async () => { const result = await api('documents/validate', 'POST', documentInput()); status = result.valid ? 'Engine validation passed' : result.errors.map(e => e.message).join('; ') })}>Validate deployment document</button>
          <button disabled={busy} onclick={() => action(saveDocument)}>Save shared document</button>
        {/if}
        {#if docId}<p>Revision {revision}</p><button disabled={busy} onclick={() => action(exportDocument)}>Export shared document</button>{/if}
      </div>
      <div>
        <h3>Shared workspace</h3>
        <label>Saved workspace <select aria-label="Saved deployment workspace" value={workspaceId} onchange={e => action(() => loadWorkspace(e.target.value))}><option value="">New workspace</option>{#each workspaces as w}<option value={w.id}>{w.name} · revision {w.revision}</option>{/each}</select></label>
        {#if canEdit}
          <button onclick={() => { workspaceId = ''; workspaceRevision = 0; bindings = {} }}>New shared workspace</button>
          <label>Name <input aria-label="Deployment workspace name" bind:value={workspaceName} /></label>
          <label>Server-set revision <select aria-label="Workspace server-set revision" bind:value={setRef}><option value="">Select revision</option>{#each docs.filter(d => d.kind === 'server-set') as d}<option value={`${d.id}:${d.revision}`}>{d.name} · {d.revision}</option>{/each}</select></label>
          <label>Workspace-config revision <select aria-label="Workspace config revision" bind:value={configRef}><option value="">Select revision</option>{#each docs.filter(d => d.kind === 'workspace-config') as d}<option value={`${d.id}:${d.revision}`}>{d.name} · {d.revision}</option>{/each}</select></label>
          <button disabled={busy || !setRef || !configRef} onclick={() => action(saveWorkspace)}>Save shared workspace</button>
          <h3>My SSH credentials</h3>
          <p>Only metadata is returned. Keys and passwords stay private.</p>
          <label>Label <input aria-label="Credential label" bind:value={credentialLabel} /></label>
          <label>Kind <select aria-label="Credential kind" bind:value={credentialKind}><option value="private-key">Private key</option><option value="password">Password</option></select></label>
          <label>SSH user <input aria-label="SSH user" bind:value={sshUser} /></label>
          {#if credentialKind === 'private-key'}
            <label>Private key <textarea aria-label="Private SSH key" bind:value={privateKey} spellcheck="false"></textarea></label>
            <label>Passphrase <input aria-label="SSH passphrase" type="password" bind:value={passphrase} /></label>
          {:else}<label>Password <input aria-label="SSH password" type="password" bind:value={sshPassword} /></label>{/if}
          <button disabled={busy} onclick={() => action(async () => { const input = { label: credentialLabel, kind: credentialKind, sshUser }; if (credentialKind === 'private-key') { input.privateKey = privateKey; if (passphrase) input.passphrase = passphrase } else input.password = sshPassword; const c = await api('credentials', 'POST', input); privateKey = ''; passphrase = ''; sshPassword = ''; await refresh(); selectedCredential = c.id; status = 'Saved encrypted personal credential' })}>Save personal credential</button>
          {#if workspaceId}
            <label>Target server <select aria-label="Binding server" bind:value={serverRef}><option value="">Select server</option>{#each hosts as h}<option value={typeof h === 'string' ? h : h.name || h.addr}>{typeof h === 'string' ? h : h.name || h.addr}</option>{/each}</select></label>
            <label>My credential <select aria-label="Binding credential" bind:value={selectedCredential}><option value="">Select my credential</option>{#each credentials as c}<option value={c.id}>{c.label}</option>{/each}</select></label>
            <button disabled={busy || !serverRef || !selectedCredential} onclick={() => action(async () => { bindings = await api(`workspaces/${workspaceId}/credential-bindings`, 'PUT', { serverRef, credentialId: selectedCredential }); status = 'Saved personal SSH binding' })}>Bind my credential</button>
            <button disabled={busy || !serverRef || !selectedCredential} onclick={() => action(async () => { access = await api(`credentials/${selectedCredential}/check`, 'POST', { workspaceId, serverRef }); status = access.allowedOperations.includes('deploy') ? 'SSH deployment access verified' : access.reason ?? 'Access denied' })}>Check SSH access</button>
            <p data-testid="personal-bindings">My bindings: {Object.keys(bindings).join(', ') || 'none'}</p>
            {#if access}<p data-testid="ssh-access">Authenticated: {access.authenticated ? 'yes' : 'no'} · deploy: {access.allowedOperations.includes('deploy') ? 'allowed' : 'denied'} · {access.hostIdentity}</p>{/if}
          {/if}
        {/if}
      </div>
    </div></fieldset>
  {/if}
  <p role="status" data-testid="deployment-status">{status}</p>
</section>

<style>
  .deployment { margin-top: 24px; padding: 20px; border: 1px solid #36465b; border-radius: 12px; background: #111a26; }
  h3 { color: #b9cce7; margin-top: 24px; }
  .editor-controls { border: 0; padding: 0; margin: 0; min-width: 0; }
  .columns { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }
  label { display: grid; gap: 5px; margin: 10px 0; }
  input, select, textarea, button { font: inherit; color: inherit; border: 1px solid #4e6380; border-radius: 5px; background: #182332; padding: 8px; min-width: 0; }
  textarea { height: 90px; }
  button { cursor: pointer; margin: 4px; }
  button:disabled { opacity: .45; cursor: default; }
  [role=status] { color: #c7d7ed; overflow-wrap: anywhere; }
  @media (max-width: 700px) { .columns { grid-template-columns: 1fr; } }
</style>
