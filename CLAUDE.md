# Chainbench project instructions

## Continuing the Web UI work

Read `docs/dev/web-ui-claude-handoff.md` first. It records the paused work,
source-of-truth documents, uncommitted implementation, failed native proof,
verification commands, and exact resume point. The paste-ready continuation
prompt with Ouroboros usage is `docs/dev/web-ui-claude-resume-prompt.md`.
Reuse the validated Seed; inspect active workers before any `ooo run` or Ralph
start, preserve uncommitted inputs, and distinguish QA from formal evaluation.
Then read the approved Seed,
specification, acceptance contract, and all fourteen worklist entries it links.

The handoff is a snapshot, not a completion certificate. Inspect current files
and `git status` before editing. The user paused Codex implementation to move
this work to Claude Code. Resume implementation when the user requests it.

When resuming, continue until all approved requirements are verified. Before
every next implementation stage, list all fourteen items, distinguish partial
implementation from full acceptance, and define that stage's completion tests.
Use meaningful RED -> GREEN, required regression checks, then commit the verified
stage and continue. Never award completion for mocks, skipped scenarios, stale
receipts, missing evidence, environmental failures, or artifact existence alone.
Do not reduce the approved scope or silently ignore unsupported input.

## Git and review rules

- Keep unrelated user changes intact. In this checkout `.gitignore` was already
  staged by the user; refactoring-preparation documents and baselines are theirs.
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
- The current branch is `codex/web-ui`, with draft PR #443. Do not switch to main
  or create a duplicate implementation branch just to start this handoff.

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
  checks. See the handoff's rejected cleanup action before considering deletion.
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
