# 리팩토링 전 동작 기준선

수집 시각: 2026-09-13 19:18–19:22 KST. 사용자 요청으로 이전의 기준선 수집 보류를 해제하고 실행했다. 제품 코드와 참조 체인 소스는 수정하지 않았다. 이번 결과는 현재 바이너리와 테스트로 확보한 기능 기준선이며, 전체 호환성 인증이나 성능 기준선은 아니다.

## 결과

| 대상 | 결과 | 관찰 및 증거 |
|---|---|---|
| 전체 기본 테스트 | PASS | `go test -count=1 -json -timeout=8m ./...`, 58 패키지 통과. 테스트·하위 테스트 이벤트 1,944 PASS, 27 SKIP, 0 FAIL. 테스트 없는 패키지 12개는 별도 SKIP. `unit.log`, `test-summary.json` |
| CLI 빌드 | PASS | 현재 소스로 빌드. `build-cli.log` |
| CLI help/chains/capabilities | PASS | 출력 원문을 `cli-*.log`로 고정 |
| WBFT 기본 E2E | PASS | 실제 4 validator + 1 endpoint, 블록 증가·송금 receipt/수신 잔액·컨트랙트 배포/42 반환 검증. `e2e-wbft.log` |
| Stablenet 기본 E2E | FAIL | 블록 증가와 송금 receipt 성공 검사를 지난 뒤 수신 잔액 `0 -> 0`으로 실패. 이후 컨트랙트 단계 미도달. `e2e-stablenet.log` |
| Wemix → WBFT handoff | PASS | head 30, 블록 21–30을 후속 validator 4/4 집합이 생성. 전환 후 잔액 보존·송금·컨트랙트 검증 성공. `e2e-wemix-handoff.log` |

Stablenet 실패는 현재 관찰된 기준선 결함이다. 원인은 확정하지 않았다. 실패를 수정하거나 성공으로 치환하지 않았고 재시도로 제거하지 않았다. 보존된 작업 경로는 `/tmp/cbe2e3643186464`, 영속적인 노드 로그 사본은 `stablenet-failure-logs/`다. 테스트 하네스는 실패 작업 디렉터리를 보존하고 노드를 정리했다. 수집 후 체인 바이너리/이번 임시 경로에 해당하는 잔여 프로세스 검색 결과는 0건이었다.

## 입력 식별

| 저장소 | commit | 바이너리 버전 |
|---|---|---|
| chainbench | `7f39c5627e0bdaccaa8e207a67767d71579283f5` | 현재 소스에서 CLI 빌드 |
| go-wemix | `1350376a680489298b3203a44d9ebddb7747f9cc` | 0.10.14-stable |
| go-wbft | `7af50e45db4a48a2a77e565c7ff7bde0e8f95505` | 1.0.0-alpha |
| go-stablenet | `0937ac5c93d4f56ded5b24899f381d3d1f208c02` | 1.1.0-stable |

`source-snapshots.json`은 네 저장소의 commit, 수집 시작 상태, 추적 파일별 SHA-256 및 합성 해시를 담는다. chainbench에는 준비 문서/수집 산출물 등의 untracked 파일이 있었고, Stablenet에는 `.claude/`, `CLAUDE.md`가 untracked 상태였다. 추적 파일의 변경은 없었다. untracked 파일의 내용은 이 소스 해시 범위에 포함되지 않는다.

`binaries.json`은 실제 실행한 바이너리 경로와 SHA-256을 담는다. `*-version.log` 및 `*-build-info.log`는 바이너리 자체의 버전·빌드 정보를 보존한다. 체인 바이너리는 이번에 재빌드하지 않았다. 따라서 바이너리 기준선은 그 해시를 기준으로 비교한다.

## 재실행과 판정

`commands.json`에 명령 인자, 작업 디렉터리, 환경 설정, 시작 시각, 제한 시간, 종료 코드, 실행 시간이 기록돼 있다. `collect.py`는 실제 수집에 사용한 실행기다. 실행할 때마다 새 UTC 디렉터리를 만든다. 절대 경로는 이 환경에 맞춰져 있다.

```sh
python3 docs/dev/architecture/baselines/20260913T101856Z/collect.py
```

Go 도구체인은 캐시에 설치된 Go 1.25.13의 `GOROOT/bin`을 PATH 앞에 두고 `GOTOOLCHAIN=local`로 고정했다. `GOPROXY=off`, `GOSUMDB=off`로 실행했으므로 이미 설치된 모듈이 필요하다. 빌드 캐시와 테스트 임시 파일은 `/tmp` 아래에 둔다. localhost TCP/IPC 및 자식 프로세스를 허용하는 환경이 필요하다. 전체 기본 테스트 제한은 패키지당 8분, E2E는 각 6분이며 외부 실행기에도 별도 제한과 프로세스 그룹 종료 처리가 있다.

성공은 종료 코드와 테스트 이벤트를 함께 읽어 판정한다. SKIP은 검증 성공으로 세지 않는다. 재현 시 값이 변하는 시간·PID·임시 경로·포트·트랜잭션 해시는 의미를 구분하여 비교한다. Stablenet의 현재 FAIL은 원인 규명 전까지 알려진 실패로 유지하며, 리팩토링으로 새로 생긴 회귀와 구별한다.

## 범위와 한계

- CLI/DSL/MCP/설정/결과/세션 관련 기존 기본 테스트를 실행했다. 외부 계약별로 모든 동작을 망라한 golden fixture는 아직 아니다.
- 원격 SSH/sudo와 별도 라이브 설정이 필요한 테스트 등 27개 SKIP의 이름은 `test-summary.json`에 있다.
- 세 체인의 대표 경로를 선택했다. Stablenet hardfork/합의 장애/동기화/거버넌스 전체, WBFT 장애/동기화 전체, Wemix 독립 체인 전체 및 기타 tagged E2E는 이번에 실행하지 않았다.
- race, 벤치마크, 부하/성능 수치, 장시간 안정성 검사는 수행하지 않았다. 기록된 실행 시간은 성능 합격 기준이 아니다.
- 최초 `20260913T101706Z`, `20260913T101731Z` 시도는 실행기 도구체인 설정 오류다. `20260913T101809Z`는 샌드박스 포트 제한에 걸렸다. 이들은 환경 실패 기록이며 제품 실패 집계에서 제외했다. 현재 디렉터리가 권한 제한 해소 후 유효한 기준선이다.

`artifact-sha256.json`은 보고서 작성 완료 시점의 파일 무결성 목록이다.
