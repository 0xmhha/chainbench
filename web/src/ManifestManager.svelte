<script>
  let { webSession = undefined, assetRevision = 0 } = $props()
  let username=$state(''), password=$state(''), authorization=$state(''), actor=$state(null)
  let items=$state([]), assets=$state([]), selected=$state(''), imported=$state(null), template=$state('')
  let assetId=$state(''), status=$state(''), busy=$state(false), result=$state(null)
  const current=$derived(items.find(item=>item.id===selected))
  const writable=$derived(actor?.role==='admin'||actor?.role==='operator')
  async function api(path,method='GET',body) {
    const response=await fetch('/api/v1/'+path,{method,headers:{Authorization:authorization,'Content-Type':'application/json',...(webSession?{'X-CSRF-Token':webSession.csrfToken}:{})},body:body===undefined?undefined:JSON.stringify(body)})
    if(!response.ok)throw new Error(`${response.status}: ${await response.text()}`)
    return response.json()
  }
  async function work(action){busy=true;try{await action()}catch(error){status=error.message}finally{busy=false}}
  async function refresh(){items=(await api('manifests')).items;assets=(await api('manifest-assets')).items}
  $effect(()=>{void assetRevision;if(webSession) work(async()=>{actor={id:webSession.user.id,role:webSession.user.role==='administrator'?'admin':webSession.user.role};await refresh()})})
  async function login(){await work(async()=>{authorization='Basic '+btoa(username+':'+password);actor=await api('deployment-account');password='';await refresh();status='Manifest library loaded'})}
  async function upload(event,isTemplate){await work(async()=>{const file=event.target.files[0];if(!file)return;const text=await file.text();if(isTemplate)template=text;else {const parsed=JSON.parse(text);if(parsed.manifest&&typeof parsed.template==='string'){imported=parsed.manifest;template=parsed.template}else imported=parsed;}status='Declaration loaded; validate before saving'})}
  async function validate(){await work(async()=>{const response=await api('manifests/validate','POST',{manifest:imported,template});status=response.valid?'Engine manifest validation passed':response.errors.join('; ')})}
  async function save(){await work(async()=>{const item=await api('manifests','POST',{manifest:imported,template});await refresh();selected=item.id;status='Saved immutable external manifest'})}
  function download(){const blob=new Blob([JSON.stringify({manifest:current.manifest,template:current.template},null,2)],{type:'application/json'});const url=URL.createObjectURL(blob);const a=document.createElement('a');a.href=url;a.download='chain-manifest.json';a.click();URL.revokeObjectURL(url)}
  async function setup(){await work(async()=>{result=await api(`manifests/${selected}/setup`,'POST',{assetId});status='Engine setup initialized; node remains stopped'})}
</script>

<section class="manifests" aria-label="Chain manifest management">
  <h2>Chain plugins &amp; manifests</h2>
  <p>Use an embedded plugin or import a declaration for an existing engine family. Setup prepares four isolated local nodes and initializes their databases.</p>
  {#if !actor}
    <label>Manifest username <input aria-label="Manifest username" bind:value={username} autocomplete="username" /></label>
    <label>Manifest password <input aria-label="Manifest password" type="password" bind:value={password} autocomplete="current-password" /></label>
    <button disabled={busy} onclick={login}>Open manifest library</button>
  {:else}
    <p>{actor.role} · {#if !webSession}<button onclick={()=>{actor=null;authorization='';items=[];assets=[];result=null}}>Close manifest library</button>{/if}</p>
    <label>Saved chain manifest <select aria-label="Saved chain manifest" bind:value={selected} onchange={()=>result=null}>
      <option value="">Choose a plugin or imported manifest</option>
      {#each items as item}<option value={item.id}>{item.manifest.id} · {item.source}</option>{/each}
    </select></label>
    {#if current}
      <dl><dt>Chain</dt><dd>{current.manifest.id}</dd><dt>Family</dt><dd>{current.manifest.consensus_family}</dd><dt>Dialect</dt><dd>{current.manifest.dialect}</dd><dt>Chain ID</dt><dd>{current.manifest.chain_id}</dd><dt>Protocol</dt><dd>{current.manifest.protocol||current.manifest.id}</dd></dl>
      <button onclick={download}>Export manifest bundle</button>
      <label>Verified binary asset <select aria-label="Verified binary asset" bind:value={assetId}><option value="">Choose compatible binary</option>{#each assets.filter(a=>a.chain===(current.manifest.protocol||current.manifest.id)) as asset}<option value={asset.id}>{asset.id} · {asset.sha256.slice(0,12)}</option>{/each}</select></label>
      {#if writable}<button disabled={busy||!assetId} onclick={setup}>Initialize manifest setup</button>{/if}
    {/if}
    {#if writable}
      <fieldset><legend>Import external manifest</legend>
        <label>Manifest JSON file <input aria-label="Manifest JSON file" type="file" accept=".json,application/json" onchange={e=>upload(e,false)} /></label>
        <label>Genesis template JSON file <input aria-label="Genesis template JSON file" type="file" accept=".json,application/json" onchange={e=>upload(e,true)} /></label>
        {#if imported}<p>{imported.id} · {imported.consensus_family} · {imported.dialect}</p>{/if}
        <button disabled={busy||!imported} onclick={validate}>Validate manifest</button>
        <button disabled={busy||!imported} onclick={save}>Save external manifest</button>
      </fieldset>
    {/if}
    {#if result}<p data-testid="manifest-setup">Setup {result.id} · {result.state.chain} · initialized</p>{/if}
  {/if}
  <p role="status" aria-label="Manifest status">{busy?'Working…':status}</p>
</section>

<style>
.manifests{margin-top:1.5rem;padding:1rem;border:1px solid #344154;border-radius:8px;background:#151b24}
p{color:#a7b5c7}label{display:block;margin:.6rem 0}input,select,button{font:inherit;padding:.4rem;background:#202b3b;color:#e3ebf5;border:1px solid #47576d;border-radius:4px}input,select{margin-left:.5rem;max-width:100%}button{cursor:pointer;margin:.3rem .3rem .3rem 0}button:disabled{opacity:.5;cursor:default}fieldset{border:1px solid #47576d;margin-top:1rem}dl{display:grid;grid-template-columns:7rem 1fr;gap:.3rem}dt{color:#a7b5c7}dd{margin:0}
</style>
