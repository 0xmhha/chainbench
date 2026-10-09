<script>
  import { responseError } from './api-error.mjs'
  import Field from "./DSLField.svelte"
  let { webSession = undefined, assetRevision = 0, onsaved = () => {} } = $props()
  const writable=$derived(!webSession || webSession.user.role!=='viewer')
  let contract = $state(null)
  let presets = $state([])
  let selected = $state('')
  let document = $state(null)
  let inherited = $state(null)
  let result = $state(null)
  let error = $state('')
  let pending = $state(false)
  let requestVersion = 0
  let saved = $state([]), savedId = $state(''), revision = $state(0), preview = $state(null)
  let assets = $state([]), generatedGenesis = $state(null)
  const genesisAsset = $derived(document?.genesis?.ref?.startsWith('asset:') ? document.genesis.ref.slice(6) : '')
  const genesisChoices = $derived(assets.filter(a=>a.kind==='template'&&a.compatibility?.format==='json'))
  const assetRefs = $derived(genesisAsset ? [genesisAsset] : [])

  async function shared(path, method = 'GET', data, expectedRevision) {
    const headers = { 'Content-Type': 'application/json', 'X-CSRF-Token': webSession?.csrfToken ?? '' }
    if (expectedRevision) headers['If-Match'] = `"${expectedRevision}"`
    const response = await fetch('/api/v1/' + path, { method, headers, body: data === undefined ? undefined : JSON.stringify(data) })
    if (!response.ok) throw await responseError(response)
    return response.json()
  }
  async function refreshSaved() { saved = (await shared('documents?kind=chain-preset')).items }
  $effect(() => { void assetRevision; if (webSession) { refreshSaved().catch(e => { error = e.message }); shared('assets').then(v=>{assets=v.items}).catch(e=>{error=e.message}) } })
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
      const entry = await shared(savedId ? 'documents/' + savedId : 'documents', savedId ? 'PATCH' : 'POST', { kind: 'chain-preset', name: document.id, contractVersion: '2', content: document, assetRefs }, revision)
      savedId = entry.id; revision = entry.revision; onsaved(); await refreshSaved()
    } catch(e) { error = e.message } finally { pending = false }
  }
  function exportFile() {
    const url = URL.createObjectURL(new Blob([JSON.stringify(document, null, 2)], { type: 'application/json' }))
    const link = globalThis.document.createElement('a'); link.href = url; link.download = `${document.id}.json`; link.click(); URL.revokeObjectURL(url)
  }

  $effect(() => {
    fetch('/api/v1/chain-presets').then(async response => {
      if (!response.ok) return
      presets = await response.json()
    }).catch(() => { error = 'Could not load chain presets.' })
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
        if (!response.ok) throw new Error('Could not load chain contract.')
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
    if (value === undefined) return 'Engine default'
    return value === inherited.topology?.[field] ? 'Inherited from preset' : 'Your override'
  }

  async function validate() {
    const version = ++requestVersion
    pending = true
    error = ''
    try {
      const response = await fetch('/api/v1/documents/validate', {
        method: 'POST', headers: { 'Content-Type': 'application/json', ...(webSession?{'X-CSRF-Token':webSession.csrfToken}:{}) },
        body: JSON.stringify({ kind: 'chain-preset', name: document.id, contractVersion: '2', content: document, assetRefs })
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
    <p class="eyebrow">NETWORK CONFIGURATION</p>
    <h2 id="chain-heading">Start with a chain preset</h2>
    <p class="intro">Select an engine-supported declaration, then adjust its network layout.</p>
    {#if webSession}
      <label for="saved-chain-document">공유 체인 구성</label>
      <select id="saved-chain-document" value={savedId} onchange={e => { if(e.target.value) loadSaved(e.target.value) }}><option value="">새 구성</option>{#each saved as entry}<option value={entry.id}>{entry.name} · revision {entry.revision}</option>{/each}</select>
      {#if writable}<label>구성 import <input aria-label="체인 구성 import" type="file" accept=".json,.yaml,.yml" onchange={importFile} /></label>{/if}
      {#if preview}<p>Import 원문은 서버에 암호화되어 보존됩니다. {preview.changes.join('; ')}</p>{/if}
    {/if}
    <label for="chain-preset">Chain preset</label>
    <select id="chain-preset" bind:value={selected} onchange={choose}>
      <option value="">Choose a preset</option>
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
          {#each [['bp', 'Block producers'], ['en', 'Endpoints'], ['pn', 'Proxies']] as [field, label]}
            <div>
              <label for={`count-${field}`}>{label}</label>
              <input id={`count-${field}`} disabled={!writable} type="number" min={field === 'bp' ? 1 : 0} step="1"
                value={document.topology?.[field] ?? ''} oninput={event => update(field, event.target.value)} />
              <small>{origin(field)} · {document.topology?.[field] ?? (field === 'bp' ? 4 : 0)} applied</small>
            </div>
          {/each}
        </div>
        <details>
          <summary>Optional settings</summary>
          <label for="sync-mode">Sync mode</label>
          <select disabled={!writable} id="sync-mode" value={document.topology?.syncMode ?? ''} onchange={event => update('syncMode', event.target.value)}>
            <option value="">Engine default (full)</option>
            <option value="full">Full</option>
            <option value="snap">Snap</option>
            <option value="archive">Archive</option>
          </select>
          <small>{origin('syncMode')}</small>
        </details>
      {:else}
        <p>This preset declares each node individually. Its node table is preserved.</p>
      {/if}
      {#if contract && writable}
        <details class="structured"><summary>All configuration fields</summary>
          <Field schema={contract.$defs.envSpec} root={contract} value={document} label="Chain configuration" onchange={replace} />
        </details>
      {/if}
      {#if writable}<button onclick={validate} disabled={pending}>{pending ? 'Validating…' : 'Validate configuration'}</button>{/if}
      {#if webSession && writable}<button onclick={save} disabled={pending}>공유 체인 구성 저장</button>{/if}
      {#if result?.valid}<button onclick={exportFile} disabled={pending}>체인 구성 내보내기</button>{/if}
      {#if savedId}<p role="status">공유 문서 revision {revision}</p>{/if}
      {#if result}
        <div role="status" class:valid={result.valid}>
          <strong>{result.valid ? 'Engine validation passed' : 'Configuration needs attention'}</strong>
          {#each result.errors as issue}<p>{issue.path}: {issue.message}</p>{/each}
          {#each result.warnings as warning}<p>{warning}</p>{/each}
        </div>
      {/if}
      <details><summary>Generated declaration</summary><pre>{JSON.stringify(document, null, 2)}</pre></details>
    {/if}
    {#if error}<p role="alert">{error}</p>{/if}
  </section>
{/if}

<style>
  .editor { padding: 1.5rem; margin-top: 1.5rem; border: 1px solid #354455; border-radius: 12px; background: #151e29; }
  .eyebrow { color: #89baf5; font-size: 11px; letter-spacing: .12em; margin: 0; }
  h2 { font-size: 22px; margin: .5rem 0; color: #edf3fa; }
  .intro, .description { color: #aab6c6; }
  label { display: block; margin: .8rem 0 .3rem; }
  input, select { box-sizing: border-box; width: 100%; padding: .7rem; color: #e5edf7; background: #0e1620; border: 1px solid #495b70; border-radius: 6px; font: inherit; }
  .fields { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem; }
  small { display: block; color: #9eb0c5; padding-top: .4rem; }
  details { margin: 1rem 0; } summary { cursor: pointer; color: #bacce2; }
  button { padding: .7rem 1rem; border: 0; border-radius: 6px; background: #8fc2ff; color: #11263e; font: inherit; font-weight: 600; cursor: pointer; }
  button:disabled { opacity: .5; }
  [role=status] { border-left: 3px solid #f8ae72; padding: .8rem 1rem; margin-top: 1rem; }
  [role=status].valid { border-color: #72d6a7; } [role=alert] { color: #ffaaaa; }
  pre { overflow: auto; max-height: 360px; }
  @media(max-width: 600px) { .fields { grid-template-columns: 1fr; } }
</style>
