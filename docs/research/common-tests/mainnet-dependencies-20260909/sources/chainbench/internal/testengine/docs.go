// Package testengine is the test engine (L4). Its outermost flow, RunSuite,
// is the four-part structure the architecture fixes (consolidation plan
// §0-2): ① compose the chain the DSL declares through chainsetup — the one
// composition owner since R4 — then, per spec, the interpreter runs ② the
// pre-test hooks, ③ the test, and ④ the post-test hooks. The inner Engine
// (parse -> resolve -> fingerprint -> reuse-or-build -> run -> record ->
// teardown) attaches to the composed network; it builds nothing itself.
// Chain differences enter only through plugins, never through branching here.
package testengine
