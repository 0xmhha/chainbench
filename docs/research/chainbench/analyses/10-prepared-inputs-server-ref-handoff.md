# 준비된 서버 입력 재사용과 서버별 파일 참조 인계

작성일: 2026-09-09  
확인 기준: `1b16ced` (`docs(analysis): log S4 config and S5 generate-dir progress`)  
상태: 설계 제안 및 구현 인계. 런타임 코드는 변경하지 않았다.

## 1. 추가 목적

기존 서버에서 중요한 회귀 테스트를 반복한다. 바이너리가 수정될 때마다 약속된 genesis,
config와 키로 검증하여 이전 동작에 영향을 주는지 확인한다.
한편 agentic coding의 harness로 사용할 때는 로컬에서 작은 체인을 빠르게 구성할 수 있어야 한다.
두 경우 모두 설정은 간단하고 DSL은 재사용할 수 있어야 한다.

이 문서는 [09 workspace-config 인계](09-workspace-config-refactoring-handoff.md)를 보완한다.
함께 사용할 샘플은 [workspace-config.sample.yaml](../../../../workspace-config.sample.yaml)이다.
새 파일 하나를 더 운영하게 만들지 않고 같은 workspace-config에 입력 준비 방식과 비공개 프리셋을 추가한다.
이 문서의 새 필드와 옵션은 아직 구현된 기능이 아니다. 기존 `srv://`의 지원 범위와 구분한다.

## 2. 실행 위치, 입력 준비, 체인 재사용을 구분한다

| 선택 | 제안 값 | 책임 |
|---|---|---|
| 실행 위치 | local / Docker / remote | 기존 server-set 선택과 resource 접근 계층 |
| 입력 준비 | `prepared` / `generated` | workspace-config의 `inputs.mode` |
| 체인 사용 | `fresh` / `reuse-if-matching` / `attach` | `execution.chain`과 testengine/chainsetup |

준비된 파일 사용이 기존 DB나 실행 중인 노드의 재사용을 의미하지 않는다.
같은 genesis와 키로 매번 새 datadir에 체인을 만들 수 있다.
원격에서 generated를 사용할 수 있고 로컬에서 prepared를 사용할 수도 있다.
Docker는 원격 서버를 대신하는 접근 방식이다. 별도의 DSL을 요구하지 않는다.

권장 조합:

- 서버 회귀 테스트: `prepared + fresh`. 입력을 고정하고 이전 DB 상태의 영향을 줄인다.
- 에이전트의 작은 로컬 테스트: `generated + fresh`.
- 비용이 큰 환경을 이어 사용: `reuse-if-matching`. 실제 배치·입력·건강 상태까지 검사한다.
- 이미 실행 중인 체인만 테스트: `attach`. 생성·init·재배포 없이 기존 상태를 검증한다.

`fresh`는 새 구성 ID와 자원 할당을 의미한다. 기존 중요 테스트의 프로세스를 종료하거나 datadir를 지우는 명령이 아니다.
기존 테스트가 자원을 사용 중이면 다른 슬롯을 할당하거나 부족함을 보고한다.
shared genesis/config/keyring을 여러 구성이 읽을 수 있어도 변경되는 DB와 포트는 공유하지 않는다.
지속 실행의 트리거는 CI나 상위 에이전트가 소유한다. 이번 설계 자체가 자동 스케줄이나 서버 작업을 실행하지는 않는다.

## 3. 기존 srv:// 지원과 서버 번호의 의미

확인한 코드:

- `internal/resource/machine.go`, `Parse`, 약 96행: `srv://<name>/<path>`를 서버 이름과 절대경로로 파싱한다.
- `internal/resource/opener.go`, `Opener.OpenPath`, 약 73행: Parse 후 server-set 및 Docker 변환을 사용해 Access를 연다.
- `internal/resource/serverset.go`, `SetLookup`, 약 745행: 선택한 server-set에서 이름으로 서버를 찾는다.
- 같은 파일 `Set.expand`, 약 329행: 현재 서버 Index는 `pool.hosts`의 순서로부터 `i + 1`로 만들어진다.
- 같은 파일 `HostSpec`: v2 YAML의 host 항목은 `name`, `addr`를 가진다. 독립적인 고정 index 입력은 없다.

