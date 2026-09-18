# 레거시 셸 스위트 → DSL 포팅 감사

`~/Work/github/packages/chainbench/tests` 의 셸 테스트가 이 프로젝트의 DSL·Go
테스트로 빠짐없이, 그리고 시나리오가 바뀌지 않은 채 옮겨졌는지 확인한 기록이다.

## 방법

양쪽을 AST 로 파싱해 그래프로 만든 뒤 테스트 단위로 대조했다.

| 대상 | 파서 | 산출물 |
|---|---|---|
| 셸 테스트 460개 | `tree-sitter-bash` | 노드 943 · 간선 3,866 |
| Go 소스 607개 | `tree-sitter-go` | 노드 1,413 · 간선 3,052 (DSL 포함) |
| DSL 스펙·케이스 188개 | JSON 트리 (선언형이라 그 자체가 AST) | 위에 합침 |

대조 기준은 이름이 아니라 동작이다. 각 테스트가 호출하는 RPC 메서드, 단언 대상,
검증 명제를 뽑아 맞췄다.

## 문서

| 파일 | 내용 |
|---|---|
| `01-graph-legacy-tests.md` | 레거시 셸 테스트 그래프 — 묶음 구조, 역할별 개수, RPC·단언 사용량 |
| `02-graph-chainbench.md` | 이 프로젝트 그래프 — DSL 묶음, verb 어휘, Go 패키지·테스트 계층 |
| `03-port-audit.md` | 테스트 단위 대응표와 판정, 누락·부분 포팅 목록 |

## 요약

논리 테스트 254개 중 wemix4 84개는 `wemix4-port-tracker.md` 가 대응표를 갖고 있고,
그 표의 대상이 실물로 존재하는지 확인했다. 나머지 170개의 판정은 다음과 같다.

| 판정 | 개수 |
|---|---:|
| 동등 | 124 |
| 통합 (여러 개가 하나로, 검증 항목은 유지) | 7 |
| 부분 (검증 범위가 줄어듦) | 21 |
| 누락 | 18 → **0** (전부 스펙으로 추가) |

누락 18건은 모두 stablenet 계열이었고, 그중 15건이 `post-v1.0.0-change` 였다.
이후 **18건 전부를 DSL 스펙으로 작성해 `tests/tc/` 의 해당 자리에 넣었다.**
그 과정에서 다섯 가지 DSL 문법을 새로 만들었다 — 인증 튜플 서명(`signAuthorization`),
노드 로그 읽기(`readNodeLog`), 기동 실패 기대(`swapNode ... expect:"fail"`),
`blockAdvance` 의 반대인 `blockStalled`, 그리고 한 노드에만 다른 genesis 를 주는
`swapNode ... genesisOverlay`.

부분 포팅 21건 중에서도 검증 범위가 눈에 띄게 좁던 넷을 채웠다 —
effectivegasprice 3건의 노드 간 값 일치와 wbft add-validator 의 에폭 경계다.

추가한 스펙의 목록과 원본 대비 달라진 점은 `../../../tests/tc/README.md` 4.1 절에 있다.
자세한 감사 내용은 `03-port-audit.md` 3.2 절에 있다.
