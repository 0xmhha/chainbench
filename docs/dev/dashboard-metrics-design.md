# 대시보드 metric 시각화 — 자체 동작 설계와 레퍼런스 코드베이스

> **등급: [현행 설계]** — 아직 구현하지 않는다. 착수 시점의 컨텍스트를 이 문서가 보존한다.
> 작업 상태는 worklist §1g D 트랙이 정본이다.

`chainbench-dashboard` 의 디버깅 지원 1단계는 노드 metric 을 받아 인포그래픽으로
보여주는 것이다. 이 문서는 (1) 외부 스택 없이 자체 동작한다는 결정과 그 근거,
(2) 목표 구조, (3) 구현 시 참조할 오픈소스 코드베이스를 정리한다.

## 1. 무엇이 이미 있는가 (실측)

세 체인(gstable·gwbft·gwemix) 모두 geth 계열이라 metric 경로가 두 가지다
([`docs/chain-analysis/*/cli-flags.txt`](../chain-analysis/)).

| 경로 | 켜는 법 | 형식 |
|---|---|---|
| HTTP 스크레이프 | `--metrics --metrics.addr/port` | `/debug/metrics/prometheus` 에서 Prometheus 텍스트 |
| InfluxDB 푸시 | `--metrics.influxdb`(v1/v2) | 노드가 InfluxDB 서버로 직접 전송 |

그리고 우리 쪽 부품이 이미 넷 있다.

- `collector.ScrapeMetrics` — 그 엔드포인트를 읽어 이름→값 맵으로 파싱한다. DSL 의
  metric 어세션이 이미 이걸로 돈다. 단 **라벨을 버리는 간이 파서**다(§5 참고).
- `netmap` — 노드별 metric 포트를 안다. 스크레이프 대상 목록이 공짜로 나온다.
- `chainbench-dashboard` — 이벤트를 SSE 로 브라우저에 흘리는 배관(`/events`)과
  세션 조회 API(`/api/sessions`)가 있다.
- SPA — 117줄짜리 이벤트 목록 화면. 차트는 없다.

## 2. 결정 — Prometheus·Grafana 서버 없이 자체 동작한다 (2026-08-24)

"Prometheus 형식"은 텍스트 포맷의 이름이지 서버 요구가 아니다. 자체 동작의 근거는
셋이다.

1. **필요 기능이 없다.** 그 스택이 주는 것은 장기 보관·PromQL·알림인데, 테스트벤치의
   네트워크는 분~시간 단위로 살다 사라지고 보고 싶은 것은 이번 실행의 추이다.
2. **부품이 이미 있다.** 파서·대상 목록·전달 배관·화면 골격 전부 재사용이다(§1).
3. **수집은 스크레이프가 맞다.** InfluxDB 푸시는 상시 서버를 하나 더 요구한다.

Grafana 는 배제가 아니라 **외부 선택지**다. 노드 엔드포인트가 표준 형식 그대로라,
원하는 사람은 자기 Prometheus 로 같은 주소를 긁으면 된다. 우리가 내장할 것은 없다.

## 3. 목표 구조 (외부 의존 0)

```
노드들 (--metrics)
   ↑ 몇 초 간격 HTTP 스크레이프 (collector.ScrapeMetrics 재사용, 대상은 netmap)
chainbench-dashboard
   - 메모리 링버퍼: 노드별 시계열 (이번 실행 분량, 수 MB)
   - GET /api/metrics: 스냅샷 · SSE 증분 스트림 (기존 /events 배관)
   ↓
SPA 차트: 블록 높이 · 피어 수 · txpool 을 노드별 비교
```

새로 만드는 것은 세 조각뿐이다: 주기 스크레이프 루프, 메모리 시계열 버퍼, SPA 차트.
만들지 않는 것: 파서(라이브러리로), 영속 저장소(테스트벤치에 불필요), 질의 언어.

## 4. 레퍼런스 코드베이스 — 처음부터 만들지 않기 위해

레퍼런스는 두 층으로 나눈다. **코드 모양은 범용 프로젝트에서, 지표의 의미는 이더리움
생태계에서** 빌린다. 이렇게 나눈 이유는 §4b 에 있다 — 이더리움 전용 대시보드형
스탠드얼론 중 활발히 유지되는 것을 찾지 못했다.

### 4a. 코드 모양 레퍼런스 (범용)

선정 기준: 코드를 참조·차용할 수 있는 라이선스(Apache/MIT)이고, 활발히 유지되며,
우리 모양(Go 데몬 + 내장 웹 UI + 주기 수집)과 겹칠 것. **무엇을 보여주나가 아니라
어떻게 만들었나를 빌리는 기준**이다.

