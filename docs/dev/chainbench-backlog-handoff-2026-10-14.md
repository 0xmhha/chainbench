# chainbench 후속 작업 인계 — 2026-10-14

이 문서는 새 세션을 위한 스냅숏이며 완료 증명이나 상태 정본이 아니다. 정본은
`docs/dev/chainbench-worklist.md` §0·§0a와 실제 코드·Git 상태다. 기존 인계 문서와 CLAUDE.md의
오래된 브랜치·PR·미완료 상태를 그대로 적용하지 않는다. 이 인계 파일은 미추적 상태로 남기며 커밋하지 않는다.

## 현재 체크아웃

- 저장소: `/Users/0xtopaz/work/github/0xmhha/chainbench`
- 브랜치: `feat/complete-chainbench-backlog`
- HEAD: `4ca97b33`
- 로컬 main과 캐시된 origin/main: `662fff77382eb031df687d371a8bc8f63cad581f`
- main 위 56개 커밋. 이 작업에서 push·PR은 수행하지 않았고 브랜치 upstream도 없다.
- 추적 파일의 미커밋 변경은 없다. 사용자 미추적 파일은 보존돼 있다. 이 인계 파일만 이번에 추가했다.
- 이번 인계 작성에서는 원격 fetch와 머신 전체 프로세스 전수 점검을 하지 않았다. 새 세션은 상태를 다시 확인한다.
- 현재 디스크 여유 약 28 GiB. 값은 변하므로 실제 증명 직전에 재검사한다.

Web CLI 전환 PR #448은 이미 main에 병합됐다. `refactor/web-cli-adapter`의 남은 읽기 전환을
처음부터 다시 하지 않는다. 현재 작업은 사용자 승인한 후속 14개 작업을 마무리하는 것이다.

최근 커밋:

| 커밋 | 검증된 변경 |
|---|---|
| `4ca97b33` | TX-017·기존 가스 소진 케이스의 실제 쓰기 뒤 롤백·가스·정확한 수수료. 세 체인 6개 실제 케이스·race·전체 검사 |
| `0269d8f6` | NODE-002 이관 전후 블록/트랜잭션/영수증 일치, 높이 19에서 포크 20을 넘어 36까지 실제 동기화 |
| `a8a0e256` | 완료된 b-5·netmap 재평가의 정본 상태 정정. §0 제품 잔여는 네 항목 |
| `a6d8d877` | 최신 원격 go-wemix에서 W3 실제 재현·인계. 결함 해결은 아님 |
| `8d517fa9` | 전체 언어 AST 인벤토리·그래프, A8/A7b·netmap 재평가 |

## 먼저 읽을 파일

1. `docs/dev/chainbench-worklist.md` §0a 전체 표와 그 뒤의 최근 단계 기록·최종 공통 게이트.
2. `docs/dev/architecture/mainnet-config-worklist.md`의 현행 잔여와 L1/L2/R2–R6 기준.
3. `docs/tc/wemix4.0/01-test-scenarios.md` — 원문 87개 요구.
4. `docs/dev/transaction-failure-evidence.json`, `data-migration-evidence.json`,
   `gov-staking-reactivation-evidence.json`, `gov-staking-resume-evidence.json`.
5. `docs/dev/chain-handover-2026-09-12.md`의 최신 W3·W4 절과 `wemix-w3-evidence.json`, `sync-path-evidence.json`.
6. `docs/dev/storage-document-evidence.json`, `runtime-storage-evidence.json`,
   `runtime-storage-failure-evidence.json`, `browser-dependency-storage-evidence.json`.
7. `docs/dev/web-ui-worklist.json`, `web-ui-worklist.md`, `architecture/chainbench-web-ui.seed.yaml`,
   `architecture/web-execution-adapter-decision.md`의 승인된 계약과 인수 기준.
8. `docs/dev/web-cli-refactor-handoff.md`의 환경·규칙·부록만 참고한다. 거기의 오래된 남은 일·브랜치는 정본을 덮어쓰지 않는다.
9. `CLAUDE.md`와 적용 가능한 AGENTS.md. SAGE Core Team 리뷰 금지 규칙을 지킨다.

