# Hardfork presets

A named bundle of the facts a hardfork test needs, in the same sense as the key
preset under `presets/keys` and a chain's manifest: a declaration points at one
and overrides what it needs to.

A case names one with `env.upgrade.preset`, which resolves to
`presets/chain/<id>.yaml`. An explicit path goes in `env.upgrade.profile`
instead, for a file that lives somewhere else.

These used to sit in `profiles/`, beside the profiles for connecting to a
remote chain. The two are not the same kind of thing, and the directory said
they were.

## What a preset holds

The fork's name and block, the devp2p network id, how many nodes seal on each
side, and which binary each side runs.

**Less than it looks.** Three of the blocks these files carry restate facts
rather than choose anything, and each has a test saying so: the validator set
is what the key set derives (`profile_derivable_test.go`), the governance
policy is `poa.DefaultEnv` field for field (`profile_governance_test.go`), and
`plan_order` exists because a handoff plan puts its producers first while the
preset's producer-capable entry is last (`profile_planorder_test.go`).
