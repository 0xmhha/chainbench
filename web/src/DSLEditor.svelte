<script>
  import { responseError } from './api-error.mjs'
 import Field from './DSLField.svelte'
 import {references,finishedGenesisRef,caseAssetRefs,presetIds,sameDeclaration} from './dsl-form.js'
 let { webSession = undefined, assetRevision = 0, onsaved = () => {} } = $props()
 let username=$state(''),password=$state(''),authorization=$state(''),actor=$state(null)
 let contract=$state(null),vocabulary=$state(null),document=$state(null),original=$state(null),presets=$state({})
 let status=$state(''),valid=$state(false),busy=$state(false),fingerprint=$state('')
 let savedCases=$state([]),savedId=$state(''),revision=$state(0),presetRefs=$state([])
 let assets=$state([]),genesisBackup=$state(null)
 const genesisAssets=$derived(assets.filter(a=>a.kind==='template'&&a.compatibility?.format==='json'))
 const selectedGenesis=$derived(caseAssetRefs(document)[0]??'')
 const suggestions=$derived(references(document))
 async function api(path,method='GET',body,expectedRevision){const response=await fetch('/api/v1/'+path,{method,headers:{Authorization:authorization,'Content-Type':'application/json',...(webSession?{'X-CSRF-Token':webSession.csrfToken}:{}),...(expectedRevision?{'If-Match':`"${expectedRevision}"`}:{})},body:body===undefined?undefined:JSON.stringify(body)});if(!response.ok)throw await responseError(response);return response.json()}
 async function refreshSaved(){savedCases=(await api('documents?kind=case')).items}
 async function loadSaved(id){if(!id)return;await work(async()=>{const item=await api('documents/'+encodeURIComponent(id));savedId=item.id;revision=item.revision;presetRefs=item.presetRefs??[];for(const ref of presetRefs){const preset=await api('documents/'+encodeURIComponent(ref.id)+'?revision='+ref.revision);presets={...presets,[preset.content.id]:preset.content}};original=null;genesisBackup=null;change(structuredClone(item.content));if(actor.role!=='viewer'){const prepared=await api('test-cases/import','POST',{content:document,presets});fingerprint=prepared.semanticFingerprint}valid=true;status=`공유 테스트 revision ${revision}`})}
 async function save(){await work(async()=>{presetRefs=await pinPresets();const item=await api(savedId?'documents/'+savedId:'documents',savedId?'PATCH':'POST',{kind:'case',name:document.id,contractVersion:'2',content:document,assetRefs:caseAssetRefs(document),presetRefs},revision);savedId=item.id;revision=item.revision;onsaved();await refreshSaved();status=`공유 테스트 revision ${revision} 저장됨`})}
 // A case names presets by id; saving pins the shared revision of each, and
 // registers a loaded preset file that no shared revision matches.
 async function pinPresets(){
  const ids=presetIds(document);if(!ids.length)return []
  const shared=(await api('documents?kind=chain-preset')).items,refs=[]
  for(const id of ids){
   const wanted=presets[id]
   let doc=shared.find(d=>d.content?.id===id&&(!wanted||sameDeclaration(d.content,wanted)))
   if(!doc){if(!wanted)throw new Error(`Preset ${id} is not shared; load its file under Referenced preset files`);doc=await api('documents','POST',{kind:'chain-preset',name:id,contractVersion:'2',content:wanted,assetRefs:[]})}
   refs.push({id:doc.id,revision:doc.revision})
  }
  return refs
 }
 // Presets the case names but no file was loaded for come from shared revisions.
 async function fillSharedPresets(declaration){
  const missing=presetIds(declaration).filter(id=>!presets[id]);if(!missing.length)return
  const shared=(await api('documents?kind=chain-preset')).items
  for(const id of missing){const doc=shared.filter(d=>d.content?.id===id).at(-1);if(doc)presets={...presets,[id]:doc.content}}
 }
 async function work(fn){busy=true;try{await fn()}catch(e){valid=false;status=e.message}finally{busy=false}}
 $effect(()=>{if(webSession)work(async()=>{actor={id:webSession.user.id,role:webSession.user.role==='administrator'?'admin':webSession.user.role};vocabulary=await api('vocabulary');contract=await api('contracts/dsl');await refreshSaved();status='DSL contract loaded'})})
 $effect(()=>{assetRevision;if(actor)api('assets').then(v=>{assets=v.items}).catch(e=>{status=e.message})})
 async function login(){await work(async()=>{authorization='Basic '+btoa(username+':'+password);actor=await api('deployment-account');password='';vocabulary=await api('vocabulary');contract=await api('contracts/dsl');await refreshSaved();status='DSL contract loaded'})}
 async function upload(event){await work(async()=>{const file=event.target.files[0];if(!file)return;const parsed=JSON.parse(await file.text());await fillSharedPresets(parsed);savedId='';revision=0;presetRefs=[];genesisBackup=null;original=structuredClone(parsed);document=structuredClone(parsed);valid=false;fingerprint='';const prepared=await api('test-cases/import','POST',{content:parsed,presets});document=prepared.content;valid=true;fingerprint=prepared.semanticFingerprint;status=prepared.migrationMessage??(prepared.migrated?'v1 migrated; executable meaning preserved':'Imported; engine validation passed')})}
 async function uploadPreset(event){await work(async()=>{for(const file of event.target.files){const preset=JSON.parse(await file.text());if(!preset.id)throw new Error('Preset needs id');presets={...presets,[preset.id]:preset}};status='Preset references loaded'})}
 function change(next){document=next;valid=false;fingerprint='';status='Edited; validate before export'}
 function selectGenesis(id){
  const next=structuredClone($state.snapshot(document)),legacy=next.schemaVersion==='1'
  const env=legacy?next.chain:next.chainPreset
  if(id&&!selectedGenesis)genesisBackup=legacy?Object.fromEntries(Object.entries(env).filter(([key])=>key.startsWith('genesis'))):structuredClone(env.genesis??null)
  if(legacy){for(const key of Object.keys(env))if(key.startsWith('genesis'))delete env[key];if(id)env.genesisExisting='asset:'+id;else if(genesisBackup)Object.assign(env,genesisBackup)}
  else if(id)env.genesis={mode:'existing',ref:'asset:'+id}
  else if(genesisBackup)env.genesis=structuredClone($state.snapshot(genesisBackup))
  else delete env.genesis
  if(!id)genesisBackup=null
  change(next)
 }
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
  <button onclick={()=>{original=null;savedId='';revision=0;genesisBackup=null;change({schemaVersion:"2",kind:"case",id:"new-test",chainPreset:{chain:"stablenet",binaries:{default:"gstable"},topology:{bp:4}},steps:[{expect:"blockNumber",is:1}]})}}>New test scenario</button>
  <label>Referenced preset files <input aria-label="Referenced preset files" type="file" multiple accept=".json,application/json" onchange={uploadPreset} /></label>
  <label>Import test JSON <input aria-label="Import test JSON" type="file" accept=".json,application/json" onchange={upload} /></label>
  {#if document&&contract}
   {#if document.schemaVersion==='1'||typeof document.chainPreset==='object'}
    <label>테스트 완성 genesis 자료 <select aria-label="테스트 완성 genesis 자료" disabled={busy} value={selectedGenesis} onchange={e=>selectGenesis(e.target.value)}><option value="">기존 생성 설정 사용</option>{#each genesisAssets as asset}<option value={asset.id}>{asset.name} · {asset.checksum.slice(0,12)}</option>{/each}</select></label>
    <p>선택한 파일을 그대로 사용합니다. 선택을 해제하면 이전 생성 설정을 복원합니다.</p>
    {#if finishedGenesisRef(document)&&!selectedGenesis}<p>기존 선언 참조: {finishedGenesisRef(document)} · 실행하려면 등록된 자료를 선택하세요.</p>{/if}
   {/if}
   <Field schema={document.schemaVersion==='2'?contract.$defs.caseSpec:contract.$defs.v1Spec} root={contract} value={document} onchange={change} {suggestions} />
   <button disabled={busy} onclick={validate}>Validate test scenario</button>
   <button disabled={!valid||busy} onclick={()=>download(document,'test-case.json')}>Export test scenario</button>
   {#if savedId}<button disabled={busy} onclick={()=>work(async()=>download(await api('documents/'+encodeURIComponent(savedId)+'/bundle'),'test-case.bundle.json'))}>Export saved case bundle</button>{/if}
   <button disabled={!valid||busy} onclick={save}>공유 테스트 저장</button>
   {#if original}<button onclick={()=>download(original,'original-test.json')}>Export original import</button>{/if}
   {#if presetRefs.length}<p data-testid="dsl-preset-refs">Pinned presets: {presetRefs.map(r=>r.id+' · revision '+r.revision).join(', ')}</p>{/if}
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
