# Web 실행 경로 CLI 전환 리팩토링 인수인계

> 작성 2026-10-12, 브랜치 `refactor/web-cli-adapter`(main `b9e5be39`에서 분기, 원격에 push하지 않음).
> 이 문서는 스냅숏이다. 이어받는 세션은 먼저 `git status`, `git log --oneline main..HEAD`로 현재 상태를 확인한다.
> 이 파일은 커밋하지 않는다(이전 인수인계 문서와 같은 관례). 결정과 단계별 결과의 정본은 커밋된
> `docs/dev/architecture/web-execution-adapter-decision.md`이다.

## 1. 왜 이 작업을 하는가

PR #443(Web UI)은 2026-10-12에 main으로 squash merge되었다(`b9e5be39`). 그 Web 대시보드는 `internal/app` 유스케이스를
프로세스 안에서 직접 부르고 그 위에 Web 전용 실행 조율 코드를 쌓았다. 그래서 같은 동작이 CLI와 Web 두 경로로 갈라졌고,
누락 항목을 닫는 동안 나온 결함 대부분이 그 Web 전용 층에서 나왔다. 사용자는 PR 병합 뒤 Web이 chainbench CLI를 부르도록
리팩토링하기로 했고, 다음 네 가지를 모두 권장안대로 정했다(결정 문서 6절).

1. Web은 chainbench CLI를 하위 프로세스로 실행한다(MCP는 요청을 하나씩만 처리하고 취소와 호출별 자격증명이 없어서 고르지 않았다).
2. 작업 동안 SSH 키를 0700 작업 디렉터리의 0600 임시 파일로 두고 끝나면 지우는 예외를 허용한다. 철회는 하위 프로세스를 멈춘다.
3. 자원 점유 기록과 프로세스 신원 확인은 CLI 코어로 옮긴다. 계획 검토 지문은 화면 개념이라 Web에 남긴다.
4. Seed 문구를 "Web은 chainbench CLI를 하위 프로세스로 실행한다"로 고쳤다(`docs/dev/architecture/chainbench-web-ui.seed.yaml`).

## 2. 구조 요약 (지금 코드가 어떻게 되어 있는가)

- `internal/webcli`(L6): CLI 실행기. 바이너리 찾기(`FindBinary`: 명시 경로, 대시보드 옆, PATH 순), 취소 시 SIGINT 후 30초 대기,
  CLI가 낸 마지막 오류 줄을 그대로 돌려준다. 모든 명령에 `CHAINBENCH_STOP_WITH_PARENT=1`을 넣어서 대시보드가 죽으면 CLI도 첫
  인터럽트처럼 정리하고 멈춘다. 대시보드는 CLI를 찾지 못하면 시작하지 않는다(`-chainbench` 플래그로 경로 지정 가능).
- `internal/app`은 `ChainCLI` 인터페이스(`ChainStatus`, `Run`)만 안다. `cmd/chainbench-dashboard/main.go`가 엔진, 모니터, 매니페스트
  저장소에 `UseCLI(webcli.Runner{...})`로 연결한다.
- 자격증명: 작업마다 `writeJobServerSet`(`internal/app/deployment_job_serverset.go`)이 바인딩된 로그인을 0600 파일로 쓰고, 그 파일을
  가리키는 호스트별 `ssh` 블록을 담은 서버 집합을 0700 디렉터리에 만든다. CLI에는 `--server-logins <file>`로 넘기고, 네트워크가
  기록하는 서버 집합(`--server-set`)은 비밀이 없는 원본을 그대로 쓴다. 작업이 끝나면 디렉터리를 지운다(`jobLogins`, `runControl`).
- Web 테스트와 검증에서 CLI 대역이 필요한 단위 테스트는 가짜 CLI(`statusCLI`, `recordCLI`, `runCLI`, `uploadsCLI`, `composeCLI`,
  `observeCLI`)를 쓴다. 실제 CLI 경로는 `internal/webcli` 테스트(실제 바이너리 빌드)와 실제 바이너리 증명이 확인한다.

## 3. 단계별 완료 현황

결정 문서 5절에 커밋별 기록이 있다. 요약:

| 단계 | 상태 | 핵심 커밋 |
|---|---|---|
| P0 CLI 기반 | 완료 | `node swap`/`node reset` CLI·MCP `3cd2dd05`, 잠금 경쟁 수정 `7ab443f7` |
| P1 실행기와 읽기 | 완료 | `internal/webcli`와 네트워크 목록 `b72204c7`, 모니터링 `06dc3271` |
| P2 테스트 작업 | 완료 | run 입력 `8292ce78`, 호스트별 ssh `53aec996`, `--suite` `c4896ee0`, test.run `07d36ee7`, 읽기 전용 attach `91c082c1`·`6b410863`, 부모 종료 시 정지 `673c0735`, 재시작 점유 경합 `9259f64c` |
| P3 체인 구성 | 완료(계획만 남음) | `--stage init` `df917b15`, `--chain-preset` `16f1b7ee`, `--server-logins` `a65d3971`, 계약 회귀 `ed2b2fd9`, chain up 전환 `c029caf5`, 매니페스트 setup `90c7bd87`, file upload 보강 `8d4aa365`, 테스트 바이너리 `c88c2f7c` |
| P4 노드 제어 | 완료 | CLI 코어 신원 확인 `1356c2a6`, 제어 명령 로그인 `121d308b`, 노드 제어와 정리 `b4d7d704`, 교체 바이너리 `32ab7e40`, `ApplyManifest` 삭제 `e712301f`, `chain observe` `5bba86b6` |
| 관찰 중복 제거 | 완료 | `25af0da1`: Web 노드 관찰이 `chain observe`를 읽고, Web의 프로세스 관찰 복사본 3개 파일 삭제 |
| P5 정리와 최종 인수 | 남음 | 5절 |

Web 코드에서 네트워크나 대상을 바꾸는 경로는 모두 CLI를 거친다. 확인 방법:
`grep -nE '\b(ChainNew|ChainUp|ChainAllocate|ChainKeys|ChainGenesis|ChainConfig|ChainLaunchOpts|ChainProvision|ChainInit|ChainStart|ChainStop|ChainRm|ChainRestart|NodeStart|NodeStop|NodeSwap|NodeReset|RunSuite|AttachRun|Upload)\(ctx' internal/app/web_*.go internal/app/deployment*.go internal/app/manifest*.go`
가 아무것도 내지 않아야 한다.

## 4. 이 세션 마지막 변경 (관찰 중복 제거)

- `internal/app/web_node_observations.go`: 노드별 프로세스 판정과 실행 가능 여부를 `chainbench chain observe --json`(호출자 바인딩으로
  만든 로그인)에서 읽는다. 기록된 대상 머신 비교, 원장 일치, 바이너리 바인딩, 제어 정책은 Web에 남았다.
- `verifyWebNodeProcesses`(`web_node_controls.go`)와 `Reconcile`(`web_job_reconcile.go`)은 코어 `process.ProcessTableOf`,
  `ObserveLaunch`, `ObserveVacancy`를 쓴다(아직 프로세스 안 읽기다).
- 삭제: `web_node_probe.go`, `web_node_observer.go`, `web_node_discovery.go`와 그 테스트. 테스트는
  `internal/core/process/ownership_test.go`로 옮겼다.
- 확인: 단위 테스트, `make check` 통과, WEB-06·WEB-07·WEB-08·WEB-12 실제 증명 통과. 커밋 `25af0da1`.

## 5. 남은 일 (다음 세션의 작업 목록)

각 항목은 실패하는 테스트를 먼저 확인하고(RED), 관련 WEB 기준의 실제 증명으로 회귀를 확인한 뒤 커밋한다.

1. **읽기 전용 경로의 CLI 이동(결정 범위 안).** 아직 프로세스 안에서 엔진이나 SSH를 직접 읽는 곳:
   - 테스트 계획 `testengine.PlanSuiteLayout`(`web_test_run.go`의 prepareTestRun과 executeTestRun 재계획) → `chainbench run --plan --json`.
     JSON에 계획(`ComposePlan`)과 레이아웃(요청과 노드 표)이 다 담기는지 먼저 확인한다.
   - 프리셋 계획 `testengine.PlanChainPreset`(`web_preset_composition.go`) → `chain up --chain-preset ... --plan` 같은 계획 출력이 필요하다(없음).
   - 원격 로그 수집 `WebMonitor.CollectRemoteLogs`(`web_monitor_remote.go`), 자격증명 확인 `CheckCredential`, 노드 머신 동일성 확인
     (`opener.Inspect`), 사전 확인 `verifyWebNodeProcesses`, 재시작 재확인 `Reconcile`.
   - 읽기라서 위험은 낮다. 옮길지, 어디까지 옮길지는 진행 전에 사용자에게 짧게 확인하는 편이 좋다(지연과 비용이 생긴다).
