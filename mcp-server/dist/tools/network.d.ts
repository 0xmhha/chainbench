/**
 * tools/network.ts - P2P network management and txpool inspection MCP tools.
 *
 * Provides network partition/heal, peer topology, and txpool status
 * across all nodes via admin_* and txpool_* RPC methods. Sprint 5a adds
 * chainbench_network_capabilities, which wraps the chainbench-net
 * network.capabilities wire command (provider-derived capability set).
 */
import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { type FormattedToolResponse } from "../utils/mcpResp.js";
export declare const NetworkCapabilitiesArgs: z.ZodObject<{
    network: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    network?: string | undefined;
}, {
    network?: string | undefined;
}>;
type NetworkCapabilitiesArgsT = z.infer<typeof NetworkCapabilitiesArgs>;
export declare function _networkCapabilitiesHandler(args: NetworkCapabilitiesArgsT): Promise<FormattedToolResponse>;
export declare const NetworkAttachArgs: z.ZodObject<{
    name: z.ZodString;
    rpc_url: z.ZodString;
    override: z.ZodOptional<z.ZodString>;
    provider: z.ZodOptional<z.ZodEnum<["remote", "ssh-remote"]>>;
    auth: z.ZodOptional<z.ZodObject<{
        type: z.ZodEnum<["api-key", "jwt", "ssh-password"]>;
        env: z.ZodOptional<z.ZodString>;
        header: z.ZodOptional<z.ZodString>;
        user: z.ZodOptional<z.ZodString>;
        host: z.ZodOptional<z.ZodString>;
        port: z.ZodOptional<z.ZodNumber>;
    }, "strict", z.ZodTypeAny, {
        type: "jwt" | "api-key" | "ssh-password";
        env?: string | undefined;
        user?: string | undefined;
        port?: number | undefined;
        host?: string | undefined;
        header?: string | undefined;
    }, {
        type: "jwt" | "api-key" | "ssh-password";
        env?: string | undefined;
        user?: string | undefined;
        port?: number | undefined;
        host?: string | undefined;
        header?: string | undefined;
    }>>;
    provider_meta: z.ZodOptional<z.ZodObject<{
        log_file: z.ZodOptional<z.ZodString>;
        start_cmd: z.ZodOptional<z.ZodString>;
        stop_cmd: z.ZodOptional<z.ZodString>;
        restart_cmd: z.ZodOptional<z.ZodString>;
    }, "strict", z.ZodTypeAny, {
        log_file?: string | undefined;
        start_cmd?: string | undefined;
        stop_cmd?: string | undefined;
        restart_cmd?: string | undefined;
    }, {
        log_file?: string | undefined;
        start_cmd?: string | undefined;
        stop_cmd?: string | undefined;
        restart_cmd?: string | undefined;
    }>>;
}, "strict", z.ZodTypeAny, {
    name: string;
    rpc_url: string;
    override?: string | undefined;
    provider?: "remote" | "ssh-remote" | undefined;
    auth?: {
        type: "jwt" | "api-key" | "ssh-password";
        env?: string | undefined;
        user?: string | undefined;
        port?: number | undefined;
        host?: string | undefined;
        header?: string | undefined;
    } | undefined;
    provider_meta?: {
        log_file?: string | undefined;
        start_cmd?: string | undefined;
        stop_cmd?: string | undefined;
        restart_cmd?: string | undefined;
    } | undefined;
}, {
    name: string;
    rpc_url: string;
    override?: string | undefined;
    provider?: "remote" | "ssh-remote" | undefined;
    auth?: {
        type: "jwt" | "api-key" | "ssh-password";
        env?: string | undefined;
        user?: string | undefined;
        port?: number | undefined;
        host?: string | undefined;
        header?: string | undefined;
    } | undefined;
    provider_meta?: {
        log_file?: string | undefined;
        start_cmd?: string | undefined;
        stop_cmd?: string | undefined;
        restart_cmd?: string | undefined;
    } | undefined;
}>;
type NetworkAttachArgsT = z.infer<typeof NetworkAttachArgs>;
export declare function _networkAttachHandler(args: NetworkAttachArgsT): Promise<FormattedToolResponse>;
export declare const NetworkListArgs: z.ZodObject<{}, "strict", z.ZodTypeAny, {}, {}>;
export declare function _networkListHandler(_args: z.infer<typeof NetworkListArgs>): Promise<FormattedToolResponse>;
export declare const NetworkDetachArgs: z.ZodObject<{
    name: z.ZodString;
}, "strict", z.ZodTypeAny, {
    name: string;
}, {
    name: string;
}>;
export declare function _networkDetachHandler(args: z.infer<typeof NetworkDetachArgs>): Promise<FormattedToolResponse>;
export declare function registerNetworkTools(server: McpServer): void;
export {};
