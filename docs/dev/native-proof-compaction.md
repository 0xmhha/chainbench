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

이는 새 스냅숏의 데이터 복사량을 줄이는 경로에 자동 적용된다. 업로드된 native asset을
증명 사이에 자동으로 정리하는 연결과 과거 자료 대조는 별도로 남아 있다.
