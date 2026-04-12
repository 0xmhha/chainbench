# go-stablenet AI 자동화 시스템 구축 — 작업 지시서

> **Date**: 2026-04-03
> **Project**: packages/claude-ai 고도화
> **Target**: go-stablenet 개발 라이프사이클 전체 자동화
> **Jira Project Key**: (예: STNET 또는 AIDEV — 실제 키로 교체)

---

## 1. 프로젝트 개요

### 1.1 배경

go-stablenet은 go-ethereum 포크 기반의 StableNet 블록체인 클라이언트로, 160개 패키지/781개 Go 파일 규모의 대형 프로젝트입니다. 현재 개발 과정에서 다음과 같은 반복적 비효율이 발생합니다:

| 현재 문제 | 빈도 | 영향 |
|-----------|------|------|
| 코드 수정 후 빌드/테스트를 수동으로 실행하고 결과를 직접 분석 | 매일 | 개발자 시간 소모 |
| 781개 파일 구조를 매 세션마다 AI가 재탐색 | 매 세션 | 토큰 비용 + 10분 지연 |
| 하드포크 추가 시 7단계 프로세스를 매번 수동 수행 | 분기 1-2회 | 2일 소요, 누락 리스크 |
| 체인 런타임 이슈 디버깅 시 로그 수동 수집/분석 | 주 1-2회 | 3시간+ 소요 |
| Jira 티켓 ↔ 코드 변경 ↔ PR의 수동 연결 | 매일 | 추적성 부재 |
| 코드 리뷰 시 컨벤션/패턴 수동 확인 | PR마다 | 일관성 편차 |
| 빌드/테스트 실패 시 팀 알림 수동 | 비정기 | 지연된 대응 |

### 1.2 목표

**Claude Code AI Agent 시스템을 구축하여**, 위 문제들을 자동화하고 개발 속도와 품질을 동시에 향상시킵니다.

### 1.3 기대 효과 요약

| 지표 | Before | After | 개선율 |
|------|--------|-------|--------|
| 세션 시작 → 작업 시작 | ~10분 | <30초 | 95% |
| 버그 수정 전체 사이클 | ~2시간 | ~30분 | 75% |
| 하드포크 구현 사이클 | ~2일 | ~4시간 | 75% |
| 코드 리뷰 소요 시간 | ~1시간 | ~15분 | 75% |
| 체인 디버깅 소요 시간 | ~3시간 | ~45분 | 75% |
| AI 토큰 비용 | 기준선 | -50%+ | 50% |

---

## 2. 기능별 작업 정의

> 각 기능이 **왜 필요한지**, **무엇을 만드는지**, **어떤 문제를 해결하는지** 정리

---

### Feature 1: 코드 자동 구현 지원 시스템

#### 왜 필요한가
- Claude Code가 go-stablenet 코드를 수정할 때, 프로젝트 구조/컨벤션/아키텍처를 정확히 이해해야 올바른 코드를 생성
- 현재는 CLAUDE.md와 5개 참조 문서만 제공 — AI가 매번 프로젝트를 재탐색해야 함
- 코드 생성 후 컨벤션 준수 여부를 수동으로 확인해야 함

#### 기대 결과물
- **코드 인덱싱 시스템**: 781개 Go 파일의 AST 기반 코드 그래프 → 함수/타입/호출관계 즉시 검색
- **컨벤션 자동 적용**: Go 파일 수정 시 goimports/gofmt 자동 실행 (Hook)
- **영향 분석 도구**: 파일 수정 시 영향받는 패키지/테스트 자동 식별
- **전문 Agent**: consensus-expert, contract-reviewer 등 도메인별 전문 에이전트

#### 해결하는 문제
- AI가 프로젝트 전체를 재탐색하는 시간/비용 제거 (99% 토큰 절감)
- 코드 생성 품질 향상 (컨벤션 자동 적용)
- 변경 영향 범위 사전 파악 → 사이드 이펙트 방지

---

### Feature 2: 빌드 & 테스트 자동화 시스템

#### 왜 필요한가
- 코드 수정 후 `make gstable`, `make lint`, `go test` 를 수동 실행하고 출력을 직접 분석
- 빌드 에러 발생 시 에러 메시지에서 파일/라인을 찾아 수동으로 이동
- 테스트 실패 시 실패 원인을 수동으로 추적
- 테스트 커버리지를 별도로 확인해야 함

#### 기대 결과물
- **stablenet-build MCP 서버**: 빌드/테스트/린트/커버리지/벤치마크를 AI가 직접 호출 가능한 도구
- **빌드 에러 자동 파싱**: 에러 위치(파일:라인) + 원인 분석 + 수정 제안
- **테스트 결과 구조화**: pass/fail/skip 수, 실패 상세, 스택트레이스 자동 정리
- **슬래시 커맨드**: `/stablenet-build`, `/stablenet-test`, `/stablenet-lint`

#### 해결하는 문제
- 빌드/테스트 결과 분석 시간 제거
- 빌드 에러 → 수정 → 재빌드 루프 자동화
- 테스트 커버리지 미달 사전 감지

---

### Feature 3: 체인 테스트벤치 통합 (chainbench 연동)

#### 왜 필요한가
- go-stablenet의 빌드 결과물(gstable 바이너리)로 로컬 체인 네트워크를 구성하여 실제 동작 검증 필요
- 합의(WBFT) 정상 동작, 트랜잭션 처리, 장애 복구 등은 단위 테스트만으로 검증 불가
- chainbench가 이미 구현되어 있으나, Claude Code와의 통합 워크플로우가 없음

#### 기대 결과물
- **chainbench MCP 통합**: Claude가 직접 `chainbench init/start/stop/test` 호출 가능
- **체인 검증 스킬**: 코드 수정 → 빌드 → 체인 기동 → 합의 테스트 → 결과 분석 자동화
- **트랜잭션 테스트**: 특정 TX 시나리오(전송, 수수료 위임, 시스템 컨트랙트 호출)를 체인에서 직접 실행/검증
- **장애 시뮬레이션**: 노드 크래시/복구, 2/4 밸리데이터 다운 시 합의 중단/재개 검증

