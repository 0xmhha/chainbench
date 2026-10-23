<script>
 import {onMount} from 'svelte'
 import {chartModel,logWindow,gapSummary,metricNames,timeRange,timeTicks,gapBands,latestValues,nearestPoints,pointerTime} from './metric-chart.mjs'
 import {monitorRange} from './selection.svelte.js'
 import LogViewer from './LogViewer.svelte'
 let {networkId}=$props()
 // Every metric the server collected in the range is charted unless one is chosen.
 const charts=$derived(metricNames(metrics).filter(m=>!monitorRange.metric||m.name===monitorRange.metric))
 const shownNodes=$derived(allNodes.filter(n=>!monitorRange.node||n===monitorRange.node))
 // Series colors are design tokens (--series-1 … --series-8 in App).
 const colors=[1,2,3,4,5,6,7,8].map(i=>`var(--series-${i})`)
 const W=640,H=160
 // Room around the plot for the value labels (left) and the time labels (below).
 const ML=44,MR=8,MT=6,MB=20,TICKS=4
 // One cursor time is shared by every chart, so a moment lines up across metrics.
 let cursor=$state(null)
 // A node with no sample within two collection intervals of the cursor reads as missing.
 const tolerance=$derived(2*((metrics?.coverage?.bucketSeconds||metrics?.coverage?.sampleIntervalSeconds||5)*1000))
 const clock=value=>{const d=new Date(value);return `${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}:${String(d.getSeconds()).padStart(2,'0')}`}
 function chartX(event){
  const svg=event.currentTarget,box=svg.getBoundingClientRect()
  return (event.clientX-box.left)/box.width*(W+ML+MR)-ML
 }
 // Arrow keys step the shared cursor through the chart's sample times; Enter
 // opens the logs at the cursor for the first node with a value there.
 function chartKey(event,model){
  const times=[...new Set(model.nodes.flatMap(n=>n.points.map(p=>p.time)))].sort((a,b)=>Date.parse(a)-Date.parse(b))
  if(!times.length)return
  const index=cursor?times.findIndex(t=>Date.parse(t)>=Date.parse(cursor)):-1
  if(event.key==='ArrowRight'){event.preventDefault();cursor=times[Math.min(times.length-1,index<0?0:index+1)]}
  else if(event.key==='ArrowLeft'){event.preventDefault();cursor=times[Math.max(0,index<0?times.length-1:index-1)]}
  else if((event.key==='Enter'||event.key===' ')&&cursor){event.preventDefault();const near=nearestPoints(model,cursor,tolerance).find(p=>p.value!==null);if(near)select(near.node,near,model)}
  else if(event.key==='Escape')cursor=null
 }
 let selection=0,polls=0,applied=0
 let metrics=$state(null),error=$state(''),selected=$state(null)
 const gaps=$derived(gapSummary(metrics?.coverage))
 const nodeColor=(node,all)=>colors[all.indexOf(node)%colors.length]
 const allNodes=$derived([...new Set((metrics?.series??[]).map(s=>s.nodeId))].sort())
 const time=value=>value?new Date(value).toLocaleString():'—'
 async function load(){
  const order=++polls // An older, slower reply must not replace a newer one.
  try{
   const range=timeRange(monitorRange)
   if(!range)throw new Error('시작 시각이 끝 시각보다 앞서야 합니다.')
   const response=await fetch(`/api/v1/networks/${encodeURIComponent(networkId)}/metrics?from=${encodeURIComponent(range.from)}&to=${encodeURIComponent(range.to)}`,{cache:'no-store'})
   if(!response.ok)throw new Error(`지표 조회 실패 (${response.status})`)
   const value=await response.json()
   if(order>applied){applied=order;metrics=value;error=''}
  }catch(failure){if(order>applied)error=failure.message}
 }
 // Choosing a point opens that node's logs around the point's time.
 function select(node,point,model){
  selected={node,time:point.time,value:point.value,unit:model?.unit??selected?.unit??'',source:model?.source??selected?.source??'',name:model?.name??selected?.name??''}
 }
 onMount(()=>{const timer=setInterval(()=>{if(monitorRange.preset>0)load()},5000);return()=>clearInterval(timer)})
 // A changed range reloads the charts and the range logs of the chosen node.
 $effect(()=>{void monitorRange.preset;void monitorRange.from;void monitorRange.to;load();selected=null})
 // The range logs follow the chosen node and the monitoring range.
 const rangeWindow=$derived(allNodes.includes(monitorRange.node)?timeRange({preset:monitorRange.preset,from:monitorRange.from,to:monitorRange.to}):null)
