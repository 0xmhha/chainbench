# docs/ — 문서 인덱스

chainbench는 **go-stablenet/wbft/wemix용 Go-first 다체인 테스트벤치**다. 이 디렉토리는
현행 설계·운영 문서와, Go 재설계 이전(bash CLI · TS mcp-server · `network/` wire 모듈)
아카이브를 함께 둔다.

## 먼저 읽을 것

| 문서 | 성격 |
|---|---|
| [`CHAINBENCH_GO_REDESIGN.md`](CHAINBENCH_GO_REDESIGN.md) | **아키텍처 SSoT** — 전체 설계 + 3단계 파이프라인 + 로드맵 + 현황. 여기서 시작. |
| [`dev/HandOff.md`](dev/HandOff.md) | **작업 핸드오프 SSoT** — 현재 상태 + 남은 작업 리스트. 다른 세션에서 이 문서만으로 이어감. |

## 현행 참고 문서

| 문서 | 내용 |
|---|---|
| [`CHAIN_EXTENSIBILITY_DESIGN.md`](CHAIN_EXTENSIBILITY_DESIGN.md) | 체인 플러그인 축(어댑터/매니페스트) 상세. REDESIGN의 하위 참조(부분 대체). |
| [`A0-ACCOUNTS-MULTICHAIN-SCOPE.md`](A0-ACCOUNTS-MULTICHAIN-SCOPE.md) | accounts SDK 다체인화 범위 조사(spike). 0x16 공통·Extra는 stablenet 전용. |
| [`MCP_USAGE.md`](MCP_USAGE.md) | chainbench MCP 서버 사용 가이드(도구 호출, `.mcp.json` 등록). |
| [`SECURITY_KEY_HANDLING.md`](SECURITY_KEY_HANDLING.md) | 키 취급 보안 정책·위협 모델. 프리셋 키는 **테스트 픽스처 전용**. |

## 하위 디렉토리

| 경로 | 내용 |
|---|---|
| [`dev/`](dev/) | 개발 핸드오프(`HandOff.md`) + 세션 아카이브(`session-data/`, 원본 transcript). |
| [`claudedocs/`](claudedocs/) | 외부 컨텍스트 — chainbench가 속한 상위 자동화 시스템의 제안서/지시서(cross-reference). |
| [`legacy/`](legacy/README.md) | **Go 재설계 이전 아카이브** — bash 로드맵·감사·핸드오프 8종. 전부 SUPERSEDED. |
