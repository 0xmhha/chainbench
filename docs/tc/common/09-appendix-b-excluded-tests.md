# 부록 B. 공통에서 제외한 테스트와 이유

> 출처: Confluence [부록 B. 공통에서 제외한 테스트와 이유](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987524189) (페이지 ID 2987524189, 버전 7, 최종 수정 2026-09-28)  
> 상위 페이지: [Common] Test  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김)

---

한 체인에만 있는 기능이거나, 같은 기능이라도 규칙이 갈려 정답이 하나로 모이지 않거나, 한 체인의 EVM 세대가 낮아 실행되지 않거나, 노드 실행 테스트가 아닌 항목이다. ID는 기존 것을 그대로 적는다.

## StableNet 전용 (107개)

| 기존 ID | 출처 | 이유 |
| --- | --- | --- |
| RT-A-2-05b | Regression Test Case with scenario, Regression Test Case, \[WEMIX 4.0\] 테스트 시나리오 | StableNet의 최대 수수료 하한(최소 기본 수수료와 최소 팁의 합) 검사다. 다른 두 체인의 풀에는 이 검사가 없다 |
| RT-B-04 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-B-05 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-B-06 | Regression Test Case with scenario, Regression Test Case | StableNet 헤더 팁 강제 규칙을 검사한다 |
| RT-C-01 | Regression Test Case with scenario, Regression Test Case | StableNet 헤더 팁 강제 규칙을 검사한다 |
| RT-C-02 | Regression Test Case with scenario, Regression Test Case | StableNet 헤더 팁 강제 규칙을 검사한다 |
| RT-C-06 | Regression Test Case with scenario, Regression Test Case | 기본 수수료 하한(`params.MinBaseFee`)은 StableNet 에만 있는 상수다. 다른 두 체인의 노드 프로그램에는 상수도 검사 경로도 없다. 공통 목록의 CT-FEE-006 이었다 |
| RT-E-01 | Regression Test Case with scenario, Regression Test Case | StableNet 계정 차단·인증 정책을 검사한다 |
| RT-E-02 | Regression Test Case with scenario, Regression Test Case | StableNet 계정 차단·인증 정책을 검사한다 |
| RT-E-03 | Regression Test Case with scenario, Regression Test Case | StableNet 계정 차단·인증 정책을 검사한다 |
| RT-E-04 | Regression Test Case with scenario, Regression Test Case | StableNet 계정 차단·인증 정책을 검사한다 |
| RT-E-05 | Regression Test Case with scenario, Regression Test Case | StableNet 계정 차단·인증 정책을 검사한다 |
| RT-E-06 | Regression Test Case with scenario, Regression Test Case | StableNet 계정 차단·인증 정책을 검사한다 |
| RT-E-07 | Regression Test Case with scenario, Regression Test Case | StableNet 계정 차단·인증 정책을 검사한다 |
| RT-E-08 | Regression Test Case with scenario, Regression Test Case | StableNet 계정 차단·인증 정책을 검사한다 |
| RT-E-09 | Regression Test Case with scenario, Regression Test Case | StableNet 계정 차단·인증 정책을 검사한다 |
| RT-F-1-01 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-1-02 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-1-03 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-1-04 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-1-05 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-2-01 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-2-02 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-2-03 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-2-04 | Regression Test Case with scenario | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-3-01 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-3-02 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-3-03 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-3-04 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-3-05 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-3-06 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-4-01 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-4-02 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-4-03 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-4-04 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-5-01 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-5-02 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-5-03 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-5-04 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-5-05 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-5-06 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-5-07 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-5-08 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-F-5-09 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-G-1-06 | Regression Test Case with scenario, Regression Test Case, \[WEMIX 4.0\] 테스트 시나리오 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-G-2-02 | Regression Test Case with scenario, Regression Test Case | 권장 팁의 비교 대상이 StableNet 블록 헤더의 `GasTip` 이다. 같은 `istanbul_getWbftExtraInfo` API 를 가진 WEMIX4.0 도 응답에 이 필드가 없고 WEMIX3.0 은 API 자체가 없다. 공통 목록의 CT-FEE-011 이었다 |
| RT-G-2-04 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-G-5-02 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| RT-G-5-03 | Regression Test Case with scenario, Regression Test Case | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| TC-1-1-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-07 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-08 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-09 | 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-10 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-11 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-1-1-12 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-1-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-1-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-3-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-3-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-3-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-3-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-3-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-3-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-4-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-4-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-4-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-4-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-07 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-08 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-09 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-10 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-11 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-5-12 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-4-6-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 헤더 팁 강제 규칙을 검사한다 |
| TC-4-6-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 헤더 팁 강제 규칙을 검사한다 |
| TC-4-6-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 헤더 팁 강제 규칙을 검사한다 |
| TC-5-2-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-5-2-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-5-2-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-5-2-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-5-2-05 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TC-5-2-06 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-4-4 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-1-1-01 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-1-1-02 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-1-1-03 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-1-1-04 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-4-1-01 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-4-3-01 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-4-5-01 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-4-5-02 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-4-5-03 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-5-2-01 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-5-2-02 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| TS-5-2-03 | 1st Test Scenarios | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |

