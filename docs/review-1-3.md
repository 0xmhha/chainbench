# 1-3 검토 자료: 운영 진입점에서 도달할 수 없는 함수 157개
기준: `main` b27d5c90 + 이 브랜치 커밋 2개(b4c34b5d, c1ddca69). 생성: 2026-09-26.
## 어떻게 뽑았나

1. 저장소의 모든 비테스트 `.go` 파일에서 함수·메서드 선언을 모았다(`go/packages` + `go/ast`, 2346개. `scripts/`·`docs/` 아래 별도 도구 제외).
2. 운영 진입점 세 곳(`cmd/chainbench`, `cmd/chainbench-mcp`, `cmd/chainbench-dashboard`)의 `main`과 모든 `init`에서 시작해, CHA 호출 그래프(`golang.org/x/tools`)로 닿는 함수를 모았다. 인터페이스 호출은 구현 전부로, 함수 값 호출은 같은 시그니처 전부로 넓게 잡는다 — 그래서 "닿지 않는다"는 판정은 보수적이다.
3. 1에서 2를 뺀 것이 아래 목록이다. 각 함수를 누가 부르는지는 `go/types`의 참조(Uses)로 셌다.

## 분류 기준

| 분류 | 뜻 | 판단할 때 볼 것 |
|---|---|---|
| 테스트만 참조 | 운영 코드는 부르지 않고 테스트만 부른다 | 테스트용 도우미를 운영 파일에 둔 것인지, 옛 API를 테스트만 붙들고 있는지 |
| 도달 불가 코드에서만 참조 | 운영 코드가 부르긴 하지만, 부르는 쪽도 이 목록에 있다 | 부르는 쪽과 한 묶음으로 판단 |
| 참조 없음 | 아무도 부르지 않는다 | 나중을 위한 것인지, 남은 찌꺼기인지 |

**한계.** e2e 빌드 태그 파일은 로드하지 않았다(참조 수에 없다). reflection 호출(`MethodByName`)은 저장소에 없다. "마지막 변경"은 선언 줄을 마지막으로 바꾼 커밋이다.

판단 칸: **사용**(운영에서 쓰고 있어야 한다 — 연결이 끊긴 것), **보류**(지금은 안 쓰지만 쓸 계획), **삭제**(앞으로도 안 쓴다).

## 요약

| 분류 | 개수 |
|---|---|
| 테스트만 참조 | 85 |
| 도달 불가 코드에서만 참조 | 39 |
| 참조 없음 | 33 |

## 패키지별 목록