따라서 `srv://server-01/data/genesis/base.json`은 server-set에 이름이 `server-01`인 서버의
`/data/genesis/base.json`을 뜻한다. dataRoot나 paths.genesis를 다시 앞에 붙이지 않는다.
현재 이 문법이 resource/keyring 경로에서 지원된다는 사실이 DSL의 모든 파일 입력에서 지원된다는 뜻은 아니다.
각 소비자에서 문자열을 그대로 `os.ReadFile` 또는 노드 실행 인자로 넘기지 않는지 확인해야 한다.

### 서버 식별자 제안

서버에는 사람이 알아보기 쉬운 고정 이름 `server-01`, `server-02` 등을 부여한다.
숫자는 이름의 일부다. `server-01`이 항상 server-set의 첫 번째 항목일 필요는 없다.
목록의 순서를 바꿔도 같은 이름은 같은 서버를 가리켜야 한다.

번호 선택이 필요한 경우에는 아래처럼 별도 구조화 필드를 제안한다.

```yaml
genesis:
  serverIndex: 1
  ref: genesis-wemix-regression.json
```

`serverIndex`는 선택한 server-set의 현재 1-based 순서다. 0, 음수, 범위 밖 값은 오류다.
처음 해석할 때 고정 서버 이름으로 바꾸고 server-set 식별 정보와 함께 상태에 저장한다.
resume 때 목록을 다시 세어 다른 서버로 연결하지 않는다. 이름과 대상이 바뀌면 변경을 보고한다.
`server`와 `serverIndex`를 동시에 입력하면 오류로 한다.

`srv://1/...`을 숫자 index로 새롭게 해석하지 않는다. 기존 parser는 `1`을 서버 이름으로 취급한다.
`srv://#1/...`도 제안하지 않는다. `#`는 URI fragment와 충돌한다.
고정 ID 필드를 server-set에 새로 도입하는 일은 별도 형식 변경이다. 현재 번호가 영구 ID라고 설명하면 안 된다.

## 4. 파일 참조 규칙

입력 파일을 보관한 서버와 노드를 실행할 서버는 별개의 정보다.
전자를 바꿔도 후자의 배치가 자동으로 바뀌지 않는다.

| 참조 형태 | 의미 | 상태 |
|---|---|---|
| `genesis-wemix-regression.json` | 노드가 실행될 대상의 dataRoot/paths.genesis 아래 | 09에서 제안한 용도별 참조 |
| `srv://server-01/data/genesis/base.json` | 특정 서버의 명시적 절대경로 | resource의 기존 문법. 소비자 연결 필요 |
| `{server: server-01, ref: base.json}` | 지정 서버의 환경 규칙에서 용도별 절대경로 계산 | 신규 제안 |
| `{serverIndex: 1, ref: base.json}` | 번호를 서버 이름으로 확정 후 위와 동일 처리 | 신규 제안 |
| `{localPath: /private/chainbench/fixtures/base.json}` | chainbench 실행 머신의 파일 | 신규 제안 |

bare 문자열은 새로운 계약에서는 용도별 상대경로 또는 `srv://` URI로 제한한다.
로컬 절대경로를 명시하려면 `localPath`를 쓴다. 상대 localPath는 비공개 workspace-config 파일의 위치를 기준으로 한다.
구형 CLI/API의 bare 절대경로 동작까지 일괄 변경하지 않는다.
URI 또는 객체로 지정한 절대경로는 신뢰된 비공개 설정에서만 허용하는 명시적 위치 지정이다.
09의 상대경로 제한은 DSL의 이식 가능한 실행 자원 참조 및 paths 디렉터리에 계속 적용한다.

구조화 참조는 정확히 한 형태만 허용한다. `localPath`와 `server/ref`를 섞으면 오류다.
알 수 없는 서버, 없는 명시적 server-set, 지원하지 않는 URI 구성은 로컬 접근으로 대체하지 않는다.
서버 참조에 비밀번호, 주소, SSH 포트를 직접 싣지 않는다. 인증 정보는 server-set의 기존 정책을 따른다.
query, fragment, userinfo 및 인코딩의 허용 범위를 명시해 검증하며, 경로를 두 번 decode하지 않는다.

