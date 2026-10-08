# Web UI 구현 상태

2026-10-08 기준. 승인된 요구사항과 14개 인수 조건은 변경하지 않는다.
이 문서는 구현과 검증의 진행 상태이며 전체 완료 판정이 아니다.

현재 브라우저는 대시보드, 체인·workspace, 테스트, 모니터링, 히스토리,
설정으로 이동한다. 로그인과 역할별 권한, 공유 문서 리비전, 암호화한 개인
SSH 자격증명, 매니페스트 열람·import, DSL 구조 편집을 연결했다.
문서 import는 원본을 암호화해 보관하고, actor·만료·리비전을 검사한 뒤
commit한다. 프론트 빌드는 실제 Go embed 경로에 생성한다.

실행 관리 계층은 계획의 입력과 물리 자원을 고정하고, 실행 직전에 다시
검증한다. actor와 요청 키가 같으면 같은 작업을 돌려준다. 다른 workspace라도
물리 호스트의 경로나 포트가 겹치면 실행 중인 작업과 충돌한다. 작업 상태,
단계, 부분 변경과 감사 기록은 서버에 저장한다. 브라우저 연결 종료와 로그아웃은
작업을 취소하지 않는다. 서버 재시작은 진행 중 작업을 interrupted로 기록하며
자동 재실행하지 않는다. 계정·자격증명 철회는 취소와 보존을 우선한다.
SSH 접근 검사의 메모리 자격증명에는 매 접속의 권한 검사를 연결했다. 계정·자격증명
철회, workspace·개인 binding 변경, context 취소 후에는 캐시한 값으로 새 SSH
접속을 시작할 수 없다. SSH tunnel은 기존 HTTP 연결을 재사용하는 요청도 검사한다.
승인된 실행 작업은 문서와 개인 credential ID를 고정하므로, 이후 workspace나 binding
편집이 대상을 바꾸지 않는다. 실행 작업의 매 접속은 계정·자격증명 철회와 취소를
다시 검사한다. 취소는 SSH 인증 및 실행 중 연결을 닫지만, 원격 프로세스가 종료됐다는
증거로 취급하지 않는다.

기존 엔진과 연결된 현재 실행 경로는 로컬·SSH 매니페스트 기반 구성·초기화,
노드 구동, 소유 노드 시작·정지다. 서버에 등록된 바이너리만 선택하고
체크섬·버전·dialect를 검사한다. 서버와 노드는 저장된 목록에서 선택한다.
계획은 Linux machine-id 또는 Darwin platform UUID를 해시해 같은 물리 장비를
식별하고, 경로의 기존 심볼릭 링크를 해소한다. 별칭·SSH 포트·local/SSH 경로가
달라도 같은 장비의 자원으로 판단한다. 확인할 수 없는 장비와 바이너리/대상 플랫폼
불일치는 거부한다. SSH 바이너리는 workspace의 binaries 하위 체크섬별 경로로
별도 전송한 뒤 원격 체크섬을 검증한다. 파일 전송은 SSH stdin을 사용하고 완료 후
교체하여, 키와 큰 바이너리가 원격 셸 인자에 들어가지 않는다.
attach 대상은 이 어댑터에서 실행하지 않는다. 기록된 PID는 생존 여부를
보장하지 않으므로 실시간 확인 전에는 unknown으로 표시한다.

실제 Chrome 개발 검증은 세 체인에서 각각 네 노드의 데이터베이스를 초기화하고,
바이너리의 dumpgenesis로 선언과 실제 database의 chain ID를 비교했다. 브라우저에서
검토·제출한 작업은 연결을 닫은 뒤에도 완료됐고, 중복 요청은 같은 작업을 반환했다.
소유 노드 시작·정지와 390px 화면의 가로 넘침 검사도 수행했다. 이 개발 검증은
별도 증거이며 WEB-06 등 인수 조건을 통과했다고 간주하지 않는다.

`tests/webui/run_ssh_jobs.py`는 작업 전용 localhost SSH 데몬, 실제 WBFT 바이너리,
Chrome을 사용한다. 브라우저에서 개인 binding으로 계획·제출하고, 별도 바이너리
전송, 네 노드의 실제 DB·genesis, SSH node.start/stop, local/SSH 별칭의 동일 자원
식별, 실행 전 자격증명 철회 거부 및 plaintext 키 미보관을 확인한다. 개발 증거이며
전체 원격 배포 인수 조건, 실행 중 철회·잔존 자원 대조의 완료 판정은 아니다.

