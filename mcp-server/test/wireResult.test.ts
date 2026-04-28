import { describe, it, expect } from "vitest";
import { formatWireResult } from "../src/utils/wireResult.js";
import type {
  WireCallResult,
  WireEventLine,
  WireProgressLine,
} from "../src/utils/wire.js";

const baseSuccess = (data: Record<string, unknown>): WireCallResult => ({
  result: { type: "result", ok: true, data },
  events: [],
  progress: [],
  stderr: "",
  exitCode: 0,
});

const baseFailure = (
  code: string,
  message: string,
  progress: WireProgressLine[] = [],
): WireCallResult => ({
  result: { type: "result", ok: false, error: { code, message } },
  events: [],
  progress,
  stderr: "",
  exitCode: 0,
});

describe("formatWireResult", () => {
  it("_Success_RendersDataPretty", () => {
    const out = formatWireResult(baseSuccess({ balance: "0x10" }));
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    expect(out.content[0].type).toBe("text");
    // Pretty-printed JSON of the data field
    expect(out.content[0].text).toBe(
      JSON.stringify({ balance: "0x10" }, null, 2),
    );
  });

  it("_Success_EmptyData_RendersDoneText", () => {
    const out = formatWireResult(baseSuccess({}));
    expect(out.isError).toBeFalsy();
    expect(out.content[0].text).toBe("Done.");
  });

  it("_Success_WithPhasesSummary", () => {
    const progress: WireProgressLine[] = [
      { type: "progress", step: "connect" },
      { type: "progress", step: "build_tx" },
      { type: "progress", step: "send" },
    ];
    const events: WireEventLine[] = [
      { type: "event", name: "tx.sent" },
      { type: "event", name: "tx.included" },
    ];
    const out = formatWireResult({
      result: { type: "result", ok: true, data: { hash: "0xabc" } },
      events,
      progress,
      stderr: "",
      exitCode: 0,
    });
    expect(out.isError).toBeFalsy();
    const text = out.content[0].text;
    expect(text).toContain(JSON.stringify({ hash: "0xabc" }, null, 2));
    expect(text).toContain("Phases: [3] connect, build_tx, send");
    expect(text).toContain("Events: [2] tx.sent, tx.included");
  });

  it("_Failure_RendersErrorWithCode", () => {
    const out = formatWireResult(
      baseFailure("INVALID_ARGS", "address must be 0x-prefixed"),
    );
    expect(out.isError).toBe(true);
    expect(out.content[0].text).toBe(
      "Error (INVALID_ARGS): address must be 0x-prefixed",
    );
  });

  it("_Failure_WithLastPhaseHint", () => {
    const progress: WireProgressLine[] = [
      { type: "progress", step: "connect" },
      { type: "progress", step: "estimate_gas" },
      { type: "progress", step: "broadcast" },
    ];
    const out = formatWireResult(
      baseFailure("UPSTREAM_ERROR", "rpc unreachable", progress),
    );
    expect(out.isError).toBe(true);
    const text = out.content[0].text;
    expect(text).toContain("Error (UPSTREAM_ERROR): rpc unreachable");
    expect(text).toContain("Last phase: broadcast");
  });
});
