import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { type FormattedToolResponse } from "../utils/mcpResp.js";
export declare const StopArgs: z.ZodObject<{
    network: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    network?: string | undefined;
}, {
    network?: string | undefined;
}>;
export declare function _stopHandler(args: z.infer<typeof StopArgs>): Promise<FormattedToolResponse>;
export declare const StatusArgs: z.ZodObject<{
    network: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    network?: string | undefined;
}, {
    network?: string | undefined;
}>;
export declare function _statusHandler(args: z.infer<typeof StatusArgs>): Promise<FormattedToolResponse>;
export declare const InitArgs: z.ZodObject<{
    profile: z.ZodDefault<z.ZodString>;
    project_root: z.ZodOptional<z.ZodString>;
    binary_path: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    profile: string;
    project_root?: string | undefined;
    binary_path?: string | undefined;
}, {
    profile?: string | undefined;
    project_root?: string | undefined;
    binary_path?: string | undefined;
}>;
export declare function _initHandler(args: z.infer<typeof InitArgs>): Promise<FormattedToolResponse>;
export declare const StartArgs: z.ZodObject<{
    project_root: z.ZodOptional<z.ZodString>;
    binary_path: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    project_root?: string | undefined;
    binary_path?: string | undefined;
}, {
    project_root?: string | undefined;
    binary_path?: string | undefined;
}>;
export declare function _startHandler(args: z.infer<typeof StartArgs>): Promise<FormattedToolResponse>;
export declare const RestartArgs: z.ZodObject<{
    project_root: z.ZodOptional<z.ZodString>;
    binary_path: z.ZodOptional<z.ZodString>;
}, "strict", z.ZodTypeAny, {
    project_root?: string | undefined;
    binary_path?: string | undefined;
}, {
    project_root?: string | undefined;
    binary_path?: string | undefined;
}>;
export declare function _restartHandler(args: z.infer<typeof RestartArgs>): Promise<FormattedToolResponse>;
export declare const CleanArgs: z.ZodObject<{}, "strict", z.ZodTypeAny, {}, {}>;
export declare function _cleanHandler(_args: z.infer<typeof CleanArgs>): Promise<FormattedToolResponse>;
export declare function registerLifecycleTools(server: McpServer): void;