#### 해결하는 문제
- 합의 로직 변경 후 실제 체인 환경에서의 검증 자동화
- 하드포크 적용 후 시스템 컨트랙트 업그레이드 정상 동작 확인
- TX 처리 관련 변경의 end-to-end 검증

---

### Feature 4: 런타임 디버깅 지원 시스템

#### 왜 필요한가
- 체인 운영 중 합의 멈춤, 블록 생성 지연, 노드 동기화 실패 등의 이슈 발생 시 로그를 수동 수집/분석
- 여러 노드의 로그를 교차 분석하여 타임라인을 구성해야 함
- 노드 RPC API를 통해 블록 높이, 피어 수, 트랜잭션 풀 상태 등을 수동 조회
- WBFT 합의 플로우의 복잡한 상태 전이를 이해하고 추적해야 함

#### 기대 결과물
- **디버깅 워크플로우 커맨드**: `/stablenet-debug` — 증상 수집 → 가설 수립 → 검증 → 수정의 4-Phase
- **로그 수집/분석**: chainbench의 log_search, log_timeline, log_anomaly 도구 활용
- **노드 RPC 조회**: `eth_blockNumber`, `net_peerCount`, `txpool_status` 등 자동 수집
- **WBFT 합의 분석 스킬**: Core.Start() → handleEvents() 루프 → 메시지 핸들러 흐름 추적
- **전문 Agent**: chain-debugger — 로그 패턴 매칭 + 코드 경로 추적 + 근본 원인 분석

#### 해결하는 문제
- 런타임 이슈 진단 시간 3시간 → 45분
- 여러 노드의 로그 교차 분석 자동화
- WBFT 합의 상태 전이 시각화 및 이상 감지
- 재현 시나리오 자동 구성 (chainbench 활용)

---

### Feature 5: 하드포크 구현 자동화

#### 왜 필요한가
- 하드포크 추가는 7단계 복잡한 프로세스 (컨트랙트 컴파일 → bytecode 저장 → go:embed 등록 → Config 설정 → CollectUpgrades 등록 → 체인 테스트 → genesis 해시 재계산)
- 각 단계에서 누락이 발생하면 체인 장애로 이어짐
- 기존 하드포크(Applepie, Anzeon, Boho) 패턴을 매번 수동으로 참조

#### 기대 결과물
- **하드포크 가이드 커맨드**: `/stablenet-hardfork` — 7단계 체크리스트 자동 생성
- **하드포크 스킬**: SYSTEM_CONTRACT_FLOW.md 기반 단계별 가이드 + 검증
- **Wave 실행**: 병렬 가능한 작업은 동시 수행, 의존 작업은 순차 실행
- **체인 검증 포함**: 하드포크 블록 도달 → 컨트랙트 업그레이드 확인 → 합의 정상 동작 확인
- **전문 Agent**: hardfork-architect — 하드포크 설계 + 구현 + 검증 전담

#### 해결하는 문제
- 하드포크 구현 시간 2일 → 4시간
- 단계 누락으로 인한 체인 장애 리스크 제거
- 기존 하드포크 패턴의 일관된 적용 보장

---

### Feature 6: Jira 연동 자동화

#### 왜 필요한가
- Jira 티켓을 확인하고, 관련 코드를 찾고, 브랜치를 만들고, 작업 완료 후 상태를 변경하는 과정이 모두 수동
- 티켓과 코드 변경/PR 간의 추적성이 개발자의 수동 기록에 의존
- 현재 스프린트의 할당된 티켓 목록을 별도로 확인해야 함

#### 기대 결과물
- **Jira MCP 통합**: 티켓 조회, 상태 변경, 코멘트 추가를 AI가 직접 수행
- **티켓 기반 워크플로우**: 티켓 번호 입력 → 내용 분석 → 브랜치 생성 → 상태 "In Progress"
- **자동 상태 갱신**: PR 생성 시 → "In Review", PR 머지 시 → "Done"
- **코멘트 자동 추가**: PR 링크, 변경 파일 목록, 테스트 결과 등을 티켓에 자동 기록

#### 해결하는 문제
- 티켓 ↔ 코드 변경 추적성 자동 보장
- 티켓 상태 갱신 누락 방지
- 작업 시작/종료 시 수동 Jira 조작 제거

---

### Feature 7: GitHub PR 자동화

#### 왜 필요한가
- 코드 구현 완료 후 PR 생성 시 제목/설명을 수동 작성
- Conventional Commits 형식의 커밋 메시지를 수동 작성
- PR 생성 후 Jira 티켓에 수동으로 PR 링크 추가

#### 기대 결과물
- **PR 자동 생성 커맨드**: Jira 티켓 기반으로 PR 제목/설명 자동 생성
- **Conventional Commits**: 변경 내용 분석 → `feat:`, `fix:`, `refactor:` 등 자동 분류
- **커밋 메타데이터**: AI 작업 추적을 위한 트레일러 (`Confidence`, `Scope-risk`, `Co-Authored-By`)
- **Jira 연결**: PR 생성 시 자동으로 티켓에 PR 링크 코멘트

#### 해결하는 문제
- PR 제목/설명 작성 시간 제거
- 커밋 메시지 일관성 보장 (Conventional Commits)
- PR ↔ Jira 티켓 연결 자동화

---

### Feature 8: AI 코드 리뷰 시스템

#### 왜 필요한가
- PR 코드 리뷰 시 CODE_CONVENTION.md 준수 여부를 수동 확인
- WBFT 합의 로직, 시스템 컨트랙트 등 도메인 특화 리뷰가 필요하지만 전문 지식이 필요
- 리뷰 품질이 리뷰어의 가용 시간/전문성에 따라 편차 발생