### internal/accounts (17)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `EncodeCall` | `internal/accounts/abi.go:23` | 도달 불가 코드에서만 참조 | 2 | 3 (TestEncodeCall, TestEncodeCallTransferGolden) | 2026-07-26 3e67c3a1 feat(accounts): add ABI call helpers; port system-contract read cases (A4 start) (#50) | |
| `leftPad32` | `internal/accounts/abi.go:34` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-07-26 3e67c3a1 feat(accounts): add ABI call helpers; port system-contract read cases (A4 start) (#50) | |
| `Word` | `internal/accounts/abi_encode.go:34` | 참조 없음 | 0 | 0 | 2026-07-26 648254d4 feat(accounts): add a dynamic ABI encoder (bytes/string args) (#62) | |
| `ReadString` | `internal/accounts/decode.go:42` | 테스트만 참조 | 0 | 1 (TestReadString) | 2026-07-26 748cedd5 feat(accounts): decode dynamic string returns; port token-metadata case (#63) | |
| `WordToBig` | `internal/accounts/events.go:30` | 도달 불가 코드에서만 참조 | 2 | 2 (TestTopicToAddressAndWord, TestWordAt) | 2026-07-26 30be9bd8 feat(accounts): add event-log decoding; port a contract-event regression case (#52) | |
| `FindLog` | `internal/accounts/events.go:43` | 도달 불가 코드에서만 참조 | 1 | 2 (TestFindLog) | 2026-07-26 30be9bd8 feat(accounts): add event-log decoding; port a contract-event regression case (#52) | |
| `defaultProvider.Protocol` | `internal/accounts/provider.go:61` | 참조 없음 | 0 | 0 | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `defaultProvider.HasAccountExtra` | `internal/accounts/provider.go:63` | 참조 없음 | 0 | 0 | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `defaultProvider.AddressForKey` | `internal/accounts/provider.go:65` | 참조 없음 | 0 | 0 | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `ReadUint` | `internal/accounts/read.go:18` | 테스트만 참조 | 0 | 2 (TestReadUint) | 2026-07-26 87b97d0f feat(accounts): add ReadUint contract-read helper; use it in the anzeon cases (#56) | |
| `sdkWallet.Address` | `internal/accounts/wallet.go:99` | 참조 없음 | 0 | 0 | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `sdkWallet.SendLegacy` | `internal/accounts/wallet.go:158` | 테스트만 참조 | 0 | 2 (TestSendLegacy_Validation) | 2026-07-26 688a3541 feat(accounts): expose legacy (0x00) sending; port the legacy-tx regression case (#48) | |
| `sdkWallet.SendDynamicFee` | `internal/accounts/wallet.go:191` | 테스트만 참조 | 0 | 2 (TestSendDynamicFee_Validation) | 2026-07-27 878745e3 feat(accounts): add dynamic-fee (0x2) and access-list (0x1) tx types; port a2-02/03 (B) (#80) | |
| `sdkWallet.SendAccessList` | `internal/accounts/wallet.go:229` | 테스트만 참조 | 0 | 2 (TestSendAccessList_Validation) | 2026-07-27 878745e3 feat(accounts): add dynamic-fee (0x2) and access-list (0x1) tx types; port a2-02/03 (B) (#80) | |
| `sdkWallet.SendDynamicFeeGas` | `internal/accounts/wallet.go:263` | 테스트만 참조 | 0 | 2 (TestSendExplicitGas_Validation) | 2026-07-27 ba179352 Complete burn-refund lifecycle + anzeon fee-boundary cases (B-blk1) (#83) | |
| `sdkWallet.SendLegacyGas` | `internal/accounts/wallet.go:333` | 테스트만 참조 | 0 | 2 (TestSendExplicitGas_Validation) | 2026-07-27 ba179352 Complete burn-refund lifecycle + anzeon fee-boundary cases (B-blk1) (#83) | |
| `sdkWallet.SendAccessListGas` | `internal/accounts/wallet.go:360` | 테스트만 참조 | 0 | 1 (TestSendExplicitGas_Validation) | 2026-07-27 ba179352 Complete burn-refund lifecycle + anzeon fee-boundary cases (B-blk1) (#83) | |

### internal/arch (14)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `CommentClaims` | `internal/arch/comments.go:80` | 테스트만 참조 | 0 | 1 (TestCommentsDoNotContradictTheCode) | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `citationClaims` | `internal/arch/comments.go:158` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `namingClaims` | `internal/arch/comments.go:222` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `documented` | `internal/arch/comments.go:279` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `namesCode` | `internal/arch/comments.go:313` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `unclaimed` | `internal/arch/comments.go:331` | 도달 불가 코드에서만 참조 | 4 | 0 | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `resolveFile` | `internal/arch/comments.go:345` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `pathExists` | `internal/arch/comments.go:362` | 도달 불가 코드에서만 참조 | 3 | 0 | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `Decls` | `internal/arch/naming.go:44` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-09-07 5237ec8c fix(dsl): a node-signed transaction has to reach the node that holds the key (#356) | |
| `Collisions` | `internal/arch/naming.go:156` | 테스트만 참조 | 0 | 1 (TestNamesDoNotCollide) | 2026-09-07 5237ec8c fix(dsl): a node-signed transaction has to reach the node that holds the key (#356) | |
| `Collision.Explained` | `internal/arch/naming.go:187` | 테스트만 참조 | 0 | 1 (TestNamesDoNotCollide) | 2026-09-07 5237ec8c fix(dsl): a node-signed transaction has to reach the node that holds the key (#356) | |
| `Collision.within` | `internal/arch/naming.go:207` | 도달 불가 코드에서만 참조 | 4 | 0 | 2026-09-07 5237ec8c fix(dsl): a node-signed transaction has to reach the node that holds the key (#356) | |
| `Collision.appForwards` | `internal/arch/naming.go:230` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-09-07 5237ec8c fix(dsl): a node-signed transaction has to reach the node that holds the key (#356) | |
| `packagesIn` | `internal/arch/parse.go:73` | 테스트만 참조 | 0 | 2 (readPlacement, readWriters) | 2026-08-21 e1414395 feat(arch): enforce the layering and state-ownership rules as tests (A1, A2) (#246) | |

### internal/core/collector (12)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `*Bus.Dropped` | `internal/core/collector/bus.go:65` | 테스트만 참조 | 0 | 1 (TestBus_DropsWhenFull) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `*collector.WaitLog` | `internal/core/collector/collector_impl.go:307` | 참조 없음 | 0 | 0 | 2026-08-06 ba8f46e4 feat(collector): RPC chainstate sampler and WaitLog (M7, T3.3 partial) (#186) | |
| `scanLog` | `internal/core/collector/collector_impl.go:329` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-08-06 ba8f46e4 feat(collector): RPC chainstate sampler and WaitLog (M7, T3.3 partial) (#186) | |
| `IsInputError` | `internal/core/collector/detect_errors.go:18` | 테스트만 참조 | 0 | 4 (TestDetect_MissingURL, TestDetect_RejectsNonHTTP, TestDetect_UnknownOverride) | 2026-07-26 f79a4228 feat(probe): absorb chain detection into the core, registry-driven (#31) | |
| `NewFileStore` | `internal/core/collector/filestore.go:24` | 테스트만 참조 | 0 | 3 (TestFileStore_MissingFileIsEmpty, TestFileStore_PersistsAcrossInstances) | 2026-07-24 4d932018 feat(obs): persist run results + report and state-based verify (#17) | |
| `*FileStore.SaveRun` | `internal/core/collector/filestore.go:46` | 테스트만 참조 | 0 | 2 (TestFileStore_PersistsAcrossInstances) | 2026-07-24 4d932018 feat(obs): persist run results + report and state-based verify (#17) | |
| `*FileStore.GetRun` | `internal/core/collector/filestore.go:54` | 테스트만 참조 | 0 | 1 (TestFileStore_PersistsAcrossInstances) | 2026-07-24 4d932018 feat(obs): persist run results + report and state-based verify (#17) | |
| `*FileStore.flush` | `internal/core/collector/filestore.go:68` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-07-24 4d932018 feat(obs): persist run results + report and state-based verify (#17) | |
| `NewLogger` | `internal/core/collector/logger.go:12` | 테스트만 참조 | 0 | 1 (TestLogEvent_JSON) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `LogEvent` | `internal/core/collector/logger.go:19` | 테스트만 참조 | 0 | 1 (TestLogEvent_JSON) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `*MemStore.SaveRun` | `internal/core/collector/store.go:59` | 도달 불가 코드에서만 참조 | 2 | 4 (TestMemStore_SaveGetList, TestRunsAPI) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `*MemStore.GetRun` | `internal/core/collector/store.go:67` | 도달 불가 코드에서만 참조 | 1 | 3 (TestMemStore_SaveGetList) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |

### internal/core/nodeconfig (12)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `*Args.SetIfSupported` | `internal/core/nodeconfig/args.go:54` | 참조 없음 | 0 | 0 | 2026-09-21 7cd42aac refactor: design v3 — measure the tree, make each word mean one thing, and let every verb say what it needs (#422) | |
| `*Args.Value` | `internal/core/nodeconfig/args.go:101` | 참조 없음 | 0 | 0 | 2026-09-21 7cd42aac refactor: design v3 — measure the tree, make each word mean one thing, and let every verb say what it needs (#422) | |
| `String` | `internal/core/nodeconfig/builder.go:93` | 참조 없음 | 0 | 0 | 2026-08-12 10fc830c feat(launchopt): single launch-argv assembly with dialects, modules, and customization seams (#235) | |
| `Defaults` | `internal/core/nodeconfig/config.go:25` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Merge` | `internal/core/nodeconfig/config.go:52` | 도달 불가 코드에서만 참조 | 1 | 1 (TestMerge_DoesNotMutateInputs) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Resolve` | `internal/core/nodeconfig/config.go:64` | 테스트만 참조 | 0 | 1 (TestResolve_Precedence) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Flatten` | `internal/core/nodeconfig/config.go:72` | 테스트만 참조 | 0 | 1 (TestFlatten) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `flattenInto` | `internal/core/nodeconfig/config.go:78` | 도달 불가 코드에서만 참조 | 2 | 0 | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Values.String` | `internal/core/nodeconfig/config.go:112` | 테스트만 참조 | 0 | 6 (TestFlatten, TestResolve_Precedence) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Values.Int` | `internal/core/nodeconfig/config.go:120` | 테스트만 참조 | 0 | 10 (TestFlatten, TestResolve_Precedence, TestTypedGetters_Fallback) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Values.Bool` | `internal/core/nodeconfig/config.go:131` | 테스트만 참조 | 0 | 2 (TestFlatten, TestTypedGetters_Fallback) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Values.Duration` | `internal/core/nodeconfig/config.go:145` | 참조 없음 | 0 | 0 | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |

### internal/feature (10)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `fieldsOf` | `internal/feature/bind.go:58` | 도달 불가 코드에서만 참조 | 2 | 0 | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |
| `fieldsOfType` | `internal/feature/bind.go:71` | 도달 불가 코드에서만 참조 | 2 | 0 | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |
| `Flags` | `internal/feature/bind.go:114` | 테스트만 참조 | 0 | 6 (TestComposeFeatures_TagsMatchTheCommands, TestFlags_FillTheInput, TestFlags_RefusesAShapeNoSurfaceCanCarry) | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |
| `field.intDefault` | `internal/feature/bind.go:158` | 도달 불가 코드에서만 참조 | 2 | 0 | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |
| `Schema` | `internal/feature/bind.go:175` | 테스트만 참조 | 0 | 3 (TestSchema_CarriesReadOnly, TestTags_MakeBothBindings) | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |
| `jsonType` | `internal/feature/bind.go:210` | 도달 불가 코드에서만 참조 | 2 | 0 | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |
| `Register` | `internal/feature/feature.go:80` | 도달 불가 코드에서만 참조 | 25 | 2 (TestInvoke_RefusesTheWrongInput, TestRegister_KeepsTheTypes) | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |
| `Lookup` | `internal/feature/feature.go:101` | 테스트만 참조 | 0 | 3 (TestInvoke_RefusesTheWrongInput, TestRegister_KeepsTheTypes, TestReportFeatures_TagsMatchTheCommands) | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |
| `Registered` | `internal/feature/feature.go:109` | 도달 불가 코드에서만 참조 | 1 | 2 (TestComposeFeatures_TagsMatchTheCommands, TestRegistry_CoversMoreOfAppEachTime) | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |
| `Queries` | `internal/feature/feature.go:123` | 테스트만 참조 | 0 | 2 (TestRegister_KeepsTheTypes) | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |

### internal/resource (10)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `*Inventory.Pool` | `internal/resource/inventory.go:68` | 참조 없음 | 0 | 0 | 2026-08-28 8859f4c2 feat(resource): the set knows its capacity and who holds it (P1.4, P1.5) (#304) | |
| `*Inventory.Release` | `internal/resource/inventory.go:159` | 테스트만 참조 | 0 | 2 (TestInventory_ReleaseIsRemovalNotStop) | 2026-08-28 8859f4c2 feat(resource): the set knows its capacity and who holds it (P1.4, P1.5) (#304) | |
| `Usage.Full` | `internal/resource/inventory.go:206` | 테스트만 참조 | 0 | 1 (TestInventory_FullNamesTheHolders) | 2026-08-28 8859f4c2 feat(resource): the set knows its capacity and who holds it (P1.4, P1.5) (#304) | |
| `Spec.Resolve` | `internal/resource/machine.go:356` | 테스트만 참조 | 0 | 3 (TestResolve_ServerNeedsAnInventory, TestTargetResolve) | 2026-08-25 9749b70f refactor(machine): rename core/target and ratchet the no-branching rule (V1) (#282) | |
| `Spec.ResolveWith` | `internal/resource/machine.go:367` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-08-25 9749b70f refactor(machine): rename core/target and ratchet the no-branching rule (V1) (#282) | |
| `Spec.ResolveWithMap` | `internal/resource/machine.go:375` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-08-25 9749b70f refactor(machine): rename core/target and ratchet the no-branching rule (V1) (#282) | |
| `Assign` | `internal/resource/pool.go:106` | 테스트만 참조 | 0 | 12 (TestAssign_ConsumesHostsBeforeSlots, TestAssign_IsDeterministic, TestAssign_KeepsAGivenLabel) | 2026-08-28 bc64b26c refactor: P1 — the node vocabulary and the resource module find their owners (#301) | |
| `Plan` | `internal/resource/ports.go:43` | 테스트만 참조 | 0 | 4 (TestPortReservation_IsHonest, TestPortReservation_LeavesRoomForBothEtcdPorts) | 2026-08-28 bc64b26c refactor: P1 — the node vocabulary and the resource module find their owners (#301) | |
| `*Set.Path` | `internal/resource/serverset.go:229` | 참조 없음 | 0 | 0 | 2026-08-28 bc64b26c refactor: P1 — the node vocabulary and the resource module find their owners (#301) | |
| `Ports.HasMetrics` | `internal/resource/serverset_lookup.go:13` | 테스트만 참조 | 0 | 2 (TestPorts_MetricsNeedsRoomInTheStep) | 2026-09-20 1c662e27 refactor: files named for one job, and lowerCase down from 235 lines (#420) | |

### internal/core/statemachine (9)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `*Machine.States` | `internal/core/statemachine/contract.go:81` | 참조 없음 | 0 | 0 | 2026-09-25 5a206871 refactor: state machines named by design-v3 02, with contracts and no silent stalls (#426) | |
| `*Machine.Audit` | `internal/core/statemachine/contract.go:95` | 테스트만 참조 | 0 | 4 (TestContract_AuditFindsAnEmissionNobodyTakes, TestContractsAreConsistent, TestRunContractsAreConsistent) | 2026-09-25 5a206871 refactor: state machines named by design-v3 02, with contracts and no silent stalls (#426) | |
| `*Machine.takenOnPath` | `internal/core/statemachine/contract.go:118` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-09-25 5a206871 refactor: state machines named by design-v3 02, with contracts and no silent stalls (#426) | |
| `*Machine.ContractTable` | `internal/core/statemachine/contract.go:200` | 테스트만 참조 | 0 | 2 (TestContractTableMatchesTheDesign, TestRunContractTableMatchesTheDesign) | 2026-09-25 5a206871 refactor: state machines named by design-v3 02, with contracts and no silent stalls (#426) | |
| `*ring.all` | `internal/core/statemachine/logrec.go:47` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-09-24 cde3a08f refactor: the composition walks a hierarchical state machine (#425) | |
| `*Machine.Controller` | `internal/core/statemachine/machine.go:78` | 테스트만 참조 | 0 | 2 (TestPost_FromAChildReachesTheController) | 2026-09-24 cde3a08f refactor: the composition walks a hierarchical state machine (#425) | |
| `*Machine.Post` | `internal/core/statemachine/machine.go:186` | 테스트만 참조 | 0 | 2 (TestPost_FromAChildReachesTheController, TestPost_OnlyEnqueuesAndIsDrainedAtTheEndOfTheNextSend) | 2026-09-24 cde3a08f refactor: the composition walks a hierarchical state machine (#425) | |
| `*Machine.Tree` | `internal/core/statemachine/machine.go:279` | 도달 불가 코드에서만 참조 | 1 | 1 (TestTreeAndPath) | 2026-09-24 cde3a08f refactor: the composition walks a hierarchical state machine (#425) | |
| `*Machine.Dump` | `internal/core/statemachine/machine.go:294` | 테스트만 참조 | 0 | 3 (TestLog_KeepsTheLastMessagesAndOverwritesTheOldest, entered, lastRec) | 2026-09-24 cde3a08f refactor: the composition walks a hierarchical state machine (#425) | |

### internal/core/keyring/store (8)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `RawFileBackend.Load` | `internal/core/keyring/store/backend.go:51` | 참조 없음 | 0 | 0 | 2026-08-27 ddc8c7c4 refactor(keyring): the store speaks the model's names, not aliases of them (#296) | |
| `KeystoreBackend.Load` | `internal/core/keyring/store/backend.go:93` | 참조 없음 | 0 | 0 | 2026-08-27 ddc8c7c4 refactor(keyring): the store speaks the model's names, not aliases of them (#296) | |
| `DeclaredKeys.Dir` | `internal/core/keyring/store/declared.go:54` | 참조 없음 | 0 | 0 | 2026-09-07 c8578e87 feat(blueprint): one document that says what a network is, from declaration to live chain (N2-N6) (#357) | |
| `Extend` | `internal/core/keyring/store/generate.go:123` | 테스트만 참조 | 0 | 4 (TestExtend_PlainSetStaysPlain, TestExtend_PromotingIntoABLSSetKeepsItLoadable, TestExtend_RejectsMoreValidatorsThanIdentities) | 2026-09-21 7cd42aac refactor: design v3 — measure the tree, make each word mean one thing, and let every verb say what it needs (#422) | |
| `*KeySet.Dir` | `internal/core/keyring/store/keyset.go:61` | 참조 없음 | 0 | 0 | 2026-08-26 c21b3acf refactor(keyring): four layers, honest names, and no netmap coupling (#295) | |
| `*KeySet.Install` | `internal/core/keyring/store/keyset.go:141` | 테스트만 참조 | 0 | 2 (TestKeySet_Install) | 2026-08-27 ddc8c7c4 refactor(keyring): the store speaks the model's names, not aliases of them (#296) | |
| `PresetKeys.Dir` | `internal/core/keyring/store/source.go:52` | 테스트만 참조 | 0 | 2 (TestPresetKeys_LoadsAndChecksCapacity) | 2026-08-28 ed8c2e6d refactor(chainsetup): the engine builds, the store sources keys, poa bootstraps (P6.1) (#313) | |
| `GeneratedKeys.Dir` | `internal/core/keyring/store/source.go:99` | 테스트만 참조 | 0 | 2 (TestGeneratedKeys_ReusesAnExistingSet) | 2026-08-28 ed8c2e6d refactor(chainsetup): the engine builds, the store sources keys, poa bootstraps (P6.1) (#313) | |

### internal/core/node (8)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `Layout.NodekeyPath` | `internal/core/node/layout.go:134` | 테스트만 참조 | 0 | 1 (TestLayout_CompositionIsolation) | 2026-08-28 32554bf7 refactor(node): one fact record per node, and paths from one layout (P2) (#303) | |
| `Layout.KeystoreDir` | `internal/core/node/layout.go:139` | 참조 없음 | 0 | 0 | 2026-08-28 32554bf7 refactor(node): one fact record per node, and paths from one layout (P2) (#303) | |
| `Layout.StaticNodesPath` | `internal/core/node/layout.go:144` | 참조 없음 | 0 | 0 | 2026-08-28 32554bf7 refactor(node): one fact record per node, and paths from one layout (P2) (#303) | |
| `Layout.IPCPath` | `internal/core/node/layout.go:151` | 참조 없음 | 0 | 0 | 2026-08-28 32554bf7 refactor(node): one fact record per node, and paths from one layout (P2) (#303) | |
| `NodeSet.HasCapability` | `internal/core/node/node.go:158` | 테스트만 참조 | 0 | 3 (TestHasCapability) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Offset` | `internal/core/node/node.go:170` | 테스트만 참조 | 0 | 3 (TestOffset) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `*Map.Labels` | `internal/core/node/placement.go:84` | 테스트만 참조 | 0 | 2 (TestAssign_KeepsAGivenLabel) | 2026-08-28 bc64b26c refactor: P1 — the node vocabulary and the resource module find their owners (#301) | |
| `Topology.Counts` | `internal/core/node/topology.go:155` | 테스트만 참조 | 0 | 1 (TestLoad_RolesSyncBootnode) | 2026-07-31 4d377c34 feat(topology): config-file-driven per-node local network layout (#158) | |

### internal/core/session (8)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `*env.Dir` | `internal/core/session/environment_impl.go:26` | 참조 없음 | 0 | 0 | 2026-08-06 0970c4a1 feat(session): implement artifact store, env reuse, records (M1) (#178) | |
| `*env.Fingerprint` | `internal/core/session/environment_impl.go:27` | 참조 없음 | 0 | 0 | 2026-08-06 0970c4a1 feat(session): implement artifact store, env reuse, records (M1) (#178) | |
| `*env.DataPath` | `internal/core/session/environment_impl.go:28` | 참조 없음 | 0 | 0 | 2026-08-06 0970c4a1 feat(session): implement artifact store, env reuse, records (M1) (#178) | |
| `Composition.Lock` | `internal/core/session/lock.go:64` | 도달 불가 코드에서만 참조 | 1 | 8 (TestLock_ADeadRunsLockIsStaleAndTakenOver, TestLock_AnotherHostsLockIsNotJudged, TestLock_FreeThenHeldThenReleased) | 2026-08-25 cfcb4d97 feat: know what is running — interrupt, workspace lock, port occupancy (A1-A3) (#267) | |
| `IDFor` | `internal/core/session/read.go:123` | 테스트만 참조 | 0 | 2 (TestList_And_ChainstatePaths, writeSession) | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `*record.Dir` | `internal/core/session/record_impl.go:39` | 참조 없음 | 0 | 0 | 2026-08-06 0970c4a1 feat(session): implement artifact store, env reuse, records (M1) (#178) | |
| `*sess.ID` | `internal/core/session/session_impl.go:150` | 참조 없음 | 0 | 0 | 2026-08-26 c21b3acf refactor(keyring): four layers, honest names, and no netmap coupling (#295) | |
| `*sess.Keys` | `internal/core/session/session_impl.go:152` | 참조 없음 | 0 | 0 | 2026-08-26 c21b3acf refactor(keyring): four layers, honest names, and no netmap coupling (#295) | |

### internal/consensus/poa (6)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `Info.Bootstrapped` | `internal/consensus/poa/info.go:42` | 테스트만 참조 | 0 | 3 (TestReadInfo_EmptyClusterIsNotBootstrapped, TestReadInfo_ParsesTheConsoleString, TestReadInfo_ToleratesTheMemberObjectShape) | 2026-08-28 ed8c2e6d refactor(chainsetup): the engine builds, the store sources keys, poa bootstraps (P6.1) (#313) | |
| `Info.Cluster` | `internal/consensus/poa/info.go:47` | 도달 불가 코드에서만 참조 | 2 | 1 (TestWaitEtcdCluster_ToleratesATransientFirstRead) | 2026-08-28 ed8c2e6d refactor(chainsetup): the engine builds, the store sources keys, poa bootstraps (P6.1) (#313) | |
| `WaitEtcdCluster` | `internal/consensus/poa/info.go:78` | 테스트만 참조 | 0 | 2 (TestWaitEtcdCluster_ReportsTheStateItSaw, TestWaitEtcdCluster_ToleratesATransientFirstRead) | 2026-08-28 ed8c2e6d refactor(chainsetup): the engine builds, the store sources keys, poa bootstraps (P6.1) (#313) | |
| `Family.RPCNamespace` | `internal/consensus/poa/poa.go:34` | 테스트만 참조 | 0 | 2 (TestFamily_StaticFacts) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Family.ValidatorsMethod` | `internal/consensus/poa/poa.go:35` | 테스트만 참조 | 0 | 2 (TestFamily_StaticFacts) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Config.JSON` | `internal/consensus/poa/wemixconfig.go:63` | 테스트만 참조 | 0 | 1 (TestConfig_JSONAndValidate) | 2026-07-26 44451751 Concurrent wemix→wbft hardfork handoff framework + live E2E (#27) | |

### internal/core/process (6)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `*Ledger.History` | `internal/core/process/ledger.go:130` | 테스트만 참조 | 0 | 4 (TestHardforkExecute_SupersedesTheLedgerKeepingThePriorRevision, TestLedgerSupersede_PreservesPriorAsRevision, TestLedgerSupersede_SurvivesReopen) | 2026-09-03 74d7048d feat: chain-test execution roadmap E0A through E9 (#339) | |
| `*Ledger.FindBinary` | `internal/core/process/ledger.go:174` | 테스트만 참조 | 0 | 2 (TestLedger_RecordsQueriesAndSurvivesReopen) | 2026-08-25 96a1a93d refactor(process): the run ledger owns what runs where (V4) (#285) | |
| `StopNodeSet` | `internal/core/process/lifecycle.go:29` | 테스트만 참조 | 0 | 2 (TestStopNodeSet_IsBestEffort, TestStopNodeSet_SkipsNodesWithNoPID) | 2026-09-01 f7d7b8a5 refactor(process,resource): merge the execution and machine concerns (R3) (#327) | |
| `NewLocalDriverWithExec` | `internal/core/process/local.go:43` | 테스트만 참조 | 0 | 1 (TestLocalDriver_LaunchWritesLog) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `*LocalDriver.Provision` | `internal/core/process/local.go:59` | 테스트만 참조 | 0 | 1 (TestLocalDriver_ProvisionWritesConfig) | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `*RemoteDriver.Provision` | `internal/core/process/remote.go:80` | 테스트만 참조 | 0 | 2 (TestRemoteDriver_NonZeroExitIsError, TestRemoteDriver_ProvisionLaunchStop) | 2026-07-26 08c677fc feat(driver): add a RemoteDriver that manages nodes over SSH (#55) | |

### internal/chainsetup (5)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `*Manager.Tree` | `internal/chainsetup/manager.go:503` | 테스트만 참조 | 0 | 1 (TestManagerTreeIsTheTreeTheDesignDrew) | 2026-09-24 cde3a08f refactor: the composition walks a hierarchical state machine (#425) | |
| `*Workspace.Preflight` | `internal/chainsetup/occupancy.go:32` | 참조 없음 | 0 | 0 | 2026-09-20 1c662e27 refactor: files named for one job, and lowerCase down from 235 lines (#420) | |
| `*Workspace.Keys` | `internal/chainsetup/steps_keys.go:196` | 테스트만 참조 | 0 | 9 (TestConfig_PinnedFileIsUsedVerbatim, TestConfig_PinnedFileMissingIsReported, TestConfig_PortableRefResolvesUnderWorkspaceConfig) | 2026-09-24 cde3a08f refactor: the composition walks a hierarchical state machine (#425) | |
| `*Workspace.Lock` | `internal/chainsetup/workspace.go:109` | 참조 없음 | 0 | 0 | 2026-08-25 cfcb4d97 feat: know what is running — interrupt, workspace lock, port occupancy (A1-A3) (#267) | |
| `InWorkspace` | `internal/chainsetup/workspace_open.go:28` | 도달 불가 코드에서만 참조 | 20 | 1 (TestRollBackLaunch_LeavesWhatWasAlreadyRunning) | 2026-09-21 7cd42aac refactor: design v3 — measure the tree, make each word mean one thing, and let every verb say what it needs (#422) | |

### internal/core/remote (4)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `ValidateAuth` | `internal/core/remote/auth.go:60` | 테스트만 참조 | 0 | 2 (TestValidateAuth) | 2026-07-26 45e4b737 feat(remote): absorb remote RPC auth and SSH tunnel into the core (#30) | |
| `ReadFile` | `internal/core/remote/files.go:30` | 참조 없음 | 0 | 0 | 2026-08-21 4578c661 refactor(keyring): one package and one command group for key material (#244) | |
| `DialTunnelClient` | `internal/core/remote/ssh.go:64` | 테스트만 참조 | 0 | 6 (TestDialSSH_Validation, TestDialTunnelClient_BadPasswordNoLeak, TestDialTunnelClient_HostKeyMismatch) | 2026-07-26 45e4b737 feat(remote): absorb remote RPC auth and SSH tunnel into the core (#30) | |
| `tunnelTransport` | `internal/core/remote/ssh.go:75` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-07-26 45e4b737 feat(remote): absorb remote RPC auth and SSH tunnel into the core (#30) | |

### internal/app (3)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `TxReceipt.Succeeded` | `internal/app/chainops.go:118` | 테스트만 참조 | 0 | 1 (TestTxWait_ReturnsTheReceiptOnceMined) | 2026-09-06 b8af4b44 refactor: every CLI and MCP registration reaches its feature through app (U2-U8) (#352) | |
| `RunSuitesOut.Failed` | `internal/app/runsuites.go:77` | 테스트만 참조 | 0 | 2 (TestRunSuitesOut_TotalsSeparatesTheThreeOutcomes) | 2026-09-10 9b6b0930 fix: resolve the 14 open monitoring issues, with live docker verification (#371) | |
| `DeclaredAttach` | `internal/app/workflow.go:176` | 테스트만 참조 | 0 | 5 (TestDeclaredAttach_ComposingSpecsDeclareNothing, TestDeclaredAttach_OneRunIsOneNetwork) | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |

### internal/core/health (3)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `Participation.Formed` | `internal/core/health/participation.go:52` | 테스트만 참조 | 0 | 4 (TestParticipants_AHealthyNetworkIsFormed, TestParticipants_FindsTheValidatorThatNeverSeals, TestParticipants_SaysNothingWhenTheChainNamesNoSealer) | 2026-09-07 4d6d370a fix(health): wait for a network to be formed, not merely for its head to move (#358) | |
| `Participation.Describe` | `internal/core/health/participation.go:55` | 테스트만 참조 | 0 | 6 (TestParticipants_AHealthyNetworkIsFormed, TestParticipants_FindsTheValidatorThatNeverSeals, TestParticipants_SaysNothingWhenTheChainNamesNoSealer) | 2026-09-07 4d6d370a fix(health): wait for a network to be formed, not merely for its head to move (#358) | |
| `Participants` | `internal/core/health/participation.go:73` | 테스트만 참조 | 0 | 5 (TestParticipants_AHealthyNetworkIsFormed, TestParticipants_AnUnreadableBlockShrinksTheWindow, TestParticipants_FindsTheValidatorThatNeverSeals) | 2026-09-07 4d6d370a fix(health): wait for a network to be formed, not merely for its head to move (#358) | |

### internal/core/origin (3)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `Origin.Rank` | `internal/core/origin/origin.go:74` | 도달 불가 코드에서만 참조 | 1 | 4 (TestLadderIsTheLineTheDesignFixed, TestRank) | 2026-09-22 a5386b0e docs(design-v3): one declaration, one vocabulary, one priority line (#424) | |
| `Origin.Known` | `internal/core/origin/origin.go:84` | 테스트만 참조 | 0 | 1 (TestRank) | 2026-09-22 a5386b0e docs(design-v3): one declaration, one vocabulary, one priority line (#424) | |
| `Ladder` | `internal/core/origin/origin.go:88` | 테스트만 참조 | 0 | 3 (TestLadderIsNotSharedState, TestRank) | 2026-09-22 a5386b0e docs(design-v3): one declaration, one vocabulary, one priority line (#424) | |

### internal/testengine (3)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `ReadPlan` | `internal/testengine/planfile.go:53` | 테스트만 참조 | 0 | 2 (TestPlanFile_MissingIsAnError, TestPlanFile_SurvivesTheRunThatMadeIt) | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `afterFailedSetup` | `internal/testengine/suite.go:380` | 테스트만 참조 | 0 | 1 (AfterFailedSetupForTest) | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |
| `sortedScopes` | `internal/testengine/verifylaunch.go:207` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-09-20 6ed4b37c refactor: absorb the hardfork handoff, plan before composing, and close what the live run found (#419) | |

### internal/consensus/wbft (2)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `Family.RPCNamespace` | `internal/consensus/wbft/wbft.go:29` | 참조 없음 | 0 | 0 | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |
| `Family.ValidatorsMethod` | `internal/consensus/wbft/wbft.go:30` | 참조 없음 | 0 | 0 | 2026-07-24 05ef22c5 feat: Go-first multi-chain test-bench redesign (G0-G8) (#16) | |

### internal/core/filestore (2)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `New` | `internal/core/filestore/filestore.go:88` | 테스트만 참조 | 0 | 5 (TestProvision_ReusesIdenticalOverwritesDifferent, TestProvision_UploadIfAbsent, TestProvision_WritesFiles) | 2026-08-26 c21b3acf refactor(keyring): four layers, honest names, and no netmap coupling (#295) | |
| `*Provisioner.Provision` | `internal/core/filestore/filestore.go:92` | 테스트만 참조 | 0 | 5 (TestProvision_ReusesIdenticalOverwritesDifferent, TestProvision_UploadIfAbsent, TestProvision_WritesFiles) | 2026-08-06 cfea822b feat(provision): materialize node inputs with upload-if-absent (M5, T3.1) (#184) | |

### internal/core/inspector (2)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `Host.HostPort` | `internal/core/inspector/hosts.go:20` | 도달 불가 코드에서만 참조 | 1 | 0 | 2026-08-28 93ea8981 refactor(inspector): the target is asked what is there, before launch (P3.3) (#308) | |
| `Hosts` | `internal/core/inspector/hosts.go:25` | 테스트만 참조 | 0 | 1 (TestHosts_ReportsTheUnreachable) | 2026-08-28 93ea8981 refactor(inspector): the target is asked what is there, before launch (P3.3) (#308) | |

### cmd/chainbench (1)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `ReadOnlyPaths` | `cmd/chainbench/query.go:96` | 테스트만 참조 | 0 | 3 (TestQuery_HoldsExactlyWhatDeclaredItself, TestReadOnly_DoesNotCoverAWriteOrASecret, TestReadOnly_TheTwoSurfacesAgree) | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |

### internal/chains/stablenet/govbind (1)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `ExtractProposalID` | `internal/chains/stablenet/govbind/govbind.go:143` | 테스트만 참조 | 0 | 3 (TestExtractProposalID) | 2026-07-27 3350a74d Multi-chain separation remediation S1-S5 (#66) | |

### internal/core/blueprint (1)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `portFields` | `internal/core/blueprint/resolve.go:282` | 테스트만 참조 | 0 | 1 (TestMergePorts_CoversEveryPort) | 2026-09-07 c8578e87 feat(blueprint): one document that says what a network is, from declaration to live chain (N2-N6) (#357) | |

### internal/core/genesis (1)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `BuildNetwork` | `internal/core/genesis/genesis.go:81` | 테스트만 참조 | 0 | 3 (TestBuildNetwork_AppliesOverridesAndOverlay, TestBuildNetwork_PlainBuildMatchesBuild, TestBuildNetwork_RejectsBadForkOrdering) | 2026-08-14 f7097ee1 refactor(specs): retire pipeline/setup, complete DSL case migration batch (#240) | |

### internal/core/keyring (1)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `DefaultHDPath` | `internal/core/keyring/source.go:34` | 테스트만 참조 | 0 | 2 (TestSources_AllYieldAPrivateKey, TestSources_Failures) | 2026-08-21 4578c661 refactor(keyring): one package and one command group for key material (#244) | |

### internal/core/lifecycle (1)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `Status.IsFailure` | `internal/core/lifecycle/status.go:33` | 참조 없음 | 0 | 0 | 2026-09-21 7cd42aac refactor: design v3 — measure the tree, make each word mean one thing, and let every verb say what it needs (#422) | |

### internal/core/rpc (1)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `*Client.BlockMiner` | `internal/core/rpc/client.go:409` | 참조 없음 | 0 | 0 | 2026-09-07 4d6d370a fix(health): wait for a network to be formed, not merely for its head to move (#358) | |

### internal/dsl (1)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `Spec.Get` | `internal/dsl/spec.go:188` | 테스트만 참조 | 0 | 4 (TestGet_DotPath) | 2026-08-06 8b044520 feat: freeze component interfaces (T0.1) (#177) | |

### internal/mcp (1)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `*Server.ReadOnlyTools` | `internal/mcp/server.go:28` | 테스트만 참조 | 0 | 3 (TestReadOnlyTools_ReadsTheSameDeclaration, TestReadOnly_CoversNoToolThatWrites, TestReadOnly_TheTwoSurfacesAgree) | 2026-09-08 956600c3 feat(surface): one registration behind the flags a person types and the schema an agent reads (#361) | |

### internal/testsupport (1)

| 함수 | 위치 | 분류 | 운영 참조 | 테스트 참조 | 마지막 변경 | 판단 |
|---|---|---|---|---|---|---|
| `ServersBuildDir` | `internal/testsupport/servers.go:19` | 테스트만 참조 | 0 | 9 (TestLive_DownloadDirRecoversARootOwnedTree, TestLive_ElevatedDownloadReadsARootOwnedFile, TestLive_LogsReadsANodeLogFromItsServer) | 2026-08-28 bc64b26c refactor: P1 — the node vocabulary and the resource module find their owners (#301) | |

## 부록: 209건이 실제로 밟은 흐름 (RUN-EACH, 커버리지 빌드)

209건 모두 pass. 한 번 이상 실행된 함수 1257개, 209건 모두에서 실행된 함수 851개(= 어떤 케이스든 반드시 지나는 기본 경로).

| 패키지 | 기본 경로 함수 | 일부 케이스만 | 합계(실행된 것) |
|---|---|---|---|
| internal/chainsetup | 263 | 60 | 323 |
| internal/testengine | 120 | 27 | 147 |
| internal/testhelper | 7 | 113 | 120 |
| internal/core/session | 41 | 6 | 47 |
| internal/consensus/poa | 3 | 38 | 41 |
| internal/core/nodeconfig | 32 | 3 | 35 |
| internal/dsl | 23 | 9 | 32 |
| internal/core/node | 21 | 9 | 30 |
| internal/dsl/interp | 19 | 10 | 29 |
| internal/core/keyring/store | 8 | 19 | 27 |
| internal/core/rpc | 10 | 17 | 27 |
| internal/accounts | 4 | 22 | 26 |
| internal/core/process | 23 | 2 | 25 |
| internal/resource | 25 | 0 | 25 |
| internal/core/genesis | 3 | 21 | 24 |
| internal/dsl/assert | 0 | 24 | 24 |
| cmd/chainbench/chaincmd | 24 | 0 | 24 |
| internal/core/statemachine | 24 | 0 | 24 |
| cmd/chainbench/keyringcmd | 21 | 0 | 21 |
| internal/core/collector | 17 | 2 | 19 |
| internal/preset | 13 | 4 | 17 |
| internal/core/registry | 16 | 0 | 16 |
| internal/chainsetup/verb | 12 | 4 | 16 |
| internal/consensus/wbft | 3 | 12 | 15 |
| internal/core/keyring/derive | 7 | 1 | 8 |
| cmd/chainbench/networkcmd | 8 | 0 | 8 |
| cmd/chainbench/lifecyclecmd | 8 | 0 | 8 |
| cmd/chainbench/suitecmd | 8 | 0 | 8 |
| internal/app | 7 | 0 | 7 |
| internal/core/filestore | 7 | 0 | 7 |

케이스별로 실행된 함수 전체는 `flow/execd.json`에 있다(케이스 id → 함수 목록).
