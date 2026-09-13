# 분석과 근거

분석 입력 revision: `7f39c5627e0bdaccaa8e207a67767d71579283f5`. 아래 내용은 보존된 분석 결과의 검토용 발췌다. 전체 그래프의 최신성이나 전역 참조 검증 완료를 주장하지 않는다. 선택된 근거의 repository/snapshot/path/location/hash/원문은 [근거 발췌 JSON](evidence-excerpts.json)에 함께 보관한다. 원래 수집 시점의 미추적 문서는 이 커밋의 소스 파일이라고 간주하지 않는다.

## 목적과 분석 방법

세 geth 계열 체인의 네트워크 구성·검증·테스트를 공통 Go 코어와 CLI/MCP/dashboard에서 일관되게 제공한다. 합의 패밀리 재사용과 체인별 선언적 확장이 비전이다.

문서는 문서·절을 대상으로 X⁰=주제, complement=필수 논항, adjunct=선택 수식, specifier=대상·맥락을 적용했다. 의미 주석의 confirmed/candidate/unknown과 근거 구절을 보존한다. 개별 문장 구문 분석을 완료했다는 뜻은 아니다. 코드는 Go AST와 타입 정보를 사용해 import/reference/call/implements를 구별했다. 정적 implements는 런타임 호출을 확정하지 않으며 동적 대상은 unknown으로 유지한다.

보존 산출물의 보고 수치: Go 파일 723개, 문서 108개·절 1,685개, 호출 지점 28,415개, 별도 타입 변환 1,048개. 이 수치는 수집 snapshot의 보고값이며 현재 PR의 재추출 결과가 아니다.

## F1 · 구성 소유권이 테스트 오케스트레이션에 걸쳐 있다

지위: `candidate` (관측 사실과 설계 해석의 확정 정도를 구별).

관측: testengine/compose.go imports chainsetup and consensus/upgrade; compositionOf constructs either composition input.

해석: 테스트 선언의 환경 해석과 네트워크 구성 정책 변경이 같은 모듈에 모인다. 응집도 저하 위험이며 import 방향 위반이라고 단정하지 않는다.

근거: `ev:20a1d59977e9aa47fa0ed2f7107bea8167b623c2666f7d2cb06d73b28d8ac390`, `ev:7551b641431c3bf631233dbc2754c89b1dd8cd6d8344e0e5844187a599994355`, `ev:2f25bc6d42da781775cf15188a332557252c635e2d12c225d10ab2be2c3fa231`

## F2 · app의 표면 계약과 구성 타입 소유권이 결합되어 있다

지위: `candidate` (관측 사실과 설계 해석의 확정 정도를 구별).

관측: app/net.go aliases chainsetup inputs and forwards calls through chainsetupDeps.

해석: 한 구현을 공유하는 장점이 있다. 다만 내부 구성 타입 변경이 app 입력에도 전파되어 독립 변경의 폭을 줄인다. forwarding 자체는 중복 결함이 아니다.

근거: `ev:f95ba0f333b62b7ed8607ce214b8244b5e8f19fb99db292a08fdf3fc328699d7`, `ev:bf304cdf79a8be8a1f3c2aa001274558a25c48c25a35fd5aaf0449238dee13f4`, `ev:2927509ef84e189d697e8a9309c85d8d38a4943033a1fdd2a2ab9798927a6ed6`

## F3 · 문서의 표면 경로 설명이 현재 코드와 상충한다

지위: `confirmed` (관측 사실과 설계 해석의 확정 정도를 구별).

관측: README says CLI calls core directly; networkcmd imports app and invokes DetectNetwork.

해석: 설계 문구만으로 레이어를 재분류하면 실제 호출 경로를 오판한다. 층 번호와 기능 책임은 별도 축으로 검토해야 한다.

근거: `ev:fef97b7fae1488dcf8b264cc4038a311c5bfdcde761e20c3b774aba5ec0d895d`, `ev:b70f032c7ebf81797c757e4e2ba5760bb862e8b0f3f31aa9ce7088faf8fd741d`, `ev:50869b3af4d3aa8a6ef14edffa46108c28b73e73d306bab23067c41d378b9698`

## F4 · DSL 실행의 주입 경계는 실제 호출 대상과 다르다

지위: `confirmed` (관측 사실과 설계 해석의 확정 정도를 구별).

관측: interp.Registry is an interface passed in Deps.Actions; dispatch is name-based.

해석: 현재 주입은 체인 어휘로부터 해석기를 분리하는 반대 근거다. 인터페이스 만족 간선으로 런타임 결합을 주장하면 안 된다. 기능 등록·실행 경계의 변경 영향은 이름 계약에 걸린다.

근거: `ev:5f910ba1e49a520e2c1a28cd73a01b2ba320621242e7782ecb67493cacc3b687`, `ev:33b99f2e8fcf9b655f80ae6b8a2b31ae9c7796b3fd904332deca78f0751eceb1`, `ev:e7fdf7d8902392dfb8f24acdb1a7d52e06b80a421a0c096099529c1b2402f914`

## F5 · 과거 중복 진단을 현재 결함으로 재사용할 수 없다

지위: `confirmed` (관측 사실과 설계 해석의 확정 정도를 구별).

관측: The September 10 review explicitly records September 11 fixes and corrections to its measurements.

해석: 응집도 진단의 확정 근거로 과거 중복 개수를 쓰지 않는다. 현재 graph의 파일·호출 위치만 현재 사실로 사용한다.

근거: `ev:88191dccc80fba8b05e60d965a0edbb8bbe20d1f150eb09bbb268aa9d863a80e`, `ev:1f969aeb905aa8c49aeae3aa209bbe389812e64b0ac01f1ea08fb44ed75a6fb4`

## F6 · receipt/latest 관측 경계의 실패와 원인은 구별한다

지위: `confirmed` (관측 사실과 설계 해석의 확정 정도를 구별).

관측: Original baseline FAIL; restored node1/3/4 show block 1, 1 ETH and successful receipt; node2/5 genesis. Instrumented 150 observations show 1 ETH; one unmodified rerun PASS. head maps RPC errors to -1; waitAdvancing compares height increases, allowing error-to-genesis to appear as progress. These code weaknesses do not establish the original failure cause.

해석: restored state contradicts permanent transfer loss; instrumented and unmodified passes do not erase original failure

근거: `ev:5dfd021e8b04425e46a10b5f2ff552a71f6fc092fc3f270d6c8b8e25278801d2`, `ev:ec869ea971a42e671ffa5dc034a3e66006925dd931e527ca9b194a93970ab7fd`, `ev:c20f6d48b5963d3120e5f6d3adda7736647b7627b885f703003b4c3bbb9cacd3`, `ev:7285a01ec0cc18424086f478855ff5b0f0741046853403035be7c1eee7a1dc93`, `ev:a94dbc1853066c60ea94afa7bc7987824c4b34d51f8b970d1fcaeb7f6385e87d`, `ev:239a89c47f41fca599ca7d50be045b76a49bbdba44c37d7211783ffb7db733dc`

## 근거 읽기와 한계

F1/F2는 변경 위험 후보이며 확정 결함이 아니다. F4는 유지해야 하는 기존 주입 경계이고 F5는 과거 수치를 현재 결함으로 오용하지 않기 위한 제약이다. F6의 확정은 기록된 실패·관측·코드 경계에 한정하며 최초 Stablenet 실패 원인은 미확정이다. 전체 분석에 대한 전역 ID 검사와 최신 입력 재확인은 미완료다.
