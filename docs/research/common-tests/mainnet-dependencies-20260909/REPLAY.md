# 최신 코드로 다시 분석하는 절차

새 결과 디렉터리를 만들고 이번 보존본을 덮어쓰지 않는다. 이 자료의 판정은 특정 소스와 실행 프로필에 대한 정적 검토이며 새 코드에 자동 승계할 결론이 아니다.

1. 세 저장소와 chainbench의 HEAD, tracked 변경, 분석에 사용할 untracked 입력을 기록한다. 현재 작업 파일을 수정하지 않고 별도 sources 디렉터리에 복사한다. 읽기 전후 해시가 다른 파일은 다시 수집한다.
2. 두 Markdown 원본과 tests/tc의 모든 파일을 새로 수집한다. 이전 45행·29행·189개라는 수치를 고정 답으로 사용하지 않는다. 추가·삭제·내용 변경을 해시와 원본 ID로 비교한다.
3. Makefile의 실제 타깃에서 build/ci.go를 따라가 GOOS, GOARCH, CGO, 태그와 진입 패키지를 확인한다. 바이너리 파일 이름과 chainbench manifest의 이름이 다른 경우 별도 기록한다.
4. 각 저장소에서 아래와 같은 의존성 목록을 추출한다. `-mod=readonly`를 사용한다. `-e`를 사용했다면 프로세스 종료 코드만 보지 않고 모든 package의 Error/DepsErrors가 비어 있는지 확인한다. 필요한 모듈이 없을 때는 정상적으로 확보한 뒤 재추출하고 불완전한 목록을 확정 결과로 쓰지 않는다.

```bash
# 각 대상 저장소에서 실행. out은 새 결과 디렉터리의 절대경로다.
go list -mod=readonly -deps -json ./cmd/gwemix > "$out/go-wemix-darwin-arm64.json"

# go-wbft의 make gwemix 조건
go list -mod=readonly -tags=urfave_cli_no_docs,ckzg -deps -json ./cmd/gwemix > "$out/go-wbft-darwin-arm64-release.json"

# go-stablenet의 make gstable 조건
go list -mod=readonly -tags=urfave_cli_no_docs,ckzg -deps -json ./cmd/gstable > "$out/go-stablenet-darwin-arm64-release.json"

# Linux 대상 예: 실제 해당 프로젝트의 추가 태그/옵션도 함께 반영
env GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
  go list -mod=readonly -tags=rocksdb -deps -json ./cmd/gwemix > "$out/go-wemix-linux-amd64-release.json"
```

5. 연속 JSON 객체인 go list 출력을 순차 디코딩한다. 프로젝트 내부 Dir의 GoFiles+CgoFiles를 선택하고 원래 상대경로를 유지하여 별도의 build-selected 디렉터리에 복사한다. symlink는 실제 경로로 정규화한다. 테스트 파일, IgnoredGoFiles, 다른 command의 파일은 포함하지 않는다. C/assembly/embed와 외부 의존은 별도 목록으로 남긴다. 이번 `build/*-selected.json`이 결과 형식 예시다.
6. 기존 codemine 추출기로 선택 파일만 AST 파싱한다. 새 모듈을 외부에서 실행하는 대신 설치된 도구를 사용한다. 이번 환경은 Python 3.12와 codemine의 tree-sitter site-packages를 사용했다.

```bash
env PYTHONPATH="$CODEMINE_PLUGIN_ROOT/.venv/lib/python3.12/site-packages" \
  python3 "$CODEMINE_PLUGIN_ROOT/scripts/extract_graph.py" \
  --repo "$selected_source" --out "$graph_output" --module-depth 8

python3 "$CODEMINE_PLUGIN_ROOT/scripts/graph_report.py" \
  --graph "$graph_output/code-graph.json"
```

7. go list의 전체 ImportPath/Imports로 별도의 정확한 패키지 그래프를 만든다. AST 추출기의 이름 기반 간선을 완전한 함수 호출 그래프로 해석하지 않는다. 소스 인용 파일이 선택 파일 목록에 포함되는지도 검사한다.
8. 원본 테스트의 steps와 기대 결과를 읽고 공통성 및 go-wemix 적용성을 다시 판정한다. 문서 명세 구현, JSON 환경 변경, 체인별 기대값 계산, 기능 미지원, 외부 환경 미확인을 구분한다. TC-128처럼 한 체인에 대응 기능이 없으면 임의의 설정값이나 SKIP으로 공통 성공을 만들지 않는다.
9. dependency 필드는 JSON key와 step 문맥으로 분류한다. description/schemaVersion을 RPC 의존으로 세지 않는다. 송금 수신 주소와 계약 주소, literal과 `$binding`, 계정 역할과 실제 키 값을 구분한다. 민감값은 문서에 재출력하지 않는다.
10. 최소한 수수료, 합의/장애, signer, fork gate에 대해 반대 검토한다. 이번 counter-review는 이전에 있었던 오류의 예시이며 새 판정의 정답표가 아니다.
11. 모든 목록과 요약 개수, 소스 해시, 인용, Mermaid를 검증한다. 실제 테스트 실행 여부는 별도 상태로 남긴다.

```bash
python3 "$CODEMINE_PLUGIN_ROOT/scripts/verify_citations.py" --docs "$run/analyses" --repo "$run"
python3 "$CODEMINE_PLUGIN_ROOT/scripts/lint_mermaid.py" --docs "$run/analyses"
```

현재 설치 경로는 `/Users/wm-it-25_0220/.codex/plugins/cache/0xmhha-devkit/codemine/0.5.2+codex.20260908153838`다. 새 실행 환경에서는 경로와 도구 체인을 먼저 확인한다. 이번 분석의 소스 스냅샷은 선택 Go 파일 외의 설명 자료도 포함한 보존본이며 바이너리를 바로 빌드할 수 있는 완전한 checkout은 아니다.