#### 기대 결과물
- **코드 리뷰 커맨드**: `/stablenet-review` — PR diff 분석 + 컨벤션 체크 + 도메인 리뷰
- **리뷰 항목**:
  - CODE_CONVENTION.md 준수 여부 (네이밍, import 정렬, 에러 처리)
  - 보안 취약점 (레이스 컨디션, 바운더리 체크, 크립토 안전성)
  - 합의 로직 영향 분석 (consensus/ 패키지 변경 시)
  - 시스템 컨트랙트 호환성 (systemcontracts/ 변경 시)
  - 테스트 충분성 (관련 테스트 존재 여부, 커버리지)
- **전문 Agent**: consensus-expert(합의), contract-reviewer(컨트랙트)

#### 해결하는 문제
- 리뷰 시간 1시간 → 15분
- 컨벤션 위반 자동 감지
- 도메인 특화 리뷰 품질 균일화
- 보안 취약점 사전 탐지

---

### Feature 9: Slack 알림 자동화

#### 왜 필요한가
- 빌드 실패, 테스트 실패, PR 생성 등의 이벤트를 팀에 수동 공유
- 긴급 이슈(합의 멈춤 등) 발생 시 빠른 팀 통보 필요
- 하드포크 관련 코드 변경은 전체 팀 인지 필요

#### 기대 결과물
- **Slack Webhook 통합**: 주요 이벤트 발생 시 지정 채널에 자동 알림
- **채널별 알림 라우팅**:
  - `#stablenet-dev`: 빌드 실패, 테스트 실패
  - `#code-review`: PR 생성, 리뷰 요청
  - `#stablenet-releases`: 하드포크 관련 변경
  - `#stablenet-qa`: 체인 테스트 실패

#### 해결하는 문제
- 이벤트 알림 지연/누락 제거
- 팀 인지 속도 향상
- 수동 공유 작업 제거

---

### Feature 10: 토큰 사용량 모니터링 통합

#### 왜 필요한가
- AI 사용 시 토큰 비용이 발생하지만 실시간 추적이 안 됨
- 비효율적인 컨텍스트 사용으로 불필요한 비용 발생
- 팀/프로젝트별 사용량 비교 및 최적화 기준 부재

#### 기대 결과물
- **token-monitor Hooks 통합**: 매 도구 사용 후 토큰 상태 표시
- **세션 종료 리포트**: 세션별 총 사용량, 번레이트, 빌링블록 상태
- **비용 최적화 기반 데이터**: codebase-memory 도입 전후 토큰 사용량 비교

#### 해결하는 문제
- 토큰 비용 가시성 확보
- 비효율적 사용 패턴 식별 및 개선
- 비용 절감 효과 측정

---

### Feature 11: 세션 간 컨텍스트 유지 시스템

#### 왜 필요한가
- Claude Code 세션이 종료되면 작업 컨텍스트가 모두 사라짐
- 다음 세션에서 "어디까지 했는지"를 다시 설명해야 함
- 반복적으로 발생하는 이슈 패턴을 학습하지 못함

#### 기대 결과물
- **handoff.md 시스템**: 세션 종료 시 자동으로 다음 세션 인수인계 파일 생성
  - 진행 중인 작업, 실패한 테스트, 미해결 이슈, 다음 단계
- **Session Start Hook**: 세션 시작 시 handoff.md + git status + 할당 티켓 자동 표시
- **plan.md 템플릿**: 작업 계획 구조화

#### 해결하는 문제
- 세션 전환 시 컨텍스트 손실 제거
- "어디까지 했지?" 재설명 시간 제거
- 작업 연속성 보장

---

## 3. 작업 단계 (Phase)

```
Phase 1: 기반 강화          ─── 마크다운 파일만으로 구현 (코드 없음)
  │                              Commands + Skills + Hooks + Agents 정의
  │
Phase 2: MCP 서버 구축      ─── TypeScript 개발 + 외부 도구 통합
  │                              stablenet-build MCP + codebase-memory 설치
  │
Phase 3: 외부 시스템 연동    ─── Jira + GitHub PR + Slack 통합
  │                              Atlassian MCP + Webhook + gh CLI
  │
Phase 4: 워크플로우 자동화   ─── E2E 워크플로우 구현
  │                              하드포크, 디버깅, 전체 개발 사이클
  │
Phase 5: 최적화 & 학습      ─── 세션 메모리 + 비용 최적화
                                  handoff + token 모니터링 + 피드백
```

---

## 4. Jira 티켓 계획

> **Epic**: go-stablenet AI 자동화 시스템 구축
> **각 Feature = Story**, **세부 작업 = Sub-task**

---

### Epic: go-stablenet AI 자동화 시스템 구축

---

### Phase 1: 기반 강화

#### Story 1-1: 코드 자동 구현 지원 — 슬래시 커맨드 & 스킬 구축

**제목**: `[AI-Auto] 코드 분석/구현 지원을 위한 Claude Code 커맨드 및 스킬 구축`

**설명**:
```
## 배경
현재 packages/claude-ai에는 /stablenet-review-code 1개 커맨드만 존재.
개발자가 Claude Code를 활용하여 코드 분석, 빌드, 테스트, 린트, 영향 분석 등을 
수행할 때 매번 수동으로 지시해야 하며, 일관된 워크플로우가 없음.

## 작업 내용
1. 슬래시 커맨드 6개 추가 (.claude/commands/)
   - /stablenet-build: 빌드 실행 + 에러 분석 + 수정 제안
   - /stablenet-test: 패키지별 테스트 + 결과 분석 + 실패 원인 추적
   - /stablenet-lint: make lint + CODE_CONVENTION.md 대조 + 자동 수정
   - /stablenet-impact: 변경 파일 → 영향 패키지 → 관련 테스트 식별
   - /stablenet-hardfork: 하드포크 추가 7단계 체크리스트
   - /stablenet-debug: 체인 디버깅 4-Phase 워크플로우

2. 스킬 8개 작성 (.claude/skills/)
   - build-and-verify, test-driven-dev, systematic-debugging
   - consensus-analysis, hardfork-guide, impact-analysis
   - chain-verification, session-handoff

3. CLAUDE.md 확장 (사용 가능한 도구/커맨드/스킬 문서화)

## 산출물
- .claude/commands/*.md (6개 파일)
- .claude/skills/*/SKILL.md (8개 파일)
- CLAUDE.md 업데이트

## 수용 기준
- 각 커맨드가 Claude Code에서 / 입력 시 자동완성됨
- 각 스킬이 관련 키워드 입력 시 자동 활성화됨
- 기존 /stablenet-review-code 정상 동작 유지
```

