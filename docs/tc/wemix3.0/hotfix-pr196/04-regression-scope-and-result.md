# [PR196] 회귀 테스트 수행 범위 및 결과 기록

> 출처: Confluence [[PR196] 회귀 테스트 수행 범위 및 결과 기록](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2977988792) (페이지 ID 2977988792, 버전 3, 최종 수정 2026-09-10)  
> 상위 페이지: [WEMIX3.0] Hotfix(PR-196) 테스트 항목  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김. 표 안의 링크는 모두 [02-focused-tests-detailed-procedure.md](02-focused-tests-detailed-procedure.md)에 해당한다)

---

## 기록 원칙

이 페이지는 집중 테스트 28개의 수행 범위와 실행 결과를 기록한다. 테스트 내용과 기대 결과는 집중 테스트 요약 및 상세 실행 절차를 기준으로 한다. 현재 28개 모두 미실행이며, 아래 표는 결과를 입력하기 위한 양식이다.

28개는 집중 테스트의 항목 수다. 컨트랙트 배포, 상태 변경 및 이벤트는 네 가지 트랜잭션 유형을 각각 실행하므로 실제 실행 횟수와 같지 않다. 신규 테스트는 기존 문서의 제목으로 구분한다. PR196-TC-003은 기존 테스트 wemix-node-crash에 부여한 관리 ID다. 두 집중 문서의 기존 표기는 그대로 유지한다.

## 실행 기준

| 기록 항목 | 값 |
| --- | --- |
| 패치 적용 전 버전(전체 커밋 SHA) | 미기록 |
| 패치 적용 후 통합 버전(전체 커밋 SHA) | 미확정 |
| 실행 회차 및 수행자 | 미기록 |
| 체인 ID와 제네시스 설정 | 미기록 |
| 활성 하드포크와 수수료 규칙 | 미기록 |
| 노드별 버전·역할·RPC 주소 | 미기록 |
| 블록 크기·메시지 크기 제한 및 블록 생성 설정 | 미기록 |
| 테스트 계정·자금·컨트랙트 정보 | 미기록. 비밀키와 인증정보는 기록하지 않음 |

## 판정 기준

통과: 필수 세부 조건을 모두 실행했고 실제 결과가 기대 결과와 일치한다. 실패: 실제 결과가 기대 결과와 다르다. 판정 보류: 환경이나 적용 정책을 확인할 수 없거나 근거 자료가 부족하다. 미실행: 아직 실행하지 않았다. 일부 조건만 실행한 항목은 전체 통과로 기록하지 않는다.

실제 결과에는 측정값과 관찰한 동작을 적는다. 근거에는 실행 명령 또는 절차, RPC 응답, 트랜잭션·블록 해시, receipt, 상태 비교 결과와 로그의 보관 위치를 연결한다. 실패 항목에는 원인, 관련 이슈 및 재실행 결과를 함께 남긴다. 재실행 시 이전 결과를 삭제하지 않고 실행 회차를 구분한다.

## 집중 테스트 실행 결과

현재 집계: 전체 28개, 미실행 28개, 통과 0개, 실패 0개, 판정 보류 0개. 실행 결과를 입력할 때 이 집계도 함께 갱신한다. 아래 기대 결과는 요약 표를 옮긴 것이며, 최종 판정에는 연결된 상세 절차의 모든 조건을 적용한다.

