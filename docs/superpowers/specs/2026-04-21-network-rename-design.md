# Network Rename Design (HAL → network)

> **작성일**: 2026-04-21
> **목적**: 기존 `hal/` Go 모듈·바이너리·문서 용어를 `network` 기반으로 전면 리네이밍한다. "HAL (Hardware Abstraction Layer)"은 하드웨어 드라이버 추상화를 뜻하는 관용어인데 본 프로젝트는 체인 네트워크를 추상화하므로 의미가 맞지 않는다. Sprint 2 착수 전에 용어를 정리한다.

---

## 1. 배경 · 결정 근거

- HAL은 플랜 초안에서 "예시 메타포"로 사용된 잠정 용어였음
- 실제 추상화 대상은 체인 네트워크 (local / remote / SSH-remote) × (stablenet / wbft / wemix / ethereum)
- `network`는 Go 표준 라이브러리와 충돌하지 않음 (stdlib는 `net`이며 `network`는 부재)
- VISION_AND_ROADMAP §5.1~5.16이 이미 "Network Abstraction" 프레이밍을 일관되게 사용 중
- 기존 Go 타입(`NetworkController`, `Network`, `Node`) 이름 재사용 가능 → 소스 코드 변경 최소

대안(`net`, `chainctl`, `broker`, `fabric` 등) 비교는 §10 참조.

---

## 2. Scope

### 2.1 변경 대상

- **Go 모듈**: 경로·디렉토리·바이너리
- **Go 코드**: `//go:generate` 경로, import path, `tools.go`, README
- **문서**: VISION_AND_ROADMAP, 플랜 문서, inline 주석
- **플랜 파일명**: `2026-04-20-hal-foundation.md` → `2026-04-20-network-foundation.md`
- **기타**: `.mcp.json`, `install.sh`, `settings.local.json` 등에서 `chainbench-hal` 참조

### 2.2 변경 제외 (현재 이름 그대로)

- Go 타입: `NetworkController`, `LocalDriver`, `RemoteDriver`, `Network`, `Node`, `Auth` — 이미 올바른 명명
- `lib/chain_adapter.sh`, `lib/adapters/*.sh` — 체인 어댑터(직교 축), 리네이밍 무관
- JSON Schema `$id` URI (`https://chainbench.io/schema/*.json`) — `hal` 참조 없음
- 기존 commit message · git 히스토리 — 불변

---

## 3. 리네이밍 매핑

| 현재 | 변경 후 | 종류 |
|---|---|---|
| `hal/` | `network/` | 디렉토리 |
| `github.com/0xmhha/chainbench/hal` | `github.com/0xmhha/chainbench/network` | Go 모듈 경로 |
| `hal/cmd/chainbench-hal/` | `network/cmd/chainbench-net/` | 바이너리 소스 경로 |
| `chainbench-hal` | `chainbench-net` | 빌드 산출 바이너리 이름 |
| `hal/schema/` | `network/schema/` | 스키마 경로 |
| `hal/internal/types/` | `network/internal/types/` | 생성 타입 경로 |
| `hal/tools.go` | `network/tools.go` | 툴 핀 파일 |
| `hal/README.md` | `network/README.md` | 모듈 README |
| "HAL" (개념 용어) | "Network Abstraction" 또는 "network layer" | docs 용어 |
| `docs/superpowers/plans/2026-04-20-hal-foundation.md` | `docs/superpowers/plans/2026-04-20-network-foundation.md` | Plan 파일명 |

Package 이름(subdir 기준):
- `network/cmd/chainbench-net/`: `package main` (변경 없음)
- `network/schema/`: `package schema` (변경 없음)
- `network/internal/types/`: `package types` (변경 없음)
- `network/tools.go`: `package tools` (빌드 태그 `tools`, 변경 없음)

→ import 경로만 `github.com/0xmhha/chainbench/hal/...` → `.../network/...`

---

## 4. 바이너리 이름 결정

