# P2-1b — json_helpers.sh backend unification

> 작성일: 2026-06-30
> 상태: SPEC (검토 대기)
> 선행: P2-1a (profile.sh python 추출), `docs/REFACTORING_PLAN.md` §2.3 CC-B1
> 짝 plan: `docs/superpowers/plans/2026-06-30-p2-1b-json-helpers-unification.md`

---

## 1. Goal

`lib/json_helpers.sh` 의 **jq/python3 이중 백엔드를 단일 python 백엔드로** 통합한다. 7개 공개 함수가 각각 jq 경로 + python 경로 2벌을 유지하는 중복을 제거하고, 로직을 `scripts/json_backend.py`(subcommand dispatch) 한 곳으로 모은다. python write 는 **atomic(tmp+os.replace)** 으로 만들어 jq 경로가 갖던 원자성을 보존한다.

---

## 2. 왜 단일화가 안전한가 (경험적 검증 완료)

두 백엔드를 실사용 패턴(read scalar/missing/nested-numeric-key, stdin, array_len, write int/bool/str/newkey, merge, get_result, has_error)에 대해 직접 비교 → **boolean `false` read 한 가지를 빼고 전부 byte-identical**.

- **유일 divergence**: `cb_json_read` 가 boolean `false` 를 읽을 때 — jq 경로는 `jq '... // empty'` 의 `//` 가 `false` 도 empty 로 처리해 **default 반환(잠복 버그)**, python 은 `"false"` 정상 반환.
- **영향 없음**: `cb_json_read` 호출처는 pids.json 노드 메타(포트/pid 등 숫자·문자열)뿐 — boolean 필드는 저장/조회되지 않음. 따라서 잠복 버그는 현재 미발현.
- → python-only 전환은 **모든 실사용에 behavior-identical** + jq false-read 잠복 버그 수정(부수 효과).

현재 jq 설치 환경에서 기본 백엔드가 jq 이므로, 통합 후 출력은 "현재(jq) 출력과 동일"해야 하고, 위 검증이 그것을 보장한다(false 케이스만 의도적 개선).

---

## 3. 비-목표

- jq 자체를 프로젝트에서 제거하지 않음 — 테스트·`cmd_test.sh` 등은 여전히 jq 사용. 본 작업은 **json_helpers 내부 이중 백엔드만** 제거.
- `cb_hex_to_dec`/`cb_dec_to_hex` 불변(이중 백엔드 아님 — bash + python fallback, 그대로).
- 새 기능/인터페이스 변경 0 — 7함수의 argv/stdout/exit 계약 불변.

---

## 4. 설계

`scripts/json_backend.py <subcommand> <args...>`:

| subcommand | 대응 함수 | 비고 |
|---|---|---|
| `read <file> <dotpath> <default>` | cb_json_read | 기존 python read 로직 그대로 |
| `read-stdin <dotpath> <default>` | cb_json_read_stdin | JSON 을 stdin 으로 |
| `array-len <file> <dotpath>` | cb_json_array_len | |
| `write <file> <dotpath> <value>` | cb_json_write | **atomic**(tmp + os.replace) |
| `merge <file> <override_json>` | cb_json_merge | **atomic** |
| `get-result <json>` | cb_json_get_result | .result / .error → stderr+exit1 |
| `has-error <json>` | cb_json_has_error | exit code only |

`json_helpers.sh`: 백엔드 감지(`_CB_JSON_BACKEND`)·`case` 분기·jq 경로·`_cb_dot_to_jq`·`_cb_auto_type_jq` 전부 제거. 7함수는 thin wrapper(`python3 "${_CB_JSON_SCRIPTS_DIR}/json_backend.py" <sub> ...`). 파일 헤더 주석 "Uses jq when available" → "single python backend".

---

## 5. 안전망 (test-first)

`tests/unit/tests/json-helpers-contract.sh`: 7함수의 실사용 계약을 잠그는 characterization. **추출 전 현재 코드(jq)에서 green**, 추출 후 동일 green.
- read: string/int/bool(true)/missing→default/nested numeric key(`nodes.1.x`)/top-level.
- **bool false read**: python 정상 동작(`"false"`)을 명시적으로 잠금 — 통합으로 고쳐지는 잠복 버그를 문서화.
- read_stdin, array_len, write(각 타입 후 read-back), merge(후 read-back), get_result(ok/err), has_error(yes/no).
- atomic write: 큰 값 다회 write 후 파일이 valid JSON 유지.

기존 안전망: pids/remote/rpc 관련 유닛 + 전체 regression.

---

## 6. Error / 계약 불변

- read/array_len: 누락→default/0, exit 동일.
- write/merge: 성공 0, 실패 1. atomic 으로 부분쓰기 없음(개선).
- get_result: error→stderr+exit1, result 출력 동일.
- has_error: exit 0(있음)/1(없음).

---

## 7. Out-of-Scope / 후속

- P2-2 bash 대형 파일 분할.
- jq 의존 제거(프로젝트 전역) — 별도 판단.

---

## 8. 예상 커밋 (~4)

1. `docs: add P2-1b spec + plan for json_helpers unification`
2. `test(json): contract tests locking json_helpers behavior (both backends)`
3. `refactor(json): single python backend via scripts/json_backend.py (atomic writes)`
4. `docs+chore(p2-1b): refactoring-plan + remaining-work + version bump`
