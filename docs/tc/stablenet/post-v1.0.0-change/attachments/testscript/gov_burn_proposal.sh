#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# GovMinter Burn Proposal CLI Script
# Contract: GovMinter (v2) at 0x1003
# =============================================================================

GOV_MINTER="0x0000000000000000000000000000000000001003"
RPC_URL="http://172.21.132.1:8601"
PRIVATE_KEY="<REDACTED_PRIVATE_KEY>"


# --------------- Gas 최소값 (StableNet 프로토콜 파라미터) ---------------
# MinBaseFee:    20_000_000_000_000 wei (params/protocol_params.go:135)
# InitialGasTip: 27_600_000_000_000 wei (params/protocol_params.go:138)
MIN_BASE_FEE="20000000000000"
MIN_GAS_TIP="27600000000000"

# --------------- 환경 변수 (필수) ---------------
# RPC_URL        : RPC 엔드포인트
# PRIVATE_KEY    : 거버넌스 멤버 private key
# -------------------------------------------------

check_env() {
    local missing=()
    [[ -z "${RPC_URL:-}" ]] && missing+=("RPC_URL")
    [[ -z "${PRIVATE_KEY:-}" ]] && missing+=("PRIVATE_KEY")
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo "ERROR: 다음 환경 변수를 설정하세요: ${missing[*]}" >&2
        echo "  export RPC_URL=\"http://172.21.132.15:8545\"" >&2
        echo "  export PRIVATE_KEY=\"0x...\"" >&2
        exit 1
    fi
}

get_sender() {
    cast wallet address "$PRIVATE_KEY"
}

# RPC에서 현재 baseFee를 조회하고, 최소값과 비교하여 gas 파라미터를 결정
resolve_gas_params() {
    local base_fee_hex
    base_fee_hex=$(cast rpc eth_getBlockByNumber "latest" "false" --rpc-url "$RPC_URL" \
        | jq -r '.baseFeePerGas')

    local base_fee
    base_fee=$(cast to-dec "$base_fee_hex")

    # baseFee가 프로토콜 최소값보다 낮으면 최소값 사용
    if [[ "$base_fee" -lt "$MIN_BASE_FEE" ]]; then
        base_fee="$MIN_BASE_FEE"
    fi

    local gas_tip
    gas_tip=$(cast rpc eth_maxPriorityFeePerGas --rpc-url "$RPC_URL" | jq -r '.')
    gas_tip=$(cast to-dec "$gas_tip")

    # gasTip이 프로토콜 최소값보다 낮으면 최소값 사용
    if [[ "$gas_tip" -lt "$MIN_GAS_TIP" ]]; then
        gas_tip="$MIN_GAS_TIP"
    fi

    # feeCap = baseFee + gasTip
    GAS_TIP="$gas_tip"
    GAS_PRICE=$(( base_fee + gas_tip ))

    echo "[gas] baseFee=${base_fee} tip=${GAS_TIP} feeCap=${GAS_PRICE}"
}

# =============================================================================
# 1. proposeBurn — Burn Proposal 생성
# =============================================================================
cmd_propose_burn() {
    local amount_ether="${1:?Usage: $0 propose-burn <amount_ether> <withdrawal_id> <reference_id> [memo]}"
    local withdrawal_id="${2:?Usage: $0 propose-burn <amount_ether> <withdrawal_id> <reference_id> [memo]}"
    local reference_id="${3:?Usage: $0 propose-burn <amount_ether> <withdrawal_id> <reference_id> [memo]}"
    local memo="${4:-}"

    local sender
    sender=$(get_sender)
    local amount_wei
    amount_wei=$(cast to-wei "$amount_ether")
    local timestamp
    timestamp=$(date +%s)

    echo "=== proposeBurn ==="
    echo "  sender:        $sender"
    echo "  amount:        ${amount_ether} ether (${amount_wei} wei)"
    echo "  withdrawalId:  $withdrawal_id"
    echo "  referenceId:   $reference_id"
    echo "  memo:          $memo"
    echo "  timestamp:     $timestamp"
    echo ""

    local proof_data
    proof_data=$(cast abi-encode \
        "f(address,uint256,uint256,string,string,string)" \
        "$sender" \
        "$amount_wei" \
        "$timestamp" \
        "$withdrawal_id" \
        "$reference_id" \
        "$memo")

    cast send "$GOV_MINTER" \
        "proposeBurn(bytes)(uint256)" \
        "$proof_data" \
        --value "${amount_ether}ether" \
        --gas-price "$GAS_PRICE" \
        --priority-gas-price "$GAS_TIP" \
        --private-key "$PRIVATE_KEY" \
        --rpc-url "$RPC_URL"
}

