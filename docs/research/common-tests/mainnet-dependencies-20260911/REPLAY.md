# 최신 코드로 다시 분석하는 절차

새 결과 디렉터리를 만들고 이 보존본을 덮어쓰지 않는다. 판정은 특정 시점의 소스에 대한 정적 검토이며 새 코드에 자동 승계할 결론이 아니다. 이전 결과는 비교 대상으로만 쓴다(`analyses/counter-review.md`가 그 예다).

이번 실행에 쓴 스크립트는 전부 `scripts/`에 있다. 경로는 스크립트 안에 절대경로로 박혀 있으므로 새 환경에서는 먼저 고친다.

1. 네 저장소의 HEAD와 tracked 변경을 기록한다. `source-manifest.json`에 tests/tc 파일 해시를 남긴다. 분석 중 다른 세션이 저장소를 바꿀 수 있으므로 미러 시점 HEAD를 따로 적고, 끝날 때 미러와 작업 트리를 다시 비교한다(이번엔 go-wbft가 움직였다).
2. 빌드 선택 파일을 뽑는다.

```bash
cd go-wemix   && go list -mod=readonly -e -deps -json ./cmd/gwemix > $OUT/build/go-wemix-darwin-arm64.json
cd go-wbft    && go list -mod=readonly -e -tags=urfave_cli_no_docs,ckzg -deps -json ./cmd/gwemix > $OUT/build/go-wbft-darwin-arm64-release.json
cd go-stablenet && go list -mod=readonly -e -tags=urfave_cli_no_docs,ckzg -deps -json ./cmd/gstable > $OUT/build/go-stablenet-darwin-arm64-release.json
python3 scripts/select_build_files.py <project> <repo> <golist.json> build-selected/<project> build/<project>-selected.json
```

   `-e`를 썼으면 결과의 `errors`가 비어 있는지 확인한다. go-wbft의 Makefile 타깃은 `gwemix`이고 산출물 이름도 `gwemix`다.
3. AST 그래프를 만든다. codemine 플러그인의 venv site-packages를 PYTHONPATH로 준다.

```bash
R=~/.claude/plugins/cache/0xmhha-devkit/codemine/0.5.2
env PYTHONPATH=$R/.venv/lib/python3.12/site-packages python3 $R/scripts/extract_graph.py --repo build-selected/<p> --out graph/<p> --module-depth 8
python3 $R/scripts/graph_report.py --graph graph/<p>/code-graph.json --top 25 > graph/<p>-report.md
```

4. 테스트 의존 필드를 뽑는다: `python3 scripts/extract_tc_dependencies.py $OUT` → `analyses/tc-dependencies.json`. 필드 분류 규칙은 스크립트 안에 있다. 새 verb나 필드가 생기면 규칙을 더한다.
5. 케이스 단계를 읽고 판정한다. `scripts/classify_tc.py`의 규칙표를 현재 파일 내용으로 다시 검토한다. 파일 이름 패턴이 아니라 단계 내용이 근거다. 새 파일은 반드시 직접 읽는다. 명세 문서 행은 `scripts/parse_scenario_docs.py` → `scripts/classify_docs.py`.
6. 클라이언트 차이와 하네스 구조는 코드를 다시 읽고 `scripts/build_chain_diff_tx_fee_account.py`, `scripts/build_chain_diff_b.py`, `scripts/build_harness_draft.py`의 표를 고친다. 이 생성기들은 인용 줄의 코드 조각을 현재 파일에서 읽어 넣으므로 줄이 밀리면 생성이 실패한다. 그때 줄번호를 고친다.
7. 문서를 만든다: `python3 scripts/build_docs.py $OUT`(목록·후보·색인·대응표·반대 검토), 초안 두 개를 합쳐 `chain-differences.md`, 하네스 초안을 `harness-dependencies.md`로. `configuration-and-change-scope.md`와 `README.md`는 손으로 쓴다.
8. 검증한다.

```bash
python3 scripts/verify_all_citations.py $OUT '{"go-wemix":"...","go-wbft":"...","go-stablenet":"...","chainbench":"..."}'
python3 $R/scripts/verify_citations.py --docs $OUT/analyses --repo <go-*·chainbench·run 심볼릭 링크를 모은 디렉터리>
python3 $R/scripts/lint_mermaid.py --docs $OUT/analyses
python3 scripts/validate_run.py $OUT
```

9. Confluence는 Atlassian MCP(`getConfluenceContentDescendants` → `getConfluenceContent` detail=full, markdown)로 폴더 2914091137 하위 페이지를 전부 읽어 `sources/confluence/<id>-<slug>.md`와 `index.json`을 만든다(이번엔 29개, 352,862자). 쓰기 도구는 쓰지 않는다. 내부망 IP·enode·원격 스크립트 본문은 저장 전에 뺀다. 그다음 `scripts/classify_confluence.py`로 명세 ID 단위 판정(`analyses/confluence-spec-catalog.json/.md`)을 만들고 `tc-spec-id-crosswalk.json`의 ID 존재를 확인한다. 페이지 버전(`index.json`의 version)이 바뀐 페이지만 다시 읽으면 된다.
10. 실행 검증은 별도 작업이다. "설정 분리" 판정 케이스를 세 체인 env로 실제 돌린 결과를 기록해야 PASS를 말할 수 있다.