## 전체 14개 상태

| 번호 | 작업 | 상태 |
|---|---|---|
| 1 | 증명 보존·WAL 압축 | 진행. 구현/보존 검증은 완료, 과거 receipt 전수 대조 잔여 |
| 2 | 문서 정합성 | 완료한 범위 유지 |
| 3 | suite 실패 테스트 | 완료 |
| 4 | dashboard 정리 경합 | 완료 |
| 5 | feature 등록 | 완료 |
| 6 | MCP 레지스트리 소비 | 완료 |
| 7 | 서버 배치·BLS | 완료 |
| 8 | 요구 19개 충분성 | 해당 검토·보강 완료. 다른 트랙/최종 인수까지 완료한 것은 아님 |
| 9 | 메인넷 live | 진행. Linux fleet·전체 L1·R3/R4/P4/R5 등 잔여 |
| 10 | 공통 TC 도구 | 준비물 공유·관측 가능 14개 동기화 완료. W4 때문에 WEMIX CT-NODE-007 미완료 |
| 11 | 수직 슬라이스 | 진행. 아래 원문 충분성·legacy 대조 잔여 |
| 12 | b-5 재평가 | 완료 |
| 13 | netmap·전체 언어 AST | 해당 단계 완료. 이후 HEAD에서 인벤토리 재측정 가능 |
| 14 | W3 | 최신 재현·인계 완료. 외부 결함 미해결이므로 제품 항목 열림 |

추가 공통 마지막 관문: 후속 HEAD의 Web 14개 고정 소스 capture + 독립 재현 + 전체 필수 race/live.
과거 main의 소스 지문 `3887afccc004727a051e913ce61d937e37a7274cd1036bc3b0372bb3d15a70bc`의
인수는 이 후속 HEAD의 인수가 아니다.

## 추천 진행 순서와 구체 잔여

### 1. 수직 슬라이스 원문 충분성과 legacy 대조

- **TX-018:** 원문은 `eth_estimateGas`로 추정하고 `estimate - 1`로 보내 status 0, gasUsed == gasLimit,
  상태 롤백을 요구한다. 현행 CT-CONTRACT-007은 실제 쓰기 뒤 무한 반복으로 50000 가스를 소진한다.
  롤백·전량 소비·정확한 수수료는 검증했지만 원문 입력 절차는 미검증이다.
  WBFT ethapi는 상향 오차 0.015를 허용하므로 `estimate - 1`이 항상 부족하다는 전제도 성립한다고 가정하면 안 된다.
  다음으로 유한한 쓰기 fixture에서 실제 추정치·한도·영수증·상태를 대조한다. 실패 기대를 낮추거나
  원문을 조용히 바꾸지 않는다. 요구 변경 결정이 필요하면 구체적인 관찰을 제시한 뒤 확인한다.
- **GOV-019:** 정상 재활성화의 정확한 계약 목록·후속 에폭 후보/검증자 복귀·이벤트는 검증했다.
  그러나 원문의 최소 금액 미달 “revert 없이 단순 stake 추가”와 현행 v1 계약의 명확한 revert가 충돌한다.
  현재 증명은 eth_call error 3 및 정확한 사유·상태 불변이며, 미달 거래의 채굴 성공 증명이 아니다.
- **원문 87개 전수:** 삭제된 과거 tracker는 84개 ID였다. 이름/함수 존재와 실제 검증 명제를 구분해 현재 케이스와 대조한다.
  WBFT-010의 암호학적 Randao/XOR, 블록 주기 조건, GOV-021의 조기 실행 거절·정확한 이벤트 등은
  충분성이 확인된 것으로 가정하지 말고 실제 원문·단언을 재검토한다.
- **legacy:** `docs/dev/legacy-port-audit/`의 기록과 현행 소비자·케이스를 대조한다.
  당시 소스 경로와 숫자는 역사 기록이므로 원본이 실제 존재하는지부터 확인한다.
