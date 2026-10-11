package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestListsHonourCursorAndLimit(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenWebAuth(root, "team")
	if err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	s := NewServer(bus, nil, WithDeployments(store, WebAuthenticator(auth)), WithWebSecurity(auth, store))
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	admin, token, err := auth.Bootstrap("admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	actor := app.DeploymentActor{ID: admin.User.ID, Role: "admin"}
	for i := range 30 {
		if _, err = store.SaveDocument(actor, "", 0, app.DeploymentDocumentInput{Kind: "server-set", Name: fmt.Sprintf("pool-%02d", i), ContractVersion: "2", Content: json.RawMessage(`{"version":2,"pool":{"hosts":[{"name":"a","addr":"10.0.0.1"}],"slots":2,"ports":{"p2p":{"base":31000,"step":10},"rpc":{"base":8600,"step":10}}}}`), AssetRefs: []string{}}); err != nil {
			t.Fatal(err)
		}
		if _, err = auth.CreateUser(actor, fmt.Sprintf("user-%02d", i), "long-user-password", "viewer"); err != nil {
			t.Fatal(err)
		}
	}
	get := func(path string) (int, []json.RawMessage, *string) {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		var page struct {
			Items      []json.RawMessage `json:"items"`
			NextCursor *string           `json:"nextCursor"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &page)
		return w.Code, page.Items, page.NextCursor
	}
	for _, list := range []struct {
		path  string
		total int
	}{{"/api/v1/documents", 30}, {"/api/v1/users", 31}} {
		code, first, next := get(list.path)
		if code != 200 || len(first) != 25 || next == nil {
			t.Fatalf("%s first page: %d %d %v", list.path, code, len(first), next)
		}
		code, rest, end := get(list.path + "?cursor=" + *next)
		if code != 200 || len(rest) != list.total-25 || end != nil {
			t.Fatalf("%s second page: %d %d %v", list.path, code, len(rest), end)
		}
		seen := map[string]bool{}
		for _, item := range append(first, rest...) {
			if seen[string(item)] {
				t.Fatalf("%s repeats an item across pages", list.path)
			}
			seen[string(item)] = true
		}
		if code, small, _ := get(list.path + "?limit=3"); code != 200 || len(small) != 3 {
			t.Errorf("%s limit=3: %d %d", list.path, code, len(small))
		}
		for _, bad := range []string{"?limit=0", "?limit=101", "?cursor=absent"} {
			if code, _, _ := get(list.path + bad); code != 400 {
				t.Errorf("%s%s: %d", list.path, bad, code)
			}
		}
	}
}
