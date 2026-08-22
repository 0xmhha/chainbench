# chainbench 기능 정의 문서 (Feature Spec) — F1~F16

> **[정본]** F1~F16 동작 계약(AC).
> 이 문서는 *무엇을 만들어야 하는가* 를 정한다. 설계 제안이 여기와 어긋나면 **제안을 고친다.**

> 지위: 각 기능의 **동작 계약**. 형식: 입력 / 동작 / 출력 / 에러·엣지 / **수용기준(AC, 검증가능)**.
> 근거: `chainbench-requirements-review.md`(요구/결정), `chainbench-design.md`(인터페이스). 각 F는 요구번호·design앵커를 명시.
> AC는 향후 자기테스트로 검증 가능하도록 구체·측정가능하게 기술한다. design §9의 미결은 각 F에서 **확정**한다.

---

## F1. 세션 · 아티팩트 레이아웃  (요구 14,17,28,33,34,35 · design §3.1,§4)
- **입력**: 수행할 테스트/스위트 목록, 커맨드 문자열, 시작 시각(UTC).
- **동작**: `.chainbench/<UTC-YYYYMMDD-HHMMSS>/` 를 생성하고 `keys/`·`environments/`·`tests/` 3축을 둔다. 모든 경로는 `session`이 단독 파생(소유권 단일화). 다른 계층(driver/collector/testrun)은 경로를 직접 만들지 않고 session API로만 접근.
- **출력**: `session.json`(커맨드·시각·테스트 목록/순서·요약), 각 축의 파일(F2·F10 등이 채움).
- **에러·엣지**: 디스크 쓰기 실패 → 세션 시작 실패로 즉시 중단(테스트 무의미). 타임스탬프로 세션 유일; 동시 커맨드는 서로 다른 세션.
- **AC**
  1. 한 커맨드 실행 = 정확히 하나의 `<session>/` 생성.
  2. 서로 다른 체인/구성 테스트 2개 → `environments/`에 env-id 2개, `tests/`에 순번 폴더 2개.
  3. 같은 구성 연속 테스트 2개 → `environments/`에 env-id **1개**, `tests/` 2개가 동일 `env-ref`.
  4. 임의 노드 로그·키·테스트 결과가 session 하위 **결정적 경로**에 존재(경로 규칙 위반 0건).

## F2. 키 레지스트리  (요구 17,30 · design §3.5)
- **입력**: 키 이름(op1/bp1/acctA), 소스(Random | LocalFile<path> | RemoteDownload<server,path>), **`EnsureOpts{NeedBLS, BLS BLSDeriver}`**(validator 키면 BLS/PoP 필요), (remote 실행 시) 업로드 대상 경로. (design §3.5)
- **동작**: 이름→키를 `keys/<name>/`에 일원화 저장. Random=생성, LocalFile=복사, RemoteDownload=ssh 다운로드. **`NeedBLS`면 주입된 `BLSDeriver`(외부 `bootnode` 위임)로 BLS/PoP 생성 — Deriver 없으면 오류**(누출 방지). Random 키를 remote에서 쓰면 약속 경로로 **업로드**(driver.FileProvisioner). 노드키(신원)와 서명키(계정)를 같은 레지스트리에서 이름으로 참조.
- **출력**: `keys/<name>/{private,address,bls?,pop?}`(bls/pop은 NeedBLS일 때); 신규 생성 키는 genesis 등록에 사용(요구 17).
- **에러·엣지**: 이름 충돌 → 오류(덮어쓰기 금지). remote 다운로드 실패 → 명확한 오류(부분 상태 금지). **NeedBLS인데 BLSDeriver/`bootnode` 부재 → 명확 오류**(BLS 없이 진행 금지). private 키는 로그·아티팩트 본문에 노출 금지(경로/주소만).
- **AC**
  1. `Ensure(ctx, "op1", Random, "", {})` 두 번 호출 시 동일 키 반환(멱등), `keys/op1/` 1개.
  2. RemoteDownload한 키의 주소가 remote 원본과 일치.
  3. Random 키로 remote 실행 시 remote 약속 경로에 해당 키 존재.
  4. 서명 시 `keys/`의 이름으로 signer 지정 → 해당 주소로 서명됨.
  5. `Ensure(ctx, validator키, Random, "", {NeedBLS:true, BLS:deriver})` → BLS/PoP 채워짐; deriver 없으면 오류.

