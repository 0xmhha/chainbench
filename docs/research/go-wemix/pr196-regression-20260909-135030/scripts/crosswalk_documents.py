"""Materialize reviewed scenario-to-JSON coverage without treating names as proof."""
from __future__ import annotations

import json
from collections import Counter
from pathlib import Path

RUN = Path(__file__).resolve().parent.parent
TC = RUN / "sources/chainbench/tests/tc"

# Each suffix is resolved against all 189 JSON files; ambiguity fails closed.
COMMON = {
 1: ("chain_specific", ["08-legacy-transfer"], "타입0·상태1·수신액은 검사하지만 gasPrice 준비에 istanbul_getWbftExtraInfo를 호출하고 effectiveGasPrice==gasPrice는 검사하지 않는다."),
 2: ("chain_specific", ["09-dynamic-fee-tx"], "타입2·성공만 검사한다. WBFT tip으로 fee를 만들며 문서의 effectiveGasPrice 산식은 검사하지 않는다."),
 3: ("partial", ["10-access-list-tx"], "createAccessList 후 타입1·성공 검사. 일반 수신자라 accessList가 비어도 통과하며 주소·storage key 포함/할인 검증은 없다."),
 4: ("partial", ["01-fee-delegated-transfer"], "sender/feePayer 분리 전송과 수신액을 확인하지만 두 지불 계정의 value·gas 차감 내역은 비교하지 않는다."),
 5: ("partial", ["02-fd-sender-sig-invalid-rejected", "05-fee-delegated-sender-sig-invalid-rejected"], "sendRawTampered(which=sender) 액션을 호출한다. JSON에 invalid sender/signature 오류 종류 단언이 없다."),
 6: ("partial", ["03-fd-feepayer-sig-invalid-rejected", "06-fee-delegated-feepayer-sig-invalid-rejected"], "sendRawTampered(which=feepayer) 액션을 호출한다. JSON에 feePayer 서명 오류 종류 단언이 없다."),
 7: ("partial", ["04-feepayer-insufficient-rejected", "07-fee-delegated-unfunded-feepayer-rejected"], "새 feePayer를 미충전한 채 reject를 요구한다. 부족 잔액 오류 종류를 특정하지 않아 다른 거부 원인도 분리해야 한다."),
 8: ("partial", ["11-nonce-ordering", "11b-out-of-order-nonces-mine"], "2,1,0 제출 후 최종 nonce==3을 검사한다. 각 tx의 receipt·blockNumber·transactionIndex로 실제 포함 순서를 직접 대조하지 않는다."),
 9: ("partial", ["12-dynamic-fee-below-basefee-rejected", "14-feecap-below-min-rejected"], "feeCap/tipCap 낮은 값으로 reject를 검사한다. baseFee 미달과 최소 tip 미달을 분리하지 않으며 로컬·원격 정책별 오류 단언이 없다."),
 10: ("partial", ["14-insufficient-funds-rejected"], "잔액+1 ETH 값의 전송 거부는 검사하나 insufficient funds 오류 종류를 특정하지 않는다."),
 11: ("partial", ["15-gas-limit-exceeds-block-rejected", "11-gaslimit-exceeded-rejected"], "헤더 gasLimit+1 제출 거부 패턴이지만 오류 원인을 특정하지 않는다. anzeon/11은 WBFTExtra tip 의존을 제거해야 한다."),
 12: ("partial", ["16-effective-gas-price", "02b-effective-gas-price-regular-bp-en", "02-effective-gas-price-regular"], "16은 단일 노드 EGP>0이고 02b는 BP/EN EGP equality이나 baseFee 하한을 검사하지 않는다. 02는 WBFT GasTip 산식 의존으로 go-wemix 적용 불가."),
 13: ("partial", ["17-replacement-tx", "17b-same-nonce-replacement"], "nonce gap·20% fee 인상·tx2 mined 및 tx1 unmined 검증. 교체 tx2의 status==1은 JSON에서 별도 확인하지 않는다."),
 14: ("exact", ["19-contract-roundtrip", "go-wemix/tx/01-wemix-tx-and-contract"], "배포 후 얻은 주소의 codeAt!=0x를 직접 검사해 문서 핵심 기대값을 충족한다. 체인 환경 이식·실행 확인은 별개다."),
 15: ("no_match", [], "범용 setter 호출 후 getter 상태 변경을 확인하는 동일 fixture는 찾지 못했다. 시스템 컨트랙트 approve/거버넌스 사례는 체인 전용이므로 대체하지 않는다."),
 16: ("exact", ["19-contract-roundtrip", "go-wemix/tx/01-wemix-tx-and-contract"], "배포한 returner에 읽기 call을 수행해 정확한 32-byte 값 42를 확인한다. 상태 변경 트랜잭션 없이 조회하는 핵심 기대를 충족한다."),
 17: ("partial", ["22-estimate-gas"], "주소0x1에 빈 data estimateGas>=21000만 요구한다. 컨트랙트 실제 실행 gasUsed와 추정치를 비교하지 않는다."),
 18: ("partial", ["23-eth-call-revert-returns-error"], "callError만 단언하여 Revert Reason 내용을 디코딩·대조하지 않는다. fixture의 PUSH0 지원도 go-wemix에서 확인해야 한다."),
 19: ("partial", ["24-revert-tx-status-zero", "01-negative-tx-revert"], "실패 status0/revert는 확인하지만 상태 rollback과 미사용 gas 환불·잔액 차감 계산은 검사하지 않는다."),
 20: ("partial", ["25-out-of-gas-consumes-all"], "expect=revert와 gasUsed=50000을 검사한다. OOG를 다른 EVM 실패와 구분하는 오류 원인·잔액 환불 오라클은 보강 필요."),
 21: ("partial", ["basic/01-basic-consensus", "basic/03-basic-rpc-health", "remote/01-remote-rpc-health"], "basic consensus는 blockAdvance를 검사하나 연속 RPC 비감소 관측과 동일하지 않다. 나머지는 단일 높이 범위만 검사하여 정지를 탐지하지 못한다."),
 22: ("partial", ["27-genesis-balance", "remote/03-remote-balance-check", "27b-value-transfer"], "잔액>0 또는 문자열 비어있지 않음/전송 증분은 확인한다. genesis의 정확한 알려진 잔액과 비교하지 않는다."),
 23: ("partial", ["27b-value-transfer", "basic/05-basic-tx-send", "go-wemix/tx/01-wemix-tx-and-contract"], "sendTx 후 성공·잔액은 검사하지만 원시 RLP 입력과 채굴 전 txpool 관측을 독립 검증하지 않는다."),
 24: ("exact", ["29-logs-query-well-formed", "36-contract-event-emitted", "23-token-approve-sets-allowance"], "36은 이벤트 발생 후 address/topics/fromBlock/toBlock으로 조회하고 topic0를 대조하여 문서의 양성 조건을 충족한다. 29는 count>=0뿐이며 23은 StableNet 시스템 토큰 의존이다. 각 match의 개별 coverage를 구분했다."),
 25: ("partial", ["30-chain-id", "remote/02-remote-chain-info", "go-wemix/chain-up/01-wemix-chain-up"], "chainId>0 또는 hex 형식만 검사한다. genesis 설정값과 정확히 같음을 단언하지 않는다."),
 26: ("partial", ["31-ws-subscribe-new-heads"], "30초 내 newHeads 1건을 요구한다. 매 블록 이벤트의 누락·순서·헤더 일관성은 검사하지 않는다."),
 27: ("partial", ["32-ws-subscribe-logs"], "주소 필터 구독 후 이벤트 tx를 보내 1건 수신한다. topic 필터 및 이벤트 payload 전체 대조는 없다."),
 28: ("partial", ["01-block-transactions-field"], "number/hash 형식과 transactions nonnil만 검사한다. fullTx=true와 parentHash 및 tx 상세는 검사하지 않는다."),
 29: ("partial", ["02-block-by-hash-consistency"], "hash·number만 비교하고 Block 객체 전체 equality는 없다. latest를 두 번 읽어 경합할 수 있다."),
 30: ("partial", ["03-transaction-by-hash-fields"], "from/to 및 blockNumber/value 존재를 확인한다. nonce/input과 모든 상세값은 검증하지 않는다."),
 31: ("partial", ["04-transaction-receipt-fields", "16-effective-gas-price"], "status·EGP·logs 또는 gasUsed 일부만 검사한다. 원문의 모든 receipt 필드를 한 번에 정합성 대조하지 않는다."),
 32: ("partial", ["05-transaction-count-increments"], "전송 전후 nonce +1과 성공을 비교한다. 기존 총 트랜잭션 수 재계산은 하지 않는다."),
 33: ("partial", ["07-gas-price-positive", "07b-gas-price-equals-basefee-plus-tip"], "07은 gasPrice>0만, 07b는 WBFTExtra tip 산식이다. go-wemix oracle 결과 검증을 새로 연결해야 한다."),
 34: ("chain_specific", ["08-max-priority-fee-equals-gastip"], "WBFTExtra.GasTip과 완전 일치를 요구하여 go-wemix oracle 검증으로 대체할 수 없다."),
 35: ("partial", ["09-fee-history-well-formed"], "oldestBlock/baseFeePerGas만 검사하고 percentile=[]로 호출한다. gasUsedRatio와 percentile tip 검증이 빠졌다."),
 36: ("partial", ["18-txpool-status"], "pending/queued hex 형식만 검사하며 두 종류의 트랜잭션을 주입하지 않는다."),
 37: ("partial", ["19-txpool-content-well-formed"], "pending/queued nonnil만 검사하며 계정별 주입 tx와 실제 내용을 대조하지 않는다."),
 38: ("exact", ["21-fee-delegate-sign-rpc-present"], "잘못된 인자를 methodPresent로 검사하는 동일 smoke 목적이다. 서명 기능 성공을 의미하지 않는다."),
 39: ("partial", ["30-chain-id", "04-genesis-block-hash-consistent", "go-wemix/chain-up/01-wemix-chain-up"], "chainId 양수·노드 간 genesis hash 일치는 있으나 주어진 genesis 기준 고정 hash·chainId 동시 대조는 없다."),
 40: ("partial", ["basic/04-basic-sync", "fault/03-fault-node-recover"], "sameBlockHash 및 재시작 catchup은 검사하나 full sync 경로와 상태 재실행/stateRoot를 직접 관측하지 않는다."),
 41: ("partial", ["02b-effective-gas-price-regular-bp-en", "basic/04-basic-sync"], "EN receipt equality·sameBlockHash는 snap pivot/state 경로 완료 증거가 아니다. syncmode·snap 경로 및 stateRoot 확인 필요."),
 42: ("partial", ["fault/03-fault-node-recover", "fault/02-fault-node-crash", "go-wemix/fault/01-wemix-node-crash", "samples/02-sample-lifecycle"], "공통 fault/recover는 재시작 이후 catchup/hash 확인. go-wemix 전용 crash는 restart 호출로 끝나 재합류 확인이 없다. DB 손상/보존 증거 보강 필요."),
 43: ("partial", ["basic/02-basic-peers", "20-admin-peers-populated", "go-wemix/chain-up/01-wemix-chain-up"], "peer count 또는 admin_peers 일부를 확인하나 bootnodes로 연결되었는지와 양쪽 RPC 정보의 연계 확인은 없다."),
 44: ("partial", ["fault/03-fault-node-recover"], "노드 중단 후 높이12 catchup/hash만 검사한다. downloader 큐 경로 및 gap>=10 조건을 명시 보장하지 않는다."),
 45: ("partial", ["basic/04-basic-sync", "basic/06-basic-txpool-propagation"], "노드 간 hash/잔액 동기화 결과만 확인하며 NewBlock/NewBlockHashes 및 block fetcher 진입은 관측하지 않는다."),
}