저장한 DSL 케이스는 `test.run` 작업으로 선택한 리비전과 실행 내용을 고정한다.
승인 직전 리비전 변경은 거절하고, 승인 후 공유 문서 편집은 실행 내용을 바꾸지 않는다.
기존 `PlanSuite`와 `RunSuite`가 같은 선언을 해석하고 실제 노드 배치·genesis·실행
옵션·테스트를 수행한다. Web 어댑터는 생산자 수를 덮어쓰지 않으며 BP/EN/PN 전체의
포트를 계획에 포함한다. 등록된 기본 바이너리는 이름 참조까지 검증한 파일에 연결하고,
로컬/SSH 대상의 체크섬 경로에 원래 체인 바이너리 이름으로 전송한다. 소스 파일 이름을
바꿔 등록해도 IPC 초기화가 다른 소켓을 기다리지 않게 한다. 케이스 원본은 보존한다.
테스트 실패/blocked는 작업을 실패로 기록하며 실제 엔진 세션 참조와 판정을 보관한다.
기본 처리는 보존이고, 명시한 정리는 소유 네트워크에만 적용한다.

현재 테스트 작업은 기본 내장 체인, 단일 등록 바이너리, count/명시적 node table 배치, 준비된 키 자료를
연결한 범위다. 외부 테스트 매니페스트, 테스트의 이름 참조 프리셋, 파일 자료,
혼합 바이너리/upgrade, generate 키와 선언한 계정, 별도 포트/경로 override,
attach 실행의 계약은 아직 연결하지 않았다. 이를 무시하거나 다른 구성으로 실행하지
않고 거절한다. 전체 어휘/인자의 실제 실행 인수는 미완료다. `/tests` 화면에서 저장
케이스를 선택해 검토·실행하고 작업의 세션 결과 링크로 히스토리를 연다.
Web 엔진 세션은 기본 서버 데이터 디렉터리의 `sessions`에서 캡처한다. 기존 CLI
artifact root와 `web:` 참조를 분리해 동일 세션 이름의 충돌을 피한다. 실행 중 먼저
발견한 세션을 완료로 고정하지 않으며, 작업 연결이 저장되면 판정과 메타데이터를 갱신한다.

히스토리는 엔진 세션의 판정·어세션·단계·참조 자료와 Web 작업 결과를 별도 사본으로
보관한다. 검색, 체인·workspace·실행자·상태·케이스·기간 필터, 페이지 이동,
2–4개 실행의 테스트 판정 비교, 민감 값을 제거한 JSON bundle 다운로드,
관리자 삭제를 연결했다. 과거 세션의 미기록 값과 metric·로그 자료 없음은 표시한다.
비교 가능 판정에는 체인·바이너리·환경 fingerprint와 공통 테스트가 필요하다.
삭제는 하나의 원자적 snapshot 교체로 결과 사본·tombstone·감사를 함께 저장한다.
진행 중 작업은 삭제할 수 없으며 CLI 원본 세션, 공유 자료, 실제 노드 데이터와
작업·중복 요청 기록을 삭제하지 않는다. 링크·키·DB·바이너리를 수집하지 않으며,
캡처 파일당 2 MiB, 세션당 8 MiB의 제외 항목은 목록에 남긴다.

현재 구성 작업은 서버가 준비한 키 디렉터리를 제한된 크기의 사본으로 캡처해 암호화하여
보관하고, 그 SHA-256을 계획과 승인된 작업에 고정한다. 승인 전 원본 변경은 계획 충돌로
거절하고 승인 후에는 원본 대신 사본을 사용한다. 실제 엔진은 개인 권한의 사본 경로를
읽는다. 링크·특수/과대 파일·취소·암호문 및 엔진용 사본의 변조는 실행 전에 거절한다.
브라우저 계획과 작업 기록에는 키 원문을 넣지 않는다. 사용자 업로드·키 선택 화면은
아직 미구현이며 이 고정 경로만으로 전체 구성 import와 배포 인수를 통과하지 않는다.

