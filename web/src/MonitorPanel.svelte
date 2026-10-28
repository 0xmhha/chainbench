<script>
 import { formatTime } from './labels.mjs'
 import StatusBadge from './StatusBadge.svelte'
 import {onMount} from 'svelte'
 import {sharedJobStream} from './job-stream.svelte.js'
 import {liveStale,shownState} from './node-view.mjs'
 import {selection,selectWorkspace,selectNetwork,monitorRange} from './selection.svelte.js'
 import {timeRange} from './metric-chart.mjs'
 import {follow} from './nav.mjs'
 import {webApi} from './web-api.mjs'
 import {allItems} from './pages.mjs'
 import NodeMetrics from './NodeMetrics.svelte'
 import PhaseSteps from './PhaseSteps.svelte'
 import {shortId} from './dashboard-view.mjs'
 let {webSession=undefined}=$props()
 const feed=sharedJobStream(),live=feed.state
 let now=$state(Date.now()),cancelling=$state(''),cancelError=$state(''),names=$state({})
 const api=$derived(webApi(webSession))
 $effect(()=>{if(webSession)allItems(api,'workspaces').then(items=>{names=Object.fromEntries(items.map(w=>[w.id,w.name]))}).catch(e=>{cancelError=e.message})})
 const snapshot=$derived(live.snapshot)
 const stale=$derived(liveStale(live,now))
 const time=formatTime
 // Every workspace the user can read, plus any a job or network still names.
 const workspaceIds=$derived([...new Set([...Object.keys(names),...(snapshot?.networks??[]).map(n=>n.workspaceId),...(snapshot?.jobs??[]).map(j=>j.workspaceId)].filter(Boolean))].sort())
 const networks=$derived((snapshot?.networks??[]).filter(n=>!selection.workspaceId||n.workspaceId===selection.workspaceId))
 const shown=$derived(networks.filter(n=>!selection.networkId||n.id===selection.networkId))
 const jobs=$derived((snapshot?.jobs??[]).filter(j=>!selection.workspaceId||j.workspaceId===selection.workspaceId))
 const activeJob=job=>!job.terminal
 const mayCancel=job=>activeJob(job)&&webSession&&webSession.user.role!=='viewer'&&(job.actorId===webSession.user.id||webSession.user.role==='administrator')
 async function cancel(job){
  if(!window.confirm(`${job.operation} 작업(${job.id})을 취소합니다. 노드는 작업의 보존 설정에 따라 처리됩니다.`))return
  cancelling=job.id;cancelError=''
  try{await api(`jobs/${job.id}/cancel`,'POST',{})}catch(e){cancelError=e.message}finally{cancelling=''}
 }
 onMount(()=>{
  const stop=feed.start()
  const clock=setInterval(()=>now=Date.now(),1000)
  return()=>{stop();clearInterval(clock)}
 })
