<script>
 // Every test of a run as a ledger row, failures first. The chosen test opens
 // beneath its row with the assertions and steps that broke; choosing another
 // test moves the strip and the logs above to its window.
 import StatusBadge from './StatusBadge.svelte'
 import {assertDiff} from './run-timeline.mjs'
 import {formatTime} from './labels.mjs'
 let {rows=[],chosen=null,onchoose=()=>{},onnode=()=>{},nodes=[]}=$props()
 const FAILED=['failed','blocked']
 const duration=ms=>ms===null?'—':ms<1000?`${ms}ms`:ms<60000?`${(ms/1000).toFixed(1)}초`:`${Math.floor(ms/60000)}분 ${Math.round((ms%60000)/1000)}초`
 const shown=value=>typeof value==='string'?value:JSON.stringify(value)
 const summary=row=>[row.failedAsserts.length?`단언 ${row.failedAsserts.length}건 실패`:'',row.stepErrors.length?`스텝 오류 ${row.stepErrors.length}건`:'',!row.failedAsserts.length&&!row.stepErrors.length?row.reason:''].filter(Boolean).join(' · ')
</script>
<section class="ledger" aria-label="테스트 기록">
 <h3>테스트 기록 <small>{rows.length}건 · 실패와 막힘 먼저</small></h3>
 <div class="table-wrap"><table class="tests">
  <thead><tr><th>시작</th><th>테스트</th><th>판정</th><th>걸린 시간</th><th>근거</th></tr></thead>
  <tbody>
   {#each rows as row (row.seq)}
    <tr class:chosen={chosen===row.id} class:failed={FAILED.includes(row.status)}>
     <td class="num">{row.startedAt?formatTime(row.startedAt).slice(11):'—'}</td>
     <td><button type="button" class="link name" aria-pressed={chosen===row.id} onclick={()=>onchoose(row)}>{row.id}</button></td>
     <td><StatusBadge state={row.status} /></td>
     <td class="num">{duration(row.durationMs)}</td>
     <td class="why">{summary(row)}</td>
    </tr>
    {#if chosen===row.id&&(row.failedAsserts.length||row.stepErrors.length||(row.reason&&FAILED.includes(row.status)))}
     <tr class="open"><td colspan="5"><div class="cause" data-testid="run-cause" data-test={row.id}>
      {#if row.failedAsserts.length}
       <table class="asserts"><thead><tr><th>실패한 단언</th><th>노드</th><th>기대</th><th>실제</th></tr></thead><tbody>
        {#each row.failedAsserts as a}{@const diff=assertDiff(a.expected,a.actual)}<tr>
         <td data-label="단언"><code>{a.assert}</code></td>
         <td data-label="노드">{#if a.on&&nodes.includes(a.on)}<button type="button" class="link" onclick={()=>onnode(a.on)}>{a.on}</button>{:else}{a.on||'—'}{/if}</td>
         <td data-label="기대"><code class="struck">{shown(a.expected)}</code></td>
         <td data-label="실제"><code class="actual">{shown(a.actual)}</code>{#if diff}<small class="diff">차이 {diff}</small>{/if}</td>
        </tr>{/each}
       </tbody></table>
      {/if}
      {#if row.stepErrors.length}<ul class="steps">{#each row.stepErrors as s}<li><code>#{s.index} {s.type}</code>{#if s.on} · {#if nodes.includes(s.on)}<button type="button" class="link" onclick={()=>onnode(s.on)}>{s.on}</button>{:else}{s.on}{/if}{/if} · <span class="error-text">{s.error}</span></li>{/each}</ul>{/if}
      {#if !row.failedAsserts.length&&!row.stepErrors.length}<p class="error-text">{row.reason}</p>{/if}
     </div></td></tr>
    {/if}
   {/each}
  </tbody>
 </table></div>
</section>
<style>
 .ledger { padding-top: var(--space-4); }
 h3 small { margin-left: var(--space-2); font: 400 var(--text-xs) var(--font-sans); color: var(--text-muted); }
 .num { font: var(--text-sm) var(--font-mono); font-variant-numeric: tabular-nums; white-space: nowrap; }
 .name { font: 600 var(--text-sm) var(--font-mono); text-decoration: none; color: var(--text); text-align: left; white-space: normal; }
 .name:hover { text-decoration: underline; }
 @media (pointer: coarse) { .name, td .link, .steps .link { min-height: 44px; } }
 tr.failed .name { color: var(--danger); }
 tr.chosen > td { background: var(--surface-raised); }
 tr.chosen .name { border-bottom: 2px solid currentColor; }
 .why { color: var(--text-muted); font-size: var(--text-sm); }
 tr.failed .why { color: var(--danger); }
 tr.open > td { padding: 0 0 var(--space-3) var(--space-5); background: var(--surface-raised); border-left: 0; }
 .asserts { margin-top: var(--space-1); }
 .struck { text-decoration: line-through; color: var(--text-muted); }
 .actual { color: var(--danger); font-weight: 700; }
 .diff { display: block; color: var(--danger); font: var(--text-xs) var(--font-mono); }
 .steps { margin: var(--space-2) 0 0; padding-left: var(--space-4); font-size: var(--text-sm); }
 .error-text { color: var(--danger); }
 @media (max-width: 700px) {
  .tests > thead th:nth-child(1), .tests > tbody > tr:not(.open) > td:nth-child(1), .tests > thead th:nth-child(5), .tests > tbody > tr:not(.open) > td:nth-child(5) { display: none; }
  tr.open > td { padding-left: 0; }
  .asserts thead { display: none; }
  .asserts, .asserts tbody, .asserts tr, .asserts td { display: block; }
  .asserts tr { padding: var(--space-2) 0; border-bottom: var(--rule); }
  .asserts td { display: grid; grid-template-columns: 3rem minmax(0, 1fr); gap: var(--space-2); padding: 2px 0; border: 0; }
  .asserts td[data-label]::before { content: attr(data-label); color: var(--text-muted); font-size: var(--text-xs); }
  .asserts td code { overflow-x: auto; overflow-wrap: normal; white-space: nowrap; }
  .asserts td .diff { grid-column: 2; }
 }
</style>