## F3. TestSpec DSL 스키마  (요구 3,29,30,32 · design §3.2,§4.3)
- **입력**: JSON 정의서(파일 또는 인라인).
- **동작**: `Parse`가 **필수/옵션** 검증(JSON schema). 필수: `schemaVersion`(F16-O2), `id`, `chain(name + binary|binaries)`, `assertions`. 옵션: 그 외 전부. 값이 multiple이면 `,` 분리, 중첩 접근은 닷경로(`a.b.c`). `applicableChains`로 체인 적용성 선언.
  - **genesis(요구 7 "필수")**: spec에 별도로 적지 않아도 **chain 플러그인의 `GenesisTemplate()`이 기본 genesis를 제공**하므로 genesis는 **항상 존재**. spec의 `chain.genesisOverlay`/`hardforks`는 그 위에 얹는 **선택적 override**. 즉 "genesis 없음" 상태는 불가.
  - **확정(§9): `chain.name` = 이 테스트가 도는 대상 체인. `applicableChains` = 호환 체인 집합(예: `"wbft,stablenet"` 같은 합의계열).** `chain.name`은 `applicableChains`에 **포함**되어야 하고, `applicableChains` 미지정 시 `[chain.name]`으로 간주. 스위트가 대상 체인을 `applicableChains` 내에서 바꿔 재사용(체인-스윕) 가능. 현재 세션 대상 체인이 `applicableChains`에 없으면 **SKIP**.
- **출력**: 파싱된 `Spec`; 검증 실패 시 라인/필드가 명시된 오류.
- **에러·엣지**: 필수 누락/타입 불일치 → 파싱 실패(실행 안 함). 알 수 없는 필드 → 경고 또는 오류(엄격모드 선택).
- **AC**
  1. 필수 필드 누락 정의서 → `Parse` 오류(필드명 포함).
  2. `applicableChains:"wbft"` 테스트를 stablenet 세션에서 실행 → SKIP(수행 안 함), status=skip 기록.
  3. `a.b.c` 닷경로로 중첩값 조회 성공, `x,y,z` multiple이 3원소로 파싱.
  4. 동일 정의서 2회 파싱 → 동일 `Spec`(결정적).

## F4. Endpoint(target) 선택  (요구 4,13,15,16 · design §3.1)
- **입력**: 셀렉터(신원 `node7` / 역할 별칭 `bp1` / 역할+index `{role,index}` / `bp:any`·`en:any` — 신원과 별칭은 같은 노드의 두 표기, [[netmap-design]] §2.5a), 스텝의 `on`, 검증의 `on`/`onEach`, 스펙의 `defaultOn`.
- **동작**: 스텝 `on`=tx 제출 노드, 검증 `on`=조회 노드, `onEach`=다중 노드. 미지정 시 `defaultOn`. 해석기가 셀렉터를 `env.json.nodes[].rpc`로 해석(local=포트상이/remote=동일포트+다른IP).
- **출력**: 해석된 rpc url(들).
- **에러·엣지**: 매칭 노드 없음 → 오류(어느 셀렉터가 실패인지). `bp:any`는 활성 bp 중 택1(결정적: 최소 index).
- **AC**
  1. `on:"bp1"` → bp1의 rpc로 전송/조회.
  2. `on:"en1"`의 `istanbul_isValidator` == false, `on:"bp1"` == true.
  3. `onEach:["bp1","en1"]` 검증이 두 노드 모두 조회.
  4. 없는 셀렉터 → 명확한 오류(테스트 실패, hang 없음).

