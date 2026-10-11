<script>
  import { responseError } from './api-error.mjs'
  import { allItems } from './pages.mjs'
  import { draftMeta, contractChanged, changedDefinitions } from './contract-draft.mjs'
  import Field from "./DSLField.svelte"
  import { tick } from 'svelte'
  import { topologyChoices } from './chain-contract.mjs'
  import ChainSurfacePanel from './ChainSurfacePanel.svelte'
  import { selectChain } from './selection.svelte.js'
  let { webSession = undefined, assetRevision = 0, onsaved = () => {} } = $props()
  const writable=$derived(!webSession || webSession.user.role!=='viewer')
  let contract = $state(null)
  let presets = $state([])
  let selected = $state('')
  let document = $state(null)
  let inherited = $state(null)
  let result = $state(null), fieldsOpen = $state(false)
  // An engine error is shown on its field in the full form, and its summary
  // entry opens that form and focuses the field (or the closest one shown).
  const errorMap = $derived(Object.fromEntries((result?.errors ?? []).filter(e => e.path?.startsWith('/content')).map(e => [e.path, e.message])))
  async function showIssue(path) {
    fieldsOpen = true
    await tick()
    for (let at = path; at.startsWith('/content'); at = at.slice(0, at.lastIndexOf('/'))) {
      const control = globalThis.document.querySelector(`.structured [data-field-path="${at}"]`)?.querySelector('input, select, textarea')
      if (control) { control.focus(); return }
    }
  }
  let error = $state('')
  let pending = $state(false)
  let requestVersion = 0
  let saved = $state([]), savedId = $state(''), revision = $state(0), preview = $state(null)
  let assets = $state([]), generatedGenesis = $state(null)
  const genesisAsset = $derived(document?.genesis?.ref?.startsWith('asset:') ? document.genesis.ref.slice(6) : '')
  const genesisChoices = $derived(assets.filter(a=>a.kind==='template'&&a.compatibility?.format==='json'))
  const assetRefs = $derived(genesisAsset ? [genesisAsset] : [])
  const topology = $derived(topologyChoices(contract))
  // The chain being configured becomes every screen's selected chain.
  $effect(() => { if (document?.chain) selectChain(document.chain) })

  async function shared(path, method = 'GET', data, expectedRevision) {
    const headers = { 'Content-Type': 'application/json', 'X-CSRF-Token': webSession?.csrfToken ?? '' }
    if (expectedRevision) headers['If-Match'] = `"${expectedRevision}"`
    const response = await fetch('/api/v1/' + path, { method, headers, body: data === undefined ? undefined : JSON.stringify(data) })
    if (!response.ok) throw await responseError(response)
    return response.json()
  }
  async function refreshSaved() { saved = await allItems(shared, 'documents?kind=chain-preset') }
  $effect(() => { void assetRevision; if (webSession) { refreshSaved().catch(e => { error = e.message }); allItems(shared, 'assets').then(v=>{assets=v}).catch(e=>{error=e.message}) } })
  function chooseGenesis(id) {
    if (id) {
      if (!genesisAsset) generatedGenesis = structuredClone($state.snapshot(document.genesis??{mode:'template'}))
      replace({...document,genesis:{mode:'existing',ref:'asset:'+id}})
    } else replace({...document,genesis:generatedGenesis??{mode:'template'}})
  }
  async function loadSaved(id) {
    try {
      const entry = await shared('documents/' + encodeURIComponent(id))
      savedId = entry.id; revision = entry.revision; selected = ''; preview = null; generatedGenesis = null
      replace(structuredClone(entry.content)); inherited = structuredClone(entry.content)
      contract = await shared('contracts/chain-preset?chain=' + encodeURIComponent(entry.content.chain))
    } catch(e) { error = e.message }
  }
  async function importFile(event) {
    const file = event.target.files[0]; if (!file) return
    pending = true; error = ''
    try {
      const imported = await shared('documents/import', 'POST', { filename: file.name, format: file.name.endsWith('.json') ? 'json' : 'yaml', source: await file.text(), kind: 'chain-preset' })
      preview = imported
      if (!imported.validation.valid) throw new Error(imported.validation.errors.map(e => e.message).join('; '))
      const entry = imported.redactedDocuments[0]
      savedId = ''; revision = 0; selected = ''; generatedGenesis = null; replace(structuredClone(entry.content)); inherited = structuredClone(entry.content)
      contract = await shared('contracts/chain-preset?chain=' + encodeURIComponent(entry.content.chain))
      result = imported.validation
    } catch(e) { error = e.message } finally { pending = false }
  }
  async function save() {
    pending = true; error = ''
    try {
      const entry = await shared(savedId ? 'documents/' + savedId : 'documents', savedId ? 'PATCH' : 'POST', { kind: 'chain-preset', name: document.id, ...draftMeta(contract), content: document, assetRefs }, revision)
      savedId = entry.id; revision = entry.revision; onsaved(); await refreshSaved()
    } catch(e) {
      error = e.message
      if (contractChanged(e.message)) {
        // The engine contract changed under the draft: load the new one,
        // validate the draft against it and say which parts differ.
        const before = contract
        contract = await shared('contracts/chain-preset?chain=' + encodeURIComponent(document.chain))
        await validate()
        error = '엔진 계약이 바뀌었습니다. 바뀐 부분: ' + (changedDefinitions(before, contract).join(', ') || '세부 항목') + '. 새 계약으로 다시 검증했으니 결과를 확인한 뒤 다시 저장하세요.'
      }
    } finally { pending = false }
  }
  function exportFile() {
    const url = URL.createObjectURL(new Blob([JSON.stringify(document, null, 2)], { type: 'application/json' }))
    const link = globalThis.document.createElement('a'); link.href = url; link.download = `${document.id}.json`; link.click(); URL.revokeObjectURL(url)
  }

  $effect(() => {
    fetch('/api/v1/chain-presets').then(async response => {
      if (!response.ok) return
      presets = await response.json()
    }).catch(() => { error = '체인 프리셋을 불러오지 못했습니다.' })
  })

  async function choose(event) {
    selected = event.target.value
    requestVersion++
    const preset = presets.find(p => p.id === selected)
    document = preset ? structuredClone($state.snapshot(preset.content)) : null
    inherited = preset ? structuredClone($state.snapshot(preset.content)) : null
    result = null
    error = ''
    pending = false
    contract = null
    savedId = ''; revision = 0; preview = null; generatedGenesis = null
    if (preset) {
      try {
        const response = await fetch(`/api/v1/contracts/chain-preset?chain=${encodeURIComponent(preset.chain)}`)
        if (!response.ok) throw new Error('체인 계약을 불러오지 못했습니다.')
        const schema = await response.json()
        if (selected === preset.id) contract = schema
      } catch (e) { if (selected === preset.id) error = e.message }
    }
  }

  function update(field, value) {
    requestVersion++
    const topology = { ...(document.topology ?? {}) }
    if (value === '') delete topology[field]
    else topology[field] = field === 'syncMode' ? value : Number(value)
    document = { ...document, topology }
    result = null
    pending = false
  }

  function replace(value) {
    requestVersion++
    document = value
    result = null
    pending = false
  }

  function origin(field) {
    const value = document.topology?.[field]
    if (value === undefined) return '엔진 기본값'
    return value === inherited.topology?.[field] ? '프리셋에서 물려받음' : '직접 바꿈'
  }

  async function validate() {
    const version = ++requestVersion
    pending = true
    error = ''
    try {
      const response = await fetch('/api/v1/documents/validate', {
        method: 'POST', headers: { 'Content-Type': 'application/json', ...(webSession?{'X-CSRF-Token':webSession.csrfToken}:{}) },
        body: JSON.stringify({ kind: 'chain-preset', name: document.id, contractVersion: draftMeta(contract).contractVersion, content: document, assetRefs })
      })
      if (!response.ok) throw await responseError(response)
      const validation = await response.json()
      if (version === requestVersion) result = validation
    } catch (e) {
      if (version === requestVersion) error = e.message
    } finally {
      if (version === requestVersion) pending = false
    }
  }
