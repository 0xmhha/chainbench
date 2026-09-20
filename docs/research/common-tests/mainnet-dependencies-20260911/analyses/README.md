# 세 체인 공통 테스트의 메인넷 의존 분석 (2026-09-11)

WEMIX3.0(go-wemix), WEMIX4.0(go-wbft), StableNet(go-stablenet)에서 같은 의미로 돌릴 수 있는 테스트를 가려내고, 각 테스트가 어느 메인넷 값에 의존하는지, 무엇을 설정으로 빼고 무엇을 따로 구현해야 하는지 정리했다. 2026-09-09 자료가 있었지만 그때 코드 기준이라 결론과 줄번호를 승계하지 않았다. 테스트 파일 195개, 명세 문서 행 74개, 세 클라이언트 소스, 하네스 소스를 2026-09-11 작업 트리에서 다시 읽었다.

네 가지 완료 조건에 대한 답은 [설정 분리와 변경 범위](configuration-and-change-scope.md)에 있다. 나머지 문서는 그 근거다.

## 결과 요약

| 구분 | JSON 케이스 | 명세 문서 행 | 합계 |
|---|---:|---:|---:|
| 판정 대상 | 195 | 74 | 269 |
| 세 체인 공통 후보 | 99 | 45 | 144 |
| 그중 설정만 분리 | 54 | 20 | 74 |
| 그중 설정 분리 + 체인별 기대값 | 32 | 22 | 54 |
| 그중 별도 구현·소유 환경 필요 | 13 | 3 | 16 |
| 2체인 후보 (WEMIX4.0+StableNet) | 17 | 27 | 44 |
| 2체인 후보 (WEMIX3.0+WEMIX4.0) | 1 | 1 | 2 |
| 2체인 후보 (WEMIX3.0+StableNet) | 1 | 0 | 1 |
| 공통 제외 | 77 | 1 | 78 |

Confluence 명세 ID 단위로는 330개(관리 ID 37개 제외)를 따로 판정했다. 세 체인 공통 104개(설정 분리 38, 설정 분리+기대값 59, 별도 구현 7), 2체인 63개, StableNet 전용 105개, WEMIX4.0 전용 28개, WEMIX3.0 전용 17개, 전환 2개, 실행 테스트 아님 11개다. 같은 ID가 여러 페이지에 있으면 한 번만 세고, 2nd Change T-1~T-5는 페이지별로 센다. 표는 [Confluence 명세 ID 판정](confluence-spec-catalog.md)이다.

후보 수는 원본 항목 수다. 같은 목적의 문서 행과 JSON, 이미 이식된 사본이 각각 세어진다. 운영 메인넷 RPC에서 바로 돌릴 수 있는 수가 아니다. 실행은 하지 않았다.

## 판정 값의 뜻

- 공통 범위: `세 체인 공통` / `2체인` / `StableNet 전용` / `전환 전용`(go-wemix→go-wbft handoff) / `하네스 전용`(체인 무관 파생 검사).
- 분리 방식: `설정 분리`는 단계와 assertion을 그대로 두고 값만 체인 프로필로 빼면 되는 것. `설정 분리 + 체인별 기대값`은 값을 빼도 같은 입력의 기대 결과가 체인마다 달라 oracle이 필요한 것. `별도 구현/처리`는 하네스에 경로가 없거나 소유한 프로세스가 있어야 하는 것.

## 읽을 문서

| 문서 | 내용 |
|---|---|
| [설정 분리와 변경 범위](configuration-and-change-scope.md) | 완료 조건 1~4의 답. 의존 요소, 설정 항목, 별도 구현 항목, 변경 작업 순서 |
| [전체 테스트 목록](all-tests.md) | 269개 항목의 판정과 이유 |
| [Confluence 명세 ID 판정](confluence-spec-catalog.md) | Chainbench 폴더 페이지 29개의 테스트 ID 330개를 명세 기대 결과 기준으로 판정. PR196 우선순위와 StableNet 실행 결과 연결 |
| [공통 수행 후보](common-candidates.md) | 세 체인·2체인 후보와 각 항목의 설정/구현 목록, 공통 제외 이유 |
| [의존 필드 색인](dependency-index.md) | 195개 JSON의 계정·RPC·컨트랙트·Chain ID·포크·수수료·토폴로지·바이너리 필드 전수 |
| [클라이언트 차이](chain-differences.md) | 세 클라이언트 코드 차이 18항목(CD-A-01~09, CD-B-01~09), 근거 318건 |
| [하네스 분석](harness-dependencies.md) | chainbench 실행 구조 10항목(H-01~10), 근거 152건 |
| [빌드 기준 AST 그래프](build-ast-graphs.md) | 빌드 선택 파일과 tree-sitter 그래프, 모듈 fan-in 비교 |
| [명세 ID 대응표](existing-tc-specs.md) | 실행 ID와 Confluence 명세 ID |
| [반대 검토](counter-review.md) | 2026-09-09 판정과 달라진 17개 파일 |
| [검증 결과](validation.json) / [재분석 절차](../REPLAY.md) | 개수·해시·인용 검증과 최신 코드 재처리 방법 |

