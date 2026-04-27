# claudedocs/ — 외부 컨텍스트 자료

이 디렉토리는 chainbench 의 **외부 컨텍스트** — 즉 chainbench 가 어떤 더 큰
시스템 안에서 어떤 위치를 차지하는지 — 를 보존한다.

`packages/claude-ai` 기반 go-stablenet 자동화 시스템(coding agent 시스템) 의
작업 지시서/제안서 모음으로, 그 시스템 안에서 chainbench 가
**Feature 3 — 체인 테스트벤치 통합 (evaluation tool)** 위치를 차지한다는
사실의 cross-reference 자료다.

## 파일

| 파일 | 작성일 | 역할 |
|---|---|---|
| `AUTOMATION_SYSTEM_PROPOSAL.md` | 2026-04-03 | 자동화 시스템 v1 제안서 |
| `AUTOMATION_SYSTEM_PROPOSAL_v2.md` | 2026-04-03 | v2 (30+ 레퍼런스 프로젝트 분석 추가) |
| `WORK_INSTRUCTION.md` | 2026-04-03 | Jira 티켓 단위로 분해한 작업 지시서 |

## chainbench 와의 관계

- chainbench 의 **상위 비전** 은 `docs/VISION_AND_ROADMAP.md` §1.1 에서 명시:
  (A) coding agent evaluation harness, (B) 독립 도구.
- **모드 (A)** 의 외부 컨텍스트가 본 디렉토리. 본 디렉토리 문서는 chainbench 가
  자동화 시스템 안에서 어떤 책임을 갖고 어떤 표면(MCP) 으로 호출되는지의
  근거를 제공한다.
- 단 **모드 (B)** 독립 사용 가능성은 항상 보장됨. chainbench 자체 sprint 우선순위·
  아키텍처 결정은 본 디렉토리가 아닌 `docs/VISION_AND_ROADMAP.md` 와
  `docs/NEXT_WORK.md` 가 SSoT.

## 본 디렉토리 문서 갱신 정책

- 본 디렉토리 3개 파일은 외부 시스템(`packages/claude-ai`) 의 산출물이므로
  chainbench 작업 중 **수정하지 않는다**. 외부 시스템에서 갱신된 사본을 받아
  여기에 동기화하는 형태.
- chainbench 가 모드 (A) 의 요구를 어떻게 충족하는지의 추적은
  `docs/EVALUATION_CAPABILITY.md` 가 담당.
