# 공통 테스트 family와 원본 추적

> ID 표기는 Confluence 원래 명세 ID를 우선한다. 실행 ID는 별도 표기한다. ID 대응은 전체 명세 충족이나 PASS를 뜻하지 않는다. [ID 대응표](existing-tc-specs.md)

총 263개 원본 중 138개를 세 체인 공통 후보로 분류했다(exact-config 42, adapter-required 96). 체인 전용 124개와 외부 조건 미확인 1개는 공통 수행에서 제외한다. 후보는 격리 compose 환경 기준이며 공개 메인넷 RPC에서 즉시 실행할 수 있다는 뜻이 아니다.

명세와 JSON이 겹쳐도 원본 ID를 보존했다. 아래 family 개수는 독립 실행 수가 아니다. 각 행의 실제 assertion, 의존 필드와 설정/구현 범위는 all-tests.json에 있다. 체인별 근거는 chain-differences.md의 CD 항목, 하네스 제약은 harness-dependencies.md를 함께 읽는다.


## evm-contract

원본 18개 / 세 체인 공통 후보 18개.

공통 fork에서 실행 가능한 fixture bytecode를 배포하여 상태/eth_call/estimateGas/revert/OOG를 검사한다. 최신 compiler 기본 opcode를 세 체인 모두 지원한다고 가정하지 않는다.

공통 후보 ID: RT-A-3-01 / TX-005 (분석 DOC-C-014), RT-A-3-02 (분석 DOC-C-015), RT-A-3-03 (분석 DOC-C-016), RT-A-3-04 (분석 DOC-C-017), RT-A-3-05 (분석 DOC-C-018), RT-A-3-06 / TX-017 (분석 DOC-C-019), RT-A-3-07 / TX-018 (분석 DOC-C-020), TC-4-2-02 (분석 DOC-D-026), RT-A-3-01 (분석 TC-039), RT-A-3-05 (분석 TC-040), RT-A-3-06 (분석 TC-041), RT-A-3-07 (분석 TC-042), RT-A-4-04 / RPC-014 [부분 대응] (분석 TC-044), wemix-tx-and-contract [명세 ID 미확인] (분석 TC-056), RT-A-3-04 (분석 TC-064), stablenet-negative-tx-revert [명세 ID 미확인] (분석 TC-072), wbft-tx-and-contract [명세 ID 미확인] (분석 TC-077), stablenet-register-contract [명세 ID 미확인] (분석 TC-086)

전체 원본 ID: RT-A-3-01 / TX-005 (분석 DOC-C-014), RT-A-3-02 (분석 DOC-C-015), RT-A-3-03 (분석 DOC-C-016), RT-A-3-04 (분석 DOC-C-017), RT-A-3-05 (분석 DOC-C-018), RT-A-3-06 / TX-017 (분석 DOC-C-019), RT-A-3-07 / TX-018 (분석 DOC-C-020), TC-4-2-02 (분석 DOC-D-026), RT-A-3-01 (분석 TC-039), RT-A-3-05 (분석 TC-040), RT-A-3-06 (분석 TC-041), RT-A-3-07 (분석 TC-042), RT-A-4-04 / RPC-014 [부분 대응] (분석 TC-044), wemix-tx-and-contract [명세 ID 미확인] (분석 TC-056), RT-A-3-04 (분석 TC-064), stablenet-negative-tx-revert [명세 ID 미확인] (분석 TC-072), wbft-tx-and-contract [명세 ID 미확인] (분석 TC-077), stablenet-register-contract [명세 ID 미확인] (분석 TC-086)

체인별 차이: CD-08, CD-09

- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## external-faucet

원본 1개 / 세 체인 공통 후보 0개.

각 네트워크 faucet 서비스와 자금 지원 여부를 확인하기 전 공통 수행 가능 여부를 확정하지 않는다.

공통 후보 ID: 없음

전체 원본 ID: stablenet-faucet-funds [명세 ID 미확인] (분석 TC-085)

체인별 차이: CD-01, CD-08

## fault-topology

원본 10개 / 세 체인 공통 후보 10개.

격리 환경에서 각 합의가 허용하는 장애수·membership을 적용한 뒤 회복/블록진행/고정높이hash를 검사한다. PoA와 WBFT quorum 수치를 동일하게 강제하지 않는다.

