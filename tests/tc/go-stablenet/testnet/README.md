# tests/tc/go-stablenet/testnet — cases that attach to a network we did not compose

Every other case stands up the network it needs and tears it down afterwards.
These **attach to a go-stablenet network that is already running.** The premises
are different, so the assertions have to be written differently too.

## Running them

The address comes only from `GSTABLE_TESTNET_RPC`. **There is no default.**

```sh
export GSTABLE_TESTNET_RPC=http://<testnet endpoint>:8545
bin/chainbench run tests/tc/go-stablenet/testnet/02-block-advances.json
```

Without the variable the run stops and says why. A default — the shape
`stablenet-attached` uses — would let a forgotten variable attach to the local
network quietly, and the run would measure this machine while reporting it as
the testnet. That is why this preset has none.

Before attaching, the engine checks the endpoint really is go-stablenet: it
refuses unless `web3_clientVersion` carries the binary name the manifest records
(`gstable`). The chain id is not used for that decision — on a network we did
not compose, an id other than our built-in default is the ordinary case.

## What can be asserted, and what cannot

**Key-preset labels are not used.** This preset declares no `keysDir`. The
`node1`, `faucet` and `dev1` entries under `presets/keys` are addresses our own
genesis created, so on somebody else's network they do not exist and a label
would ask for the balance of an unrelated address. A label that does not resolve
stops the case there, which is better than quietly asking the wrong question.

The only label a case may use is one it declares itself. The funded account is
bound by `stablenet-testnet-funded` through `accounts.payer.keyFile`, and that
key is read from a file outside the repository that `GSTABLE_TESTNET_KEY_FILE`
points at. A read-only case needs neither, and uses `stablenet-testnet`.

**Nothing is compared against an absolute value.** Block height and balances
start from numbers we do not know, and other traffic is mixed in. "Exactly
1 ETH" holds only on a network we built.

**"Greater than zero" is not the way out either.** A block number cannot be
negative, so that assertion passes whatever the node answers, and a node stuck
at genesis still shows green. This repository removed several of those on
2026-09-29.

What works instead has this shape.

| What is measured | Why it holds on somebody else's network |
|---|---|
| It goes up from here (`blockAdvance`) | Change is knowable without the absolute value |
| The shape is right (hex, a 32-byte hash) | Independent of state |
| Self-consistency (the same block read back by its hash) | No outside traffic can get in between |
| The same within one run (chain id asked twice) | Shows a proxy with two chains behind it |
| The receipt and before/after difference of a transaction **we sent** | Only our own hash is read, so other traffic is irrelevant |

## What is here

| File | What it checks |
|---|---|
| `01-chain-identity` | clientVersion says Gstable; the chain id is hex and the same when asked twice |
| `02-block-advances` | Blocks keep coming |
| `03-block-fields-well-formed` | The head block's number, hash, parentHash and transactions |
| `04-block-by-hash-consistency` | A block four behind the head, read back by its hash, is the same block |
| `05-value-transfer` | 1 Gwei from payer to a fresh address: the receipt, the recipient's balance, and the payer's before/after difference |

The head block is avoided because it can be reorganised.

Only 05 uses `stablenet-testnet-funded`. The amount is 1 Gwei to spend as little
as possible of somebody's funds on a shared network. The recipient is an address
this run just created, so its starting balance is zero and an absolute value
holds. The payer's absolute balance is unknown, so only the before/after
difference is checked, against value plus `gasUsed × effectiveGasPrice`.

## The key file format

`keyFile` reads a file holding **raw hex**. A keystore JSON is refused because a
declaration has nowhere to put a password, and the refusal names the way out.

```sh
chainbench keyring new    --keyring-dir ~/.chainbench/keys/gstable-testnet --count 1 --validators 0
chainbench keyring export --keyring-dir ~/.chainbench/keys/gstable-testnet --name node1 --yes
```

Put the hex in a file **outside** the repository, `chmod 600`, and point the
variable at it. Funding that address is a person's job — we have no right to
call a faucet on somebody else's network.

## How far this is verified

All five passed **against the real go-stablenet testnet** on 2026-10-01. That
network answered as `Gstable/v1.1.0-stable-71e3f820`, chain id 8283, at a height
near 21.1M. 05 really did send 1 Gwei, and both the recipient's balance and the
payer's difference came out exact.

**One hazard when standing a local network in for the testnet.** Do not use
`node1`'s key as the payer. node1 produces blocks and earns rewards in the same
window, which breaks the difference assertion. Fund a fresh account on that
network and use its key. A testnet payer is not a producer, so the assertion
itself needs no change.

What is not known: how that network behaves under latency or a rate limit. Five
cases passed in a row; nothing here says what happens when it is busy.
