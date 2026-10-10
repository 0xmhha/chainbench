<script>
  import DeploymentFields from "./DeploymentFields.svelte"
  let { schema, value, path = 'content', change } = $props()
  let propertyName = $state('')
  function defaultValue(s) {
    if (s.enum) return s.enum[0]
    if (s.type === 'object') return Object.fromEntries((s.required ?? []).map(k => [k, defaultValue(s.properties[k])]))
    if (s.type === 'array') return []
    if (s.type === 'integer') return 0
    if (s.type === 'boolean') return false
    return ''
  }
  function field(k, v) { change({ ...value, [k]: v }) }
  function omit(k) { const next = { ...value }; delete next[k]; change(next) }
</script>

{#if schema.type === 'object'}
  <fieldset>
    <legend>{path}</legend>
    {#each Object.entries(schema.properties ?? {}) as [key, spec] (key)}
      {@const required = schema.required?.includes(key)}
      {#if required || value?.[key] !== undefined}
        <div class="field">
          <DeploymentFields schema={spec} value={value?.[key]} path={`${path}.${key}`} change={v => field(key, v)} />
          {#if !required}<button type="button" onclick={() => omit(key)}>Omit {key}</button>{/if}
        </div>
      {:else}
        <button type="button" onclick={() => field(key, defaultValue(spec))}>Add {key}</button>
      {/if}
    {/each}
    {#if schema.additionalProperties}
      {#each Object.entries(value ?? {}) as [key, entry] (key)}
        <DeploymentFields schema={schema.additionalProperties} value={entry} path={`${path}.${key}`} change={v => field(key, v)} />
        <button type="button" onclick={() => omit(key)}>Remove {key}</button>
      {/each}
      <label>Entry name <input aria-label={`${path}.entry-name`} bind:value={propertyName} /></label>
      <button type="button" disabled={!propertyName || Object.hasOwn(value ?? {}, propertyName)} onclick={() => { field(propertyName, defaultValue(schema.additionalProperties)); propertyName = '' }}>Add entry</button>
    {/if}
  </fieldset>
{:else if schema.type === 'array'}
  <fieldset>
    <legend>{path}</legend>
    {#each value ?? [] as entry, i (i)}
      <DeploymentFields schema={schema.items} value={entry} path={`${path}[${i}]`} change={v => change(value.map((old, j) => j === i ? v : old))} />
      <button type="button" onclick={() => change(value.filter((_, j) => j !== i))}>Remove item {i + 1}</button>
    {/each}
    <button type="button" onclick={() => change([...(value ?? []), defaultValue(schema.items)])}>Add host</button>
  </fieldset>
{:else if schema.enum}
  <label>{path}<select aria-label={path} value={value} onchange={e => change(schema.type === 'integer' ? Number(e.target.value) : e.target.value)}>
    {#each schema.enum as option}<option value={option}>{option}</option>{/each}
  </select></label>
{:else if schema.type === 'boolean'}
  <label><input type="checkbox" aria-label={path} checked={value ?? false} onchange={e => change(e.target.checked)} /> {path}</label>
{:else if schema.type === 'integer'}
  <label>{path}<input aria-label={path} type="number" step="1" min={schema.minimum} max={schema.maximum} value={value ?? 0} oninput={e => change(e.target.valueAsNumber)} /></label>
{:else}
  <label>{path}<input aria-label={path} value={value ?? ''} oninput={e => change(e.target.value)} /></label>
{/if}

<style>
  fieldset { border: 1px solid #36465b; border-radius: 8px; padding: 12px; margin: 10px 0; min-width: 0; }
  legend { color: #a4b9d7; font-size: 12px; }
  label { display: grid; gap: 5px; margin: 8px 0; overflow-wrap: anywhere; }
  input, select, button { font: inherit; color: inherit; background: #182332; border: 1px solid #4e6380; border-radius: 5px; padding: 7px; }
  button { margin: 4px; cursor: pointer; }
  .field { margin: 5px 0; }
</style>