공통 후보 ID: fault-network-partition [명세 ID 미확인] (분석 TC-007), fault-node-crash [명세 ID 미확인] (분석 TC-008), fault-node-recover [명세 ID 미확인] (분석 TC-009), fault-p2p-topology [명세 ID 미확인] (분석 TC-010), fault-two-down [명세 ID 미확인] (분석 TC-011), fault-txpool-leader-change [명세 ID 미확인] (분석 TC-012), wemix-node-crash [명세 ID 미확인] (분석 TC-054), stablenet-proxied-pn-routing [명세 ID 미확인] (분석 TC-071), wbft-node-crash [명세 ID 미확인] (분석 TC-076), sample-lifecycle-node-restart [명세 ID 미확인] (분석 TC-083)

전체 원본 ID: fault-network-partition [명세 ID 미확인] (분석 TC-007), fault-node-crash [명세 ID 미확인] (분석 TC-008), fault-node-recover [명세 ID 미확인] (분석 TC-009), fault-p2p-topology [명세 ID 미확인] (분석 TC-010), fault-two-down [명세 ID 미확인] (분석 TC-011), fault-txpool-leader-change [명세 ID 미확인] (분석 TC-012), wemix-node-crash [명세 ID 미확인] (분석 TC-054), stablenet-proxied-pn-routing [명세 ID 미확인] (분석 TC-071), wbft-node-crash [명세 ID 미확인] (분석 TC-076), sample-lifecycle-node-restart [명세 ID 미확인] (분석 TC-083)

체인별 차이: CD-10

- PoA/etcd와 WBFT membership/quorum별 허용 장애수 설정; latest hash만으로 특정 downloader/fetcher 경로 검증했다고 보지 않음
- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 장애 주입과 leader 교체를 PoA/etcd 또는 WBFT별로 처리한다.
- 환경 구성·사전조건 검증

## fee-delegation

원본 13개 / 세 체인 공통 후보 13개.

세 체인 공통 type 0x16의 sender/feePayer 서명·잔액 책임을 검사한다. RPC 존재 검사와 성공 서명·실제 무효서명 거부를 구별한다.

공통 후보 ID: RT-D-01 / TX-004 (분석 DOC-C-004), RT-D-03 / TX-014 (분석 DOC-C-005), RT-D-04 / TX-015 (분석 DOC-C-006), RT-D-05 / TX-016 (분석 DOC-C-007), RT-G-5-01 (분석 DOC-C-038), RT-G-5-01 (분석 TC-027), RT-D-01 (분석 TC-045), RT-D-03 (분석 TC-046), RT-D-04 (분석 TC-047), RT-D-05 (분석 TC-048), RT-D-03 / TX-014 [부분 대응] (분석 TC-049), RT-D-04 / TX-015 [부분 대응] (분석 TC-050), RT-D-05 / TX-016 [부분 대응] (분석 TC-051)

전체 원본 ID: RT-D-01 / TX-004 (분석 DOC-C-004), RT-D-03 / TX-014 (분석 DOC-C-005), RT-D-04 / TX-015 (분석 DOC-C-006), RT-D-05 / TX-016 (분석 DOC-C-007), RT-G-5-01 (분석 DOC-C-038), RT-G-5-01 (분석 TC-027), RT-D-01 (분석 TC-045), RT-D-03 (분석 TC-046), RT-D-04 (분석 TC-047), RT-D-05 (분석 TC-048), RT-D-03 / TX-014 [부분 대응] (분석 TC-049), RT-D-04 / TX-015 [부분 대응] (분석 TC-050), RT-D-05 / TX-016 [부분 대응] (분석 TC-051)

체인별 차이: CD-03, CD-06, CD-07

- type 0x16 sender/feePayer 서명, 실제 무효 서명 fixture, 잔액 차감 주체와 RPC 서명 권한 준비
- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## fee-observation

원본 10개 / 세 체인 공통 후보 10개.

같은 tx와 포함 블록의 receipt 값을 노드 간 비교하거나 실제 tx type의 fee 계산식을 적용한다. StableNet header GasTip/authorized 정책은 별도 oracle이다.

