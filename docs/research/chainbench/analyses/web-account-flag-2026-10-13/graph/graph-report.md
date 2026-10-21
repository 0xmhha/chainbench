## Module sizes (top by LOC)

| module | files | loc | types | funcs | languages |
|---|---|---|---|---|---|
| internal/app | 182 | 28089 | 205 | 942 | go |
| internal/chainsetup | 128 | 22899 | 139 | 838 | go |
| internal/testengine | 74 | 11956 | 59 | 415 | go |
| internal/testhelper | 54 | 11522 | 64 | 393 | go |
| internal/resource | 44 | 6704 | 51 | 238 | go |
| cmd/chainbench | 25 | 5377 | 5 | 166 | go |
| internal/dashboard | 43 | 4993 | 13 | 116 | go |
| tests/webui | 94 | 4928 | 19 | 165 | python |
| internal/core/keyring | 28 | 4573 | 45 | 179 | go |
| internal/dsl | 21 | 4566 | 19 | 141 | go |
| internal/mcp | 37 | 4541 | 6 | 168 | go |
| internal/arch | 28 | 4525 | 18 | 114 | go |
| internal/core/session | 35 | 4136 | 37 | 173 | go |
| internal/core/process | 40 | 4104 | 32 | 165 | go |
| internal/core/nodeconfig | 20 | 2811 | 25 | 114 | go |
| cmd/chainbench/keyringcmd | 14 | 2783 | 11 | 118 | go |
| internal/core/collector | 29 | 2770 | 30 | 111 | go |
| internal/consensus/poa | 18 | 2722 | 13 | 108 | go |
| internal/core/blueprint | 14 | 2255 | 18 | 61 | go |
| internal/core/node | 21 | 2193 | 17 | 89 | go |

## Fan-in (most-imported internal modules)

| module | imported by N modules |
|---|---|
| internal/core/node | 25 |
| internal/chains/all | 23 |
| internal/core/registry | 20 |
| internal/app | 19 |
| internal/resource | 15 |
| internal/preset | 15 |
| cmd/chainbench/surface | 13 |
| internal/core/session | 12 |
| internal/core/process | 12 |
| internal/core/filestore | 11 |
| internal/core/collector | 10 |
| internal/core/rpc | 10 |
| internal/mcp | 10 |
| internal/accounts | 9 |
| internal/core/keyring | 9 |
| internal/chainsetup | 8 |
| internal/core/nodeconfig | 7 |
| internal/core/remote | 6 |
| internal/testsupport | 5 |
| internal/dsl | 5 |

## Top supertypes / adopted interfaces

| supertype | adoption count |
|---|---|
| (unittest.TestCase) | 19 |

## Totals

- modules: 77, files: 1241, loc: 173460, types: 1085, type_edges: 19