# =============================================================================
# 2. cancelProposal — Proposal 취소
# =============================================================================
cmd_cancel() {
    local proposal_id="${1:?Usage: $0 cancel <proposal_id>}"

    echo "=== cancelProposal ==="
    echo "  proposalId: $proposal_id"
    echo ""

    cast send "$GOV_MINTER" \
        "cancelProposal(uint256)" \
        "$proposal_id" \
        --gas-price "$GAS_PRICE" \
        --priority-gas-price "$GAS_TIP" \
        --private-key "$PRIVATE_KEY" \
        --rpc-url "$RPC_URL"
}

# =============================================================================
# 3. claimBurnRefund — 취소된 Proposal 자산 Claim
# =============================================================================
cmd_claim() {
    echo "=== claimBurnRefund ==="
    echo "  sender: $(get_sender)"
    echo ""

    cast send "$GOV_MINTER" \
        "claimBurnRefund()" \
        --gas-price "$GAS_PRICE" \
        --priority-gas-price "$GAS_TIP" \
        --private-key "$PRIVATE_KEY" \
        --rpc-url "$RPC_URL"
}

# =============================================================================
# 4. approveProposal — YES 투표
# =============================================================================
cmd_approve() {
    local proposal_id="${1:?Usage: $0 approve <proposal_id>}"

    echo "=== approveProposal (YES) ==="
    echo "  proposalId: $proposal_id"
    echo ""

    cast send "$GOV_MINTER" \
        "approveProposal(uint256)" \
        "$proposal_id" \
        --gas-price "$GAS_PRICE" \
        --priority-gas-price "$GAS_TIP" \
        --private-key "$PRIVATE_KEY" \
        --rpc-url "$RPC_URL"
}

# =============================================================================
# 5. disapproveProposal — NO 투표
# =============================================================================
cmd_disapprove() {
    local proposal_id="${1:?Usage: $0 disapprove <proposal_id>}"

    echo "=== disapproveProposal (NO) ==="
    echo "  proposalId: $proposal_id"
    echo ""

    cast send "$GOV_MINTER" \
        "disapproveProposal(uint256)" \
        "$proposal_id" \
        --gas-price "$GAS_PRICE" \
        --priority-gas-price "$GAS_TIP" \
        --private-key "$PRIVATE_KEY" \
        --rpc-url "$RPC_URL"
}

# =============================================================================
# Usage
# =============================================================================
usage() {
    cat <<USAGE
Usage: $0 <command> [args...]

Commands:
  propose-burn <amount_ether> <withdrawal_id> <reference_id> [memo]
      Burn proposal 생성 (amount 만큼 native coin 예치)

  cancel <proposal_id>
      Proposal 취소 (다른 멤버 투표 전에만 가능)

  claim
      취소/만료된 proposal의 예치금 환불

  approve <proposal_id>
      Proposal에 YES 투표

  disapprove <proposal_id>
      Proposal에 NO 투표

Environment:
  RPC_URL       RPC endpoint (e.g. http://172.21.132.15:8545)
  PRIVATE_KEY   Governance member private key

Examples:
  export RPC_URL="http://172.21.132.15:8545"
  export PRIVATE_KEY="0xac0974..."

  $0 propose-burn 100 "WD-20260413-001" "REF-20260413-001" "test burn"
  $0 approve 1
  $0 disapprove 1
  $0 cancel 1
  $0 claim
USAGE
}

# =============================================================================
# Main
# =============================================================================
main() {
    local cmd="${1:-}"
    shift || true

    case "$cmd" in
        propose-burn)  check_env; resolve_gas_params; cmd_propose_burn "$@" ;;
        cancel)        check_env; resolve_gas_params; cmd_cancel "$@" ;;
        claim)         check_env; resolve_gas_params; cmd_claim "$@" ;;
        approve)       check_env; resolve_gas_params; cmd_approve "$@" ;;
        disapprove)    check_env; resolve_gas_params; cmd_disapprove "$@" ;;
        -h|--help|"")  usage ;;
        *)             echo "Unknown command: $cmd" >&2; usage; exit 1 ;;
    esac
}

main "$@"