남은 구현은 체인 프리셋 전체를 실행 계획에 적용하는 경로, 사용자 파일 업로드와
업로드 키 자료 선택, 바이너리·설정 교체, 원격 실행 중 철회/부분 실패의
전체 인수 검증과 재시작 후 잔존 자원 대조,
노드 초기화, DSL 테스트 실행 작업, 재시작 후 자원 대조, metric 수집·차트와 시점별
로그, 전체 DSL 범위의 실행 연결과 캡처 확장이다. 히스토리의 JSON 사본은
노드 로그·metric archive 전체를 아직 포함하지 않는다. 기존 문서의 전체 인수 조건을 구현하고
각 조건의 실제 실행 증거를 다시 생성해야 완료를 판정할 수 있다.

## Retained resource ownership and conflict review

Terminal jobs continue to exclude other workspaces while resources are retained,
unknown, interrupted, or cleanup has failed. The owner workspace can request
subsequent controls; active overlapping jobs still conflict. A later verified
cleanup releases only matching physical host and canonical target roots, with
persisted acceptance order establishing that cleanup follows retained work.
Incomplete cleanup or missing ordering evidence does not release ownership.

`GET /api/v1/plans/{planId}/conflicts` is restricted to the plan's authenticated
operator/administrator. It exposes job, workspace and actor identifiers and
resolved conflicting claims, never the accepted execution payload or credentials.
Foreign plan identifiers return 404; expired reviews return 409. The SPA shows
these owners, disables execution while conflicts exist and allows refresh after
cleanup. Starting a job checks exclusion atomically again; review is not a lock.
PID-free residual reconciliation and concurrent independent execution coverage
remain unfinished, so WEB-11/12 are not complete.

Job acceptance probes targets outside the durable store lock, keeping job reads,
progress, cancellation and independent target acceptance responsive. It reacquires
the lock to recheck current plan expiry, concurrent idempotency acceptance and
physical exclusion before the single durable write. Actor authorization and request
cancellation are checked again after the probe. Once durably accepted, execution
continues with its detached lifetime. Controlled probe tests verify this protocol;
full live multi-host failure and internal test fault scopes remain open.

## Recorded process observation and control checks

Operators can explicitly request `GET /api/v1/networks/{networkId}/observations`
with their own target access. The service checks the physical target against its
stored identity, probes the recorded PID and compares the complete launch argv.
Darwin uses the target process table, including a matching executable column;
Linux uses the existing process inspector. Missing, unreadable and mismatching
processes remain distinct from running nodes. The response never includes argv,
credentials or raw inspection errors. The UI separates recorded and observed PIDs.
This observation does not rewrite state or adopt a process.

Positive recorded PIDs are checked when planning node controls and again before
executing them. Missing or mismatching processes are refused instead of stopping
whatever currently owns that PID. Exact argv comparison on Darwin cannot confirm
arguments containing spaces after its existing process-table splitting; those
cases are refused. PID-free residual process discovery, executable identity beyond
launch metadata and persisted reconciliation remain open.

Retained test networks provision the registered executable under its native name.
Node controls now bind this owned copy or the reviewed source executable to the
same selected asset checksum, refusing identical bytes at undeclared locations.
Target bytes are verified during planning, before execution and immediately before
the node control verb. This preserves native startup naming without making the
original asset filename a prerequisite for controlling test-owned nodes. Full
multi-binary replacement and file upload contracts remain unfinished.

## Restart during a native test

Test execution persists the accepted physical target privately before binary
provisioning and node launch. A server stopped inside the native suite can then
observe its recorded owned processes without requiring suite completion first.
Starting the server only marks unfinished jobs interrupted; it does not resume
the suite, stop nodes or reset data. Repeating the original idempotency key
returns the interrupted job. Explicit controls and a rerun use new plans and jobs.

The owned browser fixture kills and restarts only its disposable dashboard while
a native four-node WBFT test is running. It checks unchanged launch records and
data-directory identity, authenticated live PID observations, alias exclusion,
explicit stop and a new native test result while retaining the interrupted job.
This is development evidence for recorded-process recovery, not full WEB-12
acceptance: unrecorded residual processes, partial launch failure, stale PIDs and
unavailable remote access still require reconciliation coverage.

## Saved presets in composition jobs

Composition jobs can select `arguments.chainPresetRef` with a saved document ID
and explicit revision instead of a separate producer count. The server requires
the current revision at acceptance, then keeps its immutable content even if the
shared document is edited. The UI offers saved presets belonging to the selected
chain and shows the pinned revision and actual engine plan before execution.