</script>
<section class="panel" aria-label="실행 상태 모니터링">
 <div class="heading"><div><h2>실행 상태</h2><p>브라우저 연결과 독립적으로 수행되는 서버 작업의 마지막 확정 상태입니다.</p></div><button onclick={()=>feed.refresh()}>실행 상태 새로고침</button></div>
 <div class="filters">
  <label>Workspace<select name="monitor-panel-select-1" aria-label="모니터링 Workspace" value={selection.workspaceId} onchange={e=>{selectWorkspace(e.target.value);selectNetwork('')}}><option value="">전체 Workspace</option>{#each workspaceIds as id}<option value={id}>{names[id]??id}</option>{/each}</select></label>
  <label>네트워크<select name="monitor-panel-select-2" aria-label="모니터링 네트워크" value={selection.networkId} onchange={e=>{selectNetwork(e.target.value);monitorRange.node=''}}><option value="">전체 네트워크</option>{#each networks as n}<option value={n.id}>{n.id} · 노드 {n.nodes.length}개</option>{/each}</select></label>
  <label>시간 범위<select name="preset" aria-label="모니터링 시간 범위" bind:value={monitorRange.preset}><option value={15}>최근 15분</option><option value={60}>최근 1시간</option><option value={360}>최근 6시간</option><option value={1440}>최근 24시간</option><option value={0}>직접 지정</option></select></label>
  {#if monitorRange.preset===0}<label>시작<input name="from" aria-label="모니터링 시작 시각" type="datetime-local" bind:value={monitorRange.from} /></label><label>끝<input name="to" aria-label="모니터링 끝 시각" type="datetime-local" bind:value={monitorRange.to} /></label>{/if}
  <label>노드<select name="node" aria-label="모니터링 노드" bind:value={monitorRange.node}><option value="">모든 노드</option>{#each [...new Set(shown.flatMap(n=>n.nodes.map(x=>x.id)))].sort() as id}<option value={id}>{id}</option>{/each}</select></label>
 </div>
 {#if monitorRange.preset===0&&!timeRange(monitorRange)}<p class="error" role="alert">시작 시각이 끝 시각보다 앞서야 합니다.</p>{/if}
 <p data-testid="job-observation-health" data-cursor={snapshot?.cursor}>작업 스트림: {live.connection} · 확정 버전: {snapshot?.version??'확인 중'} · 마지막 관측: {time(live.lastObserved)}{stale?' · stale':''}</p>
 {#if live.gap}<p class="notice" role="status">{live.gap}</p>{/if}
 {#if live.connection==='연결 끊김'}<p class="notice">마지막 수신 상태를 표시합니다. 연결 복원 전에는 최신 상태로 판단하지 마세요.</p>{/if}
 {#if live.error}<p class="error" role="alert">{live.error}</p>{/if}
 {#if cancelError}<p class="error" role="alert">{cancelError}</p>{/if}
 <p class="muted">현재 서버의 전체 작업 구독자 전달 드롭: {live.drops??'확인 중'} · 이벤트 버스 전달 드롭: {live.busDrops??'확인 중'}. 현재 브라우저만의 누락 수는 아닙니다.</p>
 {#if snapshot}
  {#if jobs.length===0}<p class="empty">저장된 실행 작업이 없습니다.</p>{:else}
   <div class="table-wrap"><table class="stack"><thead><tr><th>작업</th><th>상태</th><th>진행 단계</th><th>노드 처리 · 기록</th><th><span class="sr">취소</span></th></tr></thead><tbody>{#each jobs as job (job.id)}
    <tr data-job-id={job.id}>
     <td data-label="작업"><a href={'/jobs/'+job.id} onclick={e=>follow(e,'/jobs/'+job.id)}>{job.operation}</a><small><code title={job.id}>{shortId(job.id)}</code> · {names[job.workspaceId]??job.workspaceId}</small></td>
     <td data-label="상태"><StatusBadge state={job.state} /></td>
     <td data-label="진행 단계"><PhaseSteps phases={job.phases??[]} /></td>
     <td data-label="노드 처리">{job.nodeDisposition??'확인되지 않음'}{#if job.runIds?.length}<small>실행 기록 {job.runIds.length}건</small>{/if}{#if job.unresolvedResources?.length}<small class="warn-text">미해결 자원 {job.unresolvedResources.length}건</small>{/if}</td>
     <td>{#if mayCancel(job)}<button disabled={cancelling===job.id||job.state==='cancelling'} onclick={()=>cancel(job)}>작업 취소</button>{/if}</td>
    </tr>
   {/each}</tbody></table></div>
  {/if}
  <h3>보관된 네트워크 기록</h3><p class="muted">스냅샷 시점: {time(snapshot.observedAt)}. 노드 상태는 서버 수집기가 5초마다 본 마지막 모습이고, 기록된 PID는 실제 가동 증거가 아닙니다. 노드를 제어하기 전에는 노드 상세에서 상태를 다시 확인합니다.</p>
  {#each shown as network (network.id)}<div class="network">
   <p class="folio"><span>네트워크 <code title={network.id}>{network.id}</code></span><span>{network.ownership} · 노드 {network.nodes.length}개</span></p>
   <div class="table-wrap"><table class="stack"><thead><tr><th>노드</th><th>역할</th><th>상태</th><th>관측</th><th>기록 PID</th></tr></thead><tbody>{#each network.nodes as node (node.id)}{@const shown=shownState(node.state,stale)}
    <tr><td data-label="노드"><a href={`/nodes/${network.id}/${node.id}`} onclick={e=>follow(e,`/nodes/${network.id}/${node.id}`)}>{node.id}</a></td><td data-label="역할">{node.role}</td><td data-label="상태"><StatusBadge state={shown.state} last={shown.last} /></td><td class="num" data-label="관측">{node.observationReason!=='not_observed'&&node.state!=='unknown'?time(node.observedAt).slice(11):'—'}</td><td class="num" data-label="기록 PID">{node.pid||'없음'}</td></tr>
   {/each}</tbody></table></div>
  </div>{/each}
  <h3>노드 지표</h3><p class="muted">서버가 5초마다 각 노드의 RPC와 metrics endpoint에서 수집해 만료 없이 보관한 값입니다. 점을 선택하면 그 시점 전후의 노드 로그를 보여 줍니다.</p>
  {#each shown as network (network.id)}<div class="network"><p class="folio"><span>네트워크 <code title={network.id}>{network.id}</code></span></p><NodeMetrics networkId={network.id}/></div>{:else}<p class="empty">지표를 수집할 네트워크가 없습니다.</p>{/each}
 {:else if !live.error}<p class="muted" role="status">작업 상태를 불러오는 중…</p>
 {/if}
</section>
<style>
 .heading{display:flex;justify-content:space-between;align-items:center;gap:var(--space-4)}
 h3{margin-top:var(--space-5);padding-top:var(--space-2);border-top:var(--rule-strong)}
 p{color:var(--text-muted)}
 .filters{display:flex;flex-wrap:wrap;gap:var(--space-4);margin-top:var(--space-4)}
 td small{display:block;overflow-wrap:anywhere}
 .num{font:var(--text-sm) var(--font-mono);font-variant-numeric:tabular-nums;white-space:nowrap}
 .warn-text{color:var(--warn)}
 .network{margin-top:var(--space-3);overflow-wrap:anywhere}
 .network .folio{margin:0}
 .network .table-wrap{margin-top:0}
 .sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0);white-space:nowrap}
 @media(max-width:800px){
  .heading{align-items:start;flex-direction:column}
 }
</style>
