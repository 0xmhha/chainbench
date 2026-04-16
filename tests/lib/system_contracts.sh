#!/usr/bin/env bash
# tests/lib/system_contracts.sh — go-stablenet system contract constants
# Source this file: source tests/lib/system_contracts.sh
#
# Contains addresses, function selectors, event topics, and account extra masks
# for go-stablenet system contracts. No external dependencies.

[[ -n "${_CB_SYSTEM_CONTRACTS_LOADED:-}" ]] && return 0
readonly _CB_SYSTEM_CONTRACTS_LOADED=1

# ---- System Contract Addresses ----
# From params/protocol_params.go and genesis config
readonly SC_NATIVE_COIN_ADAPTER="0x0000000000000000000000000000000000001000"
readonly SC_GOV_VALIDATOR="0x0000000000000000000000000000000000001001"
readonly SC_GOV_MASTER_MINTER="0x0000000000000000000000000000000000001002"
readonly SC_GOV_MINTER="0x0000000000000000000000000000000000001003"
readonly SC_GOV_COUNCIL="0x0000000000000000000000000000000000001004"

# ---- Precompile Addresses ----
# From params/protocol_params.go
readonly SC_BLS_POP_PRECOMPILE="0x0000000000000000000000000000000000B00001"
readonly SC_NATIVE_COIN_MANAGER="0x0000000000000000000000000000000000B00002"
readonly SC_ACCOUNT_MANAGER="0x0000000000000000000000000000000000B00003"

# ---- Misc Constants ----
readonly SC_ZERO_ADDRESS="0x0000000000000000000000000000000000000000"

# ---- Account Extra Bit Masks ----
# From core/types/state_account_extra.go
# Bit 63 = Blacklisted, Bit 62 = Authorized
readonly ACCOUNT_EXTRA_MASK_BLACKLISTED_HEX="0x8000000000000000"
readonly ACCOUNT_EXTRA_MASK_AUTHORIZED_HEX="0x4000000000000000"

# ---- Event Topics (keccak256 of event signature) ----
# Transfer(address indexed from, address indexed to, uint256 value)
readonly EVENT_TRANSFER="0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
# AuthorizedTxExecuted — from params/protocol_params.go
readonly EVENT_AUTHORIZED_TX_EXECUTED="0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373"
# Approval(address indexed owner, address indexed spender, uint256 value)
readonly EVENT_APPROVAL="0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"

# ---- Function Selectors (first 4 bytes of keccak256) ----

# NativeCoinAdapter — ERC-20 interface
readonly SEL_TRANSFER="0xa9059cbb"            # transfer(address,uint256)
readonly SEL_BALANCE_OF="0x70a08231"          # balanceOf(address)
readonly SEL_APPROVE="0x095ea7b3"             # approve(address,uint256)
readonly SEL_TRANSFER_FROM="0x23b872dd"       # transferFrom(address,address,uint256)
readonly SEL_ALLOWANCE="0xdd62ed3e"           # allowance(address,address)
readonly SEL_TOTAL_SUPPLY="0x18160ddd"        # totalSupply()
readonly SEL_NAME="0x06fdde03"               # name()
readonly SEL_SYMBOL="0x95d89b41"             # symbol()
readonly SEL_DECIMALS="0x313ce567"           # decimals()

# Governance — common proposal lifecycle
readonly SEL_APPROVE_PROPOSAL="0x98951b56"    # approveProposal(uint256)
readonly SEL_EXECUTE_PROPOSAL="0x0d61b519"    # executeProposal(uint256)
readonly SEL_CANCEL_PROPOSAL="0xe0a8f6f5"     # cancelProposal(uint256)
readonly SEL_EXPIRE_PROPOSAL="0xe1b526b0"     # expireProposal(uint256)
readonly SEL_PROPOSALS="0x013cf08b"           # proposals(uint256)
readonly SEL_QUORUM="0x1703a018"              # quorum()
readonly SEL_EXPIRY_BLOCKS="0x669a1c1e"       # expiryBlocks()

