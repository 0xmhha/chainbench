// Package supervisor owns node bring-up and lifecycle (DDD context C3): it
// launches nodes, gates on real health (etcd leader ready, fork crossed),
// classifies failures instead of labeling them "flaky", and tears down without
// leaving orphan processes. Stopping a node also stops its embedded etcd;
// removing the datadir is a separate operation.
package supervisor
