<script>
  import { apiClient } from './api-client.mjs'
  import { webApi } from './web-api.mjs'
  import { allItems } from './pages.mjs'
  import { draftMeta, contractChanged, changedDefinitions } from './contract-draft.mjs'
  import { selection, selectWorkspace } from './selection.svelte.js'
  import DeploymentForm from './DeploymentForm.svelte'
  import { fieldTitle, changedPaths } from './contract-form.mjs'
  import { formatTime } from './labels.mjs'
  import { schemaProblems, fieldPath } from './schema-problems.mjs'
  import { tick } from 'svelte'
  import BundleImport from './BundleImport.svelte'
  let { webSession = undefined, onsaved = () => {} } = $props()
  let contract = $state(null), docs = $state([]), workspaces = $state([]), credentials = $state([])
  let kind = $state('server-set'), name = $state('서버 풀'), content = $state({}), docId = $state(''), revision = $state(0)
  let workspaceId = $state(''), workspaceName = $state('팀 workspace'), workspaceRevision = $state(0), setRef = $state(''), configRef = $state('')
  let selectedCredential = $state(''), serverRef = $state(''), bindings = $state({}), status = $state(''), access = $state(null), busy = $state(false)
  const role = $derived(webSession?.user.role)
  const canEdit = $derived(role === 'administrator' || role === 'operator')
  // problems are what the form or the server found wrong in the open document.
  let problems = $state([])
  // revisions are the open document's saved revisions, newest first; viewing
  // is a past one opened read-only, compared field by field with the latest.
  let revisions = $state([]), viewing = $state(null)
  const latest = $derived(docs.find(d => d.id === docId && d.revision === revision)?.content ?? content)
  const differs = $derived(viewing ? changedPaths(viewing.content, latest) : new Set())
  // who names the account that saved a revision: "나" for the signed-in one,
  // otherwise a short form of its id (the full id is in the title).
  const who = id => id === webSession?.user.id ? '나' : id.length > 12 ? id.slice(0, 8) + '…' : id
  async function loadRevisions(id) { revisions = id ? (await api(`documents/${id}/revisions`)).items : [] }
  async function view(rev) { viewing = rev === revision ? null : await api(`documents/${docId}?revision=${rev}`) }
  // Restoring saves the past content as a new revision; nothing is rewritten.
  async function restore() {
    const from = viewing.revision
    content = structuredClone($state.snapshot(viewing.content)); viewing = null
    await check(); await saveDocument()
    status = `revision ${from}의 내용을 revision ${revision}로 저장했습니다`
  }
  const problemMap = $derived(Object.fromEntries(problems.filter(p => p.path).map(p => [p.path, p.message])))
  async function focusProblem(path) { await tick(); globalThis.document.querySelector(`[data-field-path="${CSS.escape(path)}"] input, [data-field-path="${CSS.escape(path)}"] select`)?.focus() }
  // The contract's own rules are checked first; the engine then judges the rest.
  async function check() {
    problems = schemaProblems(contract[kind], content)
    if (problems.length) throw new Error(`입력 ${problems.length}곳을 고쳐야 합니다.`)
    const result = await api('documents/validate', 'POST', documentInput())
    if (!result.valid) { problems = serverProblems(result.errors); throw new Error('엔진 검증 실패') }
  }
  // serverProblems places the engine's refusals on the fields they point at.
  function serverProblems(errors) {
    return (errors ?? []).map(e => ({ path: fieldPath(contract?.[kind], e.path), message: e.message }))
  }
  const kindNames = { 'server-set': '서버 풀', 'workspace-config': '대상 경로' }
  const api = $derived(webApi(webSession))
  async function action(fn) { busy = true; status = ''; try { await fn() } catch (error) { status = error.message } finally { busy = false } }
  async function refresh() {
    docs = (await allItems(api, 'documents')).filter(d => d.kind === 'server-set' || d.kind === 'workspace-config')
    workspaces = await allItems(api, 'workspaces')
    credentials = canEdit ? await allItems(api, 'credentials') : []
    for (const workspace of workspaces) { for (const ref of workspace.documents) { if (!docs.some(d => d.id === ref.id && d.revision === ref.revision)) { docs = [...docs, await api(`documents/${ref.id}?revision=${ref.revision}`)] } } }
  }
  function newDocument(nextKind) {
    kind = nextKind; docId = ''; revision = 0; name = kindNames[kind]; revisions = []; viewing = null; problems = []
    content = kind === 'server-set' ? { version: 2, pool: { hosts: [{ name: 'local', addr: '127.0.0.1' }], slots: 1, ports: { p2p: { base: 31000, step: 10 }, rpc: { base: 8600, step: 10 } } } } : { version: 1, dataRoot: '/data/chainbench', paths: Object.fromEntries(['binaries','configs','genesis','keystore','keyrings','nodes','runtime','logs'].map(k => [k,k])), control: { artifactRoot: 'chainbench-out' }, inputs: { mode: 'generated' }, execution: { chain: 'fresh' } }
  }
  function loadDocument(id) {
    const d = docs.filter(d => d.id === id).sort((a, b) => b.revision - a.revision)[0]; if (!d) return
    kind = d.kind; name = d.name; content = structuredClone($state.snapshot(d.content)); docId = d.id; revision = d.revision; viewing = null; problems = []
    action(() => loadRevisions(d.id))
  }
  async function loadWorkspace(id) {
    workspaceId = id; selectWorkspace(id)
    const w = workspaces.find(w => w.id === id); if (!w) return
    workspaceName = w.name; workspaceRevision = w.revision
    for (const ref of w.documents) { if (!docs.some(d => d.id === ref.id && d.revision === ref.revision)) { const pinned = await api(`documents/${ref.id}?revision=${ref.revision}`); docs = [...docs, pinned] } }
    setRef = ''; configRef = ''
    for (const r of w.documents) { const d = docs.find(d => d.id === r.id && d.revision === r.revision); if (d?.kind === 'server-set') setRef = `${r.id}:${r.revision}`; if (d?.kind === 'workspace-config') configRef = `${r.id}:${r.revision}` }
    bindings = canEdit ? await api(`workspaces/${id}/credential-bindings`) : {}; access = null
  }
  function documentInput() { return { kind, name, content, ...draftMeta(contract), assetRefs: [] } }
  async function saveDocument() {
    try { const d = await api(docId ? `documents/${docId}` : 'documents', docId ? 'PATCH' : 'POST', documentInput(), revision); docId = d.id; revision = d.revision; onsaved(); await refresh(); await loadRevisions(docId); status = `공유 문서 revision ${revision} 저장됨` }
    catch (e) {
      if (!contractChanged(e)) { if (e.errors?.length) problems = serverProblems(e.errors); throw e }
      const before = contract; contract = await api('contracts/deployment')
      const result = await api('documents/validate', 'POST', documentInput())
      throw new Error('엔진 계약이 바뀌었습니다. 바뀐 부분: ' + (changedDefinitions(before, contract).join(', ') || '세부 항목') + '. 새 계약 검증 ' + (result.valid ? '통과' : '실패: ' + result.errors.map(x => x.message).join('; ')) + '. 확인한 뒤 다시 저장하세요.')
    }
  }
  async function saveWorkspace() { const documents = [setRef, configRef].map(ref => { const [id, rev] = ref.split(':'); return { id, revision: Number(rev) } }); const w = await api(workspaceId ? `workspaces/${workspaceId}` : 'workspaces', workspaceId ? 'PATCH' : 'POST', { name: workspaceName, documents }, workspaceRevision); onsaved(); await refresh(); await loadWorkspace(w.id); status = `공유 workspace revision ${w.revision} 저장됨` }
  function download(blob, filename) { const url = URL.createObjectURL(blob); const a = document.createElement('a'); a.href = url; a.download = filename; a.click(); URL.revokeObjectURL(url) }
  async function exportDocument() { download(new Blob([JSON.stringify(await api(`documents/${docId}/export`), null, 2)], { type: 'application/json' }), `${kind}.json`); status = '공유 선언을 내보냈습니다' }
  async function exportWorkspace() { const response = await apiClient(webSession).send(`workspaces/${workspaceId}/export`); download(await response.blob(), `${workspaceName}.bundle.json`); status = 'workspace 묶음을 내보냈습니다' }
  $effect(() => { if (webSession) action(async () => { contract = await api('contracts/deployment'); await refresh(); newDocument('server-set'); if (selection.workspaceId && workspaces.some(w => w.id === selection.workspaceId)) await loadWorkspace(selection.workspaceId) }) })
  const hosts = $derived.by(() => { const id = setRef.split(':')[0]; const rev = Number(setRef.split(':')[1]); return docs.find(d => d.id === id && d.revision === rev)?.content.pool.hosts ?? [] })
  const hostName = h => typeof h === 'string' ? h : h.name || h.addr
