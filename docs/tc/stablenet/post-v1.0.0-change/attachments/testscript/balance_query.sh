#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Balance Query & Transfer Script — native coin & ERC20 token 잔액 조회/전송
#
# Native coin:  cast balance / cast send (eth_getBalance / eth_sendTransaction)
# ERC20 token:  balanceOf / transfer / name / symbol / decimals / totalSupply
# NCA (0x1000): NativeCoinAdapter — native coin 을 ERC20 로 표현
# =============================================================================

NCA="0x0000000000000000000000000000000000001000"
RPC_URL="${RPC_URL:-http://172.21.132.1:8601}"

# --------------- Gas 최소값 (StableNet 프로토콜 파라미터) ---------------
MIN_BASE_FEE="20000000000000"
MIN_GAS_TIP="27600000000000"

# --------------- 환경 변수 ---------------
# RPC_URL     : RPC 엔드포인트 (기본: 172.21.132.1:8601)
# PRIVATE_KEY : send/transfer 시 필요
# -----------------------------------------

check_env() {
    if [[ -z "${RPC_URL:-}" ]]; then
        echo "ERROR: RPC_URL 환경 변수를 설정하세요" >&2
        echo "  export RPC_URL=\"http://172.21.132.1:8601\"" >&2
        exit 1
    fi
}

check_env_with_key() {
    check_env
    if [[ -z "${PRIVATE_KEY:-}" ]]; then
        echo "ERROR: PRIVATE_KEY 환경 변수를 설정하세요" >&2
        echo "  export PRIVATE_KEY=\"0x...\"" >&2
        exit 1
    fi
}

# RPC에서 baseFee/gasTip 조회 후 프로토콜 최소값과 비교
resolve_gas_params() {
    local base_fee_hex
    base_fee_hex=$(cast rpc eth_getBlockByNumber "latest" "false" --rpc-url "$RPC_URL" \
        | jq -r '.baseFeePerGas')

    local base_fee
    if [[ -n "$base_fee_hex" && "$base_fee_hex" != "null" ]]; then
        base_fee=$(cast to-dec "$base_fee_hex" 2>/dev/null || echo "0")
    else
        base_fee=0
    fi

    if [[ "$base_fee" -lt "$MIN_BASE_FEE" ]]; then
        base_fee="$MIN_BASE_FEE"
    fi

    local gas_tip_hex
    gas_tip_hex=$(cast rpc eth_maxPriorityFeePerGas --rpc-url "$RPC_URL" 2>/dev/null | jq -r '.' 2>/dev/null)

    local gas_tip
    if [[ -n "$gas_tip_hex" && "$gas_tip_hex" != "null" ]]; then
        gas_tip=$(cast to-dec "$gas_tip_hex" 2>/dev/null || echo "0")
    else
        gas_tip=0
    fi

    if [[ "$gas_tip" -lt "$MIN_GAS_TIP" ]]; then
        gas_tip="$MIN_GAS_TIP"
    fi

    GAS_TIP="$gas_tip"
    GAS_PRICE=$((base_fee + gas_tip))

    echo "[gas] baseFee=${base_fee} tip=${GAS_TIP} feeCap=${GAS_PRICE}"
}

# =============================================================================
# Native Coin 조회
# =============================================================================

# native balance (wei)
cmd_native() {
    local addr="${1:?Usage: $0 native <address>}"

    echo "=== Native Balance ==="
    echo "  address: $addr"
    local wei
    wei=$(cast balance "$addr" --rpc-url "$RPC_URL" | xargs)
    wei="${wei:-0}"
    local ether
    ether=$(cast from-wei "${wei}" 2>/dev/null || echo "N/A")
    echo "  balance: ${wei} wei (${ether} ether)"
}

