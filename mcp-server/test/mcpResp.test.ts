import { describe, it, expect } from "vitest";
import { errorResp } from "../src/utils/mcpResp.js";

describe("errorResp", () => {
  it("_Default_ShapesInvalidArgsResponse", () => {
    expect(errorResp("foo")).toEqual({
      content: [{ type: "text", text: "Error (INVALID_ARGS): foo" }],
      isError: true,
    });
  });
});