2. **Web 전용 코드 정리(P5).** 이동 뒤 쓰이지 않게 된 코드를 지운다. 후보: `app.Deps.ServerLookup`와 `remote.Credentials.BeforeDial`
   (Web 메모리 자격증명 seam. 읽기 경로가 남아 있는 동안은 아직 쓰인다), `testengine.PlanChainPreset`의 Web 전용 사용 등.
   `grep`으로 생산 코드 사용처가 0인지 확인하고 지운다.
3. **MCP 대응.** 이번에 CLI에 더한 입력 중 MCP 도구에 없는 것: `run`의 `--binary-for`, `--read-only-keys`, `--suite`,
   `--no-node-control`, `--server-logins`; `chain up`의 `--stage init`, `--chain-preset`, `--server-logins`; `file upload`의
   `--sha256`, `--mode`. 모듈 원칙(CLI와 MCP가 같은 유스케이스)에 따라 필요한 것을 더할지 사용자와 정한다.
4. **최종 인수(P5).** 14개 기준을 고정 소스로 다시 통과시킨다. 이전 방식 그대로:
   - `/private/tmp/claude-501/frozen-final.sh`(capture, WEB-14는 `GWBFT_BIN` 등 포함)와 `frozen-recheck.sh`
     (`bash tests/webui/verify.sh --criterion X --require-live` 독립 재현). 실행 중에는 저장소 파일을 고치지 않는다.
   - 리팩토링 중 각 기준을 개별 capture로 통과시켰지만(아래 표), 고정 소스 한 주기와 독립 재현, WEB-14(race 전체와 라이브 게이트),
     WEB-10은 아직 돌리지 않았다.
   - 끝나면 영수증(`docs/dev/web-ui-evidence/WEB-*`)을 갱신해 커밋하고, push와 PR 생성(영어, draft, SAGE 리뷰 요청 금지, CODEOWNERS 확인)은
     사용자에게 물은 뒤 한다.
5. **사전 결함(이번 작업과 무관, 보고만 함).**
   - `cmd/chainbench/suitecmd`의 기존 테스트 3개가 `156ccbd8`부터 없는 파일 `tests/tc/basic/01-basic-consensus.json`을 가리켜서
     "파일 없음"만 확인한다.
   - `internal/dashboard` `TestBusyStartNamesTheHolderOverHTTP`가 임시 폴더 정리 경합으로 드물게 실패한다(30회 반복에서는 재현 안 됨).

## 6. 리팩토링 중 실제 증명 결과 (개별 capture)

`python3 tests/webui/capture.py WEB-xx chainbench-out/web-ui-acceptance`로 돌렸다. 마지막 통과 시점:

| 기준 | 마지막 통과 이후 바뀐 것 |
|---|---|
| WEB-01, 02, 03, 04, 11 | P3 중 통과, 그 뒤 P4와 관찰 변경 |
| WEB-05, 09, 13 | P4(노드 제어 CLI 전환) 뒤 통과 |
| WEB-06, 07, 08, 12 | 관찰 중복 제거 뒤 통과(이 세션 마지막, 최종 커밋 `25af0da1` 소스) |
| WEB-10, WEB-14 | 리팩토링 뒤 아직 돌리지 않음 |

로컬 영수증과 현재 소스를 비교하는 Python 테스트(`tests/webui/test_evidence_web*.py`)는 해당 기준을 다시 capture하기 전까지
"artifact changed"로 실패한다. 이것은 신선도 검사라서 최종 인수에서 다시 capture하면 통과한다.

## 7. 이 세션 종료 시점 상태