기계 판독용: `tc-catalog.json`(195), `doc-catalog.json`(74), `confluence-spec-catalog.json`(367), `tc-dependencies.json`, `tc-spec-id-crosswalk.json`, `chain-differences.json`, `harness-dependencies.json`, `citation-check.json`.

## 이번 분석에서 새로 확인한 것

1. 하네스의 `expect: reject`는 어떤 전송 오류든 통과시킨다. 거부 이유가 틀려도 PASS다. 공통 케이스는 체인별 sentinel 문자열로 판정해야 한다.
2. 케이스 12개가 fee 입력을 `istanbul_getWbftExtraInfo.gasTip`에서 만든다(읽기 24회). 이 필드는 StableNet 헤더에만 있다.
3. `regression/ethereum/23`의 계약 fixture는 PUSH0을 쓴다. go-wemix는 PUSH0을 기본 instruction set에 넣지 않았다.
4. chainbench wemix manifest의 `wemix_getValidators`, `wemix_getReward`는 go-wemix에 없는 메서드다. `wemix` namespace는 Brioche 설정이 있을 때만 열린다.
5. 7702는 go-wbft가 Croissant 높이로, go-stablenet은 anzeon 설정 존재로 gate한다. P256은 Croissant vs Boho다. 두 체인 사이에서도 preflight가 다르다.
6. StableNet에는 블록 보상이 없다. baseFee를 이전 epoch 검증자에게 배분한다.
7. Confluence 원문 대조로 판정 7건을 고쳤다. eth_gasPrice/maxPriorityFee 검사(RT-G-2-01/02)는 식이 세 클라이언트 같아 세 체인 공통으로, baseFee 상한(RT-C-07)은 WEMIX3.0에도 governance 상한이 있어 2체인으로, 바이너리 교체·genesis 불일치 케이스(TC-3-1-04, TC-4-1-03)는 목적이 공통이라 세 체인(별도 구현)으로 옮겼다. 팀의 PR196 144개 목록(WEMIX3.0 후보)과 대조하면 JSON 97개 중 88개가 일치한다.
8. 2026-09-09 판정 중 17개 파일이 바뀌었다. 대부분은 adapter-required였던 순수 조회 케이스(peerCount, sameBlockHash, eth_syncing, admin_peers 등)를 설정 분리로 옮긴 것과, StableNet 헤더 tip 정책 케이스를 StableNet 전용으로 옮긴 것이다.

## 소스 기준

| 프로젝트 | HEAD (미러 시점) | 비고 |
|---|---|---|
| go-wemix | `902f9fce85c1` | tracked 변경 없음 |
| go-wbft | `b1dc5a8a28d1` | 분석 중 다른 세션이 HEAD를 옮겼다. 이후 바뀐 파일 4개(core/blockchain*.go, rpc/client.go, rpc/websocket.go)는 인용되지 않았다 |
| go-stablenet | `0937ac5c93d4` | tracked 변경 없음 |
| chainbench | `cc501aedaab1` | tests/tc 198개 파일 해시를 `../source-manifest.json`에 기록. 분석 중 #384(`5ddf7f71`)가 들어와 `internal/app/upgrade.go`, `internal/consensus/upgrade/handoff.go` 인용 5건의 줄을 다시 맞췄다 |

## 한계

- Confluence Chainbench 폴더 하위 페이지 29개는 Atlassian MCP로 읽어 `../sources/confluence/`에 보존했다(읽기만 했고 쓰지 않았다). 2026-09-09 전사본의 ID 집합과 같았고, 그 뒤 추가된 PR196 페이지 5개와 빈 페이지 `[Common] Test`(2026-09-11 생성)가 더 있었다. 두 시나리오 명세서 Markdown(`common_test_scenarios.md`, `dual_chain_test_scenarios.md`)에 해당하는 Confluence 페이지는 이 폴더에 없다. 보존본에서 내부망 IP, bootnode enode, 원격 제어 스크립트 본문은 뺐다.
- 실행 결과 페이지(2026-04-13 StableNet 결과)는 TC-* 64건의 Pass/제외를 담고 있어 명세 ID 판정 표에 결과 열로 붙였다. WEMIX4.0·Regression 결과 페이지에는 항목별 결과가 없다.
- 노드를 띄우지 않았다. 모든 판정은 정적이다. "설정 분리"로 판정한 케이스도 세 체인에서 실제로 PASS한다는 뜻이 아니다.
- AST 그래프는 패키지 import와 선언 목록이다. 함수 호출 경로는 grep과 코드 읽기로 확인했다.
- `build-selected/` 미러는 chainbench 모듈 안에 Go 파일을 두므로 `go mod tidy`를 막았다(#384 커밋 메시지). 각 미러에 원본 `go.mod`를 넣어 별도 모듈로 분리했고, `go list ./...`가 미러를 잡지 않는 것을 확인했다.
- 운영 메인넷의 실제 fork 활성 상태, validator 수, 열린 RPC namespace는 확인하지 않았다. go-wbft mainnet 설정의 Croissant 높이는 코드에 TODO로 남아 있다.
