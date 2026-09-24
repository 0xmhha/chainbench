## Module sizes (top by LOC)

| module | files | loc | types | funcs | languages |
|---|---|---|---|---|---|
| internal/chainsetup | 106 | 19905 | 123 | 707 | go |
| internal/testengine | 61 | 10255 | 57 | 350 | go |
| internal/testhelper | 37 | 7887 | 42 | 274 | go |
| internal/resource | 32 | 5550 | 43 | 207 | go |
| cmd/chainbench | 23 | 4823 | 4 | 147 | go |
| internal/core/keyring | 28 | 4574 | 45 | 179 | go |
| internal/app | 35 | 4421 | 45 | 201 | go |
| internal/mcp | 31 | 3958 | 6 | 151 | go |
| internal/dsl | 18 | 3914 | 18 | 120 | go |
| internal/arch | 22 | 3500 | 13 | 85 | go |
| internal/core/process | 35 | 3318 | 22 | 137 | go |
| cmd/chainbench/keyringcmd | 14 | 2783 | 11 | 118 | go |
| internal/core/collector | 28 | 2710 | 30 | 108 | go |
| internal/consensus/poa | 18 | 2661 | 13 | 105 | go |
| docs/research/common-tests | 17 | 2588 | 0 | 31 | python |
| internal/core/nodeconfig | 16 | 2461 | 24 | 104 | go |
| internal/core/session | 20 | 2455 | 25 | 108 | go |
| internal/core/blueprint | 14 | 2255 | 18 | 61 | go |
| internal/dsl/interp | 13 | 2028 | 32 | 97 | go |
| internal/core/node | 21 | 1908 | 17 | 80 | go |
| internal/core/genesis | 11 | 1813 | 12 | 61 | go |
| internal/core/registry | 14 | 1724 | 24 | 59 | go |
| internal/accounts | 22 | 1664 | 9 | 87 | go |
| tests/e2e | 13 | 1597 | 2 | 62 | go |
| internal/chainsetup/verb | 14 | 1578 | 46 | 49 | go |
| cmd/chainbench/chaincmd | 13 | 1518 | 1 | 49 | go |
| internal/core/statemachine | 8 | 1361 | 12 | 54 | go |
| cmd/chainbench/suitecmd | 6 | 1165 | 3 | 38 | go |
| internal/core/remote | 10 | 1153 | 9 | 51 | go |
| internal/consensus/wbft | 8 | 1126 | 3 | 49 | go |
| internal/core/health | 5 | 878 | 14 | 50 | go |
| internal/core/lifecycle | 4 | 846 | 2 | 10 | go |
| internal/nodemonitor | 7 | 813 | 14 | 32 | go |
| internal/core/rpc | 4 | 811 | 8 | 44 | go |
| internal/feature | 8 | 784 | 4 | 23 | go |
| cmd/chainbench/lifecyclecmd | 9 | 699 | 0 | 24 | go |
| internal/dashboard | 8 | 671 | 2 | 29 | go |
| internal/core/filestore | 5 | 664 | 7 | 27 | go |
| internal/dsl/assert | 3 | 587 | 1 | 44 | go |
| internal/chains/stablenet | 5 | 571 | 0 | 32 | go |

## Fan-in (most-imported internal modules)

| module | imported by N modules |
|---|---|
| internal/core/node | 23 |
| internal/chains/all | 21 |
| internal/core/registry | 19 |
| internal/app | 17 |
| internal/preset | 15 |
| cmd/chainbench/surface | 13 |
| internal/resource | 13 |
| internal/core/process | 11 |
| internal/core/session | 11 |
| internal/core/filestore | 11 |
| internal/core/collector | 10 |
| internal/core/rpc | 10 |
| internal/mcp | 10 |
| internal/accounts | 9 |
| internal/core/keyring | 9 |
| internal/chainsetup | 7 |
| internal/core/remote | 5 |
| internal/testsupport | 5 |
| internal/dashboard | 4 |
| internal/core/home | 4 |
| internal/core/nodeconfig | 4 |
| internal/dsl | 4 |
| cmd/chainbench/resourcecmd | 3 |
| internal/core/health | 3 |
| internal/chains/wemix | 3 |
| internal/chains/stablenet | 3 |
| internal/chainsetup/verb | 3 |
| internal/core/report | 3 |
| internal/consensus/poa | 3 |
| cmd/chainbench/exitcode | 2 |
| cmd/chainbench/keyringcmd | 2 |
| internal/feature | 2 |
| internal/core/genesis | 2 |
| internal/core/wait | 2 |
| internal/core/origin | 2 |
| internal/core/lifecycle | 2 |
| internal/chains/external | 2 |
| internal/core/blueprint | 2 |
| internal/core/statemachine | 2 |
| internal/core/preflight | 2 |

## Top supertypes / adopted interfaces

(no inheritance edges extracted — expected for Go/Rust-only trees)

## Totals

- modules: 75, files: 852, loc: 124520, types: 798, type_edges: 0
