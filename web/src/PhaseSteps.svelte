<script>
 // A job's phases as a row of ticks, one per phase in order, with the count
 // done and the phase now running (or the one that failed) written beside it.
 // The ticks are drawn, the words carry the meaning.
 let {phases=[]}=$props()
 const done=$derived(phases.filter(p=>p.state==='succeeded').length)
 const now=$derived(phases.find(p=>p.state==='failed')??phases.find(p=>p.state==='running')??null)
 const word=$derived(now?`${now.name}${now.state==='failed'?' 실패':''}`:done===phases.length&&phases.length?'모두 끝남':'대기')
</script>
<span class="phase-steps"><span class="ticks" aria-hidden="true">{#each phases as p}<i class={p.state} title="{p.name} · {p.state}"></i>{/each}</span><small class:failed={now?.state==='failed'}>{done}/{phases.length} · {word}</small></span>
<style>
 .phase-steps { display: inline-grid; gap: 2px; min-width: 0; }
 .ticks { display: flex; gap: 2px; width: 120px; max-width: 100%; }
 .ticks i { flex: 1; height: 6px; border: 1px solid var(--border-strong); }
 .ticks i.succeeded { background: var(--ok); border-color: var(--ok); }
 .ticks i.running { background: var(--accent); border-color: var(--accent); }
 .ticks i.failed { background: var(--danger); border-color: var(--danger); }
 small { display: block; font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
 small.failed { color: var(--danger); }
</style>
