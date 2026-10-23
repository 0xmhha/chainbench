<script>
 // A run read failure-first: the verdict split, the tests on the run's time
 // axis, and for the chosen test the assertions that broke beside the node
 // logs of the moments it ran.
 import StatusBadge from './StatusBadge.svelte'
 import LogViewer from './LogViewer.svelte'
 import {testRows,verdictSegments,placeOnAxis,testLogSpan} from './run-timeline.mjs'
 import {stateLabel,formatTime} from './labels.mjs'

 let {run,nodes=[]}=$props()

 const PAD_SECONDS=5
 const FAILED=['fail','blocked']

 const observations=$derived(run?.summary?.observations??null)
 const rows=$derived(testRows(run?.summary))
 const axisFrom=$derived(observations?.from??run?.startedAt??rows.find(r=>r.startedAt)?.startedAt??null)
 const axisTo=$derived(observations?.to??run?.finishedAt??[...rows].reverse().find(r=>r.finishedAt)?.finishedAt??null)
 const placed=$derived(placeOnAxis(rows,axisFrom,axisTo))
 const timed=$derived(placed.filter(r=>r.left!==null))
 const segments=$derived(verdictSegments(run?.summary?.counts))
 const failures=$derived(rows.filter(r=>FAILED.includes(r.status)))

 let chosen=$state(null)
 // A node picked from an assertion or step row overrides the default log node.
 let pickedNode=$state(null)
 const selected=$derived(rows.find(r=>r.seq===chosen)??failures[0]??null)
 const span=$derived(selected?testLogSpan(selected,PAD_SECONDS):null)
 const logNodes=$derived(nodes.length?nodes:selected?.nodes??[])
 // The node an assertion failed on is the likeliest cause, then a step error's node.
 const evidenceNodes=$derived([...(selected?.failedAsserts??[]).map(a=>a.on),...(selected?.stepErrors??[]).map(s=>s.on)].filter(Boolean))
 const firstNode=$derived(pickedNode&&logNodes.includes(pickedNode)?pickedNode:evidenceNodes.find(n=>logNodes.includes(n))??logNodes[0]??'')
 const failedTest=$derived(FAILED.includes(selected?.status))
 function choose(seq){chosen=seq;pickedNode=null}
 function showNode(node){pickedNode=node}

 const time=formatTime
 const duration=ms=>ms===null?'시각 없음':ms<1000?`${ms}ms`:ms<60000?`${(ms/1000).toFixed(1)}초`:`${Math.floor(ms/60000)}분 ${Math.round((ms%60000)/1000)}초`
 const verdictName=Object.fromEntries(['fail','blocked','pass','skip'].map(s=>[s,stateLabel(s)]))
 const shown=value=>typeof value==='string'?value:JSON.stringify(value)
</script>