## F5. wait 프리미티브  (요구 32,37 · design §3.2)
- **입력**: `waitReceipt` / `waitBlocks:N` / `waitEpoch` / `waitSeconds:S` / `waitFork:<name>`; 독립 스텝 또는 스텝/검증의 `waitFor`; timeout(명시 또는 기본).
- **동작**: 조건 충족까지 대기. **모든 wait는 timeout 필수**. 초과 시 hang 없이 즉시 FAIL(또는 ERROR) + 아티팩트. 테스트 전체 timeout 별도.
  - **기본값(확정)**: `waitReceipt=30s`, `waitBlocks`=`N*blockPeriod + 여유`, `waitEpoch`=`epochLength*blockPeriod + 여유`, `waitSeconds`=명시, `waitFork`=해당 블록 도달까지(테스트 timeout 상한).
- **출력**: 충족 시점의 관측값(블록번호/영수증/로그위치)을 steps.json에 기록.
- **에러·엣지**: timeout → 실패 사유("waited X for Y")와 마지막 관측값 기록. 음수/0 N → 검증오류.
- **AC**
  1. 영수증 나오면 `waitReceipt`가 즉시 반환(불필요 대기 없음).
  2. 존재하지 않는 조건 → timeout 후 **정확히** 실패(무한 대기 0건).
  3. `waitFork:croissant`가 fork 블록 도달 시 통과.

## F6. Assert 함수 + provenance  (요구 32,33,34 · design §3.2,§4.4)
- **입력**: `source`(rpc/func/log), 대상(`on`/`onEach`), `assert`(함수명), `expected`, (log) `match`·`waitLog`.
- **동작**: 출처에서 실측값을 얻어 함수로 기대값과 비교. 함수 세트(확정): `Equal, NotEqual, Nil, NotNil, True, False, Contains, NotContains, Greater(OrEqual), Less(OrEqual), Len, Regexp, InDelta(±tol), ElementsMatch, EqualCI(주소 대소문자무시), EqualWith(커스텀), EqualHashAt(크로스노드 hash)`. wei/address/hex/bool **타입 인지** 비교. 각 검증은 **출처 provenance**를 남긴다.
- **출력(assert.json 항목)**: `{id, source, on, assert, expected, actual, pass}` + 출처별: rpc=`method·params·raw`, func=`func·args·return`, log=`logFile·lines·byteOffset·extracted`.
- **에러·엣지**: 출처 접근 실패(rpc 오류/로그 없음) → 검증 실패(사유 기록). 타입 불일치 비교 → 명시적 오류.
- **AC**
  1. `Len`으로 validators==7 검증이 raw 응답과 함께 기록됨.
  2. log 검증이 `logFile`+`lines`+`extracted`를 남겨 그 라인으로 역추적 가능.
  3. `EqualCI`가 대소문자 다른 동일 주소를 pass 처리.
  4. `InDelta`가 reward 누적(±gas) 케이스를 tol 내로 pass.

## F7. fingerprint ↔ preAction  (요구 28 · design §3.1,§3.2)
- **입력(fingerprint)**: **resolved 선언 config**(flag>config>default 적용 후) = `binaries-set + genesis + config + topology + hardforks + placement`(local↔remote 포함, O1). **입력(preAction)**: 런타임 상태 요구(ensureChain/ensureStaker 등).
- **fingerprint 길이(L5)**: 전체 = `hex(sha256)` 64자(env.json `fingerprint`에만 기록). **폴더명 env-id = `"env-"+앞 12hex`(16자)** — 전체 해시를 폴더명으로 쓰면 경로 초과 위험이므로 축약.
- **동작**: 다음 테스트의 fingerprint를 현재 `environments/<env-id>`와 비교 → 일치=재사용(setup skip), 불일치=새 env-id 재구성. **체인 미접촉**(선언값만). 런타임 상태는 preAction이 **idempotent**하게 RPC 확인 후 없으면 수행.
- **출력**: env.json의 `fingerprint`; 재사용/재구성 결정 로그.
- **에러·엣지**: fingerprint는 resolved 값 대상(같은 config파일+다른 flag=다른 env). 런타임 상태를 fingerprint에 **섞지 않음**.
- **AC**
  1. 동일 선언 config 두 테스트 → env 1개(재사용), 두 번째 setup skip.
  2. flag 하나만 다른 두 테스트 → env **2개**(재구성).
  3. `ensureStaker(A)` 두 번 → 두 번째는 이미 등록 확인 후 skip(중복 등록 0).
  4. 런타임 상태 변화(staker 추가)가 fingerprint를 바꾸지 않음(재구성 유발 안 함).

