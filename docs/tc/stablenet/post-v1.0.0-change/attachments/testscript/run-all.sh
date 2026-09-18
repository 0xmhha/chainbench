#!/bin/bash
# run-all.sh - 하드포크 테스트 전체/선택 실행
#
# 사용법:
#   ./run-all.sh                    # 전체 실행
#   ./run-all.sh section1           # Section 1만 실행
#   ./run-all.sh tc-1-1-01          # 단일 TC 실행
#   ./run-all.sh --list             # TC 목록 출력
#   ./run-all.sh --results          # 결과 요약 출력

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RESULT_DIR="${SCRIPT_DIR}/results"
mkdir -p "$RESULT_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ── TC 정의 ──
# Section 1-1: GovMinter 업그레이드
SECTION_1_1=(
  "tc-1-1-01.sh"
  "tc-1-1-02.sh"
  "tc-1-1-03.sh"
  "tc-1-1-04.sh"
  "tc-1-1-05.sh"
  "tc-1-1-06.sh"
  "tc-1-1-07.sh"
  "tc-1-1-08.sh"
  "tc-1-1-09.sh"
  "tc-1-1-10.sh"
  "tc-1-1-11.sh"
  "tc-1-1-12.sh"
)

# Section 1-2: secp256r1
SECTION_1_2=(
  "tc-1-2-01.sh"
  "tc-1-2-02.sh"
  "tc-1-2-03.sh"
  "tc-1-2-04.sh"
  "tc-1-2-05.sh"
  "tc-1-2-06.sh"
)

# Section 1-3: 가스비
SECTION_1_3=(
  "tc-1-3-gas.sh"
)

# Section 2: 취약점 패치
SECTION_2=(
  "tc-section2-vuln.sh"
)

# Section 3: 성능
SECTION_3=(
  "tc-3-1-bench.sh"
  "tc-3-1-04.sh"
)

# Section 4: 버그 수정
SECTION_4=(
  "tc-4-1-01.sh"
  "tc-4-1-02.sh"
  "tc-4-1-03.sh"
  "tc-4-2-gas-estimate.sh"
  "tc-4-3-string.sh"
  "tc-4-4-01.sh"
  "tc-4-4-02.sh"
  "tc-4-5-01.sh"
  "tc-4-5-09.sh"
  "tc-4-6-01.sh"
)

# Section 5: 내부 개선
SECTION_5=(
  "tc-5-1-build.sh"
  "tc-5-2-06.sh"
  "tc-5-3-01.sh"
)

ALL_TCS=("${SECTION_1_1[@]}" "${SECTION_1_2[@]}" "${SECTION_1_3[@]}" "${SECTION_2[@]}" "${SECTION_3[@]}" "${SECTION_4[@]}" "${SECTION_5[@]}")

# ── 함수 ──

print_list() {
  echo -e "${BOLD}하드포크 테스트 케이스 목록${NC}"
  echo ""
  echo -e "  ${CYAN}Section 1-1: GovMinter 업그레이드 (Privatenet)${NC}"
  for tc in "${SECTION_1_1[@]}"; do echo "    $tc"; done
  echo -e "  ${CYAN}Section 1-2: secp256r1 서명 검증${NC}"
  for tc in "${SECTION_1_2[@]}"; do echo "    $tc"; done
  echo -e "  ${CYAN}Section 1-3: 최소 가스비${NC}"
  for tc in "${SECTION_1_3[@]}"; do echo "    $tc"; done
  echo -e "  ${CYAN}Section 2: 취약점 패치 (단위 테스트)${NC}"
  for tc in "${SECTION_2[@]}"; do echo "    $tc"; done
  echo -e "  ${CYAN}Section 3: 성능 벤치마크${NC}"
  for tc in "${SECTION_3[@]}"; do echo "    $tc"; done
  echo -e "  ${CYAN}Section 4: 버그 수정${NC}"
  for tc in "${SECTION_4[@]}"; do echo "    $tc"; done
  echo -e "  ${CYAN}Section 5: 내부 개선${NC}"
  for tc in "${SECTION_5[@]}"; do echo "    $tc"; done
  echo ""
  echo "  총 ${#ALL_TCS[@]}개 스크립트"
}

