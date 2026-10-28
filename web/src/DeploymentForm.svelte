<script>
  import ContractField from './ContractField.svelte'
  import DeploymentFields from './DeploymentFields.svelte'
  import { schemaDefault } from './schema-problems.mjs'
  import { remainder } from './contract-form.mjs'
  // The deployment document laid out as a ledger: what an operator fills in
  // every time comes first as rows (servers, port bands, purpose directories),
  // and everything else the contract offers waits under "추가 설정" in the
  // general form, so a field the engine adds later still appears there.
  let { kind, schema, value, change, problems = {}, readonly = false, changed = new Set() } = $props()

  // Fields this layout places itself, below the content.
  const placed = {
    'server-set': ['version', 'pool.hosts', 'pool.slots', 'pool.ports'],
    'workspace-config': ['version', 'dataRoot', 'paths', 'control.artifactRoot', 'inputs', 'execution'],
  }
  // The document's own description is already above the form, not repeated here.
  const rest = $derived({ ...(placed[kind] ? remainder(schema, placed[kind]) : schema), description: undefined })
  const restHasValues = $derived(Object.keys(rest.properties ?? {}).some(k => value?.[k] !== undefined && !(rest.required ?? []).includes(k)))

  // set returns the content with one value replaced along keys.
  function set(keys, v) {
    const next = structuredClone($state.snapshot(value) ?? {})
    let at = next
    for (const k of keys.slice(0, -1)) { at[k] ??= {}; at = at[k] }
    if (v === undefined) delete at[keys.at(-1)]; else at[keys.at(-1)] = v
    change(next)
  }
  const prop = (...keys) => keys.reduce((s, k) => s?.properties?.[k], schema)
  const fieldProps = (p, keys) => ({ path: p, problems, readonly, changed: changed.has(p), change: v => set(keys, v) })

  // Servers: the engine also reads a bare address as a host, so a string row
  // is shown by its address and written back as an object when edited.
  const hostSpec = $derived(prop('pool', 'hosts')?.items)
  const hosts = $derived((value?.pool?.hosts ?? []).map(h => typeof h === 'string' ? { addr: h } : h))
  function setHost(i, k, v) { set(['pool', 'hosts'], hosts.map((h, j) => j === i ? { ...h, [k]: v } : h)) }

  const bands = ['p2p', 'rpc', 'ws', 'auth', 'metrics']
  const bandSpec = b => prop('pool', 'ports', b)
  // P2P and RPC are always shown: left empty, the engine fills its built-in band.
  const primary = ['p2p', 'rpc']
  const bandRequired = b => primary.includes(b) || prop('pool', 'ports')?.required?.includes(b)

  const pathKeys = $derived(Object.keys(prop('paths')?.properties ?? {}))
</script>