The existing suite composition lowering produces the setup request: counts or
per-node tables, endpoint sync mode, peering, inline genesis overlays/hardforks,
scoped launch options and configuration knobs reach the same native verbs used by
the CLI. Launch and config lowering sorts keys so repeated review cannot change
the fingerprint merely because Go map iteration differs. Binary selection binds
the declared default to the registered asset and key inputs to the encrypted
snapshot. Table order also determines physical port claims.

The three-chain browser fixture verifies five actual node databases per preset,
the reviewed chain ID, scoped launch options, a table beginning with an endpoint,
stale-revision rejection and preservation of accepted content after edits. Each
chain also starts its archive endpoint, serves the reviewed chain ID over RPC,
passes recorded-process observation and stops explicitly. Archive is rendered as
full native sync with pruning disabled and explicit archive retention flags,
including config-free relaunches; writing archive as a sync mode was invalid. File
references, mixed binaries, generated/custom keys, declared accounts, external
preset execution and attachments remain refused until their pinned contracts are
connected. This is partial WEB-01/02 evidence, not completion of all preset fields
or every native option.

## Uploaded assets and native execution

Authenticated operators can register one multipart file with its kind through
`POST /api/v1/assets`. Viewer list/detail responses expose immutable IDs, exact
SHA-256 checksums, byte counts and compatibility metadata. The server chooses
private storage paths and publishes the bytes and receipt together; unfinished
uploads do not appear in the library. Filename traversal, links, empty or
oversized files and changed bytes are refused. Binary uploads are limited to
256 MiB and other files to 16 MiB. Storage failures return safe errors without
private filesystem paths.

Native binaries pass OS/architecture header inspection before version/help
probes. Observed program identity, commit, required flags and dialect determine
their supported built-in chain. Probe output and duration are bounded. This is
observed executable compatibility, not a source audit or proof of trust. The
settings library lists uploaded metadata and refreshes manifest selections. Jobs
resolve provisioned or uploaded IDs without accepting a client filesystem path,
show the selected ID/checksum and recheck bytes before executing the pinned job.

Configuration/template JSON and YAML, JSON ABI and hexadecimal bytecode can be
registered with format inspection. Known credential fields, PEM private keys and
encrypted key stores are refused as shared assets. Personal SSH material still
uses the encrypted credential store. File format inspection does not establish
engine compatibility: configuration/template/material execution references other than finished genesis,
uploaded private key trees, TOML and full test asset binding remain open.

The owned fixture uploads all three native binaries through the actual browser,
restarts the daemon, verifies persisted metadata and runs twelve native database
initializations plus explicit start/RPC/PID observation/stop jobs. Changing an
accepted asset rejects execution before database creation. API/storage tests
cover role/CSRF/origin checks, exact bytes, interrupted staging and links. This is
partial WEB-02/04/06 evidence; the fourteen full acceptance criteria remain open.


## Registered finished genesis in saved presets

A saved chain preset can select a registered JSON template using
`genesis.ref: asset:<id>` and `mode: existing`. Its outer `assetRefs` must exactly
match this declaration. Shared revisions and imports preserve the registered ID;
exports retain the reference rather than a private filesystem path. Planning
refuses unknown IDs, wrong kinds, arbitrary paths and incompatible generation
overrides. The review includes the selected ID and full SHA-256 checksum.

The engine materializes the exact bytes in a private immutable snapshot and
projects only its execution copy to the existing CLI genesis path contract. Both
the source and snapshot are rechecked before execution. Changed snapshots are
refused rather than silently repaired. Concurrent identical publications preserve
the bytes; links, changed files and cancelled materialization are refused. Native
genesis validation and initialization still run through the existing chain verbs;
format inspection alone does not establish full chain compatibility.

The browser offers registered JSON templates, replaces generation set/overlay
explicitly, and restores the previous generation declaration when undoing a file
selection in the current edit. The owned three-chain fixture uploads finished
genesis files, selects and saves them in the UI, reviews their checksums and
initializes twelve actual native databases. It compares the deployed file bytes
and each database's `dumpgenesis` chain ID. Wrong kinds, unknown references and
changes to either source or snapshot reject execution before network creation.
This is partial WEB-01/02/06 evidence. Genesis templates, per-binary genesis,
configuration/material inputs, uploaded key trees and the full acceptance list
remain open.


## Explicit node tables in test jobs