</script>

{#if presets.length || webSession}
  <section class="editor" aria-labelledby="chain-heading">
    <p class="eyebrow">네트워크 구성</p>
    <h2 id="chain-heading">체인 프리셋에서 시작</h2>
    <p class="intro">엔진이 지원하는 선언을 고른 뒤 네트워크 배치를 조정합니다.</p>
    {#if webSession}
      <label for="saved-chain-document">공유 체인 구성</label>
      <select id="saved-chain-document" value={savedId} onchange={e => { if(e.target.value) loadSaved(e.target.value) }}><option value="">새 구성</option>{#each saved as entry}<option value={entry.id}>{entry.name} · revision {entry.revision}</option>{/each}</select>
      {#if writable}<label>구성 import <input aria-label="체인 구성 import" type="file" accept=".json,.yaml,.yml" onchange={importFile} /></label>{/if}
      {#if preview}<p>Import 원문은 서버에 암호화되어 보존됩니다. {preview.changes.join('; ')}</p>{/if}
    {/if}
    <label for="chain-preset">체인 프리셋</label>
    <select id="chain-preset" bind:value={selected} onchange={choose}>
      <option value="">프리셋 선택</option>
      {#each presets as preset (preset.id)}
        <option value={preset.id}>{preset.chain} · {preset.id}</option>
      {/each}
    </select>
    {#if document}
      <p class="description">{presets.find(p => p.id === selected)?.description}</p>
      {#if webSession}
        <label>완성된 genesis 자료<select aria-label="완성된 genesis 자료" disabled={!writable} value={genesisAsset} onchange={e=>chooseGenesis(e.currentTarget.value)}><option value="">프리셋에서 생성</option>{#if genesisAsset&&!genesisChoices.some(a=>a.id===genesisAsset)}<option value={genesisAsset}>사용할 수 없는 자료 · {genesisAsset}</option>{/if}{#each genesisChoices as a}<option value={a.id}>{a.name} · {a.checksum.slice(0,12)}</option>{/each}</select></label>
        <small>완성 파일을 그대로 사용하며 기존 genesis set/overlay를 대체합니다. 이번 편집에서 선택한 파일을 해제하면 이전 생성 설정으로 돌아갑니다. 저장된 파일 구성에는 기본 생성 설정이 적용됩니다.</small>
      {/if}
      {#if !document.topology?.nodes}
        <div class="fields">
          {#each topology.counts as f}
            <div>
              <label for={`count-${f.id}`}>{f.title}</label>
              <input id={`count-${f.id}`} disabled={!writable} type="number" min={f.minimum} step="1"
                value={document.topology?.[f.id] ?? ''} oninput={event => update(f.id, event.target.value)} />
              <small>{origin(f.id)} · {document.topology?.[f.id] ?? topology.defaults[f.id]}개 적용</small>
            </div>
          {/each}
        </div>
        <details>
          <summary>선택 설정</summary>
          {#if topology.sync}
            <label for="sync-mode">{topology.sync.title}</label>
            <select disabled={!writable} id="sync-mode" value={document.topology?.syncMode ?? ''} onchange={event => update('syncMode', event.target.value)}>
              <option value="">엔진 기본값 ({topology.sync.default})</option>
              {#each topology.sync.options as option}<option value={option}>{option}</option>{/each}
            </select>
            <small>{origin('syncMode')}</small>
          {/if}
        </details>
      {:else}
        <p>이 프리셋은 노드를 하나씩 선언합니다. 노드 표는 그대로 유지됩니다.</p>
      {/if}
      {#if contract && writable}
        <details class="structured" bind:open={fieldsOpen}><summary>모든 구성 필드</summary>
          <Field schema={contract.$defs.envSpec} root={contract} value={document} label="체인 구성" errors={errorMap} onchange={replace} />
        </details>
      {/if}
      {#if writable}<button onclick={validate} disabled={pending}>{pending ? '검증 중…' : '체인 구성 검증'}</button>{/if}
      {#if webSession && writable}<button onclick={save} disabled={pending}>공유 체인 구성 저장</button>{/if}
      {#if result?.valid}<button onclick={exportFile} disabled={pending}>체인 구성 내보내기</button>{/if}
      {#if savedId}<p role="status">공유 문서 revision {revision}</p>{/if}
      {#if result}
        <div role="status" class:valid={result.valid}>
          <strong><span aria-hidden="true">{result.valid ? '✓' : '✕'}</span> {result.valid ? '엔진 검증 통과' : '고쳐야 할 곳이 있습니다'}</strong>
          {#each result.errors as issue}<p><button class="link" onclick={() => showIssue(issue.path)}>{issue.path}</button> {issue.message}</p>{/each}
          {#each result.warnings as warning}<p>{warning}</p>{/each}
        </div>
      {/if}
      <details><summary>생성된 선언</summary><pre>{JSON.stringify(document, null, 2)}</pre></details>
      {#if webSession && document.chain}<ChainSurfacePanel {webSession} chain={document.chain} />{/if}
    {/if}
    {#if error}<p role="alert">{error}</p>{/if}
  </section>
{/if}

<style>
  .editor { padding: 24px; margin-top: 24px; border: 1px solid var(--border); border-radius: 10px; background: var(--surface); }
  .eyebrow { color: var(--accent); font-size: 11px; letter-spacing: .12em; margin: 0; }
  h2 { font-size: 22px; margin: 8px 0; color: var(--text); }
  .intro, .description { color: var(--text-muted); }
  label { display: block; margin: .8rem 0 .3rem; }
  input, select { box-sizing: border-box; width: 100%; padding: 10px; color: var(--text); background: var(--bg); border: 1px solid var(--border-strong); border-radius: 6px; font: inherit; }
  .fields { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem; }
  small { display: block; color: var(--text-muted); padding-top: 6px; }
  details { margin: 16px 0; } summary { cursor: pointer; color: var(--accent-strong); }
  button { padding: 10px 16px; border: 0; border-radius: 6px; background: var(--accent); color: var(--bg); font: inherit; font-weight: 600; cursor: pointer; }
  button.link { background: none; color: var(--danger); padding: 0; text-decoration: underline; font-weight: 400; font-family: ui-monospace, monospace; }
  button:disabled { opacity: .5; }
  [role=status] { border-left: 3px solid var(--warn); padding: 12px 16px; margin-top: 16px; }
  [role=status].valid { border-color: var(--ok); } [role=alert] { color: var(--danger); }
  pre { overflow: auto; max-height: 360px; }
  @media(max-width: 600px) { .fields { grid-template-columns: 1fr; } }
</style>
