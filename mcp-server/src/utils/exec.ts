import { execSync } from "child_process";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

// CHAINBENCH_DIR: prefer env var (set by chainbench-mcp wrapper),
// fall back to relative path from compiled dist location.
export const CHAINBENCH_DIR = process.env.CHAINBENCH_DIR || resolve(__dirname, "../../..");

// MCP clients typically spawn the server process with cwd set to the
// project root (the directory containing .mcp.json). Capturing this once
// at startup lets resolve_binary() find <project>/build/bin/<binary>.
const MCP_CLIENT_CWD = process.cwd();

// Augment PATH for non-interactive shell contexts (MCP server, IDE, etc.)
// where the user's login profile may not be sourced.
function buildPath(): string {
  const base = process.env.PATH || "";
  const extras = ["/usr/local/bin", "/opt/homebrew/bin", "/usr/bin", "/bin"];
  const parts = base.split(":");
  const seen = new Set(parts);
  const missing = extras.filter((p) => !seen.has(p));
  return missing.length > 0 ? `${base}:${missing.join(":")}` : base;
}

export function buildEnv(extra?: Record<string, string>): Record<string, string | undefined> {
  return {
    ...process.env,
    PATH: buildPath(),
    CHAINBENCH_DIR,
    ...extra,
  };
}

// NOTE: args must be pre-validated by callers before being passed here.
// This utility is intended for internal CLI invocation only, not for
// passing raw user input. Callers are responsible for sanitizing args.
export function runChainbench(
  args: string,
  options?: { cwd?: string },
): { stdout: string; stderr: string; exitCode: number } {
  const cmd = `${CHAINBENCH_DIR}/chainbench.sh ${args}`;
  const cwd = options?.cwd || MCP_CLIENT_CWD;
  try {
    const stdout = execSync(cmd, {
      cwd,
      encoding: "utf-8",
      timeout: 120000,
      env: buildEnv(),
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
