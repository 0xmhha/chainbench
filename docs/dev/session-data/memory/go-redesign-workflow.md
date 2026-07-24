---
name: go-redesign-workflow
description: Branch/PR workflow and layout for the chainbench Go-first multi-chain redesign
metadata: 
  node_type: memory
  type: project
  originSessionId: 305c46ea-8b16-4e4d-bb11-a9b29922f3b5
  modified: 2026-07-24T12:29:05.209Z
---

The chainbench Go-first multi-chain redesign (SSoT: `docs/CHAINBENCH_GO_REDESIGN.md`)
runs across two repos. Work in chunks: one feature branch per chunk, squash-merged
to `main` via PR, then a fresh branch for the next chunk.

**First chunk MERGED (2026-07-24):** chainbench PR #16 (`feat/go-redesign-core`)
and accounts PR #5 (`feat/multichain-protocol`) are squash-merged to their mains.
G0-G8 core + CLI (9 cmds) + MCP (6 tools) + dashboard are on `main`.
Continue on NEW branches off main (e.g. `feat/obs-report`).

- chainbench (`/Users/wm-it-25_0220/Work/github/chainbench`): root Go module
  `github.com/0xmhha/chainbench`; the legacy `network/` module is a nested module,
  untouched, absorbed incrementally. New code under `pkg/` + `manifests/` + `tests/`.
- accounts (`/Users/wm-it-25_0220/Work/github/accounts`): the `protocol` package.
  chainbench go.mod uses `replace github.com/0xmhha/accounts => ../accounts` (local,
  co-developed). CI note: switch to a tag/pseudo-version when accounts is versioned.

**Why:** user directive (2026-07-24) — a commit guard blocks committing while the
chainbench working dir is on `main`, so always work on a feature branch. The user
squash-merges PRs themselves and syncs main, then continues.

**How to apply:** before committing, ensure both repos are on their feature
branch. Commit messages in English, NO Co-Authored-By line (user preference).
Phase status is tracked in the roadmap table of [[go-redesign-workflow]]'s SSoT
doc. Roadmap: A0/A1 (accounts) done; chainbench G0–G4 core done; G5 (wbft chain),
G6 (MCP Go), G7 (poa/wemix+etcd), G8 (hardfork+dashboard) remain, plus the
cross-cutting `network/` absorption (drivers/probe/wire, real genesis golden,
retire bash/TS).
