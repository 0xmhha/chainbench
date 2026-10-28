<script>
 // One recorded test run read on the cause bench: the recorder strip over the
 // chosen test's window, the node's logs beside its gauge readings, then the
 // test ledger. The job detail and the history detail read a run the same way.
 // A run without tests is read over its whole recorded window instead.
 import CauseBench from './CauseBench.svelte'
 import TestLedger from './TestLedger.svelte'
 import {testRows,testLogSpan} from './run-timeline.mjs'
 import {benchTest} from './job-view.mjs'

 // nodes are the network's node names; test and node name the chosen test and
 // log node, and a parent binds them to keep the choice (the job detail keeps
 // it in the URL). anchor is the id the page's failure trail scrolls to.
 let {run,nodes=[],networkId=null,test=$bindable(''),node=$bindable(''),anchor='cause',logTestid='cause-logs'}=$props()

 const ALL='*',PAD_SECONDS=5
 const observations=$derived(run?.summary?.observations??null)
 const network=$derived(observations?.networkId??networkId)
 const rows=$derived(testRows(run))
 const chosen=$derived(benchTest(rows,test))
 const span=$derived.by(()=>{
  if(!chosen)return null
  const s=testLogSpan(chosen,PAD_SECONDS)
  return s.exact?s:observations?.from&&observations?.to?{from:observations.from,to:observations.to,exact:false}:null
 })
 // A run still being recorded has no end yet; it is read up to the page's opening.
 const runSpan=$derived(observations?.from?{from:observations.from,to:observations.to??new Date().toISOString()}:null)
 const benchNodes=$derived(nodes.length?nodes:chosen?.nodes??[])
 // The node an assertion or a step failed on is the likeliest cause.
 const evidenceNode=$derived([...(chosen?.failedAsserts??[]).map(a=>a.on),...(chosen?.stepErrors??[]).map(s=>s.on)].find(n=>n&&benchNodes.includes(n))??benchNodes[0]??'')
 const shownNode=$derived(node&&(benchNodes.includes(node)||node===ALL)?node:evidenceNode)
 const nodeName=$derived(shownNode===ALL?'모든 노드':shownNode)
 function choose(row){test=row.id;node=''}
</script>

{#if chosen&&span&&network}
 <section class="cause" id={anchor} aria-label="원인 영역">
  <CauseBench networkId={network} nodes={benchNodes} bind:node={()=>shownNode,v=>{node=v}} from={span.from} to={span.to} start={chosen.startedAt} end={chosen.finishedAt} tests={rows} chosen={chosen.id} onchoose={choose} logTitle={`${nodeName} 로그 · ${chosen.id} 구간${span.exact?`(앞뒤 ${PAD_SECONDS}초)`:'(실행 전체)'} · 오류·경고 먼저`} {logTestid} />
 </section>
 <TestLedger {rows} chosen={chosen.id} onchoose={choose} onnode={n=>{node=n;document.getElementById(anchor)?.scrollIntoView({block:'start',behavior:'smooth'})}} nodes={benchNodes} />
{:else if !rows.length&&runSpan&&network&&benchNodes.length}
 <section class="cause" id={anchor} aria-label="실행 구간 로그">
  <CauseBench networkId={network} nodes={benchNodes} bind:node={()=>shownNode,v=>{node=v}} from={runSpan.from} to={runSpan.to} start={observations.from} end={observations.to} logTitle={`${nodeName} 로그 · 실행 구간 전체 · 오류·경고 먼저`} {logTestid} />
 </section>
{:else}
 {#if rows.length}<TestLedger {rows} chosen={chosen?.id} onchoose={choose} nodes={benchNodes} />{/if}
 <p class="muted none">{observations?'이 실행 구간의 노드 이름을 찾지 못해 로그와 지표를 열 수 없습니다.':'이 실행에는 보관된 노드 로그·지표 구간이 없습니다.'}</p>
{/if}

<style>
 .none { font-size: var(--text-sm); padding: var(--space-3) 0; border-bottom: var(--rule); margin: 0; }
</style>
