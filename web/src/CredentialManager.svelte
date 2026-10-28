<script>
  import { webApi } from './web-api.mjs'
  import { allItems } from './pages.mjs'
  import ConfirmAction from './ConfirmAction.svelte'
  let { webSession, kinds = [] } = $props()
  const api = $derived(webApi(webSession))
  let items = $state([]), workspaces = $state([]), status = $state(''), busy = $state(false), checks = $state({}), pendingRevoke = $state('')
  let label = $state('SSH 로그인'), kind = $state('private-key'), sshUser = $state(''), privateKey = $state(''), sshPassword = $state(''), passphrase = $state(''), accountKey = $state('')
  const kindName = id => kinds.find(k => k.id === id)?.label ?? id
  const workspaceName = id => workspaces.find(w => w.id === id)?.name ?? id
  async function work(fn) { busy = true; status = ''; try { await fn() } catch (e) { status = e.message } finally { busy = false } }
  async function refresh() { items = await allItems(api, 'credentials'); workspaces = await allItems(api, 'workspaces') }
  $effect(() => { if (webSession) work(refresh) })
  function clearSecrets() { privateKey = ''; passphrase = ''; sshPassword = ''; accountKey = '' }
  async function create() {
    const input = kind === 'account-key' ? { label, kind, accountKey } : { label, kind, sshUser }
    if (kind === 'private-key') { input.privateKey = privateKey; if (passphrase) input.passphrase = passphrase } else if (kind === 'password') input.password = sshPassword
    try { await api('credentials', 'POST', input) } finally { clearSecrets() }
    await refresh(); status = '암호화한 개인 자격증명을 저장했습니다'
  }
  async function check(c, binding) {
    const access = await api(`credentials/${c.id}/check`, 'POST', binding)
    checks = { ...checks, [`${c.id}/${binding.workspaceId}/${binding.serverRef}`]: access }
  }
  const revokeMessage = c => `${c.label} 자격증명을 폐기합니다.${c.activeJobs.length ? ` 이 자격증명을 쓰는 진행 중 작업 ${c.activeJobs.length}개가 취소되고 노드는 보존됩니다.` : ''} 폐기한 자격증명으로는 다시 접속하지 않으며, 폐기는 되돌릴 수 없습니다.`
  async function revoke(c) {
    pendingRevoke = ''
    await api(`credentials/${c.id}`, 'DELETE'); await refresh(); status = `${c.label} 자격증명을 폐기했습니다`
  }
</script>

<section class="panel" aria-label="개인 자격증명">
  <h2>개인 자격증명</h2>
  <p class="muted">자격증명은 서버에 암호화해 저장되고 본인에게만 보입니다. 비밀 값은 다시 보여 주지 않습니다.</p>
  {#if items.length === 0}<p class="muted">등록한 자격증명이 없습니다.</p>{:else}
    <ul class="credentials">{#each items as c (c.id)}
      <li data-testid="credential-row">
        <div class="row-head"><strong>{c.label}</strong><span>{kindName(c.kind)}</span>
          <span class="state" class:revoked={c.revoked}><span aria-hidden="true">{c.revoked ? '⊘' : '●'}</span> {c.revoked ? '폐기됨' : '사용 가능'}</span>
          <span>진행 중 작업 {c.activeJobs.length}개</span>
          {#if !c.revoked}<button disabled={busy} onclick={() => { pendingRevoke = c.id }}>{c.label} 폐기</button>{/if}
        </div>
        {#if pendingRevoke === c.id && !c.revoked}<ConfirmAction label="자격증명 폐기 확인" message={revokeMessage(c)} confirmLabel={`${c.label} 폐기 확정`} {busy} onconfirm={() => work(() => revoke(c))} oncancel={() => { pendingRevoke = '' }} />{/if}
        {#if c.bindings.length === 0}<p class="muted">연결한 서버가 없습니다.</p>{:else}
          <ul>{#each c.bindings as b}{@const access = checks[`${c.id}/${b.workspaceId}/${b.serverRef}`]}
            <li>{workspaceName(b.workspaceId)} · {b.serverRef}
              {#if !c.revoked}<button disabled={busy} onclick={() => work(() => check(c, b))}>{b.serverRef} 접근 확인</button>{/if}
              {#if access}<span data-testid="credential-access">인증 {access.authenticated ? '성공' : '실패'} · 배포 {access.allowedOperations.includes('deploy') ? '허용' : '거절'}{access.reason ? ` · ${access.reason}` : ''}</span>{/if}
            </li>
          {/each}</ul>
        {/if}
      </li>
    {/each}</ul>
  {/if}
  <form onsubmit={e => { e.preventDefault(); work(create) }}>
    <fieldset disabled={busy}><legend>새 자격증명</legend>
      <label>이름 <input name="label" aria-label="자격증명 이름" bind:value={label} required /></label>
      <label>종류 <select name="kind" aria-label="자격증명 종류" bind:value={kind}>{#each kinds as k}<option value={k.id}>{k.label}</option>{/each}</select></label>
      {#if kind !== 'account-key'}<label>SSH 사용자 <input name="ssh-user" aria-label="SSH 사용자" bind:value={sshUser} autocomplete="off" required /></label>{/if}
      {#if kind === 'account-key'}
        <label>테스트 계정 개인키 <input name="account-key" aria-label="테스트 계정 개인키" type="password" bind:value={accountKey} autocomplete="off" required /></label>
      {:else if kind === 'private-key'}
        <label>SSH 개인키 <textarea name="private-key" aria-label="SSH 개인키" bind:value={privateKey} autocomplete="off" required></textarea></label>
        <label>개인키 암호 <input name="passphrase" aria-label="개인키 암호" type="password" bind:value={passphrase} autocomplete="off" /></label>
      {:else}
        <label>SSH 비밀번호 <input name="ssh-password" aria-label="SSH 비밀번호" type="password" bind:value={sshPassword} autocomplete="off" required /></label>
      {/if}
      <button>개인 자격증명 저장</button>
    </fieldset>
  </form>
  <p role="status" data-testid="credential-status">{status}</p>
</section>

<style>
  .credentials { list-style: none; padding: 0; } .credentials > li { border-top: 1px solid var(--border-subtle); padding: var(--space-3) 0; }
  .row-head { display: flex; flex-wrap: wrap; gap: var(--space-3); align-items: center; }
  .state { color: var(--ok); } .state.revoked { color: var(--danger); }
  label { margin: var(--space-3) 0; }
</style>
