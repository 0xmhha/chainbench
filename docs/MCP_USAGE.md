# Chainbench MCP — 사용 가이드 (한 장)

Chainbench 의 MCP 서버는 LLM/코딩 에이전트(Claude Code 등)가 체인을 init/start/stop,
tx 전송, 컨트랙트 배포·호출, 로그/이벤트 조회, 원격·SSH 네트워크 attach 등을 **도구(tool)
호출**로 수행하게 한다. stdio 전송(JSON-RPC) 기반이며, 프로젝트의 `.mcp.json` 에 등록해 쓴다.

> TL;DR
> ```bash
> bash setup.sh                                              # 1) MCP 빌드 + PATH 등록 (1회)
> go -C network build -o bin/chainbench-net ./cmd/chainbench-net   # 2) wire 바이너리 빌드 (1회)
> chainbench mcp enable                                      # 3) 현재 프로젝트의 .mcp.json 에 등록
> ```

---

## 1. 사전 요구사항
- Node ≥ 18 + npm, Go ≥ 1.25, Python 3, git, bash, curl
- 체크아웃 위치: 런처는 기본적으로 `CHAINBENCH_DIR=$HOME/.chainbench` 로 해석 (다른 경로면 `CHAINBENCH_DIR` 를 export)

## 2. 설치 (1회)

### 2.1 MCP 서버 빌드 + PATH 등록 — `setup.sh`
```bash
cd "$CHAINBENCH_DIR"   # 예: ~/.chainbench (또는 체크아웃 경로)
bash setup.sh
```
- `[1/3]` `mcp-server` 빌드 (`npm install` + `npm run build` → `mcp-server/dist/index.js`)
- `[2/3]` `chainbench`, `chainbench-mcp` 런처를 `/usr/local/bin` 에 심링크 등록
- `[3/3]` 완료

### 2.2 Go wire 바이너리 빌드 (필수 — setup.sh 가 안 함)
`init/start/stop/restart/clean/status`, `node_rpc`, `network_*` 등 **wire 경유 도구**는
Go 바이너리 `chainbench-net` 을 spawn 한다. `resolveBinary()` 가 `CHAINBENCH_NET_BIN` →
`$CHAINBENCH_DIR/bin` → `$CHAINBENCH_DIR/network/bin` 순으로 찾으므로 한 번 빌드해 둔다:
```bash
go -C network build -o bin/chainbench-net ./cmd/chainbench-net
```
> Go 측 코드를 바꾸면 재빌드 필요. 안 빌드하면 wire 도구만 `chainbench-net binary not found` 로 실패하고, bash-spawn 도구(test/log/profile/remote 등)는 동작한다.

## 3. 프로젝트에 등록 — `chainbench mcp`
사용할 프로젝트 디렉토리에서:
```bash
chainbench mcp enable                       # 현재 dir 의 .mcp.json 에 등록
chainbench mcp enable --target /path/proj   # 다른 dir 지정
chainbench mcp status                        # 등록 여부 확인
chainbench mcp disable                       # 해제
```
`.mcp.json` 에 아래가 기록된다 (**머신 독립적 — 절대경로 없음**):
```json
{ "mcpServers": { "chainbench": { "command": "chainbench-mcp" } } }
```
MCP 클라이언트(Claude Code)는 프로젝트의 `.mcp.json` 을 읽어 자동 연결한다.

## 4. 동작 원리
- `bin/chainbench-mcp` 런처가 런타임에 `CHAINBENCH_DIR`(기본 `$HOME/.chainbench`)을 풀고
  `mcp-server/dist/index.js`(stdio MCP 서버, `StdioServerTransport`)를 실행.
- MCP 도구는 두 경로로 chainbench 를 호출: (a) `callWire` → Go `chainbench-net` (NDJSON wire),
  (b) `runChainbench` → bash CLI spawn. 둘 다 `CHAINBENCH_DIR` 기준.

## 5. 노출 도구 (50개, 그룹별 요약)
| 그룹 | 도구 |
|---|---|
| 라이프사이클 | `chainbench_init` · `_start` · `_stop` · `_restart` · `_clean` · `_status` · `_state_compact` |
| 노드/Tx | `chainbench_node_start` · `_node_stop` · `_node_rpc` · `_tx_send` · `_tx_wait` · `_txpool_inspect` · `_account_state` · `_contract_deploy` · `_contract_call` · `_events_get` |
| 테스트/리포트 | `chainbench_test_run` · `_test_list` · `_test_regression` · `_test_hardfork` · `_test_run_remote` · `_report` · `_failure_context` |
| 네트워크 | `chainbench_network_attach` · `_network_detach` · `_network_list` · `_network_capabilities` · `_network_peers` · `_network_topology` · `_network_partition` |
| 원격 | `chainbench_remote_add` · `_remote_list` · `_remote_info` · `_remote_remove` · `_remote_rpc` |
| 합의 | `chainbench_consensus_status` · `_consensus_health` · `_consensus_validators` · `_consensus_block_info` |
| 로그 | `chainbench_log_search` · `_log_timeline` |
| 설정/프로파일 | `chainbench_config_get` · `_config_set` · `_config_list` · `_profile_get` · `_profile_set` · `_profile_send` |
| 스키마/스펙 | `chainbench_schema_query` · `_spec_lookup` |

> `network_attach`/`_detach`/`_list` 는 remote + ssh-remote 노드 구성·관리 (자격증명은 env-var 이름만 전달, 시크릿 인라인 금지).

## 6. 개발/직접 실행 (참고)
```bash
cd mcp-server
npm run start   # node dist/index.js  (빌드본 stdio 서버)
npm run dev     # tsx src/index.ts    (소스 직접 — 개발)
npm test        # vitest
```

## 7. 트러블슈팅
| 증상 | 원인 / 해결 |
|---|---|
| MCP 'chainbench' not connected | `setup.sh` 미실행(빌드/심링크) 또는 `chainbench-mcp` 가 PATH 에 없음 → `bash setup.sh` |
| `MCP server not found at .../dist/index.js` | MCP 미빌드 → `cd mcp-server && npm install && npm run build` |
| wire 도구만 `chainbench-net binary not found` | Go 바이너리 미빌드 → §2.2. 또는 `CHAINBENCH_NET_BIN` 으로 절대경로 지정 |
| 잘못된 체크아웃을 가리킴 | `CHAINBENCH_DIR` 가 기본값(`$HOME/.chainbench`)과 다르면 export 후 클라이언트 재시작 |

---

참고 소스: `lib/cmd_mcp.sh`(enable/disable/status) · `bin/chainbench-mcp`(런처) ·
`mcp-server/src/index.ts`(stdio 서버 + 도구 등록) · `mcp-server/src/utils/wire.ts`(`resolveBinary`).
