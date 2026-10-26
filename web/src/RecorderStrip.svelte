<script>
 // The recorder strip: the chosen test, the node's warnings and errors, its
 // gauges and the other nodes' errors on one time axis, read between two
 // cursors. A click moves B, a Shift+click moves A; with the strip focused the
 // arrow keys move B by a second and Shift+arrows move A.
 import {formatTime,gapText} from './labels.mjs'
 import {failureMetrics} from './metric-chart.mjs'
 import {stripX,stripTime,tracePath,measureBetween} from './recorder.mjs'

 let {from,to,node,metrics=null,events=[],otherEvents=[],tests=[],chosen=null,onchoose=()=>{},a=$bindable(),b=$bindable(),onreset=()=>{}}=$props()

 const ERRORS=['ERROR','CRIT']
 const step=$derived(metrics?.coverage?.bucketSeconds||metrics?.coverage?.sampleIntervalSeconds||5)
 const gauges=$derived(metrics?failureMetrics(metrics,node):[])
 const otherBlock=$derived.by(()=>{
  // The highest block any other node reached at each sample time, drawn faint.
  const name=gauges[0]?.name
  const byTime=new Map()
  for(const s of metrics?.series??[])if(s.name===name&&s.nodeId!==node)for(const p of s.samples)if(typeof p.value==='number')byTime.set(p.time,Math.max(byTime.get(p.time)??-Infinity,p.value))
  return [...byTime].map(([time,value])=>({time,value})).sort((x,y)=>Date.parse(x.time)-Date.parse(y.time))
 })
 const gaps=$derived((metrics?.coverage?.gaps??[]).filter(g=>g.scope!=='node'||g.nodeId===node).map(g=>({x:stripX(g.from<from?from:g.from,from,to),x2:stripX(g.to>to?to:g.to,from,to),gap:g})).filter(g=>g.x!==null&&g.x2!==null&&g.x2>g.x))
 const warns=$derived(events.filter(e=>!ERRORS.includes(e.level)))
 const errors=$derived(events.filter(e=>ERRORS.includes(e.level)))
 const measure=$derived(metrics&&a&&b?measureBetween(metrics,node,a,b):null)
 const ax=$derived(stripX(a,from,to)),bx=$derived(stripX(b,from,to))
 const clock=v=>formatTime(v).slice(11)

 let plot=$state()
 // Which cursor a click or the arrows move; Shift picks the other one.
 let moving=$state('b')
 const target=event=>(event.shiftKey?(moving==='a'?'b':'a'):moving)
 function place(event){
  // Positions count from the trace area, after the channel labels.
  const box=plot.getBoundingClientRect(),lab=parseFloat(getComputedStyle(plot).getPropertyValue('--lab'))||0
  const x=event.clientX-box.left-lab
  if(x<0)return
  const time=stripTime((x/(box.width-lab))*100,from,to)
  if(target(event)==='a')a=time;else b=time
 }
 function key(event){
  if(event.key!=='ArrowLeft'&&event.key!=='ArrowRight')return
  event.preventDefault()
  const by=event.key==='ArrowRight'?1000:-1000
  const move=v=>{const t=Math.min(Date.parse(to),Math.max(Date.parse(from),Date.parse(v)+by));return new Date(t).toISOString()}
  if(target(event)==='a')a=move(a);else b=move(b)
 }
 const offset=v=>Math.round((Date.parse(v)-Date.parse(from))/1000)
 const bIsError=$derived(errors.some(e=>Math.abs(Date.parse(e.time)-Date.parse(b))<500))
 const chosenRow=$derived(tests.find(r=>r.id===chosen))
 const ticks=$derived(Array.from({length:7},(_,i)=>stripTime((i/6)*100,from,to)))
 const bar=row=>{const x=stripX(row.startedAt,from,to)??(Date.parse(row.startedAt)<Date.parse(from)?0:null),x2=stripX(row.finishedAt,from,to)??(Date.parse(row.finishedAt)>Date.parse(to)?100:null);return x===null||x2===null?null:{x,w:Math.max(.6,x2-x)}}
 const fmt=v=>v===null||v===undefined?'—':v
</script>

