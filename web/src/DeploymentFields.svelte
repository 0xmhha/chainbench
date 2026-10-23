<script>
  import DeploymentFields from "./DeploymentFields.svelte"
  // problems maps a field path to what is wrong with it; that field is marked
  // invalid and described by the message.
  let { schema, value, path = 'content', change, problems = {} } = $props()
  const problem = $derived(problems[path] ?? '')
  const invalid = $derived(problem ? { 'aria-invalid': 'true', 'aria-describedby': `${path}-problem` } : {})
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
          <DeploymentFields schema={spec} value={value?.[key]} path={`${path}.${key}`} {problems} change={v => field(key, v)} />
          {#if !required}<button type="button" onclick={() => omit(key)}>{key} 빼기</button>{/if}
        </div>
      {:else}
        <button type="button" onclick={() => field(key, defaultValue(spec))}>{key} 추가</button>
      {/if}
    {/each}
    {#if schema.additionalProperties}
      {#each Object.entries(value ?? {}) as [key, entry] (key)}
        <DeploymentFields schema={schema.additionalProperties} value={entry} path={`${path}.${key}`} {problems} change={v => field(key, v)} />
        <button type="button" onclick={() => omit(key)}>{key} 삭제</button>
      {/each}
      <label>항목 이름 <input name="property-name" aria-label={`${path}.entry-name`} bind:value={propertyName} /></label>
      <button type="button" disabled={!propertyName || Object.hasOwn(value ?? {}, propertyName)} onclick={() => { field(propertyName, defaultValue(schema.additionalProperties)); propertyName = '' }}>항목 추가</button>
    {/if}
  </fieldset>
{:else if schema.type === 'array'}
  <fieldset>
    <legend>{path}</legend>
    {#each value ?? [] as entry, i (i)}
      <DeploymentFields schema={schema.items} value={entry} path={`${path}[${i}]`} {problems} change={v => change(value.map((old, j) => j === i ? v : old))} />
      <button type="button" onclick={() => change(value.filter((_, j) => j !== i))}>{i + 1}번 항목 삭제</button>
    {/each}
    <button type="button" onclick={() => change([...(value ?? []), defaultValue(schema.items)])}>항목 추가</button>
  </fieldset>
{:else if schema.enum}
  <label data-field-path={path}>{path}<select name={path} aria-label={path} {...invalid} value={value} onchange={e => change(schema.type === 'integer' ? Number(e.target.value) : e.target.value)}>
    {#each schema.enum as option}<option value={option}>{option}</option>{/each}
  </select></label>
{:else if schema.type === 'boolean'}
  <label data-field-path={path}><input name={path} type="checkbox" aria-label={path} {...invalid} checked={value ?? false} onchange={e => change(e.target.checked)} /> {path}</label>
{:else if schema.type === 'integer'}
  <label data-field-path={path}>{path}<input name={path} aria-label={path} {...invalid} type="number" step="1" min={schema.minimum} max={schema.maximum} value={value ?? 0} oninput={e => change(e.target.valueAsNumber)} /></label>
{:else}
  <label data-field-path={path}>{path}<input name={path} aria-label={path} {...invalid} value={value ?? ''} oninput={e => change(e.target.value)} /></label>
{/if}
{#if problem}<p id={`${path}-problem`} class="problem">{problem}</p>{/if}

<style>
  fieldset { padding: var(--space-2) 0 var(--space-2) var(--space-3); margin: var(--space-2) 0; border-top: 0; border-left: 1px solid var(--border-subtle); }
  legend { font-size: var(--text-xs); }
  label { margin: var(--space-2) 0; overflow-wrap: anywhere; }
  .problem { color: var(--danger); margin: 0 0 var(--space-2); }
  button { margin: var(--space-1); }
  .field { margin: var(--space-1) 0; }
</style>
