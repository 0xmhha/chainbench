#!/usr/bin/env bash
# tests/regression/lib/node_ctrl/local.sh — 로컬 노드제어 백엔드 (W8)
#
# 로컬 노드는 chainbench로 미리 기동되어 있다고 가정한다. ensure는 헬스체크만(없으면 no-op).

[[ -n "${_CB_NODECTRL_LOCAL_LOADED:-}" ]] && return 0
_CB_NODECTRL_LOCAL_LOADED=1

# cb_nodectrl_local_ensure
# 로컬은 기동 가정 → no-op. (필요 시 chainbench.sh status로 헬스체크를 추가할 수 있다.)
cb_nodectrl_local_ensure() {
  :
}
