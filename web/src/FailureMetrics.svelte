<script>
 // The failed node's chain progress, peers and queued transactions over the
 // same window as its logs, with the failure moment marked, so a stall or a
 // lost peer next to the failure is visible without leaving the page.
 import {formatTime,gapText} from './labels.mjs'
 import {readApi} from './api-client.mjs'
 import {decodeMetrics} from './api-decode.mjs'
 import {chartModel,gapBands,nearestPoints,failureMetrics,timeX} from './metric-chart.mjs'
 import {monitorRange,selectNetwork} from './selection.svelte.js'
 import {go} from './nav.mjs'

 let {networkId,node,from,to,at=null,atLabel='실패 시점'}=$props()

 const W=240,H=56
 let metrics=$state(null),error=$state(''),loading=$state(false)
 let generation=0

 const gauges=$derived(metrics?failureMetrics(metrics,node):[])
 const tolerance=$derived(2*((metrics?.coverage?.bucketSeconds||metrics?.coverage?.sampleIntervalSeconds||5)*1000))

 async function load(){
  const mine=++generation
  loading=true;error=''
  try{
   const value=await readApi(`networks/${encodeURIComponent(networkId)}/metrics?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,{cache:'no-store',decode:decodeMetrics})
   if(mine===generation)metrics=value
  }catch(failure){if(mine===generation){metrics=null;error=failure.message}}
  finally{if(mine===generation)loading=false}
 }
 $effect(()=>{void networkId;void from;void to;if(networkId&&from&&to)load()})

 // The value at the failure moment, or a dash when the node had no sample
 // near it: a gap is never read as the last value before it.
 function valueAt(model){
  if(!at||!model.nodes.length)return null
  return nearestPoints(model,at,tolerance)[0]?.value??null
 }
 const pad=n=>String(n).padStart(2,'0')
 // datetime-local wants minutes; the window is widened to whole minutes.
 const local=(ms)=>{const d=new Date(ms);return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`}
 function openMonitoring(event){
  event.preventDefault()
  selectNetwork(networkId)
  Object.assign(monitorRange,{preset:0,from:local(Math.floor(Date.parse(from)/60000)*60000),to:local(Math.ceil(Date.parse(to)/60000)*60000),node,metric:''})
  go('/monitoring')
 }
</script>

<section class="failure-metrics" data-testid="failure-metrics" data-node={node} aria-label="{node} 실패 구간 지표">
 <header>
  <h4>{node} 지표 · 같은 구간</h4>
  <a href="/monitoring" onclick={openMonitoring}>모니터링 화면에서 이 구간 보기</a>
 </header>
 {#if error}<p class="error" role="alert">지표를 불러오지 못했습니다. {error}</p>
 {:else if loading&&!metrics}<p class="muted">지표를 불러오는 중…</p>
 {:else if metrics&&gauges.every(g=>!g.series.length)}<p class="muted">이 구간에 {node}의 지표가 수집되지 않았습니다.{metrics.coverage?.gaps?.length?' 수집 누락 구간이 있습니다.':''}</p>
 {:else if metrics}
  <div class="row">
   {#each gauges as g (g.label)}
    {@const model=chartModel({...metrics,series:g.series},g.name,{width:W,height:H})}
    {@const marker=model.nodes.length?timeX(model,at):null}
    {@const value=valueAt(model)}
    <figure>
     <figcaption><span>{g.label}</span>{#if at}<b title="{atLabel}의 값">{value??'—'}</b>{/if}</figcaption>
     {#if !model.nodes.length}<p class="none">수집 안 됨 <code>{g.name}</code></p>{:else}
      <svg viewBox="0 0 {W} {H}" preserveAspectRatio="none" role="img" aria-label="{g.label} {model.yMin}–{model.yMax}{model.unit?' '+model.unit:''}, 실패 시점 {value??'값 없음'}">
       {#each gapBands(model,metrics.coverage) as band}<rect class="gap" x={band.x} y="0" width={band.width} height={H}><title>누락 · {gapText(band.gap).text}</title></rect>{/each}
       {#each model.nodes as series}<path d={series.path} fill="none" />{/each}
       {#if marker!==null}<line class="marker" x1={marker} y1="0" x2={marker} y2={H} />{/if}
      </svg>
      <p class="scale"><span>최소 {model.yMin}</span><span>최대 {model.yMax}{model.unit?' '+model.unit:''}</span></p>
     {/if}
    </figure>
   {/each}
  </div>
  <p class="legend"><i class="m"></i>{atLabel} {formatTime(at)} · <i class="g"></i>수집 누락 · 값은 <code>{gauges.map(g=>g.name).join(', ')}</code></p>
 {/if}
</section>

<style>
 .failure-metrics { margin-top: var(--space-4); }
 header { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: baseline; gap: var(--space-1) var(--space-3); }
 header a { font-size: var(--text-sm); }
 .row { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 180px), 1fr)); gap: var(--space-3); margin-top: var(--space-2); }
 figure { margin: 0; min-width: 0; }
 figcaption { display: flex; justify-content: space-between; gap: var(--space-2); color: var(--text-muted); font-size: var(--text-xs); }
 figcaption b { color: var(--text); font: 600 var(--text-sm) var(--font-mono); font-variant-numeric: tabular-nums; }
 svg { display: block; width: 100%; height: 56px; margin-top: var(--space-1); border-bottom: 1px solid var(--border-subtle); overflow: visible; }
 path { stroke: var(--accent); stroke-width: 1.5; vector-effect: non-scaling-stroke; }
 .marker { stroke: var(--danger); stroke-width: 1.5; stroke-dasharray: 3 2; vector-effect: non-scaling-stroke; }
 .gap { fill: var(--warn-bg); }
 .scale { display: flex; justify-content: space-between; margin: 2px 0 0; color: var(--text-faint); font: var(--text-xs) var(--font-mono); font-variant-numeric: tabular-nums; }
 .none { margin: var(--space-2) 0 0; color: var(--text-faint); font-size: var(--text-xs); }
 .legend { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-1) var(--space-2); margin: var(--space-2) 0 0; color: var(--text-faint); font-size: var(--text-xs); }
 .legend i { display: inline-block; width: 10px; height: 10px; }
 .legend .m { width: 2px; background: var(--danger); } .legend .g { background: var(--warn-bg); border: 1px solid var(--warn-border); }
 .muted { font-size: var(--text-sm); }
</style>
