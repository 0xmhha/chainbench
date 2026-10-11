<script>
 import StatusBadge from './StatusBadge.svelte'
 import {onMount} from 'svelte'
 import {jobStream} from './job-stream.svelte.js'
 import {selection,selectWorkspace,selectNetwork,monitorRange} from './selection.svelte.js'
 import {timeRange} from './metric-chart.mjs'
 import {follow} from './nav.mjs'
 import {webApi} from './web-api.mjs'
 import {allItems} from './pages.mjs'
 import NodeMetrics from './NodeMetrics.svelte'
 let {webSession=undefined}=$props()
 const feed=jobStream(),live=feed.state
 let now=$state(Date.now()),cancelling=$state(''),cancelError=$state(''),names=$state({})
 const api=$derived(webApi(webSession))
 $effect(()=>{if(webSession)allItems(api,'workspaces').then(items=>{names=Object.fromEntries(items.map(w=>[w.id,w.name]))}).catch(e=>{cancelError=e.message})})
 const snapshot=$derived(live.snapshot)
 const stale=$derived(live.lastObserved!==null&&now-Date.parse(live.lastObserved)>5000)
 const time=value=>value?new Date(value).toLocaleString():'—'
 // Every workspace the user can read, plus any a job or network still names.
 const workspaceIds=$derived([...new Set([...Object.keys(names),...(snapshot?.networks??[]).map(n=>n.workspaceId),...(snapshot?.jobs??[]).map(j=>j.workspaceId)].filter(Boolean))].sort())
 const networks=$derived((snapshot?.networks??[]).filter(n=>!selection.workspaceId||n.workspaceId===selection.workspaceId))
 const shown=$derived(networks.filter(n=>!selection.networkId||n.id===selection.networkId))
 const jobs=$derived((snapshot?.jobs??[]).filter(j=>!selection.workspaceId||j.workspaceId===selection.workspaceId))
 const activeJob=job=>!['succeeded','failed','cancelled','interrupted'].includes(job.state)
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
  <label>Workspace<select aria-label="모니터링 Workspace" value={selection.workspaceId} onchange={e=>{selectWorkspace(e.target.value);selectNetwork('')}}><option value="">전체 Workspace</option>{#each workspaceIds as id}<option value={id}>{names[id]??id}</option>{/each}</select></label>
  <label>네트워크<select aria-label="모니터링 네트워크" value={selection.networkId} onchange={e=>{selectNetwork(e.target.value);monitorRange.node=''}}><option value="">전체 네트워크</option>{#each networks as n}<option value={n.id}>{n.id} · 노드 {n.nodes.length}개</option>{/each}</select></label>
  <label>시간 범위<select aria-label="모니터링 시간 범위" bind:value={monitorRange.preset}><option value={15}>최근 15분</option><option value={60}>최근 1시간</option><option value={360}>최근 6시간</option><option value={1440}>최근 24시간</option><option value={0}>직접 지정</option></select></label>
  {#if monitorRange.preset===0}<label>시작<input aria-label="모니터링 시작 시각" type="datetime-local" bind:value={monitorRange.from} /></label><label>끝<input aria-label="모니터링 끝 시각" type="datetime-local" bind:value={monitorRange.to} /></label>{/if}
  <label>노드<select aria-label="모니터링 노드" bind:value={monitorRange.node}><option value="">모든 노드</option>{#each [...new Set(shown.flatMap(n=>n.nodes.map(x=>x.id)))].sort() as id}<option value={id}>{id}</option>{/each}</select></label>
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
   <div class="table"><table><thead><tr><th>작업 · Workspace</th><th>동작 · 상태</th><th>진행 단계</th><th>노드 처리 · 결과</th></tr></thead><tbody>{#each jobs as job (job.id)}
    <tr data-job-id={job.id}><td><a href={'/jobs/'+job.id} onclick={e=>follow(e,'/jobs/'+job.id)}><code>{job.id}</code></a><small>{names[job.workspaceId]??job.workspaceId}</small>{#if mayCancel(job)}<button disabled={cancelling===job.id||job.state==='cancelling'} onclick={()=>cancel(job)}>작업 취소</button>{/if}</td><td>{job.operation}<strong><StatusBadge state={job.state} /></strong></td><td>{#each job.phases??[] as phase}<small>{phase.name}: {phase.state}</small>{/each}</td><td>{job.nodeDisposition??'확인되지 않음'}{#each job.runIds??[] as id}<small>세션 {id}</small>{/each}{#if job.unresolvedResources?.length}<small>미해결 자원 {job.unresolvedResources.length}건</small>{/if}</td></tr>
   {/each}</tbody></table></div>
  {/if}
  <h3>보관된 네트워크 기록</h3><p class="muted">스냅샷 시점: {time(snapshot.observedAt)}. 기록된 PID는 실제 가동 증거가 아닙니다. 체인 화면에서 노드 상태를 확인할 수 있습니다.</p>
  {#each shown as network (network.id)}<div class="network"><strong>{network.id}</strong><span>{network.ownership} · {network.nodes.length}개 노드</span>{#each network.nodes as node (node.id)}<small><a href={`/nodes/${network.id}/${node.id}`} onclick={e=>follow(e,`/nodes/${network.id}/${node.id}`)}>{node.id}</a> · {node.role} · <StatusBadge state={node.state} /> · 기록 PID {node.pid||'없음'}</small>{/each}</div>{/each}
  <h3>노드 지표</h3><p class="muted">서버가 5초마다 각 노드의 RPC와 metrics endpoint에서 수집해 만료 없이 보관한 값입니다. 점을 선택하면 그 시점 전후의 노드 로그를 보여 줍니다.</p>
  {#each shown as network (network.id)}<div class="network"><strong>{network.id}</strong><NodeMetrics networkId={network.id}/></div>{:else}<p class="empty">지표를 수집할 네트워크가 없습니다.</p>{/each}
 {/if}
</section>
<style>
 .panel{margin-top:24px;border:1px solid var(--border);border-radius:10px;background:var(--surface);padding:24px;min-width:0}
 .heading{display:flex;justify-content:space-between;align-items:center;gap:16px}h2{margin:0;font-size:18px}h3{font-size:16px;margin-top:24px}p,.network span{color:var(--text-muted)}
 button{color:var(--accent-strong);background:var(--surface-raised);border:1px solid var(--border-strong);border-radius:6px;padding:8px 12px}
 .filters{display:flex;flex-wrap:wrap;gap:16px;margin-top:16px}label{display:grid;grid-template-columns:minmax(0,1fr);gap:4px;min-width:0;max-width:100%}input{font:inherit;color:inherit;background:var(--surface-raised);border:1px solid var(--border-strong);border-radius:6px;padding:8px;min-height:40px}select{font:inherit;color:inherit;background:var(--surface-raised);border:1px solid var(--border-strong);border-radius:6px;padding:8px;min-height:40px;max-width:100%}a{color:var(--accent)}
 .notice,.error{padding:12px;border-left:3px solid var(--warn);background:var(--warn-bg);color:var(--warn)}.error{border-color:var(--danger);color:var(--danger);background:var(--danger-bg)}
 .table{overflow:auto}table{width:100%;border-collapse:collapse}th,td{padding:12px;text-align:left;border-bottom:1px solid var(--border-subtle);vertical-align:top}th{color:var(--text-muted);font-size:12px}td{overflow-wrap:anywhere}small,strong{display:block}small{color:var(--text-muted)}code{font-size:11px}.network{padding:12px;border:1px solid var(--border);border-radius:6px;margin-top:8px;overflow-wrap:anywhere}
 @media(max-width:800px){.panel{padding:16px}.heading{align-items:start;flex-direction:column}table,tbody,tr,td{display:block}thead{display:none}td{padding:8px 0;border:0}tr{padding:12px 0;border-bottom:1px solid var(--border)}}
</style>