DUAL = {
 1: ("partial", ["01-wemix-brioche-block-reward"], "높이5 보상>0과 높이15 보상이 더 작음을 검사한다. 모든 halving 경계·정확 보상 산식은 대조하지 않는다."),
 2: ("chain_specific", ["01-block-period-one-second"], "연속3개 블록의 timestamp 차이1을 직접 확인하지만 WBFT 주기 기준으로 go-wemix 대상이 아니다."),
 3: ("chain_specific", ["02-wbft-seals-quorum", "basic/07-basic-wbft-consensus"], "02는 서명 nonempty만, basic07은 prevCommittedSeal 개수>=3이다. 현재 committed/prepared의 quorum·검증과 동일하지 않다."),
 4: ("chain_specific", ["03-epoch-transition-carries-epoch-info", "04b-validator-add-member-epoch-activates"], "epochInfo 존재 및 validator 추가·활성 검사이나 WBFT 거버넌스/epoch 전용이다."),
 5: ("chain_specific", ["10-gastip-governance-updates-header", "14-stablenet-gastip-field"], "StableNet 거버넌스·WBFT GasTip의 변경/존재 검사다."),
 6: ("chain_specific", ["fault/02-fault-node-crash", "fault/06-fault-txpool-leader-change", "go-wbft/fault/01-wbft-node-crash"], "고정 노드 중단 후 blockAdvance만 검사한다. 현재 proposer 식별·실제 round change·새 proposer 전환을 확인하지 않는다."),
 7: ("chain_specific", ["fault/02-fault-node-crash"], "재시작 후 동일 hash를 비교하나 round change 발생 확인 및 N/N+1 parentHash 연결 대조가 없다."),
 8: ("no_match", [], "연속 proposer 균등 순환을 단언하는 JSON을 찾지 못했다. basic-consensus 제목의 참여 문구만으로 대응시키지 않았다."),
 9: ("chain_specific", ["11-prev-seals-quorum"], "prevCommittedSeal 수>=quorum은 있으나 이전 블록 committer 집합과 동일 구성원을 포함하는지 비교하지 않는다."),
 10: ("chain_specific", ["11-prev-seals-quorum"], "prevPreparedSeal 수>=quorum은 있으나 이전 블록 prepare signer와 구성원 대조가 없다."),
 11: ("chain_specific", ["13-randao-and-mixdigest-present"], "WBFT randaoReveal·mixHash nonempty 검사이며 암호학적 생성/검증은 없다."),
 12: ("no_match", [], "서명이 quorum 미만인 블록을 만들고 수락 거부를 확인하는 JSON은 없다. 노드 중단과는 다른 검증이다."),
 13: ("chain_specific", ["fault/02-fault-node-crash", "go-wbft/fault/01-wbft-node-crash"], "1노드 중단과 진행은 있지만 원문의 6/7·quorum5 구성을 쓰지 않는다."),
 14: ("chain_specific", ["fault/05-fault-two-down"], "2/4 중단 후 blockHalt와 복구는 있으나 원문의 4/7·quorum5 구성과 다르다."),
 15: ("no_match", [], "validator3의 1장애·전원 필요 조건을 명시한 JSON은 없다."),
 16: ("no_match", [], "validator6 중1장애 및 CommitSigners==5 조건을 명시한 JSON은 없다."),
 17: ("no_match", [], "validator6 중2장애에서 halt를 명시한 JSON은 없다."),
 18: ("chain_specific", ["11-node-address-returned"], "주소 regex만 검사하여 각 노드 실제 signing address와 대조하지 않는다."),
 19: ("chain_specific", ["12-validator-set-nonempty", "12b-validator-set-count"], "validator 목록 존재·개수만 확인하며 지정 블록의 정확한 집합을 대조하지 않는다."),
 20: ("chain_specific", ["13-commit-signers-quorum"], "commit signers nonnil만 요구한다. quorum 개수·validator 부분집합 여부가 빠졌다."),
 21: ("chain_specific", ["14-wbft-extra-info-fields"], "committed/prepared 존재만 검사하며 일반 블록 EpochInfo null·Randao·PrevSeal 전체를 확인하지 않는다."),
 22: ("chain_specific", ["15-istanbul-status-fields"], "WBFT status의 sealerActivity/author/blockRange/roundStats 존재 검사다."),
 23: ("chain_specific", ["16-is-validator-flags"], "노드별 true만 확인하며 비validator false 조건이 없다."),
 24: ("chain_specific", ["18-set-code-delegation"], "type4·status1·delegation code를 확인하지만 go-wemix에서 해당 타입은 미지원이다."),
 25: ("chain_specific", ["17-estimategas-authorizationlist-cost"], "auth1/auth2 비용 증가와 범위를 직접 검사하지만 EIP-7702 미지원으로 go-wemix에서 실행 불가다."),
 26: ("partial", ["17-estimategas-authorizationlist-cost", "22-estimate-gas"], "17에 일반 전송 baseline21000 단언이 있으나 파일 전체는7702 의존이다. baseline 단계만 추출해야 하며 22는 >=21000만 검사한다."),
 27: ("chain_specific", ["01-secp256r1-precompile-valid"], "유효 fixture의 32byte 1 반환을 확인하지만 go-wemix 0x100 precompile 미지원이다."),
 28: ("chain_specific", ["02-secp256r1-precompile-invalid"], "반환!=1만 확인해 원문의 빈 반환보다 약하다. go-wemix 미지원 주소의 빈 응답을 성공으로 오인하면 안 된다."),
 29: ("chain_specific", ["03-secp256r1-precompile-short-input"], "짧은 입력·빈 입력에서0x 반환을 확인하지만 go-wemix 미지원 주소의 응답으로 대체할 수 없다."),
}