**Sub-tasks**:
- `[AI-Auto] /stablenet-build 커맨드 작성`
- `[AI-Auto] /stablenet-test 커맨드 작성`
- `[AI-Auto] /stablenet-lint 커맨드 작성`
- `[AI-Auto] /stablenet-impact 커맨드 작성`
- `[AI-Auto] /stablenet-hardfork 커맨드 작성`
- `[AI-Auto] /stablenet-debug 커맨드 작성`
- `[AI-Auto] Skills 8개 SKILL.md 작성`
- `[AI-Auto] CLAUDE.md 확장 (도구/커맨드/스킬 문서화)`

---

#### Story 1-2: Hooks & 권한 시스템 설정

**제목**: `[AI-Auto] Claude Code Hooks 및 권한 설정 구축`

**설명**:
```
## 배경
Claude Code의 Hook 시스템을 활용하면 이벤트 기반 자동화가 가능하나,
현재 hooks가 전혀 설정되어 있지 않음. 또한 settings.local.json의 권한이 
3개(python3, WebSearch, gh pr)로 제한되어 있어 Go 개발 도구 사용 시 
매번 수동 승인이 필요.

## 작업 내용
1. hooks.json 작성
   - SessionStart: git status + 최근 커밋 + 토큰 상태 표시
   - PreToolUse(git commit): 빌드/린트 확인 알림
   - PreToolUse(git push --force): 차단 + --force-with-lease 안내
   - PostToolUse(Edit|Write): 토큰 사용량 표시
   - Stop: 세션 요약 (변경사항 + 토큰 리포트)

2. settings.local.json 확장
   - Go 빌드 도구: go, make, golangci-lint, goimports, gofmt
   - Git/GitHub: git, gh (전체)
   - 보조 도구: chainbench, token-monitor, curl, jq
   - MCP: chainbench, token-monitor, stablenet-build, codebase-memory, context7

## 산출물
- .claude/hooks.json
- .claude/settings.local.json (확장)

## 수용 기준
- 세션 시작 시 프로젝트 상태가 자동 표시됨
- force push 시도 시 차단되고 안내 메시지 출력
- Go 개발 도구(make, go test 등) 실행 시 권한 승인 팝업 없음
```

---

#### Story 1-3: 전문 Agent 시스템 구축

**제목**: `[AI-Auto] go-stablenet 도메인 전문 AI Agent 정의`

**설명**:
```
## 배경
합의 로직(WBFT), 시스템 컨트랙트, 하드포크 등 도메인 전문 지식이 필요한 
작업에서 일반적인 AI 응답으로는 정확도가 부족. 전문 Agent를 정의하여 
도메인별 컨텍스트와 검증 프로세스를 강제.

## 작업 내용
1. Agent 5개 정의 (.claude/agents/)
   - consensus-expert (opus): WBFT 합의 엔진 전문
     - 필수 참조: CLAUDE_DEV_GUIDE §13-17, handler.go
     - 검증 요구: chainbench 합의 테스트 통과
   - contract-reviewer (opus): 시스템 컨트랙트 리뷰
     - 필수 참조: SYSTEM_CONTRACT_FLOW.md
     - 검증 요구: 버전 호환성, 스토리지 레이아웃 확인
   - build-resolver (sonnet): 빌드 에러 해결
     - 자동 활성화: make 실패 시
   - chain-debugger (sonnet): 체인 런타임 디버깅
     - 도구: chainbench log/rpc + codebase-memory trace
   - hardfork-architect (opus): 하드포크 설계 + 구현
     - 필수 참조: SYSTEM_CONTRACT_FLOW.md, 기존 하드포크 패턴

2. Agent 라우팅 규칙을 CLAUDE.md에 문서화

## 산출물
- .claude/agents/*.md (5개 파일)
- CLAUDE.md 라우팅 규칙 섹션

## 수용 기준
- "consensus" 키워드 포함 작업 시 consensus-expert 자동 제안
- 각 Agent가 필수 참조 파일을 반드시 읽은 후 작업 수행
- 합의 로직 변경 시 opus 모델 사용 강제
```

---

#### Story 1-4: 세션 컨텍스트 유지 시스템

**제목**: `[AI-Auto] 세션 간 컨텍스트 유지를 위한 handoff 시스템 구축`

**설명**:
```
## 배경
Claude Code 세션 종료 시 작업 컨텍스트가 모두 사라짐. 다음 세션에서 
"어디까지 했는지" 다시 설명해야 하며, 진행 중인 이슈/실패한 테스트 등의 
정보가 유실됨.

## 작업 내용
1. templates/handoff.md 작성 — 세션 종료 시 작성 템플릿
   - 진행 중인 작업
   - 실패한 테스트 / 미해결 이슈
   - 다음 단계
   - 주요 의사결정 사항

2. templates/plan.md 작성 — 작업 계획 구조화 템플릿

3. session-handoff 스킬 — Stop hook에서 handoff.md 작성 유도

4. SessionStart hook — handoff.md 존재 시 자동 로드

## 산출물
- templates/handoff.md, templates/plan.md
- session-handoff 스킬 (Phase 1에서 이미 생성)
- hooks.json의 SessionStart/Stop 동작 확장

## 수용 기준
- 세션 종료 시 handoff.md 작성 여부 확인 메시지 출력
- 다음 세션 시작 시 이전 handoff 내용 자동 표시
- plan.md 작성 시 구조화된 형식 가이드 제공
```

