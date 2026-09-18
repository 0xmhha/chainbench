# Commit ChangeLog

> 출처: Confluence [Commit ChangeLog](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2875326583) (페이지 ID 2875326583, 버전 11, 최종 수정 2026-08-05)  
> 상위 페이지: 테스트(v1.0.0이후 변경 사항)  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김. 원본의 인라인 댓글 표시와 날짜 매크로는 변환에서 빠짐)

---

## Commit Change History

| No. | Baseline Commit | Latest Commit | Updated At | Summary |
| --- | --- | --- | --- | --- |
| 1 |  | `940e9f2` |  | 최초 변경사항 |
| 2 | `c37994e` | `54a5cbd` |  | 추가 변경사항 |

## 2차 변경사항 

- Commit Range : `c37994e`\~`54a5cbd`


1. fix(txpool): restore cumulative affordability enforcement with fee-delegation accounting ([#116](https://github.com/stable-net/go-stablenet/commit/54a5cbd599f0ba3ef6cc906e7e43c29beea5b761))

- 요약
    - ender/fee-payer별 누적 잔액 검증을 분리하여 fee-delegated 트랜잭션의 누적 초과인출 방어 로직 복원.
    - sender는 tx.Value(), fee-payer는 tx.FeeCost() 기준으로 별도 추적
- 런타임 테스트 가능 여부 : 가능

---

1. fix: fix fee-delegated transaction signing in wallet and transaction API ([#114](https://github.com/stable-net/go-stablenet/commit/4444af086cb98ced96bd54470d982fb65106301e))

- 요약 
    - KeyStore·scwallet 서명 백엔드에 FeeDelegateDynamicFeeTx 타입 처리 추가. 
    - fee-payer가 의도하지 않은 tx 타입에 서명하는 것을 막는 타입 불일치 가드도 추가
- 런타임 테스트 가능 여부 : 가능 

---

1. fix: pre-allocate AccessList slice in SetSenderTx before copying ([#115](https://github.com/stable-net/go-stablenet/commit/f4c9490f73f9f5f7a3a059ab6b55e634117407dc))

- 요약
    - SetSenderTx에서 AccessList 복사 전 목적지 슬라이스를 사전 할당하지 않아 모든 항목이 손실되고 서명 복원이 실패하던 버그 수정
- 런타임 테스트 가능 여부 : 가능 

---

1. fix: reject block with nil GasTip in verifyGasTip ([#113](https://github.com/stable-net/go-stablenet/commit/b46ef9ef3bc34d9204043e0b70628c6ff2ea39a6))

- 요약
    - verifyGasTip에서 nil GasTip을 governance 값과의 불일치로 처리, 값이 없는 경우에도 검증 실패하도록 강화
- 런타임 테스트 가능 여부 : 불가 (악의적 노드 필요)

---

1. fix: fix dataraces in testlog and snap sync test (ethereum#29301) ([#112](https://github.com/stable-net/go-stablenet/commit/7725aaea07b460e1f4905f17b93886ed5bef84b0)) 

- 요약
    - testlog bufHandler에 sync.Mutex 추가, snap sync 테스트의 데이터 레이스 두 건 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림패치)

---

1. chore: make evm staterunner always output stateroot to stderr (ethereum#29290, ethereum#29298) ([#111](https://github.com/stable-net/go-stablenet/commit/c3466d48cda9228ce20f9735e9d2290a4e3a4baf))

- 요약
    - evm statetest가 --json 옵션 유무와 관계없이 항상 stateroot를 stderr로 출력하도록 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림패치)

---

1. chore: sync v1.1.0 version and Boho fork block into dev ([#110](https://github.com/stable-net/go-stablenet/commit/14d502a11d2decfe4a5b6a2f8f369dd2366d3b82)) 

- 요약
    -  v1.1.0 버전 정보 및 StableNet Testnet의 Boho 하드포크 블록 번호(→14,408,500)를 dev 브랜치에 동기화
- 런타임 테스트 가능 여부 : 불가 (런타임테스트아님)

---

1. fix: check invalid chainID first (ethereum#29275) ([#106](https://github.com/stable-net/go-stablenet/commit/43bcf51dbca66133e00853321abcde89dd35c0a7)) 

- 요약
    - NewKeyedTransactorWithChainID에서 키 주소 도출 전에 chainID 유효성 검사가 먼저 수행되도록 순서 변경
- 런타임 테스트 가능 여부 : 제외 (업스트림패치)

---

9\. fix: fix prestate nonce on contract creation (ethereum#29099, wbft#206) ([#107](https://github.com/stable-net/go-stablenet/commit/a1d1771c66745721a15735e67a78b8d10e8b4f3a))

- 요약
    - 컨트랙트 생성 시 prestateTracer가 nonce=1 대신 0을 pre-state로 보고하던 오류 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

10\. docs: document GovCouncil system contract in README ([#108](https://github.com/stable-net/go-stablenet/commit/e343baf86b8cb14aadc0ab2c8a83345737e9689c))

- 요약
    - README에 GovCouncil 시스템 컨트랙트(0x…1004) 추가. Anzeon config 스니펫과 genesis.json 예시에 govCouncil 블록 추가
- 런타임 테스트 가능 여부 : 불가 (런타임 동작 아님)

---

11\. fix(devp2p): fix decode raw RLP value (ethereum#29257) ([#105](https://github.com/stable-net/go-stablenet/commit/12be31bfc361ee972bf74f087ab0c058d3c88d11))

- 요약
    - devp2p enrdump에서 raw RLP 값 포맷 시 rlp.SplitString을 사용하여 여분의 길이 prefix 바이트가 출력되던 문제 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

12\. fix: fix concurrency issue for JS-tracing a block (ethereum#29238) ([#104](https://github.com/stable-net/go-stablenet/commit/4a8e37eb000ca5b5e4a5fb98db0fac98bed719c0))

- 요약
    - traceBlockParallel에서 여러 worker goroutine이 단일 blockCtx를 공유하여 GetHash 클로저의 캐시 슬라이스에 데이터 레이스 발생하던 문제 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

13\. fix: enforce feePayer validation for fee-delegated transactions ([#103](https://github.com/stable-net/go-stablenet/commit/88efe089b154f97da2d9cacec45d72250c9c8fa1))

- 요약
    - 블록 실행 경로에서 fee-delegated 트랜잭션의 feePayer 서명 검증 누락 수정
    - txpool·RPC에서는 검증하고 있었으나 블록 생성 시 우회 가능했던 취약점 해소
- 런타임 테스트 가능 여부 : 불가 (악의적 노드 필요)

---

14\. fix: pre-initialize PrevPrepared/PrevCommitted on epoch transition ([#91](https://github.com/stable-net/go-stablenet/commit/74f960333f383c0fb2125a9d222c0cb7a4caaff7))

- 요약
    - Epoch 전환 시 이전 검증자를 PrevPrepared/PrevCommitted 맵에 사전 초기화, 서명 제출이 없는 검증자가 sealer 활동 API 응답에서 누락되던 문제 수정
- 런타임 테스트 가능 여부 : 가능

---

15\. fix: prevent Exp from mutating the base argument (ethereum#29233) ([#102](https://github.com/stable-net/go-stablenet/commit/7b0517c59053413f94f71aad3387b58fbdf7ff6c))

- 요약
    - common/math.Exp에서 base를 복사 후 제곱 연산하여 호출자의 인자 값이 의도치 않게 변경되던 버그 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

16\. fix: close response body on RPC HTTP client error path (ethereum#29223) ([#100](https://github.com/stable-net/go-stablenet/commit/698de573b110d7fb5b5cc6f491839c3a689287ac))

- 요약
    - RPC HTTP 클라이언트에서 오류 응답 수신 시 response body를 닫지 않아 연결이 누수되던 문제 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

17\. fix: eliminate data race on dialTask.dest field (ethereum#29235) ([#101](https://github.com/stable-net/go-stablenet/commit/ca6b244e2a5d504dfba170d1a2869de79a275c86))

- 요약
    - dialTask.dest 필드를 atomic.Pointer로 교체하여 dial goroutine과 dialScheduler 메인 루프 간 데이터 레이스 제거
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

18\. fix: replace path.Join with filepath.Join for file system paths (ethereum#29227, #29479, #29489) ([#99](https://github.com/stable-net/go-stablenet/commit/0e85a3314da7c5dbdad16a58f3751cba5b4bf79b))

- 요약
    - 파일 시스템 경로 조합에 path.Join 대신 OS 환경 맞는 filepath.Join 사용으로 Windows 경로 구분자 버그 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

19\. test: fix error message argument order in TestTCPPipeBidirections (ethereum#29207) ([#98](https://github.com/stable-net/go-stablenet/commit/b27c3230657951ee3ccd3181c4a230b00d86ae44))

- 요약
    - TestTCPPipeBidirections에서 t.Fatalf 호출 시 expected/got 인자 순서가 뒤바뀐 오류 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

20\. test: fix wrong error msg of datadir testcase (ethereum#29183) ([#97](https://github.com/stable-net/go-stablenet/commit/2b899b9eb8f9069dd4cb880bedc994f48a35d577))

- 요약 
    - TestWelcome의 datadir 오류 메시지 문구 수정 및 console modules 출력에 대한 어서션 추가
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

21\. refactor: initialize gasRemaining with = instead of += (ethereum#29149) ([#96](https://github.com/stable-net/go-stablenet/commit/53a4c07d9a8efd9be380ff171a18f530f7db94d6))

- 요약
    - buyGas()에서 gasRemaining 초기화 시 += 대신 = 사용, 기존 initialGas 초기화 스타일과 일치시킴
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

22\. fix: fix commandHasFlag (ethereum#29091) ([#95](https://github.com/stable-net/go-stablenet/commit/34b9142f480c9e095a3a3ca2ea0d8dc3a35e20ac))

- 요약
    - commandHasFlag()가 항상 false를 반환하여 devp2p discovery 서브커맨드에서 기본 bootnode가 자동 설정되지 않던 버그 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

23\. fix: parseDumpConfig should not return closed db (ethereum#29100) ([#94](https://github.com/stable-net/go-stablenet/commit/e68e92aef8051971b4f4571f070316354904cce4))

- 요약 
    - parseDumpConfig가 내부에서 DB를 열고 닫은 후 닫힌 핸들을 반환하던 문제 수정. DB 열기/닫기 책임을 호출자로 이전
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

24\. test: implement accessList conversion in test helper (ethereum#29089) ([#93](https://github.com/stable-net/go-stablenet/commit/470cb66bb8a2f13f283e90c6ec84ec7950bc14bb))

- 요약
    - argsFromTransaction 테스트 헬퍼의 AccessList 변환 TODO 구현
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

25\. fix: fix panic in recoverable (ethereum#29107) ([#92](https://github.com/stable-net/go-stablenet/commit/dffe4a33a7eb4720a1e2d0f9103c6839b8d117c2))

- 요약 
    - state history freezer 미사용 시 pathdb.Database.Recoverable()에서 nil 역참조 panic 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

26\. fix: harden WBFT backlog against future-message flooding ([#89](https://github.com/stable-net/go-stablenet/commit/a49baa68e9ef49e1c9e824bc36e10128384b86a4))

- 요약 
    - future-sequence 메시지에 라운드 임계값 검증 및 per-validator 큐 크기 제한(64개) 추가.
    - knownMessages LRU 우회 공격 차단으로 backlog DoS 방어 강화
- 런타임 테스트 가능 여부 : 불가 

---

27\. chore: update .gitignore rules ([#90](https://github.com/stable-net/go-stablenet/commit/e67eb9498e034914f7192ebcdea3b77fc6042162))

- 요약
    - .env\*, \*.pem, \*.key, .aws, .azure, .ssh 등 민감 파일 및 자격증명 경로 .gitignore에 추가
- 런타임 테스트 가능 여부 : 제외 

---

28\. fix: remove DecodeVanityData and return vanityData as raw hex ([#88](https://github.com/stable-net/go-stablenet/commit/57d52f77917d0db430465a25d442ef8d186cb87a))

- 요약
    - 블록 vanity 데이터 디코딩 실패 시 index out-of-range panic 및 타입 어서션 panic 수정
    - DecodeVanityData 제거 후 raw hex 반환으로 변경
- 런타임 테스트 가능 여부 : 가능

---

29\. fix: harden istanbul\_status RPC against resource exhaustion and data integrity issues ([#86](https://github.com/stable-net/go-stablenet/commit/d7cff3df90251357a90fc9ec697505455058b59d))

- 요약
    - istanbul\_status RPC에 블록 범위 상한 및 epoch 기반 검증자 집합 캐싱 추가
    - 음수 블록 번호 uint64 변환 언더플로우도 수정
- 런타임 테스트 가능 여부 : 가능

---

30\. fix: reject stale-view justifications in isJustified ([#85](https://github.com/stable-net/go-stablenet/commit/c051d50beb60ed0e7be4c9263971094db29fd9b9))

- 요약
    - isJustified에서 ROUND-CHANGE justification의 view(Sequence+Round) 검증 추가, 이전 라운드의 stale justification 재사용 공격 차단
- 런타임 테스트 가능 여부 : 불가 (악의적 노드 필요)

---

31\. fix: prevent forgery and duplicate votes in WBFT justification ([#84](https://github.com/stable-net/go-stablenet/commit/9978930ba62380a428f67ad6ff664a6a52e4a547))

- 요약
    - JustificationPrepares 내 Prepare 메시지 서명 검증 추가 및 Round-Change/Prepare 쿼럼 집계 시 중복 투표 방지 강화
- 런타임 테스트 가능 여부 : 불가 (악의적 노드 필요)

---

32\. fix: set zero Balance for params-only alloc entries in initializeGovCouncil ([#83](https://github.com/stable-net/go-stablenet/commit/3eada119e5577b02c81bb8c346fd82b20649c71f))

- 요약
    - initializeGovCouncil에서 params 전용 alloc 항목의 Balance를 nil 대신 zero로 설정, genesis dumpgenesis 시 required 태그 위반으로 실패하던 문제 수정
- 런타임 테스트 가능 여부 : 제외(런타임동작아님/CLI를 이용한 검증필요) - **테스트가 불가능한 이유는??**

---

1.  fix: race conditions in newRoundChangeTimer ([#82](https://github.com/stable-net/go-stablenet/commit/c37994e9b12a54bcd240164cb5e91f45d3696f6f))

- 요약
    - newRoundChangeTimer에서 클로저 캡처 시점 오류로 stale cancel 포인터를 읽는 race condition 두 건 수정
- 런타임 테스트 가능 여부 : 불가 (Race condition 재현 필요)

---

34\. ci: add master-ci workflow ([#79](https://github.com/stable-net/go-stablenet/commit/940e9f281edbdbc3df088a14e77a106908bfcb5d))

- 요약
    - master 브랜치 PR 대상 CI 워크플로(master-ci.yml) 추가
    - make gstable 빌드 검증 포함
- 런타임 테스트 가능 여부 : 제외 (런타임 동작 아님)
