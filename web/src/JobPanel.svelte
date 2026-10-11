<script>
  import { responseError } from './api-error.mjs'
  import { allItems } from './pages.mjs'
  import { onMount } from 'svelte'
  import Field from './DSLField.svelte'
  import JobCard from './JobCard.svelte'
  import { go, follow } from './nav.mjs'
  import { selection, selectWorkspace, takeControl } from './selection.svelte.js'
  import { caseAttaches, caseChain, declaredBinaries, keyFileAccounts } from './dsl-form.js'
  let { webSession = undefined, testOnly = false, catalogRevision = 0 } = $props()
  let workspaces=$state([]), manifests=$state([]), assets=$state([]), jobs=$state([]), documents=$state([]), networks=$state([])
  let workspaceId=$state(selection.workspaceId), manifestId=$state(''), assetId=$state(''), serverRef=$state('')
  let operation=$state('chain.setup'), validators=$state(4), nodeId=$state('node1'), retention=$state('retain')
  let caseIds=$state([]), observedNetwork=$state(null), presetId=$state(''), runDefinitionId=$state(''), runDefinitionName=$state('')
  let configContract=$state(null), configOverrides=$state({}), replacementAssetId=$state(''), configAssetId=$state(''), configAssets=$state([]), configVersion=0
  const observationNames={running:'가동 중',stopped:'정지 확인',unrecorded_running:'기록되지 않은 실행 발견',missing:'기록된 프로세스 없음',ownership_mismatch:'실행 기록 불일치',unknown:'확인 필요'}
  const observationReasons={binary_launch_unavailable:'선택 노드의 실행 파일을 실행할 수 있는지 확인되지 않음; 재실행 전에 권한과 파일을 확인해야 함',binary_binding_unavailable:'선택 노드의 별도 실행 파일에 검토된 등록 근거가 없음',ledger_mismatch:'실행 기록과 프로세스 기록의 PID 또는 소유 범위가 다름',probe_unavailable:'관측할 수 없음',discovery_unavailable:'잔류 프로세스를 확인할 수 없음',no_matching_process:'일치하는 프로세스 없음',unrecorded_pid_found:'실제 PID를 발견했지만 기록에는 반영하지 않음',unrecorded_argv_mismatch:'같은 노드 경로를 사용하는 다른 실행 발견',no_recorded_pid:'기록된 PID 없음',argv_mismatch:'실행 인자가 기록과 다름',recorded_pid_absent:'기록된 프로세스가 없음'}
  let conflicts=$state([]), conflictsChecked=$state(false), conflictsAt=$state('')
  let plan=$state(null), busy=$state(false), error=$state(''), loaded=$state(false)
  let catalogLoading=$state(false), catalogMessage=$state(''), loadVersion=0
  let operations=$state([])
  const offered=$derived(operations.filter(o=>!testOnly||o.scope==='test'))
  $effect(()=>{if(offered.length&&!offered.some(o=>o.id===operation))operation=offered[0].id})
  const actor=$derived(webSession?.user)
  const canEdit=$derived(actor && actor.role!=='viewer')
  const workspace=$derived(workspaces.find(w=>w.id===workspaceId))
  const manifest=$derived(manifests.find(m=>m.id===manifestId))
  const nodeControl=$derived(operation.startsWith('node.'))
  const testRun=$derived(operation==='test.run')
  const testAttach=$derived(operation==='test.attach')
  const caseJob=$derived(testRun||testAttach)
  const configSwap=$derived(operation==='node.swap')
  const logCollection=$derived(operation==='network.monitor')
  const configSchema=$derived(configContract?.['$defs']?.envSpec?.properties?.config?.additionalProperties)
  const presets=$derived(documents.filter(d=>d.kind==='chain-preset'&&d.content.chain===(manifest?.manifest.protocol||manifest?.manifest.id)))
  const preset=$derived(presets.find(d=>d.id===presetId))
  let binaryAssets=$state({}), accountBindings=$state({}), accountKeys=$state([])
  const cases=$derived(documents.filter(d=>d.kind==='case'&&caseChain(d,documents)===(manifest?.manifest.protocol||manifest?.manifest.id)&&caseAttaches(d,documents)===testAttach))
  // A saved run definition is usable when every case revision it pins is one of the listed cases.
  const runDefinitions=$derived(documents.filter(d=>d.kind==='run-definition'&&(d.caseRefs??[]).every(r=>cases.some(c=>c.id===r.id&&c.revision===r.revision))))
  const runDefinition=$derived(runDefinitions.find(d=>d.id===runDefinitionId))
  function applyRunDefinition(){caseIds=runDefinition?runDefinition.caseRefs.map(r=>r.id):caseIds;changed()}
  async function saveRunDefinition(){const chosen=caseIds.map(id=>cases.find(c=>c.id===id)).filter(Boolean);const def=await api('documents','POST',{kind:'run-definition',name:runDefinitionName,contractVersion:'2',content:{cases:chosen.map(c=>c.content.id)},assetRefs:[],caseRefs:chosen.map(c=>({id:c.id,revision:c.revision}))});await load();runDefinitionId=def.id;runDefinitionName=''}
  const keyedAccounts=$derived(testAttach?[...new Set(cases.filter(c=>caseIds.includes(c.id)).flatMap(c=>keyFileAccounts(c,documents)))].sort():[])
  $effect(()=>{if(testAttach&&webSession)allItems(api,'credentials').then(items=>{accountKeys=items.filter(c=>c.kind==='account-key'&&!c.revoked)}).catch(()=>{})})
  const namedBinaries=$derived(testRun?[...new Map(cases.filter(c=>caseIds.includes(c.id)).flatMap(c=>declaredBinaries(c,documents)).map(b=>[b.name,b])).values()]:[])
  // Attached networks belong to someone else: the server refuses composition,
  // control and cleanup, and the form says so before any plan is requested.
  const attached=$derived(documents.some(d=>d.kind==='workspace-config'&&d.content?.execution?.chain==='attach'&&workspace?.documents.some(r=>r.id===d.id&&r.revision===d.revision)))
  const servers=$derived(documents.find(d=>d.kind==='server-set'&&workspace?.documents.some(r=>r.id===d.id&&r.revision===d.revision))?.content.pool.hosts??[])
  const nodes=$derived(networks.find(n=>n.workspaceId===workspaceId)?.nodes??[])
  const observedNodes=$derived(observedNetwork?.workspaceId===workspaceId?observedNetwork.nodes:null)
  const controlNodes=$derived((observedNodes??nodes).filter(n=>(!configSwap||observedNodes)&&n.supportedControls.includes(operation)&&(operation!=='node.reset'||(['en','pn'].includes(n.role)&&(n.pid>0||observedNodes&&n.state==='stopped')))))
  const currentBinarySHA=$derived(controlNodes.find(n=>n.id===nodeId)?.binarySHA256??assets.find(a=>a.id===assetId)?.sha256)
  const replacementAssets=$derived(assets.filter(a=>a.chain===(manifest?.manifest.protocol||manifest?.manifest.id)&&a.sha256!==currentBinarySHA))
  const active=j=>['accepted','running','cancelling'].includes(j.state)
  async function api(path,method='GET',data,key) {
    const headers={'Content-Type':'application/json','X-CSRF-Token':webSession?.csrfToken??''}
    if(key)headers['Idempotency-Key']=key
    const r=await fetch('/api/v1/'+path,{method,headers,body:data===undefined?undefined:JSON.stringify(data)})
    if(!r.ok)throw await responseError(r)
    return r.json()
  }

  $effect(()=>{
    const chain=manifest?.manifest.protocol||manifest?.manifest.id,version=++configVersion
    configContract=null;configOverrides={};replacementAssetId='';configAssetId='';changed()
    allItems(api,'assets').then(items=>{configAssets=items.filter(a=>a.kind==='configuration')}).catch(()=>{configAssets=[]})
    if(chain)api('contracts/chain-preset?chain='+encodeURIComponent(chain)).then(v=>{if(version===configVersion)configContract=v}).catch(e=>{if(version===configVersion)error=e.message})
  })
  async function work(fn){busy=true;error='';try{await fn()}catch(e){error=e.message}finally{busy=false}}
  async function refresh(){const values=await Promise.all([api('jobs'),api('networks')]);[jobs,networks]=values.map(v=>v.items);if(plan)await checkConflicts()}
  async function load(){
    const version=++loadVersion,userId=webSession?.user.id
    catalogLoading=true;changed()
    try {
      const values=await Promise.all([allItems(api,'workspaces').then(items=>({items})),api('manifests'),api('manifest-assets'),api('jobs'),allItems(api,'documents').then(items=>({items})),api('networks'),api('contracts/jobs')])
      if(version!==loadVersion||webSession?.user.id!==userId)return
      operations=values[6].operations
      const missing=new Map()
      for(const w of values[0].items)for(const ref of w.documents){
        if(!values[4].items.some(d=>d.id===ref.id&&d.revision===ref.revision))missing.set(`${ref.id}:${ref.revision}`,ref)
      }
      const pinned=await Promise.all([...missing.values()].map(ref=>api(`documents/${encodeURIComponent(ref.id)}?revision=${ref.revision}`)))
      if(version!==loadVersion||webSession?.user.id!==userId)return
      values[4].items=[...values[4].items,...pinned];
      [workspaces,manifests,assets,jobs,documents,networks]=values.map(v=>v.items);loaded=true
      catalogMessage='저장된 구성과 실행 자료를 불러왔습니다. 새 계획을 확인하세요.'
      if(canEdit)credentialLabels=Object.fromEntries((await allItems(api,'credentials')).map(c=>[c.id,c.label+(c.revoked?' (폐기됨)':'')]))
      applyControl(takeControl())
    } catch(e){if(version===loadVersion)throw e}
    finally{if(version===loadVersion)catalogLoading=false}
  }
  $effect(()=>{void catalogRevision;if(webSession)work(load)})
  onMount(()=>{const poll=setInterval(()=>{if(actor&&loaded)refresh().catch(e=>{error=e.message})},3000);return()=>clearInterval(poll)})
  async function observeNodes(){const result=await api(`networks/${workspaceId}/observations`);observedNetwork=result;networks=networks.map(n=>n.id===result.id?result:n)}
  // A control chosen on a node page fills the form; the binary is the one the
  // node runs when observation tied it to a registered file.
  function applyControl(c){
    if(!c)return
    workspaceId=c.workspaceId;operation=c.operation;nodeId=c.nodeId
    const node=networks.find(n=>n.workspaceId===c.workspaceId)?.nodes.find(n=>n.id===c.nodeId)
    const asset=assets.find(a=>a.id===node?.binaryAssetId)
    if(asset){assetId=asset.id;manifestId=manifests.find(m=>(m.manifest.protocol||m.manifest.id)===asset.chain)?.id??''}
    catalogMessage=`노드 페이지에서 고른 ${c.nodeId} 제어입니다. 남은 항목을 고르고 실행 계획을 확인하세요.`
  }
  function changed(){plan=null;conflicts=[];conflictsChecked=false;conflictsAt=''}
  async function checkConflicts(){const id=plan?.id;if(!id)return [];conflictsChecked=false;const result=await api(`plans/${id}/conflicts`);if(plan?.id===id){conflicts=result.items;conflictsAt=result.observedAt;conflictsChecked=true}return result.items}
  // The input a reviewed plan came from, to show its bindings and documents.
  let planInput=$state(null), missingAccess=$state([]), credentialLabels=$state({})
  async function createPlan(input){planInput=input;return api('plans','POST',input)}
  // A server without my SSH credential is named, and nothing can run until it is bound.
  async function prepare(){missingAccess=[];try{await preparePlan()}catch(e){if(e.details?.missingAccess?.length){missingAccess=e.details.missingAccess;throw new Error('계획을 만들 수 없습니다. 아래 서버에 내 SSH 자격증명을 먼저 연결하세요.')}throw e}}
  async function preparePlan(){changed();const bindings=await api(`workspaces/${workspaceId}/credential-bindings`);if(logCollection){const spread=serverRef==='*all*';plan=await createPlan({workspaceId,operation,documentRefs:workspace.documents,assetRefs:[],credentialBindings:spread?bindings:bindings[serverRef]?{[serverRef]:bindings[serverRef]}:{},nodeIds:[],retention:'retain',arguments:spread?{allServers:true}:{serverRef}});await checkConflicts();return}const selectedCases=caseIds.map(id=>cases.find(c=>c.id===id)).filter(Boolean);const dependencies=caseJob?selectedCases.flatMap(c=>c.assetRefs):!nodeControl&&preset?preset.assetRefs:[];const spread=serverRef==='*all*';plan=await createPlan({workspaceId,operation,documentRefs:workspace.documents,assetRefs:[...new Set([assetId,...dependencies,...namedBinaries.map(b=>binaryAssets[b.name]).filter(Boolean),...(configSwap&&replacementAssetId?[replacementAssetId]:[]),...(configSwap&&configAssetId?[configAssetId]:[])])],credentialBindings:spread?bindings:bindings[serverRef]?{[serverRef]:bindings[serverRef]}:{},...(keyedAccounts.length?{accountBindings:Object.fromEntries(keyedAccounts.map(l=>[l,accountBindings[l]??'']))}:{}),nodeIds:nodeControl?[nodeId]:[],retention:nodeControl||testAttach?'retain':retention,arguments:{manifestId,assetId,...(spread?{allServers:true}:{serverRef}),...(caseJob?{...(runDefinition?{runDefinitionRef:{id:runDefinition.id,revision:runDefinition.revision}}:{caseRefs:selectedCases.map(c=>({id:c.id,revision:c.revision}))}),...(namedBinaries.length?{binaryAssets:Object.fromEntries(namedBinaries.map(b=>[b.name,binaryAssets[b.name]??'']))}:{})}:nodeControl?(configSwap?{...(configAssetId?{configAssetId}:Object.keys(configOverrides).length?{configOverrides}:{}),...(replacementAssetId?{replacementAssetId}:{})}:{}):preset?{chainPresetRef:{id:preset.id,revision:preset.revision}}:{validators})}});await checkConflicts()}
  // A started job opens its own page, which follows it live.
  async function start(){if((await checkConflicts()).length)return;let job;try{job=await api('jobs','POST',{planId:plan.id},plan.id)}catch(e){await checkConflicts();throw e}changed();await refresh();go('/jobs/'+job.id)}
