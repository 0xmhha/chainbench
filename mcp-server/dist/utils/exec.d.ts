export declare const CHAINBENCH_DIR: string;
export declare function buildEnv(extra?: Record<string, string>): Record<string, string | undefined>;
export declare function shellEscapeArg(arg: string): string;
export declare function runChainbench(args: string, options?: {
    cwd?: string;
}): {
    stdout: string;
    stderr: string;
    exitCode: number;
};
