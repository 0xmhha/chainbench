<script>
  import { webApi } from './web-api.mjs'
  import { openWebAccess } from './web-access.mjs'
  import { onUnauthorized } from './api-client.mjs'
  import { expiryDelay } from './session-watch.mjs'
  import { onMount } from 'svelte'
  let { session = $bindable(undefined), ready = $bindable(false) } = $props()
  let username = $state(''), password = $state(''), setupToken = $state(''), bootstrap = $state(false), status = $state(''), busy = $state(false)
  let accountsEnabled = $state(true)
  const api = $derived(webApi(session))
  async function work(fn) { busy = true; status = ''; try { await fn() } catch (e) { status = e.message } finally { busy = false } }
  async function loadAccess() {
    const access = await openWebAccess()
    accountsEnabled = access.accountsEnabled
    bootstrap = access.bootstrapRequired
    session = access.session
  }
  onMount(async () => {
    try {
      await loadAccess()
    } catch (e) { session = null; status = e.message } finally { ready = true }
  })
  // The session ends at the time the server stated, or earlier when a request
  // is refused (signed out or revoked elsewhere); returning to the tab asks once.
  let checking = false
  async function recheck() {
    if (checking) return
    checking = true
    try { await loadAccess() } catch (e) { status = e.message } finally { checking = false }
  }
  $effect(() => {
    if (!session) return
    const delay = expiryDelay(session)
    const timer = delay === null ? undefined : setTimeout(recheck, delay + 1000)
    const stop = onUnauthorized(recheck)
    const returned = () => { if (document.visibilityState === 'visible') api('auth/me').catch(() => {}) }
    document.addEventListener('visibilitychange', returned)
    return () => { clearTimeout(timer); stop(); document.removeEventListener('visibilitychange', returned) }
  })
  async function login() { await work(async () => { try { session = await api(bootstrap ? 'bootstrap' : 'auth/login', 'POST', { username, password, ...(bootstrap ? { setupToken } : {}) }) } finally { password = ''; setupToken = '' } bootstrap = false }) }
</script>
<section aria-label="계정 접속">
  {#if ready && session === null && accountsEnabled}
    <h2>{bootstrap ? '첫 관리자 만들기' : '로그인'}</h2>
    {#if bootstrap}<p>비공개 Web 데이터 디렉터리의 설정 토큰을 입력하세요.</p>{/if}
    <form onsubmit={e => { e.preventDefault(); login() }}>
      <label>사용자 이름 <input name="username" aria-label="계정 사용자 이름" bind:value={username} autocomplete="username" required /></label>
      <label>비밀번호 <input name="password" aria-label="계정 비밀번호" type="password" bind:value={password} autocomplete={bootstrap ? 'new-password' : 'current-password'} required /></label>
      {#if bootstrap}<label>설정 토큰 <input name="setup-token" aria-label="설정 토큰" type="password" bind:value={setupToken} required /></label>{/if}
      <button disabled={busy}>{bootstrap ? '관리자 만들기' : '로그인'}</button>
    </form>
  {:else if session && accountsEnabled}
    <p data-testid="account-session">{session.user.username} · {session.user.role} · {session.mode}
      <button onclick={() => work(async () => { await api('auth/logout', 'POST'); session = null })}>로그아웃</button>
    </p>
  {/if}
  <p role="status" aria-label="계정 상태">{status}</p>
</section>
<style>
  section:has(form, [data-testid=account-session]) { border-bottom: 1px solid var(--border-subtle); padding: var(--space-4) 0; }
  p[role=status]:empty { display: none; }
  label { margin: var(--space-3) 0; max-width: 360px; }
  button { margin: var(--space-1); }
</style>
