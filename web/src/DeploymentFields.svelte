<script>
  import DeploymentFields from "./DeploymentFields.svelte"
  import ContractField from "./ContractField.svelte"
  import { schemaDefault } from "./schema-problems.mjs"
  // The general form for contract fields a laid-out form does not place
  // itself: objects become titled groups, optional fields are added and left
  // out by their titles, and maps and lists grow entry by entry. problems maps
  // a field path to the engine's refusal; changed is the set of paths a past
  // revision differs at.
  let { schema, value, path = 'content', change, problems = {}, readonly = false, changed = new Set(), title = undefined } = $props()
  let entryName = $state('')
  const key = $derived(path.split(/[.[\]]/).filter(Boolean).at(-1))
  function field(k, v) { change({ ...value, [k]: v }) }
  function omit(k) { const next = { ...value }; delete next[k]; change(next) }
</script>

{#if schema.type === 'object'}
  <fieldset data-field-group={path}>
    {#if title ?? schema.title}<legend>{title ?? schema.title} <code>{key}</code></legend>{/if}
    {#if schema.description}<p class="hint">{schema.description}</p>{/if}
    {#each Object.entries(schema.properties ?? {}) as [k, spec] (k)}
      {@const required = schema.required?.includes(k)}
      {#if required || value?.[k] !== undefined}
        <div class="entry">
          <DeploymentFields schema={spec} value={value?.[k]} path={`${path}.${k}`} {problems} {readonly} {changed} change={v => field(k, v)} />
          {#if !required && !readonly}<button type="button" class="quiet" data-omit-path={`${path}.${k}`} onclick={() => omit(k)}>{spec.title ?? k} 빼기</button>{/if}
        </div>
      {:else if !readonly}
        <button type="button" class="quiet" data-add-path={`${path}.${k}`} onclick={() => field(k, schemaDefault(spec))}>{spec.title ?? k} 추가</button>
      {/if}
    {/each}
    {#if schema.additionalProperties}
      {#each Object.entries(value ?? {}) as [k, entry] (k)}
        <div class="entry">
          <DeploymentFields schema={schema.additionalProperties} value={entry} path={`${path}.${k}`} title={k} {problems} {readonly} {changed} change={v => field(k, v)} />
          {#if !readonly}<button type="button" class="quiet" data-omit-path={`${path}.${k}`} onclick={() => omit(k)}>{k} 삭제</button>{/if}
        </div>
      {/each}
      {#if !readonly}
        <div class="add-entry">
          <label>새 항목 이름 <input name="property-name" aria-label={`${schema.title ?? key} 새 항목 이름`} bind:value={entryName} /></label>
          <button type="button" data-add-path={`${path}.*`} disabled={!entryName || Object.hasOwn(value ?? {}, entryName)} onclick={() => { field(entryName, schemaDefault(schema.additionalProperties)); entryName = '' }}>항목 추가</button>
        </div>
      {/if}
    {/if}
  </fieldset>
{:else if schema.type === 'array'}
  <fieldset data-field-group={path}>
    <legend>{title ?? schema.title ?? key} <code>{key}</code></legend>
    {#each value ?? [] as entry, i (i)}
      <div class="entry">
        <DeploymentFields schema={schema.items} value={entry} path={`${path}[${i}]`} title={`${schema.items.title ?? schema.title ?? key} ${i + 1}`} {problems} {readonly} {changed} change={v => change(value.map((old, j) => j === i ? v : old))} />
        {#if !readonly}<button type="button" class="quiet" data-omit-path={`${path}[${i}]`} onclick={() => change(value.filter((_, j) => j !== i))}>{i + 1}번 삭제</button>{/if}
      </div>
    {/each}
    {#if !readonly}<button type="button" data-add-path={path} onclick={() => change([...(value ?? []), schemaDefault(schema.items)])}>항목 추가</button>{/if}
  </fieldset>
{:else}
  <ContractField {schema} {path} {value} {change} {problems} {readonly} changed={changed.has(path)} label={title} />
{/if}

<style>
  fieldset { padding: var(--space-2) 0 var(--space-3); }
  legend code, .add-entry code { color: var(--text-faint); font-size: var(--text-xs); font-weight: 400; margin-left: var(--space-1); }
  .hint { margin: 0 0 var(--space-2); color: var(--text-muted); font-size: var(--text-xs); max-width: 70ch; }
  .entry { border-bottom: 1px solid var(--border-subtle); padding-bottom: var(--space-1); }
  .quiet { padding: 2px var(--space-2); margin: var(--space-1) var(--space-2) var(--space-1) 0; font-size: var(--text-sm); }
  .add-entry { display: flex; flex-wrap: wrap; align-items: end; gap: var(--space-2); margin-top: var(--space-2); }
</style>
