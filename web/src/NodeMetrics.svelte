<script>
 import {onMount} from 'svelte'
 import {chartModel,logWindow,gapSummary} from './metric-chart.mjs'
 let {networkId}=$props()
 const charts=[['block_height','블록 높이 (RPC)'],['peer_count','피어 수 (RPC)'],['chain_head_block','헤드 블록 (metrics)'],['txpool_pending','대기 트랜잭션 (metrics)']]
 const colors=['#60a5fa','#f472b6','#34d399','#fbbf24','#a78bfa','#f87171','#22d3ee','#a3e635']
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
   const response=await fetch(`/api/v1/networks/${encodeURIComponent(networkId)}/metrics`,{cache:'no-store'})
   if(!response.ok)throw new Error(`지표 조회 실패 (${response.status})`)
   const value=await response.json()
   if(order>applied){applied=order;metrics=value;error=''}
  }catch(failure){if(order>applied)error=failure.message}
 }
 async function select(node,point){
  const choice=++selection // Plain counter: $state wraps objects in proxies.
  selected={node,time:point.time,value:point.value};logs=null;logError=''
  const {from,to}=logWindow(point.time,30)
  try{
   const response=await fetch(`/api/v1/nodes/${encodeURIComponent(networkId+'.'+node)}/logs?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&limit=100`,{cache:'no-store'})
   if(!response.ok)throw new Error(`로그 조회 실패 (${response.status})`)
   const value=await response.json()
   if(selection===choice)logs=value // A later selection supersedes this reply.
  }catch(failure){if(selection===choice)logError=failure.message}
 }
 onMount(()=>{load();const timer=setInterval(load,5000);return()=>clearInterval(timer)})
</script>
<div class="metrics">
 {#if error}<p class="error" role="alert">{error}</p>{/if}
 {#if metrics}
  <div data-testid="metric-coverage" class:incomplete={!metrics.coverage.complete}>
   <p>수집 범위 {time(metrics.coverage.from)} – {time(metrics.coverage.to)} · {metrics.coverage.sampleIntervalSeconds}초 간격{metrics.coverage.downsampled?' · 구간 평균으로 축약됨':''} · {metrics.coverage.complete?'누락 없음':'누락 구간 있음'}</p>
   {#if gaps.length}<ul>{#each gaps as gap}<li>{gap}</li>{/each}</ul>{/if}
   <p class="muted">수집하지 못했거나 노드가 노출하지 않는 지표는 0으로 그리지 않고 누락으로 표시합니다.</p>
  </div>
  <div class="legend">{#each allNodes as node}<span><i style:background={nodeColor(node,allNodes)}></i>{node}</span>{/each}</div>
  <div class="grid">
   {#each charts as [name,label] (name)}
    {@const model=chartModel(metrics,name,{width:W,height:H})}
    <figure>
     <figcaption>{label}{#if model.nodes.length} · {model.yMin}–{model.yMax} {model.unit}{/if}</figcaption>
     {#if model.nodes.length===0}<p class="muted">이 범위에 수집된 값이 없습니다.</p>{:else}
      <svg data-testid={'metric-chart-'+name} viewBox={`-6 -6 ${W+12} ${H+12}`} role="img" aria-label={label}>
       {#each model.nodes as series (series.node)}
        <path d={series.path} data-node={series.node} fill="none" stroke={nodeColor(series.node,allNodes)} stroke-width="2"/>
        {#each series.points as point (point.time)}
         <circle cx={point.x} cy={point.y} r="4" fill={nodeColor(series.node,allNodes)} data-node={series.node} data-time={point.time} role="button" tabindex="0" aria-label={`${series.node} ${time(point.time)} ${point.value} ${model.unit} 시점 로그 보기`} onclick={()=>select(series.node,point)} onkeydown={event=>{if(event.key==='Enter'||event.key===' ')select(series.node,point)}}/>
        {/each}
       {/each}
      </svg>
     {/if}
    </figure>
   {/each}
  </div>
  {#if selected}
   <section class="logs" data-testid="time-linked-logs" data-selected-time={selected.time} data-log-node={logs?selected.node:undefined}>
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
 {:else if !error}<p class="muted">지표를 불러오는 중입니다.</p>{/if}
</div>
<style>
 .metrics{margin-top:12px;min-width:0}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,320px),1fr));gap:12px}
 figure{margin:0;padding:12px;border:1px solid #253247;border-radius:6px;min-width:0}figcaption{color:#dbeafe;font-size:13px;margin-bottom:8px}svg{width:100%;height:auto;display:block}circle{cursor:pointer}circle:focus{outline:2px solid #fcd34d}
 .legend{display:flex;flex-wrap:wrap;gap:12px;margin:8px 0}.legend span{display:flex;align-items:center;gap:6px;color:#a8b5c8;font-size:13px}.legend i{display:inline-block;width:10px;height:10px;border-radius:50%}
 [data-testid="metric-coverage"] p,ul{color:#a8b5c8;font-size:13px;margin:4px 0}.incomplete ul{color:#fcd34d}li{overflow-wrap:anywhere}
 .logs{margin-top:12px;padding:12px;border:1px solid #334155;border-radius:6px}h4{margin:0 0 8px;font-size:14px}label{display:block;color:#a8b5c8;font-size:13px;margin-bottom:8px}select{margin-left:8px;color:#dbeafe;background:#192536;border:1px solid #475569;border-radius:6px;padding:4px 8px}
 ol{list-style:none;padding:0;margin:0;max-height:320px;overflow:auto}ol li{display:grid;grid-template-columns:auto 1fr;gap:12px;padding:4px 0;border-bottom:1px solid #253247}time{color:#a8b5c8;font-size:12px;white-space:nowrap}code{font-size:12px;white-space:pre-wrap;overflow-wrap:anywhere}
 .muted,.empty{color:#a8b5c8;font-size:13px}.notice,.error{padding:8px;border-left:3px solid #fcd34d;background:#30251a;color:#fcd34d}.error{border-color:#fca5a5;color:#fca5a5;background:#301d25}
 @media(max-width:800px){ol li{grid-template-columns:1fr;gap:2px}}
</style>
