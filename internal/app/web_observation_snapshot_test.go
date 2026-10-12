package app

import (
	"context"
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

func TestObservationSnapshotIsRequestScopedAndRechecksRevocation(t *testing.T) {
	calls := 0
	denied := false
	r := &webTargetReader{check: func(context.Context, string) error {
		if denied {
			return ErrDeploymentForbidden
		}
		return nil
	}, read: func(context.Context, ResourceReadIn) (ResourceReadOut, error) {
		calls++
		return ResourceReadOut{Target: &resource.Inspection{HostIdentity: "machine"}, Checksums: map[string]string{"/binary": "accepted"}}, nil
	}}
	request := ResourceReadIn{ReadRequest: resource.ReadRequest{Target: resource.Spec{DataRoot: "/root"}, Checksums: []string{"/binary"}}}
	snapshot := r.observationSnapshot()
	out, err := snapshot.query(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	out.Checksums["/binary"] = "modified"
	out.Target.HostIdentity = "modified"
	out, err = snapshot.query(context.Background(), request)
	if err != nil || calls != 1 || out.Checksums["/binary"] != "accepted" || out.Target.HostIdentity != "machine" {
		t.Fatal("snapshot was not reused independently", out, calls, err)
	}
	if _, err = r.observationSnapshot().query(context.Background(), request); err != nil || calls != 2 {
		t.Fatal("next request reused stale observation", calls, err)
	}
	if _, err = r.query(context.Background(), request); err != nil || calls != 3 {
		t.Fatal("execution read used observation snapshot", calls, err)
	}
	denied = true
	if _, err = snapshot.query(context.Background(), request); !errors.Is(err, ErrDeploymentForbidden) || calls != 3 {
		t.Fatal("revoked snapshot reused", calls, err)
	}
}

func TestObservationSnapshotDoesNotCacheLiveReadsOrErrors(t *testing.T) {
	calls := 0
	r := &webTargetReader{check: func(context.Context, string) error { return nil }, read: func(context.Context, ResourceReadIn) (ResourceReadOut, error) {
		calls++
		if calls == 1 {
			return ResourceReadOut{}, errors.New("transient read failure")
		}
		return ResourceReadOut{}, nil
	}}
	s := r.observationSnapshot()
	request := ResourceReadIn{ReadRequest: resource.ReadRequest{Target: resource.Spec{DataRoot: "/root"}, Inspect: true}}
	_, _ = s.query(context.Background(), request)
	if _, err := s.query(context.Background(), request); err != nil || calls != 2 {
		t.Fatal("failure was cached", calls, err)
	}
	for _, live := range []ResourceReadIn{{ReadRequest: resource.ReadRequest{Process: &resource.ProcessRead{}}}, {ReadRequest: resource.ReadRequest{Log: &resource.LogRead{}}}} {
		before := calls
		for i := 0; i < 2; i++ {
			if _, err := s.query(context.Background(), live); err != nil {
				t.Fatal(err)
			}
		}
		if calls != before+2 {
			t.Fatal("live reads were cached")
		}
	}
}
