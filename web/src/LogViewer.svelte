<script>
 import { formatTime, gapText } from './labels.mjs'
 // One log reader for every screen: search over the whole window on the
 // server, highlight the hits here, filter by level, read one node or all of
 // them interleaved by time, and jump to the lines around a moment.
 import {untrack} from 'svelte'
 import {responseError} from './api-error.mjs'
 import {LOG_LEVELS,levelTone,highlightSegments,mergeByTime,logQuery,visibleRange,nearestIndex} from './log-view.mjs'

 let {networkId,nodes=[],node=$bindable(nodes[0]??''),from=null,to=null,runId=null,initialLevels=[],focus=null,title='',height=420,testid='log-viewer',...attributes}=$props()

 const ALL='*'
 const PAGE=100
 const ROW=22
 const SEARCH_DELAY=300
 const CONTEXT_SECONDS=30

 let search=$state(''),applied=$state(''),levels=$state(untrack(()=>[...initialLevels]))
 let context=$state(null) // {from,to,time}: a narrowed window around one line
 let byNode=$state({}),cursors=$state({}),totals=$state({}),coverage=$state({})
 let loading=$state(false),error=$state(''),loadedFor=$state('')
 let selected=$state(-1),scrollTop=$state(0)
 let viewport=$state(),searchBox=$state()
 let generation=0,timer

 const shownNodes=$derived(node===ALL?nodes:[node])
 const span=$derived(context??{from,to})
 const entries=$derived(mergeByTime(byNode,shownNodes))
 const total=$derived(Object.values(totals).reduce((a,b)=>a+b,0))
 const more=$derived(Object.values(cursors).some(Boolean))
 // A short result keeps the viewer to its lines instead of an empty box.
 const viewHeight=$derived(Math.min(height,entries.length*ROW+14))
 const range=$derived(visibleRange({scrollTop,height,rowHeight:ROW,count:entries.length,overscan:10}))
 // Gaps are listed per node; the same outage on several nodes is one line.
 const gaps=$derived(Object.entries(coverage).flatMap(([label,c])=>(c?.gaps??[]).map(g=>({label,...g}))))
 const gapReasons=$derived([...new Set(gaps.map(g=>g.reason))])
 const gapWords=$derived([...new Set(gaps.map(g=>gapText(g.reason).text.replace(/^\S+ · /,'')))])
 const stamp=$derived(stampText(Object.values(coverage).map(c=>c?.timestampSource)))
 const filtered=$derived(applied.trim()!==''||levels.length>0)
 const statusTail=$derived([entries.length<total?`${entries.length.toLocaleString()}줄 불러옴`:'',selected>=0?`${selected+1}번째 선택`:'',stamp?`시각 출처 ${stamp}`:''].filter(Boolean).map(part=>` · ${part}`).join(''))
 const time=formatTime
 // Rows show the clock only; the window and the selected line carry the date.
 const clock=value=>{const d=new Date(value);return Number.isNaN(d.getTime())?'—':`${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}:${String(d.getSeconds()).padStart(2,'0')}.${String(d.getMilliseconds()).padStart(3,'0')}`}

 function stampText(sources){
  if(!sources.length)return ''
  if(sources.every(s=>s==='source'))return '노드 로그 기록 시각'
  if(sources.every(s=>s==='collection'))return '서버 수집 시각'
  return '노드 기록 시각과 수집 시각 혼합'
 }

 async function page(label,cursor){
  const query=logQuery({runId:cursor||context?null:runId,from:cursor?null:span.from,to:cursor?null:span.to,search:applied,levels,cursor,limit:PAGE})
  const response=await fetch(`/api/v1/nodes/${encodeURIComponent(networkId+'.'+label)}/logs?${query}`,{cache:'no-store'})
  if(!response.ok)throw await responseError(response)
  return response.json()
 }

 // reload fetches the first page of every shown node. A later call supersedes
 // an earlier one still in flight, so a slow reply never overwrites a newer one.
 async function reload(){
  const mine=++generation,labels=shownNodes
  if(!networkId||!labels.length||labels.includes(''))return
  loading=true;error='';selected=-1;scrollTop=0
  if(viewport)viewport.scrollTop=0
  try{
   const pages=await Promise.all(labels.map(l=>page(l,null)))
   if(mine!==generation)return
   byNode=Object.fromEntries(labels.map((l,i)=>[l,pages[i].entries]))
   cursors=Object.fromEntries(labels.map((l,i)=>[l,pages[i].nextCursor]))
   totals=Object.fromEntries(labels.map((l,i)=>[l,pages[i].total??pages[i].entries.length]))
   coverage=Object.fromEntries(labels.map((l,i)=>[l,pages[i].coverage]))
   loadedFor=node
   const target=context?.time??focus
   if(target)queueMicrotask(()=>choose(nearestIndex(entries,target)))
  }catch(failure){if(mine===generation){error=failure.message;byNode={};cursors={};totals={}}}
  finally{if(mine===generation)loading=false}
 }

 async function loadMore(){
  if(loading||!more)return
  const mine=generation
  loading=true
  try{
   const pending=Object.entries(cursors).filter(([,c])=>c)
   const pages=await Promise.all(pending.map(([l,c])=>page(l,c)))
   if(mine!==generation)return
   const next={...byNode},nextCursors={...cursors}
   pending.forEach(([l],i)=>{next[l]=[...next[l],...pages[i].entries];nextCursors[l]=pages[i].nextCursor})
   byNode=next;cursors=nextCursors
  }catch(failure){if(mine===generation)error=failure.message}
  finally{if(mine===generation)loading=false}
 }

 // Every input that changes what the server returns starts a fresh read.
 $effect(()=>{void networkId;void runId;void node;void nodes.length;void span.from;void span.to;void applied;void levels.length;untrack(reload)})

 function typed(value){
  search=value
  clearTimeout(timer)
  timer=setTimeout(()=>{applied=search},SEARCH_DELAY)
 }
 function submit(event){event.preventDefault();clearTimeout(timer);applied=search}
 function clearSearch(){clearTimeout(timer);search='';applied=''}
 function toggle(level){levels=levels.includes(level)?levels.filter(l=>l!==level):[...levels,level]}

 function choose(index){
  if(index<0||index>=entries.length)return
  selected=index
  if(!viewport)return
  const top=index*ROW
  if(top<viewport.scrollTop||top+ROW>viewport.scrollTop+height)viewport.scrollTop=Math.max(0,top-height/2)
 }
 function step(by){choose(selected<0?(by>0?0:entries.length-1):Math.min(entries.length-1,Math.max(0,selected+by)))}

 function aroundLine(entry){
  const at=Date.parse(entry.time)
  clearSearch();levels=[]
  context={from:new Date(at-CONTEXT_SECONDS*1000).toISOString(),to:new Date(at+CONTEXT_SECONDS*1000).toISOString(),time:entry.time}
 }

 function keydown(event){
  const typing=event.target instanceof HTMLInputElement||event.target instanceof HTMLSelectElement
  if(event.key==='/'&&!typing){event.preventDefault();searchBox?.focus();return}
  if(event.key==='Escape'&&event.target===searchBox){clearSearch();return}
  if(typing)return
  if(event.key==='n'||event.key==='j'||event.key==='ArrowDown'){event.preventDefault();step(1)}
  else if(event.key==='N'||event.key==='k'||event.key==='ArrowUp'){event.preventDefault();step(-1)}
 }
 function scrolled(){
  scrollTop=viewport.scrollTop
  if(viewport.scrollTop+height>=viewport.scrollHeight-ROW*5)loadMore()
 }
 async function copy(text){try{await navigator.clipboard.writeText(text)}catch{/* The clipboard may be unavailable; the line stays selectable. */}}