공통 후보 ID: RT-A-2-08 (분석 DOC-C-012), RT-G-2-01 / RPC-016 (분석 DOC-C-033), RT-G-2-02 (분석 DOC-C-034), RT-G-2-03 / RPC-017 (분석 DOC-C-035), TC-4-6-02 (분석 TC-016), RT-A-2-08 (분석 TC-036), RT-G-2-01 (분석 TC-059), RT-G-2-03 (분석 TC-060), TC-4-6-02 (분석 TC-105), RT-G-2-01 [부분 대응] (분석 TC-133)

전체 원본 ID: RT-A-2-08 (분석 DOC-C-012), RT-G-2-01 / RPC-016 (분석 DOC-C-033), RT-G-2-02 (분석 DOC-C-034), RT-G-2-03 / RPC-017 (분석 DOC-C-035), TC-4-6-02 (분석 TC-016), RT-A-2-08 (분석 TC-036), RT-G-2-01 (분석 TC-059), RT-G-2-03 (분석 TC-060), TC-4-6-02 (분석 TC-105), RT-G-2-01 [부분 대응] (분석 TC-133)

체인별 차이: CD-06

- Istanbul GasTip 직접 조회를 체인별 fee observation으로 분리; 동일 포함 블록에서 실제 tx type에 맞는 영수증 비교
- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## fee-policy

원본 12개 / 세 체인 공통 후보 11개.

부모 블록 gasUsed/target/최저·최고 baseFee 및 pool local/remote 정책에 따라 기대값을 계산한다. feeCap<baseFee라는 사실만으로 submit 즉시 reject를 요구하지 않는다. RT-C-07 (분석 TC-128)의 최대 clamp는 WBFT post-Croissant에 없어 공통 제외. RT-C-06 (분석 TC-127) 하한은 WBFT의 0을 허용하는 profile로 분리하며 양의 최소값을 요구하지 않는다.

공통 후보 ID: RT-A-2-05a (분석 DOC-C-009), TC-1-3-04 (분석 TC-013), TC-1-3-05 (분석 TC-014), TC-1-3-06 (분석 TC-015), RT-A-2-05a (분석 TC-033), RT-C-03 [부분 대응] (분석 TC-124), RT-C-04 [부분 대응] (분석 TC-125), RT-C-05 [부분 대응] (분석 TC-126), RT-C-06 (분석 TC-127), TC-1-3-03 [부분 대응] (분석 TC-129), TC-1-3-02 [부분 대응] (분석 TC-130)

전체 원본 ID: RT-A-2-05a (분석 DOC-C-009), TC-1-3-04 (분석 TC-013), TC-1-3-05 (분석 TC-014), TC-1-3-06 (분석 TC-015), RT-A-2-05a (분석 TC-033), RT-C-03 [부분 대응] (분석 TC-124), RT-C-04 [부분 대응] (분석 TC-125), RT-C-05 [부분 대응] (분석 TC-126), RT-C-06 (분석 TC-127), RT-C-07 (분석 TC-128), TC-1-3-03 [부분 대응] (분석 TC-129), TC-1-3-02 [부분 대응] (분석 TC-130)

체인별 차이: CD-06

- WBFT의 baseFee 하한은 0이며 양의 protocol minimum을 요구하지 않는다. profile별 하한 비교는 상한/하한 도달 경계 검증을 대신하지 않는다.
- 수수료 하한·local/remote·pool rejection 대 pending·target gasUsed·min/max baseFee oracle 분리; 하드코딩 비율/금액 그대로 쓰지 않음
- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## genesis-upgrade

원본 4개 / 세 체인 공통 후보 4개.

각 구현체의 genesis fixture를 준비하고 같은 체인 이전/이후 바이너리의 DB/서명 조회 보존을 검사한다. 서로 다른 세 프로젝트 바이너리의 무조건 상호 교환 테스트가 아니다.

공통 후보 ID: RT-A-1-01 (분석 DOC-C-039), TC-3-1-04 (분석 TC-017), TC-4-1-03 (분석 TC-018), TC-5-3-01 (분석 TC-019)

전체 원본 ID: RT-A-1-01 (분석 DOC-C-039), TC-3-1-04 (분석 TC-017), TC-4-1-03 (분석 TC-018), TC-5-3-01 (분석 TC-019)

체인별 차이: CD-01, CD-08

- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## handoff-specific

원본 1개 / 세 체인 공통 후보 0개.

WEMIX3에서 WEMIX4로의 전환 시나리오이며 세 독립 체인의 공통 기능 테스트가 아니다.

