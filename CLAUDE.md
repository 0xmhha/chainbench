# Chainbench project instructions

## Web UI status

The Web UI renewal is finished; WEB-01..14 passed on 72a7b464. Read
`docs/dev/web-ui-status.md` first: what is done, what is still to decide,
what was not verified, and the principles still in force. Screen rules are in
`web/DESIGN.md`, users and scenes in `web/PRODUCT.md`, the API contract in
`docs/dev/web-ui.openapi.yaml`. Use Ouroboros only for qa, lateral and
evaluate; implement directly, without `ooo run` or Ralph.

Before each next stage, define its completion tests and use meaningful
RED -> GREEN. Never award completion for mocks, skipped scenarios, stale
receipts, missing evidence, environmental failures, or artifact existence alone.
New entry points add to existing flows: keep the mouse-driven setup, and have
new APIs call the existing plan and job functions.

## Git and review rules

- Keep unrelated user changes intact.
- Do not use `git add .`, `git commit -a`, destructive resets, or cleaning the
  checkout. Commit only explicit files owned by the current work.
- Commit and PR titles use English Conventional Commit style. Explain concrete
  behavior without stage jargon, obscure abbreviations, emoji, or co-author tags.
  Keep each complete commit message within 300 characters.
- PRs use English. The user intends squash merge; summarize titles rather than
  concatenate lengthy commit bodies when preparing the squash message.
- Never request review from SAGE Core Team, including
  `SAGE-X-project/sage-core-dev`, directly or through automatic assignment.
  Inspect CODEOWNERS/workflows/settings before creating or making a PR ready.
  Keep a PR draft if its base automatically assigns that team.
- Branch from main, one PR per change, and squash-merge only after all CI
  checks pass.

## Architecture and verification

- Reuse Go use cases in `internal/app`, existing CLI/MCP engines, and Svelte 5.
  Respect `internal/arch`. Avoid an unrelated repository refactor.
- Form choices come from engine-owned schemas, registries, and validated chain
  surfaces. Use `docs/chain-analysis` and the selected binary's actual help.
- `web` builds into `internal/dashboard/spa`. Finish the SPA build before any
  Go embed build; freeze source throughout a source-bound proof.
- Use genuinely compatible native binaries and exclusively owned local/SSH
  fixtures. Never substitute simulated processes for blockchain acceptance.
- Recorded PIDs and cached node state are not proof of liveness or ownership.
  Do not signal processes based only on their stored PID.
- On this Mac use `WEBUI_RUNTIME_ROOT=/private/tmp/cbui` to keep IPC paths short.
  `GOCACHE=/private/tmp/chainbench-go-cache` is the task's isolated cache.
- Serialize native proofs that share fixed RPC/P2P/SSH ports and original
  acceptance captures that rebuild the same embedded SPA.
- Disk below 2 GiB causes existing genesis preflight refusals. Preserve those
  checks. Native proofs refuse to start below 20 GiB free
  (`tests/webui/runtime_storage.py`).
- WEB-14 needs chain binaries built at the commits `docs/chain-analysis`
  records (stablenet 4d17eea, wbft d535dd3, wemix 4538a9f), not the chain
  repositories' HEAD.
- Secrets belong in private encrypted stores, not shared export, documentation,
  logs, patches, or portable handoff bundles.

## Common checks

Run from the project root unless specified:

```sh
GOCACHE=/private/tmp/chainbench-go-cache make check
GOCACHE=/private/tmp/chainbench-go-cache make build
node --test tests/webui/*.test.mjs
WEBUI_RUNTIME_ROOT=/private/tmp/cbui PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/webui -p 'test_*.py'
```

The frontend build is `npm run build` from `web/`. Full final acceptance also
requires full race tests, e2e vet, and all fourteen fresh independent live proofs
specified by the approved Seed; the shorter checks above do not replace them.