**`chainbench-net`** 채택.

근거:
- 기존 sibling 바이너리 `chainbench-mcp` 와 동일 접미사 스타일 (3-글자 축약)
- `kubectl`/`systemctl` 축약 관례
- 모듈 경로 `.../network` 와 바이너리명 `chainbench-net` 은 독립 축 — 충돌 없음
- `chainbench-network` 은 길고 중복감(chainbench + network + chain context)

---

## 5. 개념 용어 변경 방침

### 5.1 소스/주석/README

"HAL" 리터럴 → "network" 로 치환 (소문자 기술 용어).

### 5.2 문서 (VISION_AND_ROADMAP)

- 섹션 제목 "§5.15 HAL 아키텍처 상세" → "§5.15 Network Abstraction 아키텍처 상세"
- 본문 "HAL 인터페이스" → "Network Abstraction 인터페이스" 또는 "NetworkController 인터페이스"
- Diagram annotation "← HAL boundary" → "← Network Abstraction boundary"

### 5.3 Android HAL 메타포 (§5.15 서두)

전면 리프레임:

> **영감 (Inspiration)**: Android HAL의 "상위는 하위 구현을 모르고 command-in / event-out 인터페이스로만 통신" 패턴에서 영감을 받음. 단, 본 프로젝트는 **하드웨어가 아닌 체인 네트워크**를 추상화하므로 `Hardware Abstraction Layer` 명칭 대신 **`Network Abstraction`** 으로 명명한다.

"HAL 메타포" 표현은 모두 "Network Abstraction (HAL 패턴에서 영감)" 또는 단순히 "Network Abstraction" 으로 치환.

---

## 6. 마이그레이션 전략

### 6.1 단일 atomic 커밋

커밋 메시지: `refactor: rename HAL module to network`

**선택 근거**:
- 중간 상태(모듈 경로만 바뀐 시점, 바이너리명만 바뀐 시점 등) 전부 빌드 불가 → 분할 커밋 의미 없음
- 리뷰어가 전체 의도를 한 diff로 파악
- Bisect 시 단일 boundary로 regression 추적 명확

### 6.2 작업 순서 (단일 커밋 내부 작업 순서 — 중간 커밋 생성하지 않음)

아래 11 단계는 **하나의 working tree 편집 안에서 순차 수행** 후 마지막에 단일 커밋을 생성한다. 중간에 커밋하면 빌드 깨진 상태의 commit이 생기므로 피한다.

1. `git mv hal network` (디렉토리 rename, git history 보존)
2. `git mv network/cmd/chainbench-hal network/cmd/chainbench-net` (cmd dir rename)
3. `network/go.mod` 모듈 경로 수정
4. 모든 `.go` 파일의 import path 일괄 치환
5. `network/README.md`, `network/internal/types/doc.go` 등 README·doc 내용 치환
6. `network/schema/schema.go` 내 주석 치환
7. VISION_AND_ROADMAP.md 일괄 치환 + §5.15 리프레임
8. Plan 파일명 rename (`2026-04-20-hal-foundation.md` → `network-foundation.md`) + 내용 치환
9. 기타: `.mcp.json`, `install.sh`, `settings.local.json` 등 외부 파일 스캔·수정
10. 빌드·테스트 검증
11. 단일 커밋 + push

---

## 7. 검증 체크리스트

리네이밍 직후:

- [ ] `cd network && go build ./...` — exit 0
- [ ] `cd network && go test ./...` — 모든 패키지 PASS
- [ ] `cd network && go vet ./...` — 경고 0
- [ ] `cd network && gofmt -l .` — 출력 empty
- [ ] `cd network && go build -tags tools ./...` — 툴 import 정상
- [ ] `go build -o bin/chainbench-net ./network/cmd/chainbench-net && ./bin/chainbench-net version` → `chainbench-net 0.0.0-dev` 출력
- [ ] `scripts/inventory/list-adapter-functions.sh` — 정상 동작
- [ ] `scripts/inventory/scan-binary-hardcoding.sh` — 정상 동작
- [ ] `git grep -iw 'hal' -- '*.go' '*.sh' '*.md' '*.json' '*.mod' ':!node_modules' ':!*.min.*'` 결과:
  - (a) §5.15의 의도적 "Android HAL" 영감 참조 1~2곳
  - (b) 기존 commit message 인용부 (역사적 맥락) — 허용
  - 그 외 **매치 없어야 함**
