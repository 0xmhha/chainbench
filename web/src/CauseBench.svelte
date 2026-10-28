<script>
 // The cause bench under a failure: the recorder strip over the failure
 // window, then the node's logs beside its gauge readings. It reads the
 // window's metrics, the node's warnings and errors and the other nodes'
 // errors, and keeps the cursors on the first warning and the first error.
 // at is cursor B's moment once the reader moves it; a parent keeps it in
 // the URL, and the bench opens on it while it falls inside the window.
 import {readApi} from './api-client.mjs'
 import {decodeLogs,decodeMetrics} from './api-decode.mjs'
 import LogViewer from './LogViewer.svelte'
 import RecorderStrip from './RecorderStrip.svelte'
 import Readings from './Readings.svelte'
 import {cursorDefaults} from './recorder.mjs'
 import {LOG_PAGE_MAX,logQuery} from './log-view.mjs'

 let {networkId,nodes=[],node=$bindable(),from,to,start=null,end=null,tests=[],phases=[],chosen=null,onchoose=()=>{},logTitle='',logTestid='cause-logs',focus=null,at=$bindable('')}=$props()

 const ALL='*'
 let metrics=$state(null),events=$state([]),otherEvents=$state([]),error=$state('')
 let a=$state(null),b=$state(null)
 let generation=0

 async function logs(label,levels){
  const q=logQuery({from,to,levels,limit:LOG_PAGE_MAX})
  const page=await readApi(`nodes/${encodeURIComponent(networkId+'.'+label)}/logs?${q}`,{cache:'no-store',decode:decodeLogs})
  return page.entries.map(e=>({...e,node:label}))
 }
 async function load(){
  const mine=++generation
  error=''
  try{
   const [m,mineEvents,others]=await Promise.all([
    readApi(`networks/${encodeURIComponent(networkId)}/metrics?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,{cache:'no-store',decode:decodeMetrics}),
    node&&node!==ALL?logs(node,['ERROR','WARN','CRIT']):Promise.resolve([]),
    Promise.all(nodes.filter(n=>n!==node).map(n=>logs(n,['ERROR','CRIT']).catch(()=>[]))).then(x=>x.flat()),
   ])
   if(mine!==generation)return
   metrics=m;events=mineEvents;otherEvents=others
   resetCursors()
  }catch(failure){if(mine===generation)error=failure.message}
 }
 function resetCursors(){const c=cursorDefaults({events,start:start??from,end:end??to,at,from,to});a=c.a;b=c.b}
 function moveB(v){b=v;at=v}
 function reset(){at='';resetCursors()}
 $effect(()=>{void networkId;void node;void from;void to;void nodes.length;if(networkId&&from&&to)load()})
</script>

{#if error}<p class="error" role="alert">원인 영역의 지표나 로그를 불러오지 못했습니다. {error}</p>{/if}
{#if node&&node!==ALL}
 <RecorderStrip {from} {to} {node} {metrics} {events} {otherEvents} {tests} {phases} {chosen} {onchoose} bind:a bind:b={()=>b,moveB} onreset={reset} />
{:else}
 <p class="muted pick-node">노드 하나를 고르면 그 노드의 기록계 띠와 계기 기록을 봅니다.</p>
{/if}
<div class="bench" class:single={!node||node===ALL}>
 <div class="logs">
  <LogViewer {networkId} {nodes} bind:node={()=>node,v=>{node=v}} {from} {to} initialLevels={['ERROR','WARN']} focus={b??focus} title={logTitle} testid={logTestid} height={360} />
 </div>
 {#if node&&node!==ALL&&metrics}<Readings {metrics} {node} />{/if}
</div>

<style>
 .bench { display: grid; grid-template-columns: minmax(0, 1fr) 320px; gap: 0; border-bottom: var(--rule-strong); }
 .bench.single { grid-template-columns: minmax(0, 1fr); }
 .logs { min-width: 0; padding-right: var(--space-4); border-right: 1px solid var(--border); }
 .bench.single .logs { padding-right: 0; border-right: 0; }
 .bench :global(.readings) { padding: var(--space-3) 0 var(--space-3) var(--space-4); }
 .pick-node { font-size: var(--text-sm); padding: var(--space-3) 0; border-bottom: var(--rule); margin: 0; }
 @media (max-width: 900px) {
  .bench { grid-template-columns: minmax(0, 1fr); }
  .logs { padding-right: 0; border-right: 0; }
  .bench :global(.readings) { padding-left: 0; border-top: var(--rule); }
 }
</style>
