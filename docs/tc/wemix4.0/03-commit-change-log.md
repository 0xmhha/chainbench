# [WEMIX 4.0] Commit Change Log

> 출처: Confluence [[WEMIX 4.0] Commit Change Log](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2878996534) (페이지 ID 2878996534, 버전 1, 최종 수정 2026-08-05)  
> 상위 페이지: [WEMIX4.0] Test  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김)

---

# Commit Change History

| No. | Baseline Commit | Latest Commit | Updated At | Summary |
| --- | --- | --- | --- | --- |
| 1 |  |  |  | 최초 변경사항 |
| 2 | `3d7a0a5` | `9318f96` | 2026-08-05 | 추가 변경사항 |

## 2차 변경사항

- Commit Range: `3d7a0a5`\~`9318f96`

---

1. fix: prevent forgery and duplicate votes in WBFT justification ([#193](https://github.com/wemixarchive/go-wemix-wbft/commit/768341aaf470376cbcd6bbdb993383dedfae888d))

- 요약
    - JustificationPrepares 내 Prepare 메시지 서명 검증 추가 및 Round-Change/Prepare 쿼럼 집계 시 중복 투표 방지 강화
- 런타임 테스트 가능 여부 : 불가 (악의적 노드 필요)

---

1. fix: reject stale-view justifications in isJustified ([#194](https://github.com/wemixarchive/go-wemix-wbft/commit/587b7c3ab9847a1bd47f2e18a59a26edf29efafb))

- 요약
    - isJustified에서 ROUND-CHANGE justification의 view(Sequence+Round) 검증 추가, 이전 라운드의 stale justification 재사용 공격 차단
- 런타임 테스트 가능 여부 : 불가 (악의적 노드 필요)

---

1. fix: harden istanbul\_status RPC against resource exhaustion and data integrity issues ([#195](https://github.com/wemixarchive/go-wemix-wbft/commit/377c71939c673af8def29316f8701f9579187f39))

- 요약
    - istanbul\_status RPC에 블록 범위 상한 및 epoch 기반 검증자 집합 캐싱 추가
    - 음수 블록 번호 uint64 변환 언더플로우도 수정
- 런타임 테스트 가능 여부 : 가능

---

1. fix: harden WBFT backlog against future-message flooding ([#196](https://github.com/wemixarchive/go-wemix-wbft/commit/30be1f2fbe5c3eff92cfdb9a3d77731d4b612b44))

- 요약
    - future-sequence 메시지에 라운드 임계값 검증 및 per-validator 큐 크기 제한(64개) 추가
    - knownMessages LRU 우회 공격 차단으로 backlog DoS 방어 강화
- 런타임 테스트 가능 여부 : 불가

---

1. fix(txpool): restore cumulative affordability enforcement with fee-delegation accounting ([#197](https://github.com/wemixarchive/go-wemix-wbft/commit/5ba5d627b06ec53268f0dc11b4b68a61dd14f4a5))

- 요약
    - sender/fee-payer별 누적 잔액 검증을 분리하여 fee-delegated 트랜잭션의 누적 초과인출 방어 로직 복원
    - sender는 tx.Value(), fee-payer는 tx.FeeCost() 기준으로 별도 추적
- 런타임 테스트 가능 여부 : 가능

---

1. chore: add sensitive files to gitignore ([#198](https://github.com/wemixarchive/go-wemix-wbft/commit/803fba1a2812377cb65647c276ed39c5a5cfde13))

- 요약
    - .env\\*, \\*.pem, \\\*.key, .aws, .azure, .ssh 등 민감 파일 및 자격증명 경로 .gitignore에 추가
- 런타임 테스트 가능 여부 : 제외

---

1. ci: remove dev-test-work workflow ([#200](https://github.com/wemixarchive/go-wemix-wbft/commit/5322e249b04e63081c193832125a7d79627b503a))

- 요약
    - dev-test-work.yml CI 워크플로 제거
- 런타임 테스트 가능 여부 : 제외 (런타임 동작 아님)

---

1. fix: remove DecodeVanityData and return vanityData as raw hex ([#201](https://github.com/wemixarchive/go-wemix-wbft/commit/2ea9fd0f289463e53817e1deba1361c3de631010))

- 요약
    - 블록 vanity 데이터 디코딩 실패 시 index out-of-range panic 및 타입 어서션 panic 수정
    - DecodeVanityData 제거 후 raw hex 반환으로 변경
- 런타임 테스트 가능 여부 : 가능

---

1. triedb/pathdb: fix panic in recoverable (ethereum#29107) ([#202](https://github.com/wemixarchive/go-wemix-wbft/commit/0c69c5cf54ee804737d409d1d671f754e37e194e))

- 요약
    - state history freezer 미사용 시 pathdb.Database.Recoverable()에서 nil 역참조 panic 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. internal/ethapi: pass in accesslist in test (ethereum#29089) ([#203](https://github.com/wemixarchive/go-wemix-wbft/commit/f3428a7e372a6622f44b918a4fedf7fb75f57924))

- 요약
    - argsFromTransaction 테스트 헬퍼의 AccessList 변환 TODO 구현
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. cmd/geth: parseDumpConfig should not return closed db (ethereum#29100) ([#204](https://github.com/wemixarchive/go-wemix-wbft/commit/e9ca24f5fc5aed1e001e02cf47481a7fc6693fd8))

- 요약
    - parseDumpConfig가 내부에서 DB를 열고 닫은 후 닫힌 핸들을 반환하던 문제 수정. DB 열기/닫기 책임을 호출자로 이전
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. cmd/devp2p: fix commandHasFlag (ethereum#29091) ([#205](https://github.com/wemixarchive/go-wemix-wbft/commit/391ec6e9def069919b22676ca99662db3dacd40c))

- 요약
    - commandHasFlag()가 항상 false를 반환하여 devp2p discovery 서브커맨드에서 기본 bootnode가 자동 설정되지 않던 버그 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. fix: fix prestate nonce on contract creation (ethereum#29099) ([#206](https://github.com/wemixarchive/go-wemix-wbft/commit/b8edb2168a7d717ee4cb759b40e362b4b2cce464))

- 요약
    - 컨트랙트 생성 시 prestateTracer가 nonce=1 대신 0을 pre-state로 보고하던 오류 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. fix: enforce feePayer validation for fee-delegated transactions ([#207](https://github.com/wemixarchive/go-wemix-wbft/commit/924dc3b6f30a361360e4a542495a90403eecaed7))

- 요약
    - 블록 실행 경로에서 fee-delegated 트랜잭션의 feePayer 서명 검증 누락 수정
    - txpool·RPC에서는 검증하고 있었으나 블록 생성 시 우회 가능했던 취약점 해소
- 런타임 테스트 가능 여부 : 불가 (악의적 노드 필요)

---

1. core: initialize `gasRemaining` with `=` instead of `+=` (ethereum#29149) ([#208](https://github.com/wemixarchive/go-wemix-wbft/commit/97846d3d99fbd3095d7eb7cd359ae46168f0237c))

- 요약
    - buyGas()에서 gasRemaining 초기화 시 += 대신 = 사용, 기존 initialGas 초기화 스타일과 일치시킴
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. console: fix the wrong error msg of datadir testcase (ethereum#29183) ([#209](https://github.com/wemixarchive/go-wemix-wbft/commit/c4385c694545559ca360686c1b4cfbfc02ceb965))

- 요약
    - TestWelcome의 datadir 오류 메시지 문구 수정 및 console modules 출력에 대한 어서션 추가
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. p2p/simulations/adapters: fix error messages in TestTCPPipeBidirections (ethereum#29207) ([#210](https://github.com/wemixarchive/go-wemix-wbft/commit/28fe3870ee7a0782ac181cc3f111e265772a6f9a))

- 요약
    - TestTCPPipeBidirections에서 t.Fatalf 호출 시 expected/got 인자 순서가 뒤바뀐 오류 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. fix: replace path.Join with filepath.Join for file system paths (ethereum#29227, #29479, #29489) ([#211](https://github.com/wemixarchive/go-wemix-wbft/commit/c96835f01de93d21a015dbbaf092c2b86d14b102))

- 요약
    - 파일 시스템 경로 조합에 path.Join 대신 OS 환경 맞는 filepath.Join 사용으로 Windows 경로 구분자 버그 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. fix: fix leaks caused by http body not close ([#212](https://github.com/wemixarchive/go-wemix-wbft/commit/08a37ccefbf837a43caf95c8367e44fd55660a60))

- 요약
    - RPC HTTP 클라이언트에서 오류 응답 수신 시 response body를 닫지 않아 연결이 누수되던 문제 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. p2p: fix race in dialScheduler (ethereum#29235) ([#213](https://github.com/wemixarchive/go-wemix-wbft/commit/a2f2f0305260d37b4d02801b48d5650ce13782d7))

- 요약
    - dialTask.dest 필드를 atomic.Pointer로 교체하여 dial goroutine과 dialScheduler 메인 루프 간 데이터 레이스 제거
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. common/math: copy result in Exp (ethereum#29233) ([#214](https://github.com/wemixarchive/go-wemix-wbft/commit/5d64f454b742a4ac7612c5b169b98b17faeb1d28))

- 요약
    - common/math.Exp에서 base를 복사 후 제곱 연산하여 호출자의 인자 값이 의도치 않게 변경되던 버그 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. eth/tracers: fix concurrency issue for JS-tracing a block (ethereum#29238) ([#215](https://github.com/wemixarchive/go-wemix-wbft/commit/9cfb6a29b36e3fd2bd667ae69bb6275b7b79abd2))

- 요약
    - traceBlockParallel에서 여러 worker goroutine이 단일 blockCtx를 공유하여 GetHash 클로저의 캐시 슬라이스에 데이터 레이스 발생하던 문제 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. cmd/devp2p: fix decoding of raw RLP ENR attributes (ethereum#29257) ([#216](https://github.com/wemixarchive/go-wemix-wbft/commit/5a040b472ce23bcef3126658ca422a62eccba768))

- 요약
    - devp2p enrdump에서 raw RLP 값 포맷 시 rlp.SplitString을 사용하여 여분의 길이 prefix 바이트가 출력되던 문제 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. accounts/abi/bind: check invalid chainID first (ethereum#29275) ([#217](https://github.com/wemixarchive/go-wemix-wbft/commit/d68f94db8ce48e5b1b24576f3c712eabe9e2a0c8))

- 요약
    - NewKeyedTransactorWithChainID에서 키 주소 도출 전에 chainID 유효성 검사가 먼저 수행되도록 순서 변경
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. chore: make evm staterunner always output stateroot to stderr (ethereum#29290, ethereum#29298) ([#218](https://github.com/wemixarchive/go-wemix-wbft/commit/8dc3de14916817300b60e2e541dc82847253a49f))

- 요약
    - evm statetest가 --json 옵션 유무와 관계없이 항상 stateroot를 stderr로 출력하도록 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. fix: fix fee-delegated transaction signing in wallet and transaction API ([#219](https://github.com/wemixarchive/go-wemix-wbft/commit/fdb232872357aa11a8ca507167677cc26096e66b))

- 요약
    - KeyStore·scwallet 서명 백엔드에 FeeDelegateDynamicFeeTx 타입 처리 추가
    - fee-payer가 의도하지 않은 tx 타입에 서명하는 것을 막는 타입 불일치 가드도 추가
- 런타임 테스트 가능 여부 : 가능

---

1. eth/protocols/snap, internal/testlog: fix dataraces (ethereum#29301) ([#220](https://github.com/wemixarchive/go-wemix-wbft/commit/5a99446d28411bd52b549e500b75644fd278a4b7))

- 요약
    - testlog bufHandler에 sync.Mutex 추가, snap sync 테스트의 데이터 레이스 두 건 수정
- 런타임 테스트 가능 여부 : 제외 (업스트림 패치)

---

1. fix: pre-allocate AccessList slice in SetSenderTx before copying ([#221](https://github.com/wemixarchive/go-wemix-wbft/commit/9318f962a56a02fad62193ad143f9ea6467d2557))

- 요약
    - SetSenderTx에서 AccessList 복사 전 목적지 슬라이스를 사전 할당하지 않아 모든 항목이 손실되고 서명 복원이 실패하던 버그 수정
- 런타임 테스트 가능 여부 : 가능
