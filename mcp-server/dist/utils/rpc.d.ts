interface NodeInfo {
    http_port: number;
    status: string;
    p2p_port?: number;
    [key: string]: unknown;
}
interface PidsState {
    nodes: Record<string, NodeInfo>;
}
export declare function loadPidsState(): PidsState;
export declare function getRunningNodeIds(): string[];
export declare function getNodePort(nodeId: string): number;
export declare function rpcCall(nodeId: string, method: string, params?: unknown[]): Promise<unknown>;
export declare function rpcCallAll(method: string, params?: unknown[], nodeIds?: string[]): Promise<Record<string, unknown>>;
export declare function toHex(n: number): string;
export declare function fromHex(hex: string): number;
export {};