---

#### Story 1-5: install.sh 업데이트

**제목**: `[AI-Auto] 설치 스크립트 확장 — 새 파일 포함`

**설명**:
```
## 배경
install.sh가 현재 8개 파일만 다운로드. 새로 추가되는 commands, skills,
agents, hooks, templates, mcp.json을 포함하도록 확장 필요.

## 작업 내용
1. install.sh 수정 — 추가 파일 다운로드 목록 확장
   - .claude/commands/ (6개 신규)
   - .claude/skills/ (8개 디렉토리)
   - .claude/agents/ (5개)
   - .claude/hooks.json
   - templates/ (2개)
   - .mcp.json

2. install-local.sh 동일 수정
3. uninstall.sh — 추가된 파일 정리 로직

## 산출물
- install.sh, install-local.sh, uninstall.sh 업데이트
- README.md 업데이트 (새 파일 목록 반영)

## 수용 기준
- curl 원라이너로 전체 시스템 설치 가능
- 기존 설정 백업 후 설치 (기존 동작 유지)
- uninstall 시 추가된 파일 모두 정리
```

---

### Phase 2: MCP 서버 구축

#### Story 2-1: stablenet-build MCP 서버 개발

**제목**: `[AI-Auto] go-stablenet 빌드/테스트/분석 MCP 서버 개발`

**설명**:
```
## 배경
Claude Code가 빌드/테스트/린트 결과를 구조화된 형태로 직접 호출할 수 있는 
MCP(Model Context Protocol) 서버가 필요. 현재는 Bash로 명령 실행 후 
출력을 텍스트로 파싱해야 하며, 에러 위치/원인을 AI가 정확히 파악하기 어려움.

## 작업 내용
TypeScript MCP 서버 개발 (@modelcontextprotocol/sdk 사용)

제공 도구 (10개):
1. stablenet_build — make gstable/all, 에러 위치(file:line) 파싱
2. stablenet_test — go test + 결과 구조화 (pass/fail/skip, 실패 상세)
3. stablenet_test_coverage — go test -coverprofile + 미커버 라인
4. stablenet_lint — golangci-lint + 규칙별 분류
5. stablenet_impact — go list -deps 기반 변경 영향 분석
6. stablenet_deps — 패키지 의존성 트리
7. stablenet_bench — go test -bench + 결과 파싱
8. stablenet_bench_compare — 커밋 간 벤치마크 비교
9. stablenet_vet — go vet 결과
10. stablenet_status — 현재 빌드 상태, 바이너리 버전

## 산출물
- mcp-servers/stablenet-build/ (TypeScript 프로젝트)
- .mcp.json에 등록
- 단위 테스트

## 수용 기준
- Claude Code에서 stablenet_build 호출 시 빌드 실행 + 구조화된 결과 반환
- 빌드 에러 시 파일:라인:메시지 형태로 정확한 위치 반환
- 테스트 실패 시 실패 테스트명 + 스택트레이스 반환
- stdio 전송 방식으로 Claude Code와 통신
```

**Sub-tasks**:
- `[AI-Auto] MCP 서버 프로젝트 구조 생성 + 기본 설정`
- `[AI-Auto] stablenet_build, stablenet_status 도구 구현`
- `[AI-Auto] stablenet_test, stablenet_test_coverage 도구 구현`
- `[AI-Auto] stablenet_lint, stablenet_vet 도구 구현`
- `[AI-Auto] stablenet_impact, stablenet_deps 도구 구현`
- `[AI-Auto] stablenet_bench, stablenet_bench_compare 도구 구현`
- `[AI-Auto] MCP 서버 단위 테스트 작성`

---

#### Story 2-2: codebase-memory-mcp 통합

**제목**: `[AI-Auto] 코드 그래프 인덱싱 MCP 서버 (codebase-memory) 통합`

**설명**:
```
## 배경
781개 Go 파일로 구성된 go-stablenet 프로젝트에서 AI가 매 세션마다 
파일/함수를 grep으로 탐색하면 대량의 토큰이 소모됨.
codebase-memory-mcp는 AST 기반 코드 그래프로 99% 토큰 절감 가능.

## 작업 내용
1. codebase-memory-mcp 바이너리 설치 (macOS arm64)
2. .mcp.json에 등록
3. go-stablenet 프로젝트 초기 인덱싱 수행 및 성능 측정
4. 활용 가이드 문서화 (search, trace, impact, architecture 도구)

## 산출물
- codebase-memory-mcp 설치 스크립트 또는 가이드
- .mcp.json 업데이트
- 초기 인덱싱 결과 리포트 (인덱싱 시간, 파일 수, 심볼 수)

## 수용 기준
- Claude Code 세션에서 codebase-memory search 호출 가능
- "handleCommitMsg 함수의 호출자를 찾아줘" 요청 시 trace로 즉시 결과 반환
- grep 대비 토큰 사용량 90%+ 절감 확인
```

---

#### Story 2-3: context7 및 chainbench/token-monitor MCP 통합 설정

**제목**: `[AI-Auto] MCP 서버 통합 구성 (.mcp.json)`

**설명**:
```
## 배경
chainbench(13도구)와 token-monitor(6도구)가 이미 MCP 서버를 제공하지만
go-stablenet 프로젝트의 .mcp.json에 등록되어 있지 않음.
context7(Go/geth 라이브러리 문서)도 통합하여 최신 API 참조 가능하게 함.

## 작업 내용
1. .mcp.json 통합 구성
   - stablenet-build (Phase 2-1에서 개발)
   - chainbench (기존)
   - token-monitor (기존)
   - codebase-memory (Phase 2-2에서 설치)
   - context7 (npm 설치)

2. 각 MCP 서버 연동 테스트
3. CLAUDE.md에 MCP 서버 목록 및 사용법 문서화

## 산출물
- .mcp.json (5개 서버 통합)
- MCP 서버별 연동 테스트 결과

## 수용 기준
- Claude Code 세션에서 5개 MCP 서버 모두 정상 로드
- 각 MCP 서버의 도구가 호출 가능
```

