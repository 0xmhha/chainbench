# P2-1 — profile.sh python extraction — Plan

> 짝 spec: `2026-06-29-p2-1-profile-python-extraction.md` · 범위: profile.sh 추출만 (json_helpers 후속).
> 전략: **test-first** (현재 동작 잠금 → 추출 → 동일 green). 커밋: English, no co-author, no emoji.

---

## Task 0 — spec + plan 커밋
- 브랜치 `refactor/p2-1-profile-python-extract`.
- commit: `docs: add P2-1 spec + plan for profile.sh python extraction`

## Task 1 — 안전망: characterization 테스트 (추출 전, 현재 코드에서 green)
- **골든 JSON 동등성** `tests/unit/tests/profile-merge-golden.sh`: `default`/`regression`/`hardfork-boho-pre`(상속)/`minimal` 각각 `_cb_python_merge_yaml` 출력을 커밋된 골든(`tests/unit/golden/profile-merged/<name>.json`)과 byte-identical 비교. (CHAINBENCH_DIR 고정, overlay 없는 상태로 결정론 보장.)
  - 골든 파일은 **현재 코드 출력으로 생성**(이게 잠그는 기준선).
- **에러/비출력 동작** `tests/unit/tests/profile-load-contracts.sh`:
  - missing parent(`extends: __nope__`) → load 실패 + 에러.
  - validation 실패(필수필드 누락 프로파일 fixture) → `load_profile` 비-0.
  - YAML quote/coerce: 따옴표 경로/bool/null 캐스팅이 merged JSON 에 올바른 타입.
  - circular(A↔B fixture) → depth-10 에러.
- 현재 코드로 전부 green 확인(기준선 확정).
- commit: `test(profile): characterization + golden tests locking merge/inherit/validate`

## Task 2 — 추출: scripts/merge_profile.py + scripts/extract_json.py
- `scripts/merge_profile.py`: profile.sh:36–277 의 Python 본문 **그대로** + shebang + docstring. argv[1..3] 동일.
- `scripts/extract_json.py`: profile.sh:317–345 본문 그대로 + shebang + docstring.
- `lib/profile.sh`: 두 함수를 thin wrapper 로 교체. `_CB_SCRIPTS_DIR` 해석(`${BASH_SOURCE[0]}` → repo `scripts/`). 두 heredoc 삭제.
- 실행권한(`chmod +x`) — 단 `python3 <path>` 호출이라 필수는 아님(일관성).
- **검증**: Task 1 골든/contracts + profile-env-override + profile-overlay-merge + adapter-mapping + 전체 regression 전부 green(=byte-identical 동작).
- commit: `refactor(profile): extract merge_profile.py + extract_json.py to scripts/`

## Task 3 — P1-4a: stablenet chain_id SSoT
- `lib/adapters/stablenet.sh`: chain_id fallback `8283` 를 `CB_STABLENET_CHAIN_ID`(defaults.generated.sh) 에서 heredoc argv 로 주입. (`source` 후 argv 추가 → Python `chain.get("chain_id", int(sys.argv[N]))`.)
- 그 외 wbft 기본값은 불변(범위 밖).
- **검증**: adapter-mapping + genesis 관련 regression green. merged genesis 의 chainId 불변(8283).
- commit: `refactor(adapters): source stablenet chain_id default from SSoT (P1-4a)`

## Task 4 — 문서 + 버전
- REFACTORING_PLAN §3 표 P2-1 부분완료(profile.sh ✅, json_helpers 후속) + §6.2 갱신(P1-4a ✅, json_helpers 잔여 분리).
- REMAINING_WORK: P2-1 부분완료 반영.
- 버전 0.10.0 → 0.11.0(또는 patch — 동작 불변 refactor면 minor 불필요, 0.10.1). **0.10.1 patch** 권장(behavior-preserving).
- commit: `docs+chore(p2-1): refactoring-plan + remaining-work + version 0.10.1`

---

## 완료 기준
- [ ] 골든 JSON: 4개 프로파일 merged 출력이 추출 전후 byte-identical
- [ ] 에러 계약(missing parent/validation/circular) 테스트 green
- [ ] profile.sh 에서 임베디드 Python 0 (두 heredoc 제거, scripts/ 로 이동)
- [ ] P1-4a: stablenet chain_id 가 SSoT 참조, genesis chainId 불변
- [ ] 전 레이어 green: Go · vitest · bash · 전체 regression(가능 범위)
- [ ] profile.sh 줄 수 524 → ~270 대
