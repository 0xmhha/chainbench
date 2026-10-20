# 종료된 증명의 바이너리 공유 복사

`scripts/proof_native_cow.py`는 종료된 소유 런타임에 남은 동일 바이너리 복사본을
macOS APFS 공유 복사로 바꾼다. 내용은 유지하고 각 파일은 독립 inode를 갖는다.
하드링크나 증명 삭제는 사용하지 않는다. 기본 실행은 계획만 출력한다.

수동 CLI와 `tests/webui/verify.sh`의 capture/recheck 종료 후 자동 실행을 지원한다.
전체 스토리지 증가 문제와 과거 자료 부재 항목은 계속 열려 있다.

## 실행

현재 실행에서 출처와 실제 SHA를 확인한 세 체인의 선택 manifest를 사용한다.
`WEBUI_SELECTED_BINARIES`는 그 manifest의 절대 경로이고,
`WEBUI_REPLACEMENT_BINARY`는 별도 검증한 교체 바이너리의 절대 경로다.
실제 증명이나 독립 재현이 종료된 뒤 하나씩 실행한다.

```sh
python3 -B scripts/proof_native_cow.py \
  --root /private/tmp/cbui \
  --catalog "$WEBUI_SELECTED_BINARIES" \
  --additional-binary "$WEBUI_REPLACEMENT_BINARY"
```

적용은 같은 인자에 `--apply --audit /private/tmp/<새이름>.jsonl`을 추가한다.
감사 파일은 새 파일이어야 하며 기존 파일을 지정하면 변경 전에 거절한다.
추가 바이너리가 없으면 `--additional-binary`를 생략한다.

## 적용 조건과 보존

- manifest의 세 family와 원본 바이너리 SHA·실행 권한을 대조한다.
- 소유 marker와 쌍을 이룬 회수 시작/완료 기록, 회수 대상 부재를 확인한다.
- 루트의 정해진 바이너리 이름, `d/fork/attach/binaries/<SHA>/`의 바이너리,
  `store/assets/<ID>/payload`만 후보로 삼는다. 일반 사용자 파일은 대상이 아니다.
- 기록된 `d/fork/attach/binaries/<SHA>/`에서는 `gstable/gwbft/gwemix`와 기존 `stablenet-node/wbft-node/wemix-node` 이름을 같은 조건으로 다룬다. 카탈로그 밖의 해시와 다른 파일 이름은 거절한다.
- 원본과 대상의 내용·mtime 외 메타데이터가 같아야 한다. 대상 mtime을 별도로 보존한다.
- 전용 할당량 20MiB 이상인 중복 대상에만 적용한다. 이미 공유된 대상은 다시 복사하지 않는다.
- 새 프로세스/열린 파일 조사와 파일·종료 기록 재대조를 교체 직전에 수행한다.
- 내용·모드·소유자·그룹·mtime·확장 속성·ACL과 대상 상위 폴더 mtime을 보존한다.
  inode와 ctime은 바뀐다. atime과 상위 폴더 ctime 보존은 주장하지 않는다.
- 복사 명령은 60초로 제한하고 실패한 임시 파일을 제거한다. 원본과 대상은 유지한다.
- 감사 기록을 먼저 동기화하고 교체 완료를 별도로 기록한다.

출력의 `skipped`에는 소유권·종료 기록·경로·카탈로그 조건을 충족하지 않은 항목이 나온다.
이를 정리 완료로 세지 않는다. `privateBytesBefore/After`는 해당 파일의 전용 할당량이며
볼륨 전체 여유 변화와 다르다. 검증을 위해 새로 만든 복사본의 절감량도 기존 증명 공간
회수와 구분한다. 실제 시험의 20GiB 사전 검사와 과거 자료 보존 정책은 별도로 유지한다.

## 확인한 범위

기본 CI의 `make check`가 복사 계약을 실행한다. 실제 APFS 검증과 독립 대조는
[측정 기록](all-code-graph-native-cow-evidence.json)에 연결한다. 전체 체인 실행이나 최종
Web 인수를 대신하지 않는다.

