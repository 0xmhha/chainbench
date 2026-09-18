# 최신 패치로 다시 검토하기

이번 결과는 특정 입력 문서, tests/tc, 로컬 dev와 PR head를 고정한 정적 분석이다. 다음 실행은 새 출력 디렉터리를 만들고 이전 결과를 덮어쓰지 않는다.

1. `gh pr view 196 --repo wemixarchive/go-wemix --json baseRefOid,headRefOid,files,body,state`로 PR 기준을 다시 읽는다. 최종 배포 대상 SHA는 PR head와 다를 수 있으므로 별도 기록한다.
2. 실제 적용할 저장소의 HEAD, branch, 작업 트리 변경을 기록한다. 이전 dev/master 분기가 해결되었는지와 로컬 기존 수정이 유지되는지 확인한다.
3. 커밋 소스는 `git archive <SHA>`로 별도 디렉터리에 보존한다. 커밋되지 않은 변경이 검증 대상이라면 그 변경까지 복사하고 수집 전후 해시를 비교한다. 사용자 작업 트리에서 checkout/reset을 하지 않는다.
4. 두 Markdown 원본과 tests/tc 전체를 다시 복사하고 경로·행·파일 해시를 남긴다. 원본 행과 파일의 추가·삭제·변경을 비교한다.
5. 기존 codemine 스크립트로 AST를 다시 추출한다. 아래 명령의 경로는 새 snapshot과 출력 경로로 바꾼다.

```bash
PYTHONPATH="$CODEMINE_SITE_PACKAGES" "$CODEMINE_PYTHON" \
  "$CODEMINE_PLUGIN_ROOT/scripts/extract_graph.py" \
  --repo /absolute/path/to/new/source \
  --out /absolute/path/to/new/graph \
  --module-depth 8 --exclude testdata --exclude tests --exclude tools

go run "$CODEMINE_PLUGIN_ROOT/scripts/go_inventory.go" \
  --repo /absolute/path/to/new/source --root . \
  --out /absolute/path/to/new/full-go-inventory.json
```

이번 환경은 codemine 0.5.2의 기존 site-packages와 시스템 Python 3.12를 함께 썼다. venv interpreter 링크가 없었기 때문이다. 플러그인 경로는 `/Users/wm-it-25_0220/.codex/plugins/cache/0xmhha-devkit/codemine/0.5.2+codex.20260908153838`, site-packages는 그 아래 `.venv/lib/python3.12/site-packages`다. 다른 환경에서는 변수를 실제 설치 경로로 설정한다. Go는 실행 환경의 캐시 권한이나 도구 체인 준비가 필요할 수 있다.

6. 문서 74행과 TC 189개라는 이전 수치를 고정 답으로 쓰지 않는다. 새 입력을 전수 추출한 뒤 새 개수를 검증한다. 원본 ID와 script 참조를 보존하며 파일 존재와 실제 실행 상태를 분리한다.
7. `scripts/catalog_documents.py`와 `crosswalk_documents.py`는 이번 입력의 수작업 판정을 저장한 재생성 도구다. **자동 의미 분석기가 아니다.** 새 입력의 행 순서나 내용이 바뀌면 분류표와 대응표를 다시 검토해야 한다. 그대로 실행해 이전 판정을 새 코드의 결론으로 사용하지 않는다.
8. 추가 N-001~N-016의 구현·미구현과 PR 기존 테스트의 확장을 확인한다. 이미 추가된 테스트를 다시 신규 제안으로 남기지 않는다. 기존 5개 Go 테스트는 이름과 내용의 보존을 각각 확인한다.
9. 필수 경로 3개 이상을 반대 검토한다. 특히 snap의 pivot 전후, 최종 RLP와 추정치, body/receipt batch를 구분한다.
10. 인용·그림·traceability model을 새 snapshot 기준으로 검사한다. 정적 분석과 실제 실행 증거를 다른 필드에 기록한다.

```bash
python3 "$CODEMINE_PLUGIN_ROOT/scripts/verify_citations.py" \
  --docs /absolute/path/to/new/run/analyses --repo /absolute/path/to/new/run
python3 "$CODEMINE_PLUGIN_ROOT/scripts/lint_mermaid.py" \
  --docs /absolute/path/to/new/run/analyses
python3 "$CODEMINE_PLUGIN_ROOT/scripts/validate_project_model.py" \
  --model /absolute/path/to/new/run/project-model.json
```

`project-model.json`은 대표 코드 경로와 시나리오 ID의 추적용이다. 전체 기능 구현의 보증이나 실행 결과가 아니다. 완전한 import 관계는 별도 graph가 기준이다. `workflow.json`은 분석 산출물 제출 단계로 두었고 구현 승인은 기록하지 않았다.

다음 요청 예시:

> PR #196의 최신 패치와 실제 통합할 go-wemix SHA를 기준으로 이 검토를 다시 수행해줘. 원본 문서와 tests/tc 항목을 전수 비교하고 기존 133개 수행 후보 및 N-001~N-016의 유지·변경·해결 여부를 정리해줘. 소스와 테스트는 수정하지 말고 새 분석 디렉터리에 기록해줘.