print_results() {
  echo -e "${BOLD}하드포크 테스트 결과 요약${NC}"
  echo -e "  결과 디렉토리: ${RESULT_DIR}"
  echo ""

  local total=0 pass=0 fail=0 blocked=0

  for result_file in "${RESULT_DIR}"/*.result; do
    [ -f "$result_file" ] || continue
    total=$((total + 1))

    local tc status
    tc=$(grep "^tc=" "$result_file" | cut -d= -f2)
    status=$(grep "^status=" "$result_file" | cut -d= -f2)

    case "$status" in
      PASS)    echo -e "  ${GREEN}PASS${NC}    $tc"; pass=$((pass + 1)) ;;
      FAIL)    echo -e "  ${RED}FAIL${NC}    $tc"; fail=$((fail + 1)) ;;
      BLOCKED) echo -e "  ${YELLOW}BLOCKED${NC} $tc"; blocked=$((blocked + 1)) ;;
      *)       echo -e "  ???     $tc ($status)" ;;
    esac
  done

  echo ""
  echo -e "  ${BOLD}총계: ${total}  |  ${GREEN}Pass: ${pass}${NC}  |  ${RED}Fail: ${fail}${NC}  |  ${YELLOW}Blocked: ${blocked}${NC}"
}

run_tc() {
  local script="$1"
  local script_path="${SCRIPT_DIR}/${script}"

  if [ ! -f "$script_path" ]; then
    echo -e "${RED}스크립트 없음: ${script}${NC}"
    return 1
  fi

  echo -e "\n${BOLD}${YELLOW}>>> ${script} 실행 <<<${NC}"
  bash "$script_path"
  local exit_code=$?

  if [ $exit_code -eq 2 ]; then
    echo -e "${YELLOW}  BLOCKED${NC}"
  elif [ $exit_code -ne 0 ]; then
    echo -e "${RED}  FAIL (exit: ${exit_code})${NC}"
  fi

  return $exit_code
}

run_section() {
  local section_name="$1"
  shift
  local scripts=("$@")

  echo -e "\n${BOLD}${CYAN}============================================${NC}"
  echo -e "${BOLD}${CYAN}  ${section_name}${NC}"
  echo -e "${BOLD}${CYAN}============================================${NC}"

  local total=${#scripts[@]} passed=0 failed=0 blocked=0

  for script in "${scripts[@]}"; do
    run_tc "$script"
    local rc=$?
    case $rc in
      0) passed=$((passed + 1)) ;;
      2) blocked=$((blocked + 1)) ;;
      *) failed=$((failed + 1)) ;;
    esac
  done

  echo -e "\n  ${BOLD}${section_name} 요약: ${total}개 중 Pass=${passed}, Fail=${failed}, Blocked=${blocked}${NC}"
}

# ── 메인 ──

case "${1:-all}" in
  --list)
    print_list
    ;;
  --results)
    print_results
    ;;
  section1|s1)
    run_section "Section 1-1: GovMinter 업그레이드" "${SECTION_1_1[@]}"
    run_section "Section 1-2: secp256r1 서명 검증" "${SECTION_1_2[@]}"
    run_section "Section 1-3: 최소 가스비" "${SECTION_1_3[@]}"
    ;;
  section1-1|s1-1)
    run_section "Section 1-1: GovMinter 업그레이드" "${SECTION_1_1[@]}"
    ;;
  section1-2|s1-2)
    run_section "Section 1-2: secp256r1 서명 검증" "${SECTION_1_2[@]}"
    ;;
  section1-3|s1-3)
    run_section "Section 1-3: 최소 가스비" "${SECTION_1_3[@]}"
    ;;
  section2|s2)
    run_section "Section 2: 취약점 패치" "${SECTION_2[@]}"
    ;;
  section3|s3)
    run_section "Section 3: 성능 벤치마크" "${SECTION_3[@]}"
    ;;
  section4|s4)
    run_section "Section 4: 버그 수정" "${SECTION_4[@]}"
    ;;
  section5|s5)
    run_section "Section 5: 내부 개선" "${SECTION_5[@]}"
    ;;
  all)
    echo -e "${BOLD}${CYAN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${CYAN}║  하드포크 테스트 전체 실행               ║${NC}"
    echo -e "${BOLD}${CYAN}║  시작: $(date '+%Y-%m-%d %H:%M:%S')              ║${NC}"
    echo -e "${BOLD}${CYAN}╚══════════════════════════════════════════╝${NC}"

    run_section "Section 1-1: GovMinter 업그레이드" "${SECTION_1_1[@]}"
    run_section "Section 1-2: secp256r1 서명 검증" "${SECTION_1_2[@]}"
    run_section "Section 1-3: 최소 가스비" "${SECTION_1_3[@]}"
    run_section "Section 2: 취약점 패치" "${SECTION_2[@]}"
    run_section "Section 3: 성능 벤치마크" "${SECTION_3[@]}"
    run_section "Section 4: 버그 수정" "${SECTION_4[@]}"
    run_section "Section 5: 내부 개선" "${SECTION_5[@]}"

    echo ""
    print_results
    ;;
  *)
    echo "사용법:"
    echo "  $0                    # 전체 실행"
    echo "  $0 section1           # Section 1 실행"
    echo "  $0 section1-1         # Section 1-1만 실행"
    echo "  $0 --list             # TC 목록"
    echo "  $0 --results          # 결과 요약"
    echo ""
    echo "  단일 TC 실행: ./tc-1-1-01.sh (각 스크립트 직접 실행)"
    echo "  Sections: section1(-1|-2|-3), section2, section3, section4, section5, all"
    ;;
esac