## F8. 크로스노드 · 분기 검증  (요구 4,34 · design §3.2)
- **입력**: `onEach` 노드셋, 검증 종류(bp참여/en싱크/무분기).
- **동작**: 같은 질의를 여러 노드에 보내 비교. **bp참여**=`istanbul_getCommitSignersFromBlock`에 기대 bp가 쿼럼 이상. **en싱크**=en `eth_blockNumber`가 bp 높이 추종(허용차 이내). **무분기**=높이 H에서 전 노드 `eth_getBlockByNumber(H).hash` 동일.
- **출력**: 노드별 관측값 + 비교결과(assert.json). 분기 감지 시 어떤 노드가 어긋났는지.
- **에러·엣지**: 일부 노드 응답없음 → 그 노드 표시 후 실패. 높이 H 미도달 → wait 후 재시도(timeout).
- **AC**
  1. 정상망: 전 노드 동일 hash → 무분기 pass.
  2. 인위적 분기(파티션): **`partition` 액션(fault, design §3.2)으로 유발** → 두 노드 hash 상이 → 검증 fail + 어긋난 노드 기록. `healPartition`으로 복구.
  3. en이 bp 높이-2 이내 → 싱크 pass.

## F9. 바이너리 셋 · 하드포크 2 type  (요구 25,26,36 · design §3.3,§4.2)
- **입력**: `chain.binary`(단일) 또는 `chain.binaries`(혼합/profile), `hardforks`(이름→블록번호).
- **동작**: 환경은 `node→(binary, buildVersion)` 집합. **단일 문자열=전 노드 동일**(쉬운 기본), 혼합은 `binaries`/`profile`. **buildVersion 획득**: setup 시 `<binary> version`(geth-family 서브커맨드) 또는 기동 후 `web3_clientVersion` RPC로 조회해 env.json에 기록(36). 하드포크는 config의 `hardforks` 값 유무로 존재 판정, 내용은 기록 수집. **type-1**(체인 업그레이드): 노드별 바이너리 집합으로 동시 실행(handoff). **type-2**(동일체인): fork 블록 **전에** fork-aware 바이너리로 교체(supervisor.ForkSwaps).
- **genesis 소싱(L3, design §3.8)**: `chain.genesis`/`genesisOverlay`로 4모드 선택 — ① existing(기존 파일 그대로) / ② build(`genesis.Build`) / ③ template+override(`Build`+`MergeOverride`) / ④ **upgrade-inherit**(wemix genesis 그대로 + 업그레이드 항목만 `MergeOverride`). 진입점은 `genesis.Build`(≠`ConsensusFamily.BuildGenesis`).
- **역할 용어(L2, design §3.9)**: **단일 도메인 어휘 `bp`/`en`/`boot`를 정의서·env.json 전반에 일관 사용**(bp=block producer, en=endpoint). 코드 enum `node.Role`("validator"/"endpoint")은 내부 식별자로만 두고 session이 도메인 용어로 직렬화. topology 파서는 "en"/"endpoint" 둘 다 수용.
- **출력**: env.json `meta.binaries`(빌드버전 포함); type-2 교체 이벤트 기록.
- **에러·엣지**: 바이너리 경로/버전 조회 실패 → 구성 실패. type-2 교체가 fork 블록을 넘긴 뒤면 오류.
- **AC**
  1. `binary:"go-wbft"` → 전 노드 동일 바이너리로 기동, env.json에 buildVersion 기록.
  2. handoff: producer=go-wemix, successor=go-wbft로 각각 기동(type-1).
  3. type-2: fork 블록 전에 교체 완료, 교체 후 fork 정상 활성.

