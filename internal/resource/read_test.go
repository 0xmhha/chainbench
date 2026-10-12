package resource

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResourceReadBoundsAndLogRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node.log")
	if err := os.WriteFile(path, []byte("first log bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	request := ReadRequest{Target: Spec{DataRoot: dir}, Log: &LogRead{Path: path, Offset: 6, Limit: 4}}
	out, err := (Opener{}).Read(context.Background(), request)
	if err != nil || out.Log == nil || string(out.Log.Data) != "log " || string(out.Log.Head) != "first log bytes" {
		t.Fatal(out, err)
	}
	if err = os.Rename(path, path+".old"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	next, err := (Opener{}).Read(context.Background(), request)
	if err != nil || next.Log == nil || next.Log.ID == out.Log.ID {
		t.Fatal("rotation identity was lost", err)
	}
	for _, log := range []LogRead{{Path: path, Offset: -1, Limit: 4}, {Path: path, Limit: 0}, {Path: path, Limit: (1 << 20) + 1}, {Path: "relative", Limit: 4}} {
		request.Log = &log
		if _, err = (Opener{}).Read(context.Background(), request); err == nil {
			t.Fatal("unbounded log read accepted", log)
		}
	}
	request.Log = &LogRead{Path: filepath.Join(dir, "absent"), Limit: 4}
	next, err = (Opener{}).Read(context.Background(), request)
	if err != nil || !next.LogMissing || next.Log != nil {
		t.Fatal("missing log became an empty successful snapshot", next, err)
	}
}