공통 후보 ID: 없음

전체 원본 ID: wemix-wbft-handoff [명세 ID 미확인] (분석 TC-078)

체인별 차이: CD-10, CD-11

## harness-vocabulary

원본 1개 / 세 체인 공통 후보 1개.

주소 파생/checksum/등록 참조 자체는 하네스 공통 기능이다. 체인 합의 또는 EVM 실행 검증으로 세지 않는다.

공통 후보 ID: stablenet-derived-vocabulary [명세 ID 미확인] (분석 TC-084)

전체 원본 ID: stablenet-derived-vocabulary [명세 ID 미확인] (분석 TC-084)

체인별 차이: CD-01, CD-08

## invalid-transaction

원본 5개 / 세 체인 공통 후보 5개.

잔액 부족 또는 블록 gasLimit 초과라는 준비 조건이 실제로 성립함을 확인하고 제출 실패와 nonce/state 불변을 검사한다. 기존 JSON이 이 모든 조건을 검사한다는 뜻은 아니다.

공통 후보 ID: RT-A-2-06 / TX-011 (분석 DOC-C-010), RT-A-2-07 / TX-012 (분석 DOC-C-011), RT-A-2-06 (분석 TC-034), RT-A-2-07 (분석 TC-035), RT-A-2-07 / TX-012 [부분 대응] (분석 TC-131)

전체 원본 ID: RT-A-2-06 / TX-011 (분석 DOC-C-010), RT-A-2-07 / TX-012 (분석 DOC-C-011), RT-A-2-06 (분석 TC-034), RT-A-2-07 (분석 TC-035), RT-A-2-07 / TX-012 [부분 대응] (분석 TC-131)

체인별 차이: CD-01, CD-08

- 원문 steps의 istanbul_getWbftExtraInfo GasTip 조회를 제거하고 체인별 수수료 공급 adapter로 대체한다. WEMIX3에서 원문 그대로 실행되지 않는다.
- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## logs-subscription

원본 6개 / 세 체인 공통 후보 6개.

동일 배포 계약·event topic·block 범위로 eth_getLogs와 WS 수신을 검사한다. WS endpoint와 event trigger 및 재연결 종료 처리가 필요하다.

공통 후보 ID: RT-A-4-04 / RPC-014 (분석 DOC-C-024), RT-A-4-06 / RPC-020 (분석 DOC-C-026), RT-A-4-07 / RPC-021 (분석 DOC-C-027), RT-A-4-04 (분석 TC-066), RT-A-4-06 (분석 TC-068), RT-A-4-07 (분석 TC-069)

전체 원본 ID: RT-A-4-04 / RPC-014 (분석 DOC-C-024), RT-A-4-06 / RPC-020 (분석 DOC-C-026), RT-A-4-07 / RPC-021 (분석 DOC-C-027), RT-A-4-04 (분석 TC-066), RT-A-4-06 (분석 TC-068), RT-A-4-07 (분석 TC-069)

체인별 차이: CD-01, CD-08

- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## modern-evm-specific

원본 10개 / 세 체인 공통 후보 0개.

7702/P256 원문의 성공 기대값은 WEMIX3에 그대로 적용되지 않는다. WEMIX4와 StableNet도 활성 fork 조건이 다르다.

공통 후보 ID: 없음

전체 원본 ID: RT-A-2-10 / TX-008 (분석 DOC-D-024), TC-4-2-01 / TC-4-2-03 (분석 DOC-D-025), TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-009 (분석 DOC-D-027), TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-019 (분석 DOC-D-028), TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-020 (분석 DOC-D-029), TC-4-2-01 / TC-4-2-03 (분석 TC-100), RT-A-2-10 (분석 TC-154), TX-009 [부분 대응] (분석 TC-187), TX-019 [부분 대응] (분석 TC-188), TX-020 [부분 대응] (분석 TC-189)

체인별 차이: CD-04, CD-05

## nonce-replacement

원본 6개 / 세 체인 공통 후보 6개.

동일 계정 nonce 순서와 동일 nonce 교체의 한 번 포함을 검사한다. 미래 nonce queue와 replacement 가격상승 policy는 profile로 분리한다.

