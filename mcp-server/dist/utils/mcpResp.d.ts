export interface FormattedToolResponse {
    content: Array<{
        type: "text";
        text: string;
        [k: string]: unknown;
    }>;
    isError?: boolean;
    [k: string]: unknown;
}
export declare function errorResp(msg: string): FormattedToolResponse;
export declare function textResult(text: string): FormattedToolResponse;
export declare function formatExecResult(result: {
    stdout: string;
    stderr: string;
    exitCode: number;
}, emptyFallback?: string): string;
