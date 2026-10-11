<script>
  import { webApi } from './web-api.mjs'
  import { allItems } from './pages.mjs'
  import { draftMeta, contractChanged, changedDefinitions } from './contract-draft.mjs'
 import Field from './DSLField.svelte'
 import {references,finishedGenesisRef,caseAssetRefs,presetIds,sameDeclaration} from './dsl-form.js'
 import {topologyChoices,draftChain} from './chain-contract.mjs'
 let { webSession = undefined, assetRevision = 0, onsaved = () => {} } = $props()
 const role=$derived(webSession?.user.role)
 let contract=$state(null),vocabulary=$state(null),document=$state(null),original=$state(null),presets=$state({})
 let status=$state(''),valid=$state(false),busy=$state(false),fingerprint=$state('')
 let savedCases=$state([]),savedId=$state(''),revision=$state(0),presetRefs=$state([])
 let assets=$state([]),genesisBackup=$state(null)
 const genesisAssets=$derived(assets.filter(a=>a.kind==='template'&&a.compatibility?.format==='json'))
 const selectedGenesis=$derived(caseAssetRefs(document)[0]??'')
 let chainContract=$state(null)
 const chain=$derived(draftChain(document,presets))
 const topology=$derived(topologyChoices(chainContract))
 const suggestions=$derived(references(document,topology.roles,topology.defaults))
 $effect(()=>{const id=chain;chainContract=null;if(webSession&&id)api('contracts/chain-preset?chain='+encodeURIComponent(id)).then(c=>{if(chain===id)chainContract=c}).catch(e=>{status=e.message})})
 const api=$derived(webApi(webSession))
 async function refreshSaved(){savedCases=await allItems(api,'documents?kind=case')}
 async function loadSaved(id){if(!id)return;await work(async()=>{const item=await api('documents/'+encodeURIComponent(id));savedId=item.id;revision=item.revision;presetRefs=item.presetRefs??[];for(const ref of presetRefs){const preset=await api('documents/'+encodeURIComponent(ref.id)+'?revision='+ref.revision);presets={...presets,[preset.content.id]:preset.content}};original=null;genesisBackup=null;change(structuredClone(item.content));if(role!=='viewer'){const prepared=await api('test-cases/import','POST',{content:document,presets});fingerprint=prepared.semanticFingerprint}valid=true;status=`공유 테스트 revision ${revision}`})}
 async function save(){await work(async()=>{presetRefs=await pinPresets();let item;try{item=await api(savedId?'documents/'+savedId:'documents',savedId?'PATCH':'POST',{kind:'case',name:document.id,...draftMeta(contract),content:document,assetRefs:caseAssetRefs(document),presetRefs},revision)}catch(e){if(!contractChanged(e.message))throw e;const before=contract;contract=await api('contracts/dsl');await api('test-cases/import','POST',{content:document,presets});throw new Error('엔진 계약이 바뀌었습니다. 바뀐 부분: '+(changedDefinitions(before,contract).join(', ')||'세부 항목')+'. 새 계약으로 다시 검증했으니 확인한 뒤 다시 저장하세요.')}savedId=item.id;revision=item.revision;onsaved();await refreshSaved();status=`공유 테스트 revision ${revision} 저장됨`})}
 // A case names presets by id; saving pins the shared revision of each, and
 // registers a loaded preset file that no shared revision matches.
 async function pinPresets(){
  const ids=presetIds(document);if(!ids.length)return []
  const shared=await allItems(api,'documents?kind=chain-preset'),refs=[]
  for(const id of ids){
   const wanted=presets[id]
   let doc=shared.find(d=>d.content?.id===id&&(!wanted||sameDeclaration(d.content,wanted)))
   if(!doc){if(!wanted)throw new Error(`프리셋 ${id}가 공유되어 있지 않습니다. 참조 프리셋 파일에서 그 파일을 불러오세요`);doc=await api('documents','POST',{kind:'chain-preset',name:id,contractVersion:draftMeta(contract).contractVersion,content:wanted,assetRefs:[]})}
   refs.push({id:doc.id,revision:doc.revision})
  }
  return refs
 }
 // Presets the case names but no file was loaded for come from shared revisions.
 async function fillSharedPresets(declaration){
  const missing=presetIds(declaration).filter(id=>!presets[id]);if(!missing.length)return
  const shared=await allItems(api,'documents?kind=chain-preset')
  for(const id of missing){const doc=shared.filter(d=>d.content?.id===id).at(-1);if(doc)presets={...presets,[id]:doc.content}}
 }
 async function work(fn){busy=true;try{await fn()}catch(e){valid=false;status=e.message}finally{busy=false}}
 $effect(()=>{if(webSession)work(async()=>{vocabulary=await api('vocabulary');contract=await api('contracts/dsl');await refreshSaved();status='DSL 계약을 읽었습니다'})})
 $effect(()=>{assetRevision;if(webSession)allItems(api,'assets').then(v=>{assets=v}).catch(e=>{status=e.message})})
 async function upload(event){await work(async()=>{const file=event.target.files[0];if(!file)return;const parsed=JSON.parse(await file.text());await fillSharedPresets(parsed);savedId='';revision=0;presetRefs=[];genesisBackup=null;original=structuredClone(parsed);document=structuredClone(parsed);valid=false;fingerprint='';const prepared=await api('test-cases/import','POST',{content:parsed,presets});document=prepared.content;valid=true;fingerprint=prepared.semanticFingerprint;status=prepared.migrationMessage??(prepared.migrated?'v1을 옮겼습니다. 실행 의미는 그대로입니다':'가져왔습니다. 엔진 검증 통과')})}
 async function uploadPreset(event){await work(async()=>{for(const file of event.target.files){const preset=JSON.parse(await file.text());if(!preset.id)throw new Error('프리셋에 id가 필요합니다');presets={...presets,[preset.id]:preset}};status='참조 프리셋을 불러왔습니다'})}
 function change(next){document=next;valid=false;fingerprint='';status='고쳤습니다. 내보내기 전에 검증하세요'}
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
 async function validate(){await work(async()=>{const result=await api('test-cases/import','POST',{content:document,presets});document=result.content;valid=true;fingerprint=result.semanticFingerprint;status='엔진 검증 통과'})}
 function download(content,name){const url=URL.createObjectURL(new Blob([JSON.stringify(content,null,2)],{type:'application/json'}));const a=globalThis.document.createElement('a');a.href=url;a.download=name;a.click();URL.revokeObjectURL(url)}