- 브랜치 `refactor/web-cli-adapter`, push하지 않음. 커밋되지 않은 추적 파일 변경은 없다(마지막 커밋 `6c54ffbf`).
- 디스크: `/private/tmp` 여유 약 16~19 GiB. `/private/tmp/cbui` 약 25 GiB 중 다수가 영수증에 참조되어 정리 규칙상 보호된다.
  `/private/tmp/chainbench-go-cache`(7.3 GiB)는 재생성 가능하므로 공간이 부족하면 `GOCACHE=/private/tmp/chainbench-go-cache go clean -cache`.
  정리 스크립트: `python3 /private/tmp/cbui-cleanup.py --delete`(인자 없이 돌리면 목록만 보인다). 20 GiB 미만일 때 쓰고, 60분 안에 바뀐 것,
  사용 중인 것, 기준별 최신 recheck, 영수증이 참조하는 것은 남긴다. `/private/tmp`는 재부팅 때 지워지므로, 없으면 같은 규칙으로 다시 쓴다.
  고정 인수 스크립트 `/private/tmp/claude-501/frozen-final.sh`, `frozen-recheck.sh`도 같은 이유로 사라질 수 있다.
- 교체 증명용 바이너리 `/private/tmp/chainbench-web-replacement-native/gwbft`(약 63 MB, 2026-10-11 생성)도 `/private/tmp`에 있어서
  재부팅 때 사라진다. 어떻게 빌드했는지는 이 세션에서 확인하지 않았다. 없으면 이전 인수인계 문서(`docs/dev/web-ui-claude-handoff.md`)에서
  만든 방법을 찾고, 찾지 못하면 사용자에게 묻는다.
- Docker 컨테이너는 쓰지 않을 때 멈춘다(`chainbench-server1` 등).
- `git stash list`에 `autostash`가 있다. 이 작업이 만든 것이 아니므로 건드리지 않는다.

## 8. 이 세션 마지막 결과

- 관찰 중복 제거를 `25af0da1`로, 결정 문서의 커밋 해시 정정(`ee81430f` → 메시지를 고친 뒤의 `1356c2a6`)을 `6c54ffbf`로 커밋했다.
- WEB-06은 처음에 "source changed during real SSH transfer interruption"으로 실패했다. 실행 중에 결정 문서와 인수인계 문서를 고친
  내 실수였다(고정 실행 중 저장소 파일 수정 금지 규칙 위반). 아무 파일도 건드리지 않고 다시 돌려 통과했다(975초).
  다음 세션도 실제 증명이 도는 동안에는 문서를 포함해 저장소 파일을 고치지 않는다.

## 8-1. 작업한 머신 (2026-10-12 기준)

- MacBook Air(Mac14,2), Apple M2, CPU 8코어, 메모리 8 GB, macOS 26.6.2(빌드 25G83), arm64. 호스트 이름 `Kevinui-MacBookAir.local`.
- Go `go1.26.8 darwin/arm64`. 저장소 `/Users/0xtopaz/work/github/0xmhha/chainbench`, 외부 체인 소스 `/Users/0xtopaz/work/github/0xmhha/chain/`.
- 디스크 228 GB 중 데이터 볼륨 여유 약 20 GB. `/private/tmp/cbui`(실제 증명 런타임)가 약 25 GB를 쓰고, 대부분이 영수증에 참조되어 정리 규칙상 남는다.
- 메모리가 8 GB라서 실제 증명은 반드시 하나씩 직렬로 돌린다. 여러 노드를 띄우는 증명 두 개를 겹치면 메모리와 포트가 부족하다.
- 다른 머신에서 이어받으면 위 경로, `/private/tmp`의 스크립트와 바이너리(9절과 부록), 외부 체인 바이너리 빌드, Docker 이미지
  `chainbench-server:ubuntu24`가 그 머신에 있는지 먼저 확인해야 한다. 이 세션에서 다른 머신에서의 실행은 확인하지 않았다.

## 9. 지켜야 할 규칙 (사용자 지시, 그대로 유지)

- 매 단계 전 전체 작업 목록과 완료 기준을 확인하고, 의미 있는 RED→GREEN과 필수 회귀를 통과한 변경만 명시 파일로 커밋한다.
  부분 완료를 전체 완료로 표시하지 않는다. 사용자 변경과 권한 거절 이력을 보존한다.
- `git add .`, `commit -a`, reset, clean, stash 금지. 커밋 메시지는 영어 Conventional Commits, 300자 이내, 이모지·co-author·Claude 표기 없음.
- 미추적 사용자 파일 보존: `CLAUDE.md`, `.ouroboros_eval_artifact.md`, 리팩토링 준비 문서들, `docs/dev/web-ui-claude-handoff.md`(덧붙이기만),
  `docs/dev/web-ui-claude-resume-prompt.md`, 이 문서.
