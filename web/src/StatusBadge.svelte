<script>
  import { statusIcon } from './status-icon.mjs'
  // A state is shown as its icon and its own words, never by color alone.
  let { state } = $props()
  const s = $derived(statusIcon(state))
</script>
<span class="status {s.tone}"><span aria-hidden="true">{s.icon}</span> {state}</span>
<style>
  .status { white-space: nowrap; } .ok { color: var(--ok); } .bad { color: var(--danger); } .active { color: var(--accent); } .warn { color: var(--warn); } .neutral { color: var(--text-muted); }
</style>