- `NODE-002`의 트랜잭션/영수증·포크 이후 동기화는 `0269d8f6`에서 이미 보강했다. 이전 “데이터 인식만” 기록을 현행 잔여로 재사용하지 않는다.
- `TX-017`과 기존 RT-A-3-07은 `4ca97b33`에서 이미 보강했다. 실패 전 SSTORE 완료를 확인했으므로 다시 쓰기 없는 fixture로 되돌리지 않는다.
- 기존 `StepResult.Gas`가 빈 값인 점을 관측했다. 마지막 native 검증은 실제 거래를 RPC로 조회해 gas/input/hash/blockHash를 대조했다.
  가스 한도를 빈 단계 기록에서 추정하지 않는다.

### 2. 메인넷 live 잔여

Linux fleet에서 출처가 검증된 HEAD 바이너리로 전체 L1을 재실행한다. R3의 남은 RPC/WS/시스템 컨트랙트
행동·P4 충분성, R4 포크/제어의 실제 영향과 전수 회귀, R5 준비/추정/전송/HTTP 오류와 실제 노드 제출 거부 구분,
전체 입력·배포·실행 출처를 대조한다. 이미 완료된 native 프로필·수치 입력·서명·행렬 등의 하위 검증은
워크리스트와 각각의 evidence.json을 읽어 재사용하되 Linux/full-mainnet 인수로 확대하지 않는다.

현재 Linux fleet/Docker의 가동 상태와 원격 실행 가능성은 이번 인계에서 새로 확인하지 않았다.

### 3. 과거 receipt 전수 대조와 보존

56개 중 28개 해시 일치, 26개 없음, 2개 다른 해시다. 목록은 `storage-document-evidence.json`에 있다.
원본/복사본/기존 증명에서 동일 bytes를 찾을 수 있는지 대조한다. 없는 자료를 재작성해서 옛 해시에 맞췄다고
주장하지 않는다. 회수 불가능하면 손실/불일치를 그대로 기록하고 새 인수와 구분한다.

### 4. 외부 결함 W3·W4

- W3: 원격 go-wemix `4538a9f5`에서 Registry 생성 블록까지만 받은 reader가 다음 블록을 unauthorized block으로
  거절하며 연결된 downloader도 실패하는 것을 재현했다. 진단 PASS는 해결 PASS가 아니다.
  과거 15노드 경합과 재시작 복구를 재검증한 것도 아니다. `02-wemix-chain-up-15`는 계속 BLOCKED.
- W4: ETH/LES fetcher의 같은 meter 이름 등록이 충돌해 ETH 실제 수신 값이 exporter에 나오지 않는다.
  WEMIX CT-NODE-007의 metric 조건을 삭제하거나 downloader 지표로 바꿔 닫지 않는다.
  W4 관측 바이너리는 기존 로컬 WEMIX `1350376a`였다. 새 원격 소스로 재확인할 필요가 있다.
- 외부 체인 소스 수정은 승인되지 않았다. 재현·인계 자료만 준비했고 체인팀에 메시지/리뷰를 보내지 않았다.

### 5. 최종 고정 소스 인수

필요한 구현·문서·SPA 빌드를 먼저 끝낸 후 소스를 고정한다. WEB-01~14 각각 capture와 독립 재현을
직렬로 실행하고 WEB-10·WEB-14와 필수 전체 race/live를 빠뜨리지 않는다.
실행 중에는 문서와 이 인계 파일도 수정하지 않는다. 옛 인수 스크립트가 현행 env·검증 목록·정리 규칙에
맞는지 읽고 수정한 후 실행한다. 실측 없이 오래된 분모/skip을 통과로 치지 않는다.

## 머신·환경·저장 공간

기존 실측 머신은 MacBook Air M2 arm64, RAM 8 GB다. 실제 증명은 항상 하나씩 직렬로 실행한다.

```sh
export WEBUI_RUNTIME_ROOT=/private/tmp/cbui
export GOCACHE=/private/tmp/chainbench-go-cache
export PYTHONDONTWRITEBYTECODE=1
export WEBUI_REPLACEMENT_BINARY=/private/tmp/chainbench-web-replacement-native/gwbft
export GSTABLE_BIN=/private/tmp/chainbench-head-native/gstable
export GWBFT_BIN=/private/tmp/chainbench-head-native/gwbft
export GWEMIX_BIN=/private/tmp/chainbench-w3-head-native/gwemix
export CHAINBENCH_E2E_FROM_BIN="$GWEMIX_BIN"
export CHAINBENCH_E2E_TO_BIN="$GWBFT_BIN"
export GOMAXPROCS=2
export GOFLAGS=-p=1
```

