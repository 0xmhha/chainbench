#!/usr/bin/env bash
# tests/env/profile.sh — 테스트 환경 프로파일 로더
#
# CHAINBENCH_TEST_ENV=local(기본)|closednet 에 따라 환경 토폴로지/배선을 로드한다.
#   - 체인 상수(chainId/epoch/gas)는 여기서 다루지 않는다 → 결정#3: tests/regression/lib/constants.sh 공용.
#   - 비밀(private key / SSH / 노드 IP)은 여기서 로드하지 않는다.
#     서명·노드제어 백엔드가 필요 시점에 secret store(tests/env/secret/)에서 직접 lazy-load 한다.
#
# 프로파일(.env)이 정의하는 심볼:
#   CB_NODE_COUNT, CB_TARGET_FMT, CB_SIGN_BACKEND, CB_NODECTRL_BACKEND,
#   CB_VALIDATOR_KEYSTORE_PASSWORD, CB_ACCT_ADDR[], CB_VALIDATOR_ADDR[], (local 한정) CB_ACCT_PK[]

[[ -n "${_CB_TEST_PROFILE_LOADED:-}" ]] && return 0
_CB_TEST_PROFILE_LOADED=1

CB_ENV_DIR="${CB_ENV_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
CB_SECRET_DIR="${CB_SECRET_DIR:-${CB_ENV_DIR}/secret}"
CHAINBENCH_TEST_ENV="${CHAINBENCH_TEST_ENV:-local}"

case "$CHAINBENCH_TEST_ENV" in
  local)     source "${CB_ENV_DIR}/local.env" ;;
  closednet) source "${CB_ENV_DIR}/closednet.env" ;;
  *)
    printf '[ERROR] unknown CHAINBENCH_TEST_ENV=%s (expected: local|closednet)\n' \
      "$CHAINBENCH_TEST_ENV" >&2
    return 1
    ;;
esac

export CHAINBENCH_TEST_ENV CB_ENV_DIR CB_SECRET_DIR
