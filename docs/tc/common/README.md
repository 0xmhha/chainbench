# [Common] Test

> 출처: Confluence [[Common] Test](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987917348) (페이지 ID 2987917348, 버전 5, 최종 수정 2026-09-11)  
> 2026-09-29 확인: Confluence 는 버전 7(2026-09-28)이지만 본문은 아직 76개 기준이다. 이 사본이 더 앞서 있고, Confluence 는 이 사본으로 갱신한다.  
> 상위 페이지: Chainbench (폴더)  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김)

하위 페이지

| 파일 | Confluence 페이지 |
| --- | --- |
| [01-common-test-list.md](01-common-test-list.md) | 공통 테스트 목록 |
| [02-mainnet-dependencies.md](02-mainnet-dependencies.md) | 메인넷별 의존 요소 |
| [03-separate-implementation-items.md](03-separate-implementation-items.md) | 별도 구현이 필요한 항목 |
| [04-separation-change-scope.md](04-separation-change-scope.md) | 공통 테스트 분리 변경 범위 |
| [05-regression-execution-bundles.md](05-regression-execution-bundles.md) | 회귀 실행 묶음 |
| [06-detailed-procedures/README.md](06-detailed-procedures/README.md) | 상세 실행 절차 |
| [06-detailed-procedures/01-node.md](06-detailed-procedures/01-node.md) | 상세 실행 절차 NODE (노드·동기화·네트워크) |
| [06-detailed-procedures/02-tx.md](06-detailed-procedures/02-tx.md) | 상세 실행 절차 TX (트랜잭션 전송·거부) |
| [06-detailed-procedures/03-fee.md](06-detailed-procedures/03-fee.md) | 상세 실행 절차 FEE (수수료·가스 정책) |
| [06-detailed-procedures/04-contract.md](06-detailed-procedures/04-contract.md) | 상세 실행 절차 CONTRACT (컨트랙트 실행) |
| [06-detailed-procedures/05-rpc.md](06-detailed-procedures/05-rpc.md) | 상세 실행 절차 RPC (조회·구독 API) |
| [06-detailed-procedures/06-fault.md](06-detailed-procedures/06-fault.md) | 상세 실행 절차 FAULT (장애·복구) |
| [07-glossary.md](07-glossary.md) | 용어집 |
| [08-appendix-a-two-chain-tests.md](08-appendix-a-two-chain-tests.md) | 부록 A. 두 체인에서만 가능한 테스트 |
| [09-appendix-b-excluded-tests.md](09-appendix-b-excluded-tests.md) | 부록 B. 공통에서 제외한 테스트와 이유 |

---

## 목적

WEMIX3.0, WEMIX4.0, StableNet 세 체인에서 같은 목적으로 실행할 수 있는 테스트를 가려내고, 그 테스트가 어느 메인넷 값에 의존하는지 밝힌다. 그리고 공통 테스트로 분리하려면 무엇을 설정으로 빼고 무엇을 따로 구현해야 하는지 정리한다.

이 페이지는 요약이다. 근거와 상세 내용은 하위 페이지에 있다.

## 어떻게 정리했나

기존 테스트 자료 세 가지를 모두 읽었다.

1. Confluence의 StableNet, WEMIX4.0, WEMIX3.0 테스트 명세 페이지 (테스트 ID 330개)
2. 실행 도구(chainbench)에 정의된 자동 테스트 195개
3. 세 체인의 노드 프로그램 소스 (현재 기준)

테스트마다 "세 체인에서 같은 목적으로 쓸 수 있는가"를 판정했다. 판정 기준은 명세에 적힌 기대 결과와 노드 프로그램의 실제 동작이다.

## 결과 요약

같은 목적의 테스트를 하나로 합쳐 공통 테스트 70개를 만들었다. 각 테스트에 새 ID(CT-)를 붙였고, 합쳐진 기존 ID는 비고에 모두 적었다. 70개 모두 세 체인에서 도는 자동 테스트가 있다.

| 영역 | 뜻 | 수 |
| --- | --- | --- |
| NODE | 노드·동기화·네트워크 | 15 |
| TX | 트랜잭션 전송·거부 | 20 |
| FEE | 수수료·가스 정책 | 7 |
| CONTRACT | 컨트랙트 실행 | 7 |
| RPC | 조회·구독 API | 15 |
| FAULT | 장애·복구 | 6 |

분리 방식으로 나누면 다음과 같다.

| 분리 방식 | 뜻 | 수 |
| --- | --- | --- |
| 설정으로 분리 | 절차를 그대로 두고 체인별 값만 바꾸면 된다 | 33 |
| 기대값은 체인별 계산 | 같은 입력에서 나와야 할 정답이 체인마다 달라 정답을 따로 구해야 한다 | 26 |
| 별도 구현 필요 | 실행 도구에 없는 기능이 필요하거나 노드를 직접 다뤄야 한다 | 11 |