바이너리 출처는 실행 전에 다음 JSON과 실제 SHA/VCS revision을 대조한다:

- `/private/tmp/chainbench-head-native/provenance.json`: StableNet `0937ac5c`, WBFT `7af50e45`, **기존 WEMIX `1350376a`**.
- `/private/tmp/chainbench-w3-head-native/provenance.json`: **최신으로 확인한 원격 WEMIX `4538a9f5`**, 별도 checkout의 깨끗한 tracked 상태와 빌드 출처.
- Stable/WBFT는 로컬 checkout HEAD였으며 현재 원격 HEAD라고 재확인한 것이 아니다. WEMIX 원격 확인도 그 시점의 값이다.
- 새 WEMIX 소스: `/private/tmp/chainbench-w3-head-source`. `go-wemix/cmd/gwemix`는 `geth`를 가리키는 tracked symlink다.
  소스 인벤토리는 링크의 대상 경로 bytes도 구분해 해시한다. 링크를 일반 파일처럼 read_bytes하지 않는다.
- Go 1.25.2로 외부 소스를 빌드하면 `GOROOT=/Users/0xtopaz/.gvm/gos/go1.25.2`, `GOTOOLCHAIN=local`과
  해당 Go 실행 파일을 함께 사용한다. 이 저장소 Go 1.26.8용 GOROOT와 섞으면 컴파일러 버전 오류가 난다.
- `/private/tmp` 자료는 재부팅/정리 후 없을 수 있다. 파일 존재만으로 출처가 확인됐다고 하지 않는다.

증명 직전 여유가 20 GiB 미만이면 승인된 `python3 /private/tmp/cbui-cleanup.py --delete`를 사용한다.
스크립트를 먼저 읽고 보존 규칙을 지킨다. 실제 실행이 사용 중인 데이터·로그·설정·영수증을 임의로 지우지 않는다.
전체 cbui나 Go cache를 통째로 삭제하지 않는다. 새 실제 테스트는 owned marker·사전 공간 검사와
종료 후 참조되지 않는 DB 회수를 쓰도록 만든다. `tests/webui/runtime_storage.py`와 최근 native fixture를 참고한다.

최근 두 단계는 종료 DB 0개다. transaction-failure는 8개 시도, 1466개 파일을 보존했고
NODE-002/관련 W3는 3개 시도, 738개 파일을 보존했다. 이것은 cbui 전체가 미사용이라는 의미가 아니다.

## 검증 실행 시 주의

일반 전체 검사에서는 위 native bin 변수를 빼야 gate된 실제 테스트를 의도치 않게 전부 실행하지 않는다:

```sh
env -u GSTABLE_BIN -u GWBFT_BIN -u GWEMIX_BIN \
  -u CHAINBENCH_E2E_FROM_BIN -u CHAINBENCH_E2E_TO_BIN \
  -u CHAINBENCH_W3_PROOF -u CHAINBENCH_HANDOFF_STORAGE_PROOF \
  -u CHAINBENCH_HANDOFF_REATTACH_PROOF \
  GOCACHE=/private/tmp/chainbench-go-cache GOMAXPROCS=2 GOFLAGS=-p=1 make check
```

- Go 네트워크 단위 테스트는 로컬 port bind가 필요하다. sandbox bind 거절은 제품 RED가 아니다.
  필요한 실행 권한으로 재검증하고 실패 로그를 보존한다.
- 최신 `make check`는 통과했다. 그 앞 시도는 두 케이스의 주소 참조 위치/8개 추가 때문에 corpus golden이 달라 실패했다.
  `internal/testhelper/testdata/corpus-addresses.golden`을 두 케이스만 검토·갱신해 854→862로 바뀌었다.
