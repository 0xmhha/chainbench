// wireResult.ts — turns a WireCallResult into the {content, isError} response
// shape the MCP SDK expects. Pure / synchronous: no I/O, no spawn.
//
// Success: pretty-prints data (or "Done." when empty), then optional compact
// "Phases:" / "Events:" summary lines capped at MAX_NAMES_IN_SUMMARY entries
// with a "...+N" overflow suffix.
// Failure: "Error (CODE): message" plus an optional "Last phase: <step>" hint
// when the binary recorded any progress before failing.

import type { WireCallResult } from "./wire.js";

export interface FormattedToolResponse {
  content: Array<{ type: "text"; text: string; [k: string]: unknown }>;
  isError?: boolean;
  [k: string]: unknown;
}

const MAX_NAMES_IN_SUMMARY = 8;

export function formatWireResult(
  result: WireCallResult,
): FormattedToolResponse {
  const { result: r, events, progress } = result;

  if (r.ok === false) {
    const lines: string[] = [];
    lines.push(`Error (${r.error.code}): ${r.error.message || "(no message)"}`);
    if (progress.length > 0) {
      const last = progress[progress.length - 1];
      lines.push(`Last phase: ${last.step}`);
    }
    return {
      content: [{ type: "text", text: lines.join("\n") }],
      isError: true,
    };
  }

  // Success path.
  const dataKeys = Object.keys(r.data);
  const lines: string[] = [];
  if (dataKeys.length === 0) {
    lines.push("Done.");
  } else {
    lines.push(JSON.stringify(r.data, null, 2));
  }
  if (progress.length > 0) {
    const names = progress
      .slice(0, MAX_NAMES_IN_SUMMARY)
      .map((p) => p.step);
    const overflow =
      progress.length > MAX_NAMES_IN_SUMMARY
        ? ` ...+${progress.length - MAX_NAMES_IN_SUMMARY}`
        : "";
    lines.push(`Phases: [${progress.length}] ${names.join(", ")}${overflow}`);
  }
  if (events.length > 0) {
    const names = events.slice(0, MAX_NAMES_IN_SUMMARY).map((e) => e.name);
    const overflow =
      events.length > MAX_NAMES_IN_SUMMARY
        ? ` ...+${events.length - MAX_NAMES_IN_SUMMARY}`
        : "";
    lines.push(`Events: [${events.length}] ${names.join(", ")}${overflow}`);
  }
  return { content: [{ type: "text", text: lines.join("\n") }] };
}
