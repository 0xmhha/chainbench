## Module sizes (top by LOC)

| module | files | loc | types | funcs | languages |
|---|---|---|---|---|---|
| core | 23 | 8145 | 44 | 278 | go |
| core/vm | 23 | 7511 | 67 | 338 | go |
| core/rawdb | 21 | 6773 | 30 | 369 | go |
| eth/downloader | 17 | 6273 | 36 | 168 | go |
| core/types | 31 | 6063 | 64 | 403 | go |
| trie | 16 | 5040 | 41 | 244 | go |
| eth/protocols/snap | 8 | 4803 | 44 | 106 | go |
| core/state/snapshot | 13 | 4598 | 30 | 147 | go |
| rpc | 19 | 4365 | 60 | 218 | go |
| accounts/usbwallet/trezor | 5 | 4089 | 49 | 540 | go |
| cmd/gstable | 11 | 4027 | 5 | 76 | go |
| eth | 16 | 3907 | 22 | 175 | go |

## Fan-in (most-imported internal modules)

| module | imported by N modules |
|---|---|
| log | 62 |
| core/types | 58 |
| common/math | 42 |
| rlp | 39 |
| crypto | 39 |
| params | 37 |
| common/hexutil | 35 |
| metrics | 30 |
| ethdb | 28 |
| rpc | 21 |
| core/rawdb | 21 |
| event | 21 |

## Top supertypes / adopted interfaces

(no inheritance edges extracted — expected for Go/Rust-only trees)

## Totals

- modules: 129, files: 668, loc: 184640, types: 1528, type_edges: 0
