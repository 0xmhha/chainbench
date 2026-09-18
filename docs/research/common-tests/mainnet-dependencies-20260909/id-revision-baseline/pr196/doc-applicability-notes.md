# 문서 분류 및 적용 시 주의점

범위는 common_test_scenarios.md의 시나리오 45행과 dual_chain_test_scenarios.md의 시나리오 29행이다. dual 문서 앞의 기능 지원 매트릭스 8행은 테스트가 아니라 설명 표이므로 테스트 개수에 포함하지 않았다. 테스트 이름·목적·흐름·기대값·ID 참조 전체 및 raw_row를 JSON에 보존했다. 한 행이 여러 테스트 ID나 스크립트를 가리키거나 같은 ID가 다른 행에 재등장하더라도 병합하지 않았다. 원문 ID 범위(TC-1-2-01~06)도 손실 없이 보존했으며 이를 검증되지 않은 개별 테스트 6개로 부풀리지 않았다.

## 실행 가능성의 의미

- applicable: go-wemix에 대상 기능이 구현되어 있으며 문서 절차를 이식해 수행할 수 있다. 제공된 .sh 참조의 존재 또는 성공을 의미하지 않는다.
- conditional: 로컬/원격 txpool 정책, snap 동기화 환경, Brioche 설정, 또는 EIP-7702와 일반 estimateGas baseline을 분리하는 추가 조건이 필요하다.
- excluded: 원문 기대결과가 go-wemix에 없는 WBFT/istanbul/EIP-7702/P256VERIFY 기능에 의존한다. 제외 항목의 P3는 정렬 목적이며 실행 권고가 아니다.

common의 45행은 43행 applicable + 2행 conditional이다. dual은 2행 conditional + 27행 excluded이다. 총 47행이 수행 대상으로 남지만 모두 명세 또는 이식 후보 단계다. tests/tc와의 대응은 별도 통합 목록에서 판단한다.

## 코드로 확인한 기대값 보정

1. Common 문서의 '공통'이라는 표기를 무조건 신뢰하면 안 된다. go-wemix는 타입 0x0/0x1/0x2/0x16을 지원하지만 유형별 fork gate가 있다(`sources/pr-head/core/types/transaction.go:45`, `sources/pr-head/core/tx_pool.go:651`, `sources/pr-head/core/tx_pool.go:1398`). Applepie 이전에 대납 성공을 요구할 수 없다.
2. Tip underpriced는 원격 제출의 `GasTipCap < pool.gasPrice`와 로컬의 `DropUnderPriced && EffectiveGasTip < pool.gasPrice`를 구분한다(`sources/pr-head/core/tx_pool.go:693`, `sources/pr-head/core/tx_pool.go:697`). StableNet의 Anzeon MinTip/WBFTExtra를 이식하지 않는다.
3. FeePayer 잔액 부족은 `ErrFeePayerInsufficientFunds`이며 송신자 부족도 별도 오류다(`sources/pr-head/core/tx_pool.go:709`). 문서의 느슨한 문자열 예시 대신 오류 원인과 pool 미수용을 확인한다.
4. Nonce·교체 검증은 채굴 제어가 필요하다. queued 교체는 설정된 PriceBump와 두 fee cap을 만족해야 한다(`sources/pr-head/core/tx_pool.go:182`, `sources/pr-head/core/tx_pool.go:851`). 대용량 거래 skip 이후 후속 nonce를 건너뛰는지는 기존 문서에 없으므로 별도 경계 테스트가 필요하다.
5. Receipt의 effectiveGasPrice는 London 전후 계산이 다르다(`sources/pr-head/internal/ethapi/api.go:1871`). 생산/동기화 노드의 동일 높이뿐 아니라 동일 block hash를 대조한다.
6. RPC txpool_status는 hex quantity다(`sources/pr-head/internal/ethapi/api.go:241`). 원문의 '정수'를 JSON 정수 타입으로 고정하면 오판한다. sendRawTransaction 후 pool 관측은 즉시 채굴되면 실패할 수 있다.
7. SnapSync 구현 존재는 배포 환경에서 해당 경로로 완료됨을 증명하지 않는다(`sources/pr-head/eth/downloader/downloader.go:1565`, `sources/pr-head/eth/downloader/downloader.go:1710`). full fallback과 구분해야 한다. Downloader/fetcher도 단순 block gap 수치만으로 경로 진입을 증명하지 못한다.
8. Dual의 EIP-7702 estimateGas baseline(DOC-D-026)은 authorization 없는 일반 전송만 요구한다. 이 부분은 `sources/pr-head/internal/ethapi/api.go:1340`으로 이식 가능하다. EIP-7702 지원으로 해석하지 않는다. DOC-C-017과 관련되지만 원문 행은 독립 보존했다.
9. WBFT 장애율 1/3 및 seal·round·1초 고정 주기 oracle을 SPoA에 그대로 적용하면 안 된다. 실제 엔진 구성은 `sources/pr-head/eth/ethconfig/config.go:217`, mining token 획득/반납은 `sources/pr-head/miner/worker.go:1627`, `sources/pr-head/miner/worker.go:1812`이다. 장애복구 의도만 go-wemix 전용으로 다시 설계할 수 있다.
10. secp256r1 무효 서명 호출이 빈 값을 반환하더라도 성공이 아니다. 프리컴파일 미지원 주소에 대한 빈 응답일 수 있다. 등록 집합에 0x100이 없다(`sources/pr-head/core/vm/contracts.go:48`, `sources/pr-head/core/vm/contracts.go:84`).
11. Brioche RPC는 wemixapi.Info가 없으면 nil 반환이며 fork 설정과 보상 구성도 필요하다(`sources/pr-head/eth/api.go:748`, `sources/pr-head/eth/api.go:764`, `sources/pr-head/params/config.go:443`). RPC 명칭 일치만으로 하네스 준비가 끝나지 않는다.

## 우선순위 근거와 한계

P0는 변경된 채굴 선택과 블록 수신·수입 경로에 직접 닿는 nonce, full sync, downloader, fetcher 4행이다. P1은 거래 유형·오류·실행 상태·receipt·재시작 등 필수 회귀, P2는 간접 RPC·운영 및 조건부 baseline, P3는 최소 smoke와 제외 항목이다. 작은 거래·작은 블록만 사용하는 기존 P0 테스트는 8 MiB RLP 제한이나 10 MiB 프로토콜 제한의 경계를 검증하지 못한다. 이 우선순위는 추가 경계 테스트를 대체하지 않는다(`sources/pr-head/core/block_validator.go:55`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/eth/protocols/eth/protocol.go:51`).

분석 대상 코드는 `sources/pr-head` 캡처이며 사용자의 로컬 `sources/local-head`와 구분한다. 스크립트와 노드·네트워크 테스트는 실행하지 않았다. 제공 문서의 원래 기대값은 보존했으며 보정 사항은 별도 필드로 기록했다.

## 재실행·누락 검사

`scripts/catalog_documents.py`를 실행하면 JSON/Markdown 및 document-catalog-validation.json을 재생성한다. 각 문서의 테스트 표 행을 직접 파싱하며 열 수 6개 및 문서별 행 수(45/29)가 달라지면 실패하여 새 문서에 대한 수동 재검토를 요구한다. 분류표가 위치 인덱스 기반이므로 내용·순서가 바뀐 새 문서는 자동 판정을 재사용하지 말고 common_review/dual_review를 재검토해야 한다. 현재 결과는 모든 원문 행과 코드 참조 파일·줄 범위를 검증했다.
