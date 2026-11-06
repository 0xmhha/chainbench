# 열린 작업 (Web UI 밖)

> **[정본]** Web UI 밖의 열린 작업은 이 목록 하나에 적는다. 끝난 항목은 이 목록에서 지운다.
> Web UI 의 남은 일은 [web-ui-status.md](web-ui-status.md) 에 있다.

아래 상태는 작업 트래커 두 개가 마지막으로 적은 내용을 옮긴 것이다. 이 목록을 만들면서 다시 재지는 않았다.
지난 경과가 필요하면 `git show 2b064b2e:docs/dev/chainbench-worklist.md` 와
`git show 2b064b2e:docs/dev/architecture/mainnet-config-worklist.md` 를 본다.

## 1. 메인넷 설정 live 잔여

- **L1 전체 회귀 — 진행.** 마지막 전체 실행은 2026-10-21 소스 `42c02609` 의 docker 15대 362회다. 350 통과, 2 실패, 10 BLOCKED, 미실행 0이고, 비통과 12건은 아래 항목들로 모두 설명된다.
- **L1 입력.** 고정 커밋은 StableNet `4d17eea0`·`ad0122af`·`bf17c960`, WBFT `d535dd3d`, WEMIX `4538a9f5` 다. 소스가 바뀐 뒤의 결과는 이전 실행과 합산하지 않는다. 실행 도구는 `scripts/tcsweep.sh` 다.
- **L1 실행 전제.** 바이너리는 `GSTABLE_BIN`, `GWBFT_BIN` 또는 `WBFT_BIN`(실물 이름은 `gwemix`), `WEMIX_BIN` 으로, 핸드오프 템플릿은 `GOWEMIX_TEMPLATE` 로 넘긴다. 워크스페이스 경로는 짧게 둔다. 길면 유닉스 소켓 길이 한도로 노드가 `bind: invalid argument` 로 죽는다.
- **테스트넷 송금 2건 — 열림, 사용자가 직접 처리한다.** `05` 는 payer 키 파일이 없고, `09` 는 송신자와 수신자 값이 비어 있다. 값이 없으면 실패하도록 만든 케이스다. 조회 7건은 통과했다. 실제 외부 테스트넷 대상 06~09 인수도 남았다([tests/tc/go-stablenet/testnet](../../tests/tc/go-stablenet/testnet/README.md)).
- **mixed 5건 — 범위 밖.** `wbft-stablenet` 소스가 없어 사용자 결정으로 뺐다. 다음 테스트 때 함께 본다.
- **L1 간헐 2건 — 열림.** `ethereum/08` 은 정리 단계에서 node4 정지가 시간 제한을 넘겼고, `anzeon/08` 은 블록 24 뒤 BP 연결이 끊겨 합의가 약 80초 멈췄다. 둘 다 재실행 2회가 통과했다. 정지가 늦는 이유는 아직 모르고, 연결 끊김은 [인계 10절](chain-handover-2026-09-12.md)과 같은 원인인지 확인하지 못했다.
- **R2 외부 엔드포인트 — 읽기 프로필 완료.** 남은 연계는 아래 R4 다([endpoint-profiles.md](endpoint-profiles.md)).
- **R3 능력 실측 — 진행.** 합의, PUSH0, P256, RPC 접근, WS 구독, 시스템 컨트랙트 코드 확인은 끝났다. 계약 ABI, 버전, 권한, 포크와 다른 RPC, 거래·VM, 계정 상태, 선언 필터 충분성이 남았다.
- **R4 로컬 서명 전송 — 진행.** 자산, 목록 편집, 거래 입력은 부분 완료다. 다른 포크·제어 조합과 전수 회귀가 남았다.
- **R5 거부 판정 — 진행.** 제출 분류는 부분 완료다. 다른 R5 조합과 출처 충분성이 남았다.
- **R6 다중 preset 실행 — CLI·MCP 완료.** Web 다중 대상 기동 폼과 전체 공통 TC 출처 충분성은 이 완료에 들어가지 않는다. 결과에 프로그램 해시, 버전, 하드포크 상태, 서명 방식, 건너뛴 이유를 남기는 일도 R6 밖이다([공통 TC 별도 구현 항목](../tc/common/03-separate-implementation-items.md) §2).
- **노드 간 읽기 경합 — 후보 106지점 남음.** 전수 감사 123지점 56케이스 가운데 실제로 깨진 10건 17곳만 고쳤다. 나머지는 결함 목록이 아니라 후보다. 거버넌스 케이스가 간헐적으로 깨지면 "변경을 확정한 노드와 그것을 쓰는 노드가 다른데 사이에 대기가 없다" 는 규칙으로 다시 찾고, 읽는 노드에서 `eth_getTransactionReceipt` 를 기다리게 고친다.

## 2. 수직 슬라이스 잔여

