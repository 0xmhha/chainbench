<script>
  import { tick } from 'svelte'
  import Field from './DSLField.svelte'
  import { initial } from './dsl-form.js'
  import { statementLists, statementOf, schemaAt, settingsOf, withSettings, valueAt, withValue } from './dsl-tree.mjs'
  // errors are the engine's {path, message} list for this declaration.
  let { document, contract, errors = [], suggestions, onchange } = $props()
  let selected = $state('')
  const caseSchema = $derived(document.schemaVersion === '1' ? contract.$defs.v1Spec : contract.$defs.caseSpec)
  const lists = $derived(statementLists(document))
  const settings = $derived(settingsOf(document, caseSchema))
  const byPath = $derived(Object.fromEntries(errors.filter(e => e.path).map(e => [e.path, e.message])))
  const current = $derived(lists.flatMap(l => l.items).find(i => i.path === selected) ?? null)
  // A statement that was removed or moved away leaves the settings selected.
  $effect(() => { if (selected && !current) selected = '' })
  const listOf = path => path.slice(0, path.lastIndexOf('/'))
  const indexOf = path => Number(path.slice(path.lastIndexOf('/') + 1))
  function add(list) {
    const items = valueAt(document, list.path) ?? []
    onchange(withValue(document, list.path, [...items, initial(schemaAt(caseSchema, contract, list.path.replace('/content', '') + '/0'), contract)]))
    selected = `${list.path}/${items.length}`
  }
  function move(delta) {
    const list = listOf(selected), i = indexOf(selected), items = [...(valueAt(document, list) ?? [])]
    if (i + delta < 0 || i + delta >= items.length) return
    ;[items[i], items[i + delta]] = [items[i + delta], items[i]]
    onchange(withValue(document, list, items)); selected = `${list}/${i + delta}`
  }
  function duplicate() {
    const list = listOf(selected), i = indexOf(selected), items = [...(valueAt(document, list) ?? [])]
    items.splice(i + 1, 0, structuredClone($state.snapshot(items[i])))
    onchange(withValue(document, list, items)); selected = `${list}/${i + 1}`
  }
  function remove() { const at = selected; selected = ''; onchange(withValue(document, at, undefined)) }
  // Arrow keys walk the tree; with Alt they move the selected statement.
  function key(event, path) {
    if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
    event.preventDefault()
    if (event.altKey) { selected = path; move(event.key === 'ArrowDown' ? 1 : -1); tick().then(() => focusItem(selected)); return }
    const all = ['', ...lists.flatMap(l => l.items.map(i => i.path))], next = all[all.indexOf(path) + (event.key === 'ArrowDown' ? 1 : -1)]
    if (next !== undefined) { selected = next; focusItem(next) }
  }
  function focusItem(path) { globalThis.document.querySelector(`[data-tree-path="${path}"]`)?.focus() }
  // An error opens its statement and puts focus on the field it names, or on
  // the closest enclosing field that is shown.
  export async function show(path) {
    selected = statementOf(document, path)
    await tick()
    for (let at = path; at; at = at.slice(0, at.lastIndexOf('/'))) {
      const field = globalThis.document.querySelector(`[data-field-path="${at}"]`)
      const control = field?.querySelector('input, select, textarea, button')
      if (control) { control.focus(); return }
    }
  }
</script>

<div class="statement-editor">
  {#if errors.length}
    <section class="summary" aria-label="검증 오류" data-testid="dsl-error-summary">
      <h3>검증 오류 {errors.length}건</h3>
      <ul>{#each errors as e}<li><button onclick={() => show(e.path ?? '/content')}>{e.path ?? '/content'}</button> {e.message}</li>{/each}</ul>
    </section>
  {/if}
  <div class="columns">
    <nav aria-label="statement 트리">
      <button class="item" class:selected={selected === ''} aria-current={selected === '' ? 'true' : undefined} data-tree-path="" onclick={() => selected = ''} onkeydown={e => key(e, '')}>케이스 설정</button>
      {#each lists as l (l.path)}
        <h4>{l.label}</h4>
        <ul>{#each l.items as item (item.path)}<li>
          <button class="item" class:selected={selected === item.path} class:invalid={errors.some(e => statementOf(document, e.path ?? '') === item.path)} aria-current={selected === item.path ? 'true' : undefined} data-tree-path={item.path} onclick={() => selected = item.path} onkeydown={e => key(e, item.path)}>{indexOf(item.path) + 1}. {item.summary}</button>
        </li>{/each}</ul>
        <button class="add" onclick={() => add(l)}>{l.label} 추가</button>
      {/each}
      <p class="hint">위아래 화살표로 고르고, Alt와 함께 누르면 고른 statement를 옮깁니다.</p>
    </nav>
    <div class="form">
      {#if current}
        <div class="toolbar" role="group" aria-label="고른 statement">
          <button disabled={indexOf(selected) === 0} onclick={() => move(-1)}>위로</button>
          <button disabled={indexOf(selected) === (valueAt(document, listOf(selected))?.length ?? 1) - 1} onclick={() => move(1)}>아래로</button>
          <button onclick={duplicate}>복제</button>
          <button onclick={remove}>삭제</button>
        </div>
        {#key selected}<Field schema={schemaAt(caseSchema, contract, selected.replace('/content', ''))} root={contract} value={valueAt(document, selected)} path={selected} label={current.summary} {suggestions} errors={byPath} onchange={v => onchange(withValue(document, selected, v))} />{/key}
      {:else}
        <Field schema={settings.schema} root={contract} value={settings.value} path="/content" label="케이스 설정" {suggestions} errors={byPath} onchange={v => onchange(withSettings(document, v))} />
      {/if}
    </div>
  </div>
</div>

<style>
  .summary { padding: var(--space-3) var(--space-4); border: 1px solid var(--danger-border); border-radius: var(--radius); background: var(--danger-bg); color: var(--danger); }
  .summary h3 { margin: 0 0 var(--space-2); color: var(--danger); } .summary button { min-height: 0; padding: 0; border: 0; background: none; color: var(--danger); text-decoration: underline; font-family: var(--font-mono); }
  .columns { display: grid; grid-template-columns: minmax(200px, 280px) minmax(0, 1fr); gap: var(--space-4); margin-top: var(--space-4); }
  nav { border-right: 1px solid var(--border-subtle); padding-right: var(--space-3); } nav ul { list-style: none; padding: 0; margin: 0; } h4 { margin: var(--space-3) 0 var(--space-1); }
  .item { display: block; width: 100%; text-align: left; white-space: normal; background: none; border-color: transparent; overflow-wrap: anywhere; }
  .item.selected { background: var(--surface-raised); border-color: var(--accent); } .item.invalid { color: var(--danger); }
  .add, .toolbar button { margin: var(--space-1) var(--space-1) 0 0; }
  .hint { color: var(--text-muted); font-size: var(--text-xs); }
  @media (max-width: 800px) { .columns { grid-template-columns: 1fr; } nav { border-right: 0; } }
</style>