## WEMIX4.0 전용 (28개)

| 기존 ID | 출처 | 이유 |
| --- | --- | --- |
| GOV-001 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-002 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-003 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-004 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-005 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-006 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-007 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-008 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-009 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-010 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-011 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-012 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-013 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-014 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-015 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-016 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-017 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-018 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-019 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-020 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-021 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-022 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-023 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| GOV-024 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| NODE-006 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| NODE-007 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| RPC-009 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |
| RPC-023 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다 |

## WEMIX3.0 전용 (17개)

| 기존 ID | 출처 | 이유 |
| --- | --- | --- |
| ETCD-01 | \[WEMIX3.0\] 테스트 시나리오 | WEMIX3.0의 etcd 작업·토큰 키 복구를 검사한다 |
| ETCD-02 | \[WEMIX3.0\] 테스트 시나리오 | WEMIX3.0의 etcd 작업·토큰 키 복구를 검사한다 |
| ETCD-03 | \[WEMIX3.0\] 테스트 시나리오 | WEMIX3.0의 etcd 작업·토큰 키 복구를 검사한다 |
| ETCD-04 | \[WEMIX3.0\] 테스트 시나리오 | WEMIX3.0의 etcd 작업·토큰 키 복구를 검사한다 |
| ETCD-05 | \[WEMIX3.0\] 테스트 시나리오 | WEMIX3.0의 etcd 작업·토큰 키 복구를 검사한다 |
| GOV-01 | \[WEMIX3.0\] 테스트 시나리오 | WEMIX3.0 거버넌스 멤버 변경과 노드 연동을 검사한다 |
| MINING-02 | \[WEMIX3.0\] 테스트 시나리오 | WEMIX3.0 블록 생산 규칙(따라잡기, 중복 생산 제한)을 검사한다 |
| MINING-03 | \[WEMIX3.0\] 테스트 시나리오 | WEMIX3.0 블록 생산 규칙(따라잡기, 중복 생산 제한)을 검사한다 |
| N-001 | \[PR196\] 기존 테스트 전체 144개 우선순위 목록 | WEMIX3.0 PR196 신규 테스트 가운데 집중 목록에서 뺀 항목이다 |
| N-010 | \[PR196\] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 PR196 수정의 크기 제한값(블록 8 MiB, 메시지 10 MiB)을 검사한다 |
| N-011 | \[PR196\] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 PR196 수정의 크기 제한값(블록 8 MiB, 메시지 10 MiB)을 검사한다 |
| N-012 | \[PR196\] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 PR196 수정의 크기 제한값(블록 8 MiB, 메시지 10 MiB)을 검사한다 |
| N-013 | \[PR196\] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 PR196 수정의 크기 제한값(블록 8 MiB, 메시지 10 MiB)을 검사한다 |
| N-014 | \[PR196\] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 PR196 수정의 크기 제한값(블록 8 MiB, 메시지 10 MiB)을 검사한다 |
| N-015 | \[PR196\] 집중 테스트 28개 상세 실행 절차 | WEMIX3.0 PR196 수정의 크기 제한값(블록 8 MiB, 메시지 10 MiB)을 검사한다 |
| RPC-01 | \[WEMIX3.0\] 테스트 시나리오 | WEMIX3.0 전용 조회 API와 헤더 필드를 검사한다 |
| RPC-02 | \[WEMIX3.0\] 테스트 시나리오 | WEMIX3.0 전용 조회 API와 헤더 필드를 검사한다 |

## WEMIX3.0에서 WEMIX4.0으로 넘어가는 전환 시나리오 (2개)