공통 후보 ID: RT-A-2-04 / TX-010 (분석 DOC-C-008), RT-A-2-09 / TX-013 (분석 DOC-C-013), RT-A-2-04 (분석 TC-031), TX-010 [부분 대응] (분석 TC-032), RT-A-2-09 (분석 TC-037), TX-013 [부분 대응] (분석 TC-038)

전체 원본 ID: RT-A-2-04 / TX-010 (분석 DOC-C-008), RT-A-2-09 / TX-013 (분석 DOC-C-013), RT-A-2-04 (분석 TC-031), TX-010 [부분 대응] (분석 TC-032), RT-A-2-09 (분석 TC-037), TX-013 [부분 대응] (분석 TC-038)

체인별 차이: CD-01, CD-08

- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## reward-compatibility

원본 2개 / 세 체인 공통 후보 0개.

Brioche reward는 WEMIX3/4 전용 비교이며 StableNet의 다른 보상을 같은 함수/값으로 취급하지 않는다.

공통 후보 ID: 없음

전체 원본 ID: RPC-008 (분석 DOC-D-001), BRIOCHE-02 / RPC-008 [부분 대응] (분석 TC-055)

체인별 차이: CD-11

## rpc-basics

원본 21개 / 세 체인 공통 후보 21개.

같은 고정 block/tx/address를 대상으로 원문 JSON에 명시된 필드·형식·일치 조건만 공통화한다. latest를 동시에 조회한 것만으로 동기화 완료를 증명하지 않는다.

공통 후보 ID: RT-A-4-01 / RPC-001 (분석 DOC-C-021), RT-A-4-02 / RPC-012 (분석 DOC-C-022), RT-A-4-03 (분석 DOC-C-023), RT-A-4-05 / RPC-013 (분석 DOC-C-025), RT-G-1-01 / RPC-002 (분석 DOC-C-028), RT-G-1-02 / RPC-002 (분석 DOC-C-029), RT-G-1-03 (분석 DOC-C-030), RT-G-1-04 / RPC-007 (분석 DOC-C-031), RT-G-1-05 / RPC-015 (분석 DOC-C-032), basic-rpc-health [명세 ID 미확인] (분석 TC-003), basic-tx-send [명세 ID 미확인] (분석 TC-005), RT-G-1-01 (분석 TC-020), RT-G-1-02 (분석 TC-021), RT-G-1-03 (분석 TC-022), RT-G-1-04 (분석 TC-023), RT-G-1-05 (분석 TC-024), RT-A-4-02 (분석 TC-065), RT-A-1-01 (분석 TC-067), remote-rpc-health [명세 ID 미확인] (분석 TC-079), remote-chain-info [명세 ID 미확인] (분석 TC-080), remote-balance-check [명세 ID 미확인] (분석 TC-081)

전체 원본 ID: RT-A-4-01 / RPC-001 (분석 DOC-C-021), RT-A-4-02 / RPC-012 (분석 DOC-C-022), RT-A-4-03 (분석 DOC-C-023), RT-A-4-05 / RPC-013 (분석 DOC-C-025), RT-G-1-01 / RPC-002 (분석 DOC-C-028), RT-G-1-02 / RPC-002 (분석 DOC-C-029), RT-G-1-03 (분석 DOC-C-030), RT-G-1-04 / RPC-007 (분석 DOC-C-031), RT-G-1-05 / RPC-015 (분석 DOC-C-032), basic-rpc-health [명세 ID 미확인] (분석 TC-003), basic-tx-send [명세 ID 미확인] (분석 TC-005), RT-G-1-01 (분석 TC-020), RT-G-1-02 (분석 TC-021), RT-G-1-03 (분석 TC-022), RT-G-1-04 (분석 TC-023), RT-G-1-05 (분석 TC-024), RT-A-4-02 (분석 TC-065), RT-A-1-01 (분석 TC-067), remote-rpc-health [명세 ID 미확인] (분석 TC-079), remote-chain-info [명세 ID 미확인] (분석 TC-080), remote-balance-check [명세 ID 미확인] (분석 TC-081)

체인별 차이: CD-01, CD-08

- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## stablenet-specific

원본 72개 / 세 체인 공통 후보 0개.

native coin adapter·거버넌스·blacklist/authorized·Boho/Anzeon 원래 의미를 유지한 테스트는 StableNet 전용이다. ERC20 일반 테스트로 바꾸면 별도 파생 시나리오다.

공통 후보 ID: 없음