- SAGE Core Team(`SAGE-X-project/sage-core-dev`)에 리뷰를 요청하지 않는다. PR은 영어, push와 merge는 사용자 확인 뒤.
  merge 명령은 권한 분류기가 막으므로 사용자가 `!`로 실행한다.
- 비밀은 암호화 저장소에만. 저장된 PID만으로 프로세스에 신호를 보내지 않는다(이제 CLI 코어가 강제한다).
- 2 GiB genesis preflight를 낮추지 않는다. Codex 루트 5개 삭제 거절을 우회하지 않는다. 외부 체인 소스를 고치지 않는다.
- 실제 바이너리만 쓴다(모의 프로세스로 인수하지 않음). 고정 실행 중 저장소 파일을 고치지 않는다. 실제 증명은 직렬로 돌린다.
- 환경: `WEBUI_RUNTIME_ROOT=/private/tmp/cbui`, `GOCACHE=/private/tmp/chainbench-go-cache`, `PYTHONDONTWRITEBYTECODE=1`,
  `WEBUI_REPLACEMENT_BINARY=/private/tmp/chainbench-web-replacement-native/gwbft`. WEB-14에는 `GWBFT_BIN`, `GSTABLE_BIN`, `GWEMIX_BIN`,
  `CHAINBENCH_DOCKER_SERVERS`가 필요하다(`frozen-final.sh` 참고).
- 실제 증명을 돌리기 전 세 가지를 확인한다: 관련 개발 증명과 단위 테스트, 디스크 여유(20 GiB 기준 정리), 환경 변수.
- ooo는 qa, lateral, status, evaluate에만 쓴다.
- 한국어 응답: 명사 나열 대신 문장, 줄표 금지, 배경부터, 근거 제시, 비교일 때만 표, 확인하지 않은 것은 그렇다고 쓴다.
- 새 진입점은 CLI나 MCP를 부르는 어댑터로 만든다. 같은 층에서 결함이 세 번 이상 나오면 멈추고 구조를 사용자와 다시 본다
  (메모리 `adapter-over-engine`).

## 10. 다음 세션에 붙여 넣을 시작 지시

```
chainbench 저장소(/Users/0xtopaz/work/github/0xmhha/chainbench)에서 Web 실행 경로를 chainbench CLI 호출로 바꾸는
리팩토링을 이어서 진행해. 브랜치는 refactor/web-cli-adapter이고 push하지 않은 상태야.

먼저 읽을 것:
1. docs/dev/web-cli-refactor-handoff.md (이 인수인계, 특히 5절 남은 일과 9절 규칙)
2. docs/dev/architecture/web-execution-adapter-decision.md (승인된 결정과 단계별 커밋 기록)
3. git status, git log --oneline main..HEAD 로 현재 상태 확인

다음으로 할 일은 인수인계 5절 순서야. 읽기 전용 경로를 CLI로 옮길 범위(5절 1번)와 MCP 대응(3번)은
진행 전에 나에게 짧게 확인해. 그 뒤 쓰이지 않는 Web 코드 정리, 14개 기준 고정 소스 최종 인수 순서로 가.
각 변경은 RED→GREEN과 관련 실제 증명을 통과한 뒤 명시 파일로 커밋하고, push와 PR은 나에게 묻고 해.
```

## 부록. 고정 인수 스크립트 원문 (재부팅으로 사라지면 이것으로 다시 만든다)

`/private/tmp/claude-501/frozen-final.sh` (capture):

