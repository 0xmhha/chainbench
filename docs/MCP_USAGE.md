# Chainbench MCP — 사용 가이드 (한 장)

Chainbench 의 MCP 서버는 LLM/코딩 에이전트(Claude Code 등)가 체인을 init/start/stop,
tx 전송, 컨트랙트 배포·호출, 로그/이벤트 조회, 원격·SSH 네트워크 attach 등을 **도구(tool)
호출**로 수행하게 한다. stdio 전송(JSON-RPC) 기반이며, 프로젝트의 `.mcp.json` 에 등록해 쓴다.

> TL;DR
> ```bash
> bash setup.sh                                              # 1) Go 바이너리 빌드 + PATH 등록 (1회)
> # 프로젝트의 .mcp.json 에 등록:
> #   { "mcpServers": { "chainbench": { "command": "chainbench-mcp" } } }
> ```

---

## 1. 사전 요구사항
- Go ≥ 1.25, Python 3, git, bash, curl (Node/npm 불필요)

## 2. 설치 (1회) — `setup.sh`
```bash
cd <chainbench-checkout>
bash setup.sh
```
- `[1/3]` Go 바이너리 빌드: `chainbench`(CLI), `chainbench-mcp`(MCP stdio 서버), `chainbench-dashboard`(대시보드) → `bin/`
- `[2/3]` `chainbench`, `chainbench-mcp` 를 `/usr/local/bin` 에 심링크 등록
- `[3/3]` 완료

## 3. 프로젝트에 등록
사용할 프로젝트의 `.mcp.json` 에 아래를 기록한다 (**머신 독립적 — 절대경로 없음**):
```json
{ "mcpServers": { "chainbench": { "command": "chainbench-mcp" } } }
```
MCP 클라이언트(Claude Code)는 프로젝트의 `.mcp.json` 을 읽어 자동 연결한다.

## 4. 동작 원리
- `chainbench-mcp` 는 **단일 Go 바이너리** stdio(JSON-RPC 2.0) MCP 서버다(`cmd/chainbench-mcp`,
  `internal/mcp`). 별도 wire 프로세스나 TypeScript 런타임이 없다.
- 표면 규칙은 비대칭이다(아키텍처 v2 §2). **CLI 는 코어 모듈을 직접 호출하고, MCP 는
  `internal/app` 을 거친다.** 두 표면이 같은 기능을 부르므로 동작은 같다.
- 아직 코어를 직접 부르는 도구가 남아 있고, 그 목록은 `internal/arch/mcp_imports_test.go`
  의 허용표에 이유와 함께 적혀 있다. 이 표는 줄어들기만 한다(항목이 사라지면 테스트가
  삭제를 요구한다).

## 5. 노출 도구
도구 개수는 고정이 아니다. 정적으로 등록되는 도구(`internal/mcp/tools.go` 의 `Default`)
위에, 임포트된 체인 플러그인의 capability 카탈로그에서 파생된 도구가 더해진다. **정확한
목록은 `tools/list` 로 얻는다**(§6 의 stdio 스모크가 그것이다).

| 그룹 | 도구 |
|---|---|
| 체인/셋업 | `chainbench_chains` · `_setup_plan` · `_start` · `_stop` · `_status` · `_run` |
| 검증/테스트 | `chainbench_verify` · `_test` · `_test_list` · `_report` |
| 노드/Tx | `chainbench_node_rpc` · `_tx_send` · `_tx_wait` · `_txpool` · `_account_state` · `_contract_call` · `_contract_deploy` · `_faucet` |
| 합의 | `chainbench_consensus` · `_consensus_status` · `_consensus_health` · `_consensus_block_info` |
| 네트워크 구성(`net`) | `chainbench_net_new` · `_net_keys` · `_net_allocate` · `_net_genesis` · `_net_config` · `_net_launchopts` · `_net_provision` · `_net_init` · `_net_start` · `_net_stop` · `_net_restart` · `_net_rm` · `_net_status` · `_net_health` · `_net_logs` |
| 노드 배치(`netmap`) | `chainbench_netmap_show` · `_netmap_pool` · `_netmap_plan` |
| 기존 네트워크 attach | `chainbench_network_attach` · `_network_list` · `_network_info` · `_network_detach` · `_network_peers` · `_network_topology` · `_remote_rpc` |
| 키 재료 | `chainbench_keyring_new` · `_keyring_add` · `_keyring_list` · `_keyring_show` · `_keyring_import` |
| 로그 | `chainbench_log` · `_log_timeline` |
| capability(파생) | 등록된 capability 마다 하나 — 이름은 `chainbench.<version>.<chain>.<name>`(예: `chainbench.v1.stablenet.governance.proposals`) · 목록 도구 `chainbench.capabilities` |

> keyring 에는 **export 도구가 없다.** 개인키를 내보내는 것은 CLI 전용이며, MCP 표면에
> 나타나지 않도록 테스트로 고정돼 있다.

> `network_attach` 는 RPC 엔드포인트를 probe 해 이름 붙은 네트워크로 저장한다. 자격증명은
> env-var 이름만 전달하고(시크릿 인라인 금지), `remote_rpc`/`network_topology` 가 저장된 인증으로 접근한다.

## 6. 개발/직접 실행 (참고)
```bash
go build -o bin/chainbench-mcp ./cmd/chainbench-mcp
go test ./...                                   # 전체 테스트 (실 바이너리 불필요)
# stdio 스모크: initialize + tools/list 를 파이프로 전달
printf '%s\n%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./bin/chainbench-mcp
```

## 7. 트러블슈팅
| 증상 | 원인 / 해결 |
|---|---|
| MCP 'chainbench' not connected | `setup.sh` 미실행 또는 `chainbench-mcp` 가 PATH 에 없음 → `bash setup.sh` |
| 도구 호출이 바이너리 없음으로 실패 | `_start` 등 실행 도구는 빌드된 체인 바이너리 경로(`binary` 인자)가 필요 |

---

참고 소스: `cmd/chainbench-mcp`(stdio 서버) · `internal/mcp`(도구 등록 `tools.go` + 핸들러
`*_tools.go`) · `internal/app`(MCP 가 지나는 워크플로 계층) · `internal/core/*`(코어).
