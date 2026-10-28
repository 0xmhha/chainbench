<script>
 // The first line of the job's page: the failure written in red ink with the
 // moment it was recorded, the node ringed only when the bench finds it at
 // fault (its gauges or logs), the expected value struck through
 // beside the actual one, and the trail to each piece of evidence. Jobs that
 // did not fail get the same line in their own ink. The history detail opens a
 // recorded run with the same line; scope keeps its test ids apart from the job's.
 import StatusBadge from './StatusBadge.svelte'
 import {formatTime} from './labels.mjs'
 import {assertDiff} from './run-timeline.mjs'
 let {job,head,summary,network=null,nodeCount=0,at=null,evidence=null,duration=null,elapsed=null,stale=false,connection='',trailNode='',finding=null,scope='job',onphase=()=>{},ontest=()=>{},onnode=()=>{}}=$props()
 // The bench's reading of the asserted node: null until it reads, so the
 // headline never blames a node on the assertion's target alone.
 const judged=$derived(finding&&finding.node===head.node?finding:null)
 const shown=value=>typeof value==='string'?value:JSON.stringify(value)
 // A running job's time ticks in whole seconds; a finished one keeps a tenth.
 const clock=ms=>ms<60000?`${Math.floor(ms/1000)}초`:`${Math.floor(ms/60000)}분 ${Math.floor((ms%60000)/1000)}초`
 const seconds=ms=>ms===null?'—':ms<60000?`${(ms/1000).toFixed(1)}초`:`${Math.floor(ms/60000)}분 ${Math.round((ms%60000)/1000)}초`
 const diff=$derived(evidence?assertDiff(evidence.expected,evidence.actual):'')
 const time=$derived(at?formatTime(at).slice(11):summary.active?'진행 중':'—')
 const kind=$derived({fail:'사고 기록',warn:'중단 기록',ok:'완료 기록',active:'진행 기록',pending:'판정 확인 중'}[head.tone])
</script>
<header class="casualty {head.tone}" data-testid="{scope}-headline">
 <div class="when"><span class="t">{time}</span><small>{kind}</small></div>
 <div class="what">
  <h2>{#if head.test&&head.node}{head.test} 테스트가 {#if judged?.abnormal}<span class="ring">{head.node}</span>{:else}{head.node}{/if}에서 실패했습니다{:else}{head.title}{/if}</h2>
  {#if evidence}
   <p class="ev"><code>{evidence.assert}</code> · 기대 <del>{shown(evidence.expected)}</del> · 실제 <ins>{shown(evidence.actual)}</ins>{#if diff} · 차이 <ins>{diff}</ins>{/if}</p>
  {:else if head.reason}<p class="ev">{head.reason}</p>{/if}
  {#if judged}<p class="finding" class:bad={judged.abnormal} data-testid="{scope}-finding">{judged.text}{#if !judged.abnormal&&judged.compared}. 노드보다 기대값이나 시나리오를 먼저 확인하세요.{/if}</p>{/if}
  {#if head.tone==='fail'&&(head.phase||head.test)}
   <ol class="trail" aria-label="실패 경로">
    {#if head.phase}<li><button type="button" class="link" onclick={onphase}>단계 {head.phase}</button></li>{/if}
    {#if head.test}<li><button type="button" class="link" onclick={ontest}>테스트 {head.test}</button></li>{/if}
    {#if trailNode}<li><button type="button" class="link" onclick={onnode}>노드 {trailNode} 로그</button></li>{/if}
   </ol>
  {/if}
  {#if stale}<p class="notice">실시간 연결이 {connection} 상태입니다. 보이는 상태가 최신이 아닐 수 있습니다.</p>{/if}
 </div>
 <dl class="facts">
  <dt>상태</dt><dd data-testid="{scope}-state" class="stamp"><StatusBadge state={job.state} /></dd>
  <dt>걸린 시간</dt><dd class="num" data-testid="{scope}-elapsed">{summary.active?elapsed===null?'실행 중':`실행 중 · ${clock(elapsed)} 지남`:seconds(duration)}</dd>
  <dt>{scope==='job'?'작업':'명령'}</dt><dd><code>{job.operation}</code></dd>
  <dt>네트워크</dt><dd>{#if network}<code title={network.id}>{network.id.slice(0,8)}</code> · 노드 {nodeCount}{:else}<span class="muted">기록 없음</span>{/if}</dd>
  {#if head.runId}<dt>보관 기록</dt><dd><code title={head.runId}>{head.runId}</code></dd>{/if}
  <dt>시작</dt><dd class="num">{formatTime(job.createdAt)}</dd>
 </dl>
</header>
<style>
 .casualty { display: grid; grid-template-columns: 104px minmax(0, 1fr) minmax(220px, 300px); border-bottom: 1px solid var(--border); }
 .casualty > * { padding: var(--space-4) var(--space-4) var(--space-4); min-width: 0; }
 .when { border-right: 1px solid var(--border); padding-left: 0; }
 .t { display: block; font: 700 var(--text-lg) var(--font-mono); font-variant-numeric: tabular-nums; }
 .when small { display: block; margin-top: var(--space-1); font: 600 var(--text-xs) var(--font-cond); letter-spacing: .04em; color: var(--text-muted); }
 h2 { font: 700 var(--text-display)/1.2 var(--font-sans); letter-spacing: -.01em; text-wrap: balance; }
 .fail .t, .fail h2 { color: var(--danger); }
 .warn .t, .warn h2 { color: var(--warn); }
 .ok .t, .ok h2 { color: var(--ok); }
 .ring { border: 2px solid currentColor; border-radius: 50%; padding: 0 .3em; }
 .ev { margin: var(--space-2) 0 0; font: var(--text-sm) var(--font-mono); max-width: 90ch; }
 .ev del { color: var(--text-muted); } .ev ins { text-decoration: none; color: var(--danger); font-weight: 700; }
 .finding { margin: var(--space-2) 0 0; font-size: var(--text-sm); font-weight: 600; max-width: 90ch; }
 .finding.bad { color: var(--danger); }
 .trail { display: flex; flex-wrap: wrap; gap: var(--space-1) var(--space-4); margin: var(--space-3) 0 0; padding: 0; list-style: none; }
 .trail .link { font: 600 var(--text-sm) var(--font-cond); letter-spacing: .02em; min-height: 24px; }
 @media (pointer: coarse) { .trail .link { min-height: 44px; } }
 .facts { display: grid; grid-template-columns: auto minmax(0, 1fr); gap: 3px var(--space-3); margin: 0; align-content: start; border-left: 1px solid var(--border); font-size: var(--text-sm); }
 .facts dt { font: 600 var(--text-xs) var(--font-cond); letter-spacing: .04em; color: var(--text-muted); padding-top: 2px; }
 .facts dd { margin: 0; overflow-wrap: anywhere; } .facts code { font-size: var(--text-xs); }
 .stamp :global(.status) { display: inline-flex; align-items: center; gap: 4px; padding: 0 6px; border: 2px solid currentColor; font: 700 var(--text-sm) var(--font-cond); letter-spacing: .06em; transform: rotate(-2deg); }
 .num { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
 .what .notice { margin-bottom: 0; }
 @media (max-width: 900px) {
  .casualty { grid-template-columns: minmax(0, 1fr); }
  .when { border-right: 0; padding-bottom: 0; display: flex; gap: var(--space-3); align-items: baseline; }
  .facts { border-left: 0; border-top: var(--rule); padding-left: 0; }
  .what { padding-left: 0; padding-right: 0; }
  h2 { font-size: var(--text-xl); }
 }
</style>
