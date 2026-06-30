# P2-1b — json_helpers unification — Plan

> 짝 spec: `2026-06-30-p2-1b-json-helpers-unification.md`. 전략: test-first, python 단일 + atomic.
> 검증 완료: jq vs python 실사용 byte-identical(단 bool-false read = jq 잠복버그, 미발현). 커밋: English, no co-author, no emoji.

## Task 0 — spec + plan 커밋
- 브랜치 `refactor/p2-1b-json-helpers`.
- commit: `docs: add P2-1b spec + plan for json_helpers unification`

## Task 1 — 계약 테스트 (추출 전, 현재 jq 백엔드에서 green)
- `tests/unit/tests/json-helpers-contract.sh`: §5 케이스 전체. assert_eq 기반.
- bool-false read 는 python 정상값(`false`)을 기대 → **현재 jq 백엔드에선 실패할 것**이므로, 이 한 케이스만 통합 후 green 되도록 설계(주석으로 "jq 잠복버그, 통합이 수정" 명시). 나머지는 현재도 green.
  - 구현: 테스트를 두 번 실행 안 함. 대신 bool-false 케이스는 `_CB_JSON_BACKEND=python3` 강제로 호출해 기대값 잠금(현재 코드에서도 python 경로는 정상). → 통합 후엔 강제 불필요하지만 테스트는 그대로 green.
- 현재 코드에서 전부 green 확인.
- commit: `test(json): contract tests locking json_helpers behavior`

## Task 2 — 단일 python 백엔드 추출
- `scripts/json_backend.py`: subcommand dispatch(read/read-stdin/array-len/write/merge/get-result/has-error). 각 본문은 기존 json_helpers python 경로 로직 그대로. **write/merge 는 atomic**: 같은 디렉토리에 `mktemp` → `json.dump` → `os.replace(tmp, file)`. (실패 시 tmp 정리 + exit 1.)
- `lib/json_helpers.sh`: 백엔드 감지 + case 분기 + jq 경로 + `_cb_dot_to_jq` + `_cb_auto_type_jq` 제거. 7함수 → thin wrapper. `_CB_JSON_SCRIPTS_DIR` 해석(BASH_SOURCE → ../scripts). 헤더 주석 갱신.
- **검증**: json-helpers-contract + pids 관련 유닛(`grep -rl cb_json tests/`) + 전체 bash regression(가능 범위) green. py_compile clean.
- commit: `refactor(json): single python backend via scripts/json_backend.py (atomic writes)`

## Task 3 — 문서 + 버전
- REFACTORING_PLAN: P2-1b ✅(부수: jq false-read 잠복버그 수정), CC-B1 완료, json_helpers 524→축소.
- REMAINING_WORK: P2-1 전체(a+b) 완료.
- 버전 0.10.1 → 0.10.2 (behavior-preserving refactor + 잠복버그 수정).
- commit: `docs+chore(p2-1b): refactoring-plan + remaining-work + version 0.10.2`

## 완료 기준
- [ ] json_helpers.sh 에서 jq 경로/백엔드 감지/dot_to_jq/auto_type 제거, 단일 python
- [ ] write/merge atomic(tmp+os.replace)
- [ ] 계약 테스트 + pids/rpc/remote 유닛 green, 실사용 byte-identical
- [ ] bool-false read 가 이제 `false` 반환(잠복버그 수정) — 테스트로 잠금
- [ ] 전 레이어 green: Go · vitest · bash
