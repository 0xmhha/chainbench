<script>
 import {onMount} from 'svelte'
 import {chartModel,logWindow,gapSummary,metricNames,timeRange} from './metric-chart.mjs'
 import {monitorRange} from './selection.svelte.js'
 import LogViewer from './LogViewer.svelte'
 let {networkId}=$props()
 // Every metric the server collected in the range is charted unless one is chosen.
 const charts=$derived(metricNames(metrics).filter(m=>!monitorRange.metric||m.name===monitorRange.metric))
 const shownNodes=$derived(allNodes.filter(n=>!monitorRange.node||n===monitorRange.node))
 // Series colors are design tokens (--series-1 … --series-8 in App).
 const colors=[1,2,3,4,5,6,7,8].map(i=>`var(--series-${i})`)
 const W=640,H=160
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
    <figure>
     <figcaption>{label}{#if model.nodes.length} · {model.yMin}–{model.yMax} {model.unit}{/if}</figcaption>
     {#if model.nodes.length===0}<p class="muted">이 범위에 수집된 값이 없습니다.</p>{:else}
      <svg data-testid={'metric-chart-'+m.name} data-segments={model.nodes.reduce((n,x)=>n+x.segments.length,0)} viewBox={`-6 -6 ${W+12} ${H+12}`} role="img" aria-label={label}>
       {#each model.nodes as series (series.node)}
        <path d={series.path} data-node={series.node} fill="none" style:stroke={nodeColor(series.node,allNodes)} stroke-width="2"/>
        {#each series.points as point (point.time)}
         <circle cx={point.x} cy={point.y} r="4" style:fill={nodeColor(series.node,allNodes)} data-node={series.node} data-time={point.time} role="button" tabindex="0" aria-label={`${series.node} ${time(point.time)} ${point.value} ${model.unit} 시점 로그 보기`} onclick={()=>select(series.node,point,model)} onkeydown={event=>{if(event.key==='Enter'||event.key===' ')select(series.node,point,model)}}><title>{series.node} · {time(point.time)} · {point.value} {model.unit} · {model.source}</title></circle>
        {/each}
       {/each}
      </svg>
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
</style>
