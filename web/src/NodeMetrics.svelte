<script>
 import {onMount} from 'svelte'
 import {chartModel,logWindow,gapSummary,metricNames,timeRange} from './metric-chart.mjs'
 import {monitorRange} from './selection.svelte.js'
 let {networkId}=$props()
 // Every metric the server collected in the range is charted unless one is chosen.
 const charts=$derived(metricNames(metrics).filter(m=>!monitorRange.metric||m.name===monitorRange.metric))
 const shownNodes=$derived(allNodes.filter(n=>!monitorRange.node||n===monitorRange.node))
 let rangeLogs=$state(null),rangeLogError=$state('')
 // Series colors are design tokens (--series-1 … --series-8 in App).
 const colors=[1,2,3,4,5,6,7,8].map(i=>`var(--series-${i})`)
 const W=640,H=160
 let selection=0,polls=0,applied=0
 let metrics=$state(null),error=$state(''),selected=$state(null),logs=$state(null),logError=$state('')
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
 async function select(node,point,model){
  const choice=++selection // Plain counter: $state wraps objects in proxies.
  selected={node,time:point.time,value:point.value,unit:model?.unit??selected?.unit??'',source:model?.source??selected?.source??'',name:model?.name??selected?.name??''};logs=null;logError=''
  const {from,to}=logWindow(point.time,30)
  try{
   const response=await fetch(`/api/v1/nodes/${encodeURIComponent(networkId+'.'+node)}/logs?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&limit=100`,{cache:'no-store'})
   if(!response.ok)throw new Error(`로그 조회 실패 (${response.status})`)
   const value=await response.json()
   if(selection===choice)logs=value // A later selection supersedes this reply.
  }catch(failure){if(selection===choice)logError=failure.message}
 }
 onMount(()=>{const timer=setInterval(()=>{if(monitorRange.preset>0)load()},5000);return()=>clearInterval(timer)})
 // A changed range reloads the charts and the range logs of the chosen node.
 $effect(()=>{void monitorRange.preset;void monitorRange.from;void monitorRange.to;load();selected=null;logs=null})
 $effect(()=>{
  const node=monitorRange.node,range=timeRange({preset:monitorRange.preset,from:monitorRange.from,to:monitorRange.to})
  rangeLogs=null;rangeLogError=''
  if(!node||!range||!allNodes.includes(node))return
  fetch(`/api/v1/nodes/${encodeURIComponent(networkId+'.'+node)}/logs?from=${encodeURIComponent(range.from)}&to=${encodeURIComponent(range.to)}&limit=100`,{cache:'no-store'})
   .then(async r=>{if(!r.ok)throw new Error(`로그 조회 실패 (${r.status})`);const v=await r.json();if(monitorRange.node===node)rangeLogs=v})
   .catch(e=>{rangeLogError=e.message})
 })
