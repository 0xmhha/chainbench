chainbench 저장소(/Users/0xtopaz/work/github/0xmhha/chainbench)에서 Web 실행 경로를 chainbench CLI 호출로 바꾸는 리팩토링을 이어서 진행해.
브랜치는 refactor/web-cli-adapter이고, main(b9e5be39) 위에 커밋 39개가 있으며 push하지 않은 상태야.

먼저 읽을 것:
1. docs/dev/web-cli-refactor-handoff.md — 인수인계 문서. 특히 5절(남은 일), 8-1절(머신과 환경), 9절(지켜야 할 규칙), 부록(고정 인수 스크립트).
2. docs/dev/architecture/web-execution-adapter-decision.md — 승인된 결정 네 가지와 단계별 커밋 기록.
3. git status, git log --oneline main..HEAD — 현재 상태 확인. 인수인계 문서는 스냅숏이니 실제 파일과 다르면 실제 파일을 따른다.

진행 순서는 인수인계 5절을 따른다:
1. 읽기 전용 경로(테스트 계획, 프리셋 계획, 원격 로그 수집 등)를 CLI로 옮길 범위와, 새 CLI 입력을 MCP 도구에도 넣을지는 시작 전에 나에게 짧게 확인한다.
2. 그다음 쓰이지 않게 된 Web 코드 정리, 마지막으로 14개 기준 고정 소스 최종 인수(capture와 독립 재현, WEB-10과 WEB-14 포함).

작업 규칙:
- 매 단계 전에 전체 작업 목록과 완료 기준을 확인한다. 실패하는 테스트를 먼저 확인하고(RED) 통과시킨 뒤(GREEN), 관련 실제 증명까지 통과한 변경만 명시한 파일로 커밋한다. 부분 완료를 전체 완료로 표시하지 않는다.
- 실제 증명이 도는 동안에는 문서를 포함해 저장소 파일을 고치지 않는다(지난 세션에서 이것 때문에 WEB-06이 한 번 실패했다). 실제 증명은 하나씩 직렬로 돌린다.
- 증명 전에 관련 단위 테스트, 디스크 여유(20 GiB 미만이면 python3 /private/tmp/cbui-cleanup.py --delete), 환경 변수(WEBUI_RUNTIME_ROOT=/private/tmp/cbui, GOCACHE=/private/tmp/chainbench-go-cache, PYTHONDONTWRITEBYTECODE=1, WEBUI_REPLACEMENT_BINARY=/private/tmp/chainbench-web-replacement-native/gwbft)를 확인한다.
- git add ., commit -a, reset, clean, stash는 쓰지 않는다. 커밋 메시지는 영어 Conventional Commits, 300자 이내, 이모지와 co-author와 Claude 표기 없이 쓴다.
- 미추적 사용자 파일(CLAUDE.md, 리팩토링 준비 문서, 인수인계 문서들)은 보존한다. docs/dev/web-ui-claude-handoff.md는 덧붙이기만 한다.
- push, PR 생성, merge는 나에게 묻고 한다. SAGE Core Team에는 리뷰를 요청하지 않는다.
- 답변은 한국어로, 배경부터 문장으로 설명하고, 확인하지 않은 것은 확인하지 않았다고 쓴다.