- **재평가 — 진행.** 트래커의 마지막 상태다. 원문 전체, 공개 DSL, legacy 소비자 재평가가 남았다.
- **업그레이드 T5.2 — 표시가 어긋난다.** 우선순위 표는 미완으로, 같은 문서의 태스크 표는 완료로 적는다. 완료 근거는 wemix+wbft 두 바이너리 기동과 골든 1+4, docker 15+15 의 `handoff confirmed` 다. 닫을지는 위 재평가에서 정한다.
- **wemix4 이관 T5.5 — 표시가 어긋난다.** 우선순위 표는 미완으로 적지만, 2026-09-12 확인은 47건 전부 이관으로 적는다. 원문 87개는 범위 확인 86개, 병합 1개이고 전체 인수는 아니다([검토 기록](wemix4-tc-sufficiency-review.json)).
- **legacy 제거 — 열림.** 소비자 이관 뒤에 한다. 전체 시나리오 동등성은 확인하지 못했고, 부분 포팅 21건은 현재 소비자와 다시 대조해야 한다([legacy-port-audit](legacy-port-audit/README.md)).
- **일반 계정 유효 가스 가격 — 사용자 결정 대기.** 원문 max 계산식과 현행 헤더 대체 계산이 다르다. 원문 기준을 고르기 전에는 공개 02 를 완료로 세지 않는다.

## 3. 외부 차단

chainbench 를 고쳐서는 해소되지 않는 결함이다. 전부 미해결이다. 체인팀 전달과 그 이슈 연결은 사용자가 직접 처리한다. 상세는 [체인팀 인계](chain-handover-2026-09-12.md)에 있다.

| 항목 | 결함 | 여기서 막히는 것 | 인계 절 |
|---|---|---|---|
| W1 | go-wemix `verifyBlockSig` nil 역참조로 노드가 죽는다 | 15+15 핸드오프의 mesh 준비. W3 와 같은 블록의 이웃한 줄이라 함께 고치는 것이 맞다 | 1 |
| W2 | go-wemix 부트 노드의 etcd 가 형성 뒤 사라진다 (원격 트랙의 R6) | poa 단독 원격 브링업 14 bp + 1 en | 2 |
| B1 | go-wbft `istanbul_getWbftExtraInfo` 가 `"latest"` 를 못 푼다 | DSL 의 `"@latest"` 로 우회 중이다 | 3 |
| W3 | go-wemix 거버넌스가 반만 지어진 구간에서 full sync 가 영구히 멈춘다 | `02-wemix-chain-up-15` 는 BLOCKED 로 남고, L1 의 CT-NODE-002 WEMIX 도 재발이다 | 3b |
| W4 | go-wemix ETH/LES fetcher 계량기 이름 충돌 | WEMIX CT-NODE-007. 고쳐지면 `TestNativeWemixW4FetcherCollision` 이 실패하고, 그때 `TestNativeSyncPathObservationsOnEveryChain` 에 이 조합을 되돌린다 | 6 |
| S1 | StableNet previous prepared seal 거절 | 원인 미확정이다. L1 부분 배치에서 두 번 관찰했다 | 7 |
| ProcessExtraSeal | StableNet nil 포인터 패닉 | Mac 에서 관찰했고 원인과 빈도는 미확정이다. Linux·메인넷 영향은 확인하지 못했다 | 8 |
| admin_wemixInfo | WEMIX EN 의 패닉 | EN 이 있는 WEMIX 망의 합의 확인 2건이 BLOCKED 다. BP 응답으로 우회하지 않는 것은 사용자 결정이다 | 9 |
| BP 연결 끊김 | WBFT 계열 BP 가 기동 직후 약 35초 서로 끊긴다 | 망 기동 직후의 격리 시험. CT-RPC-016 은 완전 연결을 기다리도록 고쳤다 | 10 |

## 4. 조건이 오면 보는 것

- **X5 Stablenet 기준선 실패 — 라이브 절반 열림.** 2026-09-13 기준선에서 `TestE2E_StablenetChain` 이 `recipient not credited: 0 -> 0` 으로 한 번 실패했다. 뒤의 재실행과 진단 실행은 통과했다. 복원한 DB 에는 수신 잔액이 있어 송금 누락의 증거는 아니다. 원인을 가르려면 실패 순간의 receipt 블록, 같은 노드의 latest head·상태, 다른 노드의 같은 높이 관측이 필요하다.
- **X14 `TestE2E_StablenetConsensusLifecycle` — 재발 시 분석.** 6회 중 2회 실패했고 문구를 잡지 못했다. 다음 실패의 `t.Fatalf` 문구가 예산(`waitAdvancing` 타임아웃), 체인 분기(`b-10`), 정족수 아래 생산(`b-08`)을 가른다.
- **Z1 모든 테스트를 DSL 로 — 별도 트랙.** 미착수다.
- **Z2 local·remote 두 곳 통과 — 미착수.** Z1 뒤에 한다.
- **Z3·C2 `V` 버전 표기 제거와 주석 전수 감사 — 미착수.** 표면이 두 번 바뀌지 않게 Z1, Z2 뒤 마지막에 한 번에 한다.
- **N1-b attach 의 `chain` 을 RPC 로 읽어 지운다 — 미착수.** 지금은 컨트랙트 표가 첫 호출 전에 있어야 해서 요구한다.
- **H4 2단계 주소 리터럴 허용 목록 줄이기 — 미착수.** 주소 리터럴이 0 이 됐으니 줄일 수 있다.
- **일부러 비워 둔 자리.** 네트워크 공통 건강 게이트의 `WantChainID`, stablenet 고아 핸들러 방향, `process/stop.go` 의 취소 의미는 요구가 생기면 연다.
