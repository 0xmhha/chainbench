# 빌드 기준 AST 그래프 (2026-09-11)

세 클라이언트의 실제 빌드 산출물(go-wemix `make gwemix`, go-wbft `make gwemix`, go-stablenet `make gstable`)에 포함되는 Go 파일만 골라 tree-sitter로 파싱했다. 절차: `go list -mod=readonly -deps -json ./cmd/<entry>` → 저장소 내부 GoFiles/CgoFiles 선택(`scripts/select_build_files.py`) → `build-selected/<project>/` 미러 → codemine `extract_graph.py --module-depth 8` → `graph_report.py`. 테스트 파일과 외부 모듈은 제외했다. 그래프는 패키지 import 간선과 선언(타입·함수) 목록이며 함수 호출 그래프가 아니다. 다른 두 분석 문서의 인용 파일은 이 선택 목록 안에 있는지 검사했다.

| 프로젝트 | 진입 패키지 | 빌드 태그 | 패키지 | Go 파일 | 모듈(depth 8) | 타입 | import 간선 | 파싱 오류 |
|---|---|---|---:|---:|---:|---:|---:|---:|
| go-wemix | ./cmd/gwemix | (없음, darwin/arm64) | 120 | 630 | 120 | 1936 | 831 | 0 |
| go-wbft | ./cmd/gwemix | urfave_cli_no_docs,ckzg | 135 | 685 | 135 | 1824 | 851 | 0 |
| go-stablenet | ./cmd/gstable | urfave_cli_no_docs,ckzg | 129 | 668 | 129 | 1528 | 801 | 0 |

## 공통 테스트 분석에 쓴 모듈 위치

아래는 차이 분석(chain-differences.md)의 인용 파일이 속한 모듈과 fan-in(그 모듈을 import하는 내부 모듈 수)이다. fan-in이 높은 모듈의 차이는 더 많은 경로에 영향을 준다.

| 모듈 | go-wemix (files/fan-in) | go-wbft | go-stablenet |
|---|---|---|---|
| params | 8 / 30 | 8 / 40 | 8 / 37 |
| core/types | 21 / 0 | 30 / 0 | 31 / 0 |
| core/txpool | 없음 | 5 / 0 | 5 / 0 |
| core/txpool/legacypool | 없음 | 4 / 0 | 4 / 0 |
| core | 29 / 16 | 24 / 21 | 23 / 20 |
| core/vm | 21 / 0 | 22 / 0 | 23 / 0 |
| core/state | 10 / 0 | 11 / 0 | 11 / 0 |
| consensus/misc/eip1559 | 없음 | 1 / 0 | 1 / 0 |
| consensus/misc | 4 / 0 | 2 / 0 | 2 / 0 |
| eth/gasprice | 2 / 0 | 2 / 0 | 3 / 0 |
| internal/ethapi | 6 / 0 | 6 / 0 | 6 / 0 |
| eth | 12 / 9 | 17 / 6 | 16 / 6 |
| consensus/wbft | 없음 | 5 / 0 | 5 / 0 |
| consensus/wbft/backend | 없음 | 4 / 0 | 4 / 0 |
| consensus/wbft/engine | 없음 | 2 / 0 | 2 / 0 |
| consensus/wpoa | 없음 | 6 / 0 | 없음 |
| wemix | 5 / 1 | 없음 | 없음 |
| wemix/miner | 1 / 0 | 없음 | 없음 |
| wemix/bind | 8 / 0 | 없음 | 없음 |
| eth/downloader | 16 / 0 | 17 / 0 | 17 / 0 |
| eth/protocols/snap | 7 / 0 | 8 / 0 | 8 / 0 |
| eth/fetcher | 2 / 0 | 2 / 0 | 2 / 0 |
| eth/filters | 3 / 0 | 3 / 0 | 3 / 0 |
| miner | 5 / 12 | 5 / 5 | 5 / 5 |
| cmd/utils | 6 / 0 | 5 / 0 | 5 / 0 |

모듈이 "없음"인 칸이 그 자체로 차이다. go-wemix에는 `core/txpool`(txpool 패키지가 `core/tx_pool.go`에 있다), `consensus/wbft`, `consensus/misc/eip1559`(`consensus/misc/eip1559.go`에 있다), `miner` 대신 `wemix/miner`가 있다. go-wbft에만 `consensus/wpoa`(Croissant 전환 전 엔진)가 있다. go-stablenet에는 `wemix` 모듈이 없다.

## 파일별 보고서

- [go-wemix](../graph/go-wemix-report.md) · [go-wbft](../graph/go-wbft-report.md) · [go-stablenet](../graph/go-stablenet-report.md)
- 그래프 원본: `graph/<project>/code-graph.json`, `graph/<project>/module-imports.tsv`
- 선택 파일 목록과 해시: `build/<project>-selected.json`

## 한계

- go-wbft 작업 트리는 분석 중 HEAD가 움직였다(다른 세션의 rebase). 미러는 `b1dc5a8a` 시점이며, 이후 바뀐 파일은 `core/blockchain.go`, `core/blockchain_reader.go`, `rpc/client.go`, `rpc/websocket.go` 4개다. 이 파일들은 테스트 판정에 인용되지 않았다.
- darwin/arm64 프로필만 파싱했다. 2026-09-09 자료의 linux/amd64 비교에서 OS별 차이는 fdlimit/blake2b/metrics/rocksdb 파일뿐이었다.
- 함수 호출·동적 실행 경로는 그래프에 없다. 실행 결과 판정은 하지 않았다.
