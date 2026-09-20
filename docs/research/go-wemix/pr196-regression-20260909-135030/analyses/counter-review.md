# PR196 반박 검토

아래 판정은 `sources/pr-head` 정적 코드 분석이다. 재현 테스트를 실행하지 않았으며 위험 후보를 확정 장애와 구분한다. 모든 코드 경로는 이 실행 디렉터리를 기준으로 한다.

## 1. full sync와 snap receipt import의 크기 검증 차이 — 유지, 범위 한정

**확인됨:** full import는 `sources/pr-head/eth/downloader/downloader.go:1548`의 `InsertChain` → `sources/pr-head/core/blockchain.go:1450`의 insert iterator → `sources/pr-head/core/blockchain_insert.go:124`의 `ValidateBody` → `sources/pr-head/core/block_validator.go:55`의 `block.Size()>MaxBlockSize` 검사를 지난다.

snap의 pivot 이전 데이터와 pivot block은 각각 `sources/pr-head/eth/downloader/downloader.go:1736`, `:1748`에서 `InsertReceiptChain`을 호출한다. `sources/pr-head/core/blockchain.go:917`의 구현은 연속성·헤더 존재 등 확인 후 body/receipts를 쓴다(`:1094`, `:1110`, `:1111`). 이 함수에는 `ValidateBody`/`MaxBlockSize` 검사가 없다. pivot 이후는 full sync로 넘어가므로 **snap 전체가 무검증**이라는 표현은 틀리다(`sources/pr-head/eth/downloader/downloader.go:1747` 주석).

**반론 검토:** upstream 검증은 존재한다. body root/uncle hash는 `sources/pr-head/eth/downloader/queue.go:772`, `:775`, receipt root는 `:798`에서 header와 비교한다. 해시도 packet handler에서 계산한다(`sources/pr-head/eth/protocols/eth/handlers.go:621`, `:622`, `:722`). transport 메시지 상한도 `sources/pr-head/eth/protocols/eth/handler.go:254`에서 검사한다. 그러나 root 일치와 메시지 10MiB는 블록 RLP 8MiB 검사와 동치가 아니다. 일치하는 header/roots를 가진 8MiB 초과 body가 모든 이 검사를 통과하는지에 대한 끝까지의 재현은 미확인이다.

**테스트 오라클:** 같은 초과 크기 블록 fixture를 full import, snap pre-pivot, pivot 경로에 투입한다. header/txRoot/uncleRoot/receiptRoot 및 체인 연결은 모두 정상으로 구성하여 다른 검증에서 먼저 탈락하지 않게 한다. 8MiB 적용 정책이 모든 이 경로에 동일하다는 요구라면 초과 fixture는 동일한 크기 오류로 거부되고 body/receipt DB 및 fast head가 갱신되지 않아야 한다. 역사 블록을 예외로 둘 정책이라면 적용 높이와 예외를 먼저 명문화한다. transport decoder부터 queue와 DB까지 포함하는 통합 fixture로 수신 가능한 입력임을 확인한다.

## 2. environment.size와 실제 RLP 불일치 — 유지; invalid block 생성 주장은 미확인

`sources/pr-head/miner/worker.go:864`은 `header.Size()`로 초기화하고 `:930`은 `tx.Size()`를 더한다. `:147`의 입장 조건은 `env.size + tx.Size() < 8,388,608 - 1,000,000`이다.

`sources/pr-head/core/types/block.go:187`의 Header.Size는 메모리 사용량 추정이다. 반면 `Block.Size`는 실제 RLP counter 결과다(`:405`). `Transaction.Size`는 캐시가 없으면 inner를 RLP 인코딩한 길이(`sources/pr-head/core/types/transaction.go:430`)이고, 디코드 시에는 캐시를 다른 입력 길이로 채운다(`:144`, `:155`, `:179`, `:208`). typed transaction의 실제 블록 인코딩에는 type byte와 RLP string envelope가 있다(`:101`, `:114`, `:118`). 따라서 합계가 직렬화 길이와 일치한다는 전제는 성립하지 않는다. type 0x16만의 문제가 아니라 0x01/0x02에도 해당하며, 0x16의 fee payer/signature 가변 필드도 포함해 확인해야 한다.

**과장 방지:** 1,000,000바이트 buffer가 있다. 추정 오차가 buffer를 초과해 실제 invalid block이 생성된다고 아직 입증하지 않았다. 현재 구조에서 finalized RLP 검사 추가가 반드시 필요한 코드 수정이라고 단정하지 않고, 먼저 invariant 검증을 요구한다. 보상·seal/header 확정 이후 블록이 `FinalizeAndAssemble`(`sources/pr-head/miner/worker.go:1512`, `:1716`, `:1751`)와 저장(`:815`, `:1817`)을 지나므로 oracle은 최종 block을 대상으로 해야 한다.

