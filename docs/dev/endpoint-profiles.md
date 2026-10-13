# 외부 노드 엔드포인트 프로필

`chain-preset.attach.endpoints`는 이미 실행 중인 노드의 RPC·WebSocket·metrics 주소와
노드 선택에 사용할 역할을 함께 선언한다. 기존 `attach.rpc` URL 배열도 지원하지만 두 입력을
동시에 사용할 수 없다. 역할을 생략하면 `en`이며, 입력 순서가 `node1`, `node2`의 순서다.
역할은 사용자가 선언한 정보이고 프로세스 소유권의 증거가 아니다. 이 attach 경로는 노드 제어를 제공하지 않는다.

```json
{
  "schemaVersion": "2",
  "kind": "chain-preset",
  "id": "external-stablenet",
  "chain": "stablenet",
  "attach": {
    "endpoints": [
      {
        "rpc": "${EXTERNAL_RPC_URL}",
        "role": "en",
        "ws": "${EXTERNAL_WS_URL}",
        "metrics": "${EXTERNAL_METRICS_URL}",
        "auth": {"type": "jwt", "env": "EXTERNAL_RPC_TOKEN"},
        "apis": ["eth", "net", "istanbul"]
      }
    ]
  }
}
```

케이스의 `chainPreset`에 이 프리셋을 참조하거나 객체로 넣은 뒤 `chainbench run case.json`으로
실행한다. 공개 MCP `chainbench_run`도 같은 케이스 JSON을 `spec`에 받는다. 주소의 `${NAME}`과
`${NAME:-default}`는 두 표면에서 같은 규칙으로 확장한다. 생략한 부가 전송은 생략한 상태로 남으며
WebSocket·metrics 검사는 해당 주소가 필요하다. 스키마의 enum과 허용 필드가 UI 입력 계약이다.

인증은 `jwt` 또는 `api-key`와 환경 변수 이름을 받는다. `api-key`는 선택한 `header` 이름을 사용하고
생략하면 `Authorization`이다. `jwt`는 `Authorization: Bearer …`를 사용하므로 `header`를 받지 않는다.
토큰·비밀번호 원문과 URL의 userinfo는 프로필에 넣을 수 없다. 실제 토큰은 전송에서만 해석하며
다른 노드에 전달하거나 다른 호스트로 redirect하지 않는다. 환경 변수가 비어 있으면 세션 생성 전에 거절한다.

`apis`는 운영자가 선언한 열린 namespace 목록이다. 실제 RPC 지원 확인을 대신하지 않으며 capability를
자동 부여하지 않는다. 현재 인증 연결의 실제 증명은 RPC·WS·metrics **읽기**다. SDK 로컬 서명의 인증
전송과 실제 capability probing은 worklist 9번 R4·R3에서 별도로 추적한다.

검증 범위와 자료는 [endpoint-profile-evidence.json](endpoint-profile-evidence.json), 전체 잔여와
최종 인수 상태는 [chainbench-worklist.md](chainbench-worklist.md) §0a에 있다.
