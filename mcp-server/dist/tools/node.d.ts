import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { type FormattedToolResponse } from "../utils/mcpResp.js";
export declare const NodeStopArgs: z.ZodObject<{
    node: z.ZodNumber;
}, "strict", z.ZodTypeAny, {
    node: number;
}, {
    node: number;
}>;
type NodeStopArgsT = z.infer<typeof NodeStopArgs>;
export declare function _nodeStopHandler(args: NodeStopArgsT): Promise<FormattedToolResponse>;
export declare const NodeStartArgs: z.ZodObject<{
    node: z.ZodNumber;
    binary_path: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    node: number;
    binary_path?: string | undefined;
}, {
    node: number;
    binary_path?: string | undefined;
}>;
type NodeStartArgsT = z.infer<typeof NodeStartArgs>;
export declare function _nodeStartHandler(args: NodeStartArgsT): Promise<FormattedToolResponse>;
export declare const NodeRpcArgs: z.ZodObject<{
    node: z.ZodNumber;
    method: z.ZodString;
    params: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    method: string;
    node: number;
    params?: string | undefined;
}, {
    method: string;
    node: number;
    params?: string | undefined;
}>;
type NodeRpcArgsT = z.infer<typeof NodeRpcArgs>;
export declare function _nodeRpcHandler(args: NodeRpcArgsT): Promise<FormattedToolResponse>;
export declare function registerNodeTools(server: McpServer): void;
export {};
