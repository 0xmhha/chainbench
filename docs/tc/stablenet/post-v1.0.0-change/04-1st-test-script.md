# 1st Test Script

> 출처: Confluence [1st Test Script](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2610399285) (페이지 ID 2610399285, 버전 5, 최종 수정 2026-08-05)  
> 상위 페이지: 테스트(v1.0.0이후 변경 사항)  
> 가져온 날짜: 2026-09-18

이 페이지는 본문 글 없이 첨부 파일 하나만 붙어 있다. 첨부는 두 개가 등록돼 있는데(`testscript.zip`, `testscript.txt`, 각 69,268바이트) 두 파일의 내용은 바이트 단위로 같다. `testscript.txt`는 확장자만 다른 같은 zip 파일이다.

압축을 풀면 `hardfork/` 폴더 하나에 셸 스크립트 43개가 들어 있다. 이 저장소에는 [attachments/testscript/](attachments/testscript/) 아래에 풀어서 넣었다(macOS 메타데이터 `__MACOSX/`는 제외).

**주의: 개인 키를 가렸다.** `common.sh`의 `VAL1_KEY`, `VAL2_KEY`, `VAL3_KEY`와 `gov_burn_proposal.sh`의 `PRIVATE_KEY`에 검증자 개인 키가 그대로 적혀 있었다. 저장소에는 그 값을 `<REDACTED_PRIVATE_KEY>`로 바꿔 넣었다. 원본 값은 Confluence 첨부에서 받아야 한다. 그 밖의 내용은 원본 그대로다.

## 첨부 파일 목록 (testscript.zip 내부)

| 파일 | 크기(바이트) | 용도(파일 이름 기준) |
| --- | --- | --- |
| hardfork/common.sh | 26,911 | 공통 함수·환경 변수(노드 주소, 계정, 키) |
| hardfork/run-all.sh | 7,234 | 전체 실행 |
| hardfork/node_ctrl.sh | 14,873 | 원격 노드 제어 (node script 페이지의 첨부와는 다른 버전) |
| hardfork/balance_query.sh | 17,487 | 잔액 조회 |
| hardfork/gov_query.sh | 13,576 | 거버넌스 조회 |
| hardfork/gov_burn_proposal.sh | 8,005 | 소각 제안 |
| hardfork/tc-1-1-01.sh | 15,243 | TC-1-1-01 |
| hardfork/tc-1-1-02.sh | 16,996 | TC-1-1-02 |
| hardfork/tc-1-1-03.sh | 16,542 | TC-1-1-03 |
| hardfork/tc-1-1-04.sh | 13,203 | TC-1-1-04 |
| hardfork/tc-1-1-05.sh | 2,176 | TC-1-1-05 |
| hardfork/tc-1-1-06.sh | 1,925 | TC-1-1-06 |
| hardfork/tc-1-1-07.sh | 1,330 | TC-1-1-07 |
| hardfork/tc-1-1-08.sh | 1,845 | TC-1-1-08 |
| hardfork/tc-1-1-09.sh | 2,089 | TC-1-1-09 |
| hardfork/tc-1-1-10.sh | 2,303 | TC-1-1-10 |
| hardfork/tc-1-1-11.sh | 2,411 | TC-1-1-11 |
| hardfork/tc-1-1-12.sh | 1,765 | TC-1-1-12 |
| hardfork/tc-1-2-01.sh | 1,091 | TC-1-2-01 |
| hardfork/tc-1-2-02.sh | 1,239 | TC-1-2-02 |
| hardfork/tc-1-2-03.sh | 743 | TC-1-2-03 |
| hardfork/tc-1-2-04.sh | 666 | TC-1-2-04 |
| hardfork/tc-1-2-05.sh | 976 | TC-1-2-05 |
| hardfork/tc-1-2-06.sh | 1,002 | TC-1-2-06 |
| hardfork/tc-1-3-gas.sh | 4,924 | TC-1-3 (최소 가스비) |
| hardfork/tc-section2-vuln.sh | 2,857 | 2절 취약점 패치 단위 테스트 |
| hardfork/tc-3-1-04.sh | 1,714 | TC-3-1-04 |
| hardfork/tc-3-1-bench.sh | 3,429 | TC-3-1 벤치마크 |
| hardfork/tc-4-1-01.sh | 1,647 | TC-4-1-01 |
| hardfork/tc-4-1-02.sh | 1,620 | TC-4-1-02 |
| hardfork/tc-4-1-03.sh | 1,839 | TC-4-1-03 |
| hardfork/tc-4-2-gas-estimate.sh | 3,646 | TC-4-2 (가스 추정) |
| hardfork/tc-4-3-string.sh | 3,178 | TC-4-3 (설정 문자열) |
| hardfork/tc-4-4-01.sh | 2,038 | TC-4-4-01 |
| hardfork/tc-4-4-02.sh | 1,294 | TC-4-4-02 |
| hardfork/tc-4-5-01.sh | 878 | TC-4-5-01 |
| hardfork/tc-4-5-09.sh | 1,143 | TC-4-5-09 |
| hardfork/tc-4-6-01.sh | 1,501 | TC-4-6-01 |
| hardfork/tc-5-1-build.sh | 2,110 | TC-5-1 (빌드) |
| hardfork/tc-5-2-06.sh | 1,105 | TC-5-2-06 |
| hardfork/tc-5-3-01.sh | 2,580 | TC-5-3-01 |
