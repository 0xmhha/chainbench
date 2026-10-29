<script>
 // Running a finished job again: the server prepares a fresh plan from the
 // job's recorded input (the same pinned revisions and the owner's own
 // bindings), the owner reviews it here, and only "검토한 계획 실행" starts it.
 // The plan-and-start flow on the chains screen is untouched; this reuses its
 // two requests (conflicts, then POST jobs) for one recorded input.
 import {apiClient} from './api-client.mjs'
 import {formatTime} from './labels.mjs'
 import {go} from './nav.mjs'
 let {job,webSession}=$props()
 const client=$derived(apiClient(webSession))
 let plan=$state(null),conflicts=$state([]),checked=$state(false),busy=$state(false),error=$state('')
 async function work(fn){busy=true;error='';try{await fn()}catch(e){error=e.message}finally{busy=false}}
 async function check(){
  const id=plan.id;checked=false
  const result=await client.request(`plans/${encodeURIComponent(id)}/conflicts`)
  if(plan?.id===id){conflicts=result.items;checked=true}
  return result.items
 }
 async function prepare(){plan=await client.request(`jobs/${encodeURIComponent(job.id)}/plan`,{method:'POST'});await check()}
 async function start(){
  if((await check()).length)return
  const started=await client.request('jobs',{method:'POST',body:{planId:plan.id},idempotencyKey:plan.id})
  go('/jobs/'+started.id)
 }
</script>

<section class="rerun" aria-label="다시 실행" data-testid="job-rerun">
 {#if !plan}
  <button type="button" disabled={busy} onclick={()=>work(prepare)}>다시 실행 계획 만들기</button>
  <span class="muted">같은 입력으로 새 계획을 만듭니다. 검토한 뒤에 실행합니다.</span>
 {:else}
  <h3>다시 실행 계획</h3>
  <p>{job.operation} 작업을 원래 입력 그대로 다시 실행합니다. 케이스와 문서는 원래 리비전을 쓰고, 그 사이 바뀐 내용은 반영하지 않습니다.</p>
  <dl>
   <dt>순서</dt><dd>{plan.phases.join(' → ')}</dd>
   {#if plan.changes?.length}<dt>바뀌는 것</dt><dd><ul>{#each plan.changes as c}<li>{c}</li>{/each}</ul></dd>{/if}
   <dt>쓰는 자원</dt><dd><ul>{#each plan.resources as r}<li><code>{r}</code></li>{/each}</ul></dd>
   <dt>유효 기한</dt><dd class="num">{formatTime(plan.expiresAt)}</dd>
  </dl>
  {#if !checked}<p class="muted">자원 충돌을 확인하는 중입니다.</p>
  {:else if conflicts.length}<p class="error">다른 작업이 이 자원을 쓰거나 보존하고 있어 실행할 수 없습니다. {conflicts.map(c=>`${c.operation} ${c.jobId.slice(0,8)}`).join(', ')}</p>
  {:else}<p>확인된 자원 충돌이 없습니다. 실행 직전에 다시 검사합니다.</p>{/if}
  <div class="actions">
   <button type="button" class="primary" disabled={busy||!checked||conflicts.length>0} onclick={()=>work(start)}>검토한 계획 실행</button>
   <button type="button" disabled={busy} onclick={()=>work(check)}>충돌 다시 확인</button>
   <button type="button" disabled={busy} onclick={()=>{plan=null;conflicts=[];checked=false}}>그만두기</button>
  </div>
 {/if}
 {#if error}<p class="error" role="alert">다시 실행 계획을 만들거나 시작하지 못했습니다. {error}</p>{/if}
</section>

<style>
 .rerun { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-2) var(--space-3); padding: var(--space-3) 0; border-bottom: var(--rule); }
 .rerun > h3, .rerun > p, .rerun > dl, .rerun > .actions { flex-basis: 100%; margin: 0; }
 .muted { font-size: var(--text-sm); }
 dl { display: grid; grid-template-columns: 7rem minmax(0, 1fr); gap: var(--space-1) var(--space-4); font-size: var(--text-sm); }
 dt { color: var(--text-muted); } dd { margin: 0; overflow-wrap: anywhere; } dd ul { margin: 0; padding-left: 1.1em; }
 @media (max-width: 700px) { dl { grid-template-columns: 1fr; } }
</style>
