// wireArgs.ts — builds the wire-envelope `args` object from a zod-parsed tool
// input.
//
// Tool handlers repeatedly accumulate optional fields with the
// `if (x !== undefined) wireArgs.x = x` idiom so that absent fields are omitted
// from the envelope (letting chainbench-net apply its own defaults). This
// helper expresses that contract once: required fields go in `base`, optional
// fields are copied from `source` only when defined.

export function buildWireArgs(
  source: Record<string, unknown>,
  optionalKeys: readonly string[],
  base: Record<string, unknown> = {},
): Record<string, unknown> {
  const out: Record<string, unknown> = { ...base };
  for (const key of optionalKeys) {
    if (source[key] !== undefined) out[key] = source[key];
  }
  return out;
}
