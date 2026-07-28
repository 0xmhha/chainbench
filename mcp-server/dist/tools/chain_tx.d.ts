import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { type FormattedToolResponse } from "../utils/mcpResp.js";
export declare const TxSendArgs: z.ZodObject<{
    network: z.ZodString;
    node_id: z.ZodOptional<z.ZodString>;
    signer: z.ZodString;
    mode: z.ZodEnum<["legacy", "1559", "set_code", "fee_delegation"]>;
    to: z.ZodOptional<z.ZodString>;
    value: z.ZodOptional<z.ZodString>;
    data: z.ZodOptional<z.ZodString>;
    gas: z.ZodOptional<z.ZodUnion<[z.ZodString, z.ZodNumber]>>;
    nonce: z.ZodOptional<z.ZodUnion<[z.ZodString, z.ZodNumber]>>;
    gas_price: z.ZodOptional<z.ZodString>;
    max_fee_per_gas: z.ZodOptional<z.ZodString>;
    max_priority_fee_per_gas: z.ZodOptional<z.ZodString>;
    authorization_list: z.ZodOptional<z.ZodArray<z.ZodObject<{
        chain_id: z.ZodString;
        address: z.ZodString;
        nonce: z.ZodString;
        signer: z.ZodString;
    }, "strict", z.ZodTypeAny, {
        nonce: string;
        address: string;
        chain_id: string;
        signer: string;
    }, {
        nonce: string;
        address: string;
        chain_id: string;
        signer: string;
    }>, "many">>;
    fee_payer: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    network: string;
    signer: string;
    mode: "legacy" | "1559" | "set_code" | "fee_delegation";
    data?: string | undefined;
    value?: string | undefined;
    node_id?: string | undefined;
    nonce?: string | number | undefined;
    to?: string | undefined;
    gas?: string | number | undefined;
    gas_price?: string | undefined;
    max_fee_per_gas?: string | undefined;
    max_priority_fee_per_gas?: string | undefined;
    authorization_list?: {
        nonce: string;
        address: string;
        chain_id: string;
        signer: string;
    }[] | undefined;
    fee_payer?: string | undefined;
}, {
    network: string;
    signer: string;
    mode: "legacy" | "1559" | "set_code" | "fee_delegation";
    data?: string | undefined;
    value?: string | undefined;
    node_id?: string | undefined;
    nonce?: string | number | undefined;
    to?: string | undefined;
    gas?: string | number | undefined;
    gas_price?: string | undefined;
    max_fee_per_gas?: string | undefined;
    max_priority_fee_per_gas?: string | undefined;
    authorization_list?: {
        nonce: string;
        address: string;
        chain_id: string;
        signer: string;
    }[] | undefined;
    fee_payer?: string | undefined;
}>;
type TxSendArgsT = z.infer<typeof TxSendArgs>;
export declare function _buildTxSendWireArgs(args: TxSendArgsT): {
    wireCommand: "node.tx_send" | "node.tx_fee_delegation_send";
    wireArgs: Record<string, unknown>;
} | {
    error: string;
};
export declare function _txSendHandler(args: TxSendArgsT): Promise<FormattedToolResponse>;
export declare const ContractDeployArgs: z.ZodObject<{
    network: z.ZodString;
    node_id: z.ZodOptional<z.ZodString>;
    signer: z.ZodString;
    mode: z.ZodEnum<["legacy", "1559"]>;
    bytecode: z.ZodString;
    abi: z.ZodOptional<z.ZodString>;
    constructor_args: z.ZodOptional<z.ZodArray<z.ZodUnknown, "many">>;
    value: z.ZodOptional<z.ZodString>;
    gas: z.ZodOptional<z.ZodUnion<[z.ZodString, z.ZodNumber]>>;
    nonce: z.ZodOptional<z.ZodUnion<[z.ZodString, z.ZodNumber]>>;
    gas_price: z.ZodOptional<z.ZodString>;
    max_fee_per_gas: z.ZodOptional<z.ZodString>;
    max_priority_fee_per_gas: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    network: string;
    signer: string;
    mode: "legacy" | "1559";
    bytecode: string;
    value?: string | undefined;
    node_id?: string | undefined;
    nonce?: string | number | undefined;
    abi?: string | undefined;
    gas?: string | number | undefined;
    gas_price?: string | undefined;
    max_fee_per_gas?: string | undefined;
    max_priority_fee_per_gas?: string | undefined;
    constructor_args?: unknown[] | undefined;
}, {
    network: string;
    signer: string;
    mode: "legacy" | "1559";
    bytecode: string;
    value?: string | undefined;
    node_id?: string | undefined;
    nonce?: string | number | undefined;
    abi?: string | undefined;
    gas?: string | number | undefined;
    gas_price?: string | undefined;
    max_fee_per_gas?: string | undefined;
    max_priority_fee_per_gas?: string | undefined;
    constructor_args?: unknown[] | undefined;
}>;
type ContractDeployArgsT = z.infer<typeof ContractDeployArgs>;
export declare function _buildContractDeployWireArgs(args: ContractDeployArgsT): {
    wireArgs: Record<string, unknown>;
} | {
    error: string;
};
export declare function _contractDeployHandler(args: ContractDeployArgsT): Promise<FormattedToolResponse>;
export declare function registerChainTxTools(server: McpServer): void;
export {};