`srv://server-01/genesis/base.json`은 기존 계약상 `/genesis/base.json`이다.
이를 몰래 `/data/genesis/base.json`으로 바꾸면 안 된다.
root가 바뀌어도 참조를 유지하려면 `{server: server-01, ref: base.json}` 형태를 쓴다.
초기 버전의 workspace 경로 규칙은 공통이다. 서버별 루트 재정의가 필요하면 09의 후속 설계를 함께 확정한다.

## 5. prepared 프리셋 제안

다음은 workspace-config에 들어갈 신규 구조다. 기존 `dataRoot`, `paths`, `control`과 함께 사용한다.

```yaml
inputs:
  mode: prepared
  preset: regression

presets:
  regression:
    genesis: srv://server-01/data/genesis/genesis-wemix-regression.json
    keyring: srv://server-01/data/keys/regression-keys
    configs:
      default: srv://server-01/data/configs/config-wemix-regression.toml

execution:
  chain: fresh
```

위 예시는 모든 원본이 server-01에 있다는 뜻이다. 모든 노드를 server-01에 배치한다는 뜻은 아니다.
root 조합 방식을 사용하려면 개별 값을 다음처럼 바꾼다.

```yaml
genesis:
  server: server-01
  ref: genesis-wemix-regression.json
```

DSL은 논리 config 이름을 선택하고 비공개 `configs` 맵에서 파일로 연결한다.
예: DSL의 `sync-test`를 `configs.sync-test`에 연결한다.
`default`는 DSL이 이름을 지정하지 않았을 때만 사용한다. 지정한 이름이 없으면 default로 대체하지 않는다.
역할별·노드별 config 선택은 기존 DSL의 노드 정의가 소유한다. 환경 파일에 역할 구성과 우선순위를 중복 구현하지 않는다.
genesis와 keyring은 같은 프리셋으로 묶어 약속된 신원 조합을 검증한다.
DSL에 이미 명시한 genesis·key source와 프리셋이 충돌하면 조용히 덮어쓰지 않고 오류로 보고한다.

prepared의 규칙:

- 파일이 없거나 읽을 수 없으면 중단한다. generated로 자동 전환하지 않는다.
- 원본을 수정하지 않는다. 필요한 실행용 복사본과 원본의 대응을 기록한다.
- genesis의 초기 합의 참여자와 사용할 키의 공개 신원을 체인별 계약에 따라 검증한다.
- 완성 config의 내부 경로·포트·파일 형식이 배치와 일치해야 한다.
- config 원본 그대로 사용과 템플릿 렌더링은 09의 별도 의미를 유지한다.
- 실행할 바이너리는 별도 입력이다. prepared가 이전 바이너리까지 고정한다는 뜻은 아니다.

## 6. generated와 간단한 실행

```yaml
inputs:
  mode: generated

execution:
  chain: fresh
```

generated는 테스트용 키 및 genesis/config를 기존 Builder로 준비하는 방식이다.
체인 바이너리를 소스에서 자동 빌드하거나 다운로드한다는 뜻은 아니다.
노드 수와 역할은 DSL 또는 기존 구성 입력이 소유한다.
새 키와 실행용 파일은 구성별 전용 영역에 두고 공유 프리셋을 덮어쓰지 않는다.
`inputs.preset`은 prepared에서 필수이며 generated와 함께 지정하면 오류로 한다.

CLI는 두 경우 모두 기존에 제안한 `--server-set`과 `--workspace-config` 조합을 유지한다.
새로운 모드별 명령을 여러 개 만들지 않는다. 향후 `--input-mode` 같은 override를 추가한다면
해석된 최종 설정을 출력하고 DSL/프리셋과 충돌할 때 거부한다.
`attach`에서는 inputs.mode가 파일 생성으로 이어지지 않는다. 준비된 기준을 검사에 사용할 수는 있다.
기존 `--attach`와 `execution.chain`을 함께 입력하면 같은 값만 허용하거나 한쪽만 지정하게 한다.