</script>

<section aria-label="서버 실행 작업" class="jobs">
  <div class="heading"><div><h2>실행 작업</h2><p>계획을 검토하고 서버에 작업을 맡깁니다. 브라우저 연결이 끊겨도 진행 상태는 보관됩니다.</p></div>{#if actor}<button disabled={busy} onclick={()=>work(load)}>새로고침</button>{/if}</div>
  {#if actor}
    {#if canEdit}
      <fieldset disabled={busy||catalogLoading} onchange={changed}><legend>새 작업</legend>
        <p>로컬 또는 SSH 서버에 구성합니다. SSH 대상은 서버 설정에서 연결한 내 자격증명을 사용하며, 장비·경로·바이너리를 실행 전에 검증합니다. 테스트는 선택한 케이스의 구성 선언으로 실행하고 실제 세션 판정을 보관합니다.</p>
        <div class="inputs">
          <label>Workspace<select aria-label="작업 Workspace" bind:value={workspaceId} onchange={()=>selectWorkspace(workspaceId)}><option value="">선택</option>{#each workspaces as w}<option value={w.id}>{w.name} · r{w.revision}</option>{/each}</select></label>
          {#if !logCollection}<label>매니페스트<select aria-label="작업 매니페스트" bind:value={manifestId} onchange={()=>{assetId='';caseIds=[];presetId='';changed()}}><option value="">선택</option>{#each manifests as m}<option value={m.id}>{m.manifest.id}</option>{/each}</select></label>
          <label>바이너리<select aria-label="작업 바이너리" bind:value={assetId}><option value="">등록 파일 선택 · 계획에서 검증</option>{#each assets.filter(a=>a.chain===(manifest?.manifest.protocol||manifest?.manifest.id)) as a}<option value={a.id}>{a.id} · {a.sha256.slice(0,12)}</option>{/each}</select></label>{/if}
          <label>작업<select aria-label="작업 종류" bind:value={operation}>{#each offered as o}<option value={o.id} title={o.description}>{o.label}</option>{/each}</select></label>
          <label>서버<select aria-label="작업 서버 이름" bind:value={serverRef}><option value="">서버 선택</option>{#each servers as host}<option value={typeof host==='string'?host:host.name||host.addr}>{typeof host==='string'?host:host.name||host.addr}</option>{/each}{#if operation!=='test.run'&&servers.length>1}<option value="*all*">서버 세트 전체</option>{/if}</select></label>{#if serverRef==='*all*'}<p data-testid="spread-explanation">구성 작업은 노드를 서버 세트의 모든 서버에 순서대로 나누어 배치합니다. 이미 여러 서버에 나뉜 네트워크의 노드 제어·attach 테스트·로그 수집도 이 선택으로 하며, 각 노드는 자기 서버에서 확인하고 제어합니다. 원격 서버마다 내 SSH 자격증명이 필요합니다.</p>{/if}
          {#if !testRun&&!nodeControl&&!logCollection}<label>저장한 체인 구성<select aria-label="작업 체인 구성" bind:value={presetId}><option value="">기본 구성 · 생산자 수 지정</option>{#each presets as p}<option value={p.id}>{p.name} · r{p.revision}</option>{/each}</select></label>{/if}
          {#if !testRun&&!nodeControl&&!preset&&!logCollection}<label>생산자 수<input aria-label="작업 생산자 수" type="number" min="1" max="128" bind:value={validators} /></label>{/if}
          {#if nodeControl}<label>소유 노드<select aria-label="작업 노드" bind:value={nodeId}><option value="">노드 선택</option>{#each observedNodes??nodes as n}{@const usable=controlNodes.some(c=>c.id===n.id)}<option value={n.id} disabled={!usable}>{n.id} · {n.role} · {n.state}{usable?'':' · 불가: '+(n.controlLimits?.[operation]??'이 노드에는 이 제어를 쓸 수 없습니다')}</option>{/each}</select></label>{#if nodeId&&!controlNodes.some(c=>c.id===nodeId)}<p class="error" data-testid="node-control-limit">{(observedNodes??nodes).find(n=>n.id===nodeId)?.controlLimits?.[operation]??'선택한 노드에는 이 제어를 쓸 수 없습니다.'}</p>{/if}{:else if !logCollection&&!testAttach}<label>종료 후 처리<select aria-label="작업 종료 후 처리" bind:value={retention}><option value="retain">노드와 자료 보존</option><option value="cleanup">소유 노드 정리</option></select></label>{/if}
        </div>{#if configSwap}<section aria-label="노드 설정 변경"><label>교체할 바이너리<select aria-label="노드 교체 바이너리" bind:value={replacementAssetId}><option value="">현재 바이너리 유지</option>{#each replacementAssets as a}<option value={a.id}>{a.id} · {a.sha256.slice(0,12)}</option>{/each}</select></label>{#if currentBinarySHA}<p>현재 실행 파일 SHA-256: <code>{currentBinarySHA}</code></p>{/if}<p>엔진이 제공하는 설정과 값으로 선택한 노드의 설정 파일과 실행 인자를 함께 변경합니다. genesis·노드 데이터와 다른 노드는 보존합니다. 먼저 노드 상태를 확인하세요. 별도 설정 파일이 지정된 노드는 이 방식으로 변경할 수 없습니다. 등록된 바이너리를 선택하면 대상 장비와 기존 실행 옵션을 검증한 뒤 선택 노드의 실행 파일만 교체합니다.</p><label>등록된 설정 파일<select aria-label="노드 설정 파일" bind:value={configAssetId} onchange={changed}><option value="">파일 대신 아래 선택지 사용</option>{#each configAssets as a}<option value={a.id}>{a.name} · {a.checksum.slice(0,12)}</option>{/each}</select></label>{#if configAssetId}<p>선택한 등록 파일의 값(엔진이 렌더하는 syncMode·httpHost·metricsHost)을 계획에서 검증하고 체크섬으로 고정해 적용합니다.</p>{:else if configSchema}<Field schema={configSchema} root={configContract} value={configOverrides} path="/node-config" label="변경할 설정" onchange={v=>{configOverrides=v;changed()}} />{:else}<p>설정 계약을 불러오는 중…</p>{/if}</section>{/if}{#if operation==='node.restart'}<p>선택한 소유 노드만 기록된 바이너리와 실행 인자로 재실행합니다. 설정·genesis·노드 데이터와 다른 노드는 보존합니다. 실행 전에 파일과 실제 프로세스 상태를 다시 확인합니다.</p>{/if}{#if operation==='node.reset'}<p>선택한 비생산자의 기존 노드 데이터를 제거하고 기록된 genesis로 초기화합니다. 노드는 정지 상태로 남습니다. 생산자는 초기화할 수 없습니다. 비생산자는 실제 실행 또는 정지 상태와 입력을 확인한 경우에만 초기화하며, PID가 없는 정지 노드는 먼저 노드 상태를 확인하세요.</p>{/if}{#if !testRun&&!nodeControl&&preset}<p>체인 구성 {preset.name} · r{preset.revision}의 노드 배치와 옵션을 적용합니다. 실제 병합 결과는 실행 계획에서 확인하세요.</p>{/if}{#if caseJob}<div class="cases"><h3>실행할 저장 케이스</h3>{#if testAttach}<p data-testid="attach-explanation">이 Workspace에 기록된 네트워크에 연결해 케이스를 읽기 전용으로 실행합니다. 케이스가 적은 rpc·keysDir·provides 대신 기록된 노드 주소·키·기능을 씁니다. 노드를 시작·정지·제거하지 않으며 노드와 자료는 보존합니다. attach를 선언한 케이스만 보입니다.</p>{/if}{#if !testAttach}<div class="run-definitions"><label>저장된 실행 목록<select aria-label="실행 목록" bind:value={runDefinitionId} onchange={applyRunDefinition}><option value="">직접 선택</option>{#each runDefinitions as d}<option value={d.id}>{d.name} · r{d.revision} · 케이스 {d.caseRefs.length}개</option>{/each}</select></label>{#if runDefinition}<p>실행 목록 {runDefinition.name} r{runDefinition.revision}의 순서와 케이스 리비전으로 실행합니다.</p>{:else}<label>선택 순서를 실행 목록으로 저장<input aria-label="새 실행 목록 이름" bind:value={runDefinitionName} placeholder="실행 목록 이름" /></label><button disabled={busy||!runDefinitionName||!caseIds.length} onclick={()=>work(saveRunDefinition)}>실행 목록 저장</button>{/if}</div>{/if}<p>선택한 리비전의 내용을 계획에 고정합니다. 노드 배치는 케이스 선언을 따릅니다. 케이스는 선택한 순서대로 실행하므로 노드를 멈추거나 바꾸는 케이스는 마지막에 고르세요.</p>{#each cases as c}<label class="case"><input type="checkbox" aria-label={'실행 케이스 '+c.id} checked={caseIds.includes(c.id)} onchange={e=>{caseIds=e.target.checked?[...caseIds,c.id]:caseIds.filter(id=>id!==c.id);runDefinitionId='';changed()}} />{#if caseIds.includes(c.id)}<span class="order" data-testid={'case-order-'+c.id}>{caseIds.indexOf(c.id)+1}</span>{/if}{c.name} · r{c.revision}</label>{:else}<p>이 체인의 저장 케이스가 없습니다. 테스트 정의를 저장한 뒤 새로고침하세요.</p>{/each}{#if keyedAccounts.length}<h3>key file 계정</h3><p>케이스가 서버의 key file로 읽는 계정입니다. 설정에 저장한 내 테스트 계정 키를 고르세요. 실행하는 동안만 비공개 파일로 쓰고 끝나면 지웁니다.</p>{#each keyedAccounts as label}<label>{label}<select aria-label={'테스트 계정 키 '+label} value={accountBindings[label]??''} onchange={e=>{accountBindings={...accountBindings,[label]:e.target.value};changed()}}><option value="">내 계정 키 선택</option>{#each accountKeys as k}<option value={k.id}>{k.label}</option>{/each}</select></label>{/each}{/if}{#if namedBinaries.length}<h3>케이스가 선언한 추가 바이너리</h3><p>각 이름에 그 이름이 실행하는 체인의 등록 파일을 고릅니다. 계획에서 검증하고 체크섬으로 고정합니다.</p>{#each namedBinaries as b}<label>{b.name} ({b.chain})<select aria-label={'테스트 바이너리 '+b.name} value={binaryAssets[b.name]??''} onchange={e=>{binaryAssets={...binaryAssets,[b.name]:e.target.value};changed()}}><option value="">등록 파일 선택</option>{#each assets.filter(a=>a.chain===b.chain) as a}<option value={a.id}>{a.id} · {a.sha256.slice(0,12)}</option>{/each}</select></label>{/each}{/if}</div>{/if}{#if attached}<p class="error" role="status" data-testid="attached-network">이 Workspace는 attach 네트워크입니다. Web UI는 attach 노드를 구성·제어·정리하지 않으며 서버도 해당 요청을 거절합니다.</p>{/if}<button disabled={attached||!workspace||(!logCollection&&(!manifest||!assetId))||!serverRef||(nodeControl&&!controlNodes.some(n=>n.id===nodeId))||(caseJob&&(!cases.some(c=>caseIds.includes(c.id))||namedBinaries.some(b=>!binaryAssets[b.name])||keyedAccounts.some(l=>!accountBindings[l])))||(configSwap&&((!replacementAssetId&&!configAssetId&&!Object.keys(configOverrides).length)||(!configAssetId&&Object.keys(configOverrides).length&&!configSchema)))||busy} onclick={()=>work(prepare)}>실행 계획 확인</button>
      </fieldset>
      <button disabled={busy||!workspaceId||!networks.some(n=>n.workspaceId===workspaceId)} onclick={()=>work(observeNodes)}>노드 상태 확인</button>
      {#if observedNetwork?.workspaceId===workspaceId}<section class="observations" aria-label="노드 실제 관측"><h3>현재 프로세스 관측</h3><p>기록된 PID와 실제 실행 인자를 확인합니다. 이 조회는 노드나 실행 기록을 변경하지 않습니다.</p><ul>{#each observedNetwork.nodes as n}<li><strong><a href={`/nodes/${observedNetwork.id}/${n.id}`} onclick={e=>follow(e,`/nodes/${observedNetwork.id}/${n.id}`)}>{n.id}</a> · {observationNames[n.state]??n.state}</strong><p>기록 PID {n.pid} · 관측 PID {n.observedPid||'확인 안 됨'} · {new Date(n.observedAt).toLocaleString()}</p>{#if n.binaryAssetId}<p>등록 바이너리 <code>{n.binaryAssetId}</code> · SHA-256 <code>{n.binarySHA256}</code></p>{/if}{#if n.observationReason}<p>{observationReasons[n.observationReason]??'관측할 수 없음'}</p>{/if}</li>{/each}</ul></section>{/if}
      {#if missingAccess.length}<section class="missing" aria-label="부족한 접근" data-testid="missing-access"><h3>부족한 접근</h3><p>다음 서버는 SSH로 접속하는데 내 자격증명이 연결되어 있지 않습니다. 위의 공유 배포 구성에서 workspace를 고르고 내 자격증명을 연결한 뒤 다시 확인하세요. 자격증명 등록은 <a href="/settings" onclick={e=>follow(e,'/settings')}>설정</a>에서 합니다.</p><ul>{#each missingAccess as m}<li><code>{m}</code></li>{/each}</ul></section>{/if}
      {#if plan}<article class="plan" aria-label="실행 계획"><h3>실행 전 검토</h3>
        <dl class="review" data-testid="plan-review">
          <dt>필요한 접근</dt><dd data-testid="plan-access">{plan.requiredAccess.join(', ')||'없음'}</dd>
          <dt>내 자격증명 연결</dt><dd data-testid="plan-bindings">{Object.entries(planInput?.credentialBindings??{}).map(([server,id])=>`${server} → ${credentialLabels[id]??id}`).join(', ')||'로컬 대상이라 필요 없음'}</dd>
          <dt>고정한 문서</dt><dd data-testid="plan-documents">{plan.documentRefs.map(r=>{const d=documents.find(x=>x.id===r.id&&x.revision===r.revision);return `${d?.name??r.id} (${d?.kind??'문서'}) r${r.revision}`}).join(', ')}</dd>
          <dt>바이너리 surface</dt><dd data-testid="plan-surfaces">{#each plan.chainSurfaces??[] as cs}<span class="surface">{cs.chainId} · 자료 {cs.assetId} · 계약 {cs.contractVersion} · 지문 <code>{cs.fingerprint.slice(0,12)}</code></span>{:else}고정한 surface 없음{/each}</dd>
          <dt>바뀌는 내용</dt><dd><ul class="changes">{#each plan.changes as change}<li><pre>{change}</pre></li>{/each}</ul></dd>
        </dl><p>순서: {plan.phases.join(' → ')}</p><ul>{#each plan.resources as resource}<li><code>{resource}</code></li>{/each}</ul><p>유효 기한: {new Date(plan.expiresAt).toLocaleString()}</p><section class="conflicts" aria-label="자원 충돌" aria-live="polite"><h4>자원 사용 확인</h4>{#if !conflictsChecked}<p>현재 충돌 상태를 확인해 주세요.</p>{:else if conflicts.length}<p class="error">다른 작업이 이 자원을 사용하거나 보존하고 있습니다. 정리가 확인될 때까지 실행할 수 없습니다.</p><ul>{#each conflicts as conflict}<li><strong>{conflict.operation} · {conflict.state}</strong><p>작업 <code>{conflict.jobId}</code><br />Workspace {workspaces.find(w=>w.id===conflict.workspaceId)?.name??conflict.workspaceId} · <code>{conflict.workspaceId}</code><br />실행 주체 <code>{conflict.actorId}</code> · 자원 처리 {conflict.nodeDisposition}</p><ul>{#each conflict.resources as resource}<li><code>{resource.hostIdentity} · {resource.executable?'실행 중인 노드 바이너리 '+resource.executable:resource.dataPath??'경로 없음'}</code>{#if resource.ports?.length}<p>포트: {resource.ports.join(', ')}</p>{/if}</li>{/each}</ul></li>{/each}</ul>{:else}<p>확인된 자원 충돌이 없습니다. 실행 직전에 다시 검사합니다.</p>{/if}{#if conflictsAt}<p>확인 시각: {new Date(conflictsAt).toLocaleString()}</p>{/if}<button disabled={busy} onclick={()=>work(checkConflicts)}>충돌 다시 확인</button></section><button disabled={busy||!conflictsChecked||conflicts.length>0} onclick={()=>work(start)}>검토한 계획 실행</button></article>{/if}
    {/if}
    <p role="status" aria-label="실행 자료 상태">{catalogLoading?'저장된 구성을 불러오는 중…':catalogMessage}</p>
    <div class="records" aria-live="polite">{#each jobs as job (job.id)}<JobCard {job} {webSession} link onchanged={()=>work(refresh)} />{:else}<p>보관된 작업이 없습니다.</p>{/each}</div>
  {/if}
  {#if error}<p role="alert" class="error">{error}</p>{/if}
</section>

<style>
.observations{margin-top:16px;padding:16px;border:1px solid #344154;border-radius:8px}.conflicts{border-top:1px solid #344154;margin-top:16px;padding-top:12px}.changes pre{margin:0;font:inherit;white-space:pre-wrap}.cases{margin-top:20px}.case{display:flex;flex-direction:row;align-items:center;margin:8px 0}.case input{min-height:20px;width:20px}.jobs{margin-top:24px;padding:24px;border:1px solid #344154;border-radius:12px;background:#151b24}.heading{display:flex;justify-content:space-between;gap:16px}.heading p{color:#a7b5c7}h2{margin:0 0 8px}h3{margin-top:0}fieldset{border:1px solid #344154;margin-top:20px;padding:16px;border-radius:8px}.inputs{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:16px}label{display:flex;flex-direction:column;gap:8px}input,select,button{font:inherit;color:#e3ebf5;background:#202b3b;border:1px solid #47576d;border-radius:6px;min-height:40px;padding:8px 12px;max-width:100%}button{cursor:pointer;margin-top:16px}button:disabled{opacity:.45;cursor:default}button:focus-visible,input:focus-visible,select:focus-visible{outline:2px solid #67d8c5;outline-offset:3px}.missing{margin-top:16px;padding:16px;border:1px solid #fca5a5;border-radius:8px}.review{display:grid;grid-template-columns:10rem minmax(0,1fr);gap:8px 16px}.review dt{color:#a8b5c8}.review dd{margin:0}.surface{display:block}a{color:#93c5fd}.plan{margin-top:16px;padding:16px;border:1px solid #344154;border-radius:8px}.error{color:#f9abae}code,pre{font-family:ui-monospace,monospace;overflow-wrap:anywhere;white-space:pre-wrap}li{margin-top:8px}p{overflow-wrap:anywhere}
</style>