<section class="run-timeline" data-testid="run-timeline" aria-label="실행 타임라인">
 {#if segments.length}
  <div class="verdicts" role="img" aria-label={segments.map(s=>`${verdictName[s.status]} ${s.count}`).join(', ')}>
   {#each segments as s}<span class="seg {s.status}" style:width="{s.percent}%" title="{verdictName[s.status]} {s.count}"></span>{/each}
  </div>
  <p class="legend">{#each segments as s}<span class="key {s.status}"><i></i>{verdictName[s.status]} <b>{s.count}</b></span>{/each}{#if axisFrom&&axisTo}<span class="range">{time(axisFrom)} – {time(axisTo)}</span>{/if}</p>
 {:else}
  <p class="muted">이 실행에는 테스트 판정이 없습니다.</p>
 {/if}

 {#if timed.length}
  <div class="axis" aria-label="테스트 시간축">
   {#each timed as row (row.seq)}
    <button type="button" class="bar {row.status}" class:chosen={selected?.seq===row.seq} style:left="{row.left}%" style:width="{row.width}%" title="{row.id} · {verdictName[row.status]??row.status} · {duration(row.durationMs)}" aria-label="{row.id} {verdictName[row.status]??row.status}" onclick={()=>choose(row.seq)}></button>
   {/each}
  </div>
 {/if}

 {#if rows.length}
  <div class="split">
   <ol class="tests" aria-label="테스트">
    {#each rows as row (row.seq)}
     <li><button type="button" class="test" class:chosen={selected?.seq===row.seq} aria-pressed={selected?.seq===row.seq} onclick={()=>choose(row.seq)}>
      <StatusBadge state={row.status} />
      <span class="name">{row.id}</span>
      <span class="meta">{[duration(row.durationMs),row.failedAsserts.length?`단언 ${row.failedAsserts.length}건 실패`:'',row.stepErrors.length?`스텝 오류 ${row.stepErrors.length}건`:''].filter(Boolean).join(' · ')}</span>
     </button></li>
    {/each}
   </ol>

   {#if selected}
    <div class="cause" data-testid="run-cause" data-test={selected.id}>
     <h3><StatusBadge state={selected.status} /> {selected.id}</h3>
     <p class="muted">{time(selected.startedAt)} – {time(selected.finishedAt)} · {duration(selected.durationMs)}</p>
     {#if selected.reason}<p class={FAILED.includes(selected.status)?'error':'muted'}>{selected.reason}</p>{/if}
     {#if selected.failedAsserts.length}
      <div class="table-wrap"><table>
       <thead><tr><th>실패한 단언</th><th>노드</th><th>기대</th><th>실제</th><th><span class="sr">로그</span></th></tr></thead>
       <tbody>{#each selected.failedAsserts as a}<tr><td><code>{a.assert ?? a.id}</code></td><td>{a.on||'—'}</td><td><code>{shown(a.expected)}</code></td><td><code class="actual">{shown(a.actual)}</code></td><td>{#if a.on&&logNodes.includes(a.on)}<button type="button" class="link" onclick={()=>showNode(a.on)}>{a.on} 로그</button>{/if}</td></tr>{/each}</tbody>
      </table></div>
     {/if}
     {#if selected.stepErrors.length}
      <ul class="steps">{#each selected.stepErrors as s}<li><code>#{s.index} {s.type}</code>{s.on?` · ${s.on}`:''}{' · '}<span class="error-text">{s.error}</span>{#if s.on&&logNodes.includes(s.on)}{' '}<button type="button" class="link" onclick={()=>showNode(s.on)}>{s.on} 로그</button>{/if}</li>{/each}</ul>
     {/if}
     {#if observations&&logNodes.length}
      {#if !span.exact}<p class="notice">이 기록에는 테스트별 시각이 없어 실행 전체 범위의 로그를 엽니다.</p>{/if}
      {#key `${selected.seq}-${firstNode}`}
       <LogViewer networkId={observations.networkId} nodes={logNodes} node={firstNode} from={span.exact?span.from:null} to={span.exact?span.to:null} runId={span.exact?null:run.id} initialLevels={failedTest||!span.exact?['ERROR','WARN']:[]} focus={selected.finishedAt} title={span.exact?`${firstNode} · 이 테스트 구간(앞뒤 ${PAD_SECONDS}초)${failedTest?' · ERROR·WARN 먼저':''}`:`${firstNode} · 실행 전체 범위 · ERROR·WARN 먼저`} testid="cause-logs" />
      {/key}
     {:else if !observations}
      <p class="muted">이 실행에는 보관된 노드 로그·지표 구간이 없습니다.</p>
     {/if}
    </div>
   {/if}
  </div>
 {/if}
</section>

<style>
 .run-timeline { margin-top: var(--space-4); min-width: 0; }
 .verdicts { display: flex; gap: 2px; height: 10px; border-radius: 999px; overflow: hidden; background: var(--border-subtle); }
 .seg { flex: none; min-width: 4px; }
 .seg.fail, .key.fail i { background: var(--danger); } .seg.blocked, .key.blocked i { background: var(--warn); }
 .seg.pass, .key.pass i { background: var(--ok); } .seg.skip, .key.skip i { background: var(--border-strong); }
 .legend { display: flex; flex-wrap: wrap; gap: var(--space-2) var(--space-4); margin: var(--space-2) 0 0; color: var(--text-muted); font-size: var(--text-sm); }
 .key { display: inline-flex; align-items: center; gap: 6px; } .key i { width: 8px; height: 8px; border-radius: 2px; } .key b { color: var(--text); font-variant-numeric: tabular-nums; }
 .range { margin-left: auto; font-variant-numeric: tabular-nums; }
 .axis { position: relative; height: 28px; margin-top: var(--space-3); border-radius: var(--radius-sm); background: var(--bg); border: 1px solid var(--border-subtle); }
 .bar { position: absolute; top: 5px; height: 16px; min-height: 0; min-width: 4px; padding: 0; border: 0; border-radius: 3px; opacity: .75; }
 .bar.fail { background: var(--danger); } .bar.blocked { background: var(--warn); } .bar.pass { background: var(--ok); } .bar.skip { background: var(--border-strong); }
 .bar:hover, .bar.chosen { opacity: 1; box-shadow: 0 0 0 2px var(--text); }
 .split { display: grid; grid-template-columns: minmax(220px, 300px) minmax(0, 1fr); gap: var(--space-4); margin-top: var(--space-4); align-items: start; }
 .tests { list-style: none; margin: 0; padding: 0; display: grid; gap: 2px; max-height: 640px; overflow: auto; }
 .test { display: grid; grid-template-columns: auto minmax(0, 1fr); gap: 2px var(--space-2); width: 100%; min-height: 0; padding: var(--space-2); text-align: left; white-space: normal; background: none; border-color: transparent; }
 .test.chosen { background: var(--surface-raised); border-color: var(--accent-border); }
 .name { overflow-wrap: anywhere; font-family: var(--font-mono); font-size: var(--text-sm); }
 .meta { grid-column: 2; color: var(--text-muted); font-size: var(--text-xs); font-variant-numeric: tabular-nums; }
 .cause { min-width: 0; }
 .cause h3 { display: flex; gap: var(--space-2); align-items: baseline; }
 .cause > p { margin: var(--space-1) 0; }
 .actual { color: var(--danger); }
 .steps { margin: var(--space-2) 0; padding-left: var(--space-4); }
 .error-text { color: var(--danger); }
 .sr { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
 @media (max-width: 900px) { .split { grid-template-columns: minmax(0, 1fr); } .tests { max-height: 280px; } }
</style>
