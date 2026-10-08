<script>
 import Field from './DSLField.svelte'
 import {references} from './dsl-form.js'
 let { webSession = undefined } = $props()
 let username=$state(''),password=$state(''),authorization=$state(''),actor=$state(null)
 let contract=$state(null),vocabulary=$state(null),document=$state(null),original=$state(null),presets=$state({})
 let status=$state(''),valid=$state(false),busy=$state(false),fingerprint=$state('')
 let savedCases=$state([]),savedId=$state(''),revision=$state(0)
 const suggestions=$derived(references(document))
 async function api(path,method='GET',body,expectedRevision){const response=await fetch('/api/v1/'+path,{method,headers:{Authorization:authorization,'Content-Type':'application/json',...(webSession?{'X-CSRF-Token':webSession.csrfToken}:{}),...(expectedRevision?{'If-Match':`"${expectedRevision}"`}:{})},body:body===undefined?undefined:JSON.stringify(body)});const data=await response.json();if(!response.ok)throw new Error(data.errors?.map(e=>e.message).join('; ')??`HTTP ${response.status}`);return data}
 async function refreshSaved(){savedCases=(await api('documents?kind=case')).items}
 async function loadSaved(id){if(!id)return;await work(async()=>{const item=await api('documents/'+encodeURIComponent(id));savedId=item.id;revision=item.revision;original=null;change(structuredClone(item.content));if(actor.role!=='viewer'){const prepared=await api('test-cases/import','POST',{content:document,presets});fingerprint=prepared.semanticFingerprint}valid=true;status=`공유 테스트 revision ${revision}`})}
 async function save(){await work(async()=>{const item=await api(savedId?'documents/'+savedId:'documents',savedId?'PATCH':'POST',{kind:'case',name:document.id,contractVersion:'2',content:document,assetRefs:[]},revision);savedId=item.id;revision=item.revision;await refreshSaved();status=`공유 테스트 revision ${revision} 저장됨`})}
 async function work(fn){busy=true;try{await fn()}catch(e){valid=false;status=e.message}finally{busy=false}}
 $effect(()=>{if(webSession)work(async()=>{actor={id:webSession.user.id,role:webSession.user.role==='administrator'?'admin':webSession.user.role};vocabulary=await api('vocabulary');contract=await api('contracts/dsl');await refreshSaved();status='DSL contract loaded'})})
 async function login(){await work(async()=>{authorization='Basic '+btoa(username+':'+password);actor=await api('deployment-account');password='';vocabulary=await api('vocabulary');contract=await api('contracts/dsl');await refreshSaved();status='DSL contract loaded'})}
 async function upload(event){await work(async()=>{const file=event.target.files[0];if(!file)return;const parsed=JSON.parse(await file.text());savedId='';revision=0;original=structuredClone(parsed);document=structuredClone(parsed);valid=false;fingerprint='';const prepared=await api('test-cases/import','POST',{content:parsed,presets});document=prepared.content;valid=true;fingerprint=prepared.semanticFingerprint;status=prepared.migrationMessage??(prepared.migrated?'v1 migrated; executable meaning preserved':'Imported; engine validation passed')})}
 async function uploadPreset(event){await work(async()=>{for(const file of event.target.files){const preset=JSON.parse(await file.text());if(!preset.id)throw new Error('Preset needs id');presets={...presets,[preset.id]:preset}};status='Preset references loaded'})}
 function change(next){document=next;valid=false;fingerprint='';status='Edited; validate before export'}
 async function validate(){await work(async()=>{const result=await api('test-cases/import','POST',{content:document,presets});document=result.content;valid=true;fingerprint=result.semanticFingerprint;status='Engine validation passed'})}
 function download(content,name){const url=URL.createObjectURL(new Blob([JSON.stringify(content,null,2)],{type:'application/json'}));const a=globalThis.document.createElement('a');a.href=url;a.download=name;a.click();URL.revokeObjectURL(url)}
</script>
<section aria-label="Structured DSL editor" class="dsl">
 <h2>Test scenario editor</h2><p>Build ordered do/expect statements from the engine contract. Imported fields remain visible until you change them.</p>
 {#if !actor}
  <label>DSL username <input aria-label="DSL username" bind:value={username} autocomplete="username" /></label>
  <label>DSL password <input aria-label="DSL password" type="password" bind:value={password} autocomplete="current-password" /></label>
  <button disabled={busy} onclick={login}>Open DSL editor</button>
 {:else}
  <p>{actor.role} · {vocabulary?.entries.length??0} registered actions, assertions and readers</p>
  <label>공유 테스트 <select aria-label="공유 테스트" value={savedId} onchange={e=>loadSaved(e.target.value)}><option value="">새 테스트</option>{#each savedCases as item}<option value={item.id}>{item.name} · revision {item.revision}</option>{/each}</select></label>
  {#if actor.role!=='viewer'}
  <button onclick={()=>{original=null;savedId='';revision=0;change({schemaVersion:"2",kind:"case",id:"new-test",chainPreset:{chain:"stablenet",binaries:{default:"gstable"},topology:{bp:4}},steps:[{expect:"blockNumber",is:1}]})}}>New test scenario</button>
  <label>Referenced preset files <input aria-label="Referenced preset files" type="file" multiple accept=".json,application/json" onchange={uploadPreset} /></label>
  <label>Import test JSON <input aria-label="Import test JSON" type="file" accept=".json,application/json" onchange={upload} /></label>
  {#if document&&contract}
   <Field schema={document.schemaVersion==='2'?contract.$defs.caseSpec:contract.$defs.v1Spec} root={contract} value={document} onchange={change} {suggestions} />
   <button disabled={busy} onclick={validate}>Validate test scenario</button>
   <button disabled={!valid||busy} onclick={()=>download(document,'test-case.json')}>Export test scenario</button>
   <button disabled={!valid||busy} onclick={save}>공유 테스트 저장</button>
   {#if original}<button onclick={()=>download(original,'original-test.json')}>Export original import</button>{/if}
   {#if fingerprint}<p data-testid="dsl-fingerprint">Executable fingerprint: {fingerprint}</p>{/if}
   <details><summary>Import and current declaration</summary><pre>{JSON.stringify(original,null,2)}</pre><pre>{JSON.stringify(document,null,2)}</pre></details>
  {/if}
  {:else}<p>Read-only account</p>{#if document}<pre>{JSON.stringify(document,null,2)}</pre><button onclick={()=>download(document,'test-case.json')}>Export test scenario</button>{/if}{/if}
 {/if}
 <p role="status" aria-label="DSL status">{busy?'Working…':status}</p>
</section>
<style>
.dsl{margin-top:1.5rem;padding:1rem;border:1px solid #344154;border-radius:8px;background:#151b24}p{color:#a7b5c7}label{display:block;margin:.5rem 0}input,button{font:inherit;padding:.4rem;background:#202b3b;color:#e3ebf5;border:1px solid #47576d;border-radius:4px}button{cursor:pointer;margin:.3rem}button:disabled{opacity:.4}pre{max-height:18rem;overflow:auto;background:#10151d;padding:.6rem}
</style>
