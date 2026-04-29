// hex.ts — shared hex/identifier regex constants for MCP tool argument schemas.
//
// Single source of truth for the 0x-prefixed hex shapes chainbench-net's wire
// schema accepts plus the signer-alias and rpc-method identifier shapes.
// Keeping these here means the MCP boundary rejects malformed inputs with a
// Zod parse error before a wire spawn rather than surfacing them as
// chainbench-net UPSTREAM_ERROR / INVALID_ARGS after the wasted spawn.
//
// HEX_TX_HASH and HEX_TOPIC happen to share the same byte shape (0x + 64 hex
// chars) but stay as separate constants for intent clarity at call sites.
// HEX_HEX accepts arbitrary-length 0x-hex (not byte-aligned) — used for
// authorization-list chain_id and nonce, which are RLP-encoded big.Ints on
// the Go side and may be a single nibble (e.g. "0x0" for "any chain").

export const HEX_ADDRESS = /^0x[a-fA-F0-9]{40}$/;
export const HEX_DATA = /^0x([a-fA-F0-9]{2})*$/;
export const HEX_TX_HASH = /^0x[a-fA-F0-9]{64}$/;
export const HEX_TOPIC = /^0x[a-fA-F0-9]{64}$/;
export const HEX_HEX = /^0x[a-fA-F0-9]+$/;
export const HEX_STORAGE_KEY = /^0x[a-fA-F0-9]{1,64}$/;
export const SIGNER_ALIAS = /^[A-Za-z][A-Za-z0-9_]*$/;
export const RPC_METHOD = /^[a-zA-Z][a-zA-Z0-9_]*$/;
