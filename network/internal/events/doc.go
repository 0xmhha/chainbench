// Package events provides an in-process pub/sub layer built on top of
// wire.Emitter. Publishers call Bus.Publish; subscribers receive events via
// buffered channels obtained from Bus.Subscribe. The Bus also delegates each
// published event to the underlying wire.Emitter so it is written as NDJSON
// on the process stdout.
//
// Publish is non-blocking: if a subscriber's channel buffer is full, the
// event is dropped for that subscriber. This prevents a slow consumer from
// stalling the publisher.
package events