| 기존 ID | 출처 | 이유 |
| --- | --- | --- |
| NODE-001 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX3.0 데이터로 WEMIX4.0을 띄우는 전환 시나리오다 |
| NODE-002 | \[WEMIX4.0\] Test, \[WEMIX 4.0\] 테스트 시나리오 | WEMIX3.0 데이터로 WEMIX4.0을 띄우는 전환 시나리오다 |

## StableNet 의 기본 수수료 규칙 때문에 뺀 것 (3개)

세 체인이 모두 기본 수수료를 쓰지만 움직이는 규칙이 다르다. StableNet 의 anzeon 은 상승 문턱과 하강 문턱 두 개를 두고, 사용률이 상승 문턱을 넘으면 올리고 하강 문턱 아래면 내린다. WEMIX3.0 과 WEMIX4.0 은 표준 EIP-1559 대로 블록 한도의 절반 하나를 목표로 삼아, 목표보다 많이 쓰면 올리고 적게 쓰면 내린다.

그래서 같은 사용률이 한쪽에서는 상승이고 다른 쪽에서는 하락이다. 세 테스트가 쓰는 25% 채우기가 StableNet 에서는 상승 문턱을 넘지만 다른 두 체인에서는 목표에 못 미친다.

2026-09-28 에 세 체인으로 돌려 확인했다. WEMIX4.0 에서 기본 수수료가 0.34 Gwei 에서 0.22 Gwei 로 내려갔고, WEMIX3.0 에서는 이미 최저값에 닿아 있어 움직이지 않았다.

셋을 함께 뺀다. 감소 테스트는 WEMIX4.0 에서 통과했지만 재려던 것과 다른 이유였다. 표준 규칙에서 25% 채우기는 부하가 아니라 목표 미달이라, 부하를 거는 동안에도 이미 내려가고 있었다. 이 하나만 남기면 아무것도 검증하지 않는 테스트가 공통 목록에 남는다.

| 기존 ID | 출처 | 이유 |
| --- | --- | --- |
| RT-C-03 | Regression Test Case with scenario, Regression Test Case | 사용률이 높을 때 기본 수수료가 오르는지 본다. 상승 문턱은 StableNet 의 anzeon 에만 있다. 공통 목록의 CT-FEE-003 이었다 |
| RT-C-04 | Regression Test Case with scenario, Regression Test Case | 사용률이 보통일 때 기본 수수료가 그대로인지 본다. 표준 EIP-1559 에는 유지 구간이 없다. 공통 목록의 CT-FEE-004 였다 |
| RT-C-05 | Regression Test Case with scenario, Regression Test Case | 사용률이 낮을 때 기본 수수료가 내리는지 본다. 하강 문턱은 StableNet 에만 있다. 공통 목록의 CT-FEE-005 였다 |

## WEMIX3.0 의 EVM 세대 때문에 뺀 것 (0개)

한때 두 건을 여기에 두었다가 2026-09-28 에 둘 다 공통으로 되돌렸다. RT-A-3-02(CT-CONTRACT-002)와 RT-A-3-05(CT-CONTRACT-005)다.

뺀 이유는 배포하던 컨트랙트가 PUSH0 옵코드를 담아 WEMIX3.0 에서 배포되지 않는다는 것이었다. WEMIX3.0 의 EVM 이 London 세대에서 멈춰 있어 Shanghai 가 들여온 PUSH0(`0x5f`)를 모르는 것은 사실이고 바뀌지 않는다. 다만 그것은 체인이 그 테스트를 못 한다는 뜻이 아니라, 우리가 고른 바이트코드를 못 읽는다는 뜻이었다. 두 테스트가 재려던 것은 "저장값을 바꾸고 다시 읽는다" 와 "되돌리는 함수를 조회 호출하면 에러가 온다" 이고, 둘 다 세 체인이 하는 동작이다.

그래서 컨트랙트를 PUSH0 없이 손으로 써서 되돌렸다. 공통에 이미 있던 CT-CONTRACT-004 와 CT-CONTRACT-006 이 같은 방식의 조각을 쓰고 세 체인에서 돌고 있었다.

앞으로 공통 테스트에 컨트랙트를 쓸 때는 solc 가 기본으로 내는 바이트코드를 그대로 싣지 않는다. 지정 없이 컴파일하면 PUSH0 가 섞여 WEMIX3.0 에서만 배포가 실패한다.

## 체인마다 두 번째 빌드가 있어야 성립하는 것 (1개)