Saved test cases can use explicit `topology.nodes` tables with registered default
binaries and the reviewed key preset. `PlanSuiteLayout` returns the same resolved
composition request used by `RunSuite`; the web adapter takes sorted node labels
and roles directly from that request. It no longer approximates a table by
producer-first counts. Counts retain their existing placement order. Each review
shows the node label, role, P2P port and HTTP RPC port from actual allocation.

The private accepted payload records these ordered requests and checks them
again before native launch. Unsupported node config files, custom keys and named
binaries still require their own registered contracts and are refused. Existing
suite validation checks every selected case and its composition before planning.

The owned browser fixture uses unsorted declarations with an archive endpoint at
index one followed by four producers. All three native chains produce real
passing engine sessions without skips; fifteen node databases, roles, ports,
actual PID observations and endpoint chain RPC match review. Explicit owned stop
jobs precede native `dumpgenesis` reads, which require exclusive database access.
The initial fixture tried reading an active database; its lock error is not
product RED evidence. This is partial WEB-05/11 evidence. Complete DSL argument
coverage, file assets, custom key contracts, mixed binaries and attachments remain
open.

## Private node-key material at the shared document boundary

Shared chain presets and v1/v2 test cases refuse long hexadecimal material in
`topology.nodes[].key` before grammar validation, storage or export. Recognition
matches the execution boundary's conservative treatment of possible key material:
at least 32 hexadecimal digits, with optional `0x`/`0X` and surrounding whitespace.
The error never echoes the value. File references remain declarations; public
hashes, bytecode and other hexadecimal values outside this private reference field
are unaffected. The underlying DSL parser and CLI grammar are unchanged.

An invalid import returns no public declaration and cannot be committed. Its
original bytes remain actor-bound AES-GCM ciphertext, including across restart.
Editor import and validation apply the same boundary, including referenced
preset declarations supplied with a case. Import commit also revalidates previously approved declarations under current
rules rather than trusting the stored approval flag. The live authentication
fixture exercises preset and v1/v2 case save, validation and import refusal and
scans storage for its unique node-key marker. Unit tests cover export refusal,
file references, public hexadecimal text and encrypted original recovery.

This prevents new shared publication. Startup quarantine below protects legacy
records; neither change awards the complete secret-management acceptance criterion.
Private key upload/selection and execution bindings remain open.
RED/GREEN logs are recorded in the worklist. The first live run caught a
case-editor validation route that bypassed shared validation; the focused HTTP
RED and subsequent browser GREEN record closure of that additional path.

## Legacy node-key quarantine before API startup

Opening deployment storage checks every public chain/test revision and saved
import preview for node-key material. If any revision contains it, the entire
document history leaves the public namespace together; existing revision content
is never silently rewritten. The original history is preserved as authenticated
AES-GCM ciphertext bound to its document ID. An affected import preview is
invalidated and its previous public content is separately encrypted with its ID
and owner as associated data. Its original actor-private source ciphertext stays
unchanged. Unaffected documents, credentials, bindings and workspace references
are retained. References to quarantined documents fail instead of falling back.

One atomic snapshot publishes quarantine and its audit before the store is
returned to the dashboard. Failure prevents startup publication. Reopening checks
quarantine authentication and does not rewrite an already migrated snapshot;
tampered ciphertext fails closed. Known quarantined node-key values are learned
privately at startup and removed from historical JSON, plain logs and SSE output,
including keys that appeared only in an import preview. No quarantine originals
or decryption API are exposed. An owner-facing recovery workflow remains open.

Unit tests decrypt and compare complete histories, previews and source bytes,
check unaffected revisions and repeated restart, and inject actual write failure
and ciphertext tampering. The owned WEB-10 fixture seeds old preset/v1/v2 case
records only while its disposable server is stopped, restarts the dashboard and
checks public reads/export absence, old import refusal, plaintext storage scans
and actual SSE masking. This remains partial WEB-10 evidence.

## Finished genesis assets in saved test execution

The test editor selects registered JSON template assets for inline v2 environments
and legacy v1 cases. Selecting a finished file replaces generation settings;
clearing the selection restores the previously selected generation declaration.
The saved case keeps `asset:<id>` in its original grammar. Its exact dependency
list survives save, import and export. Filesystem references and unregistered or
wrong-kind assets cannot execute.

