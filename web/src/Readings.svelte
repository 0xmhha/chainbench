<script>
 // The failed node's gauges as the watch recorded them, one row per sample. A
 // reading that lags, lost peers or queued more than every other node at that
 // moment is ringed in red ink; a gap shows a dash, never the last value.
 import {formatTime} from './labels.mjs'
 import {failureMetrics} from './metric-chart.mjs'
 import {readingRows} from './recorder.mjs'
 let {metrics,node}=$props()
 const gauges=$derived(failureMetrics(metrics,node))
 const rows=$derived(readingRows(metrics,node))
 const every=$derived(metrics?.coverage?.bucketSeconds||metrics?.coverage?.sampleIntervalSeconds||5)
</script>
<section class="readings" data-testid="readings" aria-label="{node} 계기 기록">
 <h3>{node} 계기 기록 <small>{Math.round(every)}초마다</small></h3>
 {#if rows.length}
  <div class="table-wrap"><table>
   <thead><tr><th>시각</th>{#each gauges as g}<th title={g.label}>{g.short}</th>{/each}</tr></thead>
   <tbody>{#each rows as row (row.time)}<tr><td>{formatTime(row.time).slice(11)}</td>{#each row.cells as c}<td class:ab={c.abnormal} class:gap={c.value===null} title={c.abnormal?'다른 모든 노드와 다른 값':''}>{c.value??'—'}</td>{/each}</tr>{/each}</tbody>
  </table></div>
  {#if rows.some(r=>r.cells.some(c=>c.abnormal))}<p class="legend"><span class="ring">46</span> 다른 모든 노드보다 뒤처지거나(블록), 적거나(피어), 많은(대기 tx) 값</p>{/if}
 {:else}<p class="muted">이 구간에 {node}의 계기 기록이 없습니다.</p>{/if}
</section>
<style>
 .readings { min-width: 0; }
 h3 small { margin-left: var(--space-2); font: 400 var(--text-xs) var(--font-sans); color: var(--text-muted); }
 .table-wrap { margin-top: var(--space-2); }
 td { font: var(--text-sm) var(--font-mono); text-align: right; padding: 4px var(--space-2); }
 td:first-child, th:first-child { text-align: left; color: var(--text-muted); }
 th { text-align: right; }
 .ab { color: var(--danger); font-weight: 700; outline: 1.5px solid var(--danger); outline-offset: -3px; border-radius: 40%; }
 .gap { color: var(--text-faint); }
 .legend { margin: var(--space-2) 0 0; font-size: var(--text-xs); color: var(--text-muted); }
 .ring { display: inline-block; padding: 0 6px; margin-right: var(--space-1); color: var(--danger); font: 700 var(--text-xs) var(--font-mono); outline: 1.5px solid var(--danger); border-radius: 40%; }
 .muted { font-size: var(--text-sm); }
</style>