</script>

<section class="log-viewer" data-testid={testid} data-log-node={loadedFor||undefined} aria-label={title||'노드 로그'} onkeydown={keydown} {...attributes}>
 {#if title}<h3>{title}</h3>{/if}
 <div class="controls">
  {#if nodes.length>1}
   <label class="node">로그 노드
    <select name="log-node" aria-label="로그 노드" bind:value={node}>
     {#each nodes as label}<option value={label}>{label}</option>{/each}
     <option value={ALL}>모든 노드 (시간순)</option>
    </select>
   </label>
  {/if}
  <form class="search" role="search" onsubmit={submit}>
   <input bind:this={searchBox} name="log-search" type="search" aria-label="로그 검색" placeholder="로그 검색 (/)" autocomplete="off" value={search} oninput={e=>typed(e.currentTarget.value)} />
  </form>
  <div class="levels" role="group" aria-label="로그 레벨">
   {#each LOG_LEVELS as level}<button type="button" class="chip {levelTone(level)}" aria-pressed={levels.includes(level)} onclick={()=>toggle(level)}>{level}</button>{/each}
  </div>
 </div>

 <p class="status" aria-live="polite">
  {#if loading&&!entries.length}불러오는 중…
  {:else if !error}
   <span class="count">{filtered?`일치 ${total.toLocaleString()}줄`:`${total.toLocaleString()}줄`}</span>{statusTail}
  {/if}
 </p>
 {#if context}<p class="context">{time(context.time)} 전후 {CONTEXT_SECONDS}초를 보고 있습니다. <button type="button" class="link" onclick={()=>{context=null}}>원래 범위로</button></p>{/if}
 {#if error}<p class="error" role="alert">{error}</p>{/if}
 {#if gaps.length}
  <details class="notice gaps"><summary>누락 구간 {gaps.length}개 · {gapWords.join(', ')} · 이 구간의 로그는 보관되지 않았을 수 있습니다</summary>
   <ul>{#each gaps as gap}<li>{shownNodes.length>1?`${gap.label} · `:''}{time(gap.from)} – {time(gap.to)} · {gapText(gap.reason).text} <code>{gapText(gap.reason).code}</code></li>{/each}</ul>
  </details>
 {/if}
 {#if gapReasons.some(r=>r.endsWith('remote_log_collection_unavailable'))}<p class="muted">SSH 노드의 로그는 운영자가 실행 작업에서 'SSH 노드 로그 수집'을 시작한 동안에만 그 운영자의 SSH 연결로 보관됩니다.</p>{/if}

 {#if !error&&!loading&&entries.length===0}
  <p class="empty">{filtered?`이 범위(${time(span.from??coverage[shownNodes[0]]?.from)} – ${time(span.to??coverage[shownNodes[0]]?.to)})에 조건과 일치하는 줄이 없습니다.`:'이 범위에 보관된 로그가 없습니다.'}{gaps.length?' 누락 구간에는 로그가 없을 수 있습니다.':''}</p>
  {#if filtered}<p class="empty-actions"><button type="button" onclick={()=>{clearSearch();levels=[]}}>필터 해제하고 모든 줄 보기</button></p>{/if}
 {:else if entries.length}
  <div class="viewport" bind:this={viewport} style:height="{viewHeight}px" onscroll={scrolled} tabindex="0" role="listbox" aria-label="로그 줄" aria-activedescendant={selected>=0?`${testid}-row-${selected}`:undefined}>
   <ol style:height="{entries.length*ROW}px">
    {#each entries.slice(range.start,range.end) as entry, offset (range.start+offset)}
     {@const index=range.start+offset}
     <li id="{testid}-row-{index}" role="option" aria-selected={index===selected} class:selected={index===selected} style:top="{index*ROW}px" data-log-time={entry.time} data-level={entry.level} onclick={()=>choose(index)}>
      <span class="level {levelTone(entry.level)}">{entry.level??''}</span>
      <time datetime={entry.time}>{clock(entry.time)}</time>
      {#if node===ALL}<span class="from">{entry.node}</span>{/if}
      <code class="log-text">{#each highlightSegments(entry.text,applied.trim()) as part}{#if part.match}<mark>{part.text}</mark>{:else}{part.text}{/if}{/each}</code>
     </li>
    {/each}
   </ol>
  </div>
  {#if more}<button type="button" class="more" disabled={loading} onclick={loadMore}>{loading?'불러오는 중…':`다음 ${PAGE}줄 더 보기`}</button>{/if}
  {#if selected>=0&&entries[selected]}
   {@const line=entries[selected]}
   <div class="detail">
    <p class="meta"><span class="level {levelTone(line.level)}">{line.level??'레벨 없음'}</span> {time(line.time)}{#if node===ALL} · {line.node}{/if}</p>
    <pre>{line.text}</pre>
    <div class="actions">
     <button type="button" onclick={()=>copy(line.text)}>줄 복사</button>
     {#if !context}<button type="button" onclick={()=>aroundLine(line)}>이 줄 전후 {CONTEXT_SECONDS}초 보기</button>{/if}
    </div>
   </div>
  {/if}
  <p class="keys"><kbd>/</kbd> 검색 · <kbd>n</kbd> <kbd>N</kbd> 다음·이전 줄 · <kbd>Esc</kbd> 검색 지우기</p>
 {/if}
</section>

<style>
 .log-viewer { margin-top: var(--space-3); min-width: 0; }
 h3 { margin-bottom: var(--space-2); }
 .controls { display: flex; flex-wrap: wrap; align-items: end; gap: var(--space-2) var(--space-3); }
 .node { flex: 0 0 auto; }
 .search { flex: 1 1 240px; min-width: 0; } .search input { width: 100%; }
 .levels { display: flex; flex-wrap: wrap; gap: var(--space-1); }
 .chip { min-height: 30px; padding: 2px 10px; border-radius: 999px; font: 600 var(--text-xs) var(--font-mono); color: var(--text-muted); background: none; }
 .chip[aria-pressed=true] { color: var(--bg); border-color: transparent; }
 .chip.danger[aria-pressed=true] { background: var(--danger); } .chip.warn[aria-pressed=true] { background: var(--warn); }
 .chip.info[aria-pressed=true] { background: var(--accent); } .chip.faint[aria-pressed=true] { background: var(--text-muted); }
 .status, .context { margin: var(--space-2) 0; color: var(--text-muted); font-size: var(--text-sm); font-variant-numeric: tabular-nums; }
 .count { color: var(--text); font-weight: 600; }
 .viewport { position: relative; overflow: auto; border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); background: var(--bg); }
 ol { position: relative; margin: 0; padding: 0; list-style: none; min-width: max-content; }
 li { position: absolute; left: 0; right: 0; height: 22px; display: flex; align-items: center; gap: var(--space-3); padding: 0 var(--space-3) 0 0; white-space: pre; cursor: default; font: var(--text-xs)/22px var(--font-mono); }
 li:hover { background: var(--surface); }
 li.selected { background: var(--surface-raised); box-shadow: inset 0 0 0 1px var(--accent-border); }
 .level { flex: 0 0 52px; align-self: stretch; padding-left: var(--space-2); border-left: 2px solid transparent; color: var(--text-faint); font-weight: 600; }
 .level.danger { color: var(--danger); border-left-color: var(--danger); } .level.warn { color: var(--warn); border-left-color: var(--warn); }
 .level.info { color: var(--text-muted); }
 time { flex: 0 0 auto; color: var(--text-faint); font-size: var(--text-xs); }
 .from { flex: 0 0 auto; color: var(--accent); }
 .log-text { font-size: var(--text-xs); color: var(--text); overflow-wrap: normal; }
 .more { margin-top: var(--space-2); }
 .gaps summary { cursor: pointer; } .gaps ul { margin: var(--space-2) 0 0; padding-left: var(--space-4); }
 .detail { margin-top: var(--space-3); padding: var(--space-3); border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); background: var(--surface); }
 .detail .meta { margin: 0 0 var(--space-2); color: var(--text-muted); font-size: var(--text-sm); }
 .detail .level { border-left: 0; padding: 0; }
 .detail pre { max-height: 10rem; overflow: auto; margin-bottom: var(--space-2); font-size: var(--text-xs); }
 .empty-actions { margin: 0; text-align: center; }
 .keys { margin: var(--space-2) 0 0; color: var(--text-faint); font-size: var(--text-xs); }
 /* On a narrow screen the message gets the width: the level keeps its colored
    edge and its first letter (E, W, I), and the time moves to the selected
    line's detail. */
 @media (max-width: 700px) {
  li > .level { flex: 0 0 calc(1ch + 6px); padding-left: 4px; overflow: hidden; white-space: nowrap; }
  li > .level.info, li > .level.faint, li > .level.none { border-left-color: var(--border-subtle); }
  li > time { display: none; }
 }
 @media (pointer: coarse) { .keys { display: none; } }
</style>
