# go-stablenet AI 자동화 시스템 고도화 제안서 v2

> **Date**: 2026-04-03
> **Scope**: `packages/claude-ai/` 고도화 → go-stablenet 전체 개발 라이프사이클 자동화
> **Status**: Comprehensive Proposal (v2 — 30+ 레퍼런스 프로젝트 소스코드 직접 분석 기반)
> **Analysis Base**: ai-cli/ 하위 130+ 프로젝트 중 30개 핵심 프로젝트 심층 분석

---

## 목차

1. [현재 상태 분석](#1-현재-상태-분석)
2. [레퍼런스 프로젝트 심층 분석 — 적용 가능한 패턴](#2-레퍼런스-프로젝트-심층-분석)
3. [목표 아키텍처](#3-목표-아키텍처)
4. [Phase 1 — Claude Code 기반 강화 (Commands + Hooks + Skills)](#4-phase-1)
5. [Phase 2 — MCP 서버 에코시스템](#5-phase-2)
6. [Phase 3 — Agent 시스템 구축](#6-phase-3)
7. [Phase 4 — 외부 시스템 통합 (Jira/Confluence/Slack/GitHub)](#7-phase-4)
8. [Phase 5 — 지능형 워크플로우 자동화](#8-phase-5)
9. [Best Practice 종합 — 추천 시스템 구성](#9-best-practice)
10. [구현 로드맵](#10-구현-로드맵)
11. [리스크 및 고려사항](#11-리스크)

---

## 1. 현재 상태 분석

### 1.1 packages/claude-ai/ — 현재 구성

| 구성 요소 | 상태 | 내용 |
|-----------|------|------|
| CLAUDE.md | ✅ | 프로젝트 개요, 코드맵, 컨벤션 참조 |
| `/stablenet-review-code` 커맨드 | ✅ | 구조화된 코드 분석 슬래시 커맨드 (1개) |
| `.claude/docs/` 5개 가이드 | ✅ | DEV_GUIDE, SYSTEM_CONTRACT_FLOW, CODE_CONVENTION, REVIEW_GUIDE, BUILD_SOURCE_FILES |
| `settings.local.json` | ✅ | 기본 권한 3개 (python3, WebSearch, gh pr) |
| **MCP 서버** | ❌ | 없음 |
| **Hooks 시스템** | ❌ | 없음 |
| **Skills** | ❌ | 없음 |
| **Agents** | ❌ | 없음 |
| **외부 시스템 통합** | ❌ | Jira/Confluence/Slack 연동 없음 |

### 1.2 이미 구현된 보조 도구

| 도구 | 상태 | MCP 도구 수 | 핵심 역할 |
|------|------|------------|----------|
| **token-monitor** | ✅ 완료 | 6개 | 실시간 토큰 사용량 모니터링, TUI, 번레이트, 빌링블록 |
| **chainbench** | ✅ 완료 | 13개 | 로컬 체인 샌드박스, WBFT 테스트, 프로파일 기반 |

### 1.3 핵심 Gap 분석

| Gap | 영향 | 해결 레퍼런스 |
|-----|------|-------------|
| 매 세션마다 160패키지 재탐색 | 시간/토큰 낭비 | codebase-memory-mcp (99% 절감) |
| 빌드/테스트 결과 수동 분석 | 느린 피드백 루프 | stablenet-build MCP (자동 파싱) |
| 코드 수정 후 검증 수동 | 품질 리스크 | hooks + agent-forge QR gate |
| 세션 간 학습 단절 | 반복 실수 | memory-bank + kamar-taj handoff |
| Jira↔코드 수동 연결 | 추적성 부재 | Atlassian MCP + hooks |

---

## 2. 레퍼런스 프로젝트 심층 분석

> 30개 프로젝트의 **실제 소스코드**를 직접 탐색하여 go-stablenet에 적용 가능한 패턴을 추출

### 2.1 Plugin/Skill 시스템 (가장 직접적으로 적용 가능)

#### superpowers — 핵심 스킬 라이브러리 (v5.0.7)
- **11개 Skills**: TDD, 체계적 디버깅, 코드 리뷰, 브레인스토밍, 검증, 병렬 에이전트 등
- **핵심 패턴**: 각 스킬에 SKILL.md 프론트매터 (언제 사용, 어떻게 동작)
- **적용**: `/stablenet-debug` → superpowers의 `systematic-debugging` 패턴 차용
- **적용**: `/stablenet-test` → superpowers의 `test-driven-development` 패턴 차용
- **적용**: `verification-before-completion` → 빌드/테스트/린트 완료 확인 후 커밋

#### shuri (all-in-one-claude-code) — 통합 OS (v1.0)
- **26개 Agents** (haiku/sonnet/opus 모델 티어별 라우팅)
- **29개 Skills** (5개 카테고리: dev-workflow, orchestration, design, memory, research)
- **5개 Commands**: help, ignite, setup, shutdown, status
- **핵심 패턴**:
  - **모델 라우팅 테이블**: 빠른 작업→haiku, 실행→sonnet, 판단→opus
  - **자동 스킬 라우팅**: 키워드 감지 → 적합한 스킬 자동 선택
  - **완료 게이트**: 모든 태스크 완료 + 테스트 통과 + 증거 수집 후에만 완료 선언
- **적용**: go-stablenet용 모델 라우팅 (린트→haiku, 구현→sonnet, 합의 로직 리뷰→opus)

#### kamar-taj — Claude 마스터 가이드 플러그인
- **4-Layer 문제 분류 체계**: L1 Knowledge → L2 Tools → L3 Package → L4 Control
- **3가지 엔지니어링 디시플린**: Context Engineering, Prompt Engineering, Harness Engineering
- **25개 Skills** (context-engineering, governance, harness-engineering, workflow-patterns 등)
- **8 Essential Files 패턴**: about-me.md, working-rules.md, plan.md, handoff.md, templates/
- **핵심 패턴**:
  - **파일 중심 작업**: 채팅은 사라짐, 파일만 세션 간 지속
  - **handoff.md**: 세션 종료 시 다음 세션에 넘길 컨텍스트 기록
  - **Safety Hooks**: PreToolUse에서 Bash 안전성 검증, PostToolUse에서 출력 검증
- **적용**: go-stablenet handoff 시스템 — 진행 중 하드포크 작업, 실패한 테스트 등 인수인계

#### get-shit-done — 산업용 오케스트레이션 (23 Agents, 60+ Commands)
- **23개 Agent** (30-50KB 프롬프트): planner, executor, debugger, doc-writer, security-auditor 등
- **60+ Commands**: new-project, plan-phase, execute-phase, debug, verify-work, ship 등
- **5개 Hook JS 파일**: context-monitor, prompt-guard, workflow-guard, statusline, check-update
- **핵심 패턴**:
  - **Atomic Commits**: 태스크별 커밋 + 구조화된 트레일러 (Constraint, Confidence, Scope-risk)
  - **Deviation Handling**: 빠진 기능 자동 감지 (린팅이 필요하면 자동 실행)
  - **State Bridge**: `/tmp/claude-ctx-${session}.json`으로 훅 간 통신
  - **Mandatory Reads**: 각 에이전트가 반드시 읽어야 할 파일 목록 강제
- **적용**: go-stablenet 커밋 트레일러 시스템 — `Confidence: high`, `Scope-risk: consensus`

### 2.2 Agent 오케스트레이션 시스템

#### oh-my-claudecode — 멀티 에이전트 오케스트레이션
- **19개 Agents**: explorer, architect, executor, debugger, verifier 등
- **12개 Hook 이벤트**: SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop 등
- **35+ Skills**: 팀 파이프라인 (plan → prd → exec → verify → fix)
- **핵심 패턴**: 최대 6개 동시 서브에이전트, 증거 기반 검증
- **적용**: go-stablenet 코드 수정 → 빌드 → 테스트 → 검증 파이프라인

#### arc-reactor — 멀티 팀 Wave 오케스트레이션
- **10개 팀 Agent**: Director, Frontend, Backend, QA, Design, DevOps, Security, Product, Architect, Docs
- **Wave 실행**: Wave 1 (병렬) → Checkpoint → Wave 2 (의존) → Quality Gate
- **Feature 분해**: 각 Feature = 수직 슬라이스 (Planning, Frontend, Backend, Database, Tests)
- **적용**: 하드포크 작업 → Wave 1(컨트랙트 컴파일 + 코드 등록 병렬) → Wave 2(빌드+테스트) → Wave 3(체인 검증)

#### everything-claude-code — 엔터프라이즈 에코시스템
- **150+ Skills** (Go, Rust, Python, Kubernetes 등 30+ 도메인)
- **38개 Agents**: code-reviewer, security-reviewer, language-specific reviewers
- **6개 MCP 서버**: GitHub, Context7, Exa, Memory, Playwright, Sequential-thinking
- **15개 Hook 이벤트**: 지속적 학습 관찰, 거버넌스 캡처, 비용 추적
- **적용**: Go 전용 스킬 + 보안 리뷰 에이전트 + 비용 추적 훅

#### multi-agent-shogun — 군사 계층 구조 (戦国風)
- **계층**: Shogun → Karo(家老) → Ashigaru(足軽) + Gunshi(軍師)
- **Bloom Taxonomy 라우팅**: L1-L3(단순)→빠른 모델, L4-L6(전략)→강력한 모델
- **이벤트 기반 통신**: YAML mailbox + inotifywait 기반 웨이크 시그널
- **적용**: 합의 로직 변경 → L6(opus) / 린트 수정 → L1(haiku)

### 2.3 MCP 및 컨텍스트 도구

#### codebase-memory-mcp — 코드 그래프 인덱싱
- **Pure C** (제로 의존성, 단일 바이너리)
- **14개 MCP 도구**: search, trace, architecture, impact, cypher, dead_code, http_routes 등
- **성능**: Linux 커널 28M LOC/3분, 서브밀리초 쿼리, 120x 토큰 감소
- **66개 언어 지원** (Go 포함, tree-sitter 기반 AST)
- **적용**: go-stablenet 781 파일 인덱싱 → `trace` 로 함수 호출 체인 추적, `impact`로 변경 영향 분석

#### context-mode — 98% 토큰 감소
- **6개 MCP 도구**: ctx_execute, ctx_batch_execute, ctx_index, ctx_search, ctx_fetch_and_index
- **FTS5 + BM25 랭킹**: SQLite 기반 풀텍스트 검색
- **Hook 기반 라우팅**: PreToolUse에서 Bash/Read/Grep 호출을 샌드박스로 리다이렉트
- **적용**: 대용량 Go 파일 읽기 시 315KB → 5.4KB (98% 감소)

#### memory-bank — 세션 간 지식 그래프
- **9개 MCP 도구**: search, search_facts, ask_avatar, trace_fact, explore_graph, cross_project_insights
- **Fact 시스템**: decision, preference, pattern, knowledge, constraint 카테고리
- **통합 규칙**: DUPLICATE(병합), CONTRADICTION(교체), EVOLUTION(업데이트)
- **적용**: "왜 Merkle tree 대신 이 구조를 선택했는가?" → 의사결정 추적

#### vibranium — 피처 재사용 라이브러리
- **5개 MCP 도구**: search, get, create, update, list_children
- **피처 라이프사이클**: draft → registered → implementing → merged → reusable
- **적용**: 공통 패턴(시스템 컨트랙트 배포, 하드포크 설정) 재사용 가능한 스펙으로 등록

### 2.4 워크플로우 자동화 도구

#### agent-forge — 복잡도 게이트 + QR Gate
- **복잡도 티어**: Micro(1-2 파일), Standard(3-10), Full(10+) → 티어별 검증 수준 자동 결정
- **QR Gate**: 품질 리뷰 완료 전까지 커밋 차단 (PreToolUse hook)
- **Go Runtime**: BubbleTea TUI, tmux 세션, 보안 샌드박스 (readonly/restricted/standard/full)
- **적용**: 합의 로직 수정 = Full 티어 → 반드시 QR Gate + 체인 테스트 통과 필요

#### claude-task-master — PRD → 태스크 분해
- **MCP 서버**: task-master-ai
- **PRD 파서**: `.taskmaster/docs/prd.md` → 계층적 태스크 자동 생성 (1.1, 1.2, 1.2.1)
- **복잡도 분석**: 파일 영향 범위, 의존성, 아키텍처 리스크 스코어링
- **적용**: 하드포크 PRD → 자동 태스크 분해 → 의존성 그래프 → 실행 순서 결정

#### commitflow — Git 커밋 분석 + 업스트림 동기화 (Go)
- **커밋 분류**: AI 기반 diff 분석, 범위/영향도 분류
- **업스트림 추적**: 로컬 vs 업스트림 브랜치 비교, 선택적 패치 적용
- **적용**: go-ethereum 업스트림 변경사항 추적 → StableNet에 선택적 머지

#### spec-kit — 사양 주도 개발
- **워크플로우**: `/speckit.specify` → `/speckit.plan` → `/speckit.tasks`
- **Constitutional 준수 검사**: 스펙 → 구현 계획 → 실행 태스크 변환 시 일관성 검증
- **적용**: 하드포크/시스템 컨트랙트 스펙 → 구현 계획 → 체크리스트 자동 생성

### 2.5 전문 도구

#### claude-code (공식 레퍼런스) — 플러그인 시스템
- **Plugin 구조**: `.claude-plugin/plugin.json` + commands/ + agents/ + skills/ + hooks/
- **보안 hooks**: XSS, command injection, eval 등 9개 보안 패턴 탐지
- **feature-dev**: 7-phase 피처 개발 라이프사이클
- **적용**: go-stablenet을 공식 플러그인 형태로 패키징

#### openclaw — 오픈 플러그인 시스템
- **멀티 채널**: WhatsApp, Telegram, Slack, Discord, Signal
- **coding-agent 스킬**: 백그라운드 프로세스 관리, PTY vs non-PTY 실행
- **적용**: Slack 알림 채널 통합, 백그라운드 빌드 프로세스 관리

#### SuperClaude_Framework — 30 커맨드 + 20 에이전트
- **7가지 행동 모드**: Brainstorming, Deep Research, Orchestration, Token-Efficiency 등
- **8개 MCP 서버**: Tavily, Context7, Sequential-Thinking, Serena, Playwright 등
- **적용**: Token-Efficiency 모드로 대규모 코드베이스 탐색

---

## 3. 목표 아키텍처

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         Developer (Claude Code CLI)                       │
│                                                                           │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌────────────────────┐  │
│  │  Commands    │ │   Hooks     │ │   Skills    │ │    Agents          │  │
│  │  (7개)       │ │  (5이벤트)   │ │  (8개)      │ │   (5개 전문가)     │  │
│  └──────┬──────┘ └──────┬──────┘ └──────┬──────┘ └────────┬───────────┘  │
│         │               │               │                  │              │
│  ┌──────┴───────────────┴───────────────┴──────────────────┴───────────┐  │
│  │                      MCP Server Layer (6개)                          │  │
│  │  ┌──────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────────┐  │  │
│  │  │stablenet     │ │ chainbench │ │  token     │ │  codebase      │  │  │
│  │  │ -build       │ │ (✅기존)   │ │  monitor   │ │  -memory       │  │  │
│  │  │ (🆕 10도구) │ │ (13도구)   │ │  (✅기존)  │ │  (14도구)      │  │  │
│  │  └──────────────┘ └────────────┘ └────────────┘ └────────────────┘  │  │
│  │  ┌──────────────┐ ┌────────────┐                                    │  │
│  │  │  context7    │ │  atlassian │                                    │  │
│  │  │  (라이브러리) │ │  (Jira등)  │                                    │  │
│  │  └──────────────┘ └────────────┘                                    │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │  State Management (kamar-taj + agent-forge 패턴)                    │  │
│  │  plan.md │ handoff.md │ .state/memory.json │ QR-gate │ complexity  │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │  Automation Workflows                                                │  │
│  │  Jira Ticket → Branch → Implement → Build → Test → Chain Verify   │  │
│  │  → QR Gate → PR → Code Review → Slack Notify → Merge              │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Phase 1 — Claude Code 기반 강화

### 4.1 슬래시 커맨드 (1개 → 7개)

| 커맨드 | 참조 패턴 | 기능 |
|--------|----------|------|
| `/stablenet-review-code` | ✅ 기존 | 구조화된 코드 분석 |
| `/stablenet-build` | agent-forge deviation | `make gstable` → 에러 파싱 → 수정 제안 → 재빌드 |
| `/stablenet-test` | superpowers TDD | 패키지/파일 테스트 → 실패 분석 → 테이블 드리븐 테스트 제안 |
| `/stablenet-lint` | gsd deviation | `make lint` → CODE_CONVENTION.md 대조 → 자동 수정 |
| `/stablenet-impact` | codebase-memory | 변경 파일 → 영향 패키지 → 관련 테스트 → 빌드 경로 |
| `/stablenet-hardfork` | spec-kit specify | SYSTEM_CONTRACT_FLOW.md 기반 7단계 체크리스트 + chainbench 검증 |
| `/stablenet-debug` | superpowers debugging | chainbench 로그 + 노드 RPC + WBFT 플로우 분석 |

### 4.2 Hooks 시스템 (kamar-taj + agent-forge + gsd 패턴 차용)

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [{
          "type": "command",
          "command": "echo '📊 프로젝트 상태:' && git status --short && git log --oneline -3 && echo '---' && token-monitor status --current --compact 2>/dev/null || true"
        }]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Bash(git commit)",
        "hooks": [{
          "type": "command",
          "command": "echo '⚠️ 커밋 전 확인: make lint && make test-short 실행 여부' >&2"
        }]
      },
      {
        "matcher": "Bash(git push --force)",
        "hooks": [{
          "type": "command",
          "command": "echo '🚫 force push 차단. --force-with-lease 사용 권장' >&2 && exit 1"
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
          "command": "echo '📋 세션 요약:' && git diff --stat && token-monitor query --current --json 2>/dev/null || true"
        }]
      }
    ]
  }
}
```

### 4.3 Skills 디렉토리 (superpowers + shuri 패턴 차용)

```
.claude/skills/
├── build-and-verify/
│   └── SKILL.md          # 빌드 → 에러 분석 → 수정 → 재빌드 루프
├── test-driven-dev/
│   └── SKILL.md          # Red-Green-Refactor for Go
├── systematic-debugging/
│   └── SKILL.md          # 로그 → 가설 → 코드 추적 → 검증
├── consensus-analysis/
│   └── SKILL.md          # WBFT 합의 플로우 분석 전문
├── hardfork-guide/
│   └── SKILL.md          # 7단계 하드포크 추가 가이드
├── impact-analysis/
│   └── SKILL.md          # 변경 영향 분석 + 관련 테스트 식별
├── chain-verification/
│   └── SKILL.md          # chainbench 통합 체인 검증
└── session-handoff/
    └── SKILL.md          # 세션 종료 시 handoff.md 작성 (kamar-taj)
```

**SKILL.md 프론트매터 예시 (superpowers 형식):**

```markdown
---
name: build-and-verify
description: go-stablenet 빌드 실행, 에러 분석, 자동 수정 제안. "빌드", "build", "컴파일", "make" 키워드 시 자동 활성화
---

# Build and Verify

## When to Use
- 코드 수정 후 빌드 검증 필요 시
- CI 빌드 실패 원인 분석 시
- 새 패키지/파일 추가 후 빌드 확인 시

## Workflow
1. `make gstable` 실행 (또는 `make all`)
2. 에러 발생 시:
   a. 에러 메시지 파싱 (파일:라인:컬럼)
   b. 해당 파일 읽기 + 주변 컨텍스트
   c. 수정 제안 (CODE_CONVENTION.md 참조)
   d. 수정 적용 후 재빌드
3. 성공 시: 바이너리 버전 확인 + 변경 파일 대상 테스트 실행
4. 최종: 빌드 결과 요약 (성공/실패, 소요 시간, 바이너리 크기)
```

### 4.4 settings.local.json 확장

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
      "Bash(gh:*)",
      "Bash(chainbench:*)",
      "Bash(token-monitor:*)",
      "Bash(curl:*)",
      "Bash(jq:*)",
      "WebSearch",
      "mcp__chainbench__*",
      "mcp__token-monitor__*",
      "mcp__stablenet-build__*",
      "mcp__codebase-memory__*",
      "mcp__context7__*"
    ]
  }
}
```

---

## 5. Phase 2 — MCP 서버 에코시스템

### 5.1 stablenet-build MCP 서버 (신규 개발)

```
mcp-servers/stablenet-build/
├── src/
│   ├── index.ts
│   └── tools/
│       ├── build.ts          # make gstable/all + 에러 파싱
│       ├── test.ts           # 패키지별 테스트 + 결과 구조화
│       ├── lint.ts           # golangci-lint + 규칙별 분류
│       ├── impact.ts         # go list -deps 기반 영향 분석
│       ├── coverage.ts       # go test -coverprofile + 미커버 라인
│       ├── benchmark.ts      # go test -bench + 결과 비교
│       └── dependency.ts     # 패키지 의존성 그래프
├── package.json
└── tsconfig.json
```

**10개 MCP 도구:**

| 도구 | 입력 | 출력 | 참조 패턴 |
|------|------|------|----------|
| `stablenet_build` | target | 빌드 성공/실패 + 에러 위치 + 제안 | agent-forge deviation |
| `stablenet_test` | package, name | pass/fail/skip + 실패 상세 + 스택트레이스 | superpowers TDD |
| `stablenet_test_coverage` | package | 커버리지 % + 미커버 라인 + 함수별 커버리지 | gsd verify-work |
| `stablenet_lint` | package, fix | severity/file/line/rule + 자동 수정 결과 | gsd deviation |
| `stablenet_impact` | changed_files[] | 영향 패키지 + 관련 테스트 + 빌드 경로 | codebase-memory impact |
| `stablenet_deps` | package | import/imported-by 트리 | codebase-memory trace |
| `stablenet_bench` | package, name | ns/op, B/op, allocs/op | — |
| `stablenet_bench_compare` | before, after | 성능 변화 % | — |
| `stablenet_vet` | package | go vet 결과 | — |
| `stablenet_status` | — | 빌드 상태, 바이너리 버전, 최근 빌드 시간 | — |

### 5.2 외부 MCP 서버 통합

| MCP 서버 | 설치 방법 | 도구 수 | 역할 |
|---------|----------|---------|------|
| **codebase-memory-mcp** | 바이너리 설치 | 14개 | AST 기반 코드 그래프, 함수 추적, 영향 분석 |
| **context7** | npm | 2개 | Go/geth 라이브러리 최신 문서 |
| **chainbench** | ✅ 기존 | 13개 | 로컬 체인 테스트 |
| **token-monitor** | ✅ 기존 | 6개 | 토큰 사용량 |

### 5.3 .mcp.json 통합

```json
{
  "mcpServers": {
    "stablenet-build": {
      "command": "node",
      "args": [".claude/mcp-servers/stablenet-build/dist/index.js"]
    },
    "chainbench": {
      "command": "chainbench-mcp"
    },
    "token-monitor": {
      "command": "token-monitor",
      "args": ["serve", "--stdio"]
    },
    "codebase-memory": {
      "command": "codebase-memory-mcp"
    },
    "context7": {
      "command": "npx",
      "args": ["-y", "@upstash/context7-mcp@latest"]
    }
  }
}
```

---

## 6. Phase 3 — Agent 시스템 구축

> 참조: shuri(26 agents), gsd(23 agents), oh-my-claudecode(19 agents), everything-claude-code(38 agents)

### 6.1 go-stablenet 전용 Agents (5개 — 핵심만)

```
.claude/agents/
├── consensus-expert.md        # WBFT 합의 전문가
├── contract-reviewer.md       # 시스템 컨트랙트 리뷰어
├── build-resolver.md          # 빌드 에러 해결사
├── chain-debugger.md          # 체인 디버깅 전문가
└── hardfork-architect.md      # 하드포크 설계자
```

**Agent 정의 형식 (shuri + gsd 패턴 결합):**

```markdown
---
name: consensus-expert
description: WBFT 합의 엔진 코드 분석, 수정, 검증 전문. consensus/ 패키지 변경 시 자동 활성화
model: opus
tools: Read, Edit, Bash, Grep, Glob, mcp__stablenet-build__*, mcp__chainbench__*, mcp__codebase-memory__*
---

# Consensus Expert Agent

## Role
WBFT(consensus/wbft/) 합의 엔진의 코드 분석, 버그 수정, 새 기능 구현을 전문으로 수행.

## Mandatory Context
반드시 읽어야 할 파일:
- `.claude/docs/CLAUDE_DEV_GUIDE.md` §13-17 (WBFT 내부)
- `.claude/docs/SYSTEM_CONTRACT_FLOW.md` (시스템 컨트랙트 플로우)
- `consensus/wbft/core/handler.go` (핵심 이벤트 루프)

## Approach
1. codebase-memory로 관련 심볼 검색
2. 코드 읽기 전 REVIEW_GUIDE.md의 "WBFT Flow" 섹션 참조
3. 변경 시 반드시:
   a. 영향 분석 (stablenet_impact)
   b. 유닛 테스트 (stablenet_test consensus/wbft/...)
   c. chainbench 합의 테스트 (chainbench_test_run basic/wbft-consensus)
4. 완료 게이트: 모든 테스트 통과 + chainbench 검증 완료

## Safety Rules
- consensus/wbft/core/ 수정 시 반드시 opus 모델 사용
- 합의 로직 변경은 최소 1000 블록 chainbench 테스트 필요
- Bloom Level: L5-L6 (전략/아키텍처 수준)
```

### 6.2 Agent 라우팅 (shuri + shogun 패턴)

| 트리거 키워드 | Agent | 모델 |
|-------------|-------|------|
| consensus, wbft, 합의, validator | consensus-expert | opus |
| systemcontracts, 시스템 컨트랙트, governance | contract-reviewer | opus |
| build, 빌드, make, compile | build-resolver | sonnet |
| debug, 디버그, 로그, crash, panic | chain-debugger | sonnet |
| hardfork, 하드포크, upgrade | hardfork-architect | opus |
| (기타 일반 코드 작업) | (기본 Claude) | sonnet |

---

## 7. Phase 4 — 외부 시스템 통합

### 7.1 Jira 통합

**방법**: Atlassian MCP 플러그인 활용 (이미 `mcp__plugin_atlassian_atlassian__authenticate` 사용 가능)

| 워크플로우 | 자동화 내용 |
|-----------|-----------|
| 티켓 → 개발 시작 | Jira 티켓 조회 → 브랜치 생성 → 상태 "In Progress" |
| 개발 완료 → PR | PR 생성 → Jira에 PR 링크 코멘트 → 상태 "In Review" |
| PR 머지 → 완료 | 상태 "Done" → Slack 알림 |

### 7.2 Slack 통합

| 이벤트 | 알림 채널 | 메시지 |
|--------|----------|--------|
| 빌드 실패 | `#stablenet-dev` | ❌ 빌드 실패: {에러 요약} |
| PR 생성 | `#code-review` | 🔍 PR #{num}: {제목} by {author} |
| 체인 테스트 실패 | `#stablenet-qa` | ⚠️ chainbench 테스트 실패: {테스트명} |
| 하드포크 관련 변경 | `#stablenet-releases` | 🔔 하드포크 관련 코드 변경: {파일 목록} |

### 7.3 GitHub Enhanced

| 커맨드 | 기능 | 참조 |
|--------|------|------|
| `/stablenet-pr` | Jira 티켓 기반 PR 생성 (제목/설명 자동) | oh-my-claudecode |
| `/stablenet-review` | PR 코드 리뷰 (CODE_CONVENTION + REVIEW_GUIDE 기반) | everything-claude-code code-reviewer |
| `/stablenet-ci` | CI 상태 확인 + 실패 분석 | gsd verify-work |

---

## 8. Phase 5 — 지능형 워크플로우 자동화

### 8.1 End-to-End 개발 워크플로우

```
/stablenet-workflow STNET-123

  ┌─ Step 1: Context Loading ───────────────────────────┐
  │ ├─ Jira 티켓 STNET-123 조회 (Atlassian MCP)         │
  │ ├─ codebase-memory에서 관련 심볼 검색                │
  │ └─ handoff.md에서 이전 세션 컨텍스트 로드             │
  └─────────────────────────────────────────────────────┘
           ↓
  ┌─ Step 2: Planning ──────────────────────────────────┐
  │ ├─ 복잡도 분류 (agent-forge 패턴)                    │
  │ │   Micro: 직접 구현                                  │
  │ │   Standard: 영향 분석 → 구현 계획                   │
  │ │   Full: 전체 설계 → Wave 실행 계획                  │
  │ ├─ 영향 분석 (stablenet_impact)                      │
  │ └─ TodoWrite로 태스크 분해                            │
  └─────────────────────────────────────────────────────┘
           ↓
  ┌─ Step 3: Implementation ────────────────────────────┐
  │ ├─ 브랜치 생성: feature/STNET-123-xxx                │
  │ ├─ 코드 구현 (CODE_CONVENTION.md 준수)               │
  │ ├─ [Hook] goimports 자동 적용                        │
  │ ├─ [Hook] 토큰 사용량 추적                           │
  │ └─ 단위 테스트 작성 (superpowers TDD)                │
  └─────────────────────────────────────────────────────┘
           ↓
  ┌─ Step 4: Verification (agent-forge QR Gate) ────────┐
  │ ├─ stablenet_build → 빌드 성공 확인                  │
  │ ├─ stablenet_lint → 린트 통과 확인                   │
  │ ├─ stablenet_test → 관련 테스트 통과 확인            │
  │ ├─ [Full 티어] chainbench 체인 테스트                │
  │ └─ QR Gate 통과 확인                                  │
  └─────────────────────────────────────────────────────┘
           ↓
  ┌─ Step 5: Delivery ──────────────────────────────────┐
  │ ├─ Conventional Commit (gsd 트레일러 포함)            │
  │ │   feat(wbft): add burn limit to GovMinter v3       │
  │ │   Confidence: high | Scope-risk: consensus          │
  │ ├─ PR 생성 (Jira 티켓 연결)                          │
  │ ├─ Jira 상태 → "In Review"                           │
  │ └─ Slack #code-review 알림                            │
  └─────────────────────────────────────────────────────┘
           ↓
  ┌─ Step 6: Session Wrap (kamar-taj handoff) ──────────┐
  │ ├─ handoff.md 업데이트 (다음 세션 인수인계)           │
  │ ├─ 토큰 사용량 리포트                                │
  │ └─ 변경 요약 (파일 수, 테스트 결과)                   │
  └─────────────────────────────────────────────────────┘
```

### 8.2 하드포크 워크플로우 (Wave 실행 — arc-reactor 패턴)

```
/stablenet-hardfork "Cheongdam"

  Wave 1 (병렬 실행):
  ├─ [A] 컨트랙트 컴파일 → bytecode 추출
  ├─ [B] params/config.go 하드포크 필드 설계
  └─ [C] 기존 하드포크(Boho) 패턴 분석

  ── Checkpoint: 모든 산출물 확인 ──

  Wave 2 (순차 실행):
  ├─ systemcontracts/artifacts/v3/ 저장
  ├─ contracts.go에 go:embed 등록
  ├─ upgrade*() 함수 구현
  └─ CollectUpgrades()에 등록

  ── Checkpoint: 빌드 성공 확인 ──

  Wave 3 (검증):
  ├─ 기존 테스트 통과 확인
  ├─ chainbench 로컬 체인 테스트
  │   ├─ chainbench init
  │   ├─ chainbench start
  │   ├─ 하드포크 블록 도달 대기
  │   ├─ 시스템 컨트랙트 업그레이드 확인 (RPC)
  │   └─ 합의 정상 동작 확인
  └─ genesis-updater 실행

  ── Quality Gate: 모든 검증 완료 ──

  Wave 4 (문서화):
  ├─ CLAUDE_DEV_GUIDE.md 업데이트
  ├─ Confluence 하드포크 스펙 생성
  └─ PR 생성 (체크리스트 포함)
```

### 8.3 디버깅 워크플로우 (superpowers systematic-debugging 패턴)

```
/stablenet-debug "합의가 블록 1000에서 멈춤"

  Phase 1: Observe (증상 수집)
  ├─ chainbench_status → 각 노드 상태
  ├─ chainbench_node_rpc → eth_blockNumber 비교
  ├─ chainbench_log_search "error|panic|fatal"
  └─ chainbench_log_timeline → 합의 이벤트 순서

  Phase 2: Hypothesize (가설 수립)
  ├─ WBFT 플로우 참조 (CLAUDE_DEV_GUIDE §13-17)
  ├─ 로그 패턴 → 알려진 이슈 매칭
  ├─ codebase-memory trace → 관련 코드 경로
  └─ 가설 목록 + 검증 방법

  Phase 3: Test (가설 검증)
  ├─ 각 가설에 대해:
  │   ├─ 코드 읽기 (관련 함수)
  │   ├─ 재현 시도 (chainbench)
  │   └─ 확인/반증
  └─ 근본 원인 특정

  Phase 4: Fix & Verify (수정 및 검증)
  ├─ 코드 수정
  ├─ stablenet_build → 빌드
  ├─ stablenet_test → 관련 테스트
  ├─ chainbench → 체인 재현 → 수정 확인
  └─ 결과 리포트
```

---

## 9. Best Practice 종합 — 추천 시스템 구성

### 9.1 최종 파일 구조

```
packages/claude-ai/
├── CLAUDE.md                              # 프로젝트 컨텍스트 (확장)
├── README.md                              # 설치/사용 가이드 (확장)
├── install.sh / install-local.sh          # 설치 스크립트 (확장)
├── uninstall.sh                           # 제거
│
├── .claude/
│   ├── settings.local.json                # 확장된 권한 (15+)
│   │
│   ├── commands/                          # 슬래시 커맨드 (7개)
│   │   ├── stablenet-review-code.md       # ✅ 기존
│   │   ├── stablenet-build.md             # 🆕
│   │   ├── stablenet-test.md              # 🆕
│   │   ├── stablenet-lint.md              # 🆕
│   │   ├── stablenet-impact.md            # 🆕
│   │   ├── stablenet-hardfork.md          # 🆕
│   │   └── stablenet-debug.md             # 🆕
│   │
│   ├── skills/                            # 🆕 스킬 (8개)
│   │   ├── build-and-verify/SKILL.md
│   │   ├── test-driven-dev/SKILL.md
│   │   ├── systematic-debugging/SKILL.md
│   │   ├── consensus-analysis/SKILL.md
│   │   ├── hardfork-guide/SKILL.md
│   │   ├── impact-analysis/SKILL.md
│   │   ├── chain-verification/SKILL.md
│   │   └── session-handoff/SKILL.md
│   │
│   ├── agents/                            # 🆕 전문 에이전트 (5개)
│   │   ├── consensus-expert.md
│   │   ├── contract-reviewer.md
│   │   ├── build-resolver.md
│   │   ├── chain-debugger.md
│   │   └── hardfork-architect.md
│   │
│   ├── hooks.json                         # 🆕 이벤트 기반 자동화
│   │
│   └── docs/                              # 참조 문서 (기존 유지)
│       ├── CLAUDE_DEV_GUIDE.md
│       ├── SYSTEM_CONTRACT_FLOW.md
│       ├── CODE_CONVENTION.md
│       ├── REVIEW_GUIDE.md
│       └── BUILD_SOURCE_FILES.md
│
├── mcp-servers/                           # 🆕 MCP 서버
│   └── stablenet-build/
│       ├── src/index.ts
│       ├── src/tools/*.ts
│       ├── package.json
│       └── tsconfig.json
│
├── templates/                             # 🆕 세션 상태 템플릿 (kamar-taj)
│   ├── plan.md                            # 작업 계획 템플릿
│   └── handoff.md                         # 세션 인수인계 템플릿
│
├── .mcp.json                              # 🆕 MCP 서버 통합 설정
│
└── claudedocs/                            # 분석 문서
    ├── AUTOMATION_SYSTEM_PROPOSAL.md      # v1
    └── AUTOMATION_SYSTEM_PROPOSAL_v2.md   # 이 문서
```

### 9.2 레퍼런스 프로젝트 적용 매트릭스 (최종)

| 레퍼런스 | 차용 요소 | 적용 위치 | Phase | 우선순위 |
|---------|----------|----------|-------|---------|
| **superpowers** | TDD, debugging, verification 스킬 패턴 | skills/ | 1 | 🔴 |
| **shuri** | 모델 라우팅, 자동 스킬 선택, 완료 게이트 | agents/, CLAUDE.md | 1-3 | 🔴 |
| **kamar-taj** | handoff.md, 4-Layer 분류, Safety Hooks | templates/, hooks | 1 | 🔴 |
| **agent-forge** | 복잡도 게이트, QR Gate, 상태 추적 | hooks, skills/ | 1-2 | 🔴 |
| **gsd** | Atomic Commits 트레일러, Deviation Handling | hooks, CLAUDE.md | 1 | 🟡 |
| **codebase-memory-mcp** | 코드 인덱싱 (14도구, 99% 토큰 감소) | .mcp.json | 2 | 🔴 |
| **context-mode** | 98% 컨텍스트 감소, FTS5 검색 | .mcp.json | 2 | 🟡 |
| **context7** | 라이브러리 문서 최신화 | .mcp.json | 2 | 🟡 |
| **oh-my-claudecode** | 멀티 에이전트 파이프라인, 증거 기반 검증 | agents/ | 3 | 🟡 |
| **arc-reactor** | Wave 병렬 실행 + Checkpoint | hardfork workflow | 5 | 🟡 |
| **multi-agent-shogun** | Bloom Taxonomy 라우팅, 역할 분리 | agents/ 모델 선택 | 3 | 🟢 |
| **everything-claude-code** | 150+ 스킬, 보안 리뷰, 비용 추적 | 장기 확장 참조 | 3-5 | 🟢 |
| **claude-task-master** | PRD → 태스크 분해, 복잡도 분석 | hardfork workflow | 5 | 🟢 |
| **commitflow** | 업스트림 동기화, 커밋 분류 | git workflow | 4 | 🟢 |
| **spec-kit** | 사양 주도 개발, 체크리스트 생성 | hardfork workflow | 5 | 🟢 |
| **memory-bank** | 세션 간 지식 그래프, 의사결정 추적 | 장기 | 5+ | 🟢 |
| **vibranium** | 피처 재사용 라이브러리 | 장기 | 5+ | 🟢 |
| **openclaw** | 멀티 채널 알림, 백그라운드 프로세스 | Slack 통합 | 4 | 🟢 |
| **clawflows** | AGENTS.md 자동 동기화, 워크플로우 레지스트리 | 장기 | 5+ | 🟢 |

### 9.3 CLAUDE.md 확장 내용

```markdown
## Available Tools

### MCP Servers
- **stablenet-build** (10도구): 빌드, 테스트, 린트, 영향 분석, 커버리지, 벤치마크
- **chainbench** (13도구): 로컬 체인 init/start/stop/test/debug
- **token-monitor** (6도구): 토큰 사용량, 번레이트, 빌링블록
- **codebase-memory** (14도구): AST 코드 그래프, 함수 추적, 데드코드
- **context7** (2도구): 라이브러리 문서 최신화

### Slash Commands
- `/stablenet-review-code` — 코드 분석 & 리뷰
- `/stablenet-build` — 빌드 & 에러 분석
- `/stablenet-test` — 패키지 테스트 & 실패 분석
- `/stablenet-lint` — 린트 & 자동 수정
- `/stablenet-impact` — 변경 영향 분석
- `/stablenet-hardfork` — 하드포크 추가 가이드 (7단계)
- `/stablenet-debug` — 체인 디버깅 (4-Phase)

### Agents (consensus/contract 변경 시 자동 활성화)
- **consensus-expert** (opus) — WBFT 합의 전문
- **contract-reviewer** (opus) — 시스템 컨트랙트 리뷰
- **build-resolver** (sonnet) — 빌드 에러 해결
- **chain-debugger** (sonnet) — 체인 런타임 디버깅
- **hardfork-architect** (opus) — 하드포크 설계

### Skills (자동 감지)
- build-and-verify, test-driven-dev, systematic-debugging
- consensus-analysis, hardfork-guide, impact-analysis
- chain-verification, session-handoff

### Hooks
- SessionStart: git status + 토큰 상태
- PreToolUse: force push 차단, 커밋 전 확인
- PostToolUse: 토큰 추적
- Stop: 세션 요약 + handoff
```

---

## 10. 구현 로드맵

### Phase 1: 기반 강화 (1-2주) — 코드 작성 최소, 최대 효과

| # | 작업 | 난이도 | 참조 패턴 |
|---|------|--------|----------|
| 1.1 | 슬래시 커맨드 6개 추가 (`.claude/commands/`) | 낮음 | superpowers SKILL.md |
| 1.2 | Hooks 설정 (`.claude/hooks.json`) | 낮음 | kamar-taj + agent-forge |
| 1.3 | Skills 8개 작성 (`.claude/skills/`) | 낮음 | superpowers |
| 1.4 | settings.local.json 확장 | 낮음 | — |
| 1.5 | CLAUDE.md 확장 | 낮음 | shuri |
| 1.6 | templates/plan.md + handoff.md | 낮음 | kamar-taj |
| 1.7 | install.sh 업데이트 | 낮음 | — |

### Phase 2: MCP 에코시스템 (2-3주)

| # | 작업 | 난이도 | 참조 패턴 |
|---|------|--------|----------|
| 2.1 | stablenet-build MCP 서버 개발 (10도구) | 중간 | chainbench MCP 패턴 |
| 2.2 | codebase-memory-mcp 통합 | 낮음 | npm/바이너리 설치 |
| 2.3 | context7 통합 | 낮음 | .mcp.json 추가 |
| 2.4 | .mcp.json 통합 구성 | 낮음 | — |

### Phase 3: Agent 시스템 (1-2주)

| # | 작업 | 난이도 | 참조 패턴 |
|---|------|--------|----------|
| 3.1 | 전문 Agent 5개 작성 | 낮음 | shuri + gsd |
| 3.2 | Agent 라우팅 규칙 (CLAUDE.md) | 낮음 | shuri + shogun |

### Phase 4: 외부 시스템 연동 (3-4주)

| # | 작업 | 난이도 | 참조 패턴 |
|---|------|--------|----------|
| 4.1 | Atlassian MCP 플러그인 활용 (Jira) | 중간 | 기존 MCP 도구 |
| 4.2 | GitHub PR 워크플로우 커맨드 | 낮음 | gsd ship |
| 4.3 | Slack Webhook 알림 (hooks) | 중간 | openclaw |
| 4.4 | commitflow 적용 (업스트림 동기화) | 중간 | commitflow |

### Phase 5: 지능형 자동화 (4-6주)

| # | 작업 | 난이도 | 참조 패턴 |
|---|------|--------|----------|
| 5.1 | E2E 워크플로우 커맨드 | 높음 | gsd + arc-reactor |
| 5.2 | 하드포크 워크플로우 (Wave) | 높음 | arc-reactor + spec-kit |
| 5.3 | 디버깅 워크플로우 (4-Phase) | 중간 | superpowers debugging |
| 5.4 | 세션 메모리/학습 시스템 | 높음 | memory-bank + kamar-taj |

---

## 11. 리스크 및 고려사항

### 11.1 기술적 리스크

| 리스크 | 완화 방안 | 참조 |
|--------|----------|------|
| MCP 서버 불안정 | 폴백: 직접 Bash 실행 | agent-forge deviation |
| 토큰 비용 증가 | token-monitor + codebase-memory로 상쇄 | context-mode |
| Agent 품질 편차 | QR Gate + 완료 게이트 강제 | agent-forge + shuri |
| 세션 간 컨텍스트 손실 | handoff.md + memory-bank | kamar-taj |

### 11.2 ROI 예상

| 시나리오 | Before | After | 절감 | 핵심 도구 |
|---------|--------|-------|------|----------|
| 세션 시작 컨텍스트 로딩 | ~10분 | <10초 | 98% | codebase-memory |
| 버그 수정 전체 사이클 | ~2시간 | ~30분 | 75% | stablenet-build + debug |
| 하드포크 구현 | ~2일 | ~4시간 | 75% | hardfork workflow |
| 코드 리뷰 | ~1시간 | ~15분 | 75% | agents + review-code |
| 체인 디버깅 | ~3시간 | ~45분 | 75% | chainbench + debug |
| 토큰 비용 | 기준선 | -50%+ | 50%+ | codebase-memory + context-mode |

---

## 부록: 분석한 레퍼런스 프로젝트 전체 목록

### 직접 소스코드 분석 완료 (30개)

**Plugin/Skill 시스템 (4개)**:
superpowers, shuri(all-in-one-claude-code), kamar-taj, get-shit-done

**Agent 오케스트레이션 (4개)**:
oh-my-claudecode, gstack, arc-reactor, everything-claude-code

**MCP/컨텍스트 (4개)**:
context-mode, codebase-memory-mcp, memory-bank, vibranium

**워크플로우 자동화 (5개)**:
agent-forge, clawflows, commitflow, claude-task-master, all-in-one-claude-code

**멀티 에이전트 (4개)**:
multi-agent-shogun, autonomous-coding-agents, claude-code-templates, agent-skills

**전문 도구 (5개)**:
claude-code(공식), openclaw, SuperClaude_Framework, spec-kit, eventcatalog

**기존 분석 (4개)**:
token-monitor, chainbench, go-stablenet, packages/claude-ai

---

*이 제안서는 ai-cli/ 하위 130+ 프로젝트 중 30개 핵심 프로젝트의 실제 소스코드를 직접 탐색하여 작성되었습니다.*
*각 레퍼런스의 구체적 파일 경로, 패턴, 구현 방법을 기반으로 현실적 적용 방안을 제시합니다.*
