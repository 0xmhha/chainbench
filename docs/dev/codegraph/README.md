# codegraph — AST 기반 코드 그래프

`internal/` + `cmd/` 의 비-테스트 Go 소스를 `go/parser`+`go/ast` 로 파싱해 패키지·심볼·
크로스패키지 호출 그래프를 만든다. DSL 파이프라인 배선을 확인하고, 외부 분석을 실제 코드와
대조하는 데 쓴다.

## 무엇을 담나

- **codegraph.json** — 전체 그래프. 패키지마다 `name`·`files`·`imports`(모듈 내부)·exported
  `types`·`funcs`, 그리고 호출 엣지 `fromPkg.fromFunc → toPkg.toSel`.
- **pipeline.md** — 파이프라인 패키지 간 호출을 집계한 mermaid.

크로스패키지 호출은 구문으로 해석한다: Go 는 그것을 반드시 `pkg.Func(...)` 로 쓰므로, 파일별
import alias 를 풀어 `alias.Sel` 선택자를 모으면 타입체크 없이 배선이 잡힌다. 값에 대한 메서드
호출(패키지 한정이 아닌 것)은 잡히지 않는다 — 파이프라인 배선은 대부분 패키지 한정 함수 호출이라
목적에 충분하다.

## 다시 만들기

저장소 루트에서:

```bash
go run docs/dev/codegraph/main.go .
```

`codegraph.json` 을 현재 디렉터리에 쓴다(루트에서 돌리면 저장소에 남으니, 갱신 후 이 폴더로
옮기고 커밋한다). `//go:build ignore` 태그가 붙어 `go build ./...` 는 이 도구를 컴파일하지 않는다.