두 체인에서만 가능한 테스트(64개)와 한 체인 전용 테스트(152개)는 부록에 기존 ID로 정리했다.

## 세 체인이 갈리는 곳

공통 테스트가 의존하는 메인넷 값은 다섯 가지다. 자세한 내용은 [메인넷별 의존 요소](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2988376101) 페이지에 있다.

1. **테스트 계정**: 노드가 갖고 있는 계정으로 서명하는 테스트는 우리가 띄운 네트워크에서만 된다. StableNet은 계정에 차단·인증 상태가 있어 같은 송금이 거부될 수 있다.
2. **RPC 주소와 API**: 합의 정보 조회 API가 WEMIX3.0에는 없고, 관리용 API는 운영망에서 닫혀 있을 수 있다.
3. **컨트랙트 주소**: 일반 컨트랙트는 배포해서 쓰면 공통이다. 시스템 컨트랙트는 주소, 함수, 의미가 체인마다 달라 공통이 아니다.
4. **체인 ID**: WEMIX3.0과 WEMIX4.0의 코드 기본값이 같다. 체인 ID만으로 어느 체인인지 판단할 수 없다.
5. **그 밖의 설정**: 수수료 규칙(기본 수수료 변화, 최소 수수료), 하드포크 이름, 블록 주기, 노드 프로그램 이름이 다르다.

## 하위 페이지

| 페이지 | 내용 |
| --- | --- |
| [공통 테스트 목록](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2988965889) | 70개 테스트의 ID, 목적, 기대 결과, 분리 방식, 기존 ID 대응, 자동 테스트 |
| [메인넷별 의존 요소](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2988376101) | 테스트 계정, RPC 주소, 컨트랙트 주소, 체인 ID, 그 밖의 설정 |
| [별도 구현이 필요한 항목](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987884735) | 체인별 기대값 계산과 실행 도구에 추가할 기능 |
| [공통 테스트 분리 변경 범위](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987720853) | 작업 순서와 완료 기준 |
| [회귀 실행 묶음](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2986934428) | 회귀 테스트로 돌릴 때의 실행 묶음, 순서, 공유 준비물 |
| [상세 실행 절차](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987196682)(NODE, TX, FEE, CONTRACT, RPC, FAULT) | 테스트별 준비, 절차, 기대 결과, 체인별 차이 |
| [용어집](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2986868908) | 이 문서에서 쓰는 용어 |
| [부록 A. 두 체인에서만 가능한 테스트](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2986901809/A.) | 기존 ID로 정리 |
| [부록 B. 공통에서 제외한 테스트와 이유](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987524189/B.) | 기존 ID로 정리 |

---

## 2026-09-28 개정

세 체인의 노드 프로그램을 직접 읽어 공통 판정을 다시 했다. `header.GasTip()`,
`params.MinBaseFee`, `params.InitialGasTip` 은 StableNet 에만 있고, 같은
`istanbul_getWbftExtraInfo` API 를 가진 WEMIX4.0 도 응답에 `gasTip` 을 담지 않는다.

- **CT-FEE-006·CT-FEE-011 을 목록에서 빼 부록 B 로 옮겼다.** 76개 → 74개.
- CT-TX-002·003·016 과 CT-FEE-002·010 은 목록에 남기고, 자동 테스트 중 무엇이 StableNet
  전용인지 비고에 적었다. 검사하려는 동작 자체는 세 체인 공통이다.
- CT-NODE-016 은 블록 주기 1초를 WEMIX3.0 이 거버넌스로 정한다는 점을 비고에 적었다.

저장소 쪽 반영과 케이스별 내역은
[`tests/tc/common/README.md`](../../../tests/tc/common/README.md) §4 에 있다.

## 2026-09-28 개정 (2차)

1차 개정은 노드 프로그램을 읽어서 했다. 그 뒤 공통 91건을 세 체인에 각각 돌려 보니
읽어서는 보이지 않던 것이 나왔다. WEMIX3.0 의 EVM 은 London 세대에서 멈춰 있어 Shanghai 가
들여온 PUSH0 옵코드가 없다. StableNet 의 anzeon 과 WEMIX4.0 의 croissant 는 각자 그 옵코드를
켠다. 요즘 컴파일러는 별도 지정이 없으면 PUSH0 를 쓰는 바이트코드를 내므로, 컨트랙트를
배포하는 테스트가 WEMIX3.0 에서만 배포 단계부터 실패한다.

- **CT-CONTRACT-002·CT-CONTRACT-005 를 목록에서 빼 부록 B 로 옮겼다.** 74개 → 72개.
  CONTRACT 영역이 7개에서 5개가 되고, 분리 방식별로는 기대값을 체인별로 계산하는 것이
  30개에서 28개가 된다.