---

### Phase 3: 외부 시스템 연동

#### Story 3-1: Jira 연동 시스템 구축

**제목**: `[AI-Auto] Jira 연동 — 티켓 기반 개발 워크플로우 자동화`

**설명**:
```
## 배경
Jira 티켓을 확인하고, 관련 코드를 찾고, 브랜치를 만들고, 작업 완료 후 
상태를 변경하는 과정이 모두 수동. 티켓과 코드 변경 간 추적성이 부족.

## 작업 내용
1. Atlassian MCP 플러그인 설정 (인증 + 프로젝트 연결)
2. Jira 연동 커맨드/스킬 작성
   - 티켓 조회 + 관련 코드 영역 분석
   - 브랜치 자동 생성 (feature/STNET-{번호}-{설명})
   - 상태 자동 변경 (Todo → In Progress → In Review → Done)
   - PR 링크/변경사항 코멘트 자동 추가
3. SessionStart hook에 할당 티켓 목록 표시

## 산출물
- Atlassian MCP 설정 가이드
- Jira 연동 커맨드/스킬
- hooks.json 확장 (SessionStart에 Jira 정보)

## 수용 기준
- "STNET-123 작업 시작" 입력 시 → 티켓 조회 + 브랜치 생성 + 상태 변경
- PR 생성 시 → Jira 티켓에 자동 코멘트
- 세션 시작 시 → 나에게 할당된 티켓 목록 표시
```

---

#### Story 3-2: GitHub PR 워크플로우 자동화

**제목**: `[AI-Auto] GitHub PR 생성/리뷰 자동화`

**설명**:
```
## 배경
코드 구현 완료 후 PR 생성 시 제목/설명을 수동 작성해야 하며,
Conventional Commits 형식의 커밋 메시지도 수동 작성. 또한 PR 생성 시 
Jira 티켓 연결도 수동으로 해야 함.

## 작업 내용
1. PR 자동 생성 커맨드 작성
   - 변경 내용 분석 → 제목/설명 자동 생성
   - Jira 티켓 번호 자동 포함
   - Conventional Commits 형식 커밋 메시지 생성
   - 커밋 트레일러 추가 (Confidence, Scope-risk, Co-Authored-By)

2. 코드 리뷰 커맨드 작성
   - PR diff 분석
   - CODE_CONVENTION.md 준수 확인
   - 보안/성능/합의 관련 리뷰
   - 리뷰 코멘트 자동 생성

## 산출물
- PR 생성 커맨드/스킬
- 코드 리뷰 커맨드/스킬
- gh CLI 기반 자동화 스크립트

## 수용 기준
- "PR 생성해줘" 입력 시 → 변경 분석 + PR 자동 생성 + Jira 연결
- "이 PR 리뷰해줘" 입력 시 → diff 분석 + 컨벤션/보안 체크 + 코멘트 생성
- 커밋 메시지에 Conventional Commits 형식 + 트레일러 포함
```

---

#### Story 3-3: Slack 알림 시스템 구축

**제목**: `[AI-Auto] Slack Webhook 기반 개발 이벤트 알림 자동화`

**설명**:
```
## 배경
빌드 실패, 체인 테스트 실패, PR 생성 등의 개발 이벤트를 팀에 수동으로 
공유해야 함. 긴급 이슈(합의 멈춤 등) 발생 시 빠른 팀 통보 필요.

## 작업 내용
1. Slack Incoming Webhook 설정
2. 알림 스크립트 작성 (hooks에서 호출)
   - 빌드 실패 → #stablenet-dev
   - PR 생성 → #code-review
   - 체인 테스트 실패 → #stablenet-qa
   - 하드포크 관련 변경 → #stablenet-releases
3. 알림 메시지 포맷 정의 (제목, 상세, 링크)

## 산출물
- Slack Webhook 설정 가이드
- 알림 스크립트 (bash 또는 node)
- hooks.json 확장 (이벤트별 Slack 알림)

## 수용 기준
- 빌드 실패 시 → #stablenet-dev에 에러 요약 + 파일 위치 알림
- PR 생성 시 → #code-review에 PR 링크 + 변경 요약 알림
```

---

### Phase 4: 워크플로우 자동화

#### Story 4-1: E2E 개발 워크플로우 자동화

**제목**: `[AI-Auto] Jira → 구현 → 빌드 → 테스트 → PR End-to-End 워크플로우`

**설명**:
```
## 배경
Phase 1-3에서 구축한 개별 기능들(커맨드, MCP, Jira, GitHub, Slack)을 
하나의 연결된 워크플로우로 통합. 티켓 번호 입력만으로 전체 개발 사이클 수행.

## 작업 내용
1. E2E 워크플로우 커맨드 작성 (/stablenet-workflow)
   Step 1: Jira 티켓 조회 + codebase-memory 컨텍스트 로드
   Step 2: 복잡도 분류 + 영향 분석 + 작업 계획
   Step 3: 브랜치 생성 + 코드 구현 + 테스트 작성
   Step 4: 빌드 + 린트 + 테스트 + [체인 테스트] 검증
   Step 5: PR 생성 + Jira 상태 변경 + Slack 알림
   Step 6: 세션 정리 (handoff + 토큰 리포트)

2. 복잡도 기반 분기 (agent-forge 패턴)
   - Micro: 직접 구현 + 빌드/린트만 검증
   - Standard: 영향 분석 → 구현 → 전체 검증
   - Full: 설계 → Wave 실행 → 체인 테스트 포함

## 산출물
- /stablenet-workflow 커맨드
- 복잡도 분류 로직
- E2E 워크플로우 문서

## 수용 기준
- "STNET-123 작업해줘" → 전체 사이클 자동 수행
- Micro 작업: 10분 이내 완료
- Standard 작업: 30분 이내 완료
```

---

#### Story 4-2: 하드포크 자동화 워크플로우

