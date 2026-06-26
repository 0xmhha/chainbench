import { describe, it, expect } from "vitest";
import { buildWireArgs } from "../src/utils/wireArgs.js";

describe("buildWireArgs", () => {
  it("copies only defined optional keys", () => {
    const source = { a: 1, b: undefined, c: "x" };
    expect(buildWireArgs(source, ["a", "b", "c"])).toEqual({ a: 1, c: "x" });
  });

  it("merges base (required) fields ahead of optionals", () => {
    const source = { node_id: "n1", block_number: undefined };
    const out = buildWireArgs(source, ["node_id", "block_number"], {
      network: "local",
      address: "0xabc",
    });
    expect(out).toEqual({ network: "local", address: "0xabc", node_id: "n1" });
  });

  it("preserves falsy-but-defined values (0, '', false)", () => {
    const source = { zero: 0, empty: "", flag: false, gone: undefined };
    expect(buildWireArgs(source, ["zero", "empty", "flag", "gone"])).toEqual({
      zero: 0,
      empty: "",
      flag: false,
    });
  });

  it("does not mutate the base object", () => {
    const base = { network: "local" };
    buildWireArgs({ x: 1 }, ["x"], base);
    expect(base).toEqual({ network: "local" });
  });
});