<section class="strip" data-testid="recorder-strip" data-node={node} aria-label="{node} 기록계 띠">
 <div class="chart">
  <div class="axis" aria-hidden="true"><span class="lab">{clock(from)} – {clock(to)}</span><span class="ticks">{#each ticks as t}<span>{clock(t)}</span>{/each}</span></div>
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div class="plot" bind:this={plot} role="slider" tabindex="0" aria-label="커서 {moving==='a'?'A':'B'} 시각. 화살표로 1초씩 옮깁니다. Shift를 누르면 다른 커서를 옮깁니다." aria-valuemin={0} aria-valuemax={offset(to)} aria-valuenow={offset(moving==='a'?a:b)} aria-valuetext="커서 A {clock(a)}, 커서 B {clock(b)}" onclick={place} onkeydown={key}>
   {#if tests.length}
    <div class="chan"><span class="lab">테스트<small>{tests.length}건</small></span><svg viewBox="0 0 100 20" preserveAspectRatio="none">
     {#each tests as row (row.seq)}{@const p=bar(row)}{#if p}<rect class="t {row.status}" class:chosen={chosen===row.id} x={p.x} y="4" width={p.w} height="12"><title>{row.id} · {row.status}</title></rect>{/if}{/each}
    </svg>{#if chosenRow&&bar(chosenRow)}{@const p=bar(chosenRow)}<span class="barname" class:failed={['failed','blocked'].includes(chosenRow.status)} style:left="calc(var(--lab) + (100% - var(--lab)) * {p.x / 100} + 4px)">{chosenRow.id}</span>{/if}</div>
   {/if}
   <div class="chan" class:bad={errors.length}><span class="lab">{node} 로그<small>경고 {warns.length} · 오류 {errors.length}</small></span><svg viewBox="0 0 100 20" preserveAspectRatio="none">
    <line class="base" x1="0" y1="17" x2="100" y2="17"/>
    {#each warns as e}{@const x=stripX(e.time,from,to)}{#if x!==null}<line class="warn" x1={x} y1="6" x2={x} y2="17"/>{/if}{/each}
    {#each errors as e}{@const x=stripX(e.time,from,to)}{#if x!==null}<line class="err" x1={x} y1="1" x2={x} y2="17"/>{/if}{/each}
   </svg></div>
   {#each gauges as g, i (g.label)}
    {@const trace=tracePath(g.series[0]?.samples,from,to,step)}
    {@const m=measure?.gauges[i]}
    <div class="chan" class:bad={m?.abnormal}><span class="lab">{g.label}<small>{trace.segments.length?`${trace.min}–${trace.max}`:'수집 안 됨'}</small></span><svg viewBox="0 -12 100 124" preserveAspectRatio="none">
     {#each gaps as gap}<rect class="gap" x={gap.x} y="-12" width={gap.x2-gap.x} height="124"><title>누락 · {gapText(gap.gap).text}</title></rect>{/each}
     {#if i===0}{#each tracePath(otherBlock,from,to,step).segments as d}<path class="other" d={d}/>{/each}{/if}
     {#each trace.segments as d}<path class="trace" d={d}/>{/each}
    </svg></div>
   {/each}
   {#if otherEvents.length||tests.length}
    <div class="chan"><span class="lab">다른 노드<small>오류 {otherEvents.length}</small></span><svg viewBox="0 0 100 20" preserveAspectRatio="none">
     <line class="base" x1="0" y1="17" x2="100" y2="17"/>
     {#each otherEvents as e}{@const x=stripX(e.time,from,to)}{#if x!==null}<line class="err" x1={x} y1="4" x2={x} y2="17"><title>{e.node} · {clock(e.time)}</title></line>{/if}{/each}
    </svg></div>
   {/if}
   {#if ax!==null}<span class="cur a" style:left="calc(var(--lab) + (100% - var(--lab)) * {ax / 100})"><b>A {clock(a)}</b></span>{/if}
   {#if bx!==null}<span class="cur b" class:flip={bx > 70} style:left="calc(var(--lab) + (100% - var(--lab)) * {bx / 100})"><b>B {clock(b)}{bIsError?' 오류':''}</b></span>{/if}
  </div>
  {#if tests.length}<p class="pick">{#each tests as row (row.seq)}<button type="button" class="link" class:chosen={chosen===row.id} onclick={()=>onchoose(row)}>{row.id}</button>{/each}</p>{/if}
 </div>
 <div class="measure" data-testid="recorder-measure">
  <h3>측정 · A에서 B까지</h3>
  <div class="mode" role="group" aria-label="옮길 커서"><button type="button" aria-pressed={moving==='a'} onclick={()=>{moving='a'}}>A 옮기기</button><button type="button" aria-pressed={moving==='b'} onclick={()=>{moving='b'}}>B 옮기기</button><button type="button" onclick={onreset}>기본 위치로</button></div>
  {#if measure}
   <dl>
    <dt>간격</dt><dd>{measure.seconds}초</dd>
    {#each measure.gauges as g}<dt>{g.label}</dt><dd class:bad={g.abnormal}>{fmt(g.a)} → {fmt(g.b)}</dd>{/each}
    <dt>다른 노드 블록</dt><dd>{fmt(measure.others.a)} → {fmt(measure.others.b)}</dd>
   </dl>
  {:else if !metrics}<p class="muted">지표를 불러오는 중이거나 이 구간에 지표가 없습니다.</p>{/if}
  <p class="keys">띠를 누르면 고른 커서가 옮겨집니다. Shift를 누른 채 누르면 다른 커서를 옮깁니다. 빨간 값은 다른 모든 노드보다 뒤처지거나 적거나 많은 값입니다.</p>
 </div>
</section>

<style>
 .strip { display: grid; grid-template-columns: minmax(0, 1fr) 260px; border-bottom: var(--rule-strong); }
 .chart { min-width: 0; padding-right: var(--space-4); border-right: 1px solid var(--border); }
 .chart { --lab: 112px; }
 .axis, .chan { display: grid; grid-template-columns: var(--lab) minmax(0, 1fr); }
 .axis { border-bottom: 1px solid var(--border); font: var(--text-xs) var(--font-mono); color: var(--text-muted); min-height: 24px; }
 .axis .lab { padding: 6px 0; font: 600 var(--text-xs) var(--font-cond); }
 .ticks { display: flex; justify-content: space-between; gap: var(--space-2); min-width: 0; padding: 6px 0; font-variant-numeric: tabular-nums; }
 .plot { position: relative; cursor: crosshair; }
 .plot:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
 .chan { height: 40px; border-bottom: var(--rule); }
 .chan .lab { padding: 4px var(--space-2) 0 0; border-right: var(--rule); font: 600 var(--text-sm)/1.15 var(--font-sans); color: var(--text); }
 .chan .lab small { display: block; margin-top: 2px; font: var(--text-xs) var(--font-mono); color: var(--text-muted); font-weight: 400; }
 .chan .lab { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
 .chan .lab small { overflow: hidden; text-overflow: ellipsis; }
 .chan.bad .lab { color: var(--danger); }
 .chan svg { width: 100%; height: 40px; display: block; overflow: hidden; }
 .base { stroke: var(--border-subtle); stroke-width: 1; vector-effect: non-scaling-stroke; }
 .warn { stroke: var(--warn); stroke-width: 2.5; vector-effect: non-scaling-stroke; }
 .err { stroke: var(--danger); stroke-width: 3; vector-effect: non-scaling-stroke; }
 .trace { fill: none; stroke: var(--accent); stroke-width: 1.8; vector-effect: non-scaling-stroke; }
 .other { fill: none; stroke: var(--text-faint); stroke-width: 1; stroke-dasharray: 2 2; vector-effect: non-scaling-stroke; }
 .gap { fill: var(--gap-fill); }
 .t { fill: none; stroke: var(--text); stroke-width: 1; vector-effect: non-scaling-stroke; }
 .t.failed { fill: var(--danger); stroke: var(--danger); } .t.blocked { stroke: var(--danger); stroke-dasharray: 3 2; }
 .t.skipped { stroke: var(--text-faint); stroke-dasharray: 1 2; }
 .t.chosen { stroke-width: 2.5; stroke: var(--text); }
 .cur { position: absolute; top: 0; bottom: 0; width: 0; border-left: 1.5px dashed var(--text-muted); pointer-events: none; }
 .cur b { position: absolute; bottom: 2px; left: 4px; padding: 0 4px; border: 1px solid currentColor; background: var(--bg); font: 700 var(--text-xs) var(--font-mono); white-space: nowrap; }
 .cur.flip b { left: auto; right: 4px; }
 .cur b { top: -22px; bottom: auto; }
 .cur.a b { top: -22px; }
 .cur.a { color: var(--text-muted); } .cur.b { color: var(--danger); border-left: 1.5px solid var(--danger); }
 .pick { display: flex; flex-wrap: wrap; gap: var(--space-1) var(--space-3); margin: var(--space-2) 0; font-size: var(--text-sm); }
 .pick .chosen { color: var(--text); font-weight: 700; text-decoration: none; border-bottom: 2px solid var(--text); }
 .chan { position: relative; }
 .barname { position: absolute; top: 12px; font: 600 var(--text-xs) var(--font-mono); color: var(--text); pointer-events: none; white-space: nowrap; }
 .barname.failed { color: var(--on-accent); }
 .mode { display: flex; flex-wrap: wrap; gap: var(--space-1); margin-top: var(--space-2); }
 .mode button { min-height: 30px; padding: 2px 8px; font-size: var(--text-xs); }
 .mode button[aria-pressed=true] { border-color: var(--text); border-bottom-width: 3px; }
 @media (pointer: coarse) { .mode button, .pick .link { min-height: 44px; } }
 .measure { padding: var(--space-3) 0 var(--space-3) var(--space-4); }
 .measure dl { display: grid; grid-template-columns: auto 1fr; gap: 3px var(--space-3); margin: var(--space-2) 0 0; font: var(--text-sm) var(--font-mono); font-variant-numeric: tabular-nums; }
 .measure dt { font: 600 var(--text-xs) var(--font-cond); color: var(--text-muted); padding-top: 2px; } .measure dd { margin: 0; }
 .measure .bad { color: var(--danger); font-weight: 700; }
 .keys { margin: var(--space-3) 0 0; color: var(--text-faint); font-size: var(--text-xs); }
 @media (max-width: 900px) {
  .strip { grid-template-columns: minmax(0, 1fr); }
  .chart { padding-right: 0; border-right: 0; }
  .measure { padding-left: 0; border-top: var(--rule); }
  .chart { --lab: 84px; }
  .ticks span:nth-child(even) { display: none; }
  .ticks { min-width: 0; overflow: hidden; }
  .cur b { max-width: 40vw; overflow: hidden; }
 }
</style>
