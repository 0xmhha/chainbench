# 분석 중 확인된 작업 파일 변경

세 클라이언트의 보존 파일과 tests/tc·원본 문서는 분석 기준을 유지했다. 검증 시점에 아래 chainbench 파일 7개는 수집한 보존본과 달랐다. 다른 작업의 변경을 되돌리거나 이번 분석본에 섞지 않았다. 이 보고서는 source-manifest의 수집 시점까지를 기준으로 하며 구현 전 아래 실행 경로를 다시 대조한다.

검증 시각: 2026-09-09T06:01:32.676522+00:00

| 변경 파일 | 다시 확인할 범위 |
|---|---|
| cmd/chainbench/resourcecmd/resource_test.go | 테스트 fixture와 resource 동작 |
| internal/chainsetup/new.go | compose/chainsetup의 설정 전달·노드 구성·실행 단계 |
| internal/chainsetup/steps_compose.go | compose/chainsetup의 설정 전달·노드 구성·실행 단계 |
| internal/chainsetup/verbs.go | compose/chainsetup의 설정 전달·노드 구성·실행 단계 |
| internal/chainsetup/verbs_up.go | compose/chainsetup의 설정 전달·노드 구성·실행 단계 |
| internal/chainsetup/workspace.go | compose/chainsetup의 설정 전달·노드 구성·실행 단계 |
| internal/testengine/compose.go | compose/chainsetup의 설정 전달·노드 구성·실행 단계 |

최신 working tree의 전체 재분석이 필요할 때는 REPLAY.md 절차로 새 산출물 디렉터리를 만든다. 원본 263개 전수 목록과 세 클라이언트 빌드 그래프의 파일 해시는 검증 시점에도 일치했다.