## F10. data path · 라이브 로그 · 수집기  (요구 34,35 · design §3.6)
- **입력**: 각 노드 `dataPath`(로그가 쌓이는 실제 경로), 수집 대상(logs, chainstate).
- **동작**: 노드별 **append-only tail** 고루틴이 `dataPath` 로그를 `environments/<env>/logs/<node>.log`로 누적(일회성 복사 아님 → 누락 방지). chainstate(블록·bp참여·싱크·피어·분기)를 주기 수집. **out-of-process·버퍼·레이트리밋**으로 노드 무영향. remote는 ssh tail. `WaitLog`로 로그 패턴 도착 대기(완결성).
  - **확정(§9)**: chainstate 저장형식 = **jsonl 파일 + obs 미러**(대시보드 구독).
- **출력**: `logs/<node>.log`(tail 누적), `chainstate/*.jsonl`.
- **에러·엣지**: 라이브 로그가 자라는 중이어도 tail이 라인 누락 없이 누적. 로그 파일 접근 실패 → 해당 노드 수집 오류(테스트 전체는 계속). remote tail 끊김 → 재연결.
- **AC**
  1. 노드 동작 중 로그검증이 최신 라인까지 접근(스냅샷 누락 0).
  2. `WaitLog(pattern)`이 패턴 도착 시 그 파일·라인 반환.
  3. 수집 on/off 시 노드 블록생성 지표(간격) 유의미 변화 없음(무영향 근사).

## F11. pre/post · 스텝 결과 · 복구  (요구 12,27,32,37 · design §3.2,§3.3)
- **입력**: preActions/steps/postActions(모두 Action; preActions는 idempotent), 각 tx 스텝의 **결과 기대치**(`expectStatus` 기본 `"0x1"` | `expectRevert:true`), timeouts.
- **동작**: 순서 `pre → steps → assert → post`.
  - **스텝 결과 시맨틱(중요)**: tx 스텝은 **선언한 기대 결과가 충족되면 성공**이다. 기본 = 채굴+`status 0x1`. **negative 테스트**(예: 최소미달 언스테이킹 revert)는 `expectRevert:true`(또는 `expectStatus:"0x0"`)를 선언하며, 이때 **revert가 곧 성공**이다. 따라서 **tx revert가 스텝 실패인지 여부는 기대치가 결정**한다(기대와 다를 때만 스텝 실패). **인프라 실패**(전송 불가/영수증 timeout)는 기대치와 무관하게 항상 실패. 이로써 design의 "atomic step(부분성공 없음)"은 "선언된 기대 충족 = 원자적 성공"으로 성립.
  - **preAction 실패 → 테스트 미수행(BLOCKED)**. 스텝 기대 불충족 또는 assertion 실패 → 테스트 **FAIL**. **postAction 실패 → 어떤 테스트인지 기록**하되 **판정 집계는 스텝기대+assertion 기준 독립**. 미설정 액션은 skip.
