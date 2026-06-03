import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import type { ZodTypeAny } from "zod";

import { registerLifecycleTools } from "../src/tools/lifecycle.js";
import { registerNodeTools } from "../src/tools/node.js";
import { registerTestTools } from "../src/tools/test.js";
import { registerSchemaTools } from "../src/tools/schema.js";
import { registerLogTools } from "../src/tools/log.js";
import { registerRemoteTools } from "../src/tools/remote.js";
import { registerConsensusTools } from "../src/tools/consensus.js";
import { registerNetworkTools } from "../src/tools/network.js";
import { registerConfigTools } from "../src/tools/config.js";
import { registerSpecTools } from "../src/tools/spec.js";
import { registerChainTools } from "../src/tools/chain.js";

const __dirname = dirname(fileURLToPath(import.meta.url));

type ZodRawShape = Record<string, ZodTypeAny>;

// Capture every server.tool(name, …, shape, handler) registration without
// spinning up a real MCP server. The SDK keeps its registered-tool map private,
// so we mirror index.ts's registration against a stand-in that records the
// name and the input shape (the first object argument that is not a function).
function registerAll(): Map<string, ZodRawShape> {
  const tools = new Map<string, ZodRawShape>();
  const fakeServer = {
    tool: (name: string, ...rest: unknown[]) => {
      const beforeCb = rest.slice(0, -1); // drop the trailing handler
      const shape = beforeCb.find(
        (a) => typeof a === "object" && a !== null,
      ) as ZodRawShape | undefined;
      tools.set(name, shape ?? {});
    },
  };
  const s = fakeServer as any;
  registerLifecycleTools(s);
  registerNodeTools(s);
  registerTestTools(s);
  registerSchemaTools(s);
  registerLogTools(s);
  registerRemoteTools(s);
  registerConsensusTools(s);
  registerNetworkTools(s);
  registerConfigTools(s);
  registerSpecTools(s);
  registerChainTools(s);
  return tools;
}

interface Fragment {
  tools: Record<string, { input: { properties: Record<string, unknown>; required: string[] } }>;
}

function loadFragment(): Fragment {
  const p = resolve(__dirname, "fixtures/agent-subset.schema.json");
  return JSON.parse(readFileSync(p, "utf-8")) as Fragment;
}

describe("agent-facing tool contract (C1 chainbench slice / G1)", () => {
  const registered = registerAll();
  const fragment = loadFragment();
  const subset = Object.keys(fragment.tools);

  it("registers every locked subset tool by exact name", () => {
    for (const name of subset) {
      expect(registered.has(name), `missing tool: ${name}`).toBe(true);
    }
  });

  for (const name of subset) {
    describe(name, () => {
      it("input field names match the contract fragment", () => {
        const shape = registered.get(name)!;
        const actual = Object.keys(shape).sort();
        const expected = Object.keys(fragment.tools[name].input.properties).sort();
        expect(actual).toEqual(expected);
      });

      it("required input fields match the contract fragment", () => {
        const shape = registered.get(name)!;
        const actualRequired = Object.entries(shape)
          .filter(([, zt]) => !zt.isOptional())
          .map(([k]) => k)
          .sort();
        const expectedRequired = [...fragment.tools[name].input.required].sort();
        expect(actualRequired).toEqual(expectedRequired);
      });
    });
  }
});