- 부록 B 에 "WEMIX3.0 의 EVM 세대 때문에 뺀 것" 절을 더하고, 자동 테스트 두 건도
  "제외한 자동 테스트" 표에 넣었다.
- 옵코드를 낮춰 다시 컴파일하면 세 체인에서 돈다. 그래도 공통으로 세지 않기로 한 것은,
  요즘 컴파일러가 기본으로 내는 컨트랙트를 실행하지 못하는 것 자체가 체인 차이여서다.

## 2026-09-28 개정 (3차)

세 체인 실행을 이어 가다 기본 수수료 규칙도 갈린다는 것을 확인했다. StableNet 의 anzeon
은 상승 문턱과 하강 문턱 두 개를 쓰고, WEMIX3.0 과 WEMIX4.0 은 표준 EIP-1559 대로 블록
한도의 절반 하나를 목표로 삼는다. 세 테스트가 쓰는 25% 채우기가 StableNet 에서는 상승
문턱을 넘지만 다른 둘에서는 목표에 못 미쳐 반대로 내려간다.

- **CT-FEE-003·004·005 를 목록에서 빼 부록 B 로 옮겼다.** 72개 → 69개. FEE 영역이
  10개에서 7개가 되고, 기대값을 체인별로 계산하는 것이 28개에서 25개가 된다.
- 부록 B 에 "StableNet 의 기본 수수료 규칙 때문에 뺀 것" 절을 더하고, 자동 테스트 세 건도
  "제외한 자동 테스트" 표에 넣었다.
- 셋을 함께 뺀 이유가 있다. 감소 테스트는 WEMIX4.0 에서 **통과했다.** 다만 재려던 것과
  다른 이유였다. 표준 규칙에서 25% 채우기는 부하가 아니라 목표 미달이라, 부하를 거는
  동안에도 이미 내려가고 있었다. 이 하나만 남기면 아무것도 검증하지 않는 테스트가 목록에
  남는다.

## 2026-09-29 개정

목록을 자동 테스트와 한 줄씩 대조했다. 목록에는 있는데 세 체인 공통 자동 테스트가 없는 것이 14개였다. 그중 12개는 자동화된 적이 없거나 한 체인 전용으로만 있었고, 2개(CT-TX-002·003)는 StableNet 전용 `gasTip` 에 기대어 공통에서 빠져 있었다. 세 노드 프로그램을 다시 읽어 하나씩 판정했다.

- **70개 모두 공통 자동 테스트가 생겼다.** 새로 만든 것은 CT 16개, 케이스 17개다. 세 체인(Docker)에서 돌려 통과한 것만 목록에 적었다. 무엇이 새것이고 무엇이 전부터 있던 것인지는 공통 테스트 목록의 비고에 적었다.
- **두 부분은 공통이 아니라 뺐다.** CT-FEE-002 의 "최소 가스비와 같은 값은 받아들여진다" 는 경계를 세 체인이 다르게 정하고 다른 시점에 검사한다(부록 B). CT-TX-011 의 "대납자 키로 서명할 때 형식이 맞지 않으면 거부" 는 WEMIX3.0 에 그 검사가 없다(부록 A). 두 CT 는 나머지 부분으로 목록에 남는다.
- **CT-NODE-012 는 부록 B 로 옮겼다.** 체인마다 교체할 두 번째 빌드가 있어야 성립한다. 70개는 이것을 뺀 수다. CT-CONTRACT-002·005 는 PUSH0 없는 컨트랙트로 다시 써서 2026-09-28 에 목록으로 되돌렸다.
- **실행 도구에 기능 셋을 더했다.** 로컬 키로 legacy·접근 목록 형식을 보내는 것, RPC 호출이 정해진 오류로 실패해야 한다는 검사, 생산하지 않는 노드를 genesis 상태로 되돌리는 것(snap 동기화용)이다.

케이스별 판정과 근거는 저장소의 `docs/research/common-tests/doc-vs-tests-20260929.md` 에 있다.

## 이 저장소가 더한 것 (Confluence 본문 아님)

> 위 본문은 Confluence 를 그대로 옮긴 것이고, 이 절만 chainbench 저장소가 더했다.

[03-separate-implementation-items.md](03-separate-implementation-items.md) §2 "실행 도구에
추가해야 하는 것" 여덟 가지가 실제로 어디서 추적되는지는
[`docs/dev/architecture/mainnet-config-worklist.md` §8](../../dev/architecture/mainnet-config-worklist.md)
의 대응표에 있다. 여섯은 R2~R6 이 덮고, **동기화 경로 관찰과 준비물 공유 둘은 아직 어느
항목도 아니다.**
