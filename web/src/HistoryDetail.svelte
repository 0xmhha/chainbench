<script>
 // A recorded run as a page of the log: the folio, the first line in its
 // ink (the failing test and its evidence, or how the run ended), the cause
 // bench the job detail uses, and the record's facts as a ledger below.
 import CasualtyEntry from './CasualtyEntry.svelte'
 import RunBench from './RunBench.svelte'
 import PhaseWatches from './PhaseWatches.svelte'
 import ConfirmAction from './ConfirmAction.svelte'
 import {decodeMetrics} from './api-decode.mjs'
 import {runHeadline,phaseRows} from './job-view.mjs'
 import {testRows} from './run-timeline.mjs'
 import {formatTime} from './labels.mjs'

 // test, node and at are the chosen test, log node and cursor moment; the
 // history page keeps them in the URL beside the run.
 let {run,client,administrator=false,busy=false,ondownload,onremove,test=$bindable(''),node=$bindable(''),at=$bindable('')}=$props()

 const TERMINAL=['succeeded','failed','cancelled','interrupted']
 const terminal=$derived(TERMINAL.includes(run.state))
 const observations=$derived(run.summary.observations??null)
 const head=$derived(runHeadline(run))
 const headRow=$derived(testRows(run).find(r=>r.id===head.test)??null)
 const evidence=$derived(headRow?.failedAsserts[0]??null)
 const endedAt=$derived(headRow?.finishedAt??run.finishedAt??null)
 const duration=$derived(run.startedAt&&run.finishedAt?Date.parse(run.finishedAt)-Date.parse(run.startedAt):null)
 const operation=$derived(run.summary.operation??run.summary.command??run.id)
 const phases=$derived(phaseRows(run.summary.phases))
 const format=time=>time?formatTime(time):'자료 없음'

 const ANCHOR='history-cause'
 const PHASES='history-phases'
 function reveal(id=ANCHOR){document.getElementById(id)?.scrollIntoView({block:'start',behavior:'smooth'})}

 // The bench needs the run's node names; the metric series of the run's own
 // window name them. A failure here only leaves the bench without nodes.
 let nodes=$state([]),observed=$state(null),checking=$state(false),error=$state(''),pendingDelete=$state(false)
 const metricsPath=$derived(observations?`networks/${encodeURIComponent(observations.networkId)}/metrics?runId=${encodeURIComponent(run.id)}`:'')
 $effect(()=>{
  const path=metricsPath,id=run.id
  nodes=[];observed=null;pendingDelete=false
  if(!path)return
  client.request(path,{decode:decodeMetrics}).then(m=>{if(run.id===id)nodes=[...new Set(m.series.map(s=>s.nodeId))].sort()}).catch(()=>{nodes=[]})
 })
 async function check(){
  checking=true;error=''
  try{observed=await client.request(metricsPath,{decode:decodeMetrics})}catch(e){error=e.message}finally{checking=false}
 }
 async function remove(){
  error=''
  try{await onremove(run);pendingDelete=false}catch(e){error=e.message+'. 결과 목록을 다시 확인한 뒤 필요하면 삭제를 다시 시도하세요.'}
 }
</script>