# 여러 주소 native balance 일괄 조회
cmd_native_multi() {
    if [[ $# -lt 1 ]]; then
        echo "Usage: $0 native-multi <addr1> [addr2] [addr3] ..." >&2
        exit 1
    fi

    echo "=== Native Balance (multi) ==="
    printf "  %-44s %28s %20s\n" "ADDRESS" "WEI" "ETHER"
    echo "  $(printf '%0.s─' {1..94})"

    for addr in "$@"; do
        local wei ether
        wei=$(cast balance "$addr" --rpc-url "$RPC_URL" | xargs)
        wei="${wei:-0}"
        ether=$(cast from-wei "${wei}" 2>/dev/null || echo "N/A")
        printf "  %-44s %28s %20s\n" "$addr" "$wei" "$ether"
    done
}

# validator 전체 native balance
cmd_validators() {
    echo "=== Validator Native Balances ==="

    local addrs=(
        "0x518b3Efa7dB538F29615Cb9d76f4ac234EBE5893"
        "0xD76975b29BDE03F4C644851393D988CcEa9d1471"
        "0x0f51a9c1E728CAfD808F570773b4DcA201E0892D"
        "0x5b3267f014De8044FFD4d45A575Bdfe1e6395060"
        "0xd997f9A99FB33a63E07220F42143F3eE0DEEDfc0"
        "0x7379433952e44Ee51D313A4581F0bB79aa6C4ff2"
        "0xb0a24793C36c8C6489B20f213bD735B9B6749447"
    )

    printf "  %-5s %-44s %28s %20s\n" "VAL#" "ADDRESS" "WEI" "ETHER"
    echo "  $(printf '%0.s─' {1..99})"

    local i=1
    for addr in "${addrs[@]}"; do
        local wei ether
        wei=$(cast balance "$addr" --rpc-url "$RPC_URL" | xargs)
        wei="${wei:-0}"
        ether=$(cast from-wei "${wei}" 2>/dev/null || echo "N/A")
        printf "  %-5s %-44s %28s %20s\n" "VAL${i}" "$addr" "$wei" "$ether"
        i=$((i + 1))
    done
}

# =============================================================================
# ERC20 Token 조회
# =============================================================================

# token info (name, symbol, decimals, totalSupply)
cmd_token_info() {
    local token="${1:?Usage: $0 token-info <token_address>}"

    echo "=== ERC20 Token Info ==="
    echo "  contract: $token"

    local name symbol decimals supply
    name=$(cast call "$token" 'name()(string)' --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    symbol=$(cast call "$token" 'symbol()(string)' --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    decimals=$(cast call "$token" 'decimals()(uint8)' --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    supply=$(cast call "$token" 'totalSupply()(uint256)' --rpc-url "$RPC_URL" 2>/dev/null | xargs)

    echo "  name:        ${name:-N/A}"
    echo "  symbol:      ${symbol:-N/A}"
    echo "  decimals:    ${decimals:-N/A}"
    echo "  totalSupply: ${supply:-N/A}"
}

# token balanceOf
cmd_token_balance() {
    local token="${1:?Usage: $0 token-balance <token_address> <account_address>}"
    local addr="${2:?Usage: $0 token-balance <token_address> <account_address>}"

    local symbol decimals
    symbol=$(cast call "$token" 'symbol()(string)' --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    decimals=$(cast call "$token" 'decimals()(uint8)' --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    symbol="${symbol:-TOKEN}"
    decimals="${decimals:-18}"

    echo "=== Token Balance ==="
    echo "  token:   ${token} (${symbol})"
    echo "  address: ${addr}"

    local raw
    raw=$(cast call "$token" 'balanceOf(address)(uint256)' "$addr" --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    raw="${raw:-0}"

    local formatted
    if [[ "$decimals" -gt 0 ]] 2>/dev/null; then
        formatted=$(awk "BEGIN { printf \"%.${decimals}f\", ${raw} / (10 ^ ${decimals}) }" 2>/dev/null || echo "N/A")
    else
        formatted="$raw"
    fi

    echo "  balance: ${raw} (${formatted} ${symbol})"
}

# 여러 주소 token balance 일괄 조회
cmd_token_multi() {
    local token="${1:?Usage: $0 token-multi <token_address> <addr1> [addr2] ...}"
    shift

    if [[ $# -lt 1 ]]; then
        echo "Usage: $0 token-multi <token_address> <addr1> [addr2] ..." >&2
        exit 1
    fi

    local symbol decimals
    symbol=$(cast call "$token" 'symbol()(string)' --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    decimals=$(cast call "$token" 'decimals()(uint8)' --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    symbol="${symbol:-TOKEN}"
    decimals="${decimals:-18}"

    echo "=== Token Balance (multi) — ${symbol} (${token}) ==="
    printf "  %-44s %28s %20s\n" "ADDRESS" "RAW" "${symbol}"
    echo "  $(printf '%0.s─' {1..94})"

    for addr in "$@"; do
        local raw formatted
        raw=$(cast call "$token" 'balanceOf(address)(uint256)' "$addr" --rpc-url "$RPC_URL" 2>/dev/null | xargs)
        raw="${raw:-0}"
        if [[ "$decimals" -gt 0 ]] 2>/dev/null; then
            formatted=$(awk "BEGIN { printf \"%.4f\", ${raw} / (10 ^ ${decimals}) }" 2>/dev/null || echo "N/A")
        else
            formatted="$raw"
        fi
        printf "  %-44s %28s %20s\n" "$addr" "$raw" "$formatted"
    done
}

# =============================================================================
# NCA (NativeCoinAdapter) 조회 — native coin의 ERC20 표현
# =============================================================================

# NCA info
cmd_nca_info() {
    echo "=== NativeCoinAdapter (NCA) ==="
    cmd_token_info "$NCA"
}

# NCA balance
cmd_nca_balance() {
    local addr="${1:?Usage: $0 nca-balance <address>}"
    cmd_token_balance "$NCA" "$addr"
}

# =============================================================================
# 종합 조회 — native + NCA 동시 비교
# =============================================================================

cmd_compare() {
    local addr="${1:?Usage: $0 compare <address>}"

    echo "=== Native vs NCA Balance 비교 ==="
    echo "  address: $addr"
    echo ""

    local native_wei nca_raw
    native_wei=$(cast balance "$addr" --rpc-url "$RPC_URL" | xargs)
    native_wei="${native_wei:-0}"
    nca_raw=$(cast call "$NCA" 'balanceOf(address)(uint256)' "$addr" --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    nca_raw="${nca_raw:-0}"

    local native_ether nca_ether
    native_ether=$(cast from-wei "${native_wei}" 2>/dev/null || echo "N/A")
    nca_ether=$(cast from-wei "${nca_raw}" 2>/dev/null || echo "N/A")

    echo "  native (eth_getBalance): ${native_wei} wei (${native_ether} ether)"
    echo "  NCA    (balanceOf):      ${nca_raw} wei (${nca_ether} ether)"

    if [[ "$native_wei" = "$nca_raw" ]]; then
        echo "  result: MATCH"
    else
        echo "  result: MISMATCH"
        local diff
        if [[ "$native_wei" -gt "$nca_raw" ]] 2>/dev/null; then
            diff=$((native_wei - nca_raw))
        else
            diff=$((nca_raw - native_wei))
        fi
        echo "  diff:   ${diff} wei"
    fi
}

# =============================================================================
# Native Coin 전송
# =============================================================================

cmd_send() {
    local to="${1:?Usage: $0 send <to_address> <amount> [unit]}"
    local amount="${2:?Usage: $0 send <to_address> <amount> [unit]}"
    local unit="${3:-ether}"

    local sender
    sender=$(cast wallet address "$PRIVATE_KEY")

    local value="${amount}${unit}"
    local amount_wei
    amount_wei=$(cast to-wei "$amount" "$unit" 2>/dev/null || echo "$amount")

    echo "=== Native Coin Send ==="
    echo "  from:   $sender"
    echo "  to:     $to"
    echo "  amount: ${amount} ${unit} (${amount_wei} wei)"
    echo "  gas:    feeCap=${GAS_PRICE} tip=${GAS_TIP}"
    echo ""

    # 전송 전 잔액
    local bal_before
    bal_before=$(cast balance "$sender" --rpc-url "$RPC_URL" | xargs)
    echo "  sender balance (before): ${bal_before} wei"

    local result
    result=$(cast send "$to" \
        --value "$value" \
        --gas-price "$GAS_PRICE" \
        --priority-gas-price "$GAS_TIP" \
        --private-key "$PRIVATE_KEY" \
        --rpc-url "$RPC_URL" \
        --json 2>&1)

    local tx_hash
    tx_hash=$(echo "$result" | jq -r .transactionHash 2>/dev/null)

    if [[ -z "$tx_hash" || "$tx_hash" = "null" ]]; then
        echo "  ERROR: 전송 실패"
        echo "  $result"
        return 1
    fi

    echo "  tx:     $tx_hash"

    # receipt 확인
    local status gas_used
    status=$(cast receipt "$tx_hash" --rpc-url "$RPC_URL" --json 2>/dev/null | jq -r .status)
    gas_used=$(cast receipt "$tx_hash" --rpc-url "$RPC_URL" --json 2>/dev/null | jq -r .gasUsed)

    if [[ "$status" = "0x1" ]]; then
        echo "  status: SUCCESS"
    else
        echo "  status: REVERTED"
    fi
    echo "  gasUsed: $gas_used"

    # 전송 후 잔액
    local bal_after bal_to
    bal_after=$(cast balance "$sender" --rpc-url "$RPC_URL" | xargs)
    bal_to=$(cast balance "$to" --rpc-url "$RPC_URL" | xargs)
    echo ""
    echo "  sender balance (after):  ${bal_after} wei"
    echo "  to balance:              ${bal_to} wei"
}

# =============================================================================
# ERC20 Token 전송
# =============================================================================

cmd_token_transfer() {
    local token="${1:?Usage: $0 token-transfer <token_address> <to_address> <amount_raw>}"
    local to="${2:?Usage: $0 token-transfer <token_address> <to_address> <amount_raw>}"
    local amount="${3:?Usage: $0 token-transfer <token_address> <to_address> <amount_raw>}"

    local sender
    sender=$(cast wallet address "$PRIVATE_KEY")

    local symbol
    symbol=$(cast call "$token" 'symbol()(string)' --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    symbol="${symbol:-TOKEN}"

    echo "=== ERC20 Transfer ==="
    echo "  token:  ${token} (${symbol})"
    echo "  from:   $sender"
    echo "  to:     $to"
    echo "  amount: $amount (raw)"
    echo "  gas:    feeCap=${GAS_PRICE} tip=${GAS_TIP}"
    echo ""

    # 전송 전 잔액
    local bal_from_before bal_to_before
    bal_from_before=$(cast call "$token" 'balanceOf(address)(uint256)' "$sender" --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    bal_to_before=$(cast call "$token" 'balanceOf(address)(uint256)' "$to" --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    echo "  from balance (before): ${bal_from_before:-0}"
    echo "  to   balance (before): ${bal_to_before:-0}"

    local result
    result=$(cast send "$token" \
        "transfer(address,uint256)(bool)" \
        "$to" "$amount" \
        --gas-price "$GAS_PRICE" \
        --priority-gas-price "$GAS_TIP" \
        --private-key "$PRIVATE_KEY" \
        --rpc-url "$RPC_URL" \
        --json 2>&1)

    local tx_hash
    tx_hash=$(echo "$result" | jq -r .transactionHash 2>/dev/null)

    if [[ -z "$tx_hash" || "$tx_hash" = "null" ]]; then
        echo "  ERROR: 전송 실패"
        echo "  $result"
        return 1
    fi

    echo "  tx:     $tx_hash"

    local status
    status=$(cast receipt "$tx_hash" --rpc-url "$RPC_URL" --json 2>/dev/null | jq -r .status)
    if [[ "$status" = "0x1" ]]; then
        echo "  status: SUCCESS"
    else
        echo "  status: REVERTED"
    fi

    # 전송 후 잔액
    local bal_from_after bal_to_after
    bal_from_after=$(cast call "$token" 'balanceOf(address)(uint256)' "$sender" --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    bal_to_after=$(cast call "$token" 'balanceOf(address)(uint256)' "$to" --rpc-url "$RPC_URL" 2>/dev/null | xargs)
    echo ""
    echo "  from balance (after):  ${bal_from_after:-0}"
    echo "  to   balance (after):  ${bal_to_after:-0}"
}

# =============================================================================
# Usage
# =============================================================================
usage() {
    cat <<USAGE
Usage: $0 <command> [args...]

Native Coin 조회:
  native <address>                              단일 주소 native balance
  native-multi <addr1> [addr2] ...              여러 주소 native balance
  validators                                    validator 7개 전체 balance

Native Coin 전송:
  send <to_address> <amount> [unit]             native coin 전송
      unit: ether(기본), gwei, wei              (PRIVATE_KEY 필요)

ERC20 Token 조회:
  token-info <token>                            토큰 정보 (name, symbol, decimals, supply)
  token-balance <token> <address>               토큰 잔액 조회
  token-multi <token> <addr1> [addr2]...        여러 주소 토큰 잔액

ERC20 Token 전송:
  token-transfer <token> <to_address> <amount>  ERC20 토큰 전송 (raw amount)
                                                (PRIVATE_KEY 필요)

NativeCoinAdapter (0x1000):
  nca-info                                      NCA 토큰 정보
  nca-balance <address>                         NCA 잔액 조회

비교:
  compare <address>                             native vs NCA 잔액 비교

Environment:
  RPC_URL      RPC endpoint (default: http://172.21.132.1:8601)
  PRIVATE_KEY  send/transfer 시 필요

Examples:
  # 조회
  $0 native 0x518b3Efa7dB538F29615Cb9d76f4ac234EBE5893
  $0 validators
  $0 token-info 0x0000000000000000000000000000000000001000
  $0 token-balance 0x0000000000000000000000000000000000001000 0x518b3...
  $0 compare 0x518b3Efa7dB538F29615Cb9d76f4ac234EBE5893

  # 전송
  export PRIVATE_KEY="0x3c1cc949..."
  $0 send 0xD76975b29BDE03F4C644851393D988CcEa9d1471 1 ether
  $0 send 0xD76975b29BDE03F4C644851393D988CcEa9d1471 500 gwei
  $0 send 0xD76975b29BDE03F4C644851393D988CcEa9d1471 1000000000000000000 wei
  $0 token-transfer 0x<token> 0xD7697... 1000000000000000000
USAGE
}

# =============================================================================
# Main
# =============================================================================
main() {
    local cmd="${1:-}"
    shift || true

    case "$cmd" in
        # Native 조회
        native)          check_env; cmd_native "$@" ;;
        native-multi)    check_env; cmd_native_multi "$@" ;;
        validators)      check_env; cmd_validators ;;
        # Native 전송
        send)            check_env_with_key; resolve_gas_params; cmd_send "$@" ;;
        # ERC20 조회
        token-info)      check_env; cmd_token_info "$@" ;;
        token-balance)   check_env; cmd_token_balance "$@" ;;
        token-multi)     check_env; cmd_token_multi "$@" ;;
        # ERC20 전송
        token-transfer)  check_env_with_key; resolve_gas_params; cmd_token_transfer "$@" ;;
        # NCA
        nca-info)        check_env; cmd_nca_info ;;
        nca-balance)     check_env; cmd_nca_balance "$@" ;;
        # Compare
        compare)         check_env; cmd_compare "$@" ;;
        # Help
        -h|--help|"")    usage ;;
        *)               echo "Unknown command: $cmd" >&2; usage; exit 1 ;;
    esac
}

main "$@"