| 테스트 ID | 테스트 및 상세 절차 | 우선순위 | 기대 결과 요약 | 실제 결과 | 판정 | 근거·관련 이슈 |
| --- | --- | --- | --- | --- | --- | --- |
| RT-A-2-04 / TX-010 | [계정별 nonce 순서 보장](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P0 | 송신 계정별 nonce 순서대로 포함되며 트랜잭션 해시의 중복이나 누락이 없음 | 미기록 | 미실행 | 미등록 |
| RT-A-1-02 / NODE-003 | [전체 동기화(Full Sync)](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P0 | Full Sync 수행 확인, 동일 높이의 블록 해시와 stateRoot 일치 | 미기록 | 미실행 | 미등록 |
| RT-A-1-06 | [Downloader를 통한 누락 블록 동기화](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P0 | Downloader 처리 기록 확인, 같은 높이 H에서 블록과 상태 일치 | 미기록 | 미실행 | 미등록 |
| PR196-TC-003 | [노드 중단 후 블록 생성 및 복구 (wemix-node-crash)](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 다른 노드의 블록 생성 유지. 복구 노드와 기준 노드의 같은 높이 H에서 블록 해시·stateRoot가 일치하고 후속 블록 반영 | 미기록 | 미실행 | 미등록 |
| RT-A-2-02 | [동적 수수료 트랜잭션(type 0x2) 정상 처리](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | type=0x2, status=1. nonce·전송액·수수료 차감이 일치하고 effectiveGasPrice가 해당 블록의 WEMIX3.0 수수료 규칙과 일치 | 미기록 | 미실행 | 미등록 |
| RT-A-2-06 | [잔액 부족 트랜잭션 거부](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 잔액 부족 오류로 거부되며 블록·트랜잭션 풀에 미포함. 송신자 잔액과 nonce는 유지되고 블록 생성은 계속됨 | 미기록 | 미실행 | 미등록 |
| RT-A-3-01 / TX-005 | [트랜잭션 유형별 컨트랙트 배포와 반환값 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 네 유형 모두 status=1. 예상 런타임 코드·반환값 42·nonce·수수료 처리 일치 | 미기록 | 미실행 | 미등록 |
| RT-A-3-06 | [revert된 트랜잭션의 receipt 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 배포 status=1, 호출 status=0 | 미기록 | 미실행 | 미등록 |
| RT-A-3-07 | [가스 소진 트랜잭션의 실패 처리](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 블록에 포함되고 status=0, gasUsed=50,000. 실패 원인은 가스 소진 | 미기록 | 미실행 | 미등록 |
| RT-D-01 | [수수료 대납 트랜잭션(type 0x16) 정상 처리](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | type=0x16, status=1, 송신액·대납 비용 일치 | 미기록 | 미실행 | 미등록 |
| RT-A-1-03 / NODE-004 | [스냅 동기화(Snap Sync)](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | Snap Sync 수행 확인, 같은 높이 H의 stateRoot와 잔액 일치. Full Sync로 전환된 경우는 제외 | 미기록 | 미실행 | 미등록 |
| 신규 테스트 | [미포함 트랜잭션의 후속 블록 처리와 교체](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 보관 조건을 충족한 트랜잭션이 유지되고 nonce 순서대로 포함됨. 교체된 nonce에는 새 트랜잭션만 포함 | 미기록 | 미실행 | 미등록 |
| 신규 테스트 | [거부된 트랜잭션과 EVM 실행 실패의 상태 및 크기 처리](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 적용 단계에서 거부되면 상태 롤백. 블록에 포함된 실행 실패 트랜잭션은 크기와 receipt에 반영 | 미기록 | 미실행 | 미등록 |
| 신규 테스트 | [블록 생성 작업 변경 후 누적 크기와 상태 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 복사한 작업의 처리 결과가 독립적으로 유지되고 새 작업의 크기가 초기화됨. 유효 트랜잭션은 후속 블록에 포함 | 미기록 | 미실행 | 미등록 |
| 신규 테스트 | [트랜잭션 유형과 디코딩 방식별 RLP 크기 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 같은 트랜잭션의 해시와 실행 결과가 일치하며 실제 블록 크기가 8 MiB 이하 | 미기록 | 미실행 | 미등록 |
| 신규 테스트 | [블록 본문 응답의 전체 메시지 크기 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 전체 메시지가 10 MiB 이하이며 응답에서 빠진 본문도 재요청으로 수신 | 미기록 | 미실행 | 미등록 |
| 신규 테스트 | [대량 이벤트 로그의 receipt 응답과 Snap Sync 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 연결 해제와 재시도만 반복되지 않고 동기화 완료. 상한에 도달할 수 없으면 가능한 최대 크기와 근거 기록 | 미기록 | 미실행 | 미등록 |
| 신규 테스트 | [크기 제한 초과 블록 거부 후 정상 블록 처리](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 거부된 블록은 정규 체인에 반영되지 않으며 정상 노드와 블록 및 stateRoot 일치 | 미기록 | 미실행 | 미등록 |
| 신규 테스트 | [블록 크기 제한과 기존 생성 종료 조건 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 각 처리 방식의 제한을 준수하며 유효한 블록 생성 | 미기록 | 미실행 | 미등록 |
| RT-A-2-01 / TX-006 | [Legacy 트랜잭션(type 0x0) 전송과 실행 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | type=0x0, status=1, 수신액과 수수료 일치 | 미기록 | 미실행 | 미등록 |
| RT-A-2-03 / TX-007 | [Access List 트랜잭션(type 0x1) 전송과 실행 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | type=0x1, status=1. 비어 있지 않은 Access List와 실행 결과·nonce·가스 비용 일치 | 미기록 | 미실행 | 미등록 |
| RT-A-3-02 | [트랜잭션 유형별 컨트랙트 상태 변경 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | 각 유형 status=1, 저장값·nonce·비용 일치 | 미기록 | 미실행 | 미등록 |
| RT-A-4-04 / RPC-014 | [컨트랙트 실행과 이벤트 로그 확인](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P1 | status=1, 로그의 주소·topics·data·트랜잭션 해시 일치 | 미기록 | 미실행 | 미등록 |
| BRIOCHE-02 / RPC-008 (부분 대응) | [Brioche 블록 보상 조회](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P2 | reward(5)\>0, reward(15)\<reward(5). 기존 테스트의 체인 설정을 사용한 경우에 적용 | 미기록 | 미실행 | 미등록 |
| RT-A-3-04 | [eth\_estimateGas 가스 추정](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P2 | status=1, gasUsed\<=추정값 | 미기록 | 미실행 | 미등록 |
| RT-G-4-02 / RPC-018 | [트랜잭션 풀의 pending/queued 건수 조회](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P2 | 준비한 트랜잭션 목록과 계정 상태를 기준으로 산출한 건수와 일치 | 미기록 | 미실행 | 미등록 |
| RT-G-4-03 | [트랜잭션 풀의 상세 목록 조회](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P2 | 송신자, nonce, 트랜잭션 해시와 pending/queued 분류가 일치 | 미기록 | 미실행 | 미등록 |
| RT-A-4-05 / RPC-013 | [체인 ID 조회](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2979070094/PR196+28) | P3 | 모든 노드의 응답이 지정한 chainId와 일치 | 미기록 | 미실행 | 미등록 |

## 컨트랙트 테스트의 유형별 결과

아래 12개 행은 집중 테스트 3개의 세부 결과다. 별도 테스트 항목 수로 중복 집계하지 않는다. 각 행의 실제 결과에 type, status, nonce, 잔액과 수수료 변화를 기록하고, 배포 코드·저장값·이벤트 등 해당 절차의 검증값도 남긴다.

| 테스트 ID | 테스트 | type | 실제 결과 | 판정 | 근거 |
| --- | --- | --- | --- | --- | --- |
| RT-A-3-01 / TX-005 | 트랜잭션 유형별 컨트랙트 배포와 반환값 확인 | 0x0 | 미기록 | 미실행 | 미등록 |
| RT-A-3-01 / TX-005 | 트랜잭션 유형별 컨트랙트 배포와 반환값 확인 | 0x1 | 미기록 | 미실행 | 미등록 |
| RT-A-3-01 / TX-005 | 트랜잭션 유형별 컨트랙트 배포와 반환값 확인 | 0x2 | 미기록 | 미실행 | 미등록 |
| RT-A-3-01 / TX-005 | 트랜잭션 유형별 컨트랙트 배포와 반환값 확인 | 0x16 | 미기록 | 미실행 | 미등록 |
| RT-A-3-02 | 트랜잭션 유형별 컨트랙트 상태 변경 확인 | 0x0 | 미기록 | 미실행 | 미등록 |
| RT-A-3-02 | 트랜잭션 유형별 컨트랙트 상태 변경 확인 | 0x1 | 미기록 | 미실행 | 미등록 |
| RT-A-3-02 | 트랜잭션 유형별 컨트랙트 상태 변경 확인 | 0x2 | 미기록 | 미실행 | 미등록 |
| RT-A-3-02 | 트랜잭션 유형별 컨트랙트 상태 변경 확인 | 0x16 | 미기록 | 미실행 | 미등록 |
| RT-A-4-04 / RPC-014 | 컨트랙트 실행과 이벤트 로그 확인 | 0x0 | 미기록 | 미실행 | 미등록 |
| RT-A-4-04 / RPC-014 | 컨트랙트 실행과 이벤트 로그 확인 | 0x1 | 미기록 | 미실행 | 미등록 |
| RT-A-4-04 / RPC-014 | 컨트랙트 실행과 이벤트 로그 확인 | 0x2 | 미기록 | 미실행 | 미등록 |
| RT-A-4-04 / RPC-014 | 컨트랙트 실행과 이벤트 로그 확인 | 0x16 | 미기록 | 미실행 | 미등록 |

## 최종 결과 요약

| 완료 확인 항목 | 현재 상태 | 완료 시 남길 근거 |
| --- | --- | --- |
| 수행 범위 확정 | 집중 테스트 28개로 확정 | 집중 테스트 28개와 상세 절차의 필수 세부 조건을 수행 범위로 적용 |
| 집중 테스트 28개 실행 완료 | 미완료 | 집중 테스트 28개와 필수 세부 조건의 실제 결과 |
| 실패·보류 항목 정리 | 아직 실행 결과 없음 | 실패 원인, 관련 이슈, 재실행 결과 및 남은 영향 |
| 결과 문서 정리 | 양식 작성 완료, 실제 결과 미기록 | 실행 버전·환경·판정·근거 자료 및 최종 결론 |

테스트 수행 완료와 패치 검증 통과는 구분한다. 모든 테스트를 실행했더라도 실패가 있으면 패치 검증 통과로 표시하지 않는다. 현재는 테스트를 실행하지 않았으므로 최종 검증 결론은 미정이다. 상세 페이지의 댓글 중 유사 트랜잭션 테스트 통합 여부, RT-G-4-02의 건수 측정 중 상태 유지 방법, N-008의 queued 상태에서 수수료를 올려 교체하는 절차는 아직 확정되지 않았다. 실행 전에 조건을 구체화하고 근거를 남긴다. N-008에서는 크기 제한에 따른 이월과 nonce 누락으로 만든 queued 상태의 교체를 구분해 기록한다.