전체 원본 ID: basic-wbft-consensus [명세 ID 미확인] (분석 TC-087), stablenet-delayed-fork [명세 ID 미확인] (분석 TC-088), TC-5-2-05 (분석 TC-089), TC-1-1-01 / TC-1-1-10 (분석 TC-090), TC-1-1-02 (분석 TC-091), TC-1-1-03 (분석 TC-092), TC-1-1-04 (분석 TC-093), TC-1-1-05 / TC-1-1-09 (분석 TC-094), TC-1-1-06 (분석 TC-095), TC-1-1-07 (분석 TC-096), TC-1-1-11 / TC-1-1-12 (분석 TC-097), TC-4-1-01 (분석 TC-098), TC-4-1-02 (분석 TC-099), TC-5-2-01 / TC-5-2-02 (분석 TC-101), TC-5-2-04 (분석 TC-102), TC-1-1-09 / TC-1-1-10 [부분 대응] (분석 TC-103), TC-4-6-01 (분석 TC-104), TC-4-6-04 (분석 TC-106), TC-4-5-01 / TC-4-5-02 (분석 TC-107), TC-4-5-02 [부분 대응] (분석 TC-108), TC-4-5-01 / TC-4-5-02 / TC-4-5-07 [부분 대응] (분석 TC-109), TC-4-5-05 / TC-4-5-06 (분석 TC-110), TC-4-5-07 (분석 TC-111), TC-4-5-08 (분석 TC-112), TC-4-5-09 (분석 TC-113), TC-4-5-10 / TC-4-5-11 (분석 TC-114), TC-5-2-06 (분석 TC-115), TC-4-3-01 (분석 TC-116), TC-4-3-02 (분석 TC-117), TC-4-3-03 (분석 TC-118), TC-4-3-04 (분석 TC-119), TC-4-3-05 (분석 TC-120), TC-4-3-06 (분석 TC-121), RT-C-01 (분석 TC-122), RT-C-02 (분석 TC-123), RT-G-1-06 (분석 TC-132), RT-G-2-02 (분석 TC-134), RT-G-2-04 (분석 TC-135), RT-G-5-02 (분석 TC-143), RT-G-5-03 (분석 TC-144), RT-E-01 (분석 TC-145), RT-E-02 (분석 TC-146), RT-E-03 (분석 TC-147), RT-E-04 (분석 TC-148), RT-E-05 (분석 TC-149), RT-E-06 (분석 TC-150), RT-E-07 (분석 TC-151), RT-E-08 (분석 TC-152), RT-E-09 (분석 TC-153), RT-F-1-01 (분석 TC-155), RT-F-1-01 [부분 대응] (분석 TC-156), RT-F-1-02 (분석 TC-157), RT-F-1-03 [부분 대응] (분석 TC-158), RT-F-1-04 (분석 TC-159), RT-F-1-05 (분석 TC-160), RT-F-2-01 (분석 TC-161), RT-F-2-02 (분석 TC-162), RT-F-2-03 (분석 TC-163), RT-F-3-04 (분석 TC-164), RT-B-06 (분석 TC-165), RT-F-3-06 (분석 TC-166), RT-F-4-01 (분석 TC-167), RT-F-4-02 (분석 TC-168), RT-F-4-03 (분석 TC-169), RT-F-4-04 (분석 TC-170), RT-F-5-01 (분석 TC-171), RT-F-5-03 (분석 TC-172), RT-F-5-04 / RT-F-5-09 [부분 대응] (분석 TC-173), RT-F-5-05 (분석 TC-174), RT-F-5-08 (분석 TC-175), token-metadata [명세 ID 미확인] (분석 TC-176), minter-status-readable [명세 ID 미확인] (분석 TC-177)

체인별 차이: CD-07, CD-09, CD-11

## stress

원본 2개 / 세 체인 공통 후보 2개.

block time 또는 gas 부하를 profile 예산으로 검사한다. gas fill은 byte-size flood와 다르며 성공receipt를 기다린 거래는 pending 복구 근거가 아니다.

공통 후보 ID: stress-block-time [명세 ID 미확인] (분석 TC-057), stress-tx-flood [명세 ID 미확인] (분석 TC-058)

전체 원본 ID: stress-block-time [명세 ID 미확인] (분석 TC-057), stress-tx-flood [명세 ID 미확인] (분석 TC-058)