```sh
#!/bin/zsh
# Final frozen-source acceptance: no repository edits while this runs.
cd /Users/0xtopaz/work/github/0xmhha/chainbench
export WEBUI_RUNTIME_ROOT=/private/tmp/cbui GOCACHE=/private/tmp/chainbench-go-cache PYTHONDONTWRITEBYTECODE=1
export WEBUI_REPLACEMENT_BINARY=/private/tmp/chainbench-web-replacement-native/gwbft
LOG=/private/tmp/chainbench-frozen.log
digest(){ (cd tests/webui && python3 -c "import sys;from pathlib import Path;from runtime_contract import source_digest;print(source_digest(Path(sys.argv[1])))" "$PWD/../..") }
npm --prefix web run build > /private/tmp/chainbench-frozen-spa.log 2>&1; echo "spa=$? status=$(git status --short internal/dashboard/spa | wc -l | tr -d ' ')" >> $LOG
D0=$(digest); echo "digest0=$D0" >> $LOG
for c in WEB-01 WEB-02 WEB-03 WEB-04 WEB-05 WEB-06 WEB-07 WEB-08 WEB-09 WEB-10 WEB-11 WEB-12 WEB-13; do
  s=$(date +%s)
  python3 tests/webui/capture.py $c chainbench-out/web-ui-acceptance > /private/tmp/chainbench-frozen-$c.log 2>&1
  code=$?
  D=$(digest); echo "$c exit=$code seconds=$(( $(date +%s)-s )) digestSame=$([[ $D == $D0 ]] && echo yes || echo NO)" >> $LOG
done
docker start chainbench-server1 >/dev/null 2>&1; sleep 3
s=$(date +%s)
GWBFT_BIN=/Users/0xtopaz/work/github/0xmhha/chain/go-wbft/build/bin/gwemix GSTABLE_BIN=/Users/0xtopaz/work/github/0xmhha/chain/go-stablenet/build/bin/gstable GWEMIX_BIN=/Users/0xtopaz/work/github/0xmhha/chain/go-wemix/build/bin/gwemix CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build python3 tests/webui/capture.py WEB-14 chainbench-out/web-ui-acceptance > /private/tmp/chainbench-frozen-WEB-14.log 2>&1
code=$?
docker stop chainbench-server1 >/dev/null 2>&1
D=$(digest); echo "WEB-14 exit=$code seconds=$(( $(date +%s)-s )) digestSame=$([[ $D == $D0 ]] && echo yes || echo NO)" >> $LOG
echo FROZEN-DONE >> $LOG
```

`/private/tmp/claude-501/frozen-recheck.sh` (독립 재현):

```sh
#!/bin/zsh
# Independent reproduction of all fourteen criteria on the frozen source; no repository edits while this runs.
cd /Users/0xtopaz/work/github/0xmhha/chainbench
export WEBUI_RUNTIME_ROOT=/private/tmp/cbui GOCACHE=/private/tmp/chainbench-go-cache PYTHONDONTWRITEBYTECODE=1
export WEBUI_REPLACEMENT_BINARY=/private/tmp/chainbench-web-replacement-native/gwbft
LOG=/private/tmp/chainbench-recheck.log
digest(){ (cd tests/webui && python3 -c "import sys;from pathlib import Path;from runtime_contract import source_digest;print(source_digest(Path(sys.argv[1])))" "$PWD/../..") }
D0=$(digest); echo "digest0=$D0" >> $LOG
for c in WEB-01 WEB-02 WEB-03 WEB-04 WEB-05 WEB-06 WEB-07 WEB-08 WEB-09 WEB-10 WEB-11 WEB-12 WEB-13; do
  s=$(date +%s)
  bash tests/webui/verify.sh --criterion $c --require-live --output chainbench-out/web-ui-acceptance > /private/tmp/chainbench-recheck-$c.log 2>&1
  code=$?
  D=$(digest); echo "$c exit=$code seconds=$(( $(date +%s)-s )) digestSame=$([[ $D == $D0 ]] && echo yes || echo NO) $(tail -1 /private/tmp/chainbench-recheck-$c.log)" >> $LOG
done
docker start chainbench-server1 >/dev/null 2>&1; sleep 3
s=$(date +%s)
GWBFT_BIN=/Users/0xtopaz/work/github/0xmhha/chain/go-wbft/build/bin/gwemix GSTABLE_BIN=/Users/0xtopaz/work/github/0xmhha/chain/go-stablenet/build/bin/gstable GWEMIX_BIN=/Users/0xtopaz/work/github/0xmhha/chain/go-wemix/build/bin/gwemix CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build bash tests/webui/verify.sh --criterion WEB-14 --require-live --output chainbench-out/web-ui-acceptance > /private/tmp/chainbench-recheck-WEB-14.log 2>&1
code=$?
docker stop chainbench-server1 >/dev/null 2>&1
D=$(digest); echo "WEB-14 exit=$code seconds=$(( $(date +%s)-s )) digestSame=$([[ $D == $D0 ]] && echo yes || echo NO) $(tail -1 /private/tmp/chainbench-recheck-WEB-14.log)" >> $LOG
echo RECHECK-DONE >> $LOG
```
