<script>
  import { webApi } from './web-api.mjs'
  import { allItems } from './pages.mjs'
  let { webSession, roles = [] } = $props()
  const api = $derived(webApi(webSession))
  let users = $state([]), newName = $state(''), newPassword = $state(''), newRole = $state('viewer'), status = $state(''), busy = $state(false)
  const roleName = id => roles.find(r => r.id === id)?.label ?? id
  async function work(fn) { busy = true; status = ''; try { await fn() } catch (e) { status = e.message } finally { busy = false } }
  async function refresh() { users = await allItems(api, 'users') }
  $effect(() => { if (webSession) work(refresh) })
  async function create() { try { await api('users', 'POST', { username: newName, password: newPassword, role: newRole }) } finally { newPassword = '' } await refresh(); status = `${newName} 계정을 만들었습니다` }
  async function change(user, patch, message) { await api(`users/${user.id}`, 'PATCH', patch); await refresh(); status = message }
</script>

<section class="panel" aria-label="사용자 관리">
  <h2>사용자 관리</h2>
  <p class="muted">역할을 낮추거나 계정을 끄면 그 사용자의 세션과 진행 중 작업이 즉시 철회됩니다.</p>
  <ul class="users">{#each users as user (user.id)}
    <li data-testid="user-row"><strong>{user.username}</strong>
      <span class="state" class:off={!user.active}><span aria-hidden="true">{user.active ? '●' : '⊘'}</span> {user.active ? '사용 중' : '꺼짐'}</span>
      <label>역할 <select aria-label={`${user.username} 역할`} value={user.role} disabled={busy || user.id === webSession.user.id} onchange={e => work(() => change(user, { role: e.target.value }, `${user.username} 역할을 ${roleName(e.target.value)}(으)로 바꿨습니다`))}>{#each roles as r}<option value={r.id}>{r.label}</option>{/each}</select></label>
      <button disabled={busy || user.id === webSession.user.id} onclick={() => work(() => change(user, { active: !user.active }, `${user.username} 계정을 ${user.active ? '껐습니다' : '켰습니다'}`))}>{user.username} {user.active ? '끄기' : '켜기'}</button>
    </li>
  {/each}</ul>
  <form onsubmit={e => { e.preventDefault(); work(create) }}>
    <fieldset disabled={busy}><legend>새 사용자</legend>
      <label>사용자 이름 <input aria-label="새 사용자 이름" bind:value={newName} autocomplete="off" required /></label>
      <label>비밀번호 <input aria-label="새 사용자 비밀번호" type="password" bind:value={newPassword} minlength="12" autocomplete="new-password" required /></label>
      <label>역할 <select aria-label="새 사용자 역할" bind:value={newRole}>{#each roles as r}<option value={r.id}>{r.label}</option>{/each}</select></label>
      <button>사용자 만들기</button>
    </fieldset>
  </form>
  <p role="status" data-testid="user-status">{status}</p>
</section>

<style>
  .panel { margin-top: 24px; padding: 24px; border: 1px solid var(--border); border-radius: 10px; background: var(--surface); }
  .muted { color: var(--text-muted); }
  .users { list-style: none; padding: 0; } .users li { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; border-top: 1px solid var(--border-subtle); padding: 8px 0; }
  .state { color: var(--ok); } .state.off { color: var(--danger); }
  fieldset { border: 1px solid var(--border); border-radius: 8px; margin-top: 16px; }
  label { display: grid; gap: 4px; margin: 8px 0; }
  .users label { display: flex; align-items: center; gap: 8px; margin: 0; }
  input, select, button { font: inherit; color: inherit; border: 1px solid var(--border); border-radius: 6px; background: var(--surface-raised); padding: 8px; min-width: 0; }
  button { cursor: pointer; } button:disabled { opacity: .45; cursor: default; }
</style>
