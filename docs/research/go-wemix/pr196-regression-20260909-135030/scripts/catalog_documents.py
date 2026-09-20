"""Extract every scenario row and attach reviewed go-wemix applicability."""

from __future__ import annotations

import json
import re
from collections import Counter
from pathlib import Path


RUN = Path(__file__).resolve().parent.parent
API = "internal/ethapi/api.go"
COMMON_REFS = {
    1: ["core/types/transaction.go:45", "miner/worker.go:935"],
    2: ["core/types/transaction.go:47", "core/tx_pool.go:655"],
    3: ["core/types/transaction.go:46", "core/tx_pool.go:651"],
    4: ["core/types/transaction.go:48", "core/tx_pool.go:709", "core/state_transition.go:198"],
    5: ["core/tx_pool.go:688"], 6: ["core/tx_pool.go:709"],
    7: ["core/tx_pool.go:713"],
    8: ["core/tx_pool.go:701", "miner/worker.go:935", "miner/worker.go:1065"],
    9: ["core/tx_pool.go:693", "core/tx_pool.go:697"],
    10: ["core/tx_pool.go:720"], 11: ["core/tx_pool.go:673"],
    12: [f"{API}:1871"], 13: ["core/tx_pool.go:801", "core/tx_pool.go:851"],
    14: ["core/vm/evm.go:499"], 15: ["core/vm/evm.go:168"],
    16: [f"{API}:1206"], 17: [f"{API}:1340"], 18: [f"{API}:1206"],
    19: ["core/vm/evm.go:168", "core/state_transition.go:402"],
    20: ["core/vm/evm.go:168", "core/state_transition.go:314"],
    21: [f"{API}:738"], 22: [f"{API}:843"], 23: [f"{API}:1992"],
    24: ["eth/filters/api.go:327"], 25: [f"{API}:729"],
    26: ["eth/filters/api.go:208"], 27: ["eth/filters/api.go:238"],
    28: [f"{API}:950"], 29: [f"{API}:970"], 30: [f"{API}:1798"],
    31: [f"{API}:1838"], 32: [f"{API}:1779"], 33: [f"{API}:92"],
    34: [f"{API}:104"], 35: [f"{API}:119"], 36: [f"{API}:241"],
    37: [f"{API}:191"], 38: [f"{API}:2404"], 39: ["core/genesis.go:239"],
    40: ["eth/downloader/downloader.go:1475", "core/block_validator.go:55"],
    41: ["eth/downloader/downloader.go:1565", "eth/downloader/downloader.go:1710"],
    42: ["core/genesis.go:239", "miner/worker.go:1627"],
    43: [f"{API}:2304", "node/api.go:308"],
    44: ["eth/downloader/downloader.go:1475", "eth/protocols/eth/handler.go:254"],
    45: ["eth/fetcher/block_fetcher.go:231", "eth/protocols/eth/handler.go:254"],
}