## 7. 원본 사용과 전송의 경계

`srv://`는 파일 위치를 지정한다. 자동 다운로드, 모든 서버로 복사, 노드 원격 실행까지 허용하는 표시가 아니다.

- 원본과 실행 대상이 같은 서버면 기존 파일을 검증하고 읽거나, 명시된 구성의 실행용 위치에 복사한다.
- 원본과 실행 대상이 다르면 배포 계획에 source와 destination을 각각 표시한다.
- 공개 genesis/config 전송도 기존 배포 정책과 소유권 범위 안에서 수행하며 복사 후 해시를 확인한다.
- keyring/keystore의 개인 키는 참조만으로 로컬에 내려받거나 다른 서버에 복제하지 않는다.
- 노드마다 이미 있는 키를 참조할 수 있어야 한다. 중앙 keyring의 복제가 필요하면 별도로 허용된 키 배포 경로를 사용한다.
- 준비된 키의 검증은 가능한 한 대상에서 수행하고 공개 신원만 반환한다.
- 로컬 생성 테스트 키는 승인된 구성의 대상에 배포할 수 있지만 원격 운영 키를 가져오는 동작과 구분한다.

현재 keyring이 원격 파일을 읽을 수 있다는 사실만으로 개인 키가 로컬 메모리·임시 파일에 남지 않는다고 단정하지 않는다.
원격 keyring 읽기, 검증, 전송의 실제 코드 경로를 감사하고 prepared의 보안 계약을 충족하는지 확인한다.
테스트 트랜잭션을 서명하는 계정과 노드 합의 키도 구분한다. 서명 방식과 권한은 별도 선택이다.

## 8. 지속 회귀 테스트의 기준과 증거

약속된 입력은 파일명뿐 아니라 내용으로 검증한다.
첫 승인 시 genesis/config 해시와 공개 신원을 내부 기준 기록으로 남기고, 이후 변경을 감지한다.
기준 갱신은 명시적으로 수행한다. 변경된 파일을 읽고 기준도 자동 갱신하면 회귀 테스트 조건을 고정할 수 없다.
키 원문이나 비밀번호의 해시를 공개 report에 남기는 방식으로 대체하지 않는다.

각 실행에는 다음을 기록한다.

- 테스트 대상 바이너리의 해시, 확인 가능한 빌드·커밋 정보. 파일명으로 커밋을 추정하지 않는다.
- 원본 및 실행용 genesis/config 해시와 config revision.
- 사용한 키의 공개 식별자와 검증 결과.
- 선택한 server-set의 서버 이름, 당시 번호, 내부 설정 변경 감지 정보, 원본·목적지 경로.
- 구성 ID, 노드별 실행 명령과 PID 이력, 테스트 결과와 실패 로그.

기존 건강한 환경을 재사용하더라도 바이너리 변경을 놓치지 않아야 한다.
`reuse-if-matching`은 변경된 바이너리를 요구하는 테스트를 이전 프로세스에 연결하지 않는다.
원본 검사 이후 실행 전 변경도 검사하거나 검증한 실행용 복사본을 사용한다.
결과 구조와 최종 report 위치는 09 및 기존 session 규칙을 유지한다.

## 9. 민감정보와 저장소 경계

실제 server-set과 workspace-config는 저장소 밖의 비공개 디렉터리에 둔다.
저장소 안에서 사용하는 경우 앞서 추가한 `.gitignore` 이름 규칙을 따른다.
DSL에는 논리 이름만 두며 실제 서버명·경로는 프리셋에 연결한다.
공개 샘플의 `server-01`, `/data`는 예시다. 실제 환경에서 추출한 값이 아니다.

URI에는 인증 정보가 없더라도 내부 서버명과 경로가 있으므로 비공개 환경 정보로 취급한다.
원본 server-set이나 workspace-config 전체를 report에 복사하지 않는다.
private key와 password를 로그·dry-run·오류·report에 출력하지 않는다.
구성별 키 보관이 필요한 경우 keyring 규칙을 따르는 접근 제한 영역으로 분리한다.

## 10. 구현 순서와 인수 조건

09의 W1–W6에 아래 항목을 합친다. 이미 진행 중인 코드 변경과 중복 구현하지 않는다.

