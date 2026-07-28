# docs/legacy/ — Go 재설계 이전 아카이브

이 디렉토리의 문서는 chainbench가 **Go-first 다체인 테스트벤치로 재설계되기 이전**의
3-스택 아키텍처 — bash CLI(`chainbench.sh` · `lib/*.sh`), TS `mcp-server/`,
`network/` wire 모듈 — 와 그 sprint 로드맵을 기술한다. 그 3-스택은 모두 제거되었고
저장소는 단일 Go 아키텍처로 수렴했다.

**전부 SUPERSEDED.** 현행 문서는:

- 아키텍처: [`../CHAINBENCH_GO_REDESIGN.md`](../CHAINBENCH_GO_REDESIGN.md)
- 상태·남은 작업: [`../dev/HandOff.md`](../dev/HandOff.md)

아래는 역사적 기록으로만 보존한다(설계 근거·의사결정 추적).

## 파일

| 문서 | 성격 |
|---|---|
| `VISION_AND_ROADMAP.md` | bash-era 비전·로드맵 SSoT(Sprint 체크박스, 설계 결정 Q/S). |
| `NEXT_WORK.md` | 전체 작업 핸드오프(디렉토리 레이아웃·규약·P3 표 풀버전). |
| `REMAINING_WORK.md` | actionable 남은 작업 리스트(Sprint 5 시리즈). |
| `REFACTORING_PLAN.md` | clean-code/SSOT 리팩토링 트랙. |
| `ADAPTER_CONTRACT.md` | bash 어댑터(`lib/adapters/*.sh`) 계약 + Go 포팅 진행. |
| `HARDCODING_AUDIT.md` | `lib/cmd_*.sh`의 `gstable` 하드코딩 감사(Manifest 데이터화로 해소). |
| `EVALUATION_CAPABILITY.md` | sprint별 evaluation 능력 매트릭스. |
| `test-env-migration-handoff.md` | bash `feat/unified-test-env` 테스트 환경 통합 핸드오프. |

> 주의: 이 문서들 사이의 상호참조는 **이동 이전의 원본 경로**(`docs/X.md`)를 그대로 쓴다.
> 실제 위치는 모두 이 `docs/legacy/` 디렉토리다. 아카이브라 경로는 재작성하지 않았다.