| 프로젝트 | 라이선스 | 무엇을 빌리는가 |
|---|---|---|
| [prometheus/common `expfmt`](https://github.com/prometheus/common) (+ [prom2json](https://github.com/prometheus/prom2json)) | Apache-2.0 | **파서.** Prometheus 텍스트의 공식 Go 파서다. 차트에 라벨이 필요해지는 순간 collector 의 간이 파서를 이것으로 교체한다. prom2json 은 "스크레이프→JSON" 전체 흐름의 최소 예제 |
| [statsviz](https://github.com/arl/statsviz) | MIT | **1스텝과 가장 같은 모양의 실증.** in-process 시계열을 websocket 으로 브라우저에 밀어 실시간 플롯. 저장소 없이 링버퍼+스트림+플롯으로 끝나는 구조, 사용자 정의 플롯(userplot) API 의 설계 |
| [Gatus](https://github.com/TwiN/gatus) | Apache-2.0 | **데몬의 뼈대.** 단일 Go 바이너리 + 주기 체크 루프 + 메모리 저장(옵션으로 SQL 영속) + 내장 UI. 스케줄링·타임아웃·상태 보관의 구조 참조 |
| [Beszel](https://github.com/henrygd/beszel) | MIT | **차트 화면.** 경량 모니터링 허브(21k+ 스타, 활발). 노드별 지표를 시계열 차트로 비교하는 UI 의 구성과 상호작용 참조. PocketBase 의존이라 백엔드 구조는 우리와 다름 |
| [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics) (vmagent/vmui) | Apache-2.0 | **스크레이프 루프의 프로덕션 답안.** 간격·타임아웃·재시도·staleness 처리를 어떻게 하는지. 통째로 들일 물건은 아니고 패턴만 |

**코드 참조에서 제외**: Grafana(AGPLv3 — 코드를 빌리면 라이선스가 전염된다),
Netdata(UI 가 비공개 사유 라이선스).

### 4b. 이더리움 특화 레퍼런스 — 조사 결과와 쓰임새

이더리움 전용 대시보드형 프로젝트를 별도로 조사했다(2026-08). 결론: **활발히
유지되는 스탠드얼론은 없다.** 생태계는 "exporter + Grafana 대시보드 JSON" 조합으로
수렴했고, 대시보드형의 원형이던 ethstats 계열은 휴면 상태다. 그래도 세 가지는
레퍼런스로 유효하다.

| 프로젝트 | 상태·라이선스 | 무엇을 빌리는가 |
|---|---|---|
| [ethstats-server / eth-netstats](https://github.com/goerli/ethstats-server) | 휴면(goerli 포크가 마지막 계보) · GPL-3.0 | **화면 구성(UX)만.** 노드별 실시간 비교(블록 높이·피어·전파 지연·마지막 블록 시각)가 우리 1스텝이 그릴 화면의 원형이다. GPL 이라 **코드 차용 불가**, 실행해 보는 것은 가능. 참고: **세 체인 모두 `--ethstats` 플래그를 갖고 있다**(실측, cli-flags.txt) — 노드가 자체 websocket 프로토콜로 밀어주는 푸시 모델인데, 우리는 스크레이프 풀 모델을 택했으므로 이 경로는 쓰지 않는다 |
| [ethereum/nodemonitor](https://github.com/ethereum/nodemonitor) | 휴면 · ethereum 재단 | **모양의 선례.** 여러 노드를 RPC 로 폴링해 head 를 비교하는 작은 Go 유틸 + 웹페이지. "여러 노드를 긁어 한 화면에 비교"가 우리와 같은 모양임을 보여주는 실증. 규모가 작아 읽기 좋다 |
| [ethereum-metrics-exporter](https://github.com/ethpandaops/ethereum-metrics-exporter) (ethpandaops) | 활발 | **지표 이름→의미 매핑.** UI 는 없지만, geth 계열 지표를 어떻게 골라 어떤 이름으로 정리하는지의 현행 답안 |

**지표 선정 레퍼런스(코드 아님)**: geth 공개 Grafana 대시보드들
([Single Geth 13877](https://grafana.com/grafana/dashboards/13877-single-geth-dashboard/) ·
[Geth Prometheus 23174](https://grafana.com/grafana/dashboards/23174-geth-prometheus/) ·
[공식 문서](https://geth.ethereum.org/docs/monitoring/dashboards)). 운영자들이 실제로
보는 지표 목록(chain/head/block, p2p/peers, txpool/pending, eth/db/chaindata 등)을
1스텝 차트의 후보 목록으로 쓴다.

## 5. 열린 질문 (착수 시 결정)

| # | 질문 | 기울기 |
|---|---|---|
| DM-a | 차트 라이브러리 — uPlot(MIT, 초경량) vs ECharts(Apache) vs 직접 SVG | 시계열 다중 비교라면 uPlot 기울기. statsviz 도 uPlot 사용 |
| DM-b | collector 간이 파서를 언제 `expfmt` 로 바꾸나 | 라벨 있는 지표(예: `rpc/duration` 계열)를 그리는 시점. 그 전까지는 간이판으로 충분 |
| DM-c | 링버퍼 보관량 | 해상도 5s × 2h ≈ 노드당 지표당 1,440샘플 기울기. 실행이 그보다 길면 다운샘플 |
| DM-d | 스크레이프 주기와 노드 부하 | `--metrics.expensive` 는 켜지 않는 기본. 5s 기울기 |

## 6. 단계별 디버깅 지원 로드맵

| 단계 | 내용 | 비고 |
|---|---|---|
| **D1** | metric 인포그래픽 — §3 구조 구현 | 이 문서의 범위. worklist §1g D 트랙 |
| D2 | 완료 세션 화면 — `/api/sessions`·chainstate API 는 있는데 화면 소비자가 없다 | 순수 프론트 작업 |
| D3 | 로그 연계 — 이벤트/차트의 시점에서 해당 노드 로그로 점프 | collector 의 tail 배관 재사용 |