def main() -> None:
    """Write all 74 reviewed crosswalk rows and per-match assertion evidence."""
    files = sorted(TC.rglob("*.json"))
    if len(files) != 189:
        raise ValueError("Scenario count changed; review mappings before reuse")
    documents = json.loads((RUN / "analyses/document-catalog.json").read_text())
    results = []
    for document in documents:
        _, group, number = document["catalog_id"].split("-")
        coverage, suffixes, difference = (COMMON if group == "C" else DUAL)[int(number)]
        matches = []
        for suffix in suffixes:
            candidates = [path for path in files if str(path.with_suffix("").relative_to(TC)).endswith(suffix)]
            if len(candidates) != 1:
                raise ValueError(f"Ambiguous/missing suffix {suffix}: {candidates}")
            path = candidates[0]
            data = json.loads(path.read_text())
            line = next(index for index, text in enumerate(path.read_text().splitlines(), 1) if '"steps"' in text)
            matches.append({"file": str(path.relative_to(RUN)), "steps_line": line,
                            "case_id": data.get("id"),
                            "assertions": [step for step in data["steps"] if "expect" in step],
                            "source_coverage": ("partial" if path.stem == "29-logs-query-well-formed" else "chain_specific" if path.stem == "23-token-approve-sets-allowance" else coverage),
                            "execution_verified": False})
        results.append({"catalog_id": document["catalog_id"], "title": document["title"],
                        "document_source": f"{document['source_file']}:{document['source_line']}",
                        "coverage": coverage, "matches": matches, "differences": difference,
                        "go_wemix_applicability": document["applicability"],
                        "meaning": "Coverage describes source assertions, not passing execution or zero-change portability."})
    (RUN / "analyses/source-crosswalk.json").write_text(json.dumps(results, ensure_ascii=False, indent=2) + "\n")
    output = ["# 문서 ↔ tests/tc 기능별 교차참조", "",
              "74개 원문 행을 모두 유지했다. 189개 JSON 중 기능·실제 steps/expect가 대응되는 파일을 기록했다. exact는 원문 핵심 기대조건의 명세상 대응이고 실행 통과나 무수정 이식을 뜻하지 않는다. partial은 조건 일부, chain_specific은 비교 대상이 체인 전용, no_match는 같은 목적의 명시적 테스트를 찾지 못한 경우다.", "",
              "일치하는 ID·제목만으로 exact를 부여하지 않았다. .sh 참조 파일이 존재한다고 추정하지 않았다. 명시적인 source coverage이므로 다른 JSON이 실행 부수효과로 비슷한 동작을 수행하는 것까지 수집하지 않았다.", "",
              "| 문서 ID | coverage | 대응 JSON(steps 줄) | 원문 대비 차이 |", "|---|---|---|---|"]
    for result in results:
        refs = "<br>".join(f"{match['file']}:{match['steps_line']}" for match in result["matches"]) or "없음"
        output.append(f"| {result['catalog_id']} | {result['coverage']} | {refs} | {result['differences']} |")
    output.extend(["", "분류 집계: " + json.dumps(dict(Counter(result["coverage"] for result in results)), ensure_ascii=False), "",
                   "각 대응 파일의 expect 원문은 source-crosswalk.json에 보존했다. 정적 비교만 수행했으며 RPC·노드·스크립트는 실행하지 않았다."])
    (RUN / "analyses/source-crosswalk.md").write_text("\n".join(output) + "\n")


if __name__ == "__main__":
    main()
