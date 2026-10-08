<script>
 import {onMount} from 'svelte'
 import {snapshotState,applyJobChange} from './job-observation.mjs'
 let snapshot=$state(null),connection=$state('복원 중'),error=$state(''),gap=$state('')
 let drops=$state(null),busDrops=$state(null),lastObserved=$state(null),now=$state(Date.now())
 let refresh=()=>{}
 const stale=$derived(lastObserved!==null&&now-Date.parse(lastObserved)>5000)
 const time=value=>value?new Date(value).toLocaleString():'—'
 onMount(()=>{
  let active=true,epoch=0,stream=null,request=null,retry=null
  function disconnect(){stream?.close();stream=null;clearTimeout(retry)}
  async function restore(reason=''){
   if(!active)return
   if(reason)gap=reason
   disconnect();request?.abort();request=new AbortController()
   const current=++epoch
   connection='복원 중'
   try{
    const response=await fetch('/api/v1/snapshot',{signal:request.signal,cache:'no-store'})
    if(!response.ok)throw new Error(`실행 상태 조회 실패 (${response.status})`)
    const value=await response.json()
    if(!active||current!==epoch)return
    snapshot=snapshotState(snapshot,value);drops=value.jobDeliveryDropsTotal;busDrops=value.busDeliveryDropsTotal;lastObserved=value.observedAt;error=''
    stream=new EventSource('/api/v1/events?cursor='+encodeURIComponent(snapshot.cursor))
    stream.onopen=()=>{if(active&&current===epoch)connection='연결됨'}
    stream.onerror=()=>{
     if(!active||current!==epoch)return
     disconnect();connection='연결 끊김';gap='연결이 끊긴 동안의 이벤트 전체 수신을 보장하지 않습니다. 최신 실행 상태를 다시 확인합니다.'
     retry=setTimeout(()=>restore(),2000)
    }
    for(const type of ['job_changed','version_changed','resync_required','observation'])stream.addEventListener(type,message=>{
     if(!active||current!==epoch)return
     try{
      const event=JSON.parse(message.data),result=applyJobChange(snapshot,{...event,type})
      if(result.resync){restore('변경 이력 누락 또는 서버 재시작을 확인했습니다. 최신 스냅샷으로 복원합니다.');return}
      snapshot=result.state;lastObserved=event.observedAt??lastObserved
      if(event.jobDeliveryDropsTotal!==undefined)drops=event.jobDeliveryDropsTotal
     }catch{restore('수신한 변경 내용을 해석할 수 없어 최신 스냅샷을 다시 확인합니다.')}
    })
   }catch(failure){
    if(!active||current!==epoch||failure.name==='AbortError')return
    error=failure.message;connection='연결 끊김';retry=setTimeout(()=>restore(),2000)
   }
  }
  refresh=()=>restore()
  restore()
  const clock=setInterval(()=>now=Date.now(),1000)
  return()=>{active=false;epoch++;disconnect();request?.abort();clearInterval(clock)}
 })
</script>
<section class="panel" aria-label="실행 상태 모니터링">
 <div class="heading"><div><h2>실행 상태</h2><p>브라우저 연결과 독립적으로 수행되는 서버 작업의 마지막 확정 상태입니다.</p></div><button onclick={()=>refresh()}>실행 상태 새로고침</button></div>
 <p data-testid="job-observation-health" data-cursor={snapshot?.cursor}>작업 스트림: {connection} · 확정 버전: {snapshot?.version??'확인 중'} · 마지막 관측: {time(lastObserved)}{stale?' · stale':''}</p>
 {#if gap}<p class="notice" role="status">{gap}</p>{/if}
 {#if connection==='연결 끊김'}<p class="notice">마지막 수신 상태를 표시합니다. 연결 복원 전에는 최신 상태로 판단하지 마세요.</p>{/if}
 {#if error}<p class="error" role="alert">{error}</p>{/if}
 <p class="muted">현재 서버의 전체 작업 구독자 전달 드롭: {drops??'확인 중'} · 이벤트 버스 전달 드롭: {busDrops??'확인 중'}. 현재 브라우저만의 누락 수는 아닙니다.</p>
 {#if snapshot}
  {#if snapshot.jobs.length===0}<p class="empty">저장된 실행 작업이 없습니다.</p>{:else}
   <div class="table"><table><thead><tr><th>작업 · Workspace</th><th>동작 · 상태</th><th>진행 단계</th><th>노드 처리 · 결과</th></tr></thead><tbody>{#each snapshot.jobs as job (job.id)}
    <tr data-job-id={job.id}><td><code>{job.id}</code><small>{job.workspaceId}</small></td><td>{job.operation}<strong>{job.state}</strong></td><td>{#each job.phases??[] as phase}<small>{phase.name}: {phase.state}</small>{/each}</td><td>{job.nodeDisposition??'확인되지 않음'}{#each job.runIds??[] as id}<small>세션 {id}</small>{/each}{#if job.unresolvedResources?.length}<small>미해결 자원 {job.unresolvedResources.length}건</small>{/if}</td></tr>
   {/each}</tbody></table></div>
  {/if}
  <h3>보관된 네트워크 기록</h3><p class="muted">스냅샷 시점: {time(snapshot.observedAt)}. 기록된 PID는 실제 가동 증거가 아닙니다. 체인 화면에서 노드 상태를 확인할 수 있습니다.</p>
  {#each snapshot.networks as network (network.id)}<div class="network"><strong>{network.id}</strong><span>{network.ownership} · {network.nodes.length}개 노드</span>{#each network.nodes as node (node.id)}<small>{node.id} · {node.role} · {node.state} · 기록 PID {node.pid||'없음'}</small>{/each}</div>{/each}
 {/if}
</section>
<style>
 .panel{margin-top:24px;border:1px solid #334155;border-radius:10px;background:#121b28;padding:24px;min-width:0}
 .heading{display:flex;justify-content:space-between;align-items:center;gap:16px}h2{margin:0;font-size:18px}h3{font-size:16px;margin-top:24px}p,.network span{color:#a8b5c8}
 button{color:#dbeafe;background:#192536;border:1px solid #475569;border-radius:6px;padding:8px 12px}
 .notice,.error{padding:12px;border-left:3px solid #fcd34d;background:#30251a;color:#fcd34d}.error{border-color:#fca5a5;color:#fca5a5;background:#301d25}
 .table{overflow:auto}table{width:100%;border-collapse:collapse}th,td{padding:12px;text-align:left;border-bottom:1px solid #253247;vertical-align:top}th{color:#a8b5c8;font-size:12px}td{overflow-wrap:anywhere}small,strong{display:block}small{color:#a8b5c8}code{font-size:11px}.network{padding:12px;border:1px solid #334155;border-radius:6px;margin-top:8px;overflow-wrap:anywhere}
 @media(max-width:800px){.panel{padding:16px}.heading{align-items:start;flex-direction:column}table,tbody,tr,td{display:block}thead{display:none}td{padding:8px 0;border:0}tr{padding:12px 0;border-bottom:1px solid #334155}}
</style>
