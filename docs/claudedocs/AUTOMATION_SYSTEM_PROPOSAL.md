# go-stablenet AI 자동화 시스템 고도화 제안서

> **Date**: 2026-04-03
> **Scope**: `packages/claude-ai/` 고도화 → go-stablenet 개발 자동화 시스템 구축
> **Status**: Draft Proposal

---

## 목차

1. [현재 상태 분석](#1-현재-상태-분석)
2. [목표 아키텍처](#2-목표-아키텍처)
3. [Phase 1 — Claude Code 기반 강화](#3-phase-1--claude-code-기반-강화)
4. [Phase 2 — MCP 서버 에코시스템](#4-phase-2--mcp-서버-에코시스템)
5. [Phase 3 — 외부 시스템 통합 (Jira/Confluence/Slack/GitHub)](#5-phase-3--외부-시스템-통합)
6. [Phase 4 — 지능형 워크플로우 자동화](#6-phase-4--지능형-워크플로우-자동화)
7. [레퍼런스 프로젝트별 적용 전략](#7-레퍼런스-프로젝트별-적용-전략)
8. [Best Practice: 추천 시스템 구성도](#8-best-practice-추천-시스템-구성도)
9. [구현 로드맵](#9-구현-로드맵)
10. [리스크 및 고려사항](#10-리스크-및-고려사항)

---

## 1. 현재 상태 분석

### 1.1 packages/claude-ai/ — 현재 구성

| 구성 요소 | 상태 | 내용 |
|-----------|------|------|
| CLAUDE.md | ✅ 완료 | 프로젝트 개요, 코드맵, 컨벤션 참조 |
| stablenet-review-code 커맨드 | ✅ 완료 | 구조화된 코드 분석 슬래시 커맨드 |
| CLAUDE_DEV_GUIDE.md | ✅ 완료 | 아키텍처, WBFT, 시스템 컨트랙트 |
| SYSTEM_CONTRACT_FLOW.md | ✅ 완료 | 시스템 컨트랙트 배포/업그레이드 경로 |
| CODE_CONVENTION.md | ✅ 완료 | Go/Solidity 코딩 표준 |
| REVIEW_GUIDE.md | ✅ 완료 | 질문 유형별 탐색 가이드 |
| BUILD_SOURCE_FILES.md | ✅ 완료 | 160 패키지, 781 파일 매핑 |
| settings.local.json | ✅ 완료 | 기본 권한 (python3, WebSearch, gh pr) |
| **MCP 서버** | ❌ 없음 | MCP 기반 도구 통합 없음 |
| **AI Agent 시스템** | ❌ 없음 | 자동화 에이전트 없음 |
| **Hooks 시스템** | ❌ 없음 | 이벤트 기반 자동화 없음 |
| **외부 시스템 통합** | ❌ 없음 | Jira/Confluence/Slack 연동 없음 |

### 1.2 이미 구현된 보조 도구

| 도구 | 위치 | 상태 | 역할 |
|------|------|------|------|
| **token-monitor** | ai-cli/token-monitor/ | ✅ 구현 완료 | 실시간 토큰 사용량 모니터링 (Go, MCP 서버 포함, TUI) |
| **chainbench** | ai-cli/chainbench/ | ✅ 구현 완료 | 로컬 블록체인 샌드박스 테스트벤치 (Bash+TS MCP, 13 MCP 도구) |

### 1.3 현재 한계

1. **수동 컨텍스트 로딩**: Claude가 매 세션마다 160 패키지 구조를 파악해야 함
2. **단방향 지식**: 문서는 있지만 능동적 분석/검증 도구 부재
3. **빌드/테스트 피드백 루프 부재**: 코드 수정 → 빌드 → 테스트 → 결과 분석의 자동화 없음
4. **체인 런타임 검증 분리**: chainbench가 있지만 Claude Code와의 통합 워크플로우 부재
5. **협업 도구 단절**: Jira 티켓 → 코드 변경 → PR → 리뷰의 자동화된 연결 없음

---

## 2. 목표 아키텍처

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Developer (Claude Code CLI)                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐    │
│  │ /commands │  │  Hooks   │  │  Skills  │  │ Agent Orchestrator│   │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘  └────────┬─────────┘   │
│        │             │             │                  │              │
│  ┌─────┴─────────────┴─────────────┴──────────────────┴──────────┐  │
│  │                    MCP Server Layer                             │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │  │
│  │  │stablenet │ │  chain   │ │  token   │ │   codebase       │  │  │
│  │  │  -build  │ │  bench   │ │ monitor  │ │   -memory        │  │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘  │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │  │
│  │  │  jira    │ │  slack   │ │confluence│ │   github         │  │  │
│  │  │  -mcp    │ │  -mcp    │ │  -mcp    │ │   -enhanced      │  │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                  Automation Workflows                            │ │
│  │  Jira Ticket → Branch → Implement → Build → Test → Chain       │ │
│  │  Verify → PR → Code Review → Slack Notify → Merge              │ │
│  └─────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. Phase 1 — Claude Code 기반 강화

> 기존 `packages/claude-ai/` 구조를 확장하여, Claude Code의 내장 기능(commands, hooks, settings)을 최대한 활용

### 3.1 슬래시 커맨드 추가

현재 `/stablenet-review-code` 1개 → **7개 커맨드**로 확장

| 커맨드 | 목적 | 설명 |
|--------|------|------|
| `/stablenet-review-code` | 코드 분석 | ✅ 기존 (유지) |
| `/stablenet-build` | 빌드 & 검증 | `make gstable` 실행 → 에러 분석 → 수정 제안 |
| `/stablenet-test` | 테스트 실행 | 특정 패키지/파일 테스트 → 결과 분석 → 실패 원인 추적 |
| `/stablenet-lint` | 린트 & 포맷 | `make lint` → CODE_CONVENTION.md 기반 자동 수정 |
| `/stablenet-impact` | 변경 영향 분석 | 수정 파일 → 영향받는 패키지/테스트 → 빌드 경로 추적 |
| `/stablenet-hardfork` | 하드포크 가이드 | SYSTEM_CONTRACT_FLOW.md 기반 단계별 체크리스트 생성 |
| `/stablenet-debug` | 체인 디버깅 | chainbench + 노드 로그 분석 → 합의 이슈 진단 |

### 3.2 Hooks 시스템 구성

Claude Code hooks를 활용한 이벤트 기반 자동화:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [{
          "type": "command",
          "command": "echo '⚠️ Go 파일 수정 시 gofmt/goimports 적용 필요' >&2"
        }]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [{
          "type": "command",
          "command": "token-monitor status --current --compact 2>/dev/null || true"
        }]
      }
    ],
    "Stop": [
      {
        "hooks": [{
          "type": "command",
          "command": "token-monitor query --current --json 2>/dev/null || true"
        }]
      }
    ]
  }
}
```

**추가 Hook 제안:**

| Hook 이벤트 | 목적 | 동작 |
|-------------|------|------|
| `PostToolUse[Edit*.go]` | Go 파일 수정 후 | `goimports` 자동 실행 |
| `PostToolUse[Bash(make)]` | 빌드 후 | 빌드 결과 요약 → 에러 시 관련 파일 컨텍스트 제공 |
| `Stop` | 세션 종료 시 | 변경 사항 요약 + 토큰 사용량 리포트 |
| `SessionStart` | 세션 시작 시 | `git status` + 최근 커밋 + 진행 중 Jira 티켓 표시 |

### 3.3 settings.local.json 강화

```json
{
  "permissions": {
    "allow": [
      "Bash(python3:*)",
      "Bash(go:*)",
      "Bash(make:*)",
      "Bash(golangci-lint:*)",
      "Bash(goimports:*)",
      "Bash(gofmt:*)",
      "Bash(git:*)",
      "Bash(gh pr:*)",
      "Bash(gh issue:*)",
      "Bash(chainbench:*)",
      "Bash(token-monitor:*)",
      "Bash(curl:*)",
      "Bash(jq:*)",
      "WebSearch",
      "mcp__chainbench__*",
      "mcp__token-monitor__*",
      "mcp__stablenet-build__*",
      "mcp__codebase-memory__*"
    ]
  }
}
```

---

## 4. Phase 2 — MCP 서버 에코시스템

### 4.1 stablenet-build MCP 서버 (신규 개발)

go-stablenet 빌드/테스트/분석을 위한 전용 MCP 서버

```
stablenet-build-mcp/
├── src/
│   ├── index.ts
│   └── tools/
│       ├── build.ts          # make gstable, make all
│       ├── test.ts           # 패키지별 테스트 실행 & 결과 파싱
│       ├── lint.ts           # golangci-lint 실행 & 결과 구조화
│       ├── impact.ts         # 변경 파일 → 영향 패키지 분석
│       ├── coverage.ts       # 테스트 커버리지 분석
│       ├── benchmark.ts      # Go 벤치마크 실행 & 비교
│       └── dependency.ts     # 패키지 의존성 그래프
├── package.json
└── tsconfig.json
```

**제공 도구 (10개):**

| 도구 | 입력 | 출력 |
|------|------|------|
| `stablenet_build` | target (gstable\|all\|genesis_generator) | 빌드 성공/실패 + 에러 파싱 |
| `stablenet_test` | package, test_name, short, verbose | 테스트 결과 구조화 (pass/fail/skip 수, 실패 상세) |
| `stablenet_test_coverage` | package | 커버리지 % + 미커버 라인 |
| `stablenet_lint` | package, fix | 린트 결과 (severity, file, line, rule) |
| `stablenet_impact_analysis` | changed_files[] | 영향 패키지 목록 + 관련 테스트 파일 |
| `stablenet_dependency_graph` | package | 의존성 트리 (imports/imported-by) |
| `stablenet_benchmark` | package, bench_name, count | 벤치마크 결과 (ns/op, B/op, allocs/op) |
| `stablenet_benchmark_compare` | before_commit, after_commit, package | 벤치마크 전후 비교 |
| `stablenet_go_vet` | package | go vet 결과 |
| `stablenet_build_status` | — | 현재 빌드 상태, 바이너리 버전, 최근 빌드 시간 |

### 4.2 codebase-memory MCP 서버 (적용)

> 레퍼런스: `codebase-memory-mcp` — 28M LOC/3분 인덱싱, 99.2% 토큰 감소

go-stablenet (781 파일)의 코드 그래프를 인덱싱하여 Claude가 매 세션마다 전체 프로젝트를 재탐색하지 않도록 함.

**적용 방법:**
```bash
# 설치
npm install -g @pinkpixel/codebase-memory-mcp

# .mcp.json에 등록
{
  "mcpServers": {
    "codebase-memory": {
      "command": "npx",
      "args": ["-y", "@pinkpixel/codebase-memory-mcp"],
      "env": {
        "CODEBASE_DIR": "/path/to/go-stablenet"
      }
    }
  }
}
```

**기대 효과:**
- 매 세션 시작 시 프로젝트 구조 파악 시간: ~30초 → ~2초
- 컨텍스트 토큰 사용량: ~50K → ~500 tokens (99% 감소)
- 함수/타입 탐색 정확도 향상

### 4.3 MCP 서버 통합 구성

`.mcp.json` 최종 구성:

```json
{
  "mcpServers": {
    "chainbench": {
      "command": "chainbench-mcp"
    },
    "token-monitor": {
      "command": "token-monitor",
      "args": ["serve", "--stdio"]
    },
    "stablenet-build": {
      "command": "node",
      "args": ["packages/claude-ai/mcp-servers/stablenet-build/dist/index.js"],
      "env": {
        "STABLENET_DIR": "."
      }
    },
    "codebase-memory": {
      "command": "npx",
      "args": ["-y", "@pinkpixel/codebase-memory-mcp"]
    }
  }
}
```

---

## 5. Phase 3 — 외부 시스템 통합

### 5.1 Jira 통합

**목적:** Jira 티켓 ↔ 코드 변경 자동 연결

**방법 A — Atlassian MCP 플러그인 활용**
```
# 이미 사용 가능한 MCP 도구:
mcp__plugin_atlassian_atlassian__authenticate
```

**방법 B — 커스텀 Jira MCP 서버 (Go 또는 TypeScript)**

| 도구 | 기능 |
|------|------|
| `jira_get_ticket` | 티켓 상세 조회 (제목, 설명, 상태, 담당자) |
| `jira_list_sprint` | 현재 스프린트 티켓 목록 |
| `jira_update_status` | 티켓 상태 변경 (In Progress, In Review, Done) |
| `jira_add_comment` | 작업 진행 상황/PR 링크 코멘트 추가 |
| `jira_link_pr` | PR ↔ 티켓 연결 |
| `jira_get_my_tickets` | 나에게 할당된 티켓 목록 |

**워크플로우 자동화:**
```
/stablenet-jira STNET-123
  → 티켓 내용 조회
  → 관련 코드 영역 분석 (impact analysis)
  → 브랜치 생성 (feature/STNET-123-xxx)
  → 구현 계획 제안
  → 티켓 상태 "In Progress"로 변경
```

### 5.2 Confluence 통합

**목적:** 기술 문서 자동 생성/업데이트

| 도구 | 기능 |
|------|------|
| `confluence_search` | 키워드로 문서 검색 |
| `confluence_get_page` | 페이지 내용 조회 |
| `confluence_create_page` | 새 문서 생성 (하드포크 스펙, 기술 분석 등) |
| `confluence_update_page` | 문서 업데이트 |

**활용 시나리오:**
- 하드포크 구현 후 → Confluence에 기술 스펙 자동 생성
- 코드 리뷰 결과 → Confluence 기술 노트로 저장
- 시스템 컨트랙트 변경 → 아키텍처 문서 자동 업데이트

### 5.3 Slack 통합

**목적:** 개발 이벤트 알림 및 팀 커뮤니케이션

| 도구 | 기능 |
|------|------|
| `slack_notify` | 채널에 메시지 전송 |
| `slack_thread_reply` | 스레드에 답글 |
| `slack_get_channel_messages` | 채널 메시지 조회 |

**자동 알림 시나리오:**
- 빌드 실패 시 → `#stablenet-dev` 채널 알림
- PR 생성 시 → `#code-review` 채널 알림
- 하드포크 관련 변경 시 → `#stablenet-releases` 알림
- 체인 테스트 실패 시 → `#stablenet-qa` 알림

### 5.4 GitHub Enhanced 통합

기존 `gh` CLI 확장:

| 커맨드 | 기능 |
|--------|------|
| `/stablenet-pr` | Jira 티켓 기반 PR 생성 (자동 제목/설명) |
| `/stablenet-review` | PR 코드 리뷰 (REVIEW_GUIDE.md + CODE_CONVENTION.md 기반) |
| `/stablenet-ci-check` | CI 상태 확인 + 실패 분석 |

---

## 6. Phase 4 — 지능형 워크플로우 자동화

### 6.1 End-to-End 개발 워크플로우

```
┌──────────────────────────────────────────────────────────────┐
│  /stablenet-workflow STNET-123                                │
│                                                               │
│  Step 1: Context Loading                                      │
│  ├─ Jira 티켓 STNET-123 조회                                  │
│  ├─ codebase-memory에서 관련 심볼 검색                         │
│  └─ 관련 PR/이슈 히스토리 조회                                  │
│                                                               │
│  Step 2: Planning                                             │
│  ├─ 영향 분석 (impact analysis)                                │
│  ├─ 구현 계획 생성                                             │
│  └─ 테스트 계획 생성                                            │
│                                                               │
│  Step 3: Implementation                                       │
│  ├─ 브랜치 생성: feature/STNET-123-xxx                         │
│  ├─ 코드 구현 (CODE_CONVENTION.md 준수)                        │
│  ├─ goimports/gofmt 자동 적용                                  │
│  └─ 단위 테스트 작성                                            │
│                                                               │
│  Step 4: Verification                                         │
│  ├─ make gstable (빌드 검증)                                   │
│  ├─ make lint (린트 검증)                                      │
│  ├─ go test ./affected/packages/... (테스트)                   │
│  └─ [선택] chainbench 체인 테스트                               │
│                                                               │
│  Step 5: Delivery                                             │
│  ├─ Conventional Commit 메시지 생성                            │
│  ├─ PR 생성 (Jira 티켓 연결)                                   │
│  ├─ Jira 상태 → "In Review" 변경                              │
│  └─ Slack #code-review 알림                                    │
│                                                               │
│  Step 6: Reporting                                            │
│  ├─ 토큰 사용량 리포트                                         │
│  ├─ 변경 요약 (파일, 라인 수, 테스트 결과)                       │
│  └─ Confluence 기술 노트 (필요 시)                              │
└──────────────────────────────────────────────────────────────┘
```

### 6.2 하드포크 워크플로우

go-stablenet 특화 — 하드포크 추가는 복잡한 다단계 프로세스:

```
/stablenet-hardfork-workflow "Cheongdam"

  Step 1: Spec Review
  ├─ SYSTEM_CONTRACT_FLOW.md 참조
  ├─ 기존 하드포크(Applepie, Anzeon, Boho) 패턴 분석
  └─ 체크리스트 생성 (7단계)

  Step 2: Contract Preparation
  ├─ 컨트랙트 컴파일 → bytecode 추출
  ├─ systemcontracts/artifacts/v3/ 저장
  └─ contracts.go에 go:embed 등록

  Step 3: Config Integration
  ├─ params/config.go에 하드포크 필드 추가
  ├─ params/config_wbft.go에 Upgrade 정의
  ├─ CollectUpgrades()에 등록
  └─ 메인넷/테스트넷 블록 번호 설정

  Step 4: Verification
  ├─ 빌드 검증
  ├─ 기존 테스트 통과 확인
  ├─ chainbench로 로컬 체인 테스트
  │   ├─ chainbench init --profile default
  │   ├─ chainbench start
  │   ├─ 하드포크 블록 도달 대기
  │   ├─ 시스템 컨트랙트 업그레이드 확인 (RPC)
  │   └─ 합의 정상 동작 확인
  └─ genesis-updater 실행 (해시 재계산)

  Step 5: Documentation
  ├─ CLAUDE_DEV_GUIDE.md 업데이트
  ├─ Confluence 하드포크 스펙 문서 생성
  └─ PR 생성 (체크리스트 포함)
```

### 6.3 디버깅 워크플로우

```
/stablenet-debug "합의가 블록 1000에서 멈춤"

  Step 1: Information Gathering
  ├─ chainbench status → 각 노드 상태 확인
  ├─ chainbench node rpc 1 eth_blockNumber → 블록 높이
  ├─ chainbench log search "error\|panic\|fatal" → 에러 로그
  └─ chainbench log timeline → 합의 이벤트 타임라인

  Step 2: Diagnosis
  ├─ WBFT 합의 플로우 분석 (CLAUDE_DEV_GUIDE.md §13-17)
  ├─ 로그 패턴 매칭 → 알려진 이슈 탐색
  ├─ 코드 경로 추적 (codebase-memory 활용)
  └─ 가설 수립

  Step 3: Root Cause Analysis
  ├─ 관련 코드 심층 분석
  ├─ 재현 시나리오 구성
  └─ 수정 방안 제안

  Step 4: Fix & Verify
  ├─ 코드 수정
  ├─ 빌드 & 테스트
  ├─ chainbench로 재현 & 검증
  └─ 결과 리포트
```

---

## 7. 레퍼런스 프로젝트별 적용 전략

### 7.1 즉시 적용 가능 (High Value, Low Effort)

| 레퍼런스 | 적용 방법 | 기대 효과 |
|---------|----------|----------|
| **codebase-memory-mcp** | npm 설치 → .mcp.json 등록 | 99% 토큰 감소, 코드 탐색 속도 향상 |
| **token-monitor** | ✅ 이미 구현됨 → hooks로 통합 | 실시간 토큰 추적 |
| **chainbench** | ✅ 이미 구현됨 → 워크플로우 커맨드와 연결 | 체인 런타임 검증 자동화 |
| **context7** | .mcp.json에 추가 | Go/geth 라이브러리 문서 최신화 |
| **superpowers (TDD 패턴)** | hooks + commands 패턴 차용 | 테스트 주도 개발 워크플로우 |

### 7.2 커스텀 구현 필요 (High Value, Medium Effort)

| 레퍼런스 | 차용할 패턴 | 구현 내용 |
|---------|-----------|----------|
| **arc-reactor (Wave 패턴)** | 병렬 실행 + 체크포인트 | 멀티 패키지 빌드/테스트 병렬 실행 |
| **oh-my-claudecode (Git 트레일러)** | 커밋 메타데이터 | AI 작업 추적 (Co-Authored-By, Jira 티켓) |
| **agent-forge (복잡도 게이트)** | 태스크 복잡도 분류 | 간단 수정 vs 하드포크 수준의 워크플로우 자동 선택 |
| **kamar-taj (4레이어 분류)** | 문제 분류 체계 | 버그/기능/리팩토링/하드포크 자동 분류 |
| **get-shit-done (28 skills)** | 스킬 패턴 | go-stablenet 특화 스킬 라이브러리 |

### 7.3 장기 적용 (High Value, High Effort)

| 레퍼런스 | 적용 방향 | 시기 |
|---------|----------|------|
| **vestige (FSRS 메모리)** | 세션 간 학습 지속 — 자주 발생하는 이슈 패턴 학습 | Phase 4+ |
| **A2A 프로토콜** | 다중 AI 에이전트 협업 (코드 생성 + 리뷰 + 테스트) | Phase 4+ |
| **Atlassian MCP 통합** | Jira/Confluence 완전 자동화 | Phase 3 |
| **EverMemOS** | 장기 프로젝트 지식 그래프 | Phase 4+ |

---

## 8. Best Practice: 추천 시스템 구성도

### 8.1 파일 구조 (최종)

```
packages/claude-ai/
├── CLAUDE.md                           # 프로젝트 컨텍스트 (기존, 확장)
├── README.md                           # 설치/사용 가이드 (기존, 확장)
├── install.sh                          # 원격 설치 (기존, 확장)
├── install-local.sh                    # 로컬 설치 (기존)
├── uninstall.sh                        # 제거 (기존)
│
├── .claude/
│   ├── settings.local.json             # 확장된 권한 설정
│   │
│   ├── commands/                       # 슬래시 커맨드 (기존 1개 → 7개)
│   │   ├── stablenet-review-code.md    # ✅ 기존
│   │   ├── stablenet-build.md          # 🆕 빌드 & 검증
│   │   ├── stablenet-test.md           # 🆕 테스트 실행
│   │   ├── stablenet-lint.md           # 🆕 린트 & 포맷
│   │   ├── stablenet-impact.md         # 🆕 변경 영향 분석
│   │   ├── stablenet-hardfork.md       # 🆕 하드포크 가이드
│   │   └── stablenet-debug.md          # 🆕 체인 디버깅
│   │
│   ├── hooks.json                      # 🆕 이벤트 기반 자동화
│   │
│   └── docs/                           # 참조 문서 (기존)
│       ├── CLAUDE_DEV_GUIDE.md
│       ├── SYSTEM_CONTRACT_FLOW.md
│       ├── CODE_CONVENTION.md
│       ├── REVIEW_GUIDE.md
│       └── BUILD_SOURCE_FILES.md
│
├── mcp-servers/                        # 🆕 MCP 서버들
│   └── stablenet-build/                # 🆕 빌드/테스트/분석 MCP
│       ├── src/
│       │   ├── index.ts
│       │   └── tools/
│       │       ├── build.ts
│       │       ├── test.ts
│       │       ├── lint.ts
│       │       ├── impact.ts
│       │       ├── coverage.ts
│       │       ├── benchmark.ts
│       │       └── dependency.ts
│       ├── package.json
│       └── tsconfig.json
│
├── mcp.json                            # 🆕 MCP 서버 통합 설정
│
└── claudedocs/                         # 분석/제안 문서
    └── AUTOMATION_SYSTEM_PROPOSAL.md   # 이 문서
```

### 8.2 CLAUDE.md 확장 제안

현재 CLAUDE.md에 추가할 섹션:

```markdown
## Available Tools

### MCP Servers
- **chainbench**: 로컬 체인 샌드박스 (13 도구) — init, start, stop, test, debug
- **token-monitor**: 토큰 사용량 모니터링 (6 도구) — usage, burn rate, billing
- **stablenet-build**: 빌드/테스트/분석 (10 도구) — build, test, lint, impact
- **codebase-memory**: 코드 그래프 인덱싱 — 빠른 심볼 탐색

### Slash Commands
- `/stablenet-review-code` — 코드 분석 & 리뷰
- `/stablenet-build` — 빌드 & 에러 분석
- `/stablenet-test` — 패키지 테스트 & 결과 분석
- `/stablenet-lint` — 린트 & 자동 수정
- `/stablenet-impact` — 변경 영향 분석
- `/stablenet-hardfork` — 하드포크 추가 가이드
- `/stablenet-debug` — 체인 디버깅 & 합의 분석

### Automation Hooks
- PostToolUse: 토큰 사용량 추적
- PostToolUse[*.go]: goimports 자동 적용
- Stop: 세션 요약 리포트
```

### 8.3 통합 워크플로우 예시

#### 일반 개발 (Daily Development)

```bash
# 1. 세션 시작 — 자동으로 git status + Jira 할당 티켓 표시
claude

# 2. 티켓 기반 작업 시작
> /stablenet-impact STNET-456 "GovMinter에 burn limit 추가"
  → 영향 분석 결과 표시
  → 구현 계획 제안

# 3. 구현
> consensus/wbft/core/handler.go 의 handleCommitMsg를 수정해줘
  → [Hook] goimports 자동 적용
  → [Hook] 토큰 사용량 표시

# 4. 검증
> /stablenet-build
  → 빌드 성공 확인
> /stablenet-test consensus/wbft/...
  → 테스트 결과 표시
> /stablenet-lint consensus/wbft/
  → 린트 통과 확인

# 5. 체인 검증 (필요 시)
> /stablenet-debug
  → chainbench init → start → 테스트 → 결과 분석

# 6. 전달
> PR 생성해줘
  → Conventional Commit + PR 자동 생성
  → [Hook] 세션 종료 시 토큰 리포트
```

#### 하드포크 개발 (Major Feature)

```bash
> /stablenet-hardfork "Cheongdam" "GovValidator v3 업그레이드"
  → 7단계 체크리스트 생성
  → Step-by-step 가이드
  → chainbench 검증 포함
  → Confluence 문서 자동 생성
```

---

## 9. 구현 로드맵

### Phase 1: 기반 강화 (1-2주)

| # | 작업 | 우선순위 | 난이도 |
|---|------|---------|--------|
| 1.1 | 슬래시 커맨드 6개 추가 | 🔴 높음 | 낮음 |
| 1.2 | Hooks 설정 (hooks.json) | 🔴 높음 | 낮음 |
| 1.3 | settings.local.json 확장 | 🔴 높음 | 낮음 |
| 1.4 | CLAUDE.md 확장 (도구 문서화) | 🟡 중간 | 낮음 |
| 1.5 | install.sh 업데이트 (새 파일 포함) | 🟡 중간 | 낮음 |

### Phase 2: MCP 에코시스템 (2-3주)

| # | 작업 | 우선순위 | 난이도 |
|---|------|---------|--------|
| 2.1 | stablenet-build MCP 서버 개발 | 🔴 높음 | 중간 |
| 2.2 | codebase-memory-mcp 통합 | 🔴 높음 | 낮음 |
| 2.3 | chainbench MCP 워크플로우 통합 | 🟡 중간 | 낮음 |
| 2.4 | token-monitor hooks 통합 | 🟡 중간 | 낮음 |
| 2.5 | .mcp.json 통합 구성 | 🟡 중간 | 낮음 |

### Phase 3: 외부 시스템 연동 (3-4주)

| # | 작업 | 우선순위 | 난이도 |
|---|------|---------|--------|
| 3.1 | Jira MCP 서버 (또는 Atlassian 플러그인 활용) | 🔴 높음 | 중간 |
| 3.2 | GitHub PR 워크플로우 자동화 | 🔴 높음 | 낮음 |
| 3.3 | Slack 알림 통합 | 🟡 중간 | 중간 |
| 3.4 | Confluence 문서 자동화 | 🟢 낮음 | 중간 |

### Phase 4: 지능형 자동화 (4-6주)

| # | 작업 | 우선순위 | 난이도 |
|---|------|---------|--------|
| 4.1 | End-to-End 워크플로우 커맨드 | 🟡 중간 | 높음 |
| 4.2 | 하드포크 워크플로우 자동화 | 🟡 중간 | 높음 |
| 4.3 | 디버깅 워크플로우 자동화 | 🟡 중간 | 중간 |
| 4.4 | 세션 간 학습 메모리 | 🟢 낮음 | 높음 |

---

## 10. 리스크 및 고려사항

### 10.1 기술적 리스크

| 리스크 | 영향 | 완화 방안 |
|--------|------|----------|
| MCP 서버 안정성 | 빌드/테스트 도구 실패 시 워크플로우 중단 | 폴백: 직접 Bash 실행으로 대체 가능하도록 설계 |
| 토큰 비용 증가 | MCP 도구 결과 + 컨텍스트 확장 | token-monitor로 모니터링, codebase-memory로 상쇄 |
| go-stablenet 업데이트 | 코드 구조 변경 시 도구 호환성 | BUILD_SOURCE_FILES.md 버전 관리 + 자동 감지 |
| Jira/Slack API 제한 | Rate limit 초과 | 배치 처리 + 캐싱 |

### 10.2 운영 고려사항

| 항목 | 권장 |
|------|------|
| **설치 복잡도** | install.sh 하나로 전체 설정 완료 (현재와 동일) |
| **의존성 관리** | Node.js (MCP 서버), Go (빌드 도구) — 이미 개발 환경에 존재 |
| **팀 온보딩** | README.md에 Quick Start 섹션 + 각 커맨드 사용 예시 |
| **버전 관리** | claude-ai 패키지 자체를 semver로 관리 |
| **보안** | Jira/Slack 토큰은 환경변수로 관리, .mcp.json에 포함 안 함 |

### 10.3 ROI 예상

| 시나리오 | 수동 소요 | 자동화 후 | 절감 |
|---------|----------|----------|------|
| 일반 버그 수정 (코드 분석 → 수정 → 빌드 → 테스트 → PR) | ~2시간 | ~30분 | 75% |
| 하드포크 추가 (7단계 프로세스) | ~2일 | ~4시간 | 75% |
| 코드 리뷰 (패턴 확인 + 컨벤션 체크) | ~1시간 | ~15분 | 75% |
| 체인 디버깅 (노드 로그 분석 + 코드 추적) | ~3시간 | ~45분 | 75% |
| 세션 시작 컨텍스트 로딩 | ~10분 | ~10초 | 98% |

---

## 부록 A: 레퍼런스 프로젝트 적용 매트릭스

| 프로젝트 | 적용 요소 | Phase | 구현 방법 |
|---------|----------|-------|----------|
| codebase-memory-mcp | 코드 인덱싱 | 2 | npm 설치 |
| token-monitor | 토큰 모니터링 | 1 | hooks 통합 (✅ 구현됨) |
| chainbench | 체인 테스트 | 2 | 워크플로우 연결 (✅ 구현됨) |
| context7 | 라이브러리 문서 | 2 | .mcp.json 추가 |
| superpowers | TDD 패턴 | 1 | 커맨드 패턴 차용 |
| arc-reactor | 병렬 실행 | 4 | Wave 패턴 구현 |
| oh-my-claudecode | Git 트레일러 | 1 | hooks에 Co-Authored-By |
| agent-forge | 복잡도 게이트 | 3 | 워크플로우 라우팅 |
| kamar-taj | 분류 체계 | 3 | 커맨드 자동 선택 |
| get-shit-done | 스킬 라이브러리 | 2 | 커맨드 확장 |
| vestige | 세션 메모리 | 4 | 메모리 시스템 |
| Atlassian MCP | Jira/Confluence | 3 | 플러그인 활용 |

## 부록 B: 핵심 성공 지표 (KPI)

| 지표 | 현재 | 목표 (Phase 4 완료) |
|------|------|-------------------|
| 세션 시작 → 작업 시작 시간 | ~10분 | <30초 |
| 버그 수정 전체 사이클 | ~2시간 | <30분 |
| 빌드 에러 분석 시간 | ~15분 | <2분 |
| 하드포크 구현 사이클 | ~2일 | <4시간 |
| 코드 리뷰 시간 | ~1시간 | <15분 |
| 토큰 사용 효율 | 기준선 | 50%+ 절감 |
| 팀 온보딩 시간 | ~1주 | <1일 |

---

*이 제안서는 현재 보유한 모든 도구, 레퍼런스, 인프라를 분석하여 작성되었습니다.*
*Phase 1부터 순차적으로 구현하며, 각 Phase 완료 시 효과를 측정하고 다음 Phase 계획을 조정합니다.*