</script>
<div class="metrics">
 {#if error}<p class="error" role="alert">{error}</p>{/if}
 {#if metrics}
  <div data-testid="metric-coverage" class:incomplete={!metrics.coverage.complete}>
   <p>수집 범위 {time(metrics.coverage.from)} – {time(metrics.coverage.to)} · {metrics.coverage.sampleIntervalSeconds}초 간격{metrics.coverage.downsampled?` · ${Math.round(metrics.coverage.bucketSeconds)}초 구간 평균으로 축약됨`:''} · {metrics.coverage.complete?'누락 없음':'누락 구간 있음'}</p>
   {#if gaps.length}<ul>{#each gaps as gap}<li>{gap}</li>{/each}</ul>{/if}
   <p class="muted">수집하지 못했거나 노드가 노출하지 않는 지표는 0으로 그리지 않고 누락으로 표시합니다.</p>
  </div>
  <label>지표 <select aria-label={'지표 선택 '+networkId} bind:value={monitorRange.metric}><option value="">수집된 모든 지표</option>{#each metricNames(metrics) as m}<option value={m.name}>{m.name} · {m.source}{m.unit?' · '+m.unit:''}</option>{/each}</select></label>
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
   <section class="logs" data-testid="time-linked-logs" data-selected-time={selected.time} data-log-node={logs?selected.node:undefined}>
    <p data-testid="selected-point">{selected.name} · {selected.node} · {time(selected.time)}{selected.value!==null?` · ${selected.value} ${selected.unit}`:''} · 출처 {selected.source}</p>
    <h4>{time(selected.time)} 전후 30초 로그</h4>
    <label>로그 노드 <select aria-label="로그 노드" value={selected.node} onchange={event=>select(event.currentTarget.value,{time:selected.time,value:null})}>{#each allNodes as node}<option value={node}>{node}</option>{/each}</select></label>
    {#if logError}<p class="error" role="alert">{logError}</p>{:else if !logs}<p class="muted">보관된 로그를 확인하는 중입니다.</p>{:else}
     <p class="muted">로그 시각 출처: {logs.coverage.timestampSource==='source'?'노드 로그 기록 시각':logs.coverage.timestampSource==='mixed'?'노드 기록 시각과 수집 시각 혼합':'서버 수집 시각'} · {logs.coverage.complete?'누락 없음':'누락 구간 있음'}</p>
     {#each gapSummary(logs.coverage) as gap}<p class="notice">{gap}</p>{/each}
     {#if gapSummary(logs.coverage).some(g=>g.includes('remote_log_collection_unavailable'))}<p class="muted">SSH 노드의 로그는 운영자가 실행 작업에서 'SSH 노드 로그 수집'을 시작한 동안에만 그 운영자의 SSH 연결로 보관됩니다.</p>{/if}
     {#if logs.entries.length===0}<p class="empty">이 시점 전후로 보관된 로그가 없습니다.</p>{:else}
      <ol>{#each logs.entries as entry, index (index)}<li data-log-time={entry.time}><time>{time(entry.time)}</time><code class="log-text">{entry.text}</code></li>{/each}</ol>
      {#if logs.nextCursor}<p class="muted">이 구간에 로그가 더 있습니다. 더 좁은 시점을 선택하세요.</p>{/if}
     {/if}
    {/if}
   </section>
  {/if}
  {#if monitorRange.node}
   <section class="logs" data-testid="range-logs" data-log-node={monitorRange.node}>
    <h4>{monitorRange.node} 범위 로그</h4>
    {#if rangeLogError}<p class="error" role="alert">{rangeLogError}</p>{:else if !rangeLogs}<p class="muted">보관된 로그를 확인하는 중입니다.</p>{:else if rangeLogs.entries.length===0}<p class="empty">이 범위에 보관된 로그가 없습니다.</p>{:else}
     <ol>{#each rangeLogs.entries as entry, index (index)}<li data-log-time={entry.time}><time>{time(entry.time)}</time><code class="log-text">{entry.text}</code></li>{/each}</ol>
     {#if rangeLogs.nextCursor}<p class="muted">이 범위에 로그가 더 있습니다. 범위를 좁히세요.</p>{/if}
    {/if}
   </section>
  {/if}
 {:else if !error}<p class="muted">지표를 불러오는 중입니다.</p>{/if}
</div>
<style>
 .metrics{margin-top:12px;min-width:0}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,320px),1fr));gap:12px}
 figure{margin:0;padding:12px;border:1px solid var(--border-subtle);border-radius:6px;min-width:0}figcaption{color:var(--accent-strong);font-size:13px;margin-bottom:8px}svg{width:100%;height:auto;display:block}circle{cursor:pointer}circle:focus{outline:2px solid var(--warn)}
 .legend{display:flex;flex-wrap:wrap;gap:12px;margin:8px 0}.legend span{display:flex;align-items:center;gap:6px;color:var(--text-muted);font-size:13px}.legend i{display:inline-block;width:10px;height:10px;border-radius:50%}
 [data-testid="metric-coverage"] p,ul{color:var(--text-muted);font-size:13px;margin:4px 0}.incomplete ul{color:var(--warn)}li{overflow-wrap:anywhere}
 .logs{margin-top:12px;padding:12px;border:1px solid var(--border);border-radius:6px}h4{margin:0 0 8px;font-size:14px}label{display:block;color:var(--text-muted);font-size:13px;margin-bottom:8px}select{margin-left:8px;color:var(--accent-strong);background:var(--surface-raised);border:1px solid var(--border-strong);border-radius:6px;padding:4px 8px}
 ol{list-style:none;padding:0;margin:0;max-height:320px;overflow:auto}ol li{display:grid;grid-template-columns:auto 1fr;gap:12px;padding:4px 0;border-bottom:1px solid var(--border-subtle)}time{color:var(--text-muted);font-size:12px;white-space:nowrap}code{font-size:12px;white-space:pre-wrap;overflow-wrap:anywhere}
 .muted,.empty{color:var(--text-muted);font-size:13px}.notice,.error{padding:8px;border-left:3px solid var(--warn);background:var(--warn-bg);color:var(--warn)}.error{border-color:var(--danger);color:var(--danger);background:var(--danger-bg)}
 @media(max-width:800px){ol li{grid-template-columns:1fr;gap:2px}}
</style>
