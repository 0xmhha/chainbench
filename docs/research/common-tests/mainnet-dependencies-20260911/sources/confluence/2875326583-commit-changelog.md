---
id: 2875326583
title: "Commit ChangeLog"
url: https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2875326583
version: 11
createdAt: 2026-08-05T04:53:41.024Z
fetched_at: 2026-09-11T07:35:00Z
---

## Commit Change History

| No. | Baseline Commit | Latest Commit | Updated At | Summary |
| --- | --- | --- | --- | --- |
| 1 |  | `940e9f2` |  | 최초 변경사항 |
| 2 | `c37994e` | `54a5cbd` |  | 추가 변경사항 |

## 2차 변경사항 

- Commit Range : `c37994e`\~`54a5cbd` (go-stablenet)

런타임 테스트 가능으로 표시된 항목:

1. fix(txpool): restore cumulative affordability enforcement with fee-delegation accounting (#116, `54a5cbd`) — sender/fee-payer별 누적 잔액 검증 분리 — 가능
2. fix: fix fee-delegated transaction signing in wallet and transaction API (#114, `4444af0`) — KeyStore·scwallet에 FeeDelegateDynamicFeeTx 처리, fee-payer 타입 불일치 가드 — 가능
3. fix: pre-allocate AccessList slice in SetSenderTx before copying (#115, `f4c9490`) — AccessList 손실로 서명 복원 실패 수정 — 가능
4. fix: pre-initialize PrevPrepared/PrevCommitted on epoch transition (#91, `74f9603`) — epoch 전환 시 sealer 활동 API에서 검증자가 누락되던 문제 수정 — 가능
5. fix: remove DecodeVanityData and return vanityData as raw hex (#88, `57d52f7`) — vanity 디코딩 panic 수정 — 가능
6. fix: harden istanbul\_status RPC against resource exhaustion and data integrity issues (#86, `d7cff3d`) — 블록 범위 상한, epoch 캐싱, 음수 블록 번호 언더플로우 수정 — 가능

런타임 테스트 불가/제외 항목: #113(nil GasTip 거부, 악의적 노드 필요), #112·#111·#106·#107·#105·#104·#102·#100·#101·#99·#98·#97·#96·#95·#94·#93·#92(업스트림 패치), #110(v1.1.0·Testnet Boho 블록 14,408,500 동기화), #108(README), #103(feePayer 블록 실행 검증, 악의적 노드 필요), #89(backlog flooding), #90(.gitignore), #85·#84(WBFT justification), #83(params-only alloc Balance zero, CLI 검증 필요), #82(newRoundChangeTimer race), #79(master-ci).

(원문은 34개 항목 전체의 요약과 링크를 담고 있다. lossyConversion=true, lostFeatures=[time, ac:inline-comment-marker, span])
