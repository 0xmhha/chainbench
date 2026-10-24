<script>
 // The job's phases as watch entries in order, one ruled row each. Only the
 // failed watch starts open; it is closed with a double red rule.
 import StatusBadge from './StatusBadge.svelte'
 import {formatTime} from './labels.mjs'
 let {rows=[]}=$props()
 const seconds=ms=>ms===null?'—':ms<60000?`${(ms/1000).toFixed(1)}초`:`${Math.floor(ms/60000)}분 ${Math.round((ms%60000)/1000)}초`
</script>
<section class="watches" id="phases" aria-label="단계">
 <h3>단계 <small>{rows.length}개 · 당직 순서</small></h3>
 <ol>
  {#each rows as p, i (i)}
   <li class={p.state}><details open={p.open}>
    <summary><span class="t">{p.startedAt?formatTime(p.startedAt).slice(11):'—'}</span><StatusBadge state={p.state} /><span class="name">{p.name}{#if p.message&&!p.open}<small> · {p.message}</small>{/if}</span><span class="d">{seconds(p.durationMs)}</span></summary>
    <dl><dt>시각</dt><dd class="t">{p.startedAt?`${formatTime(p.startedAt)} – ${formatTime(p.finishedAt)}`:'기록 없음'}</dd>{#if p.message}<dt>메시지</dt><dd class="msg">{p.message}</dd>{/if}</dl>
   </details></li>
  {/each}
 </ol>
</section>
<style>
 .watches { padding-top: var(--space-4); }
 h3 small { margin-left: var(--space-2); font: 400 var(--text-xs) var(--font-sans); color: var(--text-muted); }
 ol { margin: var(--space-2) 0 0; padding: 0; list-style: none; border-top: var(--rule-strong); }
 li { border-bottom: 1px solid var(--border); }
 li.failed { border-bottom: 3px double var(--danger); }
 summary { display: grid; grid-template-columns: 6.5rem 9rem minmax(0, 1fr) auto; align-items: baseline; gap: var(--space-3); min-height: 40px; padding: var(--space-2) 0; cursor: pointer; }
 summary:hover { background: var(--surface); }
 .t { font: 700 var(--text-sm) var(--font-mono); font-variant-numeric: tabular-nums; }
 .name { font: var(--text-sm) var(--font-mono); overflow-wrap: anywhere; }
 .name small { color: var(--text-muted); font-family: var(--font-sans); }
 .d { font: var(--text-sm) var(--font-mono); color: var(--text-muted); }
 li.failed .t, li.failed .name, li.failed .msg { color: var(--danger); }
 li.running .name { color: var(--accent); }
 dl { display: grid; grid-template-columns: 5rem minmax(0, 1fr); gap: var(--space-1) var(--space-3); margin: 0; padding: 0 0 var(--space-3) calc(6.5rem + 9rem + 2 * var(--space-3)); font-size: var(--text-sm); }
 dt { font: 600 var(--text-xs) var(--font-cond); color: var(--text-muted); } dd { margin: 0; }
 dd.t { font-weight: 400; }
 @media (max-width: 800px) {
  summary { grid-template-columns: auto minmax(0, 1fr) auto; }
  summary .t { display: none; }
  dl { padding-left: 0; }
 }
</style>