Each reviewed suite pins the source checksum and one private copy per dependency.
Cases sharing a source share its execution path so compatible environments remain
compatible. Only the private execution declaration replaces the reference with
that path; the public document and other values, including large integers, stay
intact. The plan displays asset IDs and SHA-256 checksums. The job panel includes
case dependencies alongside its registered default binary. Source and copy bytes
are checked before writing target metadata and again after binary provisioning,
immediately before the native suite entry point. A late change fails with any
already provisioned binary recorded as a partial effect.

The finished-file header check requires `config.chainId`; it does not certify
consensus compatibility. Native setup remains responsible for validating the
actual genesis. `run_genesis_test_jobs.py` prepares valid files using the existing
CLI, drives selection, restoration, validation, save, review and execution in
Chrome, and verifies three real engine sessions without skips. Its owned nodes
provide live PID/RPC observations and twelve initialized native databases.
Explicit stop jobs precede database genesis inspection. The receipt binds results
to the source digest and compiled dashboard. This is partial WEB-02/05/06 evidence,
not completion of the full DSL, assets or nine-function acceptance scope. Templates
for generation, named presets, uploaded key trees, accounts, mixed binaries,
per-binary genesis and attachments retain separate unfinished contracts.

The live fixture exposed a PoA engine gap: using a finished file skipped the
governance input needed by `wemix deploy-governance`, documented in
`docs/chain-analysis/gwemix/cli-graph.md`. The existing genesis source now offers
an optional auxiliary-input contract. PoA uses its normal governance builder
with the accepted keys and actual producer placement, without invoking genesis
generation or replacing the uploaded bytes. Static families need no extra input.
The ordinary target artifact writer carries the resulting governance file to
native bootstrap, including remote targets. This uses the same default policy
as a generated composition; importing a custom governance policy remains a
separate unfinished configuration contract. Unit tests hold exact genesis bytes,
producer membership and ports and refuse missing placement or cancellation.

## Saved configurations in the execution catalog

A successful case, chain, deployment document or workspace save refreshes the
execution catalog without remounting its editors or reloading the page. Workspace,
binary, preset and case selections retain their IDs. An unaccepted review is
cleared and must be prepared again; accepted jobs continue independently. The
catalog includes current document revisions and any older revisions pinned by
workspaces. Saving a server or path document does not silently rebind a workspace.
The loading state prevents preparation against an incomplete refresh. A request
generation and actor check before publication also discards delayed older
responses, including delayed pinned-document reads.

`run_catalog_refresh.py` drives actual chain/server/path/workspace saves and
reviews the resulting pinned revisions without starting nodes. The native
genesis-test fixture saves multiple case revisions, delays an older HTTP response
and verifies the latest selection remains intact before three real native tests.
Both receipts bind the observations to source and dashboard digests. This is a
partial configuration-flow regression, not full acceptance of all DSL or asset
contracts.

## Explicit non-producer reset

The public reset adapter is registered as the effectful `node.reset` feature;
the architecture coverage limit remains unchanged.

`node.reset` reviews one owned endpoint or proxy with a recorded running PID.
Producer, unknown-role, missing-PID and unresolved named-binary records are refused.
The UI excludes these selections and explains that reset replaces existing node
data with the recorded genesis and leaves the node stopped. Planning and execution
verify the owned record, registered executable, full process arguments, accepted
workspace/server declarations, exact generated node paths and their physical
target identity. Config and genesis checksums must match the recorded launch
inputs before the native reset verb can remove data. Changed inputs or a redirected
node directory fail without effects. A failed native reset retains resource
exclusion and reports possible partial initialization for explicit recovery.

The existing native reset performs stop, datadir replacement and genesis init;
sibling nodes keep their processes and data. Relaunch uses a separate reviewed
`node.start` job. PID-zero records remain refused for reset until residual-process
reconciliation provides evidence of vacancy. Binary/config replacement, stopped
node reconciliation and complete WEB-07 acceptance remain open.

`run_node_reset.py` runs generated-genesis cases across all three native chains,
drives reset in Chrome, reads native `LastBlock` and `LastHeader` keys against the
live genesis block hash, checks stopped state and producer preservation, then
explicitly relaunches and stops the owned nodes. The chain-analysis CLI graphs
document the native `db get` and `dumpgenesis` commands used for these observations.
The fixture also exposed an empty-dependency serialization regression: optional
genesis snapshots now compare absent and empty lists consistently after accepted
plan persistence, while file-bearing snapshots still compare their exact contents.
