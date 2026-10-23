<script>
  import { webApi } from './web-api.mjs'
  let { webSession = undefined, assetRevision = 0 } = $props()
  let items=$state([]), assets=$state([]), selected=$state(''), imported=$state(null), template=$state('')
  let assetId=$state(''), status=$state(''), busy=$state(false), result=$state(null)
  const current=$derived(items.find(item=>item.id===selected))
  const writable=$derived(webSession?.user.role==='administrator'||webSession?.user.role==='operator')
  const api=$derived(webApi(webSession))
  async function work(action){busy=true;try{await action()}catch(error){status=error.message}finally{busy=false}}
  async function refresh(){items=(await api('manifests')).items;assets=(await api('manifest-assets')).items}
  $effect(()=>{void assetRevision;if(webSession) work(refresh)})
  async function upload(event,isTemplate){await work(async()=>{const file=event.target.files[0];if(!file)return;const text=await file.text();if(isTemplate)template=text;else {const parsed=JSON.parse(text);if(parsed.manifest&&typeof parsed.template==='string'){imported=parsed.manifest;template=parsed.template}else imported=parsed;}status='선언을 읽었습니다. 저장하기 전에 검증하세요'})}
  async function validate(){await work(async()=>{const response=await api('manifests/validate','POST',{manifest:imported,template});status=response.valid?'엔진 매니페스트 검증 통과':response.errors.join('; ')})}
  async function save(){await work(async()=>{const item=await api('manifests','POST',{manifest:imported,template});await refresh();selected=item.id;status='외부 매니페스트를 저장했습니다. 저장한 매니페스트는 바뀌지 않습니다'})}
  function download(){const blob=new Blob([JSON.stringify({manifest:current.manifest,template:current.template},null,2)],{type:'application/json'});const url=URL.createObjectURL(blob);const a=document.createElement('a');a.href=url;a.download='chain-manifest.json';a.click();URL.revokeObjectURL(url)}
  async function setup(){await work(async()=>{result=await api(`manifests/${selected}/setup`,'POST',{assetId});status='엔진 설정을 초기화했습니다. 노드는 멈춘 상태로 남습니다'})}
</script>

<section class="panel manifests" aria-label="체인 매니페스트 관리">
  <h2>체인 플러그인과 매니페스트</h2>
  <p>내장 플러그인을 쓰거나 기존 엔진 계열의 선언을 가져옵니다. 설정은 격리된 로컬 노드 네 개를 준비하고 데이터베이스를 초기화합니다.</p>
  {#if webSession}
    <label>저장된 체인 매니페스트 <select name="selected" aria-label="저장된 체인 매니페스트" bind:value={selected} onchange={()=>result=null}>
      <option value="">플러그인이나 가져온 매니페스트 선택</option>
      {#each items as item}<option value={item.id}>{item.manifest.id} · {item.source}</option>{/each}
    </select></label>
    {#if current}
      <dl><dt>체인</dt><dd>{current.manifest.id}</dd><dt>합의 계열</dt><dd>{current.manifest.consensus_family}</dd><dt>방언</dt><dd>{current.manifest.dialect}</dd><dt>Chain ID</dt><dd>{current.manifest.chain_id}</dd><dt>프로토콜</dt><dd>{current.manifest.protocol||current.manifest.id}</dd></dl>
      <button onclick={download}>매니페스트 묶음 내보내기</button>
      <label>검증된 바이너리 자료 <select name="asset-id" aria-label="검증된 바이너리 자료" bind:value={assetId}><option value="">호환 바이너리 선택</option>{#each assets.filter(a=>a.chain===(current.manifest.protocol||current.manifest.id)) as asset}<option value={asset.id}>{asset.id} · {asset.sha256.slice(0,12)}</option>{/each}</select></label>
      {#if writable}<button disabled={busy||!assetId} onclick={setup}>매니페스트 설정 초기화</button>{/if}
    {/if}
    {#if writable}
      <fieldset><legend>외부 매니페스트 가져오기</legend>
        <label>매니페스트 JSON 파일 <input name="manifest-manager-input-1" aria-label="매니페스트 JSON 파일" type="file" accept=".json,application/json" onchange={e=>upload(e,false)} /></label>
        <label>genesis 템플릿 JSON 파일 <input name="manifest-manager-input-2" aria-label="genesis 템플릿 JSON 파일" type="file" accept=".json,application/json" onchange={e=>upload(e,true)} /></label>
        {#if imported}<p>{imported.id} · {imported.consensus_family} · {imported.dialect}</p>{/if}
        <button disabled={busy||!imported} onclick={validate}>매니페스트 검증</button>
        <button disabled={busy||!imported} onclick={save}>외부 매니페스트 저장</button>
      </fieldset>
    {/if}
    {#if result}<p data-testid="manifest-setup">설정 {result.id} · {result.state.chain} · 초기화됨</p>{/if}
  {/if}
  <p role="status" aria-label="매니페스트 상태">{busy?'처리 중…':status}</p>
</section>

<style>p{color:var(--text-muted)}label{margin:var(--space-3) 0}button{margin:var(--space-1) var(--space-1) var(--space-1) 0}dl{display:grid;grid-template-columns:7rem 1fr;gap:var(--space-1)}dt{color:var(--text-muted)}dd{margin:0}</style>