<div class="folio"><span>실행 기록 · <strong>{operation}</strong> · <code>{run.id}</code></span><span>{run.startedAt?formatTime(run.startedAt).slice(0,10):'시각 없음'}{#if observations} · 네트워크 <code>{observations.networkId.slice(0,8)}</code>{#if nodes.length} · 노드 {nodes.length}{/if}{/if}</span></div>
<CasualtyEntry scope="run" job={{state:run.state,operation,createdAt:run.startedAt}} {head} summary={{active:!terminal}} network={observations?{id:observations.networkId}:null} nodeCount={nodes.length} at={endedAt} {evidence} {duration} trailNode={head.node} onphase={()=>reveal(PHASES)} ontest={()=>{test=head.test;node='';at='';reveal()}} onnode={()=>{test=head.test;node=head.node;reveal()}} />
<div class="actions">
 <button disabled={busy} onclick={()=>ondownload(run.id)}>결과 내보내기</button>
 {#if administrator}<button class="danger" disabled={busy||!terminal} title={terminal?undefined:'끝나지 않은 실행은 삭제할 수 없습니다'} onclick={()=>{pendingDelete=true}}>보관 결과 삭제</button>{/if}
</div>
{#if pendingDelete}<ConfirmAction label="보관 결과 삭제 확인" message="이 결과 사본과 이 실행 구간에만 속한 노드 지표·로그를 삭제합니다. 다른 실행이나 진행 중인 작업이 함께 쓰는 구간, 원본 CLI 세션, 공유 자료와 노드 데이터는 유지됩니다. 삭제는 되돌릴 수 없고, 삭제 기록은 감사 이력에 남습니다." confirmLabel="결과 사본 삭제 확정" {busy} onconfirm={remove} oncancel={()=>{pendingDelete=false}} />{/if}
{#if error}<p class="error" role="alert">{error}</p>{/if}

<RunBench {run} {nodes} bind:test bind:node bind:at anchor={ANCHOR} logTestid="run-logs" />
{#if phases.length}<PhaseWatches rows={phases} anchor={PHASES} />{/if}

<section class="record" aria-label="기록 정보">
 <h3>기록 정보</h3>
 <dl>
  <dt>실행</dt><dd><code>{run.id}</code></dd>
  <dt>원본 세션</dt><dd>{run.sessionRefs.join(', ')||'연결된 엔진 세션 없음'}</dd>
  <dt>시작 · 끝</dt><dd class="num">{format(run.startedAt)} – {run.finishedAt?format(run.finishedAt):'기록 없음'}</dd>
  {#if run.coverage}<dt>수집</dt><dd>{run.coverage.sampleIntervalSeconds}초마다 · {run.coverage.complete?'누락 없음':`누락 구간 ${run.coverage.gaps.length}개`}{run.coverage.downsampled?' · 다운샘플됨':''}</dd>{/if}
  {#if run.summary.missingDimensions?.length}<dt>자료 없음</dt><dd class="warn-text">{run.summary.missingDimensions.join(', ')}</dd>{/if}
  {#if run.summary.captureGaps?.length}<dt>기록 누락</dt><dd><ul>{#each run.summary.captureGaps as gap}<li>{gap}</li>{/each}</ul></dd>{/if}
  {#if observations}
   <dt>보관 구간</dt>
   <dd data-testid="run-observations">
    <span class="num">{format(observations.from)} – {observations.to?format(observations.to):'진행 중'}</span> · 네트워크 <code>{observations.networkId}</code>
    <button class="check" disabled={checking} onclick={check}>이 실행의 지표 확인</button>
    {#if observed}
     <p data-testid="run-observation-series" data-series={observed.series.length}>지표 {observed.series.length}개 · 표본 {observed.series.reduce((n,s)=>n+s.samples.length,0)}개 · {observed.coverage.complete?'누락 없음':'누락 구간 있음'}</p>
     {#if observed.coverage.gaps.length}<ul>{#each observed.coverage.gaps as gap}<li>{gap.reason} ({format(gap.from)} – {format(gap.to)})</li>{/each}</ul>{/if}
    {/if}
   </dd>
  {/if}
 </dl>
 <p class="muted note">큰 정수는 정확한 숫자 문자열로 표시합니다. 내보낸 파일은 원래 JSON 수치 형식을 유지합니다.</p>
 <details><summary>원본 JSON (테스트 판정 · 단언 · 단계 · 사용 자료)</summary><pre>{JSON.stringify(run.summary,null,2)}</pre></details>
 <details><summary>환경 fingerprint</summary><pre>{JSON.stringify(run.fingerprints,null,2)}</pre></details>
</section>

<style>
 .actions { display: flex; flex-wrap: wrap; gap: var(--space-2); padding: var(--space-3) 0; border-bottom: var(--rule); }
 .record { margin-top: var(--space-5); padding-top: var(--space-3); border-top: var(--rule-strong); }
 dl { display: grid; grid-template-columns: 8rem minmax(0, 1fr); gap: 0; margin: var(--space-2) 0 0; font-size: var(--text-sm); }
 dt, dd { padding: var(--space-2) 0; border-bottom: var(--rule); }
 dt { font: 600 var(--text-xs) var(--font-cond); letter-spacing: .04em; color: var(--text-muted); padding-top: calc(var(--space-2) + 2px); }
 dd { margin: 0; overflow-wrap: anywhere; }
 dd ul { margin: 0; padding-left: var(--space-4); }
 dd p { margin: var(--space-1) 0 0; }
 .num { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
 .warn-text { color: var(--warn); }
 .check { margin-left: var(--space-2); }
 .note { font-size: var(--text-xs); }
 summary { cursor: pointer; padding: var(--space-2) 0; color: var(--text-muted); font: 600 var(--text-sm) var(--font-cond); }
 pre { font-size: var(--text-xs); overflow: auto; padding: var(--space-3); background: var(--surface); max-height: 560px; }
 @media (max-width: 700px) {
  dl { grid-template-columns: minmax(0, 1fr); }
  dt { border-bottom: 0; padding-bottom: 0; }
  .check { display: block; margin: var(--space-2) 0 0; }
 }
</style>