- **출력**: status.json(pass/fail/blocked), steps.json(각 스텝 기대·실측·충족여부), postaction.json(성공여부·사유).
  - **노드 생명주기·fault 액션**(요구 12): `stopNode`/`startNode`/`restartNode`/`partition`/`healPartition`(+`chainMigrate`)는 스텝/액션으로 노드를 제어(procman·driver 경유). **destructive/fault 테스트**(WBFT-007/008 노드 중단→쿼럼, NODE-005 재기동→싱크복구, NODE-002 데이터 마이그레이션, **F8 분기=partition 유발**)에 필수.
  - **자산·컨트랙트 액션**: `faucet`(K) = keyreg 서명키에 gas 자금 공급(genesis alloc 미충전 신규키는 faucet 후 tx 가능). `deployContract`/`registerContract`(F) = 바이트코드 배포→주소 캡처→(bp 등록 필요 시) 컨트랙트에 노드정보 등록(배포 컨트랙트 vs 시스템 컨트랙트 구분, 요구 참조). 모두 design §3.2 액션 레지스트리 소속.
- **에러·엣지**: pre가 부분 성공 후 실패 → BLOCKED + 상태 기록(복구는 F13). post 실패가 판정을 바꾸지 않음. `expectRevert` 스텝이 오히려 성공(0x1) → 스텝 실패(기대 위반). 노드 중단 후 procman이 프로세스 종료를 검증(고아 0).
- **AC**
  1. pre 실패 테스트 → status=blocked, steps/assert 미실행.
  2. **positive 스텝(registerStaker)이 revert → 테스트 FAIL**(기대 0x1 위반).
  3. **negative 스텝(`expectRevert`)이 revert → 스텝 성공**; 오히려 0x1이면 스텝 FAIL.
  4. post 실패 테스트 → status는 스텝기대+assertion 결과대로, postaction.json에 실패 기록.
  5. 액션 미설정 → 아무 것도 안 하고 다음 단계 진행.
  6. `faucet(acctX)` 후 acctX 잔액 ≥ 요청액 → 이후 tx가 gas 부족 없이 성공.
  7. `deployContract(bytecode)` → 영수증의 컨트랙트 주소 캡처, `registerContract`로 bp 등록 후 정상 동작.

## F12. Placement · Port  (요구 15,16 · design §3.4)
- **입력**: 노드 목록(역할/sync/binary), 모드(local/remote), 포트 정책.
- **동작**: `place.Allocate`가 배치·포트 산출. **확정(§9)**: 기본 **LocalOSAssigned**(`:0` 바인드 후 회수 → 고정포트 이중바인드 근절), `--ports fixed`로 **LocalStepped**(index 결정적, 재현성) 선택. **RemotePerHost**=동일 포트+서버별 IP. etcd=peer(p2p+1).
- **출력**: NodePlacement[](host·ports·dataPath) → env.json node table.
- **에러·엣지**: 포트 소진/충돌 → 명확한 오류(연속 실행에서 재현). remote IP 미설정 → 오류.
- **AC**
  1. 연속 2회 로컬 기동(back-to-back)에서 포트 이중바인드 0건.
  2. `--ports fixed` → 동일 index 노드가 동일 포트(재현).
  3. remote 3서버 → 동일 포트·서로 다른 IP.
  4. **용량 검증(C·요구 5)**: validators<4 → 배치 전 오류; 노드 수 > 가용 용량(local 포트대역 / remote Σ서버슬롯) → 배치 전 오류(fail-fast).

