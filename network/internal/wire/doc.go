// Package wire provides transport primitives for chainbench-net:
// command envelope decoding from stdin, NDJSON event/progress/result
// emission to stdout, structured logging to stderr, and a stream
// message decoder for subprocess consumers.
//
// wire is a pure library — it never manages process lifecycle or
// dispatches to drivers. Those concerns live in higher layers.
package wire