2026-09-29 에 뺐다. 케이스는 통과하는데 재려던 것을 확인하지 못했다.

CT-NODE-012 는 노드 프로그램을 바꿔 끼워도 이미 처리한 트랜잭션이 그대로 조회되는지 본다. 특히 보낸 이는 영수증에 적힌 값이 아니라 서명에서 되찾는 값이라, 새 프로그램이 같은 값을 내놓는지가 곧 서명 호환성이다. 그러려면 **바꿔 낄 두 번째 빌드**가 있어야 하고, 그 빌드는 체인마다 따로다. 세 체인이면 여섯 개를 만들어 관리해야 한다.

지금은 StableNet 만 1.0.1 과 1.1.0 두 개를 갖고 있다. WEMIX3.0 과 WEMIX4.0 에서는 같은 바이너리를 바꿔 끼우게 되어 `web3_clientVersion` 이 교체 전후로 같다. 교체가 일어났는지 자체를 확인할 수 없으니, 뒤따르는 비교가 무엇을 증명하는지도 말할 수 없다.

재료가 없는 것이지 테스트가 틀린 것이 아니다. 두 체인의 업그레이드 빌드를 어느 커밋으로 만들지 정해지면 다시 공통으로 올릴 수 있다.

| 기존 ID | 출처 | 이유 |
| --- | --- | --- |
| TC-3-1-04 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 교체할 두 번째 빌드가 체인마다 필요하다. 세 체인이면 여섯 개다. 공통 목록의 CT-NODE-012 였고, `go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json` 으로 돌아갔다 |

## 경계를 체인마다 다르게 정해 뺀 부분 (1건, 부분)

2026-09-29 에 뺐다. 기존 ID 전체가 아니라 CT-FEE-002(최소 가스비 경계값)의 한 부분이다. CT-FEE-002 는 "미만은 거부, 초과는 성공" 으로 목록에 남는다.

뺀 것은 "최소 가스비와 같은 값을 보내면 받아들여진다" 다. RPC 로 보낸 트랜잭션에 대해 세 체인은 하한을 다르게 정하고 다른 시점에 검사한다. StableNet 은 최대 수수료가 고정 상수(최소 기본 수수료)보다 낮으면 풀에서 거부한다. WEMIX3.0 은 최대 수수료가 "다음 블록 기본 수수료 + 거버넌스 팁" 보다 낮거나 팁이 거버넌스 팁보다 낮으면 거부한다. WEMIX4.0 은 RPC 로 받은 트랜잭션을 제출 때 가격으로 거부하지 않고, 최대 수수료가 포함 블록의 기본 수수료보다 낮으면 블록에 넣지 않는다. 포함 블록의 기본 수수료는 블록마다 움직이므로 "같은 값" 을 결정적으로 만들 수도 없다.

| 기존 ID | 출처 | 이유 |
| --- | --- | --- |
| TC-1-3-01~06, TS-1-3-01·02 (같음 부분) | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 최소 가스비와 같은 값의 결과가 체인마다 다르다. 미만과 초과 부분은 공통 목록의 CT-FEE-002 로 남았다 |

## 노드를 띄우는 테스트가 아님(단위 테스트·빌드 확인) (11개)

| 기존 ID | 출처 | 이유 |
| --- | --- | --- |
| TC-3-1-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 노드를 띄우지 않는 성능 측정이다 |
| TC-3-1-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 노드를 띄우지 않는 성능 측정이다 |
| TC-3-1-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 노드를 띄우지 않는 성능 측정이다 |
| TC-5-1-01 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 빌드와 실행 파일 확인이다 |
| TC-5-1-02 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 빌드와 실행 파일 확인이다 |
| TC-5-1-03 | 1st Test Scenarios, 1st Test Cases (v1.0.0+) | 빌드와 실행 파일 확인이다 |
| TS-2-1 | 1st Test Scenarios | 노드를 띄우지 않는 단위 테스트다 |
| TS-2-2 | 1st Test Scenarios | 노드를 띄우지 않는 단위 테스트다 |
| TS-2-3 | 1st Test Scenarios | 노드를 띄우지 않는 단위 테스트다 |
| TS-2-4 | 1st Test Scenarios | 노드를 띄우지 않는 단위 테스트다 |
| TS-5-1-01 | 1st Test Scenarios | 빌드와 실행 파일 확인이다 |

## 제외한 자동 테스트