**테스트 오라클:** type 0/1/2/22 각각과 혼합 묶음, 생성 객체/DecodeRLP/UnmarshalBinary 유래 객체, nonce·signature·data 길이 RLP 경계, 최대 개수의 작은 tx와 큰 tx 조합을 대상으로 최종 `len(rlp.EncodeToBytes(block)) == uint64(block.Size()) <= 8,388,608`을 검사한다. 입장 거부 tx는 다음 후보/다음 블록 처리와 nonce 순서도 확인한다. buffer 전후 `<` 경계 자체와 최종 RLP 상한 oracle은 별도로 둔다. buffer 내 안전성이 입증되면 추정 불일치만으로 실패 판정하지 않는다.

## 3. 10MiB 메시지 한도와 body/receipt batch — body 일반론 수정, receipts 가능성은 조건부 유지

`sources/pr-head/eth/protocols/eth/handler.go:36`의 softResponseLimit는 2MiB다. body 서비스는 현재 누적 바이트가 2MiB 미만일 때 body 한 개를 추가하고 다음 반복에서 중단한다(`sources/pr-head/eth/protocols/eth/handlers.go:361`, `:366`, `:367`). **여러 개의 8MiB body가 한 batch에 계속 쌓인다는 설명은 틀리다.**

모든 대상 블록이 8MiB 이하라는 전제에서는 마지막 body 추가 전 누적량 S<2MiB, 마지막 body B는 전체 블록에서 header 등을 뺀 크기이므로 B<8MiB다. 따라서 raw body 합계 S+B<10MiB다. packet request ID/list wrapper까지 포함한 실제 wire 길이는 별도 직렬화로 확인해야 하지만 이 사실만으로 정상 body batch가 10MiB를 초과한다고 단정할 수 없다. **역사상 8MiB 초과 블록**을 serving하는 경우는 전제가 달라 별도 테스트가 필요하다.

receipts 서비스도 추가 전 2MiB 검사 후 마지막 receipt set 한 개를 추가한다(`sources/pr-head/eth/protocols/eth/handlers.go:477`, `:488`, `:492`, `:493`). 그러나 receipt set은 block body와 별도 데이터다. RLP block 8MiB 제한은 logs를 포함하는 receipts에 직접 적용되지 않는다. `sources/pr-head/params/protocol_params.go:36`의 LOG 데이터 비용(8 gas/byte), 블록 gas limit, 메모리 확장, 로그 개수/receipt 고정 overhead에 의해 실현 가능한 크기가 제한된다. 따라서 단일 receipt set 또는 `<2MiB + 마지막 set`이 10MiB를 넘는지는 실제 대상 gas 설정으로 구성·인코딩하기 전에는 확정할 수 없다. 가능한 gas limit을 임의로 높여 재현한 결과를 현재 운영 환경 문제로 표현해서는 안 된다.

**테스트 오라클:**

1. 유효한 8MiB 이하 블록들로 `ServiceGetBlockBodiesQuery`를 호출한다. 앞부분을 2MiB 직전까지 채우고 큰 마지막 body를 넣는다. body 목록의 길이가 아니라 request ID를 포함한 실제 eth packet RLP가 10MiB 이하인지 검사한다. 이 조건에서 초과가 나오면 구체 bytes와 wrapper 기여를 보고한다.
2. 실제 대상 block gas limit에서 최대 로그 데이터/다수 receipt 조합을 생성한다. receipt set 단건·softlimit 직전 앞부분+마지막 set의 실제 packet 길이와 receiver 수용 여부를 확인한다. 크기 fixture가 실제 execution gas 규칙을 통과해야 한다.
3. 10MiB-1/정확히10MiB/10MiB+1 transport 입력으로 수신 경계 자체를 별도 검증한다. 비교식은 `msg.Size > maxMessageSize`이므로 정확히10MiB는 이 크기 검사만 통과한다(`sources/pr-head/eth/protocols/eth/handler.go:254`); 유효 packet decode 성공은 추가 조건이다.
4. 기존 8MiB 초과 역사 블록/큰 receipt가 실제 존재하는지 먼저 조회한다. 존재하면 full/snap/repair/history serving이 중단되지 않아야 한다는 호환성 요구와 크기 정책을 별도로 결정한다.

## 4. local dev와 PR master 차이는 PR에서 삭제한 코드인가 — 삭제 주장 기각

`sources/pr196.json`의 base는 master `c8321fc693bef619ec8abd80c82f5a224e89802a`, head는 hotfix/block-size-limit `5a93553cb25d41770c7419cd325229ac81bb3042`다. 현재 로컬은 dev `902f9fce85c108cf24cdeb77bc70a622049748ce`다. `sources/local-to-pr-base.txt`에는 이미 PR base와 local dev 사이에 miner/worker.go, wemix/admin.go, wemix/etcdutil.go, wemix/sync.go 등의 차이가 나타난다.

PR diff의 변경 파일은 `core/block_validator.go`, 해당 test, `core/error.go`, `eth/protocols/eth/protocol.go`, `miner/worker.go`, 해당 test, `params/protocol_params.go` 7개다(`sources/pr196.diff` 각 diff header). local dev와 PR head만 비교해 사라진 로직을 PR196이 삭제했다고 판정할 수 없다. base→head 변화만 PR 변경으로 귀속하고, dev로 이식할 때 기존 로직과 크기 제한이 함께 유지되는지는 별도 병합 회귀로 검증한다.
