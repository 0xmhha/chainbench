#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# GovMinter Query Script — contract state & tx/block 조회
# Contract: GovMinter (v2) at 0x1003
# =============================================================================

GOV_MINTER="0x0000000000000000000000000000000000001003"
RPC_URL="http://172.21.132.1:8601"

# ProposalStatus enum (GovBase.sol:96-105)
PROPOSAL_STATUS=(
    "None"
    "Voting"
    "Approved"
    "Executed"
    "Cancelled"
    "Expired"
    "Failed"
    "Rejected"
)

# --------------- 환경 변수 (필수) ---------------
# RPC_URL : RPC 엔드포인트
# -------------------------------------------------

check_env() {
    if [[ -z "${RPC_URL:-}" ]]; then
        echo "ERROR: RPC_URL 환경 변수를 설정하세요" >&2
        echo "  export RPC_URL=\"http://172.21.132.15:8545\"" >&2
        exit 1
    fi
}

# =============================================================================
# Proposal 조회
# =============================================================================

# getProposal — Proposal 전체 상세 조회
cmd_proposal() {
    local id="${1:?Usage: $0 proposal <proposal_id>}"

    echo "=== getProposal($id) ==="
    local raw
    raw=$(cast call "$GOV_MINTER" \
        "getProposal(uint256)((bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8,bytes))" \
        "$id" \
        --rpc-url "$RPC_URL")

    # getProposal 반환 순서:
    #   0: actionType (bytes32)
    #   1: memberVersion (uint256)
    #   2: votedBitmap (uint256)
    #   3: createdAt (uint256)
    #   4: executedAt (uint256)
    #   5: proposer (address)
    #   6: requiredApprovals (uint32)
    #   7: approved (uint32)
    #   8: rejected (uint32)
    #   9: status (uint8) — enum ProposalStatus
    #  10: callData (bytes)
    echo "--- raw output ---"
    echo "$raw"
    echo ""

    # status enum 해석
    local status_num
    status_num=$(echo "$raw" | sed -n '10p' | tr -d ' ')
    if [[ "$status_num" =~ ^[0-9]+$ ]] && [[ "$status_num" -lt ${#PROPOSAL_STATUS[@]} ]]; then
        echo "  status: ${PROPOSAL_STATUS[$status_num]} ($status_num)"
    fi
}

# isProposalInVoting — 투표 진행 중인지 확인
cmd_is_voting() {
    local id="${1:?Usage: $0 is-voting <proposal_id>}"

    echo "=== isProposalInVoting($id) ==="
    cast call "$GOV_MINTER" \
        "isProposalInVoting(uint256)(bool)" \
        "$id" \
        --rpc-url "$RPC_URL"
}

# isProposalExecutable — 실행 가능 여부 확인
cmd_is_executable() {
    local id="${1:?Usage: $0 is-executable <proposal_id>}"

    echo "=== isProposalExecutable($id) ==="
    cast call "$GOV_MINTER" \
        "isProposalExecutable(uint256)(bool)" \
        "$id" \
        --rpc-url "$RPC_URL"
}

# canExecuteProposal — 실행 가능 여부 + 상세 사유
cmd_can_execute() {
    local id="${1:?Usage: $0 can-execute <proposal_id>}"

    echo "=== canExecuteProposal($id) ==="
    cast call "$GOV_MINTER" \
        "canExecuteProposal(uint256)(uint8,uint256)" \
        "$id" \
        --rpc-url "$RPC_URL"
}

# =============================================================================
# Contract State 조회
# =============================================================================

# currentProposalId — 최신 proposal ID
cmd_proposal_count() {
    echo "=== currentProposalId ==="
    cast call "$GOV_MINTER" "currentProposalId()(uint256)" --rpc-url "$RPC_URL"
}

# proposalExpiry — proposal 만료 기간 (초)
cmd_expiry() {
    echo "=== proposalExpiry ==="
    local seconds
    seconds=$(cast call "$GOV_MINTER" "proposalExpiry()(uint256)" --rpc-url "$RPC_URL" | xargs)
    seconds="${seconds:-0}"
    echo "  ${seconds} seconds ($(( seconds / 3600 )) hours)"
}

# memberVersion — 현재 멤버 버전
cmd_member_version() {
    echo "=== memberVersion ==="
    cast call "$GOV_MINTER" "memberVersion()(uint256)" --rpc-url "$RPC_URL"
}

# quorum — 현재 멤버 버전의 의결 정족수
cmd_quorum() {
    echo "=== quorum ==="
    local ver
    ver=$(cast call "$GOV_MINTER" "memberVersion()(uint256)" --rpc-url "$RPC_URL" | awk '{print $1}')
    ver="${ver:-1}"
    local q
    q=$(cast call "$GOV_MINTER" "getQuorum(uint256)(uint32)" "$ver" --rpc-url "$RPC_URL" | awk '{print $1}')
    echo "  memberVersion: $ver"
    echo "  quorum: $q"
    echo "  (YES ${q}표 → approve, NO ${q}표 → reject)"
}

# maxActiveProposalsPerMember — 멤버당 최대 활성 proposal 수
cmd_max_active() {
    echo "=== maxActiveProposalsPerMember ==="
    cast call "$GOV_MINTER" "maxActiveProposalsPerMember()(uint256)" --rpc-url "$RPC_URL"
}

# emergencyPaused — 긴급 정지 상태
cmd_paused() {
    echo "=== emergencyPaused ==="
    cast call "$GOV_MINTER" "emergencyPaused()(bool)" --rpc-url "$RPC_URL"
}

# =============================================================================
# 멤버 / 잔액 조회
# =============================================================================

# isMember — 특정 주소의 멤버 여부
cmd_is_member() {
    local addr="${1:?Usage: $0 is-member <address> [version]}"
    local version="${2:-1}"

    echo "=== isMember($addr, $version) ==="
    cast call "$GOV_MINTER" \
        "isMember(address,uint256)(bool)" \
        "$addr" "$version" \
        --rpc-url "$RPC_URL"
}

# members — 멤버 상세 정보
cmd_member() {
    local addr="${1:?Usage: $0 member <address>}"

    echo "=== members($addr) ==="
    cast call "$GOV_MINTER" \
        "members(address)(bool,uint32)" \
        "$addr" \
        --rpc-url "$RPC_URL"
}

# burnBalance — burn 예치 잔액
cmd_burn_balance() {
    local addr="${1:?Usage: $0 burn-balance <address>}"

    echo "=== burnBalance($addr) ==="
    local wei
    wei=$(cast call "$GOV_MINTER" "burnBalance(address)(uint256)" "$addr" --rpc-url "$RPC_URL" | xargs)
    echo "  ${wei} wei ($(cast from-wei "${wei:-0}") ether)"
}

# refundableBalance — 환불 가능 잔액
cmd_refundable() {
    local addr="${1:?Usage: $0 refundable <address>}"

    echo "=== refundableBalance($addr) ==="
    local wei
    wei=$(cast call "$GOV_MINTER" "refundableBalance(address)(uint256)" "$addr" --rpc-url "$RPC_URL" | xargs)
    echo "  ${wei} wei ($(cast from-wei "${wei:-0}") ether)"
}

# memberActiveProposalCount — 멤버의 활성 proposal 수
cmd_active_count() {
    local addr="${1:?Usage: $0 active-count <address>}"

    echo "=== memberActiveProposalCount($addr) ==="
    cast call "$GOV_MINTER" \
        "memberActiveProposalCount(address)(uint256)" \
        "$addr" \
        --rpc-url "$RPC_URL"
}

# =============================================================================
# Burn/Mint Proposal 데이터 조회
# =============================================================================

# burnProposals — burn proposal 상세
cmd_burn_proposal() {
    local id="${1:?Usage: $0 burn-proposal <proposal_id>}"

    echo "=== burnProposals($id) ==="
    cast call "$GOV_MINTER" \
        "burnProposals(uint256)(uint256,address)" \
        "$id" \
        --rpc-url "$RPC_URL"
}

# withdrawalId → proposalId 매핑
cmd_withdrawal_id() {
    local wid="${1:?Usage: $0 withdrawal-id <withdrawal_id_string>}"

    echo "=== withdrawalIdToProposalId($wid) ==="
    cast call "$GOV_MINTER" \
        "withdrawalIdToProposalId(string)(uint256)" \
        "$wid" \
        --rpc-url "$RPC_URL"
}

# reservedMintAmount — 예약된 mint 총량
cmd_reserved_mint() {
    echo "=== reservedMintAmount ==="
    local wei
    wei=$(cast call "$GOV_MINTER" "reservedMintAmount()(uint256)" --rpc-url "$RPC_URL" | xargs)
    echo "  ${wei} wei ($(cast from-wei "${wei:-0}") ether)"
}

# =============================================================================
# Block / Tx 조회
# =============================================================================

# block 정보 조회
cmd_block() {
    local block_num="${1:?Usage: $0 block <number|latest>}"

    echo "=== Block: $block_num ==="
    cast block "$block_num" --rpc-url "$RPC_URL"
}

# tx hash로 트랜잭션 정보 조회
cmd_tx() {
    local tx_hash="${1:?Usage: $0 tx <tx_hash>}"

    echo "=== Transaction ==="
    cast tx "$tx_hash" --rpc-url "$RPC_URL"
}

# tx receipt 조회 (성공/실패, 가스 사용량, 로그)
cmd_receipt() {
    local tx_hash="${1:?Usage: $0 receipt <tx_hash>}"

    echo "=== Receipt ==="
    cast receipt "$tx_hash" --rpc-url "$RPC_URL"
}

# tx receipt의 logs를 디코딩하여 출력
cmd_logs() {
    local tx_hash="${1:?Usage: $0 logs <tx_hash>}"

    echo "=== Decoded Logs ==="
    cast receipt "$tx_hash" --rpc-url "$RPC_URL" | grep -A 100 "^logs"
}

# =============================================================================
# 종합 검증 — proposal 생성 tx 검증
# =============================================================================
cmd_verify_propose() {
    local tx_hash="${1:?Usage: $0 verify-propose <tx_hash>}"

    echo "=== Tx 검증: proposeBurn ==="
    echo ""

    echo "--- 1. Receipt ---"
    local status
    status=$(cast receipt "$tx_hash" --rpc-url "$RPC_URL" | grep "^status" | awk '{print $2}')
    echo "  tx status: $status"
    if [[ "$status" != "1" ]]; then
        echo "  ERROR: tx 실패" >&2
        return 1
    fi
    echo "  tx 성공"
    echo ""

    echo "--- 2. Proposal ID (currentProposalId) ---"
    local latest_id
    latest_id=$(cast call "$GOV_MINTER" "currentProposalId()(uint256)" --rpc-url "$RPC_URL")
    echo "  latestProposalId: $latest_id"
    echo ""

    echo "--- 3. Proposal 상세 ---"
    cast call "$GOV_MINTER" \
        "getProposal(uint256)((bytes32,uint256,uint256,uint256,uint256,address,uint32,uint32,uint32,uint8,bytes))" \
        "$latest_id" \
        --rpc-url "$RPC_URL"
    echo ""

    echo "--- 4. Burn Proposal 데이터 ---"
    cast call "$GOV_MINTER" \
        "burnProposals(uint256)(uint256,address)" \
        "$latest_id" \
        --rpc-url "$RPC_URL"
}

# =============================================================================
# Usage
# =============================================================================
usage() {
    cat <<USAGE
Usage: $0 <command> [args...]

Proposal 조회:
  proposal <id>           Proposal 전체 상세
  is-voting <id>          투표 진행 중 여부
  is-executable <id>      실행 가능 여부
  can-execute <id>        실행 가능 여부 + 상세 사유

Contract State:
  proposal-count          최신 proposal ID (= 총 생성 수)
  quorum                  의결 정족수 (YES/NO 몇 표로 결정)
  expiry                  proposal 만료 기간
  member-version          현재 멤버 버전
  max-active              멤버당 최대 활성 proposal 수
  paused                  긴급 정지 상태

멤버 / 잔액:
  is-member <addr> [ver]  멤버 여부 확인
  member <addr>           멤버 상세 (active, index)
  burn-balance <addr>     burn 예치 잔액
  refundable <addr>       환불 가능 잔액
  active-count <addr>     활성 proposal 수

Burn/Mint:
  burn-proposal <id>      burn proposal 데이터 (amount, requester)
  withdrawal-id <wid>     withdrawalId → proposalId 조회
  reserved-mint           예약된 mint 총량

Block / Tx:
  block <number|latest>   블록 정보
  tx <hash>               트랜잭션 정보
  receipt <hash>          트랜잭션 receipt (성공/실패, gas, logs)
  logs <hash>             트랜잭션 로그 출력

검증:
  verify-propose <hash>   proposeBurn tx 종합 검증

Environment:
  RPC_URL    RPC endpoint (e.g. http://172.21.132.15:8545)

Examples:
  $0 proposal 1
  $0 burn-balance 0x1234...
  $0 receipt 0xabcd...
  $0 verify-propose 0xabcd...
USAGE
}

# =============================================================================
# Main
# =============================================================================
main() {
    local cmd="${1:-}"
    shift || true

    case "$cmd" in
        # Proposal
        proposal)        check_env; cmd_proposal "$@" ;;
        is-voting)       check_env; cmd_is_voting "$@" ;;
        is-executable)   check_env; cmd_is_executable "$@" ;;
        can-execute)     check_env; cmd_can_execute "$@" ;;
        # Contract state
        proposal-count)  check_env; cmd_proposal_count ;;
        quorum)          check_env; cmd_quorum ;;
        expiry)          check_env; cmd_expiry ;;
        member-version)  check_env; cmd_member_version ;;
        max-active)      check_env; cmd_max_active ;;
        paused)          check_env; cmd_paused ;;
        # Member / Balance
        is-member)       check_env; cmd_is_member "$@" ;;
        member)          check_env; cmd_member "$@" ;;
        burn-balance)    check_env; cmd_burn_balance "$@" ;;
        refundable)      check_env; cmd_refundable "$@" ;;
        active-count)    check_env; cmd_active_count "$@" ;;
        # Burn/Mint
        burn-proposal)   check_env; cmd_burn_proposal "$@" ;;
        withdrawal-id)   check_env; cmd_withdrawal_id "$@" ;;
        reserved-mint)   check_env; cmd_reserved_mint ;;
        # Block / Tx
        block)           check_env; cmd_block "$@" ;;
        tx)              check_env; cmd_tx "$@" ;;
        receipt)         check_env; cmd_receipt "$@" ;;
        logs)            check_env; cmd_logs "$@" ;;
        # Verify
        verify-propose)  check_env; cmd_verify_propose "$@" ;;
        # Help
        -h|--help|"")    usage ;;
        *)               echo "Unknown command: $cmd" >&2; usage; exit 1 ;;
    esac
}

main "$@"
