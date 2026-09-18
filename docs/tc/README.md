# 테스트 케이스 문서 (Confluence Chainbench 폴더 사본)

Confluence `platfomDev` 스페이스의 [Chainbench 폴더](https://wemade.atlassian.net/wiki/spaces/platfomDev/folder/2914091137) 아래 페이지 44개를 2026-09-18에 전부 가져와 옮긴 것이다. 페이지 계층을 그대로 폴더로 만들었고, 각 파일 머리에 원본 페이지 링크, 페이지 ID, 버전, 최종 수정일을 적었다. 본문은 Confluence가 내준 markdown 변환 결과를 손대지 않고 옮겼다. 원본 안의 링크는 Confluence 페이지를 그대로 가리킨다.

## 옮기면서 확인한 것

- 페이지 44개 전부를 파일 44개로 옮겼다. 본문이 비어 있거나 하위 페이지 목록 매크로만 있는 페이지 6개도 파일을 만들고 그 사실을 적었다.
- markdown 변환에서 빠진 매크로는 목차(TOC) 매크로뿐이다. HTML 원문을 대조해 본문 손실이 없음을 확인했다.
- 첨부 파일이 붙은 페이지는 3개다. 첨부는 해당 폴더의 `attachments/`에 넣었다.
- 첨부 `testscript.zip`(1st Test Script 페이지) 안의 `common.sh`, `gov_burn_proposal.sh`에 검증자 개인 키가 그대로 적혀 있었다. 저장소에는 `<REDACTED_PRIVATE_KEY>`로 가려서 넣었다. 그 밖의 내용은 원본 그대로다.
- 테스트 환경 표와 `node_ctrl.sh`에 폐쇄망 서버 IP(172.21.132.x)와 부트노드 enode가 원본 그대로 들어 있다.

## 파일 목록

| 파일 | Confluence 페이지 | 페이지 ID | 비고 |
| --- | --- | --- | --- |
| [wemix4.0/README.md](wemix4.0/README.md) | [WEMIX4.0] Test | 2636711400 | 87개 TC 요약표 |
| [wemix4.0/01-test-scenarios.md](wemix4.0/01-test-scenarios.md) | [WEMIX 4.0] 테스트 시나리오 | 2639659344 | 87개 TC 상세 시나리오 |
| [wemix4.0/02-test-result.md](wemix4.0/02-test-result.md) | [WEMIX 4.0] 테스트 결과 | 2693038270 | 환경·빌드 정보만 있음 |
| [wemix4.0/03-commit-change-log.md](wemix4.0/03-commit-change-log.md) | [WEMIX 4.0] Commit Change Log | 2878996534 | |
| [wemix4.0/04-2nd-change-test-cases.md](wemix4.0/04-2nd-change-test-cases.md) | [WEMIX 4.0] 2nd Change Test Cases | 2879193122 | |
| [stablenet/README.md](stablenet/README.md) | [StableNet] Test | 2599125014 | 하위 페이지 매크로만 |
| [stablenet/regression/README.md](stablenet/regression/README.md) | Regression Test | 2599026705 | 하위 페이지 매크로만 |
| [stablenet/regression/01-regression-test-case.md](stablenet/regression/01-regression-test-case.md) | Regression Test Case | 2599813145 | 104개 TC 요약표 |
| [stablenet/regression/02-regression-test-case-with-scenario.md](stablenet/regression/02-regression-test-case-with-scenario.md) | Regression Test Case with scenario | 2599256076 | 시나리오 상세 |
| [stablenet/regression/03-regression-test-result.md](stablenet/regression/03-regression-test-result.md) | Regression Test Result | 2634843002 | 환경·빌드 정보만 있음 |
| [stablenet/post-v1.0.0-change/README.md](stablenet/post-v1.0.0-change/README.md) | 테스트(v1.0.0이후 변경 사항) | 2614001728 | 본문 없음 |
| [stablenet/post-v1.0.0-change/01-1st-test-cases.md](stablenet/post-v1.0.0-change/01-1st-test-cases.md) | 1st Test Cases (v1.0.0+) | 2614231162 | 70개 TC |
| [stablenet/post-v1.0.0-change/02-1st-test-scenarios.md](stablenet/post-v1.0.0-change/02-1st-test-scenarios.md) | 1st Test Scenarios | 2599289086 | 첨부: genesis-standard.json, genesis-expiry60.json |
| [stablenet/post-v1.0.0-change/03-1st-test-result/README.md](stablenet/post-v1.0.0-change/03-1st-test-result/README.md) | 1st Test Result | 2611249353 | 본문 없음 |
| [stablenet/post-v1.0.0-change/03-1st-test-result/2026-04-13-test.md](stablenet/post-v1.0.0-change/03-1st-test-result/2026-04-13-test.md) | 2026-04-13-테스트 | 2611052714 | 실행 결과(Pass 기록), 제네시스·alloc 덤프 |
| [stablenet/post-v1.0.0-change/04-1st-test-script.md](stablenet/post-v1.0.0-change/04-1st-test-script.md) | 1st Test Script | 2610399285 | 첨부: testscript.zip(스크립트 43개, 개인 키 가림) |
| [stablenet/post-v1.0.0-change/05-commit-changelog.md](stablenet/post-v1.0.0-change/05-commit-changelog.md) | Commit ChangeLog | 2875326583 | |
| [stablenet/post-v1.0.0-change/06-2nd-change-test-cases.md](stablenet/post-v1.0.0-change/06-2nd-change-test-cases.md) | 2nd Change Test Cases | 2874802346 | |
| [stablenet/node-script.md](stablenet/node-script.md) | node script | 2611871801 | 첨부: node_ctrl.sh |
| [wemix3.0/README.md](wemix3.0/README.md) | [WEMIX3.0] Test | 2918023219 | 본문 없음 |
| [wemix3.0/01-test-items.md](wemix3.0/01-test-items.md) | [WEMIX3.0] 테스트 항목 | 2918154299 | |
| [wemix3.0/02-test-scenarios.md](wemix3.0/02-test-scenarios.md) | [WEMIX3.0] 테스트 시나리오 | 2918252564 | ETCD/MINING/BRIOCHE/GOV/RPC 14개 |
| [wemix3.0/hotfix-pr196/README.md](wemix3.0/hotfix-pr196/README.md) | [WEMIX3.0] Hotfix(PR-196) 테스트 항목 | 2978513035 | |
| [wemix3.0/hotfix-pr196/01-focused-tests-summary.md](wemix3.0/hotfix-pr196/01-focused-tests-summary.md) | [PR196] 집중 테스트 28개 요약 | 2977857698 | |
| [wemix3.0/hotfix-pr196/02-focused-tests-detailed-procedure.md](wemix3.0/hotfix-pr196/02-focused-tests-detailed-procedure.md) | [PR196] 집중 테스트 28개 상세 실행 절차 | 2979070094 | |
| [wemix3.0/hotfix-pr196/02a-test-glossary.md](wemix3.0/hotfix-pr196/02a-test-glossary.md) | [PR196] 테스트 용어집 | 2978185459 | 상세 실행 절차의 하위 페이지 |
| [wemix3.0/hotfix-pr196/03-all-144-tests-priority-list.md](wemix3.0/hotfix-pr196/03-all-144-tests-priority-list.md) | [PR196] 기존 테스트 전체 144개 우선순위 목록 | 2978054308 | |
| [wemix3.0/hotfix-pr196/04-regression-scope-and-result.md](wemix3.0/hotfix-pr196/04-regression-scope-and-result.md) | [PR196] 회귀 테스트 수행 범위 및 결과 기록 | 2977988792 | |
| [common/README.md](common/README.md) | [Common] Test | 2987917348 | |
| [common/01-common-test-list.md](common/01-common-test-list.md) | 공통 테스트 목록 | 2988965889 | CT- 76개 |
| [common/02-mainnet-dependencies.md](common/02-mainnet-dependencies.md) | 메인넷별 의존 요소 | 2988376101 | |
| [common/03-separate-implementation-items.md](common/03-separate-implementation-items.md) | 별도 구현이 필요한 항목 | 2987884735 | |
| [common/04-separation-change-scope.md](common/04-separation-change-scope.md) | 공통 테스트 분리 변경 범위 | 2987720853 | |
| [common/05-regression-execution-bundles.md](common/05-regression-execution-bundles.md) | 회귀 실행 묶음 | 2986934428 | |
| [common/06-detailed-procedures/README.md](common/06-detailed-procedures/README.md) | 상세 실행 절차 | 2987196682 | |
| [common/06-detailed-procedures/01-node.md](common/06-detailed-procedures/01-node.md) | 상세 실행 절차 NODE (노드·동기화·네트워크) | 2988539943 | |
| [common/06-detailed-procedures/02-tx.md](common/06-detailed-procedures/02-tx.md) | 상세 실행 절차 TX (트랜잭션 전송·거부) | 2987720880 | |
| [common/06-detailed-procedures/03-fee.md](common/06-detailed-procedures/03-fee.md) | 상세 실행 절차 FEE (수수료·가스 정책) | 2987819122 | |
| [common/06-detailed-procedures/04-contract.md](common/06-detailed-procedures/04-contract.md) | 상세 실행 절차 CONTRACT (컨트랙트 실행) | 2987786466 | |
| [common/06-detailed-procedures/05-rpc.md](common/06-detailed-procedures/05-rpc.md) | 상세 실행 절차 RPC (조회·구독 API) | 2988638234 | |
| [common/06-detailed-procedures/06-fault.md](common/06-detailed-procedures/06-fault.md) | 상세 실행 절차 FAULT (장애·복구) | 2987524207 | |
| [common/07-glossary.md](common/07-glossary.md) | 용어집 | 2986868908 | |
| [common/08-appendix-a-two-chain-tests.md](common/08-appendix-a-two-chain-tests.md) | 부록 A. 두 체인에서만 가능한 테스트 | 2986901809 | |
| [common/09-appendix-b-excluded-tests.md](common/09-appendix-b-excluded-tests.md) | 부록 B. 공통에서 제외한 테스트와 이유 | 2987524189 | |

## 첨부 파일

| 경로 | 출처 페이지 | 비고 |
| --- | --- | --- |
| `stablenet/post-v1.0.0-change/attachments/genesis-standard.json` | 1st Test Scenarios | |
| `stablenet/post-v1.0.0-change/attachments/genesis-expiry60.json` | 1st Test Scenarios | standard와 `govMinter.params.expiry`(604800 → 60)만 다름 |
| `stablenet/post-v1.0.0-change/attachments/testscript/` | 1st Test Script | `testscript.zip`을 푼 것. 같은 내용의 `testscript.txt`도 페이지에 붙어 있음. 개인 키 4곳 가림 |
| `stablenet/attachments/node_ctrl.sh` | node script | 페이지 본문의 스크립트와 같음 |