| 단계 | 추가 작업 | 실패 위험과 검증 |
|---|---|---|
| W1 | prepared/generated, 프리셋 참조, URI·객체·상대경로 검증 | 모호한 형태·없는 프리셋·잘못된 번호를 거부 |
| W2 | server-set 이름/번호 해석, 원본 머신 보존 | 목록 순서 변경, 동일 경로의 로컬/원격 서로 다른 파일로 검증 |
| W3 | source와 destination 분리, 확정 서버 이름 저장 | 원본 서버가 노드 배치를 바꾸지 않으며 resume이 다른 서버로 이동하지 않음 |
| W4 | prepared 소비 및 generated 기존 Builder 연결 | 누락 입력 자동 생성 금지, 고정 genesis·키 일치, 원본 보존 |
| W5 | 단계형 CLI·DSL·MCP 경로 통일 | 어떤 표면도 srv URI를 로컬 파일명이나 노드 argv로 그대로 넘기지 않음 |
| W6 | fresh/reuse/attach와 기준 입력 검사 | 진행 중인 중요 테스트 보존, 바이너리 변경 감지, 실패 로그와 결과 수집 |

추가 인수 조건:

- [ ] `srv://server-01/data/genesis/base.json`은 정확히 그 서버의 `/data/genesis/base.json`을 읽는다.
- [ ] 같은 위치의 로컬 파일이 존재해도 원격 입력 대신 사용하지 않는다.
- [ ] `srv://server-01/genesis/base.json`에 dataRoot를 암묵적으로 추가하지 않는다.
- [ ] `{server: server-01, ref: base.json}`은 workspace의 용도별 경로로 해석한다.
- [ ] `serverIndex: 1`은 한번 확정한 이름을 보존하며 목록 순서 변경으로 다른 서버를 사용하지 않는다.
- [ ] 숫자로만 된 서버 이름과 index를 구분한다.
- [ ] 원본 서버와 실행 서버가 다른 경우 전송 계획과 키 취급 정책을 검사한다.
- [ ] 같은 DSL을 로컬 prepared 및 원격 prepared에서 환경 파일만 바꿔 사용한다.
- [ ] generated로 작은 로컬 체인을 구성하고 준비된 운영 키에 접근하지 않는다.
- [ ] prepared의 입력 누락·불일치는 실행 전 실패하며 자동 생성되지 않는다.
- [ ] fresh 실행이 기존 테스트의 프로세스·DB·공유 입력을 변경하지 않는다.
- [ ] attach는 생성·배포·init을 수행하지 않는다.
- [ ] 키 내용과 인증 정보가 dry-run, 오류, report, 임시 파일에 유출되지 않는지 검사한다.

새 문법은 단계적으로 추가하되 기존 srv 절대경로 의미를 보존한다.
문제가 생기면 새 입력 모드를 비활성화하고 기존 구성의 저장된 경로를 유지한다. 자동 DB 이동이나 공유 원본 삭제로 되돌리지 않는다.

## 11. 메인 세션 전달문

> 09 workspace-config 인계에 더해 이 문서와 갱신된 workspace-config.sample.yaml을 검토해 주세요.
> 기존 서버에서 준비한 genesis/config/keyring으로 반복 회귀 테스트를 하고, 로컬에서는 generated로 작은 체인을 구성해야 합니다.
> 실행 위치, 입력 준비 방식, 체인 재사용을 분리하고, 실제 경로와 서버 참조는 비공개 프리셋에서 연결해 주세요.
> 기존 srv://<name>/<absolute-path> 의미를 유지하고 번호 선택은 별도 필드로 해석해 확정 이름을 기록해 주세요.
> 참조된 파일 서버와 노드 실행 서버를 분리하며 개인 키의 자동 다운로드·복제를 허용하지 마세요.
> 현재 코드와 사용자가 이후 확정한 규칙을 먼저 비교한 뒤 승인된 구현 범위에서 작업해 주세요.
> parser 성공뿐 아니라 대상 파일 소비, 실제 argv, 기존 중요 테스트 보존까지 검증 근거를 남겨 주세요.