체인별 차이: CD-01, CD-08

- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## sync-lifecycle

원본 18개 / 세 체인 공통 후보 18개.

프로필에 정의한 생산/피어연결/재기동/동기화 관측을 공유한다. 특정 full/snap/downloader/fetcher 경로는 경로 관측과 fixture 추가가 필요하다.

공통 후보 ID: RT-A-1-02 / NODE-003 (분석 DOC-C-040), RT-A-1-03 / NODE-004 (분석 DOC-C-041), RT-A-1-04 / NODE-005 (분석 DOC-C-042), RT-A-1-05 (분석 DOC-C-043), RT-A-1-06 (분석 DOC-C-044), RT-A-1-07 (분석 DOC-C-045), basic-consensus [명세 ID 미확인] (분석 TC-001), basic-peers [명세 ID 미확인] (분석 TC-002), basic-sync [명세 ID 미확인] (분석 TC-004), wemix-chain-up [명세 ID 미확인] (분석 TC-052), wemix-chain-up-15 [명세 ID 미확인] (분석 TC-053), RT-A-1-05 (분석 TC-061), chain-not-syncing [명세 ID 미확인] (분석 TC-062), stablenet-chain-up [명세 ID 미확인] (분석 TC-063), stablenet-chain-up-15 [명세 ID 미확인] (분석 TC-070), wbft-chain-up [명세 ID 미확인] (분석 TC-073), wbft-chain-up-15 [명세 ID 미확인] (분석 TC-074), RT-B-01 (분석 TC-178)

전체 원본 ID: RT-A-1-02 / NODE-003 (분석 DOC-C-040), RT-A-1-03 / NODE-004 (분석 DOC-C-041), RT-A-1-04 / NODE-005 (분석 DOC-C-042), RT-A-1-05 (분석 DOC-C-043), RT-A-1-06 (분석 DOC-C-044), RT-A-1-07 (분석 DOC-C-045), basic-consensus [명세 ID 미확인] (분석 TC-001), basic-peers [명세 ID 미확인] (분석 TC-002), basic-sync [명세 ID 미확인] (분석 TC-004), wemix-chain-up [명세 ID 미확인] (분석 TC-052), wemix-chain-up-15 [명세 ID 미확인] (분석 TC-053), RT-A-1-05 (분석 TC-061), chain-not-syncing [명세 ID 미확인] (분석 TC-062), stablenet-chain-up [명세 ID 미확인] (분석 TC-063), stablenet-chain-up-15 [명세 ID 미확인] (분석 TC-070), wbft-chain-up [명세 ID 미확인] (분석 TC-073), wbft-chain-up-15 [명세 ID 미확인] (분석 TC-074), RT-B-01 (분석 TC-178)

체인별 차이: CD-10

- PoA/etcd와 WBFT membership/quorum별 허용 장애수 설정; latest hash만으로 특정 downloader/fetcher 경로 검증했다고 보지 않음
- 고정 1초 차이를 공통 불변조건으로 두지 않고 합의/생산 profile에 맞는 timestamp 기대 범위로 분리한다.
- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## transaction-types

원본 8개 / 세 체인 공통 후보 8개.

세 체인 공통 type 0/1/2에 대해 송신자·수신자·nonce·receipt status를 검사한다. 입력 tip/feeCap과 fork 활성화는 profile로 제공한다.

공통 후보 ID: RT-A-2-01 / TX-006 (분석 DOC-C-001), RT-A-2-02 / TX-003 (분석 DOC-C-002), RT-A-2-03 / TX-007 (분석 DOC-C-003), RT-A-2-01 (분석 TC-028), RT-A-2-02 (분석 TC-029), RT-A-2-03 (분석 TC-030), RT-A-4-03 (분석 TC-043), sample-minimal-value-transfer [명세 ID 미확인] (분석 TC-082)

전체 원본 ID: RT-A-2-01 / TX-006 (분석 DOC-C-001), RT-A-2-02 / TX-003 (분석 DOC-C-002), RT-A-2-03 / TX-007 (분석 DOC-C-003), RT-A-2-01 (분석 TC-028), RT-A-2-02 (분석 TC-029), RT-A-2-03 (분석 TC-030), RT-A-4-03 (분석 TC-043), sample-minimal-value-transfer [명세 ID 미확인] (분석 TC-082)