## F13. 기동 소유·etcd 리더게이트·헬스 복구·teardown  (요구 3,12,37 · design §3.3, C-etcd)
- **입력**: plan, Options{LeaderGate, AlignJoinGap, MaxAttempts, Backoff, ForkSwaps}. remote 시 SSH 접속정보는 **`remote-server-config.yaml`(gitignore)** 에서 로드(L6b).
- **기동 소유(L6)**: **supervisor.BringUp이 노드 기동을 소유**(setup은 Plan+동시 provision/launch 프리미티브만 제공). 기동 후 **헬스 게이트**로 상태 확인·분류.
- **동작**: etcd: `etcdInit` 후 **리더 준비(`etcdIsReady`/`"*"`) 폴링 확인**, 조인 슬롯에 **시작 정렬**(gap은 supervisor가 **클러스터 크기 N에서 파생** — sz≤11→7s·≤23→11s·≤41→17s·else 23s; 호출자가 고정값 안 넘김, L7). 실패는 분류(`EtcdJoinFailed/EtcdStale/ForkNotCrossed/QuorumLost/RPCUnready`)하고 사유·producer 로그를 세션 보존. 백오프 재기동 또는 명확한 실패.
- **teardown·datadir(S2)**: 내장 etcd는 **프로세스 종료로 함께 종료**("살아있는 etcd 정리" 불필요). `Teardown{RemoveDataDir}`은 **종료와 별개 기능** — 재-셋업/디스크관리 시 datadir 삭제. procman이 노드별 `{PID, datadir}`를 추적해야 정확한 삭제 가능.
- **출력**: NodeSet + Diagnosis(OK/Mode/Detail/ProducerLog).
- **에러·엣지**: 원인 은닉 금지("flaky" 라벨 금지) — 항상 분류된 Mode+Detail. hang 없음(모든 대기 timeout). 실패 시 procman.StopAll로 고아 노드 0.
- **AC**
  1. producer etcd 리더 준비 전 조인시도 없음(게이트 통과 후 진행).
  2. 실패 시 Diagnosis.Mode가 실제 사유와 일치(예: 로그가 join 실패면 EtcdJoinFailed).
  3. 재기동(같은 datadir) 시 stale etcd로 인한 `cannot fetch cluster info` 재현 0 — **재-셋업 전 `RemoveDataDir`로 datadir 정리**.
  4. 어떤 실패에도 잔존 노드 프로세스 0(고아 없음). 종료 시 내장 etcd도 함께 종료.
  5. 노드 종료와 datadir 삭제가 **독립 호출**로 동작(종료만·삭제만 각각 가능).

## F14. MCP 결과 연동  (요구 31 · design §9)
- **입력**: 테스트/스위트 실행 요청(MCP tool), 진행/결과 조회 요청.
- **동작**: MCP가 세션 실행을 트리거하고, **세션 아티팩트(status.json/assert.json/session.json)를 읽어 결과 응답**. 진행중 스텝/판정을 스트림 또는 폴링으로 반환.
  - **확정(§9)**: 응답 스키마 = `{session, tests:[{id,status,assertPass,assertFail}], summary}`; 상세는 test별 assert.json 링크(provenance 포함).
- **출력**: MCP 응답(JSON), 세션 경로 참조.
- **에러·엣지**: 실행 실패/BLOCKED도 구조화 응답(사유 포함). 대용량 로그는 링크/요약(본문 폭주 금지).
- **AC**
  1. MCP로 스위트 실행 → 완료 후 pass/fail/blocked 요약 응답.
  2. 실패 테스트의 응답이 assert provenance(어느 값이 왜 틀렸는지)를 포함/링크.

## F15. 대시보드 수집  (요구 33,34 · design §9)
- **입력**: collector의 chainstate(jsonl+obs 미러), session 아티팩트.
- **동작**: 대시보드가 **블록 생성·bp 참여·노드별 싱크 높이·피어 연결·분기 여부**를 실시간 표시(요구 34)하고, 테스트별 요청/응답/결과(요구 33)를 조회. 정보는 `.chainbench/<session>/` 하위에서 읽음.
- **출력**: 대시보드 화면(체인 동작 상태·테스트 결과).
- **에러·엣지**: 수집 지연/누락이 있어도 노드/테스트에 영향 없음(F10 무영향). remote도 동일 수집.
- **AC**
  1. 실행 중 각 노드의 싱크 높이·피어·bp참여가 갱신 표시.
  2. 분기 발생 시 대시보드가 분기(노드 hash 불일치)를 표시.
  3. 테스트 완료 후 결과·판정 근거를 대시보드에서 조회.