# GovValidator — member management
readonly SEL_PROPOSE_ADD_MEMBER="0x5c646aa6"         # proposeAddMember(address,uint32)
readonly SEL_PROPOSE_REMOVE_MEMBER="0xbfbd7f4c"      # proposeRemoveMember(address,uint32)
readonly SEL_IS_ACTIVE_MEMBER="0x45ecd02f"           # isActiveMember(address)
readonly SEL_MEMBER_LIST="0xf2ad35d5"                # memberList()
readonly SEL_MEMBERS="0x08ae4b0c"                    # members(address)
readonly SEL_VALIDATOR_LIST="0x5890ef79"             # validatorList()
readonly SEL_VALIDATOR_TO_OPERATOR="0x35efc734"      # validatorToOperator(address)
readonly SEL_VALIDATOR_TO_BLS_KEY="0x70a78608"       # validatorToBlsKey(address)
readonly SEL_PROPOSE_GAS_TIP="0xeeaf6816"            # proposeGasTip(uint256)
readonly SEL_GET_GAS_TIP_GWEI="0x040bba71"           # getGasTipGwei()

# GovMinter — minter management
readonly SEL_PROPOSE_CONFIGURE_MINTER="0x898420a9"  # proposeConfigureMinter(address,uint256)
readonly SEL_PROPOSE_REMOVE_MINTER="0x93364117"     # proposeRemoveMinter(address)
readonly SEL_IS_MINTER="0xaa271e1a"                 # isMinter(address)
readonly SEL_MINTER_ALLOWANCE="0x8a6db9c3"          # minterAllowance(address)

# GovCouncil / AccountManager — blacklist & authorized account management
readonly SEL_PROPOSE_ADD_BLACKLIST="0x0d321273"          # proposeAddBlacklist(address)
readonly SEL_PROPOSE_REMOVE_BLACKLIST="0x3d4c0452"       # proposeRemoveBlacklist(address)
readonly SEL_PROPOSE_ADD_AUTHORIZED="0x93a8bb99"         # proposeAddAuthorizedAccount(address)
readonly SEL_PROPOSE_REMOVE_AUTHORIZED="0xcf44550e"      # proposeRemoveAuthorizedAccount(address)
readonly SEL_IS_BLACKLISTED="0xfe575a87"                 # isBlacklisted(address)
readonly SEL_IS_AUTHORIZED="0xfe9fbb80"                  # isAuthorized(address)
readonly SEL_BLACKLIST="0xf9f92be4"                      # blacklist(address)

# NativeCoinManager (precompile)
readonly SEL_PROPOSE_MINT="0x1e5e0426"    # proposeMint(bytes)
readonly SEL_PROPOSE_BURN="0x64e2a8fc"    # proposeBurn(bytes)
readonly SEL_BURN_BALANCE="0x98179c41"    # burnBalance(address)
readonly SEL_REFUNDABLE_BALANCE="0xb03d36cd"  # refundableBalance(address)
readonly SEL_CLAIM_BURN_REFUND="0x936834b9"   # claimBurnRefund()

# ---- Contract Name Constants ----
# Matching go-stablenet systemcontracts/contracts.go
readonly SC_NAME_NATIVE_COIN_ADAPTER="NativeCoinAdapter"
readonly SC_NAME_GOV_VALIDATOR="GovValidator"
readonly SC_NAME_GOV_MASTER_MINTER="GovMasterMinter"
readonly SC_NAME_GOV_MINTER="GovMinter"
readonly SC_NAME_GOV_COUNCIL="GovCouncil"
readonly SC_NAME_NATIVE_COIN_MANAGER="NativeCoinManager"
readonly SC_NAME_ACCOUNT_MANAGER="AccountManager"
readonly SC_NAME_BLS_POP="BlsPopPrecompile"
