<script>
  // One contract field: the contract's title is its name, the engine key sits
  // beside it so the YAML the CLI reads can be matched, the description sits
  // under it and the engine's refusal is written in red. data-field-path keeps
  // the form's path for the error summary and the acceptance scripts. A bare
  // field (a table cell) shows no label of its own; the column names it.
  let { schema, path, value, change, problems = {}, readonly = false, changed = false, label = undefined, bare = false } = $props()
  const problem = $derived(problems[path] ?? '')
  const id = $derived('field-' + path.replace(/[^A-Za-z0-9_-]/g, '_'))
  const key = $derived(path.split(/[.[\]]/).filter(Boolean).at(-1))
  const name = $derived(label ?? schema.title ?? key)
  const hint = $derived(!bare && schema.description ? id + '-hint' : '')
  const describedBy = $derived([hint, problem ? id + '-problem' : ''].filter(Boolean).join(' ') || undefined)
  const shown = $derived(value ?? (schema.type === 'boolean' ? false : ''))
  function number(raw) { return raw === '' ? undefined : Number(raw) }
</script>

<div class="field" class:bare class:changed data-field-path={path}>
  {#if bare}<span class="cell-label" aria-hidden="true">{name}</span>{/if}
  {#if !bare}<label class="name" for={id}>{name} <code>{key}</code>{#if changed}<span class="mark"> · 지금과 다름</span>{/if}</label>{/if}
  {#if schema.enum}
    <select {id} aria-label={bare ? name : undefined} aria-invalid={problem ? 'true' : undefined} aria-describedby={describedBy} disabled={readonly} value={shown} onchange={e => change(schema.type === 'integer' ? Number(e.currentTarget.value) : e.currentTarget.value)}>
      {#each schema.enum as option}<option value={option}>{option}</option>{/each}
    </select>
  {:else if schema.type === 'boolean'}
    <input {id} type="checkbox" aria-label={bare ? name : undefined} aria-invalid={problem ? 'true' : undefined} aria-describedby={describedBy} disabled={readonly} checked={shown === true} onchange={e => change(e.currentTarget.checked)} />
  {:else if schema.type === 'integer'}
    <input {id} type="number" inputmode="numeric" step="1" min={schema.minimum} max={schema.maximum} placeholder={schema.default !== undefined ? `기본 ${schema.default}` : undefined} aria-label={bare ? name : undefined} aria-invalid={problem ? 'true' : undefined} aria-describedby={describedBy} {readonly} value={shown} oninput={e => change(number(e.currentTarget.value))} />
  {:else}
    <input {id} aria-label={bare ? name : undefined} aria-invalid={problem ? 'true' : undefined} aria-describedby={describedBy} required={schema.minLength > 0} {readonly} value={shown} oninput={e => change(e.currentTarget.value)} />
  {/if}
  {#if hint}<p class="hint" id={hint}>{schema.description}</p>{/if}
  {#if problem}<p class="problem" id={id + '-problem'}>{problem}</p>{/if}
</div>

<style>
  .field { display: grid; gap: var(--space-1); margin: var(--space-3) 0; min-width: 0; }
  .field.bare { margin: 0; }
  .name { display: block; color: var(--text); font: 600 var(--text-sm) var(--font-cond); letter-spacing: .03em; }
  .name code { margin-left: var(--space-1); color: var(--text-faint); font-size: var(--text-xs); font-weight: 400; }
  .mark { color: var(--warn); font-weight: 400; }
  .changed input, .changed select { border-color: var(--warn); }
  .hint { margin: 0; color: var(--text-muted); font-size: var(--text-xs); max-width: 70ch; }
  .problem { margin: 0; color: var(--danger); font-size: var(--text-sm); }
  input[type=checkbox] { justify-self: start; }
  /* A read-only field is ruled, not boxed: it shows a value nobody can type into. */
  input[readonly], select:disabled { background: transparent; border-style: dashed; color: var(--text); opacity: 1; }
  .field:not(.bare) input[type=number] { max-width: 16ch; }
  /* In a table the column names a cell; once the table is stacked on a narrow
     screen, the cell names itself. */
  .cell-label { display: none; color: var(--text-muted); font-size: var(--text-xs); }
  @media (max-width: 900px) { .cell-label { display: block; } }
</style>
