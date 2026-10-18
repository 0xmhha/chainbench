# 종료된 증명의 바이너리 공유 복사

`scripts/proof_native_cow.py`는 종료된 소유 런타임에 남은 동일 바이너리 복사본을
macOS APFS 공유 복사로 바꾼다. 내용은 유지하고 각 파일은 독립 inode를 갖는다.
하드링크나 증명 삭제는 사용하지 않는다. 기본 실행은 계획만 출력한다.

현재 검증한 범위는 수동 CLI와 CI 계약이다. capture/recheck 사이의 자동 실행은 아직
연결하지 않았고, 전체 스토리지 증가 문제와 과거 자료 부재 항목은 계속 열려 있다.

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
