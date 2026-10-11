<script>
  import { onMount } from 'svelte'
  import { jobStream } from './job-stream.svelte.js'
  import { jobSummary } from './job-view.mjs'
  import { follow, go } from './nav.mjs'
  // runs, busDrops and connected come from the shell's legacy event stream.
  let { runs = [], busDrops = null, connected = false, onrefresh = () => {} } = $props()
  const feed = jobStream(), live = feed.state
  const jobs = $derived(live.snapshot?.jobs ?? [])
  const running = $derived(jobs.filter(j => jobSummary(j).active))
  const recent = $derived(jobs.filter(j => !jobSummary(j).active).slice(0, 10))
  const networks = $derived(live.snapshot?.networks ?? [])
  const count = (nodes, state) => nodes.filter(n => n.state === state).length
  const fmt = t => t ? new Date(t).toLocaleString() : '—'
  const runOk = s => s === 'success' || s === 'pass'
  onMount(() => feed.start())
</script>

<div class="overview">
  <article><span>실행 중 작업</span><strong data-testid="running-count">{running.length}</strong><small>서버가 진행 중인 작업</small></article>
  <article><span>기록된 네트워크</span><strong>{networks.length}</strong><small>노드 {networks.reduce((n, x) => n + x.nodes.length, 0)}개</small></article>
  <article><span>관측 누락</span><strong data-testid="observation-drops">{(live.drops ?? 0) + (busDrops ?? 0)}</strong><small>작업 {live.drops ?? '확인 중'} · 이벤트 버스 {busDrops ?? '확인 중'} · 작업 연결 {live.connection} · 이벤트 연결 {connected ? '연결됨' : '끊김'}</small></article>
</div>

<section class="panel" aria-label="실행 중 작업"><h2>실행 중 작업</h2>
  {#if running.length === 0}<p class="muted">진행 중인 작업이 없습니다.</p>{:else}
    <div class="table-wrap"><table><thead><tr><th>작업</th><th>상태</th><th>진행 단계</th><th>시작</th></tr></thead><tbody>
      {#each running as j (j.id)}<tr data-job-id={j.id} class="row" onclick={() => go('/jobs/' + j.id)}><td><a href={'/jobs/' + j.id} onclick={e => { e.stopPropagation(); follow(e, '/jobs/' + j.id) }}>{j.operation}</a><small class="mono">{j.id}</small></td><td><span aria-hidden="true">●</span> {j.state}</td><td>{j.phases.filter(p => p.state === 'succeeded').length}/{j.phases.length} · {j.phases.find(p => p.state === 'running')?.name ?? '대기'}</td><td>{fmt(j.createdAt)}</td></tr>{/each}
    </tbody></table></div>
  {/if}
</section>

<section class="panel" aria-label="노드 상태"><h2>노드 상태</h2>
  <p class="muted">작업이 마지막으로 기록한 상태입니다. 기록된 PID는 실제 가동 증거가 아니며, 실제 프로세스는 체인 화면의 노드 상태 확인으로 봅니다.</p>
  {#each networks as n (n.id)}<p><strong>{n.id}</strong> · {n.ownership} · 실행 {count(n.nodes, 'running')} · 정지 {count(n.nodes, 'stopped')} · 기타 {n.nodes.length - count(n.nodes, 'running') - count(n.nodes, 'stopped')}</p><p class="nodes">{#each n.nodes as node (node.id)}<a href={`/nodes/${n.id}/${node.id}`} onclick={e => follow(e, `/nodes/${n.id}/${node.id}`)}>{node.id} · {node.role} · {node.state}</a>{/each}</p>{:else}<p class="muted">기록된 네트워크가 없습니다.</p>{/each}
</section>

<section class="panel" aria-label="최근 결과"><div class="panel-heading"><h2>최근 결과</h2><button onclick={onrefresh}>새로고침</button></div>
  {#if recent.length}<div class="table-wrap"><table><thead><tr><th>작업</th><th>결과</th><th>노드 처리</th><th>생성</th></tr></thead><tbody>
    {#each recent as j (j.id)}<tr data-job-id={j.id} class="row" onclick={() => go('/jobs/' + j.id)}><td><a href={'/jobs/' + j.id} onclick={e => { e.stopPropagation(); follow(e, '/jobs/' + j.id) }}>{j.operation}</a></td><td class:ok={j.state === 'succeeded'} class:bad={j.state === 'failed'}><span aria-hidden="true">{j.state === 'succeeded' ? '✓' : j.state === 'failed' ? '✕' : '○'}</span> {j.state}</td><td>{j.nodeDisposition}</td><td>{fmt(j.createdAt)}</td></tr>{/each}
  </tbody></table></div>{/if}
  {#if runs.length === 0 && recent.length === 0}<div class="empty"><h3>관측된 실행이 없습니다</h3><p>체인 구성과 테스트 정의를 준비하세요.</p><a href="/chains" onclick={e => follow(e, '/chains')}>체인 구성 열기 →</a></div>{:else if runs.length}
    <h3>엔진 실행 인덱스</h3>
    <div class="table-wrap"><table><thead><tr><th>실행 ID</th><th>단계</th><th>체인</th><th>네트워크</th><th>판정</th></tr></thead><tbody>{#each runs as r (r.id)}<tr><td class="mono">{r.id}</td><td>{r.phase}</td><td>{r.chain}</td><td>{r.network}</td><td class:ok={runOk(r.status)}><span aria-hidden="true">{runOk(r.status) ? '✓' : '○'}</span> {r.status}</td></tr>{/each}</tbody></table></div>
  {/if}
</section>

<style>
  .overview { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin: 24px 0; }
  .overview article, .panel { border: 1px solid var(--border); border-radius: 10px; background: var(--surface); padding: 24px; }
  .overview article { display: grid; gap: 8px; color: var(--text-muted); } .overview strong { color: var(--text); font-size: 32px; font-weight: 550; } .overview small { font-size: 12px; }
  .panel { margin-top: 24px; } .panel-heading { display: flex; justify-content: space-between; align-items: center; gap: 16px; } h2 { font-size: 18px; margin: 0 0 12px; }
  .muted, .empty { color: var(--text-muted); } .empty { padding: 32px 8px; text-align: center; }
  a { color: var(--accent); } button { color: var(--accent-strong); background: var(--surface-raised); border: 1px solid var(--border-strong); border-radius: 6px; padding: 8px 12px; min-height: 40px; cursor: pointer; }
  .table-wrap { overflow: auto; margin-top: 16px; } table { width: 100%; border-collapse: collapse; } th, td { text-align: left; padding: 12px; border-bottom: 1px solid var(--border-subtle); } th { color: var(--text-muted); font-weight: 500; font-size: 12px; }
  .row { cursor: pointer; } .row:hover { background: var(--surface-raised); } small { display: block; color: var(--text-muted); }
  .nodes { display: flex; flex-wrap: wrap; gap: 12px; } .mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; } .ok { color: var(--ok); } .bad { color: var(--danger); }
  @media (max-width: 800px) { .overview { grid-template-columns: 1fr; gap: 8px; } }
</style>
