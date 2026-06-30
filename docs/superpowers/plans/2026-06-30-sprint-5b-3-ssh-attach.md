# Sprint 5b.3 — network.attach ssh-remote — Plan

> 짝 spec: `2026-06-30-sprint-5b-3-ssh-attach.md`. 범위: wire 레벨만. 커밋: English, no co-author, no emoji.
> 검증된 사실: `probe.Options.Client *http.Client` 존재 → 터널 transport 주입으로 probe 변경 0.

## Task 0 — spec + plan 커밋
- 브랜치 `feat/sprint-5b-3-ssh-attach`.
- commit: `docs: add Sprint 5b.3 spec + plan for ssh-remote attach`

## Task 1 — sshremote.DialTunnelClient
- `sshremote.go`: `tunnelTransport(sshC *ssh.Client) *http.Transport` 헬퍼 추출(Dial 의 인라인 transport 를 이걸로 — 동작 불변). `DialTunnelClient(creds, hostKey) (*http.Client, io.Closer, error)`: dialSSH → `&http.Client{Transport: tunnelTransport(sshC)}`, closer=sshC.
- `Dial` 이 `tunnelTransport` 재사용하도록 리팩토(기존 sshremote 테스트 green 유지).
- 테스트: `sshremote_test.go` 에 DialTunnelClient happy(기존 tunnel 서버 재사용 — mock RPC 로 GET) + bad password.
- commit: `feat(sshremote): DialTunnelClient for tunneled probe`

## Task 2 — attach ssh-remote 분기
- `handlers_network.go newHandleNetworkAttach`: req 에 `Provider string`, `ProviderMeta json.RawMessage`(또는 map) 추가.
  - provider=="" || "remote": 기존 경로 그대로(provider=remote). 무변경.
  - provider=="ssh-remote": creds=sshCredsFromNode(임시 Node{Auth}) → hostKey=ResolveHostKeyCallback → DialTunnelClient → defer close → probe.Detect(Client) → 노드(provider=ssh-remote, auth, provider_meta) → SaveRemote.
  - else: INVALID_ARGS.
- provider_meta 파싱: `map[string]any` 로 unmarshal 후 `types.NodeProviderMeta`.
- 에러 분류(§6).
- 테스트는 Task 3.
- commit: `feat(network-net): network.attach supports ssh-remote provider`

## Task 3 — attach ssh-remote 테스트
- 핸들러 테스트(package main): in-process SSH **tunnel** 서버(direct-tcpip → mock JSON-RPC: eth_chainId=0x205b + istanbul_getValidators=[] → stablenet/8283). `network.attach{provider:ssh-remote,...}` → 저장 노드 provider/chain_id/provider_meta 검증.
  - tunnel 서버 harness: 5b.1 sshremote_test 의 direct-tcpip 서버 패턴을 package main 테스트에 (handlers_ssh_test.go 의 exec 서버는 session-only라 tunnel 추가 필요 — 별도 helper).
- 에러 경로(SSH 불필요): unknown provider, ssh-password 누락/타입불일치 → INVALID_ARGS; env 빔 → UPSTREAM. password 미노출.
- remote attach 무회귀(기존 network-attach 테스트 + node-* 테스트).
- commit: `test(network-net): attach ssh-remote via tunneled probe`

## Task 4 — 문서 + 버전
- VISION 5b: 5b.3(attach 구성) 항목 추가/체크. REMAINING_WORK: ssh-remote 수동구성 v1 → attach 가능, CLI/MCP 표면만 후속.
- 버전 0.10.2 → 0.11.0(minor — 신규 구성 능력).
- commit: `docs+chore(sprint-5b-3): roadmap + remaining-work + version 0.11.0`

## 완료 기준
- [ ] network.attach{provider:ssh-remote} 가 터널 probe 로 chain_id 감지 + ssh-remote 노드(auth+provider_meta) 저장
- [ ] remote attach 무회귀(provider 미지정 경로 불변)
- [ ] 에러 분류(unknown provider/auth 불완전 → INVALID_ARGS; SSH/probe 실패 → UPSTREAM)
- [ ] password 미노출
- [ ] 전 레이어 green: Go(전 패키지, vet/gofmt) · vitest · bash · go.mod tidy 안정