</script>
<div class="metrics">
 {#if error}<p class="error" role="alert">{error}</p>{/if}
 {#if metrics}
  <div data-testid="metric-coverage" class:incomplete={!metrics.coverage.complete}>
   <p>수집 범위 {time(metrics.coverage.from)} – {time(metrics.coverage.to)} · {metrics.coverage.sampleIntervalSeconds}초 간격{metrics.coverage.downsampled?` · ${Math.round(metrics.coverage.bucketSeconds)}초 구간 평균으로 축약됨`:''} · {metrics.coverage.complete?'누락 없음':'누락 구간 있음'}</p>
   {#if gaps.length}<ul>{#each gaps as gap}<li>{gap}</li>{/each}</ul>{/if}
   <p class="muted">수집하지 못했거나 노드가 노출하지 않는 지표는 0으로 그리지 않고 누락으로 표시합니다.</p>
  </div>
  <label>지표 <select name="metric" aria-label={'지표 선택 '+networkId} bind:value={monitorRange.metric}><option value="">수집된 모든 지표</option>{#each metricNames(metrics) as m}<option value={m.name}>{m.name} · {m.source}{m.unit?' · '+m.unit:''}</option>{/each}</select></label>
  <div class="legend">{#each shownNodes as node}<span><i style:background={nodeColor(node,allNodes)}></i>{node}</span>{/each}</div>
  <div class="grid">
   {#each charts as m (m.name)}
    {@const full=chartModel(metrics,m.name,{width:W,height:H})}
    {@const model={...full,nodes:full.nodes.filter(n=>shownNodes.includes(n.node))}}
    {@const label=`${m.name} · ${m.source}`}
    {@const latest=model.nodes.length?latestValues(model):null}
    <figure>
     <figcaption>
      <span class="name">{label}</span>
      {#if latest}<span class="latest">최근 {latest.values.map(v=>`${v.node} ${v.value}`).join(' · ')}{model.unit?` ${model.unit}`:''}{#if latest.spread!==null}<span class:lag={latest.spread>0}>{' · '}노드 간 차이 {latest.spread}</span>{/if}</span>{/if}
     </figcaption>
     {#if model.nodes.length===0}<p class="muted">이 범위에 수집된 값이 없습니다.</p>{:else}
      <svg data-testid={'metric-chart-'+m.name} data-segments={model.nodes.reduce((n,x)=>n+x.segments.length,0)} viewBox={`0 0 ${W+ML+MR} ${H+MT+MB}`} role="group" tabindex="0" aria-label={`${label} 차트. 좌우 화살표로 시점을 옮기고 Enter로 그 시점 로그를 엽니다.`}
       onpointermove={event=>{cursor=pointerTime(model,chartX(event))}} onpointerleave={()=>{cursor=null}} onkeydown={event=>chartKey(event,model)}>
       <g transform={`translate(${ML},${MT})`}>
        {#each gapBands(model,metrics.coverage) as band}<rect class="gap" x={band.x} y="0" width={band.width} height={H}><title>누락 · {band.reason}</title></rect>{/each}
        <line class="base" x1="0" y1={H} x2={W} y2={H}/>
        <text class="tick y" x="-6" y="4">{model.yMax}</text><text class="tick y" x="-6" y={H}>{model.yMin}</text>
        {#each timeTicks(model,TICKS) as tick, i}<text class="tick x" x={tick.x} y={H+15} text-anchor={i===0?'start':i===TICKS-1?'end':'middle'}>{clock(tick.time)}</text>{/each}
        {#each model.nodes as series (series.node)}
         <path d={series.path} data-node={series.node} fill="none" style:stroke={nodeColor(series.node,allNodes)} stroke-width="1.75"/>
         {#each series.points as point (point.time)}
          <circle cx={point.x} cy={point.y} r="3" style:fill={nodeColor(series.node,allNodes)} data-node={series.node} data-time={point.time} role="button" tabindex="-1" aria-label={`${series.node} ${time(point.time)} ${point.value} ${model.unit} 시점 로그 보기`} onclick={()=>select(series.node,point,model)}><title>{series.node} · {time(point.time)} · {point.value} {model.unit} · {model.source}</title></circle>
         {/each}
        {/each}
        {#if cursor}
         {@const near=nearestPoints(model,cursor,tolerance)}
         {#if near.some(p=>p.x!==null)}{@const x=near.find(p=>p.x!==null).x}<line class="cursor" x1={x} y1="0" x2={x} y2={H}/>{#each near.filter(p=>p.x!==null) as p}<circle class="cursor-dot" cx={p.x} cy={p.y} r="4.5" style:stroke={nodeColor(p.node,allNodes)}/>{/each}{/if}
        {/if}
       </g>
      </svg>
      {#if cursor}{@const near=nearestPoints(model,cursor,tolerance)}<p class="readout">{near.length?clock(near[0].time):''}{#each near as p}{' · '}<span style:color={nodeColor(p.node,allNodes)}>{p.node}</span> {p.value??'—'}{/each}</p>{/if}
     {/if}
    </figure>
   {/each}
  </div>
  {#if selected}
   <div class="linked">
    <p data-testid="selected-point">{selected.name} · {selected.node} · {time(selected.time)}{selected.value!==null?` · ${selected.value} ${selected.unit}`:''} · 출처 {selected.source}</p>
    {#key selected.time}<LogViewer {networkId} nodes={allNodes} node={selected.node} {...logWindow(selected.time,30)} focus={selected.time} title="{time(selected.time)} 전후 30초 로그" testid="time-linked-logs" data-selected-time={selected.time} />{/key}
   </div>
  {/if}
  {#if monitorRange.node&&rangeWindow}
   {#key monitorRange.node}<LogViewer {networkId} nodes={[monitorRange.node]} node={monitorRange.node} from={rangeWindow.from} to={rangeWindow.to} title="{monitorRange.node} 범위 로그" testid="range-logs" />{/key}
  {/if}
 {:else if !error}<p class="muted">지표를 불러오는 중입니다.</p>{/if}
</div>
<style>
 .metrics{margin-top:var(--space-3);min-width:0}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,320px),1fr));gap:var(--space-3)}
 figure{margin:0;padding:var(--space-3);border:1px solid var(--border-subtle);border-radius:var(--radius-sm);min-width:0}figcaption{color:var(--accent-strong);font-size:var(--text-sm);margin-bottom:var(--space-2)}svg{width:100%;height:auto;display:block}circle{cursor:pointer}circle:focus{outline:2px solid var(--warn)}
 .legend{display:flex;flex-wrap:wrap;gap:var(--space-3);margin:var(--space-2) 0}.legend span{display:flex;align-items:center;gap:6px;color:var(--text-muted);font-size:var(--text-sm)}.legend i{display:inline-block;width:10px;height:10px;border-radius:50%}
 [data-testid="metric-coverage"] p,ul{color:var(--text-muted);font-size:var(--text-sm);margin:var(--space-1) 0}.incomplete ul{color:var(--warn)}li{overflow-wrap:anywhere}
 .linked{margin-top:var(--space-3);padding-top:var(--space-3);border-top:1px solid var(--border-subtle)}.linked>p{margin:0;color:var(--text-muted);font-size:var(--text-sm)}
 .muted{font-size:var(--text-sm)}
 figcaption{display:flex;flex-wrap:wrap;justify-content:space-between;gap:var(--space-1) var(--space-3)}.name{color:var(--accent-strong)}.latest{color:var(--text-muted);font-size:var(--text-xs);font-variant-numeric:tabular-nums}.lag{color:var(--warn)}
 svg{cursor:crosshair;overflow:visible}svg:focus-visible{outline:2px solid var(--accent);outline-offset:2px}
 .gap{fill:var(--warn-bg);opacity:.8}.base{stroke:var(--border-subtle)}.tick{fill:var(--text-faint);font:11px var(--font-mono)}.tick.y{text-anchor:end}
 .cursor{stroke:var(--text-muted);stroke-dasharray:3 3}.cursor-dot{fill:var(--bg);stroke-width:2;pointer-events:none}circle:not(.cursor-dot){fill-opacity:0}circle:not(.cursor-dot):hover,circle:not(.cursor-dot):focus-visible{fill-opacity:1}
 .readout{margin:var(--space-1) 0 0;font:var(--text-xs) var(--font-mono);color:var(--text-muted)}
</style>
