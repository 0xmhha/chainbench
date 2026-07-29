# tests/env — 테스트 환경 프로파일 레이어

로컬/폐쇄망 두 환경을 하나의 설정으로 다루기 위한 런타임 프로파일 레이어와,
공개 테스트용 `.env` 픽스처 + 비밀 취급 규약을 담는다.

> **참고:** 이 레이어를 소비하던 bash 회귀 스위트(`tests/regression`·`tests/lib`·
> `tests/basic`·`tests/fault`·`tests/stress`·`tests/remote`)는 Go testkit(`tests/*`)와
> 라이브 검증(`tests/repro`)으로 대체되어 폐기되었다. 남은 `.env`/`secret.example`은
> **공개 테스트 픽스처 + 비밀 경계 예시**로 유지되며, 시크릿 스캐너(`scripts/check-secrets.sh`)가
> `tests/env/*.env`를 공개 허용목록으로 참조한다.

## 파일
- `profile.sh` — `CHAINBENCH_TEST_ENV`에 맞는 `.env`를 로드(common.sh가 source).
- `local.env` — 로컬: 4노드, 정수 타깃, 공개 Hardhat 키, client_cast 서명.
- `closednet.env` — 폐쇄망: 7노드, `@stablenet-bpN` 별칭, **키 없음(노드측 서명)**, **IP 없음**.
- `secret/` — gitignore. 실제 비밀만 여기. 어떤 파일도 커밋되지 않음.
- `secret.example/` — 비밀 파일들의 형식 예시(실제 값 없음).

## 비밀 규칙 (필독)
- `secret/` 안의 어떤 파일도 `cat`/`echo`/로그/전송 금지. 비밀은 서명·노드제어 백엔드만 내부에서 읽는다.
- 폐쇄망 private key는 **이 repo에 두지 않는다**(서버 keystore에만). 노드 IP/URL은 `secret/closednet.remotes`에만.
- 비밀값이 커밋·로그·출력에 리터럴로 등장하면 즉시 중단하고 회수(rotate).

## 폐쇄망 셋업 (최초 1회)
```bash
cp tests/env/secret.example/closednet.ssh.example     tests/env/secret/closednet.ssh
cp tests/env/secret.example/closednet.remotes.example tests/env/secret/closednet.remotes
# 각 파일에 실제 값 채우고 chmod 600
```
