// wire.ts — TypeScript counterpart of lib/network_client.sh.
//
// Spawns `chainbench-net run`, writes a single-line NDJSON envelope on stdin,
// and accumulates events / progress / result lines from stdout via readline.
// Binary resolution mirrors the bash client exactly: env first, then
// $CHAINBENCH_DIR/bin, then $CHAINBENCH_DIR/network/bin, finally a shell-free
// PATH split — no `which`, no shell exec.
import { existsSync } from "node:fs";
import { spawn } from "node:child_process";
import { resolve as resolvePath, join } from "node:path";
import readline from "node:readline";
const PATH_SEP = process.platform === "win32" ? ";" : ":";
const BINARY_NAME = "chainbench-net";
const DEFAULT_TIMEOUT_MS = 120000;
export function resolveBinary() {
    const fromEnv = process.env.CHAINBENCH_NET_BIN;
    if (fromEnv && existsSync(fromEnv))
        return fromEnv;
    const dir = process.env.CHAINBENCH_DIR;
    if (dir) {
        const a = resolvePath(dir, "bin", BINARY_NAME);
        if (existsSync(a))
            return a;
        const b = resolvePath(dir, "network", "bin", BINARY_NAME);
        if (existsSync(b))
            return b;
    }
    // PATH lookup — split process.env.PATH and check each entry; no shell exec.
    const pathEntries = (process.env.PATH ?? "").split(PATH_SEP).filter(Boolean);
    for (const entry of pathEntries) {
        const candidate = join(entry, BINARY_NAME);
        if (existsSync(candidate))
            return candidate;
    }
    throw new Error("chainbench-net binary not found");
}
export async function callWire(command, args, options = {}) {
    const bin = options.binaryPath ?? resolveBinary();
    const env = { ...process.env, ...(options.envOverrides ?? {}) };
    const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
    const child = spawn(bin, ["run"], {
        env,
        stdio: ["pipe", "pipe", "pipe"],
    });
    const events = [];
    const progress = [];
    let result = null;
    let stderrBuf = "";
    const rl = readline.createInterface({ input: child.stdout });
    rl.on("line", (line) => {
        const trimmed = line.trim();
        if (!trimmed.startsWith("{"))
            return;
        let parsed;
        try {
            parsed = JSON.parse(trimmed);
        }
        catch {
            return; // Defensive: chainbench-net contracts NDJSON only, but skip malformed.
        }
        const obj = parsed;
        if (obj.type === "event") {
            events.push(obj);
        }
        else if (obj.type === "progress") {
            progress.push(obj);
        }
        else if (obj.type === "result" && result === null) {
            // First result line wins; later result lines (none should exist by
            // contract) are ignored.
            result = obj;
        }
    });
    child.stderr.on("data", (b) => {
        stderrBuf += b.toString("utf-8");
    });
    // Envelope write.
    const envelope = JSON.stringify({ command, args }) + "\n";
    child.stdin.write(envelope);
    child.stdin.end();
    let timer = null;
    let timedOut = false;
    try {
        const exitCode = await new Promise((resolveExit, rejectExit) => {
            const onExit = (code, signal) => {
                if (timer) {
                    clearTimeout(timer);
                    timer = null;
                }
                if (timedOut) {
                    rejectExit(new Error(`chainbench-net timeout after ${timeoutMs}ms`));
                    return;
                }
                resolveExit(code ?? (signal ? 128 : 0));
            };
            child.once("error", (err) => {
                if (timer) {
                    clearTimeout(timer);
                    timer = null;
                }
                rejectExit(err);
            });
            child.once("close", onExit);
            timer = setTimeout(() => {
                timedOut = true;
                child.kill("SIGTERM");
            }, timeoutMs);
        });
        // readline may still hold a partial line buffered after stdout close;
        // wait one microtask for any trailing 'line' events to fire.
        await new Promise((r) => setImmediate(r));
        if (result === null) {
            throw new Error(`chainbench-net produced no result terminator (exit ${exitCode}). stderr: ${stderrBuf.slice(0, 500)}`);
        }
        return { result, events, progress, stderr: stderrBuf, exitCode };
    }
    finally {
        if (timer)
            clearTimeout(timer);
        rl.close();
    }
}