def common_review(index: int) -> tuple[str, str, str, str]:
    """Return applicability, priority, reason and target-specific prerequisites."""
    priority = "P1"
    status = "applicable"
    reason = "지원 기능의 정상·실패 결과를 대조하여 패치 후 트랜잭션 실행 및 관측 회귀를 검증한다."
    prerequisites = "go-wemix 격리 테스트 체인과 계정·RPC 설정으로 스크립트를 이식하고 기대값을 검증해야 한다."
    if index in {8, 40, 44, 45}:
        priority = "P0"
        reason = "PR #196의 채굴 거래 선택 또는 블록 수신·검증 경로와 직접 접한다. 작은 블록 통과만으로 대용량 경계 회귀를 검증할 수 없으므로 추가 경계 테스트와 함께 우선 수행한다."
    if index in {16, 17, 18, 22, 24, 26, 27, 30, 33, 34, 35, 36, 37, 39, 43}:
        priority = "P2"
        reason = "지원 RPC·운영 기능의 간접 회귀 점검이다. 블록 생성·전파·실행 검증 이후 수행한다."
    if index in {25, 38}:
        priority = "P3"
        reason = "RPC 존재·고정 설정을 확인하는 smoke 성격으로 패치의 블록 크기·채굴 회귀 탐지력은 낮다."
    if index in {2, 3, 4, 5, 6, 7}:
        prerequisites += " Berlin/London 및 Applepie(수수료 대납) 활성 높이를 유형별로 확인한다(core/tx_pool.go:651,655,659,1398)."
    if index == 1:
        prerequisites += " StableNet의 헤더 고정 tip 기대값을 제거하고 go-wemix receipt 계산식을 적용한다."
    if index == 9:
        status = "conditional"
        prerequisites += " 로컬·원격 제출을 분리한다. 로컬은 DropUnderPriced 및 EffectiveGasTip, 원격은 GasTipCap 비교이므로 문서의 pool.gasPrice 설명만으로 판정하지 않는다."
    if index in {10, 5, 6, 7}:
        prerequisites += " 원문의 에러 문자열을 그대로 강제하지 말고 실제 오류 종류·해당 계정 잔액 검사를 대조한다."
    if index == 12:
        prerequisites += " BP/EN 역할을 생산 노드·동기화 노드로 대응하고 동일 블록 해시에서 receipt를 대조한다. London 전후 기대값을 구분한다."
    if index == 13:
        prerequisites += " txpool.pricebump 실제 값을 읽고 maxFeePerGas/maxPriorityFeePerGas 양쪽을 인상한다. gap 해소 후 교체 tx만 포함되는지 검증한다."
    if index in {14, 15, 19, 20, 24, 27}:
        prerequisites += " 이 클라이언트 EVM이 지원하는 opcode로 컴파일한 fixture가 필요하다. OOG는 intrinsic gas 이상이고 실행 소요 gas 미만으로 설정한다."
    if index == 21:
        prerequisites += " reorg 없는 제어된 체인에서 관찰 구간 내 실제 증가를 확인한다. 단순 비감소만으로 체인 정지를 통과시키면 안 된다."
    if index == 23:
        prerequisites += " 즉시 채굴될 수 있으므로 txpool 조회 성공을 필수로 하려면 채굴을 제어한다."
    if index in {26, 27}:
        prerequisites += " WebSocket 및 eth 구독 API가 활성화된 노드가 필요하다."
    if index in {33, 34}:
        prerequisites += " WBFTExtra.GasTip 기대값은 제외하고 go-wemix API·oracle 결과를 사용한다."
    if index in {36, 37}:
        prerequisites += " txpool API 노출 및 채굴 제어가 필요하다. JSON 수량은 hex quantity로 디코딩한다."
    if index == 38:
        prerequisites += " eth API 노출을 확인한다. method-not-found가 아님은 기능적 서명 성공의 증거가 아니다."
    if index in {40, 41, 42, 43, 44, 45}:
        prerequisites += " 실제 go-wemix 네트워크·genesis·peer 설정이 필요하며 다중 노드에서 동일 높이 block hash/stateRoot를 대조한다."
    if index == 41:
        status = "conditional"
        prerequisites += " snap 지원 피어·충분한 체인 길이·snapshot 상태가 필요하다. full sync로 fallback한 성공은 snap 경로 검증으로 세지 않는다."
    if index in {44, 45}:
        prerequisites += " 높이 gap만으로 경로를 단정하지 않고 downloader/fetcher 진입을 관측한다."
    return status, priority, reason, prerequisites


def dual_review(index: int) -> tuple[str, str, str, str, list[str]]:
    """Classify dual-chain scenarios against the captured PR head."""
    if index == 1:
        return ("conditional", "P2", "Brioche 보상 조회는 구현되어 있으나 패치 변경부의 간접 회귀이다.",
                "wemixapi.Info와 Brioche 설정이 유효한 체인에서 실제 fork·halving 경계를 지정한다. WEMIX4 스크립트 이식 필요.",
                ["eth/api.go:748", "eth/api.go:764", "params/config.go:443"])
    if index == 26:
        return ("conditional", "P2", "AuthorizationList가 없는 일반 estimateGas baseline은 지원된다. EIP-7702 지원을 검증한 것으로 기록하면 안 된다.",
                "7702 필드·fixture를 제거하고 일반 전송 baseline으로 이식한다. DOC-C-017과 관련 있지만 원문 행은 유지한다.",
                [f"{API}:1340", "internal/ethapi/transaction_args.go:52", "core/types/transaction.go:189"])
    if index in {24, 25}:
        return ("excluded", "P3", "typed transaction 디코더는 0x1·0x2·0x16만 수용하며 EIP-7702 0x4/AuthorizationList 실행은 지원하지 않는다.",
                "go-wemix #196 회귀 실행 대상에서 제외. 미지원 타입 거부 검증은 별도의 다른 테스트이다.",
                ["core/types/transaction.go:189", "internal/ethapi/transaction_args.go:52"])
    if index >= 27:
        return ("excluded", "P3", "등록된 EVM 프리컴파일 집합에 RIP-7212의 0x100 P256VERIFY가 없다. 빈 반환만으로 무효 서명 검증이 성공했다고 판단할 수 없다.",
                "go-wemix #196 회귀 실행 대상에서 제외한다.", ["core/vm/contracts.go:48", "core/vm/contracts.go:84"])
    return ("excluded", "P3", "이 원문은 WBFT seal·round·epoch·쿼럼 또는 istanbul RPC 의미를 요구한다. go-wemix의 engine 구성 및 mining token 경로와 다르므로 그대로 수행할 수 없다.",
            "WBFT 1초·1/3·quorum·round robin 기대값을 SPoA에 복사하지 않는다. 장애복구 목적은 go-wemix 전용 추가 테스트에서 새로 정의한다.",
            ["eth/ethconfig/config.go:217", "miner/worker.go:1627", "miner/worker.go:1812"])