- [ ] `git grep -l 'chainbench-hal'` 결과 비어 있어야 함
- [ ] `git grep -l '/hal/'` 결과 비어 있어야 함 (URL 등 특수 케이스만 예외)

---

## 8. 리스크 · 완화

| 리스크 | 영향 | 완화 |
|---|---|---|
| 외부 소비자가 `github.com/0xmhha/chainbench/hal` import | 낮음 | origin/main에 방금 push됨, 외부 import 아직 없음 → push 시점 = 리네이밍 시점 동일 처리 |
| `.mcp.json`, `settings.local.json` 등에 `chainbench-hal` 참조 | 중간 | 사전 grep으로 발견 후 동시 업데이트 |
| `install.sh`/`README.md` 등 상위 설치 문서의 바이너리 이름 참조 | 중간 | 스캔 후 함께 수정 |
| Plan 문서 파일명 변경 시 과거 커밋·이슈의 링크 깨짐 | 낮음 (아직 외부 참조 없음) | 허용 |
| Sprint 2 계획 초안이 HAL 명칭 전제로 작성됨 | 없음 (아직 미작성) | — |
| go.sum 해시 mismatch (모듈 경로 바뀌면 의존성 재해결) | 낮음 | `go mod tidy` 로 재정렬, lockfile 커밋 |

---

## 9. Out of Scope

아래는 본 리팩토링에서 건드리지 않는다:

- `chain_adapter.sh`, `lib/adapters/*.sh` (체인 어댑터 — 별도 축)
- JSON Schema `$id` URI
- Go 타입명 (`NetworkController` 등, 이미 올바름)
- 바이너리 version embed 방식 (Sprint 2 ldflags)
- `NetworkDriver` 인터페이스의 실제 구현 (Sprint 2)
- 모든 Sprint 2+ 기능

---

## 10. 대안 비교 (의사결정 기록)

| 이름 | Pros | Cons | 채택 |
|---|---|---|---|
| `hal` | 짧음 | 하드웨어 추상화 오인 | ✗ 현재, 폐기 대상 |
| `net` | 매우 짧음 | Go stdlib `net` 패키지와 이름 shadowing → alias 강제 | ✗ |
| **`network`** | stdlib 충돌 없음, VISION 문서와 일관, 의미 명확, 기존 Go 타입과 자연스럽게 매칭 | 상대적으로 긴 이름 | **✓ 채택** |
| `chainnet` | 도메인 명확 | 두 번째 n 중복, 관례 없음 | ✗ |
| `netctl` | kubectl 관례 | `ctl`이 너무 generic | ✗ |
| `fabric` / `plane` / `hub` | 역할 메타포 | 프로젝트 특수 지식 없이는 의미 불명, 블록체인/서비스메시 용어와 충돌 | ✗ |
| `broker` | Mediator 패턴 명확 | 메시지 큐(Redis/RabbitMQ) 연상, 기존 `NetworkController` 타입명과 불일치 | ✗ |

---

## 11. 완료 기준 (Definition of Done)

1. 단일 커밋 `refactor: rename HAL module to network` 생성 및 origin/main push
2. §7 검증 체크리스트 전체 통과
3. `git grep` 결과 HAL/hal 잔존 참조 = 의도된 2곳 이내
4. Sprint 2 작업이 이 새 네이밍 기준으로 착수 가능

다음 단계: `writing-plans` 스킬로 구현 플랜 작성 → `subagent-driven-development` 로 실행.
