# P2-1 — profile.sh embedded-Python extraction

> 작성일: 2026-06-29
> 상태: SPEC (검토 대기)
> 선행: `docs/REFACTORING_PLAN.md` §2.3 CC-B1 / §6.2 P2-1 · P1-4a (stablenet chain_id)
> 짝 plan: `docs/superpowers/plans/2026-06-29-p2-1-profile-python-extraction.md`
> 범위 확정(사용자 2026-06-29): **profile.sh 추출만**. json_helpers 이중백엔드 단일화는 순수 behavior-preserving 이 아니어서(atomic jq write vs in-place python) **후속/별도 설계**로 연기.

---

## 1. Goal

`lib/profile.sh` 의 임베디드 Python 두 덩이를 `scripts/` 독립 파일로 **verbatim 추출**하고, profile.sh 는 thin 래퍼로만 호출하게 한다. 동작은 100% 불변(behavior-preserving). 524줄 파일에서 ~270줄 Python 제거 → 권장치(400)에 근접.

- `_cb_python_merge_yaml`(31–279, ~240줄, `<<'PYEOF'`) → `scripts/merge_profile.py`
- `_cb_jq_get`(312–346, `<<'PYEOF'`) → `scripts/extract_json.py`

추출 안전성(조사 확정): 두 heredoc 모두 **quoted(`'PYEOF'`)** → shell 보간 0. 입력은 전부 `sys.argv`, env 읽기 없음(CHAINBENCH_DIR 도 arg). → 줄 단위 그대로 옮기고 wrapper 만 교체하면 동작 동일.

---

## 2. Non-goals

- **json_helpers.sh 단일화 안 함** — 별도 설계(atomic write 보존). 이번 범위 밖.
- **stablenet.sh genesis Python 추출 안 함** — 231줄 adapter genesis 로직은 별도. 단 P1-4a(아래)만 흡수.
- **YAML 파서/merge 로직 변경 0** — 순수 위치 이동. 개선/리팩토는 추후.
- **mktemp/cleanup/export 흐름 변경 0** — `load_profile` 데이터 흐름 불변.

---

## 3. 안전망 우선 (test-first — 위험 高 sprint 의 핵심)

추출 **전에** 현재 동작을 잠그는 characterization 테스트를 추가하고 현재 코드에서 green 확인. 추출 후 동일 테스트가 green → behavior-preserving 증명. 조사로 식별된 커버리지 갭:

| 테스트 | 잠그는 동작 |
|---|---|
| inheritance chain | `extends: regression` 등 2+단계 상속 병합(부모+자식 필드 모두) |
| circular inheritance | A↔B 순환 → depth 10 초과 에러 |
| missing parent | 존재 않는 `extends` → 에러 + load 실패 |
| validation failure | `.chain.binary`/`.nodes.validators` 누락 → `load_profile` 실패 |
| deep merge | overlay 의 nested(systemContracts 등) 병합이 형제 키 보존 |
| YAML quotes/coerce | `"/path with space"` 따옴표 해제, true/false/null/int 캐스팅 |

기존 안전망(유지): `profile-env-override.sh`, `profile-overlay-merge.sh`, `adapter-mapping.sh` + 전체 regression.

---

## 4. P1-4a 흡수 (chain_id SSoT)

`lib/adapters/stablenet.sh:45` 의 `chain.get("chain_id", 8283)` 하드코딩 fallback → `lib/defaults.generated.sh` 의 `CB_STABLENET_CHAIN_ID` 를 heredoc argv 로 주입해 SSoT 단일화. **작은 타깃 변경**(stablenet.sh genesis 로직 자체는 불변). 그 외 wbft consensus 기본값(requestTimeout/blockPeriod/epochLength)은 consensus-specific 이라 이번 범위 밖(후속).

---

## 5. 추출 설계

```bash
# lib/profile.sh (refactored — thin wrapper)
_cb_python_merge_yaml() {
  local profile_path="${1:?...}"
  python3 "${_CB_SCRIPTS_DIR}/merge_profile.py" \
    "$profile_path" "$_CB_PROFILES_DIR" "${CHAINBENCH_DIR:-}"
}
_cb_jq_get() {
  python3 "${_CB_SCRIPTS_DIR}/extract_json.py" "$1" "$2" "${3:-}"
}
```
- `_CB_SCRIPTS_DIR` = repo `scripts/` (profile.sh 가 자기 위치 기준 해석; `${BASH_SOURCE}` 패턴).
- `scripts/merge_profile.py`, `scripts/extract_json.py` 는 기존 heredoc 본문 **그대로**(argv 인덱스 동일, stdout/exit 계약 동일).
- 두 스크립트에 모듈 docstring + `#!/usr/bin/env python3` + (선택) `__main__` 가드. 동작 영향 0.

---

## 6. Error / 계약 불변

- merge: stdout=minified JSON, 실패 시 stderr `ERROR: ...` + exit 1 (동일).
- extract: stdout=값/default, exit 0 (동일).
- `load_profile` 의 `python_output="$(_cb_python_merge_yaml ...)"` 캡처·validate·export·cleanup 전부 불변.

---

## 7. Tests (요약)

1. 신규 characterization 6종(§3) — 추출 전 green, 추출 후 green.
2. 기존 profile/adapter/overlay 유닛 + 전체 regression 무회귀.
3. (선택) 추출 전/후 동일 profile 들의 merged JSON 골든 비교(byte-identical): `default`, `regression`, `hardfork-boho-pre`(상속), `minimal` — 가장 강한 behavior-preserving 증거.

---

## 8. Out-of-Scope / 후속

- json_helpers 단일화(atomic python backend) — 별도 sprint.
- stablenet.sh genesis Python 전체 추출.
- profile YAML 파서 개선(PyYAML 강제 등).

---

## 9. 예상 커밋 (~5-6)

1. `docs: add P2-1 spec + plan for profile.sh python extraction`
2. `test(profile): characterization tests locking current merge/inherit/validate behavior`
3. `refactor(profile): extract merge_profile.py + extract_json.py to scripts/`
4. `refactor(adapters): source stablenet chain_id default from SSoT (P1-4a)`
5. `docs+chore(p2-1): refactoring-plan + remaining-work + version bump`
