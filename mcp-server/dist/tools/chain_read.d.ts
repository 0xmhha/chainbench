import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { type FormattedToolResponse } from "../utils/mcpResp.js";
export declare const AccountStateArgs: z.ZodObject<{
    network: z.ZodString;
    node_id: z.ZodOptional<z.ZodString>;
    address: z.ZodString;
    fields: z.ZodOptional<z.ZodArray<z.ZodEnum<["balance", "nonce", "code", "storage"]>, "many">>;
    storage_key: z.ZodOptional<z.ZodString>;
    block_number: z.ZodOptional<z.ZodUnion<[z.ZodString, z.ZodNumber]>>;
}, "strict", z.ZodTypeAny, {
    network: string;
    address: string;
    node_id?: string | undefined;
    fields?: ("code" | "balance" | "nonce" | "storage")[] | undefined;
    storage_key?: string | undefined;
    block_number?: string | number | undefined;
}, {
    network: string;
    address: string;
    node_id?: string | undefined;
    fields?: ("code" | "balance" | "nonce" | "storage")[] | undefined;
    storage_key?: string | undefined;
    block_number?: string | number | undefined;
}>;
type AccountStateArgsT = z.infer<typeof AccountStateArgs>;
export declare function _accountStateHandler(args: AccountStateArgsT): Promise<FormattedToolResponse>;
export declare const ContractCallArgs: z.ZodObject<{
    network: z.ZodString;
    node_id: z.ZodOptional<z.ZodString>;
    contract_address: z.ZodString;
    calldata: z.ZodOptional<z.ZodString>;
    abi: z.ZodOptional<z.ZodString>;
    method: z.ZodOptional<z.ZodString>;
    args: z.ZodOptional<z.ZodArray<z.ZodUnknown, "many">>;
    block_number: z.ZodOptional<z.ZodUnion<[z.ZodString, z.ZodNumber]>>;
    from: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    network: string;
    contract_address: string;
    method?: string | undefined;
    node_id?: string | undefined;
    block_number?: string | number | undefined;
    calldata?: string | undefined;
    abi?: string | undefined;
    args?: unknown[] | undefined;
    from?: string | undefined;
}, {
    network: string;
    contract_address: string;
    method?: string | undefined;
    node_id?: string | undefined;
    block_number?: string | number | undefined;
    calldata?: string | undefined;
    abi?: string | undefined;
    args?: unknown[] | undefined;
    from?: string | undefined;
}>;
type ContractCallArgsT = z.infer<typeof ContractCallArgs>;
export declare function _contractCallHandler(args: ContractCallArgsT): Promise<FormattedToolResponse>;
export declare const EventsGetArgs: z.ZodObject<{
    network: z.ZodString;
    node_id: z.ZodOptional<z.ZodString>;
    address: z.ZodOptional<z.ZodString>;
    from_block: z.ZodOptional<z.ZodUnion<[z.ZodString, z.ZodNumber]>>;
    to_block: z.ZodOptional<z.ZodUnion<[z.ZodString, z.ZodNumber]>>;
    topics: z.ZodOptional<z.ZodArray<z.ZodUnion<[z.ZodString, z.ZodArray<z.ZodString, "many">, z.ZodNull]>, "many">>;
    abi: z.ZodOptional<z.ZodString>;
    event: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    network: string;
    event?: string | undefined;
    node_id?: string | undefined;
    address?: string | undefined;
    abi?: string | undefined;
    from_block?: string | number | undefined;
    to_block?: string | number | undefined;
    topics?: (string | string[] | null)[] | undefined;
}, {
    network: string;
    event?: string | undefined;
    node_id?: string | undefined;
    address?: string | undefined;
    abi?: string | undefined;
    from_block?: string | number | undefined;
    to_block?: string | number | undefined;
    topics?: (string | string[] | null)[] | undefined;
}>;
type EventsGetArgsT = z.infer<typeof EventsGetArgs>;
export declare function _eventsGetHandler(args: EventsGetArgsT): Promise<FormattedToolResponse>;
export declare const TxWaitArgs: z.ZodObject<{
    network: z.ZodString;
    node_id: z.ZodOptional<z.ZodString>;
    tx_hash: z.ZodString;
    timeout_ms: z.ZodOptional<z.ZodNumber>;
}, "strict", z.ZodTypeAny, {
    network: string;
    tx_hash: string;
    node_id?: string | undefined;
    timeout_ms?: number | undefined;
}, {
    network: string;
    tx_hash: string;
    node_id?: string | undefined;
    timeout_ms?: number | undefined;
}>;
type TxWaitArgsT = z.infer<typeof TxWaitArgs>;
export declare function _txWaitHandler(args: TxWaitArgsT): Promise<FormattedToolResponse>;
export declare function registerChainReadTools(server: McpServer): void;
export {};
