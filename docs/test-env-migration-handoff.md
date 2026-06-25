# 테스트 환경 통합 — 진행 상태 & 남은 작업 (핸드오프)

브랜치 `feat/unified-test-env` (worktree `chainbench-test-refactor`). 로컬/폐쇄망을
**테스트 본문 한 벌 + 런타임 프로파일**로 통합. 설계 배경/비밀경계는
`packages/chainbench/docs/MIGRATION-unified-test-env.md` 참조.

---

## ✅ 완료 (라이브 GREEN, 커밋됨)

| 항목 | 결과 |
|---|---|
| 프레임워크: env 프로파일·accessors·constants·prims(도구추상화)·서명/노드제어 백엔드 | 실증 |
| a-ethereum (a2/a3/a4) | 17/17 |
| b-wbft | 12/12 |
| c-anzeon | 7/7 |
| d-fee-delegation | 4/4 |
| e-blacklist-authorized | 9/9 |
| g-api | 23/23 |
| **누적** | **72 테스트 라이브 통과** |

핵심 산출물:
- `tests/env/` 프로파일(local/closednet) + `secret/`(gitignore) + accessors `node/acct_addr/validator_*/tx_send_as`
- `tests/regression/lib/`: `constants.sh`, `prims.sh`(cast↔python 자동감지), `sign_backends.sh`, `node_ctrl/{local,closednet}.sh`
- `common.sh` python→cast/jq 전환 + 프로파일 배선 + EN키/BLS 스크럽
- 변환 스크립트: 세션 스크래치패드 `convert.pl` (§"변환 방법" 참조)

---

## ⛔ 최우선 블로커 — 공유 체인 재초기화

작업 중 **다른 세션이 MCP 로컬 체인을 수 분마다 `default` 프로파일로 재초기화·정지**시킴(이번 세션에서 4회+ 확인). 증상: 긴 카테고리 실행 중 앞부분 통과 → 갑자기 `unlock_validator`/`get_coinbase` 실패. `default` 프로파일은 **테스트 계정을 펀딩하지 않음**.

→ **남은 검증을 끝내려면 체인 독점 확보 필수**(다른 세션의 chainbench 자동 init/사용 중단).
짧은 카테고리(≤~10)는 클로버 전 완료돼 검증됐고, 긴 것(f=27, h=40)이 막힘.

---

## ⏳ 남은 작업

### 1. f-system-contracts — f4/f5 검증
- 변환 완료·커밋됨. **f1/f2/f3 통과**, **f4/f5(minter/blacklist 거버넌스) 미검증**(클로버로 중단).
- 재개: fresh 체인에서 `run-all.sh f-system-contracts`. f4/f5는 stateful(거버넌스) → fresh 필수.

### 2. h-hardfork (40) — 미실행
- 변환은 됨(커밋됨, 미검증 명시). **실행 검증 안 됨.**
- 확인 필요: Boho hardfork 동작 테스트라 `hardfork-boho-{pre,post,delayed}` 프로파일이 필요한지, `regression` 프로파일로 충분한지. 프로파일별로 init→run 필요할 수 있음.
- `P256_PRECOMPILE` 등 일부가 constants와 충돌 가능(이미 readonly 제거로 완화).

### 3. z-layer2-e2e (5) — cast-rewire 필요
- **되돌려둠**(변환 안 함). 테스트가 "via cast"로 설계되어 `cast`를 직접 요구 → 이 머신엔 cast 없어 실패.
- 작업: 본문을 `tx_send_as`/prims 추상화로 재배선하거나, layer2 대상(원격 alias) 정의 확인.

### 4. bespoke-python 서명 테스트 — 폐쇄망 포팅
- 로컬은 python으로 통과하지만 **폐쇄망(python 없음)엔 근본 재작성 필요**:
  - 특수 tx 서명: `a2-05a`(underpriced tip), `a2-09`(replacement), `a2-10`(setcode) — **되돌려둠**.
  - 수수료 위임: `d-*`, `e-03`, `z-05`, `g5-01` — 현재 `fee_delegate.py`(env-키 교정됨) 사용. 노드측 `personal_signTransaction` 지원 시 그쪽, 미지원 시 secret-store 키 fallback(§핸드오프 원문 3.5).
  - 다수 카테고리의 `python3 <<PYEOF` 블록(c/e/f 등): 로컬 통과 중이나 폐쇄망 위해 jq/cast/노드서명으로 포팅.
  - ws subscribe: `a4-06`, `a4-07` — **되돌려둠**(웹소켓 구독, 별도).

### 5. a1-* (노드 수명주기)
- a1-01~07(genesis/sync/restart/p2p). 변환 안 함. 노드 stop/start(`chainbench.sh node`)를 쓰므로 환경별 처리 검토.

---

## 🔁 검증된 작업 루프 (재개용)

```bash
# 0) 체인 독점 확보 (다른 세션 중단)
# 1) MCP로 regression 체인 fresh init+start
#    init: profile=regression, project_root=<gstable repo>, binary_path=<.../build/bin/gstable>
#    (이번 세션: /Users/wm-it-25_0220/Work/github/test/dev-test/pr-77)
# 2) worktree를 실행 노드에 배선
cd <worktree>; export CHAINBENCH_DIR="$PWD"
cp ~/.chainbench/state/pids.json state/pids.json      # state/는 gitignore
# 3) 카테고리 변환(미변환분) + 검증
perl <scratch>/convert.pl tests/regression/<cat>/*.sh   # glob 인자로! (zsh 단어분할 안 됨)
CHAINBENCH_TEST_ENV=local bash tests/regression/run-all.sh <cat>
# 4) GREEN 확인 → 비밀스캔 → 커밋
```

### 함정/교훈 (필독)
- **fresh 체인 per stateful 카테고리**: validator/blacklist/minter 변경 테스트는 상태가 누적돼 2회차 실패. 카테고리마다 fresh init.
- **체인 프로파일은 `regression`** (chainId 0x205b). `default`는 test 계정 미펀딩.
- **`CHAINBENCH_DIR` 환경변수가 설치본(~/.chainbench)을 가리킴** → 실행 시 worktree로 export 필수.
- **macOS엔 cast/timeout 없음**: prims가 python 백엔드로 폴백. `timeout` 대신 그냥 직접 실행(ws 테스트만 hang 주의).
- **`convert.pl`은 glob 인자로** 호출(변수 unquoted는 zsh에서 단어분할 안 됨).
- **bespoke python heredoc은 convert.pl이 안 건드림** → 로컬은 동작, 폐쇄망 포팅 별도.

---

## 변환 패턴 (convert.pl이 적용)
`"1"`→`$(node 1)` · `$TEST_ACC_X_ADDR`→`$(acct_addr N)` · `send_raw_tx "T" "$..._PK"`→`tx_send_as N` ·
`python3 -c ".get('F','')"`→`jq -r '.F // empty'` · `21000`등→constants · `python_packages` 메타 제거 ·
`check_env` 뒤 `ensure_nodes_running`. 기준 예시: `a2-01-legacy-tx.sh`.

## 커밋 (브랜치 feat/unified-test-env)
49c107e 토대+lib · 0b7aa63 백엔드 · f4a655a prims · 996f1bd/9f5bbad/602a659 a-ethereum ·
a761c30 b-wbft(+EPOCH 프로파일) · 1dd30ef c-anzeon · d190492 g-api · 152989e d-fee(+PYTHONWARNINGS) ·
e-blacklist · f-system-contracts(f1/2/3 검증).