**제목**: `[AI-Auto] 하드포크 구현 자동화 — Wave 기반 단계별 실행`

**설명**:
```
## 배경
하드포크 추가는 7단계 복잡한 프로세스. 현재 SYSTEM_CONTRACT_FLOW.md에 
가이드가 있지만 수동 수행. 각 단계의 의존성을 고려한 병렬/순차 실행과 
체인 검증까지 포함한 자동화 워크플로우 필요.

## 작업 내용
1. 하드포크 워크플로우 커맨드 작성 (/stablenet-hardfork-workflow)
   Wave 1 (병렬): 컨트랙트 컴파일 + Config 설계 + 패턴 분석
   Wave 2 (순차): artifacts 저장 → go:embed 등록 → upgrade 함수 → CollectUpgrades
   Wave 3 (검증): 빌드 → 테스트 → chainbench 체인 테스트
   Wave 4 (문서): CLAUDE_DEV_GUIDE 업데이트 → PR 생성

2. 각 Wave 사이에 Checkpoint (결과 확인 후 진행)

## 산출물
- /stablenet-hardfork-workflow 커맨드
- Wave 실행 + Checkpoint 로직
- 하드포크 검증 체크리스트

## 수용 기준
- 하드포크명 입력 → Wave 기반 단계별 자동 실행
- 각 Checkpoint에서 결과 확인 + 사용자 승인 후 진행
- 체인 테스트에서 하드포크 블록 도달 + 컨트랙트 업그레이드 확인
```

---

#### Story 4-3: 디버깅 자동화 워크플로우

**제목**: `[AI-Auto] 체인 런타임 디버깅 자동화 — 4-Phase 진단 시스템`

**설명**:
```
## 배경
체인 운영 중 합의 멈춤, 블록 생성 지연 등의 이슈 발생 시 여러 노드의 
로그를 수동 수집/분석해야 함. WBFT 합의 상태 전이를 이해하고 추적하는 
전문 지식이 필요.

## 작업 내용
1. 디버깅 워크플로우 고도화
   Phase 1 (Observe): chainbench status + 노드 RPC + 로그 수집
   Phase 2 (Hypothesize): WBFT 플로우 참조 + 로그 패턴 매칭 + 가설 수립
   Phase 3 (Test): 가설별 코드 추적 + chainbench 재현
   Phase 4 (Fix): 수정 + 빌드 + 체인 검증

2. chain-debugger Agent 고도화
   - 알려진 이슈 패턴 DB
   - 자동 재현 시나리오 구성

## 산출물
- 디버깅 워크플로우 커맨드/스킬 고도화
- 알려진 이슈 패턴 문서
- 자동 재현 시나리오 가이드

## 수용 기준
- 증상 설명 입력 → 자동 로그 수집 + 타임라인 생성
- 알려진 패턴 매칭 시 즉시 해결 방안 제시
- 수정 후 chainbench로 자동 검증
```

---

### Phase 5: 최적화 & 학습

#### Story 5-1: 토큰 사용량 모니터링 통합

**제목**: `[AI-Auto] token-monitor Hooks 통합 및 비용 최적화 대시보드`

**설명**:
```
## 배경
AI 토큰 비용이 발생하지만 실시간 추적이 안 됨. 
codebase-memory 등 최적화 도구 도입 효과를 측정할 수 없음.

## 작업 내용
1. token-monitor hooks 통합
   - PostToolUse: 매 도구 사용 후 토큰 상태 표시
   - Stop: 세션 종료 시 상세 사용량 리포트
2. 비용 최적화 측정
   - codebase-memory 도입 전후 토큰 비교
   - 세션별/기능별 토큰 사용량 추적

## 산출물
- hooks.json에 token-monitor 통합
- 비용 측정 방법론 문서

## 수용 기준
- 매 도구 사용 후 현재 토큰 사용량 표시
- 세션 종료 시 총 사용량/번레이트/빌링블록 리포트
```

---

#### Story 5-2: Confluence 문서 자동화 (선택사항)

**제목**: `[AI-Auto] Confluence 기술 문서 자동 생성/업데이트`

**설명**:
```
## 배경
하드포크 스펙, 아키텍처 변경, 기술 분석 결과 등을 Confluence에 수동 작성.
AI가 코드 분석/구현 과정에서 생성한 정보를 자동으로 문서화하면 효율적.

## 작업 내용
1. Confluence API 연동 (Atlassian MCP 활용)
2. 자동 문서 생성 시나리오
   - 하드포크 구현 후 → 기술 스펙 자동 생성
   - 시스템 컨트랙트 변경 후 → 아키텍처 문서 업데이트
   - 코드 리뷰 결과 → 기술 노트 저장

## 산출물
- Confluence 연동 커맨드/스킬
- 문서 템플릿

## 수용 기준
- 하드포크 완료 후 "문서 생성해줘" → Confluence 페이지 자동 생성
```

---

## 5. Jira 티켓 요약 테이블

### Epic

| Key | 제목 | Type |
|-----|------|------|
| STNET-E1 | go-stablenet AI 자동화 시스템 구축 | Epic |

### Stories (Phase 순서)