</script>

<section class="panel deployment" aria-label="배포 구성">
  <h2>공유 배포 구성</h2>
  <p>팀이 함께 쓰는 서버 풀과 대상 경로, workspace를 만들고 내 SSH 자격증명을 서버에 연결합니다.</p>
  <p data-testid="deployment-actor">{webSession?.user.username} · {role}</p>
  <fieldset class="editor-controls" disabled={busy}><div class="columns">
    <div>
      <h3>공유 문서</h3>
      <label>저장된 문서 <select name="deployment-editor-select-1" aria-label="저장된 배포 문서" value={docId} onchange={e => loadDocument(e.target.value)}><option value="">새 문서</option>{#each docs as d}<option value={d.id}>{d.name} · {kindNames[d.kind] ?? d.kind} · revision {d.revision}</option>{/each}</select></label>
      {#if canEdit}<button onclick={() => newDocument('server-set')}>새 서버 풀</button><button onclick={() => newDocument('workspace-config')}>새 대상 경로</button>{/if}
      {#if revisions.length}
        <details class="revisions" open={revisions.length > 1 || undefined}>
          <summary>revision 기록 {revisions.length}개</summary>
          <table data-testid="document-revisions"><thead><tr><th scope="col">revision</th><th scope="col">이름</th><th scope="col">저장한 사람</th><th scope="col">시각</th></tr></thead><tbody>
            {#each revisions as r (r.revision)}
              <tr class:current={r.revision === (viewing?.revision ?? revision)}><td><button type="button" class="link" aria-pressed={r.revision === (viewing?.revision ?? revision)} onclick={() => action(() => view(r.revision))}>{r.revision}{r.revision === revision ? ' · 최신' : ''}</button></td><td>{r.name}</td><td class="mono" title={r.updatedBy}>{who(r.updatedBy)}</td><td class="mono">{formatTime(r.updatedAt)}</td></tr>
            {/each}
          </tbody></table>
        </details>
      {/if}
      {#if viewing}
        <div class="viewing" role="status" data-testid="revision-viewing">
          <p><b>revision {viewing.revision}</b> 보는 중 · 지금은 revision {revision} · <span title={viewing.updatedBy}>{who(viewing.updatedBy)}</span> · {formatTime(viewing.updatedAt)} · 지금과 다른 칸 {differs.size}개</p>
          {#if canEdit}<button type="button" onclick={() => action(restore)}>이 내용으로 되돌리기</button><small>이 내용을 새 revision으로 저장합니다. 지난 revision과 workspace가 고정한 revision은 바뀌지 않습니다.</small>{/if}
          <button type="button" onclick={() => { viewing = null }}>revision {revision}로 돌아가기</button>
        </div>
      {/if}
      {#if canEdit && !viewing}<label data-field-path="name">이름 <input name="name" autocomplete="off" aria-label="배포 문서 이름" bind:value={name} /></label>{/if}
      {#if problems.length}<section class="problems" aria-label="배포 문서 오류" data-testid="deployment-problems"><h4>고칠 곳 {problems.length}개</h4><ul>{#each problems as p}<li>{#if p.path}<button data-problem-path={p.path} onclick={() => focusProblem(p.path)}>{fieldTitle(contract?.[kind], p.path)}</button> · {/if}{p.message}</li>{/each}</ul></section>{/if}
      {#if contract}<DeploymentForm {kind} schema={contract[kind]} value={viewing?.content ?? content} readonly={!canEdit || !!viewing} changed={differs} problems={viewing ? {} : problemMap} change={v => { content = v; problems = [] }} />{/if}
      {#if canEdit && !viewing}
        <button disabled={busy} onclick={() => action(async () => { await check(); status = '엔진 검증 통과' })}>배포 문서 검증</button>
        <button disabled={busy} onclick={() => action(async () => { await check(); await saveDocument() })}>공유 문서 저장</button>
      {/if}
      {#if docId}<p>revision {revision}</p><button disabled={busy} onclick={() => action(exportDocument)}>공유 문서 내보내기</button>{/if}
    </div>
    <div>
      {#if canEdit}<BundleImport {webSession} onsaved={() => action(async () => { onsaved(); await refresh() })} />{/if}
      <h3>공유 workspace</h3>
      {#if workspaceId}<button disabled={busy} onclick={() => action(exportWorkspace)}>workspace 묶음 내보내기</button>{/if}
      <label>저장된 workspace <select name="deployment-editor-select-2" aria-label="저장된 배포 workspace" value={workspaceId} onchange={e => action(() => loadWorkspace(e.target.value))}><option value="">새 workspace</option>{#each workspaces as w}<option value={w.id}>{w.name} · revision {w.revision}</option>{/each}</select></label>
      {#if canEdit}
        <button onclick={() => { workspaceId = ''; workspaceRevision = 0; bindings = {} }}>새 공유 workspace</button>
        <label>이름 <input name="workspace-name" aria-label="배포 workspace 이름" bind:value={workspaceName} /></label>
        <label>서버 풀 revision <select name="set-ref" aria-label="workspace 서버 풀 revision" bind:value={setRef}><option value="">revision 선택</option>{#each docs.filter(d => d.kind === 'server-set') as d}<option value={`${d.id}:${d.revision}`}>{d.name} · {d.revision}</option>{/each}</select></label>
        <label>대상 경로 revision <select name="config-ref" aria-label="workspace 대상 경로 revision" bind:value={configRef}><option value="">revision 선택</option>{#each docs.filter(d => d.kind === 'workspace-config') as d}<option value={`${d.id}:${d.revision}`}>{d.name} · {d.revision}</option>{/each}</select></label>
        <button disabled={busy || !setRef || !configRef} onclick={() => action(saveWorkspace)}>공유 workspace 저장</button>
        {#if workspaceId}
          <h3>내 SSH 자격증명 연결</h3>
          <p>자격증명 등록과 폐기는 <a href="/settings">설정</a>에서 합니다. 여기서는 이 workspace의 서버에 내 자격증명을 연결합니다. 연결은 나에게만 보입니다.</p>
          <label>대상 서버 <select name="server-ref" aria-label="연결 서버" bind:value={serverRef}><option value="">서버 선택</option>{#each hosts as h}<option value={hostName(h)}>{hostName(h)}</option>{/each}</select></label>
          <label>내 자격증명 <select name="selected-credential" aria-label="연결 자격증명" bind:value={selectedCredential}><option value="">내 자격증명 선택</option>{#each credentials.filter(c => c.kind !== 'account-key' && !c.revoked) as c}<option value={c.id}>{c.label}</option>{/each}</select></label>
          <button disabled={busy || !serverRef || !selectedCredential} onclick={() => action(async () => { bindings = await api(`workspaces/${workspaceId}/credential-bindings`, 'PUT', { serverRef, credentialId: selectedCredential }); status = '내 SSH 연결 저장됨' })}>내 자격증명 연결</button>
          <button disabled={busy || !serverRef || !selectedCredential} onclick={() => action(async () => { access = await api(`credentials/${selectedCredential}/check`, 'POST', { workspaceId, serverRef }); status = access.allowedOperations.includes('deploy') ? 'SSH 배포 접근 확인됨' : access.reason ?? '접근 거절' })}>SSH 접근 확인</button>
          <p data-testid="personal-bindings">내 연결: {Object.keys(bindings).join(', ') || '없음'}</p>
          {#if access}<p data-testid="ssh-access">인증 {access.authenticated ? '성공' : '실패'} · 배포 {access.allowedOperations.includes('deploy') ? '허용' : '거절'} · {access.hostIdentity}</p>{/if}
        {/if}
      {/if}
    </div>
  </div></fieldset>
  <p role="status" data-testid="deployment-status">{status}</p>
</section>

<style>
  h3 { color: var(--text-muted); margin-top: var(--space-5); }
  .editor-controls { border: 0; padding: 0; margin: 0; min-width: 0; }
  .columns { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-5); }
  label { margin: var(--space-3) 0; }
  button { margin: var(--space-1); }
  .revisions { margin: var(--space-3) 0; } .revisions summary { cursor: pointer; padding: var(--space-2) 0; color: var(--text); font: 600 var(--text-sm) var(--font-cond); } .revisions .current { background: var(--surface-raised); } .link { min-height: 0; padding: 0; border: 0; background: none; color: var(--accent); text-decoration: underline; font-family: var(--font-mono); }
  .viewing { margin: var(--space-3) 0; padding: var(--space-2) var(--space-3); border: 1px solid var(--warn-border); background: var(--warn-bg); } .viewing p { margin: 0 0 var(--space-2); color: var(--text); } .viewing small { display: block; margin: var(--space-1) 0; }
  .problems { padding: var(--space-3); border: 1px solid var(--danger-border); border-radius: var(--radius); background: var(--danger-bg); color: var(--danger); } .problems button { min-height: 0; padding: 0; margin: 0; border: 0; background: none; color: var(--danger); text-decoration: underline; font-family: var(--font-mono); } h4 { margin: 0 0 var(--space-2); }
  [role=status] { color: var(--text-muted); overflow-wrap: anywhere; }
  @media (max-width: 700px) { [data-testid=document-revisions] :is(th, td):nth-child(2) { display: none; } .columns { grid-template-columns: 1fr; } }
</style>