## F16. 세션 운영·보안 규약  (요구 14,17,37 · design §3.1,§7 · 2차검토 누락보강)
- **입력**: 세션 실행 결과, 정의서, 키.
- **동작**
  - **spec schemaVersion(O2)**: 정의서 최상위에 `schemaVersion`(예: `"1"`) 필수 — 해석기가 버전 불일치 시 명확히 거부(장기 자산 forward-compat).
  - **키파일 권한(O3)**: `keys/<name>/private`·nodes/keystore는 **0600**, 디렉토리 **0700**으로 생성. private 값은 로그·아티팩트 본문에 노출 금지(경로/주소만). `.chainbench/`·`remote-server-config.yaml`은 gitignore(기존 규약).
  - **세션 보존/정리(O4)**: `.chainbench/<session>`은 누적되므로 보존정책 제공 — `chainbench clean [--older-than <dur>] [--keep-last N]`로 세션·datadir GC(재-셋업의 datadir 삭제와 동일 경로, F13-S2). 미정리 시 디스크 증가 경고.
  - **CI 종료코드(O5)**: 세션 커맨드는 **집계 결과를 exit code로 반환** — 전건 pass=`0`, fail 존재=`1`, blocked/인프라오류=`2`. `session.json.summary`와 일치(무인 CI 게이팅).
- **출력**: 버전거부 오류, 0600 키파일, GC 로그, 프로세스 exit code.
- **에러·엣지**: 알 수 없는 schemaVersion → 실행 거부(사유 명시). 키파일 권한 설정 실패 → 구성 실패(느슨한 권한으로 진행 금지). clean 대상이 실행중 세션이면 스킵.
- **AC**
  1. schemaVersion 누락/미지원 정의서 → 실행 거부(버전 명시).
  2. 생성된 `keys/*/private` 권한이 0600.
  3. `chainbench clean --older-than 7d`가 7일 경과 세션만 제거(실행중 세션 보존).
  4. fail 포함 스위트의 프로세스 exit code == 1(summary와 일치).

---

## 부록. 미결의 확정 요약 (design §9 대응)
| 미결 | 확정 |
|------|------|
| place 기본 포트 모드 | **LocalOSAssigned 기본**, `--ports fixed`로 stepped (F12) |
| collector chainstate 저장 | **jsonl 파일 + obs 미러** (F10) |
| applicableChains↔chain.name | chain.name=대상, applicableChains=호환집합·스윕·미적용 SKIP (F3) |
| fingerprint 대상 | **선언값 전체**(binaries+genesis+config+topology+hardforks+placement, flag>config>default 후), env-id=12hex 축약 (F7) |
| MCP 응답 스키마 | `{session,tests[],summary}` + assert provenance 링크 (F14) |
| assert 함수 세트 | testify식 네이밍 + 타입인지 + EqualHashAt (F6) |

**2차 검토 확정(코드 대조 반영):** genesis 4모드·진입점 `genesis.Build`(F9·L3) · 도메인 어휘 `bp`/`en` 표층 일관·enum 내부(F9·§3.9·L2) · etcd=wemix내장 `P2P+1` 예약(F12·L1) · supervisor가 기동 소유(F13·L6) · etcd gap은 N에서 파생(F13·L7) · datadir 삭제=종료와 별개(F13·S2) · SSH 자격증명 `remote-server-config.yaml`(F13·L6b) · 액션레지스트리 주입(design §3.2·P1) · 세마포어 `max(1,min(cores-2,N))`(§6·S1) · schemaVersion·키0600·세션GC·exit code(F16·O2~O5).

## 부록. 요구사항 ↔ F 추적
1-3→F3,F9 · 4-6→F4,F8,F12 · 7-8→F7 · 9-11,16→F2,F12,F13 · 12→F11,F13 · 13→F4 · 14,33-35→F1,F10,F15,F16 · 15-16→F12 · 17→F2,F16 · 18-26→F9,F13 · 27-28→F7,F11 · 29,30,32→F3,F5,F6 · 31→F14 · 36→F9 · 37→전체(F16 운영·보안).
