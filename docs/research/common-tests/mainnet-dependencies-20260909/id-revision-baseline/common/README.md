# 세 체인 공통 테스트와 메인넷 의존 분석

두 시나리오 문서와 tests/tc를 전수 정리했다. **원본 263개 중 go-wemix 수행 후보는 144개, 세 체인 공통 후보는 138개**다. 소스와 테스트는 수정하거나 실행하지 않았다. 실제 빌드 선택 파일의 AST 그래프를 만든 뒤 소스 경로와 테스트 기대값을 대조했다.

| 구분 | 개수 | 의미 |
|---|---:|---|
| common_test_scenarios.md | 45행 | 원본 명세 행 전체 |
| dual_chain_test_scenarios.md | 29행 | 원본 명세 행 전체 |
| tests/tc | 189개 | 모든 JSON 파일. 설명용 Markdown 3개는 별도 |
| 전체 원본 | 263개 | 문서와 JSON의 중복을 보존한 항목 수 |
| go-wemix 후보 | 144개 | 기존 wemix JSON 5개, 이식 132개, 조건부 7개 |
| 세 체인 공통 후보 | 138개 | 문서 46행과 JSON 92개 |
| 설정 분리 중심 | 42개 | 공통 의미의 assertion을 유지하고 환경·입력을 분리 |
| 별도 구현·처리 필요 | 96개 | 명세를 실행 테스트로 구현하거나 체인별 처리·판정 추가 |
| 공통 제외 | 125개 | 체인 전용 124개, 외부 faucet 미확인 1개 |

138개는 독립 실행 횟수가 아니며 운영 메인넷 RPC에서 즉시 실행 가능한 테스트 수가 아니다. 96개에 서로 다른 어댑터 96개를 만들자는 뜻도 아니다. 같은 기능의 문서와 JSON은 원본 ID를 유지하고 공통 family로 연결했다. 공개 RPC와 소유한 격리 네트워크는 실행 권한·서명·기능 노출이 달라 별도 프로필로 다룬다.

## 읽을 문서

| 문서 | 내용 |
|---|---|
| [전체 테스트 목록](all-tests.md) / [JSON](all-tests.json) | 263개 원본, 실제 기대값, 체인별 적용, 항목별 의존 필드 |
| [go-wemix 수행 후보](wemix-candidates.md) | 144개와 이식·조건부 사유 |
| [세 체인 공통 후보](common-candidates.md) | 138개 전량, 설정 분리와 별도 처리 구분 |
| [공통 기능별 대응](common-families.md) | 원본 ID와 공통 검증 목적·범위 |
| [메인넷 의존과 변경 범위](configuration-and-change-scope.md) | 요청한 설정 분리·별도 구현·변경 범위를 별도로 정리한 문서 |
| [의존 필드 색인](dependency-index.md) | 계정/RPC/계약/Chain ID 등 JSON 내 정확한 위치와 literal·binding 구분 |
| [세 클라이언트 차이](chain-differences.md) | 수수료, 계정 정책, 포크, RPC, 계약, 합의 차이와 코드 근거 |
| [하네스 실행 구조](harness-dependencies.md) | 이미 있는 공통 기능, 하드코딩, 설정 한계, 추가 구현 지점 |
| [빌드 기준 AST 그래프](build-ast-graphs.md) / [다이어그램](build-diagrams.md) | 세 프로젝트와 두 OS/arch의 실제 선택 파일·그래프 |
| [반대 검토](counter-review.md) | 오분류 4개와 수수료 하한 기대값 교정 |
| [검증 자료](validation.json) / [재분석 절차](../REPLAY.md) | 개수·해시·빌드 포함·인용 검증 및 최신 코드 재처리 방법 |

## 공통 테스트 분리에서 중요한 차이

1. **계정, RPC, 일반 계약 주소, 바이너리, Chain ID는 설정으로 분리할 수 있다.** 현재 EnvV2, 계정 label, 계약 배포 결과 binding을 우선 재사용한다. env extends는 얕은 병합이므로 중첩 설정 전체 교체를 주의한다.
2. **수수료와 합의의 기대값은 체인별 코드가 필요하다.** WEMIX governance, WBFT EIP-1559, StableNet threshold/header tip 규칙을 같은 식으로 판정하면 오류가 된다. WBFT에 없는 고정 baseFee 상한 시험은 세 체인 공통에서 제외했다.
3. **공개 RPC 전환에는 서명 경로 보강이 필요하다.** 현재 deploy/load/faucet/registerContract는 node-signed 경로이며 계정 label만 바꾸면 로컬 서명이 되는 구조가 아니다. 기존 Wallet과 sendTx를 공통 sender로 정리하는 범위를 제안했다.
4. **일반 계약과 시스템 계약은 다르게 분리한다.** 일반 계약은 공통 fixture와 주소 binding으로 공유한다. 거버넌스·보상·native coin 계약은 ABI·권한·상태 의미가 달라 전용 처리로 남긴다.
5. **7702와 P256은 세 체인 공통 성공 테스트가 아니다.** WEMIX3 미지원뿐 아니라 WBFT와 StableNet의 활성 fork도 다르다. 지원 선언이나 주소의 빈 응답을 기능 성공으로 세지 않는다.

