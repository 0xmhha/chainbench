# [Common] Test

> 출처: Confluence [[Common] Test](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987917348) (페이지 ID 2987917348, 버전 5, 최종 수정 2026-09-11)  
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

같은 목적의 테스트를 하나로 합쳐 공통 테스트 76개를 만들었다. 각 테스트에 새 ID(CT-)를 붙였고, 합쳐진 기존 ID는 비고에 모두 적었다.

| 영역 | 뜻 | 수 |
| --- | --- | --- |
| NODE | 노드·동기화·네트워크 | 16 |
| TX | 트랜잭션 전송·거부 | 20 |
| FEE | 수수료·가스 정책 | 12 |
| CONTRACT | 컨트랙트 실행 | 7 |
| RPC | 조회·구독 API | 15 |
| FAULT | 장애·복구 | 6 |

분리 방식으로 나누면 다음과 같다.

| 분리 방식 | 뜻 | 수 |
| --- | --- | --- |
| 설정으로 분리 | 절차를 그대로 두고 체인별 값만 바꾸면 된다 | 32 |
| 기대값은 체인별 계산 | 같은 입력에서 나와야 할 정답이 체인마다 달라 정답을 따로 구해야 한다 | 32 |
| 별도 구현 필요 | 실행 도구에 없는 기능이 필요하거나 노드를 직접 다뤄야 한다 | 12 |

두 체인에서만 가능한 테스트(63개)와 한 체인 전용 테스트(150개)는 부록에 기존 ID로 정리했다.

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
| [공통 테스트 목록](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2988965889) | 76개 테스트의 ID, 목적, 기대 결과, 분리 방식, 기존 ID 대응 |
| [메인넷별 의존 요소](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2988376101) | 테스트 계정, RPC 주소, 컨트랙트 주소, 체인 ID, 그 밖의 설정 |
| [별도 구현이 필요한 항목](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987884735) | 체인별 기대값 계산과 실행 도구에 추가할 기능 |
| [공통 테스트 분리 변경 범위](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987720853) | 작업 순서와 완료 기준 |
| [회귀 실행 묶음](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2986934428) | 회귀 테스트로 돌릴 때의 실행 묶음, 순서, 공유 준비물 |
| [상세 실행 절차](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987196682)(NODE, TX, FEE, CONTRACT, RPC, FAULT) | 테스트별 준비, 절차, 기대 결과, 체인별 차이 |
| [용어집](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2986868908) | 이 문서에서 쓰는 용어 |
| [부록 A. 두 체인에서만 가능한 테스트](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2986901809/A.) | 기존 ID로 정리 |
| [부록 B. 공통에서 제외한 테스트와 이유](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987524189/B.) | 기존 ID로 정리 |