</script>
<section aria-label="구조 DSL 편집기" class="dsl">
 <h2>테스트 시나리오 편집기</h2><p>엔진 계약을 바탕으로 do/expect 문을 순서대로 만듭니다. 가져온 필드는 바꾸기 전까지 그대로 보입니다.</p>
 {#if webSession}
  <p>{role} · 등록된 동작, 단언, 읽기 {vocabulary?.entries.length??0}개</p>
  <label>공유 테스트 <select aria-label="공유 테스트" value={savedId} onchange={e=>loadSaved(e.target.value)}><option value="">새 테스트</option>{#each savedCases as item}<option value={item.id}>{item.name} · revision {item.revision}</option>{/each}</select></label>
  {#if role!=='viewer'}
  <button onclick={()=>{original=null;savedId='';revision=0;genesisBackup=null;change({schemaVersion:"2",kind:"case",id:"new-test",chainPreset:{chain:"stablenet",binaries:{default:"gstable"},topology:{bp:4}},steps:[{expect:"blockNumber",is:1}]})}}>새 테스트 시나리오</button>
  <label>참조 프리셋 파일 <input aria-label="참조 프리셋 파일" type="file" multiple accept=".json,application/json" onchange={uploadPreset} /></label>
  <label>테스트 JSON 가져오기 <input aria-label="테스트 JSON 가져오기" type="file" accept=".json,application/json" onchange={upload} /></label>
  {#if document&&contract}
   {#if document.schemaVersion==='1'||typeof document.chainPreset==='object'}
    <label>테스트 완성 genesis 자료 <select aria-label="테스트 완성 genesis 자료" disabled={busy} value={selectedGenesis} onchange={e=>selectGenesis(e.target.value)}><option value="">기존 생성 설정 사용</option>{#each genesisAssets as asset}<option value={asset.id}>{asset.name} · {asset.checksum.slice(0,12)}</option>{/each}</select></label>
    <p>선택한 파일을 그대로 사용합니다. 선택을 해제하면 이전 생성 설정을 복원합니다.</p>
    {#if finishedGenesisRef(document)&&!selectedGenesis}<p>기존 선언 참조: {finishedGenesisRef(document)} · 실행하려면 등록된 자료를 선택하세요.</p>{/if}
   {/if}
   <Field schema={document.schemaVersion==='2'?contract.$defs.caseSpec:contract.$defs.v1Spec} root={contract} value={document} onchange={change} {suggestions} />
   <button disabled={busy} onclick={validate}>테스트 시나리오 검증</button>
   <button disabled={!valid||busy} onclick={()=>download(document,'test-case.json')}>테스트 시나리오 내보내기</button>
   {#if savedId}<button disabled={busy} onclick={()=>work(async()=>download(await api('documents/'+encodeURIComponent(savedId)+'/bundle'),'test-case.bundle.json'))}>저장한 테스트 묶음 내보내기</button>{/if}
   <button disabled={!valid||busy} onclick={save}>공유 테스트 저장</button>
   {#if original}<button onclick={()=>download(original,'original-test.json')}>가져온 원본 내보내기</button>{/if}
   {#if presetRefs.length}<p data-testid="dsl-preset-refs">고정한 프리셋: {presetRefs.map(r=>r.id+' · revision '+r.revision).join(', ')}</p>{/if}
   {#if fingerprint}<p data-testid="dsl-fingerprint">실행 지문: {fingerprint}</p>{/if}
   <details><summary>가져온 선언과 현재 선언</summary><pre>{JSON.stringify(original,null,2)}</pre><pre>{JSON.stringify(document,null,2)}</pre></details>
  {/if}
  {:else}<p>읽기 전용 계정입니다</p>{#if document}<pre>{JSON.stringify(document,null,2)}</pre><button onclick={()=>download(document,'test-case.json')}>테스트 시나리오 내보내기</button>{/if}{/if}
 {/if}
 <p role="status" aria-label="DSL 상태">{busy?'처리 중…':status}</p>
</section>
<style>
.dsl{margin-top:24px;padding:24px;border:1px solid var(--border);border-radius:10px;background:var(--surface)}p{color:var(--text-muted)}label{display:grid;gap:4px;margin:8px 0}input,select,button{font:inherit;padding:8px;background:var(--surface-raised);color:var(--text);border:1px solid var(--border);border-radius:6px}button{cursor:pointer;margin:4px}button:disabled{opacity:.4}pre{max-height:18rem;overflow:auto;background:var(--bg);padding:8px}
</style>