| Key | 제목 | Phase | 예상 기간 | 의존성 |
|-----|------|-------|----------|--------|
| STNET-S01 | 코드 분석/구현 지원을 위한 Claude Code 커맨드 및 스킬 구축 | 1 | 3일 | — |
| STNET-S02 | Claude Code Hooks 및 권한 설정 구축 | 1 | 1일 | — |
| STNET-S03 | go-stablenet 도메인 전문 AI Agent 정의 | 1 | 2일 | — |
| STNET-S04 | 세션 간 컨텍스트 유지를 위한 handoff 시스템 구축 | 1 | 1일 | — |
| STNET-S05 | 설치 스크립트 확장 — 새 파일 포함 | 1 | 1일 | S01-S04 |
| STNET-S06 | go-stablenet 빌드/테스트/분석 MCP 서버 개발 | 2 | 5일 | — |
| STNET-S07 | 코드 그래프 인덱싱 MCP 서버 (codebase-memory) 통합 | 2 | 2일 | — |
| STNET-S08 | MCP 서버 통합 구성 (.mcp.json) | 2 | 1일 | S06, S07 |
| STNET-S09 | Jira 연동 — 티켓 기반 개발 워크플로우 자동화 | 3 | 3일 | Phase 1 |
| STNET-S10 | GitHub PR 생성/리뷰 자동화 | 3 | 3일 | Phase 1 |
| STNET-S11 | Slack Webhook 기반 개발 이벤트 알림 자동화 | 3 | 2일 | — |
| STNET-S12 | Jira → 구현 → 빌드 → 테스트 → PR E2E 워크플로우 | 4 | 5일 | Phase 1-3 |
| STNET-S13 | 하드포크 구현 자동화 — Wave 기반 단계별 실행 | 4 | 3일 | Phase 1-2 |
| STNET-S14 | 체인 런타임 디버깅 자동화 — 4-Phase 진단 시스템 | 4 | 3일 | Phase 1-2 |
| STNET-S15 | token-monitor Hooks 통합 및 비용 최적화 | 5 | 2일 | Phase 1 |
| STNET-S16 | Confluence 기술 문서 자동 생성/업데이트 (선택) | 5 | 3일 | Phase 3 |

### Sub-tasks (Story별 세부 작업)

**S01 Sub-tasks (8개)**:
| Key | 제목 |
|-----|------|
| STNET-T01 | /stablenet-build 커맨드 작성 |
| STNET-T02 | /stablenet-test 커맨드 작성 |
| STNET-T03 | /stablenet-lint 커맨드 작성 |
| STNET-T04 | /stablenet-impact 커맨드 작성 |
| STNET-T05 | /stablenet-hardfork 커맨드 작성 |
| STNET-T06 | /stablenet-debug 커맨드 작성 |
| STNET-T07 | Skills 8개 SKILL.md 작성 |
| STNET-T08 | CLAUDE.md 확장 (도구/커맨드/스킬 문서화) |

**S06 Sub-tasks (7개)**:
| Key | 제목 |
|-----|------|
| STNET-T09 | MCP 서버 프로젝트 구조 생성 + 기본 설정 |
| STNET-T10 | stablenet_build, stablenet_status 도구 구현 |
| STNET-T11 | stablenet_test, stablenet_test_coverage 도구 구현 |
| STNET-T12 | stablenet_lint, stablenet_vet 도구 구현 |
| STNET-T13 | stablenet_impact, stablenet_deps 도구 구현 |
| STNET-T14 | stablenet_bench, stablenet_bench_compare 도구 구현 |
| STNET-T15 | MCP 서버 단위 테스트 작성 |

---

## 6. 우선순위 및 실행 전략

### 즉시 시작 가능 (코드 작성 불필요)
Phase 1 전체가 **마크다운 파일 작성만으로** 완성됩니다:
- Commands (`.md`), Skills (`SKILL.md`), Agents (`.md`), Hooks (`.json`), Templates (`.md`)
- 개발 환경 변경 없이 바로 효과를 볼 수 있음
- **추천**: Phase 1의 S01~S04를 병렬로 동시 착수

### 의존성 체인
```
S01 (커맨드) ──┐
S02 (Hooks)  ──┤
S03 (Agents) ──┼── S05 (install.sh) ──── Phase 1 완료
S04 (handoff)──┘

S06 (build MCP) ──┐
S07 (codebase)  ──┼── S08 (.mcp.json) ── Phase 2 완료
                  │
Phase 1 ──────────┼── S09 (Jira) ──┐
                  ├── S10 (GitHub) ┼── Phase 3 완료
                  └── S11 (Slack) ─┘

Phase 1-3 ────────┬── S12 (E2E) ──┐
Phase 1-2 ────────┼── S13 (Fork) ─┼── Phase 4 완료
Phase 1-2 ────────┴── S14 (Debug)─┘

Phase 1 ──────────┬── S15 (Token) ─┐
Phase 3 ──────────┴── S16 (Docs) ──┴── Phase 5 완료
```

### 팀 병렬 작업 가능 영역
- **개발자 A**: S01 + S03 (커맨드 + 에이전트)
- **개발자 B**: S02 + S04 + S05 (Hooks + handoff + 설치)
- **개발자 C**: S06 (stablenet-build MCP 개발) — Phase 2와 병렬 가능

---

## 7. 성공 기준

### Phase 1 완료 시
- [ ] Claude Code에서 `/stablenet-` 입력 시 7개 커맨드 자동완성
- [ ] 세션 시작 시 프로젝트 상태 자동 표시
- [ ] force push 차단 동작 확인
- [ ] consensus 키워드 작업 시 전문 Agent 제안

### Phase 2 완료 시
- [ ] `stablenet_build` 호출 시 빌드 실행 + 구조화된 결과 반환
- [ ] codebase-memory로 함수 추적 가능 (grep 대비 90%+ 토큰 절감)
- [ ] 5개 MCP 서버 동시 로드 + 정상 동작

### Phase 3 완료 시
- [ ] Jira 티켓 번호로 작업 시작/종료 가능
- [ ] PR 자동 생성 + Jira 연결 동작
- [ ] Slack 알림 수신 확인

### Phase 4 완료 시
- [ ] E2E 워크플로우: 티켓 → PR 30분 이내 (Standard 기준)
- [ ] 하드포크 워크플로우: Wave 실행 + 체인 검증 완료
- [ ] 디버깅: 증상 → 진단 → 수정 45분 이내

### Phase 5 완료 시
- [ ] 토큰 사용량 실시간 표시 동작
- [ ] Phase 1 대비 50%+ 토큰 절감 확인

---

*이 작업 지시서는 각 Feature의 Why/What/How를 명확히 하고, Jira 티켓 단위로 분해하여 팀이 병렬로 작업을 진행할 수 있도록 구성했습니다.*
