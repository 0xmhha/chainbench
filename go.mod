module github.com/0xmhha/chainbench

go 1.25.0

require github.com/0xmhha/accounts v0.1.0

require (
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	golang.org/x/crypto v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

// accounts is co-developed locally (branch feat/multichain-protocol). Until it
// is tagged/published, resolve it from the sibling checkout.
replace github.com/0xmhha/accounts => ../accounts
