<script>
  import { statusIcon } from './status-icon.mjs'
  import StatusMark from './StatusMark.svelte'
  import { stateLabel } from './labels.mjs'
  // A state is shown as its icon and its own words, never by color alone. The
  // Korean word leads; the engine's code follows so it stays searchable.
  let { state } = $props()
  const s = $derived(statusIcon(state))
  const label = $derived(stateLabel(state))
</script>
<span class="status {s.tone}"><StatusMark {state} /> {label || state}{#if label}<small class="code">{state}</small>{/if}</span>
<style>
  .status { white-space: nowrap; } .ok { color: var(--ok); } .bad { color: var(--danger); } .active { color: var(--accent); } .warn { color: var(--warn); } .neutral { color: var(--text-muted); }
  .code { display: inline; margin-left: var(--space-1); font: var(--text-xs) var(--font-mono); color: var(--text-faint); }
</style>
