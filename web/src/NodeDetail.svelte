<script>
  import StatusBadge from './StatusBadge.svelte'
  import LogViewer from './LogViewer.svelte'
  import { webApi } from './web-api.mjs'
  import { allItems } from './pages.mjs'
  import { latestSamples, nodeControls } from './node-view.mjs'
  import { requestControl, selectWorkspace } from './selection.svelte.js'
  import { follow, go } from './nav.mjs'
  let { networkId, nodeId, webSession } = $props()
  const api = $derived(webApi(webSession))
  let logSpan = $state(null)
  let network = $state(null), observed = $state(null), workspace = $state(null), documents = $state([]), operations = $state([])
  let metrics = $state(null), activeJobs = $state([]), error = $state(''), busy = $state(false)
  const recorded = $derived(network?.nodes.find(n => n.id === nodeId) ?? null)
  const node = $derived(observed?.nodes.find(n => n.id === nodeId) ?? recorded)
  const busyReason = $derived(activeJobs.length ? `이 네트워크에서 작업 ${activeJobs.map(j => j.operation).join(', ')}이(가) 실행 중입니다. 끝난 뒤 제어하세요.` : '')
  const controls = $derived(node ? nodeControls(node, operations, busyReason) : [])
  const latest = $derived(latestSamples(metrics, nodeId))
  const canControl = $derived(webSession?.user.role !== 'viewer')
  const time = t => t ? new Date(t).toLocaleString() : '—'
  async function work(fn) { busy = true; error = ''; try { await fn() } catch (e) { error = e.message } finally { busy = false } }
  async function load() {
    const networks = (await api('networks')).items
    network = networks.find(n => n.id === networkId) ?? null
    if (!network) throw new Error('기록된 네트워크가 없습니다.')
    selectWorkspace(network.workspaceId)
    workspace = await api('workspaces/' + network.workspaceId)
    documents = await Promise.all(workspace.documents.map(r => api(`documents/${r.id}?revision=${r.revision}`)))
    operations = (await api('contracts/jobs')).operations
    activeJobs = (await allItems(api, 'jobs?workspaceId=' + encodeURIComponent(network.workspaceId))).filter(j => ['accepted', 'running', 'cancelling'].includes(j.state))
    const to = new Date(), from = new Date(to.getTime() - 10 * 60 * 1000)
    logSpan = { from: from.toISOString(), to: to.toISOString() }
    metrics = await api(`networks/${networkId}/metrics?from=${encodeURIComponent(logSpan.from)}&to=${encodeURIComponent(logSpan.to)}`).catch(() => null)
  }
  async function observe() { observed = await api(`networks/${networkId}/observations`) }
  $effect(() => { void networkId; void nodeId; if (webSession) work(load) })
  function control(op) { requestControl(network.workspaceId, nodeId, op); go('/chains') }
</script>

<section class="panel" aria-label="노드 상세">
  <p><a href="/monitoring" onclick={e => follow(e, '/monitoring')}>← 모니터링</a></p>
  <h2>노드 {nodeId}</h2>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if node}
    <dl class="summary">
      <dt>역할</dt><dd data-testid="node-role">{node.role}</dd>
      <dt>소유</dt><dd>{network.ownership === 'owned' ? 'Web이 배포하고 기록한 노드' : 'attach 네트워크의 노드 (제어하지 않음)'}</dd>
      <dt>상태</dt><dd data-testid="node-state"><StatusBadge state={node.state} />{observed ? ` · 확인 ${time(node.observedAt)}` : ' · 기록된 상태 (실제 프로세스는 확인하지 않음)'}</dd>
      <dt>PID</dt><dd>기록 {node.pid || '없음'} · 관측 {observed ? node.observedPid || '없음' : '확인 안 함'}{node.observationReason ? ` · ${node.observationReason}` : ''}</dd>
      <dt>RPC</dt><dd data-testid="node-rpc">{#if latest.length}{#each latest as s}<span class="metric">{s.name} {s.value ?? '값 없음'}{s.unit ? ' ' + s.unit : ''} · {s.source} · {time(s.time)}</span>{/each}{:else}최근 10분 동안 수집된 값이 없습니다.{/if}</dd>
      <dt>서버</dt><dd class="mono">{node.hostIdentity || '—'}</dd>
      <dt>데이터 경로</dt><dd class="mono">{node.dataPath || '—'}</dd>
      <dt>바이너리</dt><dd class="mono">{node.binaryAssetId ? `${node.binaryAssetId} · SHA-256 ${node.binarySHA256}` : observed ? '노드별 지정 없음 · 네트워크를 만든 작업의 기본 바이너리를 씁니다' : '노드별 바이너리는 노드 상태를 확인하면 보입니다'}</dd>
      <dt>구성 revision</dt><dd>{workspace?.name} r{workspace?.revision}{#each documents as d} · {d.name} ({d.kind}) r{d.revision}{/each}</dd>
    </dl>
    <button disabled={busy} onclick={() => work(observe)}>노드 상태 확인</button>
    <h3>제어</h3>
    {#if !canControl}<p class="muted">조회 계정은 노드를 제어하지 않습니다.</p>{:else}
      <p class="muted">제어를 고르면 체인 화면의 실행 계획 검토로 이어집니다. 계획을 검토한 뒤 실행합니다.</p>
      <ul class="controls">{#each controls as c (c.id)}<li>
        <button disabled={!c.enabled || busy} aria-describedby={c.reason ? 'limit-' + c.id : undefined} onclick={() => control(c.id)}>{c.label}</button>
        {#if c.reason}<span id={'limit-' + c.id} class="reason" data-testid={'control-limit-' + c.id}>{c.reason}</span>{/if}
      </li>{/each}</ul>
    {/if}
    <h3>최근 10분 로그</h3>
    {#if logSpan}<LogViewer {networkId} nodes={[nodeId]} node={nodeId} from={logSpan.from} to={logSpan.to} testid="node-logs" />{/if}
  {:else if !error}<p class="muted">노드를 불러오는 중…</p>{/if}
</section>

<style>
  .reason { color: var(--danger); }
  .summary { display: grid; grid-template-columns: 9rem minmax(0, 1fr); gap: var(--space-2) var(--space-4); } dt { color: var(--text-muted); } dd { margin: 0; }
  .metric { display: block; }
  .controls { list-style: none; padding: 0; } .controls li { display: flex; flex-wrap: wrap; gap: var(--space-3); align-items: center; margin: var(--space-2) 0; }
  @media (max-width: 700px) { .summary { grid-template-columns: 1fr; } }
</style>