go-wemix 후보 중 세 체인 공통으로 합치지 않은 6개는 Brioche 문서/JSON 2개, 전환·혼합 바이너리 2개, faucet 1개, 고정 baseFee 상한 1개다. WEMIX에서 조건을 갖추어 수행할 수 있다는 것과 세 구현에 같은 목적의 검사를 적용할 수 있다는 것은 별개의 판정이다.

## 네트워크 값과 빌드 이름의 구분

| 비교 대상 | 코드의 mainnet Chain ID | chainbench manifest 기본 Chain ID | 실제 Makefile 바이너리 |
|---|---:|---:|---|
| WEMIX3.0 / go-wemix | 1111 | 8285 | gwemix |
| WEMIX4.0 / go-wbft | 1111 | 8284 | gwemix |
| StableNet / go-stablenet | 8282 | 8283 | gstable |

사용자 경로와 코드 구조를 위 명칭으로 대응한 비교이며 운영 배포 상태를 조회한 결과는 아니다. 실제 RPC·인증·운영 계정·계약 주소·활성 fork는 실행 프로필에서 확보해야 한다. WEMIX3/4의 동일한 Chain ID만으로 구현체를 선택하지 않는다. 근거: `sources/go-wemix/params/config.go:146`, `sources/go-wbft/params/config.go:47`, `sources/go-stablenet/params/config.go:45`, 각 하네스 manifest의 chain_id(`sources/chainbench/internal/chains/wemix/manifest.json:4`, `sources/chainbench/internal/chains/wbft/manifest.json:4`, `sources/chainbench/internal/chains/stablenet/manifest.json:4`).

go-wbft의 하네스 manifest는 binary/make_target를 gwbft로 기록하지만 제공된 저장소의 실제 타깃은 gwemix다. 현재 하네스는 자동 빌드하지 않으므로 “잘못된 make를 실행한다”는 오류로 해석하지 않았다. 실제 바이너리 경로 지정과 manifest 메타데이터 정리를 별도 작업으로 적었다. 근거는 하네스 분석 H06에 있다.

## 소스 기준과 한계

| 프로젝트 | 보존한 HEAD |
|---|---|
| go-wemix | `902f9fce85c108cf24cdeb77bc70a622049748ce` |
| go-wbft | `7af50e45db4a48a2a77e565c7ff7bde0e8f95505` |
| go-stablenet | `0937ac5c93d4f56ded5b24899f381d3d1f208c02` |
| chainbench | `d3796559bda31e8169321f59c0a6c1a044063033` + 수집 시 tracked 작업 변경 |

세 클라이언트는 tracked 변경이 없는 상태의 파일을 수집했다. chainbench는 다른 작업의 변경이 있어 HEAD만으로 재현할 수 없으므로 파일별 해시와 보존본을 기준으로 했다. 두 문서와 tests/tc는 이전 검토 입력과 같지만 하네스 코드는 현재 작업 파일 기준으로 다시 읽었다.

AST 입력은 darwin/arm64 기준 go-wemix 630개, go-wbft 685개, go-stablenet 668개 Go 파일이다. Linux/amd64도 별도로 추출했다. 패키지 의존성과 선언 그래프이며 완전한 함수 호출/동적 실행 그래프는 아니다. 소스 및 테스트를 바꾸거나 노드를 기동하지 않았으며 런타임 PASS를 주장하지 않는다. 소스 해시와 수집 시각은 `../source-manifest.json`에 있다.

최종 해시 검사에서 분석 중 변경된 chainbench 파일 7개를 확인했다. 세 클라이언트와 원본 테스트는 변경되지 않았다. 하네스의 변경 위치와 재확인 범위는 [동시 작업 변경 기록](concurrent-changes.md)에 있다. 이 보고서는 수집 시점의 보존본 기준이다.

검증 결과: 원본 263개와 공통 138개/go-wemix 144개의 대응, 6종 빌드 프로필의 선택 파일과 AST 입력 일치, 원본 파일 6,017개의 보존 해시, 코드 인용, Mermaid 구조 검사를 통과했다. tree-sitter 구문 오류 파일은 0개다. 실제 바이너리 링크와 테스트 실행은 하지 않았다.

기존 TC 번호와 실행 ID는 [기존 TC 스펙 및 ID 대응표](existing-tc-specs.md)에 별도로 정리했다. TC-001 등 분석 별칭은 원래 명세 번호가 아니다.
