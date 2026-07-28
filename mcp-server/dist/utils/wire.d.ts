export type WireEventLine = {
    type: "event";
    name: string;
    data?: Record<string, unknown>;
    ts?: string;
};
export type WireProgressLine = {
    type: "progress";
    step: string;
    done?: number;
    total?: number;
};
export type WireResultLine = {
    type: "result";
    ok: true;
    data: Record<string, unknown>;
} | {
    type: "result";
    ok: false;
    error: {
        code: string;
        message: string;
        details?: unknown;
    };
};
export type WireStreamLine = WireResultLine | WireEventLine | WireProgressLine;
export interface WireCallResult {
    result: WireResultLine;
    events: WireEventLine[];
    progress: WireProgressLine[];
    stderr: string;
    exitCode: number;
}
export interface WireCallOptions {
    envOverrides?: Record<string, string>;
    timeoutMs?: number;
    binaryPath?: string;
}
export declare function resolveBinary(): string;
export declare function callWire(command: string, args: Record<string, unknown>, options?: WireCallOptions): Promise<WireCallResult>;
