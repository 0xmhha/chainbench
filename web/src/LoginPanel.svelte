<script>
  import { onMount } from 'svelte'
  let { session = $bindable(undefined), ready = $bindable(false) } = $props()
  let username=$state(''), password=$state(''), setupToken=$state(''), bootstrap=$state(false), status=$state(''), busy=$state(false)
  let users=$state([]), newName=$state(''), newPassword=$state(''), newRole=$state('viewer')
  async function api(path,method='GET',body){
    const headers={'Content-Type':'application/json'}
    if(session?.csrfToken) headers['X-CSRF-Token']=session.csrfToken
    const r=await fetch('/api/v1/'+path,{method,headers,body:body===undefined?undefined:JSON.stringify(body)})
    if(!r.ok) throw new Error(`${r.status}: ${await r.text()}`)
    return r.status===204?null:r.json()
  }
  async function work(fn){busy=true;status='';try{await fn()}catch(e){status=e.message}finally{busy=false}}
  onMount(async()=>{
    try { const r=await fetch('/api/v1/auth/config'); if(r.status===404){session=undefined;return} if(!r.ok)throw new Error('Account service unavailable'); const c=await r.json();bootstrap=c.bootstrapRequired;try{session=await api('auth/me')}catch{session=null} }catch(e){session=null;status=e.message}finally{ready=true}
  })
  $effect(()=>{ if(!session)return;const timer=setInterval(async()=>{try{const r=await fetch('/api/v1/auth/me');if(r.status===401){session=null;users=[]}}catch{}},2000);return()=>clearInterval(timer) })
  async function login(){await work(async()=>{session=await api(bootstrap?'bootstrap':'auth/login','POST',{username,password,...(bootstrap?{setupToken}:{})});password='';setupToken='';bootstrap=false;users=[]})}
  async function refreshUsers(){users=(await api('users')).items}
</script>
<section aria-label="Account access">
  {#if ready && session===null}
    <h2>{bootstrap?'Create first administrator':'Sign in'}</h2>
    {#if bootstrap}<p>Enter the setup token from the private web data directory.</p>{/if}
    <form onsubmit={e=>{e.preventDefault();login()}}>
      <label>Username <input aria-label="Account username" bind:value={username} autocomplete="username" required /></label>
      <label>Password <input aria-label="Account password" type="password" bind:value={password} autocomplete={bootstrap?'new-password':'current-password'} required /></label>
      {#if bootstrap}<label>Setup token <input aria-label="Setup token" type="password" bind:value={setupToken} required /></label>{/if}
      <button disabled={busy}>{bootstrap?'Create administrator':'Sign in'}</button>
    </form>
  {:else if session}
    <p data-testid="account-session">{session.user.username} · {session.user.role} · {session.mode}
      <button onclick={()=>work(async()=>{await api('auth/logout','POST');session=null;users=[]})}>Sign out</button>
    </p>
    {#if session.user.role==='administrator'}
      <details><summary>User management</summary>
        <button onclick={()=>work(refreshUsers)}>List users</button>
        <form onsubmit={e=>{e.preventDefault();work(async()=>{await api('users','POST',{username:newName,password:newPassword,role:newRole});newPassword='';await refreshUsers()})}}>
          <label>New username <input aria-label="New username" bind:value={newName} required /></label>
          <label>New password <input aria-label="New password" type="password" bind:value={newPassword} minlength="12" required /></label>
          <label>Role <select aria-label="New user role" bind:value={newRole}><option value="viewer">Viewer</option><option value="operator">Operator</option><option value="administrator">Administrator</option></select></label>
          <button disabled={busy}>Create user</button>
        </form>
        {#each users as user}
          <div>{user.username} · {user.role} · {user.active?'active':'disabled'}
            <select aria-label={`Role for ${user.username}`} value={user.role} onchange={e=>work(async()=>{await api(`users/${user.id}`,'PATCH',{role:e.target.value});await refreshUsers()})}><option value="viewer">Viewer</option><option value="operator">Operator</option><option value="administrator">Administrator</option></select>
            <button disabled={user.id===session.user.id} onclick={()=>work(async()=>{await api(`users/${user.id}`,'PATCH',{active:!user.active});await refreshUsers()})}>{user.active?'Disable':'Enable'} {user.username}</button>
          </div>
        {/each}
      </details>
    {/if}
  {/if}
  <p role="status" aria-label="Account status">{status}</p>
</section>
<style>
section{border-bottom:1px solid #344154;padding:1rem 0}label{display:block;margin:.6rem 0}input,select,button{font:inherit;padding:.4rem;background:#202b3b;color:#e3ebf5;border:1px solid #47576d;border-radius:4px}button{cursor:pointer;margin:.3rem}details{padding:.5rem}input,select{margin-left:.5rem}
</style>
