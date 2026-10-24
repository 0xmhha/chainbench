<script>
  // A state's printed mark, drawn at one stroke weight: a tick, a cross, a
  // filled dot, a half-filled ring, a bang, an empty ring or a query. It is
  // decoration beside the state's words, so screen readers skip it.
  import { statusIcon } from './status-icon.mjs'
  let { state } = $props()
  const icon = $derived(statusIcon(state).icon)
</script>
<svg class="mark" viewBox="0 0 12 12" width="12" height="12" aria-hidden="true" focusable="false">
  {#if icon === 'check'}<path d="M2 6.5 5 9.5 10 3" />
  {:else if icon === 'cross'}<path d="M2.5 2.5 9.5 9.5M9.5 2.5 2.5 9.5" />
  {:else if icon === 'dot'}<circle cx="6" cy="6" r="3.5" class="fill" />
  {:else if icon === 'half'}<circle cx="6" cy="6" r="4" /><path d="M6 2a4 4 0 0 1 0 8z" class="fill" />
  {:else if icon === 'bang'}<path d="M6 1.5v6" /><circle cx="6" cy="10" r="1" class="fill" />
  {:else if icon === 'ring'}<circle cx="6" cy="6" r="4" />
  {:else}<path d="M4 4.5a2 2 0 1 1 2.6 1.9c-.4.2-.6.5-.6 1v.6" /><circle cx="6" cy="10" r=".9" class="fill" />{/if}
</svg>
<style>
  .mark { display: inline-block; flex: none; vertical-align: -1px; fill: none; stroke: currentColor; stroke-width: 1.6; stroke-linecap: square; }
  .fill { fill: currentColor; stroke: none; }
</style>
