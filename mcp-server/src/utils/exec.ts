import { execSync } from "child_process";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const CHAINBENCH_DIR = resolve(__dirname, "../../..");

// NOTE: args must be pre-validated by callers before being passed here.
// This utility is intended for internal CLI invocation only, not for
// passing raw user input. Callers are responsible for sanitizing args.
export function runChainbench(args: string): { stdout: string; stderr: string; exitCode: number } {
  const cmd = `${CHAINBENCH_DIR}/chainbench.sh ${args}`;
  try {
    const stdout = execSync(cmd, {
      cwd: CHAINBENCH_DIR,
      encoding: "utf-8",
      timeout: 120000,
      env: { ...process.env, CHAINBENCH_DIR },
    });
    return { stdout: stdout.trim(), stderr: "", exitCode: 0 };
  } catch (error: any) {
    return {
      stdout: error.stdout?.toString().trim() ?? "",
      stderr: error.stderr?.toString().trim() ?? "",
      exitCode: error.status ?? 1,
    };
  }
}