| 실행 ID | 파일 | 구분 | 이유 |
| --- | --- | --- | --- |
| stablenet-delayed-fork | go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| govminter-v2-code | go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| burn-cancel-refundable | go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| burn-reject-refundable | go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| burn-expire-refundable | go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| burn-execute-no-refundable | go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| claim-burn-refund-succeeds | go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| claim-zero-refund-reverts | go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| claim-burn-refund-double-reverts | go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| prealloc-preserved-across-boho | go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| boho-chain-config-active | go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| anzeon-active-before-boho | go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| upgrade-registry-order | go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| v1-params-init-storage | go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| burn-refund-events | go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| effective-gas-price-authorized-bp-en | go-stablenet/post-v1.0.0-change/effective-gas-price/01-effective-gas-price-authorized-bp-en.json | StableNet 전용 | StableNet 헤더 팁 강제 규칙을 검사한다 |
| effective-gas-price-regular | go-stablenet/post-v1.0.0-change/effective-gas-price/02-effective-gas-price-regular.json | StableNet 전용 | StableNet 헤더 팁 강제 규칙을 검사한다 |
| auth-tx-event-last-bp-en | go-stablenet/post-v1.0.0-change/effective-gas-price/03-auth-tx-event-last-bp-en.json | StableNet 전용 | StableNet 헤더 팁 강제 규칙을 검사한다 |
| authorized-extra-bit-synced | go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| blacklisted-extra-bit-synced | go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| stablenet-account-extra | go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| extra-union-merge | go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| dual-status-extra | go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| extra-balance-preserved | go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| invalid-extra-reject | go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| extra-state-across-delayed-boho | go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| unsupported-system-contract-version | go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| authorized-accounts-no-space | go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| authorized-accounts-space | go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| authorized-accounts-trim | go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| authorized-accounts-empty-item | go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| authorized-accounts-single | go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| authorized-accounts-empty | go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | StableNet 전용 | StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다 |
| regular-account-gastip-forced | go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | StableNet 전용 | StableNet 헤더 팁 강제 규칙을 검사한다 |
| authorized-account-gastip-free | go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | StableNet 전용 | StableNet 헤더 팁 강제 규칙을 검사한다 |
| system-contracts-deployed | go-stablenet/regression/api/06-system-contracts-deployed.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| estimate-gas-token-transfer | go-stablenet/regression/api/10-estimate-gas-token-transfer.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| token-total-supply-readable | go-stablenet/regression/api/22-token-total-supply-readable.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| token-approve-sets-allowance | go-stablenet/regression/api/23-token-approve-sets-allowance.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| sender-blacklisted-rejected | go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | StableNet 전용 | StableNet 계정 차단·인증 정책을 검사한다 |
| recipient-blacklisted-rejected | go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | StableNet 전용 | StableNet 계정 차단·인증 정책을 검사한다 |
| feepayer-blacklisted-rejected | go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | StableNet 전용 | StableNet 계정 차단·인증 정책을 검사한다 |
| address-unblacklisted-event | go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | StableNet 전용 | StableNet 계정 차단·인증 정책을 검사한다 |
| zero-address-transfer-rejected | go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | StableNet 전용 | StableNet 계정 차단·인증 정책을 검사한다 |
| precompile-transfer-rejected | go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | StableNet 전용 | StableNet 계정 차단·인증 정책을 검사한다 |
| account-blacklist-readable | go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | StableNet 전용 | StableNet 계정 차단·인증 정책을 검사한다 |
| account-authorization-readable | go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | StableNet 전용 | StableNet 계정 차단·인증 정책을 검사한다 |
| authorized-tx-executed-event | go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | StableNet 전용 | StableNet 계정 차단·인증 정책을 검사한다 |
| native-coin-adapter-code | go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| token-transfer-emits-event | go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| token-balance-readable | go-stablenet/regression/system-contracts/02-token-balance-readable.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| token-transfer-from-moves-balance | go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| mint-transfer-event | go-stablenet/regression/system-contracts/04-mint-transfer-event.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| burn-transfer-event | go-stablenet/regression/system-contracts/05-burn-transfer-event.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| mint-proposal-executes | go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| burn-proposal-executes | go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| quorum-deficient-stays-voting | go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| validator-metadata-readable | go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| gastip-governance-updates-header | go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| proposal-expiry-transitions | go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| configure-minter-proposal-executes | go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| remove-minter-executes | go-stablenet/regression/system-contracts/13-remove-minter-executes.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| masterminter-member-add-remove | go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| non-member-configure-minter-rejected | go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| blacklist-proposal-executes | go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| authorize-proposal-executes | go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| unauthorize-proposal-executes | go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| direct-blacklist-call-rejected | go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| authorized-account-added-event | go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| token-metadata | go-stablenet/regression/system-contracts/25-token-metadata.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| minter-status-readable | go-stablenet/regression/system-contracts/28-minter-status-readable.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| validator-add-member-executes | go-stablenet/regression/wbft/04-validator-add-member-executes.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| validator-add-member-epoch-activates | go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| validator-remove-member-executes | go-stablenet/regression/wbft/05-validator-remove-member-executes.json | StableNet 전용 | StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다 |
| stablenet-gastip-field | go-stablenet/regression/wbft/14-stablenet-gastip-field.json | StableNet 전용 | StableNet 헤더 팁 강제 규칙을 검사한다 |
| anzeon-basefee-increase | go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | StableNet 전용 | anzeon 의 상승 문턱을 검사한다. 다른 두 체인은 표준 EIP-1559 라 같은 사용률에서 반대로 내려간다 |
| anzeon-basefee-stable | go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | StableNet 전용 | anzeon 의 유지 구간을 검사한다. 표준 EIP-1559 에는 유지 구간이 없다 |
| anzeon-basefee-decrease | go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | StableNet 전용 | anzeon 의 하강 문턱을 검사한다. WEMIX4.0 에서 통과하지만 재려던 것과 다른 이유다 |
| register-contract | go-stablenet/vocabulary/03-register-contract.json | WEMIX3.0 의 EVM 세대 | 배포하는 NodeRegistry 바이트코드가 PUSH0 를 45번 쓴다. WEMIX3.0 에서 배포가 가스를 전부 쓰고 실패한다 |
| feecap-above-min-accepted | go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | StableNet 전용 | 경계값을 StableNet 헤더 팁으로 구한다. 세 체인 공통의 '초과' 는 공통 케이스 dynamic-feecap-above-min-accepted 가 본다(CT-FEE-002) |
| feecap-exact-min-accepted | go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | StableNet 전용 | '최소 가스비와 같으면 받아들여진다' 를 본다. 경계가 체인마다 달라 공통이 아니다(아래 '경계를 체인마다 다르게 정해 뺀 부분') |
| basefee-minimum | go-stablenet/regression/anzeon/06-basefee-minimum.json | StableNet 전용 | StableNet 의 최소 기본 수수료(MinBaseFee)를 본다. 공통 목록의 CT-FEE-006 이었다 |
| gas-price-equals-basefee-plus-tip | go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | StableNet 전용 | 팁을 StableNet 헤더 팁으로 읽는다. 세 체인 공통은 gas-price-positive 가 본다(CT-FEE-010) |
| max-priority-fee-equals-gastip | go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | StableNet 전용 | 권장 팁이 StableNet 헤더 팁과 같은지 본다. 공통 목록의 CT-FEE-011 이었다 |
| legacy-transfer | go-stablenet/regression/ethereum/08-legacy-transfer.json | StableNet 전용 | 가스 가격을 StableNet 헤더 팁으로 구한다. 세 체인 공통은 legacy-value-transfer 가 본다(CT-TX-002) |
| dynamic-fee-tx | go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | StableNet 전용 | 팁을 StableNet 헤더 팁으로 읽는다. 세 체인 공통은 dynamic-fee-transfer 가 본다(CT-TX-003) |
| gaslimit-exceeded-rejected | go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | StableNet 전용 | 수수료를 StableNet 헤더 팁으로 구한다. 세 체인 공통은 gas-limit-exceeds-block-rejected 가 본다(CT-TX-016) |
| signature-compat-across-swap | go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | 두 번째 빌드 필요 | 교체할 두 번째 빌드가 StableNet 에만 있다(위 '체인마다 두 번째 빌드가 있어야 성립하는 것') |
| stablenet-derived-vocabulary | go-stablenet/vocabulary/01-derived-address-and-checksum.json | 실행 도구 자체 검사(체인 무관) | 실행 도구 자체의 계산 기능 검사다 |
| wemix-wbft-handoff | go-wemix/handoff/01-wemix-wbft-handoff.json | WEMIX3.0에서 WEMIX4.0으로 넘어가는 전환 시나리오 | WEMIX3.0 데이터로 WEMIX4.0을 띄우는 전환 시나리오다 |
