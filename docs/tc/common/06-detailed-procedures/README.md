# 상세 실행 절차

> 출처: Confluence [상세 실행 절차](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987196682) (페이지 ID 2987196682, 버전 3, 최종 수정 2026-09-11)  
> 상위 페이지: [Common] Test  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김)

하위 페이지

| 파일 | Confluence 페이지 |
| --- | --- |
| [01-node.md](01-node.md) | 상세 실행 절차 NODE (노드·동기화·네트워크) |
| [02-tx.md](02-tx.md) | 상세 실행 절차 TX (트랜잭션 전송·거부) |
| [03-fee.md](03-fee.md) | 상세 실행 절차 FEE (수수료·가스 정책) |
| [04-contract.md](04-contract.md) | 상세 실행 절차 CONTRACT (컨트랙트 실행) |
| [05-rpc.md](05-rpc.md) | 상세 실행 절차 RPC (조회·구독 API) |
| [06-fault.md](06-fault.md) | 상세 실행 절차 FAULT (장애·복구) |

---

공통 테스트 76개의 준비, 절차, 기대 결과, 체인별 차이를 영역별 하위 페이지에 적는다. 목록과 ID 규칙은 [공통 테스트 목록](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2988965889) 페이지에 있다.

| 영역 | 뜻 | 테스트 수 | 페이지 |
| --- | --- | --- | --- |
| NODE | 노드·동기화·네트워크 | 16 | [상세 실행 절차 NODE](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2988539943/NODE) |
| TX | 트랜잭션 전송·거부 | 20 | [상세 실행 절차 TX](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987720880/TX) |
| FEE | 수수료·가스 정책 | 12 | [상세 실행 절차 FEE](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987819122/FEE) |
| CONTRACT | 컨트랙트 실행 | 7 | [상세 실행 절차 CONTRACT](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987786466/CONTRACT) |
| RPC | 조회·구독 API | 15 | [상세 실행 절차 RPC](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2988638234/RPC+API) |
| FAULT | 장애·복구 | 6 | [상세 실행 절차 FAULT](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987524207/FAULT) |

각 테스트는 같은 틀로 적는다.

- **목적**: 무엇을 확인하는지
- **의존 요소**: 테스트 계정, RPC 주소·API 노출, 컨트랙트, 체인 ID, 하드포크 활성, 수수료 값, 노드 구성·시간, 노드 프로그램, 노드 직접 제어 가운데 해당하는 것
- **분리 방식**: 설정으로 분리 / 기대값은 체인별 계산 / 별도 구현 필요
- **절차**와 **기대 결과**: 세 체인이 같이 따르는 부분
- **체인별 차이**: 값이나 규칙이 다른 곳. 그 값은 실행 설정(프로필)에 둔다
- **비고**: 기존 명세 ID, 자동 테스트 이름, PR196 우선순위