- 실제 failure witness: `go test -race ./internal/testengine -run '^TestNativeTransactionFailureRollback$' -count=1`.
  세 bin 변수를 지정해야 하며 세 체인 2개 케이스씩 실제 실행한다.
- NODE-002: e2e tag와 FROM/TO bin을 지정해 `TestWemixDataMigrationE2E`를 실행한다.
- W3 진단은 `CHAINBENCH_W3_PROOF=1`이 필요하고 진단 PASS에 issueResolved=false를 유지한다.
- 과거 전체 WEMIX native 20개는 당시 고정 소스의 이력이다. 이후 변경 뒤 현재 HEAD의 전체 회귀로 세지 않는다.
  W3 진단이 추가되어 `^TestWemix` 함수 수는 달라졌다. `/private/tmp/chainbench-handoff-regression-run.py`의
  과거 분모 20을 그대로 실행하거나 결과를 조용히 제외하지 않는다. 검사 범위를 명시해 현재 목록부터 확인한다.
- 전체 AST 도구: `scripts/inventory/all_code_graph.py`와 README. parser 도구는 `/private/tmp/chainbench-ast-tools`에 있다.
  구조적 syntax 그래프이며 실행 커버리지나 타입 분석으로 해결된 call graph라고 주장하지 않는다.
- `/private/tmp/chainbench-wemix4-reassessment-inputs.json`과 assertion-graph JSON은 과거 분석 스냅숏이다.
  마지막 NODE-002/TX-017 보강 뒤에는 오래된 누락 판단을 재사용하지 않는다.
- `/private/tmp/claude-501/frozen-final.sh`, `frozen-recheck.sh`는 현재 존재했다. 실행 전에 내용을 재검토한다.
  `tests/webui/verify.sh`, capture.py, recheck.py와 승인된 Seed/인수 계약이 실제 기준이다.

## 계속 지킬 규칙

1. 매 구현 단계 전에 전체 14개 목록·현재 완료 범위·그 단계의 완료 기준을 나열한다.
2. 의미 있는 RED→GREEN, 관련 회귀, 필요한 실제 증명을 모두 통과한 변경만 명시 파일로 커밋하고 다음 단계로 계속한다.
3. 부분 완료·환경 실패·mock·skip·과거 인수·파일 존재를 전체 완료로 바꾸지 않는다.
4. 실제 증명은 직렬이며 실행 중 저장소 소스/문서 변경 금지다. SPA 빌드는 소스 고정 전에 끝낸다.
5. `git add .`, `commit -a`, reset, clean, stash는 금지다. 커밋 메시지는 영어 Conventional Commits,
   전체 300자 이내이며 emoji/co-author/Claude 표기가 없어야 한다.
6. `CLAUDE.md`, `.ouroboros_eval_artifact.md`, 준비 Seed/검토 문서, baselines·audit와 기존 미추적 인계 자료를 보존한다.
   `docs/dev/web-ui-claude-handoff.md`는 덧붙이기만 한다.
7. push·PR 생성·merge는 사용자에게 확인한다. SAGE Core Team 및 `SAGE-X-project/sage-core-dev` 리뷰 요청은 금지다.
   자동 리뷰 배정도 확인하며 해당 팀 배정이 있으면 승인된 변경으로 제거하기 전 PR을 draft로 유지한다.
8. 외부 체인 소스는 수정하지 않는다. 저장된 PID만으로 신원 확인 없이 프로세스를 종료하지 않는다.
9. 승인된 Seed·UI/CLI 결정과 `docs/chain-analysis/`의 실제 체인 명령/옵션을 사용한다.
   ooo는 기존 합의대로 qa/lateral/status/evaluate의 보조 검토에 사용한다. 중복 evolve/run/Ralph 실행은 시작하지 않는다.
10. 한국어로 결과·근거·한계를 설명하고 확인하지 않은 것은 확인하지 않았다고 쓴다.

새 세션은 TX-018 실제 입력 검토와 87개 충분성·legacy 대조부터 이어가는 것을 권장한다.
미승인 요구 변경이나 외부 수정이 필요한 부분은 결정이 필요하다고 구분하고, 독립적인 나머지 작업은 계속 진행한다.