## 새 증명 소스의 복사

`runtime_storage.clone_file`을 사용하는 소스 스테이징·Git view·빌드 준비물 복사는 이제
파일 크기와 관계없이 APFS 공유 복사를 시도한다. `clonefile`의 ACL 복사와 소유 입력의
UID/GID 복원을 사용하며, 지원하지 않는 파일시스템은 일반 복사를 유지한다.
모든 경로는 임시 파일을 완성한 뒤 교체한다. 원본 별칭·연결된 대상과 복사 중 달라진
대상은 거절하고, 실패 시 기존 대상 파일과 원본을 보존한다.

실측에서는 소스 2,543개 파일의 논리 데이터 26,756,376바이트를 복사한 뒤 전용 데이터
할당이 0임을 확인했다. 파일/폴더의 inode·디렉터리 메타데이터는 여전히 공간을 사용한다.
복사한 의존성 디렉터리도 회수했다. 회수 기록의 `allocatedBytes`는 제거한 파일이 참조하던
블록 수이며, 공유 데이터가 있으므로 실제 볼륨 여유 증가와 동일하게 세지 않는다.
그룹이 달라 실패한 첫 시도와 수정 후 실제·독립 대조를 모두
[측정 기록](all-code-graph-source-copy-evidence.json)에 보존한다.

이는 새 스냅숏의 데이터 복사량을 줄이는 경로에 자동 적용된다. 과거 자료 대조와 다른
생성 데이터의 반복 증가 정책은 별도로 남아 있다.

## capture/recheck 종료 후 자동 실행

`verify.sh`는 증명 자식 프로세스가 종료된 뒤 성공·실패 양쪽에서
`tests/webui/after_proof_storage.py`를 호출한다. 증명이 실패하면 원래 종료 코드를 유지하고,
증명이 성공했으나 정리가 실패하면 전체 명령도 실패한다. 증명 결과 파일을 수정하지 않는다.

macOS와 명시적인 `WEBUI_SELECTED_BINARIES`가 있어야 공유 복사를 실행한다. 그 외에는
대상 0·skipped를 출력한다. 검증한 교체 바이너리는 `WEBUI_REPLACEMENT_BINARY`로 추가하며,
Linux 등 다른 출처 확인 바이너리는 절대 경로의 JSON 배열 `WEBUI_STORAGE_EXTRA_BINARIES`로
명시할 수 있다. 지정한 파일이 잘못됐으면 조용히 제외하지 않고 실패한다.

계획 단계부터 새 프로세스·열린 파일 조사를 통해 사용 중인 런타임을 제외하고, 실제 교체
직전에도 다시 확인한다. `<runtime-root>/.storage-maintenance/`는 소유한 0700 디렉터리이며
각 실행은 새 0600 JSON 보고서와 대상이 있을 때만 새 JSONL 감사를 남긴다. 정리 전후의
저장소 소스 지문과 게시 증명의 내용·모드·소유자·그룹·mtime·inode·링크 수를 비교한다.

실제 WEB-01 브라우저 실행과 별도 소스 재현을 직렬 실행해 자동 호출을 확인했다. 출처가
확인된 native 검증 복사본 한 개의 전용 할당 62,918,656바이트가 0으로 줄었고, 두 번째
호출에서는 재복사 대상이 0이었다. 실제 결과·실패한 초기 감사의 작업 경로 오류·수정한
독립 감사의 14개 변조 거절은 [측정 기록](all-code-graph-after-proof-evidence.json)에 보존한다.
이 검증 복사본의 절감량을 기존 증명 전체의 회수량이나 Web 14개 인수로 세지 않는다.

기록된 바이너리 폴더의 기존 별칭 누락은 3개 실패 대조 뒤 22개 계약으로 보강했다.
실제 종료 복사본 13개·원본 포함 14파일의 보존과 전용 할당 817,942,528바이트 감소를
독립 확인했다. 준비 검증기의 변경 전 실패와 파서 환경 복구도 [측정 기록](native-alias-compaction-evidence.json)에 보존했다.