{#if kind === 'server-set'}
  <section class="block" aria-labelledby="servers-title">
    <h4 id="servers-title">{hostSpec?.title ?? '서버'}</h4>
    <p class="hint">{prop('pool', 'hosts')?.description}</p>
    <table class="rows">
      <thead><tr><th scope="col">#</th><th scope="col">{hostSpec?.properties?.name?.title} <code>name</code></th><th scope="col">{hostSpec?.properties?.addr?.title} <code>addr</code></th>{#if !readonly}<th scope="col"><span class="sr">삭제</span></th>{/if}</tr></thead>
      <tbody>
        {#each hosts as h, i (i)}
          <tr>
            <td class="num">{i + 1}</td>
            <td><ContractField bare schema={hostSpec.properties.name} value={h.name} label={`서버 ${i + 1} 이름`} {...fieldProps(`content.pool.hosts[${i}].name`, [])} change={v => setHost(i, 'name', v)} /></td>
            <td><ContractField bare schema={hostSpec.properties.addr} value={h.addr} label={`서버 ${i + 1} 주소`} {...fieldProps(`content.pool.hosts[${i}].addr`, [])} change={v => setHost(i, 'addr', v)} /></td>
            {#if !readonly}<td><button type="button" class="quiet" data-omit-path={`content.pool.hosts[${i}]`} onclick={() => set(['pool', 'hosts'], hosts.filter((_, j) => j !== i))}>서버 {i + 1} 삭제</button></td>{/if}
          </tr>
        {/each}
      </tbody>
    </table>
    {#if !readonly}<button type="button" data-add-path="content.pool.hosts" onclick={() => set(['pool', 'hosts'], [...hosts, { name: '', addr: '' }])}>서버 추가</button>{/if}
    <ContractField schema={prop('pool', 'slots')} value={value?.pool?.slots} {...fieldProps('content.pool.slots', ['pool', 'slots'])} />
  </section>

  <section class="block" aria-labelledby="bands-title">
    <h4 id="bands-title">{prop('pool', 'ports')?.title ?? '포트 대역'}</h4>
    <p class="hint">{prop('pool', 'ports')?.description}</p>
    <table class="rows">
      <thead><tr><th scope="col">대역</th><th scope="col">시작 포트 <code>base</code></th><th scope="col">간격 <code>step</code></th>{#if !readonly}<th scope="col"><span class="sr">빼기</span></th>{/if}</tr></thead>
      <tbody>
        {#each bands.filter(b => bandSpec(b) && (bandRequired(b) || value?.pool?.ports?.[b] !== undefined)) as b (b)}
          {@const spec = bandSpec(b)}
          <tr>
            <th scope="row">{spec.title} <code>{b}</code>{#if spec.description}<small>{spec.description}</small>{/if}</th>
            <td><ContractField bare schema={spec.properties.base} value={value?.pool?.ports?.[b]?.base} label={spec.properties.base.title} {...fieldProps(`content.pool.ports.${b}.base`, ['pool', 'ports', b, 'base'])} /></td>
            <td><ContractField bare schema={spec.properties.step} value={value?.pool?.ports?.[b]?.step} label={spec.properties.step.title} {...fieldProps(`content.pool.ports.${b}.step`, ['pool', 'ports', b, 'step'])} /></td>
            {#if !readonly}<td>{#if !bandRequired(b)}<button type="button" class="quiet" data-omit-path={`content.pool.ports.${b}`} onclick={() => set(['pool', 'ports', b], undefined)}>{spec.title} 빼기</button>{/if}</td>{/if}
          </tr>
        {/each}
      </tbody>
    </table>
    {#if !readonly}
      {#each bands.filter(b => bandSpec(b) && !bandRequired(b) && value?.pool?.ports?.[b] === undefined) as b (b)}
        <button type="button" data-add-path={`content.pool.ports.${b}`} onclick={() => set(['pool', 'ports', b], schemaDefault(bandSpec(b)))}>{bandSpec(b).title} 대역 추가</button>
      {/each}
    {/if}
  </section>
{:else if kind === 'workspace-config'}
  <section class="block">
    <ContractField schema={prop('dataRoot')} value={value?.dataRoot} {...fieldProps('content.dataRoot', ['dataRoot'])} />
  </section>
  <section class="block" aria-labelledby="paths-title">
    <h4 id="paths-title">{prop('paths')?.title ?? '용도별 디렉터리'}</h4>
    <p class="hint">{prop('paths')?.description}</p>
    <table class="rows">
      <thead><tr><th scope="col">용도</th><th scope="col">경로</th></tr></thead>
      <tbody>
        {#each pathKeys as k (k)}
          <tr>
            <th scope="row">{prop('paths', k).title} <code>{k}</code></th>
            <td><ContractField bare schema={prop('paths', k)} value={value?.paths?.[k]} label={`${prop('paths', k).title} 경로`} {...fieldProps(`content.paths.${k}`, ['paths', k])} /></td>
          </tr>
        {/each}
      </tbody>
    </table>
  </section>
  <section class="block">
    <ContractField schema={prop('control', 'artifactRoot')} value={value?.control?.artifactRoot} {...fieldProps('content.control.artifactRoot', ['control', 'artifactRoot'])} />
    <div class="pair">
      <ContractField schema={prop('inputs', 'mode')} value={value?.inputs?.mode} {...fieldProps('content.inputs.mode', ['inputs', 'mode'])} />
      {#if value?.inputs?.mode === 'existing' || value?.inputs?.name !== undefined}
        <ContractField schema={prop('inputs', 'name')} value={value?.inputs?.name} {...fieldProps('content.inputs.name', ['inputs', 'name'])} />
      {/if}
      <ContractField schema={prop('execution', 'chain')} value={value?.execution?.chain} {...fieldProps('content.execution.chain', ['execution', 'chain'])} />
    </div>
  </section>
{/if}

{#if Object.keys(rest.properties ?? {}).length || rest.additionalProperties}
  <details class="more" open={restHasValues || undefined}>
    <summary>추가 설정</summary>
    <DeploymentFields schema={rest} {value} {change} {problems} {readonly} {changed} title="" />
  </details>
{/if}

<style>
  .block { margin: var(--space-4) 0; }
  h4 { margin: 0; color: var(--text); }
  h4 + .hint { margin-top: var(--space-1); }
  .hint { margin: 0 0 var(--space-2); color: var(--text-muted); font-size: var(--text-xs); max-width: 70ch; }
  .rows { margin: var(--space-2) 0; }
  .rows th, .rows td { vertical-align: middle; }
  .rows tbody th { color: var(--text); font: 600 var(--text-sm) var(--font-cond); white-space: normal; }
  .rows th small { display: block; margin-top: 2px; font: 400 var(--text-xs) var(--font-sans, inherit); color: var(--text-muted); }
  th code { margin-left: var(--space-1); color: var(--text-faint); font-size: var(--text-xs); font-weight: 400; }
  .num { width: 2.5em; color: var(--text-muted); font-family: var(--font-mono); }
  .quiet { padding: 2px var(--space-2); font-size: var(--text-sm); }
  .block > button { margin: var(--space-2) var(--space-2) 0 0; }
  .pair { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 0 var(--space-4); }
  .sr { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); white-space: nowrap; }
  .more { margin-top: var(--space-4); border-top: 1px solid var(--border-subtle); padding-top: var(--space-2); }
  summary { cursor: pointer; padding: var(--space-2) 0; color: var(--text); font: 600 var(--text-sm) var(--font-cond); letter-spacing: .03em; }
  @media (max-width: 900px) {
    .rows thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
    .rows tr { display: grid; gap: var(--space-1); padding: var(--space-2) 0; border-bottom: 1px solid var(--border-subtle); }
    .rows td, .rows th { display: block; border: 0; padding: 0; }
    .rows td + td, .rows th + td { border-left: 0; }
  }
</style>
