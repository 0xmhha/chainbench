<script>
  import { webApi } from './web-api.mjs'
  import { onMount } from 'svelte'
  let { session = $bindable(undefined), ready = $bindable(false) } = $props()
  let username = $state(''), password = $state(''), setupToken = $state(''), bootstrap = $state(false), status = $state(''), busy = $state(false)
  const api = $derived(webApi(session))
  async function work(fn) { busy = true; status = ''; try { await fn() } catch (e) { status = e.message } finally { busy = false } }
  onMount(async () => {
    try {
      const r = await fetch('/api/v1/auth/config')
      if (r.status === 404) { session = undefined; return }
      if (!r.ok) throw new Error('계정 서비스에 연결할 수 없습니다')
      bootstrap = (await r.json()).bootstrapRequired
      try { session = await api('auth/me') } catch { session = null }
    } catch (e) { session = null; status = e.message } finally { ready = true }
  })
  $effect(() => { if (!session) return; const timer = setInterval(async () => { try { const r = await fetch('/api/v1/auth/me'); if (r.status === 401) session = null } catch {} }, 2000); return () => clearInterval(timer) })
  async function login() { await work(async () => { try { session = await api(bootstrap ? 'bootstrap' : 'auth/login', 'POST', { username, password, ...(bootstrap ? { setupToken } : {}) }) } finally { password = ''; setupToken = '' } bootstrap = false }) }
</script>
<section aria-label="계정 접속">
  {#if ready && session === null}
    <h2>{bootstrap ? '첫 관리자 만들기' : '로그인'}</h2>
    {#if bootstrap}<p>비공개 Web 데이터 디렉터리의 설정 토큰을 입력하세요.</p>{/if}
    <form onsubmit={e => { e.preventDefault(); login() }}>
      <label>사용자 이름 <input aria-label="계정 사용자 이름" bind:value={username} autocomplete="username" required /></label>
      <label>비밀번호 <input aria-label="계정 비밀번호" type="password" bind:value={password} autocomplete={bootstrap ? 'new-password' : 'current-password'} required /></label>
      {#if bootstrap}<label>설정 토큰 <input aria-label="설정 토큰" type="password" bind:value={setupToken} required /></label>{/if}
      <button disabled={busy}>{bootstrap ? '관리자 만들기' : '로그인'}</button>
    </form>
  {:else if session}
    <p data-testid="account-session">{session.user.username} · {session.user.role} · {session.mode}
      <button onclick={() => work(async () => { await api('auth/logout', 'POST'); session = null })}>로그아웃</button>
    </p>
  {/if}
  <p role="status" aria-label="계정 상태">{status}</p>
</section>
<style>
  section { border-bottom: 1px solid var(--border); padding: 16px 0; }
  label { display: grid; gap: 4px; margin: 12px 0; max-width: 360px; }
  input, button { font: inherit; padding: 8px; background: var(--surface-raised); color: var(--text); border: 1px solid var(--border); border-radius: 6px; }
  button { cursor: pointer; margin: 4px; }
</style>