체인별 차이: CD-02, CD-06, CD-07

- 원문 steps의 istanbul_getWbftExtraInfo GasTip 조회를 제거하고 체인별 수수료 공급 adapter로 대체한다. WEMIX3에서 원문 그대로 실행되지 않는다.
- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## txpool

원본 5개 / 세 체인 공통 후보 5개.

원문 status/content 형식 또는 실제 전달 검사를 구별한다. pending/queued·local/remote·최소 fee·pool 정책을 고정한다.

공통 후보 ID: RT-G-4-02 / RPC-018 (분석 DOC-C-036), RT-G-4-03 (분석 DOC-C-037), basic-txpool-propagation [명세 ID 미확인] (분석 TC-006), RT-G-4-02 (분석 TC-025), RT-G-4-03 (분석 TC-026)

전체 원본 ID: RT-G-4-02 / RPC-018 (분석 DOC-C-036), RT-G-4-03 (분석 DOC-C-037), basic-txpool-propagation [명세 ID 미확인] (분석 TC-006), RT-G-4-02 (분석 TC-025), RT-G-4-03 (분석 TC-026)

체인별 차이: CD-01, CD-08

- 원문 명세 DSL 구현 또는 체인별 policy/capability adapter
- 환경 구성·사전조건 검증

## wbft-specific

원본 38개 / 세 체인 공통 후보 0개.

WBFT header seal/Istanbul validator RPC의 원래 기대값은 WEMIX3 PoA에서 충족되지 않는다. 일반 블록 진행만 남긴 파생 테스트와 분리한다.

공통 후보 ID: 없음

전체 원본 ID: RT-B-01 / WBFT-002 (분석 DOC-D-002), RT-B-02 / WBFT-001 (분석 DOC-D-003), RT-B-03 / WBFT-005 (분석 DOC-D-004), RT-B-06 (분석 DOC-D-005), RT-B-09 / WBFT-003 (분석 DOC-D-006), RT-B-10 / WBFT-003 (분석 DOC-D-007), RT-B-06 / WBFT-006 (분석 DOC-D-008), RT-B-11 / WBFT-009 (분석 DOC-D-009), RT-B-12 / WBFT-009 (분석 DOC-D-010), WBFT-010 (분석 DOC-D-011), RT-B-08 (분석 DOC-D-012), WBFT-007 (분석 DOC-D-013), WBFT-008 (분석 DOC-D-014), WBFT-011 (분석 DOC-D-015), WBFT-012 (분석 DOC-D-016), WBFT-013 (분석 DOC-D-017), RT-G-3-01 / RPC-010 (분석 DOC-D-018), RT-G-3-02 / RPC-003 (분석 DOC-D-019), RT-G-3-03 / RPC-004 (분석 DOC-D-020), RT-G-3-04 / RPC-005 (분석 DOC-D-021), RT-G-3-05 / RPC-006 (분석 DOC-D-022), RT-G-3-06 / RPC-011 (분석 DOC-D-023), e1-mixed-producers [명세 ID 미확인] (분석 TC-075), RT-G-3-01 (분석 TC-136), RT-G-3-02 (분석 TC-137), RT-B-07 [부분 대응] (분석 TC-138), RT-G-3-03 (분석 TC-139), RT-G-3-04 (분석 TC-140), RT-G-3-05 (분석 TC-141), RT-G-3-06 (분석 TC-142), RT-B-02 (분석 TC-179), RT-B-03 (분석 TC-180), RT-B-04 (분석 TC-181), RT-B-04 (분석 TC-182), RT-B-05 (분석 TC-183), RT-B-11 (분석 TC-184), WBFT-010 [부분 대응] (분석 TC-185), RT-B-06 [부분 대응] (분석 TC-186)

체인별 차이: CD-10

의존 필드 정밀 재검토: JSON의 정확한 key와 step 문맥으로 분류했다. schemaVersion/description은 제외하며, 일반 sendTx의 to는 수신 계정으로 기록한다. env.capabilities/requires/applicableChains는 별도 capability_dependencies에 두었다. source_fields와 field_bindings에 literal/binding/role/config-reference를 구분하고 실제 값은 생략했다. 빈 목록은 환경 상속 또는 사전 의존이 없다는 뜻이 아니다. 세부 행은 all-tests.json을 참조한다.