def extract_document(filename: str, prefix: str) -> list[dict[str, object]]:
    """Preserve each source row without merging reused test IDs or scripts."""
    records: list[dict[str, object]] = []
    section = ""
    lines = (RUN / "sources" / filename).read_text().splitlines()
    for line_number, line in enumerate(lines, 1):
        if line.startswith("##"):
            section = line.lstrip("# ")
        if not line.startswith("| **"):
            continue
        cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
        if len(cells) != 6:
            raise ValueError(f"Unexpected scenario columns: {filename}:{line_number}")
        index = len(records) + 1
        if prefix == "C":
            status, priority, reason, prerequisites = common_review(index)
            references = COMMON_REFS[index]
        else:
            status, priority, reason, prerequisites, references = dual_review(index)
        references = [f"sources/pr-head/{reference}" for reference in references]
        records.append({
            "catalog_id": f"DOC-{prefix}-{index:03d}", "source_kind": "document",
            "source_file": f"sources/{filename}", "source_line": line_number,
            "section": section, "category": cells[0], "references_original": cells[1],
            "original_test_ids": re.findall(r"`((?:RT|TX|RPC|NODE|WBFT|TC)-[^`]+)`", cells[1]),
            "script_references": re.findall(r"`([^`]+\.sh)`", cells[1]),
            "title": cells[2], "purpose": cells[3], "steps": cells[4], "expected": cells[5],
            "raw_row": line, "applicability": status, "priority": priority,
            "priority_scope": "PR196 regression; excluded P3 is inventory placement, not execution recommendation",
            "rationale": reason, "preconditions": prerequisites, "code_refs": references,
            "execution_readiness": "spec_only_or_port_needed" if status != "excluded" else "not_target",
            "script_presence_verified": False,
            "evidence": "Captured PR-head source inspection; no runtime tests or script execution. Original shell references are not proof of available executable files.",
        })
    expected_count = 45 if prefix == "C" else 29
    if len(records) != expected_count:
        raise ValueError(f"Expected {expected_count} rows; got {len(records)}")
    return records


def write_markdown(records: list[dict[str, object]]) -> None:
    """Write an ordered overview and every original scenario field."""
    content = ["# 문서 테스트 전체 목록", "", "common 45행 + dual 29행 = 74행. 원본 중복 ID·스크립트도 합치지 않았다. 코드 근거는 모두 캡처한 PR head 기준이며 로컬 HEAD와 구분한다.", "", "적용 가능은 기능 지원 판정이며 즉시 실행 가능을 뜻하지 않는다. 원문 .sh 존재·정상 실행은 확인하지 않았다. P0→P3 순서이며 제외 행의 P3는 목록 배치용이다.", "", "| 목록 ID | 우선순위 | 적용 판정 | 테스트 | 원문 위치 |", "|---|---|---|---|---|"]
    ordered = sorted(records, key=lambda record: (str(record["priority"]), str(record["catalog_id"])))
    for record in ordered:
        content.append(f"| {record['catalog_id']} | {record['priority']} | {record['applicability']} | {record['title']} | {record['source_file']}:{record['source_line']} |")
    for record in ordered:
        content.extend(["", f"## {record['catalog_id']} · {record['title']}", "",
                        f"- 우선순위: {record['priority']} / 적용: {record['applicability']} / 실행 준비: {record['execution_readiness']}",
                        f"- 원본: {record['source_file']}:{record['source_line']} / {record['section']}",
                        f"- 카테고리: {record['category']}", f"- 원래 ID·스크립트: {record['references_original']}",
                        f"- 목적(원문): {record['purpose']}", f"- 수행흐름(원문): {record['steps']}",
                        f"- 기대결과(원문): {record['expected']}", f"- 판정·우선순위 근거: {record['rationale']}",
                        f"- go-wemix 적용 조건: {record['preconditions']}",
                        f"- 코드: {', '.join(str(ref) for ref in record['code_refs'])}"])
    (RUN / "analyses/document-catalog.md").write_text("\n".join(content) + "\n")


def main() -> None:
    """Generate and verify the document catalog artifacts."""
    records = extract_document("common_test_scenarios.md", "C") + extract_document("dual_chain_test_scenarios.md", "D")
    for record in records:
        for reference in record["code_refs"]:
            path, number = str(reference).rsplit(":", 1)
            if not 1 <= int(number) <= len((RUN / path).read_text().splitlines()):
                raise ValueError(f"Invalid code reference: {reference}")
    (RUN / "analyses/document-catalog.json").write_text(json.dumps(records, ensure_ascii=False, indent=2) + "\n")
    write_markdown(records)
    report = {"total": len(records), "per_document": dict(Counter(str(record["source_file"]) for record in records)),
              "applicability": dict(Counter(str(record["applicability"]) for record in records)),
              "priority": dict(Counter(str(record["priority"]) for record in records)),
              "missing_rows": 0, "code_references_exist_and_in_range": True}
    (RUN / "analyses/document-catalog-validation.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")


if __name__ == "__main__":
    main()
